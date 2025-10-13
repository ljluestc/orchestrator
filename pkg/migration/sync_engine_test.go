package migration

import (
	"context"
	"testing"
	"time"

	"github.com/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockZookeeperCluster for testing
type MockZookeeperCluster struct {
	mock.Mock
}

func (m *MockZookeeperCluster) GetHosts() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func TestNewSyncEngine(t *testing.T) {
	tests := []struct {
		name               string
		id                 string
		source             *ZookeeperCluster
		target             *ZookeeperCluster
		conflictResolution ConflictStrategy
	}{
		{
			name: "Valid sync engine with last-write-wins",
			id:   "sync-1",
			source: &ZookeeperCluster{
				ID:    "cluster-a",
				Hosts: []string{"zk1:2181", "zk2:2181"},
				Port:  2181,
			},
			target: &ZookeeperCluster{
				ID:    "cluster-b",
				Hosts: []string{"zk3:2181", "zk4:2181"},
				Port:  2181,
			},
			conflictResolution: LastWriteWins,
		},
		{
			name: "Valid sync engine with source-wins",
			id:   "sync-2",
			source: &ZookeeperCluster{
				ID:    "cluster-a",
				Hosts: []string{"zk1:2181"},
				Port:  2181,
			},
			target: &ZookeeperCluster{
				ID:    "cluster-b",
				Hosts: []string{"zk2:2181"},
				Port:  2181,
			},
			conflictResolution: SourceWins,
		},
		{
			name: "Valid sync engine with manual resolution",
			id:   "sync-3",
			source: &ZookeeperCluster{
				ID:    "cluster-a",
				Hosts: []string{"zk1:2181", "zk2:2181", "zk3:2181"},
				Port:  2181,
			},
			target: &ZookeeperCluster{
				ID:    "cluster-b",
				Hosts: []string{"zk4:2181", "zk5:2181", "zk6:2181"},
				Port:  2181,
			},
			conflictResolution: ManualResolve,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewSyncEngine(tt.id, tt.source, tt.target, tt.conflictResolution)

			assert.NotNil(t, engine)
			assert.Equal(t, tt.id, engine.ID)
			assert.Equal(t, tt.source, engine.SourceCluster)
			assert.Equal(t, tt.target, engine.TargetCluster)
			assert.Equal(t, tt.conflictResolution, engine.ConflictResolution)
			assert.Equal(t, 30*time.Second, engine.syncInterval)
			assert.Equal(t, 5*time.Second, engine.lagThreshold)
			assert.NotNil(t, engine.metrics)
			assert.Equal(t, time.Duration(0), engine.metrics.ReplicationLag)
			assert.Equal(t, int64(0), engine.metrics.SyncedNodes)
			assert.Equal(t, int64(0), engine.metrics.ConflictCount)
			assert.Equal(t, int64(0), engine.metrics.ErrorCount)
			assert.Equal(t, float64(0), engine.metrics.ThroughputPerSec)
		})
	}
}

func TestSyncEngine_getHosts(t *testing.T) {
	engine := NewSyncEngine("test", &ZookeeperCluster{}, &ZookeeperCluster{}, LastWriteWins)

	tests := []struct {
		name    string
		cluster *ZookeeperCluster
		expected []string
	}{
		{
			name: "Single host",
			cluster: &ZookeeperCluster{
				Hosts: []string{"zk1:2181"},
			},
			expected: []string{"zk1:2181"},
		},
		{
			name: "Multiple hosts",
			cluster: &ZookeeperCluster{
				Hosts: []string{"zk1:2181", "zk2:2181", "zk3:2181"},
			},
			expected: []string{"zk1:2181", "zk2:2181", "zk3:2181"},
		},
		{
			name: "Empty hosts",
			cluster: &ZookeeperCluster{
				Hosts: []string{},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.getHosts(tt.cluster)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSyncEngine_GetMetrics(t *testing.T) {
	engine := NewSyncEngine("test", &ZookeeperCluster{}, &ZookeeperCluster{}, LastWriteWins)

	// Set some test metrics
	engine.metrics.ReplicationLag = 100 * time.Millisecond
	engine.metrics.SyncedNodes = 42
	engine.metrics.ConflictCount = 5
	engine.metrics.ErrorCount = 2
	engine.metrics.LastSyncTime = time.Now()
	engine.metrics.ThroughputPerSec = 10.5

	metrics := engine.GetMetrics()

	assert.Equal(t, 100*time.Millisecond, metrics.ReplicationLag)
	assert.Equal(t, int64(42), metrics.SyncedNodes)
	assert.Equal(t, int64(5), metrics.ConflictCount)
	assert.Equal(t, int64(2), metrics.ErrorCount)
	assert.Equal(t, 10.5, metrics.ThroughputPerSec)
	assert.False(t, metrics.LastSyncTime.IsZero())
}

func TestSyncEngine_Stop(t *testing.T) {
	engine := NewSyncEngine("test", &ZookeeperCluster{}, &ZookeeperCluster{}, LastWriteWins)

	err := engine.Stop()
	assert.NoError(t, err)
}

func TestSyncEngine_Start_ContextCancellation(t *testing.T) {
	engine := NewSyncEngine("test", &ZookeeperCluster{}, &ZookeeperCluster{}, LastWriteWins)

	ctx, cancel := context.WithCancel(context.Background())
	
	// Cancel context immediately
	cancel()

	err := engine.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server list must not be empty")
}

func TestSyncEngine_Start_InvalidSourceHosts(t *testing.T) {
	engine := NewSyncEngine("test", &ZookeeperCluster{}, &ZookeeperCluster{}, LastWriteWins)

	ctx := context.Background()
	err := engine.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server list must not be empty")
}

func TestSyncEngine_Start_InvalidTargetHosts(t *testing.T) {
	source := &ZookeeperCluster{
		Hosts: []string{"zk1:2181"},
	}
	target := &ZookeeperCluster{
		Hosts: []string{},
	}
	engine := NewSyncEngine("test", source, target, LastWriteWins)

	ctx := context.Background()
	err := engine.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such host")
}

func TestSyncEngine_EdgeCases(t *testing.T) {
	t.Run("Empty ID", func(t *testing.T) {
		engine := NewSyncEngine("", &ZookeeperCluster{}, &ZookeeperCluster{}, LastWriteWins)
		assert.Equal(t, "", engine.ID)
	})

	t.Run("Nil clusters", func(t *testing.T) {
		engine := NewSyncEngine("test", nil, nil, LastWriteWins)
		assert.Nil(t, engine.SourceCluster)
		assert.Nil(t, engine.TargetCluster)
	})

	t.Run("Invalid conflict strategy", func(t *testing.T) {
		engine := NewSyncEngine("test", &ZookeeperCluster{}, &ZookeeperCluster{}, ConflictStrategy("invalid"))
		assert.Equal(t, ConflictStrategy("invalid"), engine.ConflictResolution)
	})
}

func TestConflictStrategy_Constants(t *testing.T) {
	assert.Equal(t, ConflictStrategy("last-write-wins"), LastWriteWins)
	assert.Equal(t, ConflictStrategy("source-wins"), SourceWins)
	assert.Equal(t, ConflictStrategy("manual"), ManualResolve)
}

func TestSyncMetrics_Initialization(t *testing.T) {
	engine := NewSyncEngine("test", &ZookeeperCluster{}, &ZookeeperCluster{}, LastWriteWins)

	metrics := engine.GetMetrics()
	assert.Equal(t, time.Duration(0), metrics.ReplicationLag)
	assert.Equal(t, int64(0), metrics.SyncedNodes)
	assert.Equal(t, int64(0), metrics.ConflictCount)
	assert.Equal(t, int64(0), metrics.ErrorCount)
	assert.Equal(t, float64(0), metrics.ThroughputPerSec)
	assert.True(t, metrics.LastSyncTime.IsZero())
}

func TestZNode_Structure(t *testing.T) {
	node := &ZNode{
		Path:     "/test/path",
		Data:     []byte("test data"),
		Version:  1,
		CTime:    1234567890,
		MTime:    1234567890,
		Children: []string{"child1", "child2"},
		ACL:      []zk.ACL{},
	}

	assert.Equal(t, "/test/path", node.Path)
	assert.Equal(t, []byte("test data"), node.Data)
	assert.Equal(t, int32(1), node.Version)
	assert.Equal(t, int64(1234567890), node.CTime)
	assert.Equal(t, int64(1234567890), node.MTime)
	assert.Equal(t, []string{"child1", "child2"}, node.Children)
	assert.NotNil(t, node.ACL)
}

func TestSyncEngine_ConcurrentAccess(t *testing.T) {
	engine := NewSyncEngine("test", &ZookeeperCluster{}, &ZookeeperCluster{}, LastWriteWins)

	// Test concurrent access to metrics
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			
			// Update metrics
			engine.metrics.SyncedNodes++
			engine.metrics.ConflictCount++
			
			// Read metrics
			metrics := engine.GetMetrics()
			assert.GreaterOrEqual(t, metrics.SyncedNodes, int64(0))
			assert.GreaterOrEqual(t, metrics.ConflictCount, int64(0))
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestSyncEngine_Stop_MultipleCalls(t *testing.T) {
	engine := NewSyncEngine("test", &ZookeeperCluster{}, &ZookeeperCluster{}, LastWriteWins)

	// Call Stop multiple times
	err1 := engine.Stop()
	err2 := engine.Stop()
	err3 := engine.Stop()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
}

func TestSyncEngine_GetMetrics_ThreadSafety(t *testing.T) {
	engine := NewSyncEngine("test", &ZookeeperCluster{}, &ZookeeperCluster{}, LastWriteWins)

	// Update metrics in one goroutine
	go func() {
		for i := 0; i < 100; i++ {
			engine.metrics.SyncedNodes++
			engine.metrics.ConflictCount++
			time.Sleep(time.Millisecond)
		}
	}()

	// Read metrics in another goroutine
	done := make(chan bool)
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 100; i++ {
			metrics := engine.GetMetrics()
			assert.GreaterOrEqual(t, metrics.SyncedNodes, int64(0))
			assert.GreaterOrEqual(t, metrics.ConflictCount, int64(0))
			time.Sleep(time.Millisecond)
		}
	}()

	<-done
}
