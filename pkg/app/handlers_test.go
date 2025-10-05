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

func setupTestServer(t *testing.T) *Server {
	gin.SetMode(gin.TestMode)

	config := ServerConfig{
		Host:               "localhost",
		Port:               8080,
		MaxDataAge:         1 * time.Hour,
		CleanupInterval:    5 * time.Minute,
		StaleNodeThreshold: 5 * time.Minute,
	}

	server := NewServer(config)
	return server
}

func TestHealthCheck(t *testing.T) {
	server := setupTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

func TestPing(t *testing.T) {
	server := setupTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/ping", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["pong"].(bool))
}

func TestRegisterAgent(t *testing.T) {
	server := setupTestServer(t)

	registration := map[string]interface{}{
		"agent_id":  "test-agent-1",
		"hostname":  "test-host",
		"metadata":  map[string]string{"version": "1.0.0"},
		"timestamp": time.Now(),
	}

	body, _ := json.Marshal(registration)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/agents/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Verify agent was registered
	server.mu.RLock()
	agent, exists := server.agents["test-agent-1"]
	server.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, "test-agent-1", agent.AgentID)
	assert.Equal(t, "test-host", agent.Hostname)
}

func TestHeartbeat(t *testing.T) {
	server := setupTestServer(t)

	// Register agent first
	server.agents["test-agent-1"] = &AgentInfo{
		AgentID:      "test-agent-1",
		Hostname:     "test-host",
		RegisteredAt: time.Now(),
		LastSeen:     time.Now().Add(-1 * time.Minute),
	}

	oldLastSeen := server.agents["test-agent-1"].LastSeen

	heartbeat := map[string]interface{}{
		"agent_id":  "test-agent-1",
		"timestamp": time.Now(),
	}

	body, _ := json.Marshal(heartbeat)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/agents/heartbeat/test-agent-1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify last seen was updated
	server.mu.RLock()
	newLastSeen := server.agents["test-agent-1"].LastSeen
	server.mu.RUnlock()

	assert.True(t, newLastSeen.After(oldLastSeen))
}

func TestSubmitReport(t *testing.T) {
	server := setupTestServer(t)

	// Register agent first
	server.agents["test-agent-1"] = &AgentInfo{
		AgentID:      "test-agent-1",
		Hostname:     "test-host",
		RegisteredAt: time.Now(),
		LastSeen:     time.Now(),
	}

	report := probe.ReportData{
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

	body, _ := json.Marshal(report)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/reports", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	// Verify report was stored
	storedReport := server.storage.GetLatestReport("test-agent-1")
	assert.NotNil(t, storedReport)
	assert.Equal(t, "test-agent-1", storedReport.AgentID)
}

func TestGetTopology(t *testing.T) {
	server := setupTestServer(t)

	// Add some test data
	report := probe.ReportData{
		AgentID:   "test-agent-1",
		Hostname:  "test-host",
		Timestamp: time.Now(),
		HostInfo: &probe.HostInfo{
			Hostname: "test-host",
			CPUInfo: probe.CPUInfo{
				Cores: 4,
			},
		},
	}
	server.aggregator.ProcessReport(&report)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/query/topology", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	topology := response["topology"].(map[string]interface{})
	nodes := topology["nodes"].(map[string]interface{})
	assert.NotEmpty(t, nodes)
}

func TestGetStats(t *testing.T) {
	server := setupTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/query/stats", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "agents")
	assert.Contains(t, response, "websocket_clients")
	assert.Contains(t, response, "storage")
	assert.Contains(t, response, "topology")
}

func TestGetLatestReport(t *testing.T) {
	server := setupTestServer(t)

	// Add test report
	report := &probe.ReportData{
		AgentID:   "test-agent-1",
		Hostname:  "test-host",
		Timestamp: time.Now(),
	}
	server.storage.AddReport(report)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/query/agents/test-agent-1/latest", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test-agent-1", response["agent_id"])
	assert.Contains(t, response, "report")
}

func TestGetLatestReportNotFound(t *testing.T) {
	server := setupTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/query/agents/non-existent/latest", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetTimeSeries(t *testing.T) {
	server := setupTestServer(t)

	// Add test reports
	for i := 0; i < 5; i++ {
		report := &probe.ReportData{
			AgentID:   "test-agent-1",
			Hostname:  "test-host",
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		}
		server.storage.AddReport(report)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/query/agents/test-agent-1/timeseries?duration=1h", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test-agent-1", response["agent_id"])
	assert.Equal(t, "1h", response["duration"])
	points := response["points"].([]interface{})
	assert.Equal(t, 5, len(points))
}

func TestListAgents(t *testing.T) {
	server := setupTestServer(t)

	// Add test agents
	server.agents["agent-1"] = &AgentInfo{
		AgentID:  "agent-1",
		Hostname: "host-1",
	}
	server.agents["agent-2"] = &AgentInfo{
		AgentID:  "agent-2",
		Hostname: "host-2",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/agents/list", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(2), response["count"])
	agents := response["agents"].([]interface{})
	assert.Equal(t, 2, len(agents))
}

func TestServerStartStop(t *testing.T) {
	server := setupTestServer(t)

	ctx := context.Background()
	err := server.Start(ctx)
	require.NoError(t, err)

	assert.True(t, server.IsRunning())

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	err = server.Stop()
	require.NoError(t, err)

	assert.False(t, server.IsRunning())
}
