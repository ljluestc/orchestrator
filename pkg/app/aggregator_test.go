package app

import (
	"testing"
	"time"

	"github.com/ljluestc/orchestrator/pkg/probe"
	"github.com/stretchr/testify/assert"
)

func TestNewAggregator(t *testing.T) {
	agg := NewAggregator()
	assert.NotNil(t, agg)
	assert.NotNil(t, agg.topology)
	assert.NotNil(t, agg.topology.Nodes)
	assert.NotNil(t, agg.topology.Edges)
}

func TestProcessReportHostOnly(t *testing.T) {
	agg := NewAggregator()

	report := &probe.ReportData{
		AgentID:   "test-agent-1",
		Hostname:  "test-host",
		Timestamp: time.Now(),
		HostInfo: &probe.HostInfo{
			Hostname:      "test-host",
			KernelVersion: "5.10.0",
			CPUInfo: probe.CPUInfo{
				Model: "Intel Core i7",
				Cores: 8,
				Usage: 25.5,
			},
			MemoryInfo: probe.MemoryInfo{
				TotalMB:     16384,
				UsedMB:      8192,
				FreeMB:      8192,
				AvailableMB: 8192,
				Usage:       50.0,
			},
		},
	}

	agg.ProcessReport(report)

	// Verify host node was created
	topology := agg.GetTopology()
	assert.Contains(t, topology.Nodes, "test-agent-1")

	hostNode := topology.Nodes["test-agent-1"]
	assert.Equal(t, "host", hostNode.Type)
	assert.Equal(t, "test-host", hostNode.Name)
	assert.Equal(t, 8, hostNode.Metadata["cpu_cores"])
	assert.Equal(t, "Intel Core i7", hostNode.Metadata["cpu_model"])
	assert.Equal(t, uint64(16384), hostNode.Metadata["memory_total"])
}

func TestProcessReportWithContainers(t *testing.T) {
	agg := NewAggregator()

	report := &probe.ReportData{
		AgentID:   "test-agent-1",
		Hostname:  "test-host",
		Timestamp: time.Now(),
		DockerInfo: &probe.DockerInfo{
			Containers: []probe.ContainerInfo{
				{
					ID:      "container-1",
					Name:    "test-container-1",
					Image:   "nginx:latest",
					Status:  "running",
					State:   "running",
					Created: time.Now().Add(-1 * time.Hour),
					Stats: &probe.ContainerStats{
						CPUPercent:    15.5,
						MemoryUsageMB: 256,
						MemoryLimitMB: 512,
						MemoryPercent: 50.0,
					},
				},
			},
		},
	}

	agg.ProcessReport(report)

	topology := agg.GetTopology()

	// Verify container node was created
	assert.Contains(t, topology.Nodes, "container-1")
	containerNode := topology.Nodes["container-1"]
	assert.Equal(t, "container", containerNode.Type)
	assert.Equal(t, "test-container-1", containerNode.Name)
	assert.Equal(t, "test-agent-1", containerNode.ParentID)

	// Verify container stats
	assert.Equal(t, 15.5, containerNode.Metadata["cpu_usage"])
	assert.Equal(t, uint64(256), containerNode.Metadata["memory_usage"])
	assert.Equal(t, uint64(512), containerNode.Metadata["memory_limit"])

	// Verify parent-child edge was created
	edgeKey := "test-agent-1->container-1"
	assert.Contains(t, topology.Edges, edgeKey)
	edge := topology.Edges[edgeKey]
	assert.Equal(t, "parent-child", edge.Type)
	assert.Equal(t, "test-agent-1", edge.Source)
	assert.Equal(t, "container-1", edge.Target)
}

func TestProcessReportWithProcesses(t *testing.T) {
	agg := NewAggregator()

	report := &probe.ReportData{
		AgentID:   "test-agent-1",
		Hostname:  "test-host",
		Timestamp: time.Now(),
		ProcessesInfo: &probe.ProcessesInfo{
			Processes: []probe.ProcessInfo{
				{
					PID:      1234,
					Name:     "nginx",
					Cmdline:  "nginx -g daemon off;",
					State:    "R",
					PPID:     1,
					UID:      0,
					GID:      0,
					Threads:  4,
					CPUTime:  1000,
					MemoryMB: 128,
					Cgroup:   "/docker/abc123def456",
				},
			},
		},
	}

	agg.ProcessReport(report)

	topology := agg.GetTopology()

	// Verify process node was created
	processID := "test-agent-1-1234"
	assert.Contains(t, topology.Nodes, processID)
	processNode := topology.Nodes[processID]
	assert.Equal(t, "process", processNode.Type)
	assert.Equal(t, "nginx", processNode.Name)

	// Verify process metadata
	assert.Equal(t, 1234, processNode.Metadata["pid"])
	assert.Equal(t, 1, processNode.Metadata["ppid"])
	assert.Equal(t, "nginx -g daemon off;", processNode.Metadata["cmdline"])
	assert.Equal(t, "R", processNode.Metadata["state"])
	assert.Equal(t, 4, processNode.Metadata["threads"])

	// Verify parent-child edge was created
	edgeKey := processNode.ParentID + "->" + processID
	assert.Contains(t, topology.Edges, edgeKey)
}

func TestProcessReportWithNetworkConnections(t *testing.T) {
	agg := NewAggregator()

	report := &probe.ReportData{
		AgentID:   "test-agent-1",
		Hostname:  "test-host",
		Timestamp: time.Now(),
		NetworkInfo: &probe.NetworkInfo{
			Connections: []probe.NetworkConnection{
				{
					PID:         1234,
					LocalAddr:   "192.168.1.10",
					LocalPort:   8080,
					RemoteAddr:  "192.168.1.20",
					RemotePort:  443,
					Protocol:    "tcp",
					State:       "ESTABLISHED",
					ProcessName: "nginx",
				},
			},
		},
	}

	agg.ProcessReport(report)

	topology := agg.GetTopology()

	// Find network edge
	found := false
	for edgeID, edge := range topology.Edges {
		if edge.Type == "network" && edge.Protocol == "tcp" {
			found = true
			assert.Equal(t, "test-agent-1-1234", edge.Source)
			assert.Contains(t, edgeID, "192.168.1.20:443")
			assert.Equal(t, 1, edge.Connections)
			assert.Equal(t, "nginx", edge.Metadata["process_name"])
			break
		}
	}
	assert.True(t, found, "Network edge not found")
}

func TestGetNodesByType(t *testing.T) {
	agg := NewAggregator()

	// Add multiple reports
	for i := 0; i < 3; i++ {
		report := &probe.ReportData{
			AgentID:   "agent-" + string(rune('1'+i)),
			Hostname:  "host-" + string(rune('1'+i)),
			Timestamp: time.Now(),
		}
		agg.ProcessReport(report)
	}

	hostNodes := agg.GetNodesByType("host")
	assert.Len(t, hostNodes, 3)

	for _, node := range hostNodes {
		assert.Equal(t, "host", node.Type)
	}
}

func TestGetNodeByID(t *testing.T) {
	agg := NewAggregator()

	report := &probe.ReportData{
		AgentID:   "test-agent-1",
		Hostname:  "test-host",
		Timestamp: time.Now(),
	}
	agg.ProcessReport(report)

	node := agg.GetNodeByID("test-agent-1")
	assert.NotNil(t, node)
	assert.Equal(t, "test-agent-1", node.ID)
	assert.Equal(t, "host", node.Type)

	nonExistentNode := agg.GetNodeByID("non-existent")
	assert.Nil(t, nonExistentNode)
}

func TestCleanStaleNodes(t *testing.T) {
	agg := NewAggregator()

	// Add old report
	oldReport := &probe.ReportData{
		AgentID:   "old-agent",
		Hostname:  "old-host",
		Timestamp: time.Now().Add(-10 * time.Minute),
	}
	agg.ProcessReport(oldReport)

	// Add recent report
	recentReport := &probe.ReportData{
		AgentID:   "recent-agent",
		Hostname:  "recent-host",
		Timestamp: time.Now(),
	}
	agg.ProcessReport(recentReport)

	// Clean nodes older than 5 minutes
	agg.CleanStaleNodes(5 * time.Minute)

	topology := agg.GetTopology()

	// Old node should be removed
	assert.NotContains(t, topology.Nodes, "old-agent")

	// Recent node should still exist
	assert.Contains(t, topology.Nodes, "recent-agent")
}

func TestAggregatorGetStats(t *testing.T) {
	agg := NewAggregator()

	// Add test data
	report := &probe.ReportData{
		AgentID:   "test-agent-1",
		Hostname:  "test-host",
		Timestamp: time.Now(),
		DockerInfo: &probe.DockerInfo{
			Containers: []probe.ContainerInfo{
				{
					ID:    "container-1",
					Name:  "test-container",
					Image: "nginx:latest",
					State: "running",
				},
			},
		},
		ProcessesInfo: &probe.ProcessesInfo{
			Processes: []probe.ProcessInfo{
				{
					PID:  1234,
					Name: "nginx",
				},
			},
		},
	}
	agg.ProcessReport(report)

	stats := agg.GetStats()

	assert.Contains(t, stats, "total_nodes")
	assert.Contains(t, stats, "total_edges")
	assert.Contains(t, stats, "nodes_by_type")
	assert.Contains(t, stats, "edges_by_type")
	assert.Contains(t, stats, "last_update")

	// We should have: 1 host + 1 container + 1 process = 3 nodes
	assert.Equal(t, 3, stats["total_nodes"])
}

func TestExtractContainerIDFromCgroup(t *testing.T) {
	tests := []struct {
		name     string
		cgroup   string
		expected string
	}{
		{
			name:     "Docker cgroup with full ID",
			cgroup:   "/docker/abc123def456ghi789jkl012mno345pqr678stu901vwx234yz",
			expected: "abc123def456",
		},
		{
			name:     "Docker cgroup with prefix",
			cgroup:   "/system.slice/docker-abc123def456ghi789jkl012mno345pqr678stu901vwx234yz.scope",
			expected: "abc123def456",
		},
		{
			name:     "No docker in cgroup",
			cgroup:   "/system.slice/some-service.service",
			expected: "",
		},
		{
			name:     "Empty cgroup",
			cgroup:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractContainerIDFromCgroup(tt.cgroup)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTopologyConcurrency(t *testing.T) {
	agg := NewAggregator()

	// Simulate concurrent report processing
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			report := &probe.ReportData{
				AgentID:   "agent-" + string(rune('0'+id)),
				Hostname:  "host-" + string(rune('0'+id)),
				Timestamp: time.Now(),
			}
			agg.ProcessReport(report)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all nodes were created
	topology := agg.GetTopology()
	assert.GreaterOrEqual(t, len(topology.Nodes), 10)
}
