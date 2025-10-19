package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ljluestc/orchestrator/pkg/probe"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	probeID         = flag.String("probe-id", "", "Probe ID")
	hostname        = flag.String("hostname", "", "Hostname")
	appURL          = flag.String("app-url", "http://monitoring-app:8080", "Monitoring app URL")
	collectInterval = flag.Duration("collect-interval", 15*time.Second, "Collection interval")
	metricsPort     = flag.Int("metrics-port", 8090, "Metrics port")
)

func main() {
	flag.Parse()

	if *probeID == "" {
		*probeID = os.Getenv("NODE_NAME")
		if *probeID == "" {
			log.Fatal("probe-id is required")
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	// Create probe
	p, err := probe.NewProbe(probe.ProbeConfig{
		ServerURL:          *appURL,
		AgentID:            *probeID,
		CollectionInterval: *collectInterval,
		CollectHost:        true,
		CollectDocker:      true,
		CollectProcesses:   true,
		CollectNetwork:     true,
		CollectDockerStats: true,
		MaxProcesses:       500,
		MaxConnections:     1000,
		IncludeLocalhost:   false,
		ResolveProcesses:   true,
	})
	if err != nil {
		log.Fatalf("Failed to create probe: %v", err)
	}
	defer p.Stop()

	// Start metrics server
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		log.Printf("Starting metrics server on port %d", *metricsPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", *metricsPort), mux); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// Start probe
	log.Printf("Starting probe %s", *probeID)
	if err := p.Start(ctx); err != nil {
		log.Fatalf("Probe error: %v", err)
	}
}
