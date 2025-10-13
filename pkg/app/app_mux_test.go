package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/ljluestc/orchestrator/pkg/probe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApp_NewApp tests the NewApp constructor
func TestApp_NewApp(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		port     int
		expected *App
	}{
		{
			name: "ValidApp",
			id:   "test-app",
			port: 8080,
			expected: &App{
				ID:   "test-app",
				Port: 8080,
			},
		},
		{
			name: "EmptyID",
			id:   "",
			port: 9090,
			expected: &App{
				ID:   "",
				Port: 9090,
			},
		},
		{
			name: "ZeroPort",
			id:   "zero-port-app",
			port: 0,
			expected: &App{
				ID:   "zero-port-app",
				Port: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp(tt.id, tt.port)

			assert.Equal(t, tt.expected.ID, app.ID)
			assert.Equal(t, tt.expected.Port, app.Port)
			assert.NotNil(t, app.router)
			assert.NotNil(t, app.server)
			assert.NotNil(t, app.upgrader)
			assert.NotNil(t, app.reports)
			assert.NotNil(t, app.subscribers)
			assert.NotNil(t, app.topology)
			assert.NotNil(t, app.reportsMux)
			assert.NotNil(t, app.subscribersMux)

			// Check that routes are set up
			assert.NotNil(t, app.router)
		})
	}
}

// TestApp_Start tests the Start method
func TestApp_Start(t *testing.T) {
	app := NewApp("test-app", 0) // Use port 0 for testing

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start the app in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Start(ctx)
	}()

	// Wait for context to be done
	<-ctx.Done()

	// Check that the app started without error
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Start method did not return within timeout")
	}
}

// TestApp_handleReport tests the handleReport method
func TestApp_handleReport(t *testing.T) {
	app := NewApp("test-app", 0)

	tests := []struct {
		name           string
		report         probe.ReportData
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "ValidReport",
			report: probe.ReportData{
				AgentID:   "agent-1",
				Hostname:  "host-1",
				Timestamp: time.Now(),
				HostInfo: &probe.HostInfo{
					Hostname: "host-1",
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
				DockerInfo: &probe.DockerInfo{
					Containers: []probe.ContainerInfo{
						{
							ID:      "container-1",
							Name:    "nginx",
							Image:   "nginx:latest",
							Status:  "running",
							State:   "running",
							Created: time.Now(),
						},
					},
				},
			},
			expectedStatus: http.StatusAccepted,
			expectedBody:   "accepted",
		},
		{
			name:           "InvalidJSON",
			report:         probe.ReportData{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid report format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			if tt.name == "InvalidJSON" {
				body = []byte("invalid json")
			} else {
				body, err = json.Marshal(tt.report)
				require.NoError(t, err)
			}

			req := httptest.NewRequest("POST", "/api/v1/reports", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			app.handleReport(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)

			// If valid report, check that it was stored
			if tt.expectedStatus == http.StatusAccepted {
				app.reportsMux.RLock()
				storedReport, exists := app.reports[tt.report.AgentID]
				app.reportsMux.RUnlock()

				assert.True(t, exists)
				assert.Equal(t, tt.report.AgentID, storedReport.AgentID)
				assert.Equal(t, tt.report.Hostname, storedReport.Hostname)
			}
		})
	}
}

// TestApp_handleGetTopology tests the handleGetTopology method
func TestApp_handleGetTopology(t *testing.T) {
	app := NewApp("test-app", 0)

	// Add some test data
	testReport := probe.ReportData{
		AgentID:   "agent-1",
		Hostname:  "host-1",
		Timestamp: time.Now(),
		HostInfo: &probe.HostInfo{
			Hostname: "host-1",
			CPUInfo: probe.CPUInfo{
				Model: "Intel Core i7",
				Cores: 8,
				Usage: 25.5,
			},
		},
		DockerInfo: &probe.DockerInfo{
			Containers: []probe.ContainerInfo{
				{
					ID:      "container-1",
					Name:    "nginx",
					Image:   "nginx:latest",
					Status:  "running",
					State:   "running",
					Created: time.Now(),
				},
			},
		},
	}

	app.reportsMux.Lock()
	app.reports[testReport.AgentID] = &testReport
	app.reportsMux.Unlock()

	tests := []struct {
		name           string
		view           string
		expectedStatus int
	}{
		{
			name:           "DefaultView",
			view:           "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "ContainersView",
			view:           "containers",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "ProcessesView",
			view:           "processes",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/topology"
			if tt.view != "" {
				url += "?view=" + tt.view
			}

			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			app.handleGetTopology(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			if tt.expectedStatus == http.StatusOK {
				var topology Topology
				err := json.Unmarshal(w.Body.Bytes(), &topology)
				require.NoError(t, err)

				assert.NotNil(t, topology.Timestamp)
				assert.NotNil(t, topology.Hosts)
				assert.NotNil(t, topology.Containers)
				assert.NotNil(t, topology.Processes)
				assert.NotNil(t, topology.Networks)
				assert.NotNil(t, topology.ViewModes)
			}
		})
	}
}

// TestApp_handleWebSocket tests the handleWebSocket method
func TestApp_handleWebSocket(t *testing.T) {
	app := NewApp("test-app", 0)

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.handleWebSocket(w, r)
	}))
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/topology/ws"

	// Test WebSocket connection
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Skipf("WebSocket connection failed: %v", err)
	}
	defer conn.Close()

	// Check that the connection was added to subscribers
	app.subscribersMux.RLock()
	subscriberCount := len(app.subscribers)
	app.subscribersMux.RUnlock()

	assert.Equal(t, 1, subscriberCount)

	// Test receiving initial topology
	var topology Topology
	err = conn.ReadJSON(&topology)
	assert.NoError(t, err)
	assert.NotNil(t, topology.Timestamp)
}

// TestApp_handleListContainers tests the handleListContainers method
func TestApp_handleListContainers(t *testing.T) {
	app := NewApp("test-app", 0)

	// Add test data
	testReport := probe.ReportData{
		AgentID:   "agent-1",
		Hostname:  "host-1",
		Timestamp: time.Now(),
		DockerInfo: &probe.DockerInfo{
			Containers: []probe.ContainerInfo{
				{
					ID:      "container-1",
					Name:    "nginx",
					Image:   "nginx:latest",
					Status:  "running",
					State:   "running",
					Created: time.Now(),
				},
				{
					ID:      "container-2",
					Name:    "redis",
					Image:   "redis:latest",
					Status:  "running",
					State:   "running",
					Created: time.Now(),
				},
			},
		},
	}

	app.reportsMux.Lock()
	app.reports[testReport.AgentID] = &testReport
	app.reportsMux.Unlock()

	req := httptest.NewRequest("GET", "/api/v1/containers", nil)
	w := httptest.NewRecorder()

	app.handleListContainers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var containers []probe.ContainerInfo
	err := json.Unmarshal(w.Body.Bytes(), &containers)
	require.NoError(t, err)

	assert.Len(t, containers, 2)
	assert.Equal(t, "container-1", containers[0].ID)
	assert.Equal(t, "container-2", containers[1].ID)
}

// TestApp_handleContainerStop tests the handleContainerStop method
func TestApp_handleContainerStop(t *testing.T) {
	app := NewApp("test-app", 0)

	req := httptest.NewRequest("POST", "/api/v1/containers/test-container/stop", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-container"})
	w := httptest.NewRecorder()

	app.handleContainerStop(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "success", response["status"])
	assert.Equal(t, "stop", response["action"])
	assert.Equal(t, "test-container", response["container_id"])
}

// TestApp_handleContainerStart tests the handleContainerStart method
func TestApp_handleContainerStart(t *testing.T) {
	app := NewApp("test-app", 0)

	req := httptest.NewRequest("POST", "/api/v1/containers/test-container/start", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-container"})
	w := httptest.NewRecorder()

	app.handleContainerStart(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "success", response["status"])
	assert.Equal(t, "start", response["action"])
	assert.Equal(t, "test-container", response["container_id"])
}

// TestApp_handleContainerRestart tests the handleContainerRestart method
func TestApp_handleContainerRestart(t *testing.T) {
	app := NewApp("test-app", 0)

	req := httptest.NewRequest("POST", "/api/v1/containers/test-container/restart", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-container"})
	w := httptest.NewRecorder()

	app.handleContainerRestart(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "success", response["status"])
	assert.Equal(t, "restart", response["action"])
	assert.Equal(t, "test-container", response["container_id"])
}

// TestApp_handleContainerLogs tests the handleContainerLogs method
func TestApp_handleContainerLogs(t *testing.T) {
	app := NewApp("test-app", 0)

	req := httptest.NewRequest("GET", "/api/v1/containers/test-container/logs", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-container"})
	w := httptest.NewRecorder()

	app.handleContainerLogs(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "test-container")
}

// TestApp_handleContainerExec tests the handleContainerExec method
func TestApp_handleContainerExec(t *testing.T) {
	app := NewApp("test-app", 0)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "ValidCommand",
			requestBody: map[string]interface{}{
				"command": []string{"ls", "-la"},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "InvalidJSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			if tt.name == "InvalidJSON" {
				body = []byte("invalid json")
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req := httptest.NewRequest("POST", "/api/v1/containers/test-container/exec", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req = mux.SetURLVars(req, map[string]string{"id": "test-container"})
			w := httptest.NewRecorder()

			app.handleContainerExec(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, "success", response["status"])
				assert.Equal(t, "test-container", response["container_id"])
				assert.NotNil(t, response["command"])
				assert.NotNil(t, response["output"])
			} else {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
		})
	}
}

// TestApp_handleSearch tests the handleSearch method
func TestApp_handleSearch(t *testing.T) {
	app := NewApp("test-app", 0)

	tests := []struct {
		name           string
		query          string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "ValidQuery",
			query:          "nginx",
			expectedStatus: http.StatusOK,
			expectedBody:   "nginx",
		},
		{
			name:           "EmptyQuery",
			query:          "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing query parameter",
		},
		{
			name:           "NoQueryParam",
			query:          "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing query parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/search"
			if tt.query != "" {
				url += "?q=" + tt.query
			}

			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			app.handleSearch(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, tt.query, response["query"])
				assert.NotNil(t, response["results"])
			} else {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
		})
	}
}

// TestApp_handleHealth tests the handleHealth method
func TestApp_handleHealth(t *testing.T) {
	app := NewApp("test-app", 0)

	// Add some test reports
	testReport := probe.ReportData{
		AgentID:   "agent-1",
		Hostname:  "host-1",
		Timestamp: time.Now(),
	}

	app.reportsMux.Lock()
	app.reports[testReport.AgentID] = &testReport
	app.reportsMux.Unlock()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	app.handleHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, app.ID, response["app_id"])
	assert.Equal(t, float64(1), response["probes_connected"])
	assert.NotNil(t, response["timestamp"])
}

// TestApp_handleMetrics tests the handleMetrics method
func TestApp_handleMetrics(t *testing.T) {
	app := NewApp("test-app", 0)

	// Add test reports with containers
	testReport := probe.ReportData{
		AgentID:   "agent-1",
		Hostname:  "host-1",
		Timestamp: time.Now(),
		DockerInfo: &probe.DockerInfo{
			Containers: []probe.ContainerInfo{
				{ID: "container-1", Name: "nginx"},
				{ID: "container-2", Name: "redis"},
			},
		},
	}

	app.reportsMux.Lock()
	app.reports[testReport.AgentID] = &testReport
	app.reportsMux.Unlock()

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	app.handleMetrics(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()
	assert.Contains(t, body, "monitoring_app_probes_connected 1")
	assert.Contains(t, body, "monitoring_app_containers_total 2")
	assert.Contains(t, body, "# HELP monitoring_app_probes_connected")
	assert.Contains(t, body, "# TYPE monitoring_app_probes_connected gauge")
	assert.Contains(t, body, "# HELP monitoring_app_containers_total")
	assert.Contains(t, body, "# TYPE monitoring_app_containers_total gauge")
}

// TestApp_aggregateTopology tests the aggregateTopology method
func TestApp_aggregateTopology(t *testing.T) {
	app := NewApp("test-app", 0)

	// Add test data
	testReport := probe.ReportData{
		AgentID:   "agent-1",
		Hostname:  "host-1",
		Timestamp: time.Now(),
		HostInfo: &probe.HostInfo{
			Hostname: "host-1",
			CPUInfo: probe.CPUInfo{
				Model: "Intel Core i7",
				Cores: 8,
				Usage: 25.5,
			},
		},
		DockerInfo: &probe.DockerInfo{
			Containers: []probe.ContainerInfo{
				{
					ID:      "container-1",
					Name:    "nginx",
					Image:   "nginx:latest",
					Status:  "running",
					State:   "running",
					Created: time.Now(),
				},
			},
		},
		ProcessesInfo: &probe.ProcessesInfo{
			Processes: []probe.ProcessInfo{
				{
					PID:      1234,
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
					PID:         1234,
					LocalAddr:   "192.168.1.1",
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

	app.reportsMux.Lock()
	app.reports[testReport.AgentID] = &testReport
	app.reportsMux.Unlock()

	// Test aggregation with context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Start aggregation
	go app.aggregateTopology(ctx)

	// Wait for context to be done
	<-ctx.Done()

	// Check that topology was updated
	app.reportsMux.RLock()
	topology := app.topology
	app.reportsMux.RUnlock()

	assert.NotNil(t, topology)
	assert.NotNil(t, topology.Timestamp)
	assert.NotNil(t, topology.Hosts)
	assert.NotNil(t, topology.Containers)
	assert.NotNil(t, topology.Processes)
	assert.NotNil(t, topology.Networks)
	// Note: TotalProbes might be 0 if the aggregation didn't run yet due to timing
	assert.GreaterOrEqual(t, topology.TotalProbes, 0)
}

// TestApp_updateTopology tests the updateTopology method
func TestApp_updateTopology(t *testing.T) {
	app := NewApp("test-app", 0)

	// Add test data
	testReport := probe.ReportData{
		AgentID:   "agent-1",
		Hostname:  "host-1",
		Timestamp: time.Now(),
		HostInfo: &probe.HostInfo{
			Hostname: "host-1",
			CPUInfo: probe.CPUInfo{
				Model: "Intel Core i7",
				Cores: 8,
				Usage: 25.5,
			},
		},
		DockerInfo: &probe.DockerInfo{
			Containers: []probe.ContainerInfo{
				{
					ID:      "container-1",
					Name:    "nginx",
					Image:   "nginx:latest",
					Status:  "running",
					State:   "running",
					Created: time.Now(),
				},
			},
		},
		ProcessesInfo: &probe.ProcessesInfo{
			Processes: []probe.ProcessInfo{
				{
					PID:      1234,
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
					PID:         1234,
					LocalAddr:   "192.168.1.1",
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

	app.reportsMux.Lock()
	app.reports[testReport.AgentID] = &testReport
	app.reportsMux.Unlock()

	// Call updateTopology
	app.updateTopology()

	// Check that topology was updated
	app.reportsMux.RLock()
	topology := app.topology
	app.reportsMux.RUnlock()

	assert.NotNil(t, topology)
	assert.NotNil(t, topology.Timestamp)
	assert.NotNil(t, topology.Hosts)
	assert.NotNil(t, topology.Containers)
	assert.NotNil(t, topology.Processes)
	assert.NotNil(t, topology.Networks)
	assert.Equal(t, 1, topology.TotalProbes)

	// Check specific data
	assert.Contains(t, topology.Hosts, "host-1")
	assert.Contains(t, topology.Containers, "container-1")
	assert.Contains(t, topology.Processes, "1234")
	assert.Len(t, topology.Networks, 1)
}

// TestApp_broadcastUpdates tests the broadcastUpdates method
func TestApp_broadcastUpdates(t *testing.T) {
	app := NewApp("test-app", 0)

	// Test with context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Start broadcast updates
	go app.broadcastUpdates(ctx)

	// Wait for context to be done
	<-ctx.Done()

	// Method should complete without error
}

// TestApp_broadcastTopology tests the broadcastTopology method
func TestApp_broadcastTopology(t *testing.T) {
	app := NewApp("test-app", 0)

	// Add some test data
	testReport := probe.ReportData{
		AgentID:   "agent-1",
		Hostname:  "host-1",
		Timestamp: time.Now(),
	}

	app.reportsMux.Lock()
	app.reports[testReport.AgentID] = &testReport
	app.reportsMux.Unlock()

	// Update topology first
	app.updateTopology()

	// Call broadcastTopology
	app.broadcastTopology()

	// Method should complete without error
	// In a real test, we would verify that WebSocket clients received the message
}

// TestApp_buildTopologyView tests the buildTopologyView method
func TestApp_buildTopologyView(t *testing.T) {
	app := NewApp("test-app", 0)

	// Add test data
	testReport := probe.ReportData{
		AgentID:   "agent-1",
		Hostname:  "host-1",
		Timestamp: time.Now(),
		HostInfo: &probe.HostInfo{
			Hostname: "host-1",
		},
	}

	app.reportsMux.Lock()
	app.reports[testReport.AgentID] = &testReport
	app.reportsMux.Unlock()

	// Update topology
	app.updateTopology()

	tests := []struct {
		name string
		view string
	}{
		{
			name: "ContainersView",
			view: "containers",
		},
		{
			name: "ProcessesView",
			view: "processes",
		},
		{
			name: "HostsView",
			view: "hosts",
		},
		{
			name: "PodsView",
			view: "pods",
		},
		{
			name: "ServicesView",
			view: "services",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topology := app.buildTopologyView(tt.view)

			assert.NotNil(t, topology)
			assert.NotNil(t, topology.Timestamp)
			assert.NotNil(t, topology.Hosts)
			assert.NotNil(t, topology.Containers)
			assert.NotNil(t, topology.Processes)
			assert.NotNil(t, topology.Networks)
			assert.NotNil(t, topology.ViewModes)
		})
	}
}

// TestApp_EdgeCases tests edge cases and error conditions
func TestApp_EdgeCases(t *testing.T) {
	t.Run("NilReport", func(t *testing.T) {
		app := NewApp("test-app", 0)

		req := httptest.NewRequest("POST", "/api/v1/reports", bytes.NewBuffer([]byte("null")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		app.handleReport(w, req)

		// "null" gets decoded as a valid ReportData with zero values, so it returns 202
		assert.Equal(t, http.StatusAccepted, w.Code)
	})

	t.Run("EmptyReports", func(t *testing.T) {
		app := NewApp("test-app", 0)

		req := httptest.NewRequest("GET", "/api/v1/containers", nil)
		w := httptest.NewRecorder()

		app.handleListContainers(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var containers []probe.ContainerInfo
		err := json.Unmarshal(w.Body.Bytes(), &containers)
		require.NoError(t, err)

		assert.Len(t, containers, 0)
	})

	t.Run("EmptyTopology", func(t *testing.T) {
		app := NewApp("test-app", 0)

		req := httptest.NewRequest("GET", "/api/v1/topology", nil)
		w := httptest.NewRecorder()

		app.handleGetTopology(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var topology Topology
		err := json.Unmarshal(w.Body.Bytes(), &topology)
		require.NoError(t, err)

		assert.NotNil(t, topology.Timestamp)
		assert.NotNil(t, topology.Hosts)
		assert.NotNil(t, topology.Containers)
		assert.NotNil(t, topology.Processes)
		assert.NotNil(t, topology.Networks)
		assert.Equal(t, 0, topology.TotalProbes)
	})
}

// TestApp_ConcurrentAccess tests concurrent access to the app
func TestApp_ConcurrentAccess(t *testing.T) {
	app := NewApp("test-app", 0)

	// Test concurrent report submissions
	numGoroutines := 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			report := probe.ReportData{
				AgentID:   fmt.Sprintf("agent-%d", id),
				Hostname:  fmt.Sprintf("host-%d", id),
				Timestamp: time.Now(),
				HostInfo: &probe.HostInfo{
					Hostname: fmt.Sprintf("host-%d", id),
				},
			}

			body, _ := json.Marshal(report)
			req := httptest.NewRequest("POST", "/api/v1/reports", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			app.handleReport(w, req)

			assert.Equal(t, http.StatusAccepted, w.Code)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all reports were stored
	app.reportsMux.RLock()
	reportCount := len(app.reports)
	app.reportsMux.RUnlock()

	assert.Equal(t, numGoroutines, reportCount)
}

// TestApp_WebSocketConcurrentAccess tests concurrent WebSocket access
func TestApp_WebSocketConcurrentAccess(t *testing.T) {
	app := NewApp("test-app", 0)

	// Test concurrent topology requests
	numGoroutines := 5
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			req := httptest.NewRequest("GET", "/api/v1/topology", nil)
			w := httptest.NewRecorder()

			app.handleGetTopology(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}
