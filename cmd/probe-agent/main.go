package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ljluestc/orchestrator/pkg/probe"
)

var (
	serverURL          = flag.String("server", "http://localhost:8080", "Server URL")
	agentID            = flag.String("agent-id", "", "Agent ID (auto-generated if empty)")
	apiKey             = flag.String("api-key", "", "API key for authentication")
	collectionInterval = flag.Duration("interval", 30*time.Second, "Collection interval")
	heartbeatInterval  = flag.Duration("heartbeat", 60*time.Second, "Heartbeat interval")
	collectHost        = flag.Bool("collect-host", true, "Collect host information")
	collectDocker      = flag.Bool("collect-docker", true, "Collect Docker information")
	collectDockerStats = flag.Bool("collect-docker-stats", false, "Collect Docker container stats")
	collectProcesses   = flag.Bool("collect-processes", true, "Collect process information")
	collectNetwork     = flag.Bool("collect-network", true, "Collect network information")
	maxProcesses       = flag.Int("max-processes", 100, "Maximum number of processes to collect")
	maxConnections     = flag.Int("max-connections", 100, "Maximum number of network connections to collect")
	includeLocalhost   = flag.Bool("include-localhost", true, "Include localhost connections")
	includeAllProcs    = flag.Bool("include-all-processes", false, "Include all processes (not just containers)")
	resolveProcesses   = flag.Bool("resolve-processes", true, "Resolve process names for network connections")
	retryAttempts      = flag.Int("retry-attempts", 3, "Number of retry attempts for failed requests")
	retryDelay         = flag.Duration("retry-delay", 5*time.Second, "Delay between retry attempts")
)

func main() {
	flag.Parse()

	// Setup logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Probe Agent...")

	// Create probe configuration
	config := probe.ProbeConfig{
		ServerURL:           *serverURL,
		AgentID:             *agentID,
		APIKey:              *apiKey,
		CollectionInterval:  *collectionInterval,
		HeartbeatInterval:   *heartbeatInterval,
		CollectHost:         *collectHost,
		CollectDocker:       *collectDocker,
		CollectDockerStats:  *collectDockerStats,
		CollectProcesses:    *collectProcesses,
		CollectNetwork:      *collectNetwork,
		MaxProcesses:        *maxProcesses,
		MaxConnections:      *maxConnections,
		IncludeLocalhost:    *includeLocalhost,
		IncludeAllProcesses: *includeAllProcs,
		ResolveProcesses:    *resolveProcesses,
		RetryAttempts:       *retryAttempts,
		RetryDelay:          *retryDelay,
	}

	// Create probe
	p, err := probe.NewProbe(config)
	if err != nil {
		log.Fatalf("Failed to create probe: %v", err)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start probe
	if err := p.Start(ctx); err != nil {
		log.Fatalf("Failed to start probe: %v", err)
	}

	// Print configuration
	printConfig(config)

	// Wait for termination signal
	sig := <-sigChan
	log.Printf("Received signal: %v", sig)

	// Cancel context to stop background tasks
	cancel()

	// Stop probe gracefully
	log.Println("Stopping probe agent...")
	if err := p.Stop(); err != nil {
		log.Printf("Error stopping probe: %v", err)
	}

	log.Println("Probe agent stopped successfully")
}

func printConfig(config probe.ProbeConfig) {
	fmt.Println("\n=== Probe Agent Configuration ===")
	fmt.Printf("Server URL:           %s\n", config.ServerURL)
	fmt.Printf("Agent ID:             %s\n", config.AgentID)
	fmt.Printf("Collection Interval:  %s\n", config.CollectionInterval)
	fmt.Printf("Heartbeat Interval:   %s\n", config.HeartbeatInterval)
	fmt.Println("\n=== Collection Modules ===")
	fmt.Printf("Host Information:     %v\n", config.CollectHost)
	fmt.Printf("Docker Information:   %v\n", config.CollectDocker)
	if config.CollectDocker {
		fmt.Printf("  - Container Stats:  %v\n", config.CollectDockerStats)
	}
	fmt.Printf("Process Information:  %v\n", config.CollectProcesses)
	if config.CollectProcesses {
		fmt.Printf("  - Max Processes:    %d\n", config.MaxProcesses)
		fmt.Printf("  - Include All:      %v\n", config.IncludeAllProcesses)
	}
	fmt.Printf("Network Information:  %v\n", config.CollectNetwork)
	if config.CollectNetwork {
		fmt.Printf("  - Max Connections:  %d\n", config.MaxConnections)
		fmt.Printf("  - Include Localhost:%v\n", config.IncludeLocalhost)
		fmt.Printf("  - Resolve Processes:%v\n", config.ResolveProcesses)
	}
	fmt.Println("\n=== Retry Configuration ===")
	fmt.Printf("Retry Attempts:       %d\n", config.RetryAttempts)
	fmt.Printf("Retry Delay:          %s\n", config.RetryDelay)
	fmt.Println("================================")
}
