package topology

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	manager := NewManager("test-manager")

	assert.Equal(t, "test-manager", manager.ID)
	assert.NotNil(t, manager.Nodes)
	assert.NotNil(t, manager.Edges)
	assert.NotNil(t, manager.Views)
	assert.NotNil(t, manager.Metrics)
	assert.NotNil(t, manager.Subscribers)
}

func TestAddNode(t *testing.T) {
	manager := NewManager("test-manager")

	node := &Node{
		ID:     "test-node",
		Type:   "host",
		Name:   "test-host",
		Status: "healthy",
		Metadata: map[string]interface{}{
			"hostname": "test-host",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		LastSeen:  time.Now(),
	}

	manager.AddNode(node)

	assert.Contains(t, manager.Nodes, "test-node")
	assert.Equal(t, node, manager.Nodes["test-node"])
}

func TestUpdateNode(t *testing.T) {
	manager := NewManager("test-manager")

	// Add initial node
	node := &Node{
		ID:     "test-node",
		Type:   "host",
		Name:   "test-host",
		Status: "healthy",
		Metadata: map[string]interface{}{
			"hostname": "test-host",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		LastSeen:  time.Now(),
	}

	manager.AddNode(node)

	// Update node
	updatedNode := &Node{
		ID:     "test-node",
		Type:   "host",
		Name:   "updated-host",
		Status: "warning",
		Metadata: map[string]interface{}{
			"hostname": "updated-host",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		LastSeen:  time.Now(),
	}

	manager.UpdateNode(updatedNode)

	assert.Contains(t, manager.Nodes, "test-node")
	assert.Equal(t, "updated-host", manager.Nodes["test-node"].Name)
	assert.Equal(t, "warning", manager.Nodes["test-node"].Status)
}

func TestRemoveNode(t *testing.T) {
	manager := NewManager("test-manager")

	// Add node
	node := &Node{
		ID:     "test-node",
		Type:   "host",
		Name:   "test-host",
		Status: "healthy",
		Metadata: map[string]interface{}{
			"hostname": "test-host",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		LastSeen:  time.Now(),
	}

	manager.AddNode(node)
	assert.Contains(t, manager.Nodes, "test-node")

	// Remove node
	manager.RemoveNode("test-node")
	assert.NotContains(t, manager.Nodes, "test-node")
}

func TestAddEdge(t *testing.T) {
	manager := NewManager("test-manager")

	edge := &Edge{
		ID:     "test-edge",
		Source: "node1",
		Target: "node2",
		Type:   "network",
		Weight: 1.0,
		Metadata: map[string]interface{}{
			"protocol": "tcp",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	manager.AddEdge(edge)

	assert.Contains(t, manager.Edges, "test-edge")
	assert.Equal(t, edge, manager.Edges["test-edge"])
}

func TestUpdateEdge(t *testing.T) {
	manager := NewManager("test-manager")

	// Add initial edge
	edge := &Edge{
		ID:     "test-edge",
		Source: "node1",
		Target: "node2",
		Type:   "network",
		Weight: 1.0,
		Metadata: map[string]interface{}{
			"protocol": "tcp",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	manager.AddEdge(edge)

	// Update edge
	updatedEdge := &Edge{
		ID:     "test-edge",
		Source: "node1",
		Target: "node2",
		Type:   "network",
		Weight: 2.0,
		Metadata: map[string]interface{}{
			"protocol": "udp",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	manager.UpdateEdge(updatedEdge)

	assert.Contains(t, manager.Edges, "test-edge")
	assert.Equal(t, 2.0, manager.Edges["test-edge"].Weight)
	assert.Equal(t, "udp", manager.Edges["test-edge"].Metadata["protocol"])
}

func TestRemoveEdge(t *testing.T) {
	manager := NewManager("test-manager")

	// Add edge
	edge := &Edge{
		ID:     "test-edge",
		Source: "node1",
		Target: "node2",
		Type:   "network",
		Weight: 1.0,
		Metadata: map[string]interface{}{
			"protocol": "tcp",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	manager.AddEdge(edge)
	assert.Contains(t, manager.Edges, "test-edge")

	// Remove edge
	manager.RemoveEdge("test-edge")
	assert.NotContains(t, manager.Edges, "test-edge")
}

func TestInitializeDefaultViews(t *testing.T) {
	manager := NewManager("test-manager")
	manager.initializeDefaultViews()

	// Check that default views are created
	assert.Contains(t, manager.Views, "processes")
	assert.Contains(t, manager.Views, "containers")
	assert.Contains(t, manager.Views, "hosts")
	assert.Contains(t, manager.Views, "pods")
	assert.Contains(t, manager.Views, "services")

	// Check view properties
	processesView := manager.Views["processes"]
	assert.Equal(t, "Processes", processesView.Name)
	assert.Contains(t, processesView.NodeTypes, "process")
	assert.Contains(t, processesView.NodeTypes, "container")
	assert.Contains(t, processesView.NodeTypes, "host")
}

func TestMatchesSearch(t *testing.T) {
	manager := NewManager("test-manager")

	node := &Node{
		ID:   "test-node",
		Name: "test-host",
		Metadata: map[string]interface{}{
			"hostname": "test-host.example.com",
			"ip":       "192.168.1.100",
		},
	}

	// Test name search
	assert.True(t, manager.matchesSearch(node, "test"))
	assert.True(t, manager.matchesSearch(node, "host"))
	assert.False(t, manager.matchesSearch(node, "notfound"))

	// Test metadata search
	assert.True(t, manager.matchesSearch(node, "example"))
	assert.True(t, manager.matchesSearch(node, "192.168"))
	assert.False(t, manager.matchesSearch(node, "10.0.0"))
}

func TestMatchesFilter(t *testing.T) {
	manager := NewManager("test-manager")

	node := &Node{
		ID:     "test-node",
		Type:   "host",
		Status: "healthy",
		Metadata: map[string]interface{}{
			"environment": "production",
			"region":      "us-west-2",
		},
		Metrics: &NodeMetrics{
			CPUUsage: &Sparkline{
				Current: 50.0,
			},
			MemoryUsage: &Sparkline{
				Current: 75.0,
			},
		},
	}

	// Test node type filter
	filter := &Filter{
		NodeTypes: []string{"host"},
	}
	assert.True(t, manager.matchesFilter(node, filter))

	filter.NodeTypes = []string{"container"}
	assert.False(t, manager.matchesFilter(node, filter))

	// Test status filter
	filter = &Filter{
		Status: []string{"healthy"},
	}
	assert.True(t, manager.matchesFilter(node, filter))

	filter.Status = []string{"critical"}
	assert.False(t, manager.matchesFilter(node, filter))

	// Test labels filter
	filter = &Filter{
		Labels: map[string]string{
			"environment": "production",
		},
	}
	assert.True(t, manager.matchesFilter(node, filter))

	filter.Labels = map[string]string{
		"environment": "development",
	}
	assert.False(t, manager.matchesFilter(node, filter))

	// Test metrics filter
	filter = &Filter{
		Metrics: &MetricsFilter{
			CPUUsage: &RangeFilter{
				Min: 40.0,
				Max: 60.0,
			},
		},
	}
	assert.True(t, manager.matchesFilter(node, filter))

	filter.Metrics.CPUUsage.Min = 60.0
	assert.False(t, manager.matchesFilter(node, filter))
}

func TestUpdateMetrics(t *testing.T) {
	manager := NewManager("test-manager")

	// Add some nodes with different statuses
	manager.AddNode(&Node{
		ID:     "node1",
		Status: "healthy",
	})
	manager.AddNode(&Node{
		ID:     "node2",
		Status: "warning",
	})
	manager.AddNode(&Node{
		ID:     "node3",
		Status: "critical",
	})
	manager.AddNode(&Node{
		ID:     "node4",
		Status: "unknown",
	})

	// Update metrics
	manager.updateMetrics()

	metrics := manager.getMetrics()
	assert.Equal(t, 4, metrics.TotalNodes)
	assert.Equal(t, 1, metrics.HealthyNodes)
	assert.Equal(t, 1, metrics.WarningNodes)
	assert.Equal(t, 1, metrics.CriticalNodes)
	assert.Equal(t, 1, metrics.UnknownNodes)
}
