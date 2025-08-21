package main

// OrchestrationAdapter defines the interface for orchestration-specific operations
type OrchestrationAdapter interface {
	GetConfigurationErrors() []string
	GetNodePublicIP() (string, error)
	IsLeader() bool
	WatchEvents(callback func())
}
