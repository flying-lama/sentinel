package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Set up logging
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Create configuration from environment variables
	config, err := NewConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Configure log level
	configureLogging(config.LogLevel)

	// Create and initialize the sentinel
	sentinel := NewSentinel(config)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run the sentinel in a goroutine
	go func() {
		log.Println("Starting Sentinel DNS monitor...")
		sentinel.Run()
	}()

	// Wait for termination signal
	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...", sig)
}

// configureLogging sets up logging based on the configured level
func configureLogging(level string) {
	switch level {
	case "DEBUG":
		log.Println("Debug logging enabled")
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	case "INFO":
		log.SetFlags(log.Ldate | log.Ltime)
	case "ERROR":
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	default:
		log.SetFlags(log.Ldate | log.Ltime)
	}
}
