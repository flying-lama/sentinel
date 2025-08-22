package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

const OrchestrationTypeDockerSwarm = "swarm"
const OrchestrationTypeKubernetes = "kubernetes"

// Config holds the application configuration
type Config struct {
	Domain            string
	Record            string
	ServerIP          string
	InwxUser          string
	InwxPassword      string
	RecordID          int
	LogLevel          string
	OrchestrationType string
}

// Sentinel is the main application struct
type Sentinel struct {
	Config        *Config
	inwxClient    *InwxClient
	orchestration OrchestrationAdapter
}

// NewConfig creates a new Config from environment variables
func NewConfig() (*Config, error) {
	domain := getEnv("DOMAIN", "example.com")
	record := getEnv("RECORD", "lb")
	inwxUser := getEnv("INWX_USER", "")
	inwxRecordIDStr := getEnv("INWX_RECORD_ID", "")
	logLevel := getEnv("LOG_LEVEL", "INFO")
	orchestrationType := getEnv("ORCHESTRATION_TYPE", OrchestrationTypeDockerSwarm)
	var err error

	if inwxUser == "" || inwxRecordIDStr == "" {
		return nil, fmt.Errorf("INWX_USER or INWX_RECORD_ID not set")
	}

	// Try to read password from Docker secret first
	inwxPassword, err := readSecret("/run/secrets/inwx_password")
	if err != nil {
		// Fall back to environment variable if secret not available
		inwxPassword = getEnv("INWX_PASSWORD", "")
		if inwxPassword == "" {
			return nil, fmt.Errorf("INWX_PASSWORD not set and could not read from secret: %v", err)
		}
	}

	recordID, err := strconv.Atoi(inwxRecordIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid INWX_RECORD_ID: %v", err)
	}

	return &Config{
		Domain:            domain,
		Record:            record,
		InwxUser:          inwxUser,
		InwxPassword:      inwxPassword,
		RecordID:          recordID,
		LogLevel:          logLevel,
		OrchestrationType: orchestrationType,
	}, nil
}

// NewSentinel creates a new Sentinel instance
func NewSentinel(config *Config) *Sentinel {
	sentinel := &Sentinel{
		Config:     config,
		inwxClient: NewInwxClient(config),
	}

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
		s.updateDNS()
	}
}

func (s *Sentinel) updateDNS() {
	log.Println("This instance is the Leader")

	currentIP, err := s.inwxClient.GetRecordContent()
	if err != nil {
		log.Printf("Could not determine current DNS record: %v", err)
		return
	}

	if currentIP != s.Config.ServerIP {
		log.Printf("DNS points to %s, should point to %s", currentIP, s.Config.ServerIP)
		if err := s.inwxClient.UpdateDNS(s.Config.ServerIP); err != nil {
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
