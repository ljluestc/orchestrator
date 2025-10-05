package probe

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// ProbeConfig contains configuration for the probe agent
type ProbeConfig struct {
	// Client configuration
	ServerURL      string
	AgentID        string
	APIKey         string

	// Collection intervals
	CollectionInterval time.Duration
	HeartbeatInterval  time.Duration

	// Feature flags
	CollectHost      bool
	CollectDocker    bool
	CollectProcesses bool
	CollectNetwork   bool
	CollectDockerStats bool

	// Limits
	MaxProcesses   int
	MaxConnections int

	// Network options
	IncludeLocalhost  bool
	ResolveProcesses  bool

	// Process options
	IncludeAllProcesses bool

	// Retry configuration
	RetryAttempts int
	RetryDelay    time.Duration
}

// Probe is the main probe agent structure
type Probe struct {
	config           ProbeConfig
	client           *Client
	hostCollector    *HostCollector
	dockerCollector  *DockerCollector
	processCollector *ProcessCollector
	networkCollector *NetworkCollector

	hostname         string
	running          bool
	mu               sync.RWMutex
	stopCh           chan struct{}
	wg               sync.WaitGroup
}

// NewProbe creates a new probe agent
func NewProbe(config ProbeConfig) (*Probe, error) {
	// Set defaults
	if config.CollectionInterval == 0 {
		config.CollectionInterval = 30 * time.Second
	}
	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = 60 * time.Second
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 5 * time.Second
	}

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	// Generate agent ID if not provided
	if config.AgentID == "" {
		config.AgentID = fmt.Sprintf("%s-%d", hostname, time.Now().Unix())
	}

	probe := &Probe{
		config:   config,
		hostname: hostname,
		stopCh:   make(chan struct{}),
	}

	// Initialize client
	probe.client = NewClient(ClientConfig{
		ServerURL:      config.ServerURL,
		AgentID:        config.AgentID,
		APIKey:         config.APIKey,
		RequestTimeout: 30 * time.Second,
		RetryAttempts:  config.RetryAttempts,
		RetryDelay:     config.RetryDelay,
	})

	// Initialize collectors based on configuration
	if config.CollectHost {
		probe.hostCollector = NewHostCollector()
	}

	if config.CollectDocker {
		dockerCollector, err := NewDockerCollector(config.CollectDockerStats)
		if err != nil {
			return nil, fmt.Errorf("failed to create Docker collector: %w", err)
		}
		probe.dockerCollector = dockerCollector
	}

	if config.CollectProcesses {
		probe.processCollector = NewProcessCollector(
			config.IncludeAllProcesses,
			config.MaxProcesses,
		)
	}

	if config.CollectNetwork {
		probe.networkCollector = NewNetworkCollector(
			config.IncludeLocalhost,
			config.MaxConnections,
			config.ResolveProcesses,
		)
	}

	return probe, nil
}

// Start starts the probe agent
func (p *Probe) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("probe is already running")
	}
	p.running = true
	p.mu.Unlock()

	log.Printf("Starting probe agent (ID: %s, Hostname: %s)", p.config.AgentID, p.hostname)

	// Register agent
	metadata := map[string]string{
		"version": "1.0.0",
		"os":      "linux",
	}

	if err := p.client.RegisterAgent(ctx, p.hostname, metadata); err != nil {
		log.Printf("Warning: Failed to register agent: %v", err)
	} else {
		log.Printf("Agent registered successfully")
	}

	// Start collection loop
	p.wg.Add(1)
	go p.collectionLoop(ctx)

	// Start heartbeat loop
	p.wg.Add(1)
	go p.heartbeatLoop(ctx)

	log.Printf("Probe agent started")

	return nil
}

// Stop stops the probe agent
func (p *Probe) Stop() error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return fmt.Errorf("probe is not running")
	}
	p.running = false
	p.mu.Unlock()

	log.Printf("Stopping probe agent...")

	// Signal goroutines to stop
	close(p.stopCh)

	// Wait for goroutines to finish
	p.wg.Wait()

	// Close collectors
	if p.dockerCollector != nil {
		p.dockerCollector.Close()
	}

	// Close client
	p.client.Close()

	log.Printf("Probe agent stopped")

	return nil
}

// collectionLoop runs the main collection loop
func (p *Probe) collectionLoop(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.CollectionInterval)
	defer ticker.Stop()

	// Collect immediately on start
	p.collectAndSend(ctx)

	for {
		select {
		case <-ticker.C:
			p.collectAndSend(ctx)
		case <-p.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// heartbeatLoop runs the heartbeat loop
func (p *Probe) heartbeatLoop(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := p.client.Heartbeat(ctx); err != nil {
				log.Printf("Heartbeat failed: %v", err)
			}
		case <-p.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// collectAndSend collects data from all collectors and sends it to the server
func (p *Probe) collectAndSend(ctx context.Context) {
	report := &ReportData{
		Hostname: p.hostname,
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Collect host information
	if p.hostCollector != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hostInfo, err := p.hostCollector.Collect()
			if err != nil {
				log.Printf("Failed to collect host info: %v", err)
				return
			}
			mu.Lock()
			report.HostInfo = hostInfo
			mu.Unlock()
		}()
	}

	// Collect Docker information
	if p.dockerCollector != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dockerInfo, err := p.dockerCollector.Collect(ctx)
			if err != nil {
				log.Printf("Failed to collect Docker info: %v", err)
				return
			}
			mu.Lock()
			report.DockerInfo = dockerInfo
			mu.Unlock()
		}()
	}

	// Collect process information
	if p.processCollector != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			processInfo, err := p.processCollector.Collect()
			if err != nil {
				log.Printf("Failed to collect process info: %v", err)
				return
			}
			mu.Lock()
			report.ProcessesInfo = processInfo
			mu.Unlock()
		}()
	}

	// Collect network information
	if p.networkCollector != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			networkInfo, err := p.networkCollector.Collect()
			if err != nil {
				log.Printf("Failed to collect network info: %v", err)
				return
			}
			mu.Lock()
			report.NetworkInfo = networkInfo
			mu.Unlock()
		}()
	}

	// Wait for all collectors to finish
	wg.Wait()

	// Send report with retry
	if err := p.client.SendReportWithRetry(ctx, report, p.config.RetryAttempts, p.config.RetryDelay); err != nil {
		log.Printf("Failed to send report: %v", err)
	} else {
		log.Printf("Report sent successfully")
	}
}

// IsRunning returns whether the probe is running
func (p *Probe) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// GetConfig returns the probe configuration
func (p *Probe) GetConfig() ProbeConfig {
	return p.config
}
