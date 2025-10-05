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

	"github.com/ljluestc/orchestrator/pkg/app"
)

var (
	host               = flag.String("host", "0.0.0.0", "Server host address")
	port               = flag.Int("port", 8080, "Server port")
	maxDataAge         = flag.Duration("max-data-age", 1*time.Hour, "Maximum age for stored data")
	cleanupInterval    = flag.Duration("cleanup-interval", 5*time.Minute, "Cleanup interval for stale data")
	staleNodeThreshold = flag.Duration("stale-node-threshold", 5*time.Minute, "Threshold for considering nodes stale")
)

func main() {
	flag.Parse()

	// Setup logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting App Backend Server...")

	// Create server configuration
	config := app.ServerConfig{
		Host:               *host,
		Port:               *port,
		MaxDataAge:         *maxDataAge,
		CleanupInterval:    *cleanupInterval,
		StaleNodeThreshold: *staleNodeThreshold,
	}

	// Create server
	server := app.NewServer(config)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server
	if err := server.Start(ctx); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Print configuration
	printConfig(config)

	// Wait for termination signal
	sig := <-sigChan
	log.Printf("Received signal: %v", sig)

	// Cancel context to stop background tasks
	cancel()

	// Stop server gracefully
	log.Println("Stopping app server...")
	if err := server.Stop(); err != nil {
		log.Printf("Error stopping server: %v", err)
	}

	log.Println("App server stopped successfully")
}

func printConfig(config app.ServerConfig) {
	fmt.Println("\n=== App Backend Server Configuration ===")
	fmt.Printf("Host:                  %s\n", config.Host)
	fmt.Printf("Port:                  %d\n", config.Port)
	fmt.Printf("Max Data Age:          %s\n", config.MaxDataAge)
	fmt.Printf("Cleanup Interval:      %s\n", config.CleanupInterval)
	fmt.Printf("Stale Node Threshold:  %s\n", config.StaleNodeThreshold)
	fmt.Println("\n=== Endpoints ===")
	fmt.Printf("REST API:              http://%s:%d/api/v1\n", config.Host, config.Port)
	fmt.Printf("WebSocket:             ws://%s:%d/api/v1/ws\n", config.Host, config.Port)
	fmt.Printf("Health Check:          http://%s:%d/health\n", config.Host, config.Port)
	fmt.Println("\n=== Available Endpoints ===")
	fmt.Println("POST   /api/v1/agents/register     - Register a new agent")
	fmt.Println("POST   /api/v1/agents/heartbeat/:agent_id - Send heartbeat")
	fmt.Println("GET    /api/v1/agents/config/:agent_id - Get agent configuration")
	fmt.Println("GET    /api/v1/agents/list         - List all agents")
	fmt.Println("POST   /api/v1/reports             - Submit a report")
	fmt.Println("GET    /api/v1/query/topology      - Get current topology")
	fmt.Println("GET    /api/v1/query/agents/:agent_id/latest - Get latest report")
	fmt.Println("GET    /api/v1/query/agents/:agent_id/timeseries - Get time-series data")
	fmt.Println("GET    /api/v1/query/stats         - Get server statistics")
	fmt.Println("GET    /api/v1/ws                  - WebSocket connection")
	fmt.Println("========================================")
}
