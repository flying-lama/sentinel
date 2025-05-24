package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
)

// DockerClient handles communication with the Docker API
type DockerClient struct {
	client *http.Client
}

// DockerEvent represents a Docker event from the API
type DockerEvent struct {
	Type   string `json:"Type"`
	Action string `json:"Action"`
	Actor  struct {
		ID         string            `json:"ID"`
		Attributes map[string]string `json:"Attributes"`
	} `json:"Actor"`
}

// NodeInfo represents Docker Swarm node information
type NodeInfo struct {
	ID            string `json:"ID"`
	ManagerStatus *struct {
		Leader bool `json:"Leader"`
	} `json:"ManagerStatus,omitempty"`
}

// NewDockerClient creates a new Docker API client
func NewDockerClient() *DockerClient {
	return &DockerClient{
		client: &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", "/var/run/docker.sock")
				},
			},
		},
	}
}

// IsSwarmActive checks if Docker is running in swarm mode
func (d *DockerClient) IsSwarmActive() bool {
	req, err := http.NewRequest("GET", "http://localhost/swarm", nil)
	if err != nil {
		log.Printf("Error creating swarm request: %v", err)
		return false
	}

	resp, err := d.client.Do(req)
	if err != nil {
		log.Printf("Error connecting to Docker API: %v", err)
		return false
	}
	defer resp.Body.Close()

	var swarmInfo struct {
		ID string `json:"ID"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&swarmInfo); err != nil {
		log.Printf("Error parsing swarm response: %v", err)
		return false
	}

	// If swarm ID is empty, swarm is not active
	return swarmInfo.ID != ""
}

// IsSwarmLeader checks if this node is the swarm leader
func (d *DockerClient) IsSwarmLeader() bool {
	currentNodeID, err := d.GetCurrentNodeID()
	if err != nil {
		log.Printf("Error getting current node ID: %v", err)
		return false
	}

	req, err := http.NewRequest("GET", "http://localhost/nodes", nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return false
	}

	resp, err := d.client.Do(req)
	if err != nil {
		log.Printf("Error connecting to Docker API: %v", err)
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return false
	}

	// Only log the raw response if log level is DEBUG
	if getEnv("LOG_LEVEL", "INFO") == "DEBUG" {
		log.Printf("Raw nodes response: %s", string(body))
	}

	var nodes []NodeInfo
	if err := json.Unmarshal(body, &nodes); err != nil {
		log.Printf("Error parsing nodes response: %v", err)
		return false
	}

	for _, node := range nodes {
		if node.ID == currentNodeID && node.ManagerStatus != nil && node.ManagerStatus.Leader {
			return true
		}
	}

	return false
}

// WatchEvents watches Docker events for node updates
func (d *DockerClient) WatchEvents(callback func()) {
	req, err := http.NewRequest("GET", "http://localhost/events?filters={\"scope\":[\"swarm\"]}", nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return
	}

	resp, err := d.client.Do(req)
	if err != nil {
		log.Printf("Error connecting to Docker API: %v", err)
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event DockerEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			log.Printf("Error parsing event: %v", err)
			continue
		}

		if event.Type == "node" && event.Action == "update" {
			log.Println("Node update detected, checking leader status...")
			callback()
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading events: %v", err)
	}
}

// GetCurrentNodeID retrieves the ID of the current node from Docker API
func (d *DockerClient) GetCurrentNodeID() (string, error) {
	// Docker API endpoint for information about the current node
	req, err := http.NewRequest("GET", "http://localhost/info", nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error connecting to Docker API: %v", err)
	}
	defer resp.Body.Close()

	var info struct {
		Swarm struct {
			NodeID string `json:"NodeID"`
		} `json:"Swarm"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", fmt.Errorf("error parsing node info: %v", err)
	}

	if info.Swarm.NodeID == "" {
		return "", fmt.Errorf("could not determine node ID")
	}

	return info.Swarm.NodeID, nil
}

// GetNodeLabel retrieves a specific label from a node
func (d *DockerClient) GetNodeLabel(nodeID, labelName string) (string, error) {
	// Docker API endpoint for node information
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost/nodes/%s", nodeID), nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error connecting to Docker API: %v", err)
	}
	defer resp.Body.Close()

	var node struct {
		Spec struct {
			Labels map[string]string `json:"Labels"`
		} `json:"Spec"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return "", fmt.Errorf("error parsing node response: %v", err)
	}

	value, exists := node.Spec.Labels[labelName]
	if !exists {
		return "", fmt.Errorf("label %s not found on node %s", labelName, nodeID)
	}

	return value, nil
}

// GetNodePublicIP retrieves the public IP address from the node's label
func (d *DockerClient) GetNodePublicIP() (string, error) {
	// First get the node ID
	nodeID, err := d.GetCurrentNodeID()
	if err != nil {
		return "", fmt.Errorf("failed to get node ID: %v", err)
	}

	// Then retrieve the public IP label
	publicIP, err := d.GetNodeLabel(nodeID, "public_ip")
	if err != nil {
		return "", fmt.Errorf("failed to get public_ip label: %v", err)
	}

	return publicIP, nil
}
