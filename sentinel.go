package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the application configuration
type Config struct {
	Domain       string
	Record       string
	ServerIP     string
	InwxUser     string
	InwxPassword string
	RecordID     int
	LogLevel     string
}

// Sentinel is the main application struct
type Sentinel struct {
	config       *Config
	dockerClient *DockerClient
	inwxClient   *InwxClient
}

// NewConfig creates a new Config from environment variables
func NewConfig() (*Config, error) {
	domain := getEnv("DOMAIN", "example.com")
	record := getEnv("RECORD", "lb")
	inwxUser := getEnv("INWX_USER", "")
	inwxRecordIDStr := getEnv("INWX_RECORD_ID", "")
	logLevel := getEnv("LOG_LEVEL", "INFO")

	dockerClient := NewDockerClient()
	var err error
	serverIP, err := dockerClient.GetNodePublicIP()
	if err != nil {
		log.Fatalf("Error: Could not get public IP from node label: %v", err)
	}

	// Check for required configuration
	if serverIP == "" {
		return nil, fmt.Errorf("SERVER_IP not set and could not determine from node label")
	}

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
		Domain:       domain,
		Record:       record,
		ServerIP:     serverIP,
		InwxUser:     inwxUser,
		InwxPassword: inwxPassword,
		RecordID:     recordID,
		LogLevel:     logLevel,
	}, nil
}

// NewSentinel creates a new Sentinel instance
func NewSentinel(config *Config) *Sentinel {
	return &Sentinel{
		config:       config,
		dockerClient: NewDockerClient(),
		inwxClient:   NewInwxClient(config),
	}
}

// CheckAndUpdateDNS checks if this node is the leader and updates DNS if needed
func (s *Sentinel) CheckAndUpdateDNS() {
	if s.dockerClient.IsSwarmLeader() {
		log.Println("This instance is the Swarm Leader")

		currentIP, err := s.inwxClient.GetRecordContent()
		if err != nil {
			log.Printf("Could not determine current DNS record: %v", err)
			return
		}

		if currentIP != s.config.ServerIP {
			log.Printf("DNS points to %s, should point to %s", currentIP, s.config.ServerIP)
			if err := s.inwxClient.UpdateDNS(s.config.ServerIP); err != nil {
				log.Printf("DNS update failed: %v", err)
			} else {
				log.Printf("DNS update successful")
			}
		} else {
			log.Printf("DNS correctly points to %s", s.config.ServerIP)
		}
	} else {
		log.Println("This instance is not the Swarm Leader")
	}
}

// Run starts the sentinel monitoring process
func (s *Sentinel) Run() {
	log.Printf("Sentinel DNS Monitor for %s.%s started", s.config.Record, s.config.Domain)
	log.Printf("Server IP: %s", s.config.ServerIP)

	// Check if Docker is running in swarm mode
	if !s.dockerClient.IsSwarmActive() {
		log.Fatal("Docker is not running in swarm mode. Sentinel requires Docker Swarm to be active.")
	}

	// Initial check
	s.CheckAndUpdateDNS()

	// Watch for events
	for {
		log.Println("Starting Docker events monitoring...")
		s.dockerClient.WatchEvents(s.CheckAndUpdateDNS)

		log.Println("Docker events connection lost, reconnecting in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
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
