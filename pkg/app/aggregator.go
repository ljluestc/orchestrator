package app

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ljluestc/orchestrator/pkg/probe"
)

// TopologyNode represents a node in the topology (host, container, or process)
type TopologyNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // "host", "container", "process"
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata"`
	ParentID string                 `json:"parent_id,omitempty"`
}

// TopologyEdge represents a connection between nodes
type TopologyEdge struct {
	Source      string                 `json:"source"`
	Target      string                 `json:"target"`
	Type        string                 `json:"type"` // "network", "parent-child"
	Protocol    string                 `json:"protocol,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Connections int                    `json:"connections,omitempty"`
}

// TopologyView represents the aggregated topology view
type TopologyView struct {
	Nodes     map[string]*TopologyNode `json:"nodes"`
	Edges     map[string]*TopologyEdge `json:"edges"`
	Timestamp time.Time                `json:"timestamp"`
	mu        sync.RWMutex
}

// Aggregator handles report aggregation and topology building
type Aggregator struct {
	topology *TopologyView
	mu       sync.RWMutex
}

// NewAggregator creates a new report aggregator
func NewAggregator() *Aggregator {
	return &Aggregator{
		topology: &TopologyView{
			Nodes:     make(map[string]*TopologyNode),
			Edges:     make(map[string]*TopologyEdge),
			Timestamp: time.Now(),
		},
	}
}

// ProcessReport processes a report and updates the topology
func (a *Aggregator) ProcessReport(report *probe.ReportData) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.topology.mu.Lock()
	defer a.topology.mu.Unlock()

	// Create or update host node
	hostID := report.AgentID
	a.topology.Nodes[hostID] = &TopologyNode{
		ID:   hostID,
		Type: "host",
		Name: report.Hostname,
		Metadata: map[string]interface{}{
			"agent_id":  report.AgentID,
			"hostname":  report.Hostname,
			"timestamp": report.Timestamp,
		},
	}

	// Add host info metadata
	if report.HostInfo != nil {
		a.topology.Nodes[hostID].Metadata["kernel"] = report.HostInfo.KernelVersion
		a.topology.Nodes[hostID].Metadata["cpu_cores"] = report.HostInfo.CPUInfo.Cores
		a.topology.Nodes[hostID].Metadata["cpu_model"] = report.HostInfo.CPUInfo.Model
		a.topology.Nodes[hostID].Metadata["cpu_usage"] = report.HostInfo.CPUInfo.Usage
		a.topology.Nodes[hostID].Metadata["memory_total"] = report.HostInfo.MemoryInfo.TotalMB
		a.topology.Nodes[hostID].Metadata["memory_used"] = report.HostInfo.MemoryInfo.UsedMB
		a.topology.Nodes[hostID].Metadata["memory_usage"] = report.HostInfo.MemoryInfo.Usage
	}

	// Process Docker containers
	if report.DockerInfo != nil && report.DockerInfo.Containers != nil {
		for _, container := range report.DockerInfo.Containers {
			containerID := container.ID
			a.topology.Nodes[containerID] = &TopologyNode{
				ID:       containerID,
				Type:     "container",
				Name:     container.Name,
				ParentID: hostID,
				Metadata: map[string]interface{}{
					"container_id": container.ID,
					"image":        container.Image,
					"status":       container.Status,
					"created":      container.Created,
				},
			}

			// Add container stats if available
			if container.Stats != nil {
				a.topology.Nodes[containerID].Metadata["cpu_usage"] = container.Stats.CPUPercent
				a.topology.Nodes[containerID].Metadata["memory_usage"] = container.Stats.MemoryUsageMB
				a.topology.Nodes[containerID].Metadata["memory_limit"] = container.Stats.MemoryLimitMB
				a.topology.Nodes[containerID].Metadata["memory_percent"] = container.Stats.MemoryPercent
			}

			// Create parent-child edge
			edgeID := hostID + "->" + containerID
			a.topology.Edges[edgeID] = &TopologyEdge{
				Source: hostID,
				Target: containerID,
				Type:   "parent-child",
			}
		}
	}

	// Process processes
	if report.ProcessesInfo != nil && report.ProcessesInfo.Processes != nil {
		for _, process := range report.ProcessesInfo.Processes {
			processID := report.AgentID + "-" + fmt.Sprintf("%d", process.PID)
			parentID := hostID

			// If process has a cgroup (container), extract container ID as parent
			if process.Cgroup != "" {
				// Try to extract container ID from cgroup
				if containerID := extractContainerIDFromCgroup(process.Cgroup); containerID != "" {
					parentID = containerID
				}
			}

			a.topology.Nodes[processID] = &TopologyNode{
				ID:       processID,
				Type:     "process",
				Name:     process.Name,
				ParentID: parentID,
				Metadata: map[string]interface{}{
					"pid":        process.PID,
					"ppid":       process.PPID,
					"cmdline":    process.Cmdline,
					"state":      process.State,
					"threads":    process.Threads,
					"cpu_time":   process.CPUTime,
					"memory_mb":  process.MemoryMB,
					"open_files": process.OpenFiles,
					"cgroup":     process.Cgroup,
				},
			}

			// Create parent-child edge
			edgeID := parentID + "->" + processID
			a.topology.Edges[edgeID] = &TopologyEdge{
				Source: parentID,
				Target: processID,
				Type:   "parent-child",
			}
		}
	}

	// Process network connections
	if report.NetworkInfo != nil && report.NetworkInfo.Connections != nil {
		connectionMap := make(map[string]int) // Track connection counts

		for _, conn := range report.NetworkInfo.Connections {
			// Create edge key for network connection
			sourceNode := report.AgentID + "-" + fmt.Sprintf("%d", conn.PID)
			targetKey := conn.RemoteAddr + ":" + fmt.Sprintf("%d", conn.RemotePort)

			// For external connections, we might not have the target node
			// We can create a placeholder or skip
			edgeKey := sourceNode + "->" + targetKey + "-" + conn.Protocol

			// Count connections
			connectionMap[edgeKey]++

			// Create or update edge
			if edge, exists := a.topology.Edges[edgeKey]; exists {
				edge.Connections++
			} else {
				a.topology.Edges[edgeKey] = &TopologyEdge{
					Source:   sourceNode,
					Target:   targetKey,
					Type:     "network",
					Protocol: conn.Protocol,
					Metadata: map[string]interface{}{
						"local_addr":   conn.LocalAddr,
						"local_port":   conn.LocalPort,
						"remote_addr":  conn.RemoteAddr,
						"remote_port":  conn.RemotePort,
						"state":        conn.State,
						"process_name": conn.ProcessName,
					},
					Connections: 1,
				}
			}
		}
	}

	a.topology.Timestamp = time.Now()
}

// GetTopology returns the current topology view
func (a *Aggregator) GetTopology() *TopologyView {
	a.mu.RLock()
	defer a.mu.RUnlock()

	a.topology.mu.RLock()
	defer a.topology.mu.RUnlock()

	// Create a deep copy to avoid race conditions
	nodes := make(map[string]*TopologyNode)
	for k, v := range a.topology.Nodes {
		// Copy metadata
		metadata := make(map[string]interface{})
		for mk, mv := range v.Metadata {
			metadata[mk] = mv
		}

		nodes[k] = &TopologyNode{
			ID:       v.ID,
			Type:     v.Type,
			Name:     v.Name,
			Metadata: metadata,
			ParentID: v.ParentID,
		}
	}

	edges := make(map[string]*TopologyEdge)
	for k, v := range a.topology.Edges {
		// Copy metadata
		var metadata map[string]interface{}
		if v.Metadata != nil {
			metadata = make(map[string]interface{})
			for mk, mv := range v.Metadata {
				metadata[mk] = mv
			}
		}

		edges[k] = &TopologyEdge{
			Source:      v.Source,
			Target:      v.Target,
			Type:        v.Type,
			Protocol:    v.Protocol,
			Metadata:    metadata,
			Connections: v.Connections,
		}
	}

	return &TopologyView{
		Nodes:     nodes,
		Edges:     edges,
		Timestamp: a.topology.Timestamp,
	}
}

// GetNodesByType returns all nodes of a specific type
func (a *Aggregator) GetNodesByType(nodeType string) []*TopologyNode {
	a.mu.RLock()
	defer a.mu.RUnlock()

	a.topology.mu.RLock()
	defer a.topology.mu.RUnlock()

	nodes := make([]*TopologyNode, 0)
	for _, node := range a.topology.Nodes {
		if node.Type == nodeType {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

// GetNodeByID returns a specific node by ID
func (a *Aggregator) GetNodeByID(nodeID string) *TopologyNode {
	a.mu.RLock()
	defer a.mu.RUnlock()

	a.topology.mu.RLock()
	defer a.topology.mu.RUnlock()

	return a.topology.Nodes[nodeID]
}

// CleanStaleNodes removes nodes that haven't been updated in the specified duration
func (a *Aggregator) CleanStaleNodes(maxAge time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.topology.mu.Lock()
	defer a.topology.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	staleNodes := make([]string, 0)

	for nodeID, node := range a.topology.Nodes {
		if timestamp, ok := node.Metadata["timestamp"].(time.Time); ok {
			if timestamp.Before(cutoff) {
				staleNodes = append(staleNodes, nodeID)
			}
		}
	}

	// Remove stale nodes and their edges
	for _, nodeID := range staleNodes {
		delete(a.topology.Nodes, nodeID)

		// Remove edges connected to this node
		edgesToRemove := make([]string, 0)
		for edgeID, edge := range a.topology.Edges {
			if edge.Source == nodeID || edge.Target == nodeID {
				edgesToRemove = append(edgesToRemove, edgeID)
			}
		}

		for _, edgeID := range edgesToRemove {
			delete(a.topology.Edges, edgeID)
		}
	}
}

// GetStats returns statistics about the topology
func (a *Aggregator) GetStats() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	a.topology.mu.RLock()
	defer a.topology.mu.RUnlock()

	nodesByType := make(map[string]int)
	for _, node := range a.topology.Nodes {
		nodesByType[node.Type]++
	}

	edgesByType := make(map[string]int)
	for _, edge := range a.topology.Edges {
		edgesByType[edge.Type]++
	}

	return map[string]interface{}{
		"total_nodes":   len(a.topology.Nodes),
		"total_edges":   len(a.topology.Edges),
		"nodes_by_type": nodesByType,
		"edges_by_type": edgesByType,
		"last_update":   a.topology.Timestamp,
	}
}

// extractContainerIDFromCgroup extracts container ID from cgroup path
func extractContainerIDFromCgroup(cgroup string) string {
	// Cgroup path usually looks like: /docker/<container-id> or /kubepods/.../docker-<container-id>.scope
	if strings.Contains(cgroup, "docker") {
		parts := strings.Split(cgroup, "/")
		for _, part := range parts {
			if strings.HasPrefix(part, "docker-") {
				// Remove "docker-" prefix and ".scope" suffix if present
				id := strings.TrimPrefix(part, "docker-")
				id = strings.TrimSuffix(id, ".scope")
				if len(id) >= 12 {
					return id[:12] // Return first 12 chars as Docker short ID
				}
			} else if len(part) >= 12 && !strings.Contains(part, ".") {
				// Might be a raw container ID
				return part[:12]
			}
		}
	}
	return ""
}
