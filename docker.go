package main

import (
	"bufio"
	"context"
	"encoding/json"
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
	ManagerStatus struct {
		Leader bool `json:"Leader"`
	} `json:"ManagerStatus"`
	Self bool `json:"Self"`
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

	var nodes []NodeInfo
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		log.Printf("Error parsing nodes response: %v", err)
		return false
	}

	for _, node := range nodes {
		if node.Self && node.ManagerStatus.Leader {
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
