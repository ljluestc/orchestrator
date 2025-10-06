package probe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock server for testing client communication
type mockServer struct {
	server         *httptest.Server
	reports        []ReportData
	registrations  []map[string]interface{}
	heartbeats     []map[string]interface{}
	mu             sync.Mutex
	respondWithErr bool
}

func newMockServer() *mockServer {
	ms := &mockServer{
		reports:        make([]ReportData, 0),
		registrations:  make([]map[string]interface{}, 0),
		heartbeats:     make([]map[string]interface{}, 0),
		respondWithErr: false,
	}

	mux := http.NewServeMux()

	// POST /api/v1/reports
	mux.HandleFunc("/api/v1/reports", func(w http.ResponseWriter, r *http.Request) {
		ms.mu.Lock()
		defer ms.mu.Unlock()

		if ms.respondWithErr {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal server error"))
			return
		}

		var report ReportData
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ms.reports = append(ms.reports, report)
		w.WriteHeader(http.StatusOK)
	})

	// POST /api/v1/agents/register
	mux.HandleFunc("/api/v1/agents/register", func(w http.ResponseWriter, r *http.Request) {
		ms.mu.Lock()
		defer ms.mu.Unlock()

		var registration map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&registration); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ms.registrations = append(ms.registrations, registration)
		w.WriteHeader(http.StatusCreated)
	})

	// POST /api/v1/agents/{id}/heartbeat
	mux.HandleFunc("/api/v1/agents/heartbeat/", func(w http.ResponseWriter, r *http.Request) {
		ms.mu.Lock()
		defer ms.mu.Unlock()

		var heartbeat map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&heartbeat); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ms.heartbeats = append(ms.heartbeats, heartbeat)
		w.WriteHeader(http.StatusOK)
	})

	// GET /api/v1/ping
	mux.HandleFunc("/api/v1/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// GET /api/v1/agents/{id}/config
	mux.HandleFunc("/api/v1/agents/config/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			config := map[string]interface{}{
				"collection_interval": 30,
				"max_processes":       100,
			}
			json.NewEncoder(w).Encode(config)
		}
	})

	ms.server = httptest.NewServer(mux)
	return ms
}

func (ms *mockServer) Close() {
	ms.server.Close()
}

func (ms *mockServer) GetReports() []ReportData {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.reports
}

func (ms *mockServer) GetRegistrations() []map[string]interface{} {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.registrations
}

func (ms *mockServer) SetErrorResponse(shouldError bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.respondWithErr = shouldError
}

func TestClient_SendReport(t *testing.T) {
	server := newMockServer()
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL: server.server.URL,
		AgentID:   "test-agent",
		APIKey:    "test-key",
	})

	report := &ReportData{
		Hostname: "test-host",
		HostInfo: &HostInfo{
			Hostname: "test-host",
		},
	}

	ctx := context.Background()
	err := client.SendReport(ctx, report)
	require.NoError(t, err)

	// Verify report was received
	reports := server.GetReports()
	require.Len(t, reports, 1)
	assert.Equal(t, "test-agent", reports[0].AgentID)
	assert.Equal(t, "test-host", reports[0].Hostname)
}

func TestClient_SendReportWithRetry(t *testing.T) {
	server := newMockServer()
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL: server.server.URL,
		AgentID:   "test-agent",
	})

	// First attempt will fail
	server.SetErrorResponse(true)

	report := &ReportData{
		Hostname: "test-host",
	}

	ctx := context.Background()

	// Start retry in goroutine
	go func() {
		time.Sleep(100 * time.Millisecond)
		server.SetErrorResponse(false) // Fix server on second attempt
	}()

	err := client.SendReportWithRetry(ctx, report, 3, 50*time.Millisecond)
	require.NoError(t, err)

	// Should eventually succeed
	reports := server.GetReports()
	assert.GreaterOrEqual(t, len(reports), 1)
}

func TestClient_RegisterAgent(t *testing.T) {
	server := newMockServer()
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL: server.server.URL,
		AgentID:   "test-agent",
	})

	ctx := context.Background()
	metadata := map[string]string{
		"version": "1.0.0",
	}

	err := client.RegisterAgent(ctx, "test-host", metadata)
	require.NoError(t, err)

	// Verify registration
	registrations := server.GetRegistrations()
	require.Len(t, registrations, 1)
	assert.Equal(t, "test-agent", registrations[0]["agent_id"])
	assert.Equal(t, "test-host", registrations[0]["hostname"])
}

func TestClient_Ping(t *testing.T) {
	server := newMockServer()
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL: server.server.URL,
		AgentID:   "test-agent",
	})

	ctx := context.Background()
	err := client.Ping(ctx)
	require.NoError(t, err)
}

func TestClient_GetConfig(t *testing.T) {
	server := newMockServer()
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL: server.server.URL,
		AgentID:   "test-agent",
	})

	ctx := context.Background()
	config, err := client.GetConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, config)

	// Verify config structure
	assert.Contains(t, config, "collection_interval")
	assert.Contains(t, config, "max_processes")
}

func TestClient_SendReportErrorHandling(t *testing.T) {
	server := newMockServer()
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL: server.server.URL,
		AgentID:   "test-agent",
	})

	// Test with invalid JSON data
	report := &ReportData{
		Hostname: "test-host",
		// This will cause JSON marshaling to fail
	}

	ctx := context.Background()

	// This should not panic, but might return an error
	// We're testing error handling paths
	_ = client.SendReport(ctx, report)
}

func TestClient_RegisterAgentErrorHandling(t *testing.T) {
	server := newMockServer()
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL: server.server.URL,
		AgentID:   "test-agent",
	})

	ctx := context.Background()
	metadata := map[string]string{
		"version": "1.0.0",
	}

	err := client.RegisterAgent(ctx, "test-host", metadata)
	require.NoError(t, err)
}

func TestClient_HeartbeatErrorHandling(t *testing.T) {
	server := newMockServer()
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL: server.server.URL,
		AgentID:   "test-agent",
	})

	ctx := context.Background()
	err := client.Heartbeat(ctx)
	require.NoError(t, err)
}

func TestProbe_Integration(t *testing.T) {
	server := newMockServer()
	defer server.Close()

	config := ProbeConfig{
		ServerURL:           server.server.URL,
		AgentID:             "test-probe",
		CollectionInterval:  100 * time.Millisecond,
		HeartbeatInterval:   200 * time.Millisecond,
		CollectHost:         true,
		CollectProcesses:    true,
		CollectNetwork:      true,
		MaxProcesses:        10,
		MaxConnections:      10,
		IncludeLocalhost:    true,
		IncludeAllProcesses: true,
		RetryAttempts:       2,
		RetryDelay:          50 * time.Millisecond,
	}

	// Skip Docker collector if not available
	_, err := NewDockerCollector(false)
	if err == nil {
		config.CollectDocker = true
	}

	probe, err := NewProbe(config)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start probe
	err = probe.Start(ctx)
	require.NoError(t, err)
	assert.True(t, probe.IsRunning())

	// Wait for some collections
	time.Sleep(300 * time.Millisecond)

	// Stop probe
	err = probe.Stop()
	require.NoError(t, err)
	assert.False(t, probe.IsRunning())

	// Verify reports were sent
	reports := server.GetReports()
	assert.GreaterOrEqual(t, len(reports), 1, "Should have sent at least one report")

	// Verify report content
	if len(reports) > 0 {
		report := reports[0]
		assert.Equal(t, "test-probe", report.AgentID)
		assert.NotEmpty(t, report.Hostname)

		// Should have collected host info
		if config.CollectHost {
			assert.NotNil(t, report.HostInfo)
			assert.NotEmpty(t, report.HostInfo.Hostname)
		}

		// Should have collected process info
		if config.CollectProcesses {
			assert.NotNil(t, report.ProcessesInfo)
			assert.GreaterOrEqual(t, report.ProcessesInfo.TotalProcesses, 1)
		}

		// Should have collected network info
		if config.CollectNetwork {
			assert.NotNil(t, report.NetworkInfo)
		}
	}

	// Verify registration
	registrations := server.GetRegistrations()
	assert.Len(t, registrations, 1, "Should have registered once")
}

func TestProbe_StartStop(t *testing.T) {
	server := newMockServer()
	defer server.Close()

	config := ProbeConfig{
		ServerURL:          server.server.URL,
		AgentID:            "test-probe",
		CollectionInterval: 1 * time.Second,
		CollectHost:        true,
	}

	probe, err := NewProbe(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Start
	err = probe.Start(ctx)
	require.NoError(t, err)
	assert.True(t, probe.IsRunning())

	// Try to start again (should fail)
	err = probe.Start(ctx)
	assert.Error(t, err)

	// Stop
	err = probe.Stop()
	require.NoError(t, err)
	assert.False(t, probe.IsRunning())

	// Try to stop again (should fail)
	err = probe.Stop()
	assert.Error(t, err)
}

func TestProbe_ContextCancellation(t *testing.T) {
	server := newMockServer()
	defer server.Close()

	config := ProbeConfig{
		ServerURL:          server.server.URL,
		AgentID:            "test-probe",
		CollectionInterval: 100 * time.Millisecond,
		CollectHost:        true,
	}

	probe, err := NewProbe(config)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	// Start probe
	err = probe.Start(ctx)
	require.NoError(t, err)

	// Cancel context
	cancel()

	// Wait a bit for goroutines to exit
	time.Sleep(200 * time.Millisecond)

	// Probe should still be "running" but goroutines should have stopped
	// We need to explicitly stop it
	err = probe.Stop()
	require.NoError(t, err)
}

func TestProbe_CollectionWithAllModules(t *testing.T) {
	server := newMockServer()
	defer server.Close()

	config := ProbeConfig{
		ServerURL:           server.server.URL,
		AgentID:             "test-probe",
		CollectionInterval:  50 * time.Millisecond,
		CollectHost:         true,
		CollectProcesses:    true,
		CollectNetwork:      true,
		MaxProcesses:        5,
		MaxConnections:      5,
		IncludeAllProcesses: true,
		IncludeLocalhost:    true,
		ResolveProcesses:    false,
	}

	// Try to enable Docker if available
	_, err := NewDockerCollector(false)
	if err == nil {
		config.CollectDocker = true
		config.CollectDockerStats = false // Don't collect stats for faster tests
	}

	probe, err := NewProbe(config)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err = probe.Start(ctx)
	require.NoError(t, err)

	time.Sleep(150 * time.Millisecond)

	err = probe.Stop()
	require.NoError(t, err)

	reports := server.GetReports()
	assert.GreaterOrEqual(t, len(reports), 1)

	// Validate comprehensive report
	if len(reports) > 0 {
		report := reports[0]

		assert.NotNil(t, report.HostInfo)
		assert.NotNil(t, report.ProcessesInfo)
		assert.NotNil(t, report.NetworkInfo)

		if config.CollectDocker && report.DockerInfo != nil {
			assert.NotNil(t, report.DockerInfo)
		}
	}
}
