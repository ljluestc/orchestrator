package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ljluestc/orchestrator/pkg/probe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAppIntegration tests the complete app backend workflow
func TestAppIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := ServerConfig{
		Host:               "localhost",
		Port:               8080,
		MaxDataAge:         1 * time.Hour,
		CleanupInterval:    5 * time.Minute,
		StaleNodeThreshold: 5 * time.Minute,
	}

	server := NewServer(config)

	// Test 1: Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := server.Start(ctx)
	require.NoError(t, err)
	assert.True(t, server.IsRunning())

	// Give server time to initialize
	time.Sleep(100 * time.Millisecond)

	// Test 2: Register multiple agents
	agents := []string{"agent-1", "agent-2", "agent-3"}
	for _, agentID := range agents {
		registration := map[string]interface{}{
			"agent_id":  agentID,
			"hostname":  "host-" + agentID,
			"metadata":  map[string]string{"version": "1.0.0"},
			"timestamp": time.Now(),
		}

		body, _ := json.Marshal(registration)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/agents/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code, "Failed to register agent %s", agentID)
	}

	// Test 3: Submit reports from each agent
	for i, agentID := range agents {
		report := probe.ReportData{
			AgentID:   agentID,
			Hostname:  "host-" + agentID,
			Timestamp: time.Now(),
			HostInfo: &probe.HostInfo{
				Hostname:      "host-" + agentID,
				KernelVersion: "5.10.0",
				CPUInfo: probe.CPUInfo{
					Model: "Intel Core i7",
					Cores: 8,
					Usage: float64(25 + i*10),
				},
				MemoryInfo: probe.MemoryInfo{
					TotalMB:     16384,
					UsedMB:      uint64(8192 + i*1024),
					FreeMB:      uint64(8192 - i*1024),
					AvailableMB: uint64(8192 - i*1024),
					Usage:       float64(50 + i*5),
				},
			},
			DockerInfo: &probe.DockerInfo{
				Containers: []probe.ContainerInfo{
					{
						ID:      "container-" + agentID + "-1",
						Name:    "nginx-" + agentID,
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
			ProcessesInfo: &probe.ProcessesInfo{
				Processes: []probe.ProcessInfo{
					{
						PID:      1234 + i,
						Name:     "nginx",
						Cmdline:  "nginx -g daemon off;",
						State:    "R",
						PPID:     1,
						Threads:  4,
						CPUTime:  1000,
						MemoryMB: 128,
					},
				},
			},
			NetworkInfo: &probe.NetworkInfo{
				Connections: []probe.NetworkConnection{
					{
						PID:         1234 + i,
						LocalAddr:   "192.168.1." + string(rune('1'+i)),
						LocalPort:   8080,
						RemoteAddr:  "192.168.1.100",
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

		assert.Equal(t, http.StatusAccepted, w.Code, "Failed to submit report for agent %s", agentID)
	}

	// Test 4: Verify agents are listed
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/agents/list", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var agentResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &agentResponse)
	require.NoError(t, err)
	assert.Equal(t, float64(3), agentResponse["count"])

	// Test 5: Query topology
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/query/topology", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var topologyResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &topologyResponse)
	require.NoError(t, err)

	topology := topologyResponse["topology"].(map[string]interface{})
	nodes := topology["nodes"].(map[string]interface{})

	// Should have host nodes + container nodes + process nodes
	assert.GreaterOrEqual(t, len(nodes), 3, "Should have at least 3 host nodes")

	// Test 6: Get latest report for a specific agent
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/query/agents/agent-1/latest", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var reportResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &reportResponse)
	require.NoError(t, err)
	assert.Equal(t, "agent-1", reportResponse["agent_id"])

	// Test 7: Get time series data
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/query/agents/agent-1/timeseries?duration=1h", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var timeSeriesResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &timeSeriesResponse)
	require.NoError(t, err)
	assert.Equal(t, "agent-1", timeSeriesResponse["agent_id"])

	// Test 8: Get server stats
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/query/stats", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var statsResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &statsResponse)
	require.NoError(t, err)
	assert.Contains(t, statsResponse, "agents")
	assert.Contains(t, statsResponse, "storage")
	assert.Contains(t, statsResponse, "topology")
	assert.Contains(t, statsResponse, "uptime")

	// Test 9: Health check
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/health", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var healthResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &healthResponse)
	require.NoError(t, err)
	assert.Equal(t, "healthy", healthResponse["status"])

	// Test 10: Ping endpoint
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/ping", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var pingResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &pingResponse)
	require.NoError(t, err)
	assert.True(t, pingResponse["pong"].(bool))

	// Test 11: Agent heartbeat
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/agents/heartbeat/agent-1", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Test 12: Get agent config
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/agents/config/agent-1", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var configResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &configResponse)
	require.NoError(t, err)
	assert.Contains(t, configResponse, "collection_interval")
	assert.Contains(t, configResponse, "enabled_collectors")

	// Test 13: Stop server
	err = server.Stop()
	require.NoError(t, err)
	assert.False(t, server.IsRunning())

	t.Log("Integration test completed successfully!")
}

// TestAppEndToEnd tests the complete flow from agent registration to data aggregation
func TestAppEndToEnd(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := ServerConfig{
		Host:               "localhost",
		Port:               8081,
		MaxDataAge:         1 * time.Hour,
		CleanupInterval:    5 * time.Minute,
		StaleNodeThreshold: 5 * time.Minute,
	}

	server := NewServer(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := server.Start(ctx)
	require.NoError(t, err)
	defer server.Stop()

	// Scenario: Probe registers and sends multiple reports over time
	agentID := "e2e-agent-1"
	hostname := "e2e-host-1"

	// Step 1: Register agent
	registration := map[string]interface{}{
		"agent_id":  agentID,
		"hostname":  hostname,
		"metadata":  map[string]string{"env": "test"},
		"timestamp": time.Now(),
	}

	body, _ := json.Marshal(registration)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/agents/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	server.router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	// Step 2: Send periodic reports simulating real probe behavior
	for i := 0; i < 3; i++ {
		report := probe.ReportData{
			AgentID:   agentID,
			Hostname:  hostname,
			Timestamp: time.Now(),
			HostInfo: &probe.HostInfo{
				Hostname:      hostname,
				KernelVersion: "5.10.0",
				CPUInfo: probe.CPUInfo{
					Model: "Intel Core i7",
					Cores: 8,
					Usage: float64(20 + i*10),
				},
				MemoryInfo: probe.MemoryInfo{
					TotalMB:     16384,
					UsedMB:      uint64(4096 + i*1024),
					FreeMB:      uint64(12288 - i*1024),
					AvailableMB: uint64(12288 - i*1024),
					Usage:       float64(25 + i*5),
				},
			},
		}

		body, _ := json.Marshal(report)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/reports", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		server.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusAccepted, w.Code)

		// Simulate reporting interval
		time.Sleep(100 * time.Millisecond)
	}

	// Step 3: Verify time series data was stored correctly
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/query/agents/"+agentID+"/timeseries?duration=1m", nil)
	server.router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var timeSeriesResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &timeSeriesResponse)
	require.NoError(t, err)

	points := timeSeriesResponse["points"].([]interface{})
	assert.GreaterOrEqual(t, len(points), 3, "Should have at least 3 data points")

	// Step 4: Verify topology includes the agent and its data
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/query/topology", nil)
	server.router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var topologyResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &topologyResponse)
	require.NoError(t, err)

	topology := topologyResponse["topology"].(map[string]interface{})
	nodes := topology["nodes"].(map[string]interface{})
	assert.Contains(t, nodes, agentID, "Topology should contain the agent node")

	// Step 5: Get latest report and verify data freshness
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/query/agents/"+agentID+"/latest", nil)
	server.router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var latestResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &latestResponse)
	require.NoError(t, err)

	reportData := latestResponse["report"].(map[string]interface{})
	assert.Equal(t, agentID, reportData["agent_id"])
	assert.Equal(t, hostname, reportData["hostname"])

	// Step 6: Verify server stats reflect the activity
	stats := server.GetStats()
	assert.Equal(t, 1, stats["agents"])
	assert.GreaterOrEqual(t, stats["websocket_clients"].(int), 0)

	t.Log("End-to-end test completed successfully!")
}

// TestAppResilience tests the app's ability to handle error conditions
func TestAppResilience(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := setupTestServer(t)

	// Test invalid report submission (missing agent_id)
	t.Run("InvalidReport", func(t *testing.T) {
		report := map[string]interface{}{
			"hostname":  "test-host",
			"timestamp": time.Now(),
		}

		body, _ := json.Marshal(report)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/reports", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// Test getting report for non-existent agent
	t.Run("NonExistentAgent", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/query/agents/non-existent/latest", nil)
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	// Test invalid duration format
	t.Run("InvalidDuration", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/query/agents/test-agent/timeseries?duration=invalid", nil)
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// Test invalid registration (missing required fields)
	t.Run("InvalidRegistration", func(t *testing.T) {
		registration := map[string]interface{}{
			"agent_id": "test-agent",
			// Missing hostname
		}

		body, _ := json.Marshal(registration)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/agents/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestAppConfiguration tests various server configurations
func TestAppConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		config ServerConfig
	}{
		{
			name: "DefaultConfig",
			config: ServerConfig{
				// All defaults
			},
		},
		{
			name: "CustomConfig",
			config: ServerConfig{
				Host:               "0.0.0.0",
				Port:               9090,
				MaxDataAge:         2 * time.Hour,
				CleanupInterval:    10 * time.Minute,
				StaleNodeThreshold: 10 * time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			server := NewServer(tt.config)

			assert.NotNil(t, server)
			assert.NotNil(t, server.storage)
			assert.NotNil(t, server.aggregator)
			assert.NotNil(t, server.wsHub)

			// Verify defaults are applied
			config := server.GetConfig()
			if tt.config.Host == "" {
				assert.Equal(t, "0.0.0.0", config.Host)
			} else {
				assert.Equal(t, tt.config.Host, config.Host)
			}

			if tt.config.Port == 0 {
				assert.Equal(t, 8080, config.Port)
			} else {
				assert.Equal(t, tt.config.Port, config.Port)
			}
		})
	}
}
