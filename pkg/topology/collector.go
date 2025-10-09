package topology

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ljluestc/orchestrator/pkg/probe"
)

// Collector collects data from probe agents and feeds it to the topology manager
type Collector struct {
	ID            string
	TopologyURL   string
	ProbeClient   *probe.Client
	Manager       *Manager
	UpdateTicker  *time.Ticker
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewCollector creates a new topology collector
func NewCollector(id, topologyURL string, probeClient *probe.Client) *Collector {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Collector{
		ID:          id,
		TopologyURL: topologyURL,
		ProbeClient: probeClient,
		UpdateTicker: time.NewTicker(30 * time.Second),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start starts the collector
func (c *Collector) Start() error {
	log.Printf("Starting topology collector %s", c.ID)
	
	// Start collecting data from probe agents
	go c.collectFromProbes()
	
	// Start processing collected data
	go c.processCollectedData()
	
	return nil
}

// Stop stops the collector
func (c *Collector) Stop() error {
	log.Printf("Stopping topology collector %s", c.ID)
	
	c.cancel()
	c.UpdateTicker.Stop()
	
	return nil
}

// collectFromProbes collects data from probe agents
func (c *Collector) collectFromProbes() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.UpdateTicker.C:
			c.collectProbeData()
		}
	}
}

// collectProbeData collects data from all probe agents
func (c *Collector) collectProbeData() {
	// For now, generate mock data since we don't have a direct probe client
	// In a real implementation, this would collect from actual probe agents
	c.generateMockData()
}

// collectAgentData collects data from a specific agent
func (c *Collector) collectAgentData(agentID string) {
	// For now, generate mock data
	// In a real implementation, this would collect from the specific agent
	c.generateMockData()
}

// processReport processes a probe report and updates the topology
func (c *Collector) processReport(report interface{}) {
	// For now, just generate mock data
	// In a real implementation, this would process actual probe reports
	c.generateMockData()
}

// createHostNode creates a host node from a probe report
func (c *Collector) createHostNode(report interface{}) *Node {
	// For now, return nil since we're using mock data
	// In a real implementation, this would process actual probe reports
	return nil
}

// createContainerNode creates a container node from container info
func (c *Collector) createContainerNode(agentID string, container interface{}) *Node {
	// For now, return nil since we're using mock data
	// In a real implementation, this would process actual container info
	return nil
}

// createProcessNode creates a process node from process info
func (c *Collector) createProcessNode(agentID string, process interface{}) *Node {
	// For now, return nil since we're using mock data
	// In a real implementation, this would process actual process info
	return nil
}

// createNetworkEdge creates a network edge from connection info
func (c *Collector) createNetworkEdge(agentID string, connection interface{}) *Edge {
	// For now, return nil since we're using mock data
	// In a real implementation, this would process actual network connections
	return nil
}

// processCollectedData processes collected data and updates topology
func (c *Collector) processCollectedData() {
	// This would be called periodically to process and update the topology
	// For now, it's handled in the collectProbeData method
}

// Mock data generation for testing
func (c *Collector) generateMockData() {
	// Generate mock host nodes
	for i := 0; i < 3; i++ {
		hostNode := &Node{
			ID:     fmt.Sprintf("host-%d", i),
			Type:   "host",
			Name:   fmt.Sprintf("host-%d.example.com", i),
			Status: "healthy",
			Metadata: map[string]interface{}{
				"hostname": fmt.Sprintf("host-%d.example.com", i),
				"kernel":   "Linux 5.4.0",
				"cpu_cores": 8,
				"memory_total": 16384,
			},
			Metrics: &NodeMetrics{
				CPUUsage: &Sparkline{
					Values:  []float64{20.0, 25.0, 30.0, 35.0, 40.0},
					Times:   []int64{time.Now().Unix() - 300, time.Now().Unix() - 240, time.Now().Unix() - 180, time.Now().Unix() - 120, time.Now().Unix() - 60},
					Min:     10.0,
					Max:     50.0,
					Avg:     30.0,
					Current: 40.0,
				},
				MemoryUsage: &Sparkline{
					Values:  []float64{60.0, 65.0, 70.0, 75.0, 80.0},
					Times:   []int64{time.Now().Unix() - 300, time.Now().Unix() - 240, time.Now().Unix() - 180, time.Now().Unix() - 120, time.Now().Unix() - 60},
					Min:     50.0,
					Max:     90.0,
					Avg:     70.0,
					Current: 80.0,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			LastSeen:  time.Now(),
		}
		c.Manager.AddNode(hostNode)
	}
	
	// Generate mock container nodes
	for i := 0; i < 10; i++ {
		containerNode := &Node{
			ID:     fmt.Sprintf("container-%d", i),
			Type:   "container",
			Name:   fmt.Sprintf("app-%d", i),
			Status: "healthy",
			Metadata: map[string]interface{}{
				"image":   "nginx:latest",
				"state":   "running",
				"ports":   []int{80, 443},
				"labels":  map[string]string{"app": "web", "version": "1.0"},
			},
			Metrics: &NodeMetrics{
				CPUUsage: &Sparkline{
					Values:  []float64{5.0, 10.0, 15.0, 20.0, 25.0},
					Times:   []int64{time.Now().Unix() - 300, time.Now().Unix() - 240, time.Now().Unix() - 180, time.Now().Unix() - 120, time.Now().Unix() - 60},
					Min:     0.0,
					Max:     50.0,
					Avg:     15.0,
					Current: 25.0,
				},
				MemoryUsage: &Sparkline{
					Values:  []float64{30.0, 35.0, 40.0, 45.0, 50.0},
					Times:   []int64{time.Now().Unix() - 300, time.Now().Unix() - 240, time.Now().Unix() - 180, time.Now().Unix() - 120, time.Now().Unix() - 60},
					Min:     20.0,
					Max:     80.0,
					Avg:     40.0,
					Current: 50.0,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			LastSeen:  time.Now(),
		}
		c.Manager.AddNode(containerNode)
		
		// Create edge from host to container
		edge := &Edge{
			ID:     fmt.Sprintf("host-%d-container-%d", i%3, i),
			Source: fmt.Sprintf("host-%d", i%3),
			Target: fmt.Sprintf("container-%d", i),
			Type:   "container",
			Weight: 1.0,
			Metadata: map[string]interface{}{
				"container_name": fmt.Sprintf("app-%d", i),
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		c.Manager.AddEdge(edge)
	}
	
	// Generate mock process nodes
	for i := 0; i < 20; i++ {
		processNode := &Node{
			ID:     fmt.Sprintf("process-%d", i),
			Type:   "process",
			Name:   fmt.Sprintf("process-%d", i),
			Status: "healthy",
			Metadata: map[string]interface{}{
				"pid":     i + 1000,
				"cmdline": fmt.Sprintf("/usr/bin/process-%d", i),
				"state":   "R",
				"user":    1000,
			},
			Metrics: &NodeMetrics{
				CPUUsage: &Sparkline{
					Values:  []float64{1.0, 2.0, 3.0, 4.0, 5.0},
					Times:   []int64{time.Now().Unix() - 300, time.Now().Unix() - 240, time.Now().Unix() - 180, time.Now().Unix() - 120, time.Now().Unix() - 60},
					Min:     0.0,
					Max:     10.0,
					Avg:     3.0,
					Current: 5.0,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			LastSeen:  time.Now(),
		}
		c.Manager.AddNode(processNode)
		
		// Create edge from host to process
		edge := &Edge{
			ID:     fmt.Sprintf("host-%d-process-%d", i%3, i),
			Source: fmt.Sprintf("host-%d", i%3),
			Target: fmt.Sprintf("process-%d", i),
			Type:   "process",
			Weight: 1.0,
			Metadata: map[string]interface{}{
				"process_name": fmt.Sprintf("process-%d", i),
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		c.Manager.AddEdge(edge)
	}
}
