package topology

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	// Fetch topology data from the app server
	if err := c.fetchTopologyFromAppServer(); err != nil {
		log.Printf("Failed to fetch topology from app server: %v", err)
		// Fall back to mock data if app server is not available
		c.generateMockData()
	}
}

// fetchTopologyFromAppServer fetches topology data from the app server
func (c *Collector) fetchTopologyFromAppServer() error {
	// App server runs on port 8080
	appServerURL := "http://localhost:8080"
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	resp, err := client.Get(appServerURL + "/api/v1/query/topology")
	if err != nil {
		return fmt.Errorf("failed to fetch topology from app server: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("app server returned status %d", resp.StatusCode)
	}
	
	var topologyResponse struct {
		Topology struct {
			Nodes map[string]interface{} `json:"nodes"`
			Edges map[string]interface{} `json:"edges"`
		} `json:"topology"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&topologyResponse); err != nil {
		return fmt.Errorf("failed to decode topology response: %w", err)
	}
	
	// Process nodes
	for nodeID, nodeData := range topologyResponse.Topology.Nodes {
		if node, err := c.convertToNode(nodeID, nodeData); err == nil {
			c.Manager.AddNode(node)
		} else {
			log.Printf("Failed to convert node %s: %v", nodeID, err)
		}
	}
	
	// Process edges
	for edgeID, edgeData := range topologyResponse.Topology.Edges {
		if edge, err := c.convertToEdge(edgeID, edgeData); err == nil {
			c.Manager.AddEdge(edge)
		} else {
			log.Printf("Failed to convert edge %s: %v", edgeID, err)
		}
	}
	
	log.Printf("Successfully fetched topology from app server: %d nodes, %d edges", 
		len(topologyResponse.Topology.Nodes), len(topologyResponse.Topology.Edges))
	
	return nil
}

// convertToNode converts app server node data to topology manager node
func (c *Collector) convertToNode(nodeID string, nodeData interface{}) (*Node, error) {
	nodeMap, ok := nodeData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid node data format")
	}
	
	node := &Node{
		ID:   nodeID,
		Type: getString(nodeMap, "type"),
		Name: getString(nodeMap, "name"),
		Metadata: map[string]interface{}{
			"source": "app_server",
		},
	}
	
	// Copy metadata from app server
	if metadata, ok := nodeMap["metadata"]; ok {
		if metadataMap, ok := metadata.(map[string]interface{}); ok {
			for k, v := range metadataMap {
				node.Metadata[k] = v
			}
		}
	}
	
	// Set parent ID if available
	if parentID, ok := nodeMap["parent_id"]; ok {
		if parentStr, ok := parentID.(string); ok {
			node.Metadata["parent_id"] = parentStr
		}
	}
	
	return node, nil
}

// convertToEdge converts app server edge data to topology manager edge
func (c *Collector) convertToEdge(edgeID string, edgeData interface{}) (*Edge, error) {
	edgeMap, ok := edgeData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid edge data format")
	}
	
	edge := &Edge{
		ID:     edgeID,
		Source: getString(edgeMap, "source"),
		Target: getString(edgeMap, "target"),
		Type:   getString(edgeMap, "type"),
		Metadata: map[string]interface{}{
			"source": "app_server",
		},
	}
	
	// Copy metadata from app server
	if metadata, ok := edgeMap["metadata"]; ok {
		if metadataMap, ok := metadata.(map[string]interface{}); ok {
			for k, v := range metadataMap {
				edge.Metadata[k] = v
			}
		}
	}
	
	// Set protocol if available
	if protocol, ok := edgeMap["protocol"]; ok {
		if protocolStr, ok := protocol.(string); ok {
			edge.Metadata["protocol"] = protocolStr
		}
	}
	
	// Set connections count if available
	if connections, ok := edgeMap["connections"]; ok {
		if connFloat, ok := connections.(float64); ok {
			edge.Metadata["connections"] = int(connFloat)
		}
	}
	
	return edge, nil
}

// getString safely extracts a string value from a map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
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
