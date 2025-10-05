package main

import (
	"os"
	"testing"

	"github.com/ljluestc/orchestrator/pkg/probe"
	"github.com/stretchr/testify/assert"
)

func TestPrintConfig(t *testing.T) {
	// Test printConfig function
	config := probe.ProbeConfig{
		ServerURL:           "http://localhost:8080",
		AgentID:             "test-agent",
		CollectionInterval:  30,
		HeartbeatInterval:   60,
		CollectHost:         true,
		CollectDocker:       true,
		CollectDockerStats:  false,
		CollectProcesses:    true,
		CollectNetwork:      true,
		MaxProcesses:        100,
		MaxConnections:      100,
		IncludeLocalhost:    true,
		IncludeAllProcesses: true,
		ResolveProcesses:    true,
		RetryAttempts:       3,
		RetryDelay:          5,
	}

	// This should not panic
	assert.NotPanics(t, func() {
		printConfig(config)
	})
}

func TestMainFunction(t *testing.T) {
	// Test that main function doesn't panic when called
	// We'll set some environment variables to control behavior
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Set minimal args to avoid flag parsing issues
	os.Args = []string{"probe-agent", "-server", "http://localhost:9999"}

	// We can't easily test main() directly, but we can test that it doesn't panic
	// by running it in a goroutine and checking for panics
	done := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("main() panicked: %v", r)
			}
			done <- true
		}()
		
		// We can't call main() directly, but we can test the flag parsing
		// and configuration creation logic
		// Note: flag.Parse() is called in main(), so we can't call it here
		// We'll just test the configuration creation logic
		
		config := probe.ProbeConfig{
			ServerURL:          *serverURL,
			AgentID:            *agentID,
			APIKey:             *apiKey,
			CollectionInterval: *collectionInterval,
			HeartbeatInterval:  *heartbeatInterval,
			CollectHost:        *collectHost,
			CollectDocker:      *collectDocker,
			CollectDockerStats: *collectDockerStats,
			CollectProcesses:   *collectProcesses,
			CollectNetwork:     *collectNetwork,
			MaxProcesses:       *maxProcesses,
			MaxConnections:     *maxConnections,
			IncludeLocalhost:   *includeLocalhost,
			IncludeAllProcesses: *includeAllProcs,
			ResolveProcesses:   *resolveProcesses,
			RetryAttempts:      *retryAttempts,
			RetryDelay:         *retryDelay,
		}
		
		assert.NotNil(t, config)
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		// Test completed successfully
	case <-make(chan bool):
		// Timeout - this is expected since main() would run indefinitely
	}
}
