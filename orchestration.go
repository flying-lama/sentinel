package main

// OrchestrationAdapter defines the interface for orchestration-specific operations
type OrchestrationAdapter interface {
	GetConfigurationErrors() []string
	GetNodeName() (string, error)
	GetNodePublicIP() (string, error)
	IsLeader() bool
	WatchEvents(callback func())
}
