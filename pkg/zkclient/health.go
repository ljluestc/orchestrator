package zkclient

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-zookeeper/zk"
)

// ClusterHealthStatus represents the health status of the Zookeeper cluster
type ClusterHealthStatus struct {
	Healthy       bool
	QuorumActive  bool
	Leader        string
	Mode          string
	Version       string
	ConnectedTo   string
	SessionID     int64
	Latency       time.Duration
	CheckedAt     time.Time
	ErrorMessage  string
}

// HealthCheck performs a health check on the Zookeeper connection
func (c *Client) HealthCheck(ctx context.Context) (*ClusterHealthStatus, error) {
	status := &ClusterHealthStatus{
		CheckedAt: time.Now(),
	}

	if c.conn == nil {
		status.Healthy = false
		status.ErrorMessage = "not connected"
		return status, ErrNotConnected
	}

	// Check connection state
	if c.conn.State() != zk.StateHasSession {
		status.Healthy = false
		status.ErrorMessage = fmt.Sprintf("connection state is %v", c.conn.State())
		return status, fmt.Errorf("unhealthy connection state: %v", c.conn.State())
	}

	// Get session ID
	status.SessionID = c.conn.SessionID()

	// Measure latency with a simple ping (exists check on root)
	start := time.Now()
	exists, _, err := c.conn.Exists("/")
	status.Latency = time.Since(start)

	if err != nil {
		status.Healthy = false
		status.ErrorMessage = fmt.Sprintf("ping failed: %v", err)
		return status, err
	}

	if !exists {
		status.Healthy = false
		status.ErrorMessage = "root znode does not exist"
		return status, fmt.Errorf("root znode does not exist")
	}

	// Try to get server stats
	server := c.conn.Server()
	status.ConnectedTo = server

	// Check quorum status
	quorumActive, err := c.CheckQuorum(ctx)
	if err != nil {
		c.logger.Warn("Failed to check quorum status: %v", err)
		status.QuorumActive = false
	} else {
		status.QuorumActive = quorumActive
	}

	status.Healthy = true
	return status, nil
}

// CheckQuorum checks if the Zookeeper cluster has an active quorum
func (c *Client) CheckQuorum(ctx context.Context) (bool, error) {
	if c.conn == nil {
		return false, ErrNotConnected
	}

	// Try to perform a write operation (create ephemeral node)
	// If we can write, the quorum is active
	testPath := fmt.Sprintf("/zk-health-check-%d", time.Now().UnixNano())

	// Create ephemeral node
	_, err := c.conn.Create(testPath, []byte("health-check"), zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	if err != nil {
		// Check if error is due to quorum loss
		if err == zk.ErrConnectionClosed || err == zk.ErrSessionExpired {
			return false, &HealthCheckError{
				Check:  "quorum",
				Reason: "connection lost",
				Err:    err,
			}
		}
		// Other errors might not indicate quorum loss
		c.logger.Debug("Failed to create health check node: %v", err)
		return false, err
	}

	// Clean up the test node
	err = c.conn.Delete(testPath, -1)
	if err != nil {
		c.logger.Warn("Failed to delete health check node: %v", err)
	}

	return true, nil
}

// healthCheckLoop performs periodic health checks
func (c *Client) healthCheckLoop() {
	defer c.wg.Done()

	if c.config.HealthCheckInterval == 0 {
		return
	}

	ticker := time.NewTicker(c.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			c.logger.Debug("Health check routine stopping")
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			status, err := c.HealthCheck(ctx)
			cancel()

			c.healthMu.Lock()
			c.healthStatus.LastCheckTime = time.Now()
			if err != nil {
				c.healthStatus.IsHealthy = false
				c.healthStatus.LastError = err
				c.logger.Warn("Health check failed: %v", err)

				// Trigger reconnection if needed
				if c.GetState() == StateConnected {
					c.logger.Error("Health check indicates unhealthy connection")
					select {
					case c.reconnectCh <- struct{}{}:
					default:
					}
				}
			} else {
				c.healthStatus.IsHealthy = status.Healthy
				c.healthStatus.QuorumHealthy = status.QuorumActive
				c.healthStatus.SessionActive = true
				c.healthStatus.LastError = nil
				c.logger.Debug("Health check passed - Latency: %v, QuorumActive: %v",
					status.Latency, status.QuorumActive)
			}
			c.healthMu.Unlock()
		}
	}
}

// IsHealthy returns whether the client is currently healthy
func (c *Client) IsHealthy(ctx context.Context) bool {
	if !c.IsConnected() {
		return false
	}

	status, err := c.HealthCheck(ctx)
	if err != nil {
		return false
	}

	return status.Healthy && status.QuorumActive
}

// WaitForHealthy waits for the client to become healthy or until context is done
func (c *Client) WaitForHealthy(ctx context.Context, checkInterval time.Duration) error {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		if c.IsHealthy(ctx) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Continue checking
		}
	}
}

// ServerStats represents statistics from a Zookeeper server
type ServerStats struct {
	Version      string
	Mode         string
	NodeCount    int64
	TotalWatches int64
	Connections  int64
	Outstanding  int64
	MinLatency   int64
	AvgLatency   int64
	MaxLatency   int64
}

// GetServerStats retrieves statistics from the connected Zookeeper server
func (c *Client) GetServerStats(ctx context.Context) (*ServerStats, error) {
	if c.conn == nil {
		return nil, ErrNotConnected
	}

	// Get server string which contains basic info
	server := c.conn.Server()

	stats := &ServerStats{}

	// Parse server response if available
	// Note: go-zookeeper doesn't expose detailed stats directly
	// This is a simplified version
	if server != "" {
		parts := strings.Split(server, ":")
		if len(parts) >= 2 {
			// Server info available
			stats.Version = "unknown"
			stats.Mode = "unknown"
		}
	}

	return stats, nil
}

// GetHealthStatus returns the current health status
func (c *Client) GetHealthStatus() *HealthStatus {
	c.healthMu.RLock()
	defer c.healthMu.RUnlock()

	// Return a copy
	return &HealthStatus{
		IsHealthy:     c.healthStatus.IsHealthy,
		LastCheckTime: c.healthStatus.LastCheckTime,
		QuorumHealthy: c.healthStatus.QuorumHealthy,
		SessionActive: c.healthStatus.SessionActive,
		LastError:     c.healthStatus.LastError,
	}
}
