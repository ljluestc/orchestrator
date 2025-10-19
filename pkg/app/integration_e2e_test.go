package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ljluestc/orchestrator/pkg/probe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2EFullWorkflow tests the complete end-to-end workflow of the app server
func TestE2EFullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Step 1: Create and start server
	server := setupTestServer(t)
	ctx := context.Background()
	err := server.Start(ctx)
	require.NoError(t, err)
	defer server.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Step 2: Test health check
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	server.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Step 3: Register multiple agents
	numAgents := 3
	agentIDs := make([]string, numAgents)
	for i := 0; i < numAgents; i++ {
		agentID := fmt.Sprintf("e2e-agent-%d", i)
		agentIDs[i] = agentID

		registration := map[string]interface{}{
			"agent_id":  agentID,
			"hostname":  fmt.Sprintf("e2e-host-%d", i),
			"metadata":  map[string]string{"env": "test", "region": "us-west"},
			"timestamp": time.Now(),
		}

		body, _ := json.Marshal(registration)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/agents/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	}

	// Step 4: Verify agents are registered
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/agents/list", nil)
	server.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var agentListResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &agentListResponse)
	require.NoError(t, err)
	assert.Equal(t, float64(numAgents), agentListResponse["count"])

	// Step 5: Submit reports from each agent
	for i, agentID := range agentIDs {
		report := probe.ReportData{
			AgentID:   agentID,
			Hostname:  fmt.Sprintf("e2e-host-%d", i),
			Timestamp: time.Now(),
			HostInfo: &probe.HostInfo{
				Hostname:      fmt.Sprintf("e2e-host-%d", i),
				KernelVersion: "5.15.0",
				CPUInfo: probe.CPUInfo{
					Model: "Intel Xeon",
					Cores: 16,
					Usage: float64(20 + i*10),
				},
				MemoryInfo: probe.MemoryInfo{
					TotalMB:     32768,
					UsedMB:      uint64(16384 + i*1024),
					FreeMB:      uint64(16384 - i*1024),
					AvailableMB: uint64(16384 - i*1024),
					Usage:       float64(50 + i*5),
				},
			},
			DockerInfo: &probe.DockerInfo{
				Containers: []probe.ContainerInfo{
					{
						ID:      fmt.Sprintf("container-%d-1", i),
						Name:    fmt.Sprintf("web-%d", i),
						Image:   "nginx:latest",
						Status:  "running",
						State:   "running",
						Created: time.Now().Add(-1 * time.Hour),
						Stats: &probe.ContainerStats{
							CPUPercent:    15.5,
							MemoryUsageMB: 512,
							MemoryLimitMB: 1024,
							MemoryPercent: 50.0,
						},
					},
				},
			},
			ProcessesInfo: &probe.ProcessesInfo{
				Processes: []probe.ProcessInfo{
					{
						PID:      1234 + i*100,
						Name:     "nginx",
						Cmdline:  "nginx -g daemon off;",
						State:    "R",
						PPID:     1,
						UID:      0,
						GID:      0,
						Threads:  4,
						CPUTime:  1000,
						MemoryMB: 128,
						Cgroup:   fmt.Sprintf("/docker/container-%d-1", i),
					},
				},
			},
			NetworkInfo: &probe.NetworkInfo{
				Connections: []probe.NetworkConnection{
					{
						PID:         1234 + i*100,
						LocalAddr:   fmt.Sprintf("192.168.1.%d", 10+i),
						LocalPort:   80,
						RemoteAddr:  fmt.Sprintf("192.168.1.%d", 100+i),
						RemotePort:  443,
						Protocol:    "tcp",
						State:       "ESTABLISHED",
						ProcessName: "nginx",
					},
				},
			},
		}

		body, _ := json.Marshal(report)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/reports", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusAccepted, w.Code)
	}

	// Give time for reports to be processed
	time.Sleep(200 * time.Millisecond)

	// Step 6: Query topology and verify it contains all nodes
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/query/topology", nil)
	server.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var topologyResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &topologyResponse)
	require.NoError(t, err)

	topology := topologyResponse["topology"].(map[string]interface{})
	nodes := topology["nodes"].(map[string]interface{})
	edges := topology["edges"].(map[string]interface{})

	// Verify we have: 3 hosts + 3 containers + 3 processes = 9 nodes
	assert.GreaterOrEqual(t, len(nodes), 9, "Should have at least 9 nodes")

	// Verify we have edges (host->container and container->process relationships)
	assert.Greater(t, len(edges), 0, "Should have edges")

	// Step 7: Query latest report for each agent
	for _, agentID := range agentIDs {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/query/agents/%s/latest", agentID), nil)
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var reportResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &reportResponse)
		require.NoError(t, err)
		assert.Equal(t, agentID, reportResponse["agent_id"])
	}

	// Step 8: Query time-series data
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/v1/query/agents/%s/timeseries?duration=1h", agentIDs[0]), nil)
	server.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var tsResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &tsResponse)
	require.NoError(t, err)
	assert.Greater(t, int(tsResponse["count"].(float64)), 0)

	// Step 9: Send heartbeats
	for _, agentID := range agentIDs {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/agents/heartbeat/%s", agentID), nil)
		req.Header.Set("Content-Type", "application/json")
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Step 10: Get server stats
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/query/stats", nil)
	server.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var statsResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &statsResponse)
	require.NoError(t, err)

	assert.Equal(t, float64(numAgents), statsResponse["agents"])
	assert.Contains(t, statsResponse, "storage")
	assert.Contains(t, statsResponse, "topology")
	assert.Contains(t, statsResponse, "websocket_clients")

	t.Logf("E2E test completed successfully")
	t.Logf("Final stats: %+v", statsResponse)
}

// TestE2EWebSocketIntegration tests the WebSocket functionality end-to-end
func TestE2EWebSocketIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E WebSocket test in short mode")
	}

	server := setupTestServer(t)
	ctx := context.Background()
	err := server.Start(ctx)
	require.NoError(t, err)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Note: We cannot test actual WebSocket upgrade with httptest.ResponseRecorder
	// because it doesn't implement http.Hijacker interface required for WebSocket upgrades.
	// Instead, we verify that the WebSocket hub is running and can manage clients.

	// Verify server is running
	assert.True(t, server.IsRunning())

	// Verify WebSocket hub is initialized and running
	assert.NotNil(t, server.wsHub)
	assert.Equal(t, 0, server.wsHub.GetClientCount())

	// Test WebSocket hub broadcasting (without actual WebSocket connections)
	server.wsHub.BroadcastTopologyUpdate(server.aggregator.GetTopology())

	// Verify hub still operational
	assert.Equal(t, 0, server.wsHub.GetClientCount())

	t.Logf("WebSocket hub test completed - hub is operational")
}

// TestE2ECleanupWorkflow tests the cleanup functionality
func TestE2ECleanupWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E cleanup test in short mode")
	}

	// Create server with short cleanup interval
	server := setupTestServer(t)
	server.config.StaleNodeThreshold = 200 * time.Millisecond
	server.config.MaxDataAge = 200 * time.Millisecond

	ctx := context.Background()
	err := server.Start(ctx)
	require.NoError(t, err)
	defer server.Stop()

	// Add an old report
	oldReport := &probe.ReportData{
		AgentID:   "old-agent",
		Hostname:  "old-host",
		Timestamp: time.Now().Add(-300 * time.Millisecond),
	}
	server.storage.AddReport(oldReport)
	server.aggregator.ProcessReport(oldReport)

	// Wait a bit to ensure old report is stale
	time.Sleep(100 * time.Millisecond)

	// Add a recent report (this should NOT be cleaned up)
	recentReport := &probe.ReportData{
		AgentID:   "recent-agent",
		Hostname:  "recent-host",
		Timestamp: time.Now(),
	}
	server.storage.AddReport(recentReport)
	server.aggregator.ProcessReport(recentReport)

	// Manually trigger cleanup
	server.cleanup()

	// Verify cleanup worked correctly
	topology := server.aggregator.GetTopology()

	// Old agent should be cleaned (timestamp > 200ms old)
	assert.NotContains(t, topology.Nodes, "old-agent", "Old agent should be cleaned up")

	// Recent agent should still exist (timestamp < 200ms old)
	assert.Contains(t, topology.Nodes, "recent-agent", "Recent agent should still exist")

	t.Logf("Cleanup test completed - verified stale nodes removed and recent nodes retained")
}

// TestE2EConcurrentAgents tests multiple agents reporting concurrently
func TestE2EConcurrentAgents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E concurrent test in short mode")
	}

	server := setupTestServer(t)
	ctx := context.Background()
	err := server.Start(ctx)
	require.NoError(t, err)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	numAgents := 5
	reportsPerAgent := 3

	// Simulate multiple agents reporting concurrently
	done := make(chan bool, numAgents)
	for i := 0; i < numAgents; i++ {
		go func(agentIdx int) {
			agentID := fmt.Sprintf("concurrent-agent-%d", agentIdx)

			// Register agent
			registration := map[string]interface{}{
				"agent_id":  agentID,
				"hostname":  fmt.Sprintf("concurrent-host-%d", agentIdx),
				"timestamp": time.Now(),
			}
			body, _ := json.Marshal(registration)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/agents/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			server.router.ServeHTTP(w, req)

			// Send reports
			for j := 0; j < reportsPerAgent; j++ {
				report := probe.ReportData{
					AgentID:   agentID,
					Hostname:  fmt.Sprintf("concurrent-host-%d", agentIdx),
					Timestamp: time.Now(),
					HostInfo: &probe.HostInfo{
						Hostname: fmt.Sprintf("concurrent-host-%d", agentIdx),
						CPUInfo: probe.CPUInfo{
							Cores: 8,
							Usage: float64(j * 10),
						},
					},
				}

				body, _ := json.Marshal(report)
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/api/v1/reports", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				server.router.ServeHTTP(w, req)

				time.Sleep(10 * time.Millisecond)
			}

			done <- true
		}(i)
	}

	// Wait for all agents to complete
	for i := 0; i < numAgents; i++ {
		<-done
	}

	// Verify all agents were registered and processed
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/agents/list", nil)
	server.router.ServeHTTP(w, req)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(numAgents), response["count"])

	// Verify topology has all agents
	topology := server.aggregator.GetTopology()
	assert.GreaterOrEqual(t, len(topology.Nodes), numAgents)

	t.Logf("Concurrent agents test completed with %d agents", numAgents)
}
