package migration

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-zookeeper/zk"
)

// SyncEngine handles bidirectional synchronization between two Zookeeper clusters
type SyncEngine struct {
	ID                string
	SourceCluster     *ZookeeperCluster
	TargetCluster     *ZookeeperCluster
	ConflictResolution ConflictStrategy
	sourceConn        *zk.Conn
	targetConn        *zk.Conn
	syncInterval      time.Duration
	lagThreshold      time.Duration
	metrics           *SyncMetrics
	metricsMux        sync.RWMutex
}

// ZookeeperCluster represents a Zookeeper cluster configuration
// ZookeeperCluster is defined in zookeeper.go

// ConflictStrategy defines how to handle sync conflicts
type ConflictStrategy string

const (
	LastWriteWins ConflictStrategy = "last-write-wins"
	SourceWins    ConflictStrategy = "source-wins"
	ManualResolve ConflictStrategy = "manual"
)

// SyncMetrics tracks synchronization performance
type SyncMetrics struct {
	ReplicationLag    time.Duration
	SyncedNodes       int64
	ConflictCount     int64
	ErrorCount        int64
	LastSyncTime      time.Time
	ThroughputPerSec  float64
}

// ZNode represents a Zookeeper node
type ZNode struct {
	Path     string
	Data     []byte
	Version  int32
	CTime    int64
	MTime    int64
	Children []string
	ACL      []zk.ACL
}

// NewSyncEngine creates a new sync engine instance
func NewSyncEngine(id string, source, target *ZookeeperCluster, conflictResolution ConflictStrategy) *SyncEngine {
	return &SyncEngine{
		ID:                 id,
		SourceCluster:      source,
		TargetCluster:      target,
		ConflictResolution: conflictResolution,
		syncInterval:       1 * time.Second,
		lagThreshold:       50 * time.Millisecond,
		metrics: &SyncMetrics{
			LastSyncTime: time.Now(),
		},
	}
}

// Start begins the bidirectional synchronization
func (se *SyncEngine) Start(ctx context.Context) error {
	log.Printf("Starting sync engine %s: %s -> %s", se.ID, se.SourceCluster.ID, se.TargetCluster.ID)

	// Connect to source cluster
	sourceConn, _, err := zk.Connect(se.getHosts(se.SourceCluster), 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to source cluster: %w", err)
	}
	se.sourceConn = sourceConn

	// Connect to target cluster
	targetConn, _, err := zk.Connect(se.getHosts(se.TargetCluster), 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to target cluster: %w", err)
	}
	se.targetConn = targetConn

	// Perform initial snapshot
	if err := se.initialSnapshot(ctx); err != nil {
		return fmt.Errorf("initial snapshot failed: %w", err)
	}

	// Start continuous sync
	go se.continuousSync(ctx)

	// Start metrics collection
	go se.collectMetrics(ctx)

	<-ctx.Done()

	se.sourceConn.Close()
	se.targetConn.Close()

	return nil
}

// getHosts builds the Zookeeper connection string
func (se *SyncEngine) getHosts(cluster *ZookeeperCluster) []string {
	return cluster.Hosts
}

// initialSnapshot performs the initial data transfer from source to target
func (se *SyncEngine) initialSnapshot(ctx context.Context) error {
	log.Println("Starting initial snapshot...")

	startTime := time.Now()

	// Walk the source tree
	nodes, err := se.walkTree(ctx, se.sourceConn, "/mesos")
	if err != nil {
		return fmt.Errorf("failed to walk source tree: %w", err)
	}

	log.Printf("Found %d nodes to sync", len(nodes))

	// Sync nodes to target
	syncedCount := 0
	for _, node := range nodes {
		if err := se.syncNode(ctx, node); err != nil {
			log.Printf("Failed to sync node %s: %v", node.Path, err)
			continue
		}
		syncedCount++

		if syncedCount%100 == 0 {
			log.Printf("Synced %d/%d nodes", syncedCount, len(nodes))
		}
	}

	duration := time.Since(startTime)
	log.Printf("Initial snapshot complete: synced %d nodes in %v", syncedCount, duration)

	se.metricsMux.Lock()
	se.metrics.SyncedNodes = int64(syncedCount)
	se.metrics.LastSyncTime = time.Now()
	se.metricsMux.Unlock()

	return nil
}

// walkTree recursively walks the Zookeeper tree
func (se *SyncEngine) walkTree(ctx context.Context, conn *zk.Conn, path string) ([]*ZNode, error) {
	var nodes []*ZNode

	// Get node data
	data, stat, err := conn.Get(path)
	if err != nil {
		return nil, err
	}

	// Get node ACL
	acl, _, err := conn.GetACL(path)
	if err != nil {
		log.Printf("Failed to get ACL for %s: %v", path, err)
		acl = zk.WorldACL(zk.PermAll)
	}

	// Get children
	children, _, err := conn.Children(path)
	if err != nil {
		return nil, err
	}

	// Create ZNode
	node := &ZNode{
		Path:     path,
		Data:     data,
		Version:  stat.Version,
		CTime:    stat.Ctime,
		MTime:    stat.Mtime,
		Children: children,
		ACL:      acl,
	}
	nodes = append(nodes, node)

	// Recursively walk children
	for _, child := range children {
		childPath := path
		if path != "/" {
			childPath += "/"
		}
		childPath += child

		childNodes, err := se.walkTree(ctx, conn, childPath)
		if err != nil {
			log.Printf("Failed to walk child %s: %v", childPath, err)
			continue
		}
		nodes = append(nodes, childNodes...)
	}

	return nodes, nil
}

// syncNode synchronizes a single node to the target
func (se *SyncEngine) syncNode(ctx context.Context, node *ZNode) error {
	// Check if node exists on target
	exists, _, err := se.targetConn.Exists(node.Path)
	if err != nil {
		return err
	}

	if !exists {
		// Create node on target
		_, err = se.targetConn.Create(node.Path, node.Data, 0, node.ACL)
		if err != nil && err != zk.ErrNodeExists {
			return err
		}
	} else {
		// Update existing node
		_, err = se.targetConn.Set(node.Path, node.Data, -1)
		if err != nil {
			return err
		}

		// Update ACL
		_, err = se.targetConn.SetACL(node.Path, node.ACL, -1)
		if err != nil {
			log.Printf("Failed to update ACL for %s: %v", node.Path, err)
		}
	}

	return nil
}

// continuousSync performs continuous synchronization
func (se *SyncEngine) continuousSync(ctx context.Context) {
	ticker := time.NewTicker(se.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := se.performSync(ctx); err != nil {
				log.Printf("Sync error: %v", err)
				se.metricsMux.Lock()
				se.metrics.ErrorCount++
				se.metricsMux.Unlock()
			}
		}
	}
}

// performSync performs a single sync operation
func (se *SyncEngine) performSync(ctx context.Context) error {
	startTime := time.Now()

	// Watch for changes on source
	// TODO: Implement efficient change detection using Zookeeper watches

	// For now, perform periodic full sync
	nodes, err := se.walkTree(ctx, se.sourceConn, "/mesos")
	if err != nil {
		return err
	}

	syncedCount := 0
	for _, node := range nodes {
		if err := se.syncNode(ctx, node); err != nil {
			continue
		}
		syncedCount++
	}

	duration := time.Since(startTime)

	se.metricsMux.Lock()
	se.metrics.ReplicationLag = duration
	se.metrics.SyncedNodes = int64(syncedCount)
	se.metrics.LastSyncTime = time.Now()
	se.metrics.ThroughputPerSec = float64(syncedCount) / duration.Seconds()
	se.metricsMux.Unlock()

	return nil
}

// collectMetrics collects and logs metrics
func (se *SyncEngine) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			se.metricsMux.RLock()
			metrics := *se.metrics
			se.metricsMux.RUnlock()

			log.Printf("Sync Metrics - Lag: %v, Synced: %d, Conflicts: %d, Errors: %d, Throughput: %.2f nodes/sec",
				metrics.ReplicationLag,
				metrics.SyncedNodes,
				metrics.ConflictCount,
				metrics.ErrorCount,
				metrics.ThroughputPerSec,
			)

			// Check if lag exceeds threshold
			if metrics.ReplicationLag > se.lagThreshold {
				log.Printf("WARNING: Replication lag (%v) exceeds threshold (%v)",
					metrics.ReplicationLag, se.lagThreshold)
			}
		}
	}
}

// GetMetrics returns current sync metrics
func (se *SyncEngine) GetMetrics() SyncMetrics {
	se.metricsMux.RLock()
	defer se.metricsMux.RUnlock()
	return *se.metrics
}

// Stop stops the sync engine gracefully
func (se *SyncEngine) Stop() error {
	log.Printf("Stopping sync engine %s", se.ID)

	if se.sourceConn != nil {
		se.sourceConn.Close()
	}
	if se.targetConn != nil {
		se.targetConn.Close()
	}

	return nil
}
