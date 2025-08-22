package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/libdns/bunny"
	"github.com/libdns/inwx"
	"github.com/libdns/libdns"
)

const OrchestrationTypeDockerSwarm = "swarm"
const OrchestrationTypeKubernetes = "kubernetes"

const DnsProviderInwx = "inwx"
const DnsProviderBunny = "bunny"

// Config holds the application configuration
type Config struct {
	Domain            string
	Record            string
	RecordTTL         int64
	ServerIP          string
	LogLevel          string
	OrchestrationType string
	DnsProvider       string // "inwx" or "bunny"
}

// Sentinel is the main application struct
type Sentinel struct {
	Config        *Config
	DnsClient     DnsClient
	orchestration OrchestrationAdapter
}

// NewConfig creates a new Config from environment variables
func NewConfig() (*Config, error) {
	domain := getEnv("DOMAIN", "example.com")
	record := getEnv("RECORD", "lb")
	logLevel := getEnv("LOG_LEVEL", "INFO")
	orchestrationType := getEnv("ORCHESTRATION_TYPE", OrchestrationTypeDockerSwarm)
	dnsProvider := getEnv("DNS_PROVIDER", DnsProviderInwx)

	config := &Config{
		Domain:            domain,
		Record:            record,
		LogLevel:          logLevel,
		OrchestrationType: orchestrationType,
		DnsProvider:       dnsProvider,
	}

	return config, nil
}

func configureInwx(c *Config) (*inwx.Provider, error) {
	c.RecordTTL = 300

	inwxUser := getEnv("INWX_USER", "")

	if inwxUser == "" {
		return nil, fmt.Errorf("INWX_USER not set")
	}

	inwxPassword, err := readSecret("/run/secrets/inwx_password")
	if err != nil {
		inwxPassword = getEnv("INWX_PASSWORD", "")
		if inwxPassword == "" {
			return nil, fmt.Errorf("INWX_PASSWORD not set and could not read from secret: %v", err)
		}
	}

	return &inwx.Provider{
		Username: inwxUser,
		Password: inwxPassword,
	}, nil
}

func configureBunny(c *Config) (*bunny.Provider, error) {
	c.RecordTTL = 15

	bunnyAPIKey := getEnv("BUNNY_API_KEY", "")

	if bunnyAPIKey == "" {
		return nil, fmt.Errorf("BUNNY_API_KEY not set")
	}

	return &bunny.Provider{
		AccessKey: bunnyAPIKey,
	}, nil
}

// NewSentinel creates a new Sentinel instance
func NewSentinel(config *Config) *Sentinel {
	sentinel := &Sentinel{
		Config: config,
	}

	var dnsClient DnsClient
	var err error
	switch config.DnsProvider {
	case DnsProviderInwx:
		dnsClient, err = configureInwx(config)
	case DnsProviderBunny:
		dnsClient, err = configureBunny(config)
	default:
		err = errors.New("Unsupported DNS provider: " + config.DnsProvider)
	}

	if err != nil {
		log.Fatalf("Error configuring DNS provider%s: %v", config.DnsProvider, err)
	}

	sentinel.DnsClient = dnsClient

	if config.OrchestrationType == OrchestrationTypeDockerSwarm {
		sentinel.orchestration = NewDockerClient()
	} else if config.OrchestrationType == OrchestrationTypeKubernetes {
		k8sAdapter, err := NewK8sClient()
		if err != nil {
			log.Fatalf("Error creating Kubernetes orchestration: %v", err)
		}
		sentinel.orchestration = k8sAdapter
	}

	serverIP, err := sentinel.orchestration.GetNodePublicIP()
	if err != nil {
		log.Fatalf("Error: Could not get public IP: %v", err)
	}
	sentinel.Config.ServerIP = serverIP

	return sentinel
}

// CheckAndUpdateDNS checks if this node is the leader and updates DNS if needed
func (s *Sentinel) CheckAndUpdateDNS() {
	if s.orchestration.IsLeader() {
		log.Println("This instance is the Leader")
		s.updateDNS()
	}
}

func (s *Sentinel) updateDNS() {
	ctx := context.Background()
	zone := s.Config.Domain + "."

	records, err := s.DnsClient.GetRecords(ctx, zone)
	if err != nil {
		log.Printf("Could not get DNS records: %v", err)
		return
	}

	var currentIP string
	for _, record := range records {
		rr := record.RR()
		if rr.Name == s.Config.Record && rr.Type == "A" {
			currentIP = rr.Data
			break
		}
	}

	if currentIP != s.Config.ServerIP {
		log.Printf("DNS points to %s, should point to %s", currentIP, s.Config.ServerIP)

		newRecords := []libdns.Record{
			libdns.Address{
				Name: s.Config.Record,
				IP:   netip.MustParseAddr(s.Config.ServerIP),
				TTL:  time.Duration(s.Config.RecordTTL) * time.Second,
			},
		}

		_, err := s.DnsClient.SetRecords(ctx, zone, newRecords)
		if err != nil {
			log.Printf("DNS update failed: %v", err)
		} else {
			log.Printf("DNS update successful")
		}
	} else {
		log.Printf("DNS correctly points to %s", s.Config.ServerIP)
	}
}

// Run starts the sentinel monitoring process
func (s *Sentinel) Run() {
	log.Printf("Sentinel DNS Monitor for %s.%s started", s.Config.Record, s.Config.Domain)
	log.Printf("Server IP: %s", s.Config.ServerIP)

	configErrs := s.orchestration.GetConfigurationErrors()
	if len(configErrs) > 0 {
		log.Fatal("Invalid configuration: ", configErrs)
	}

	nodeName, _ := s.orchestration.GetNodeName()
	log.Printf("Node name: %s", nodeName)

	// Initial check
	s.CheckAndUpdateDNS()

	// Watch for events
	s.orchestration.WatchEvents(s.CheckAndUpdateDNS)
}

func getEnv(key, fallback string) string {
	fullKey := "SENTINEL_" + key
	if value, exists := os.LookupEnv(fullKey); exists {
		return value
	}
	return fallback
}

// readSecret reads a secret from the given path
func readSecret(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
