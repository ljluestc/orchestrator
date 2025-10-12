package ui

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewWebUI(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		port        int
		topologyURL string
	}{
		{
			name:        "Valid WebUI",
			id:          "webui-1",
			port:        8080,
			topologyURL: "http://localhost:8080",
		},
		{
			name:        "Different port",
			id:          "webui-2",
			port:        9090,
			topologyURL: "http://localhost:9090",
		},
		{
			name:        "Empty ID",
			id:          "",
			port:        8080,
			topologyURL: "http://localhost:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webUI := NewWebUI(tt.id, tt.port, tt.topologyURL)

			assert.Equal(t, tt.id, webUI.ID)
			assert.Equal(t, tt.port, webUI.Port)
			assert.Equal(t, tt.topologyURL, webUI.TopologyURL)
			assert.Nil(t, webUI.server)
		})
	}
}

func TestWebUI_Start(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		port        int
		topologyURL string
		expectError bool
	}{
		{
			name:        "Valid start",
			id:          "webui-1",
			port:        0, // Use port 0 for testing
			topologyURL: "http://localhost:8080",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webUI := NewWebUI(tt.id, tt.port, tt.topologyURL)

			// Start server in a goroutine
			errChan := make(chan error, 1)
			go func() {
				errChan <- webUI.Start()
			}()

			// Give server time to start
			time.Sleep(100 * time.Millisecond)

			// Stop the server
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := webUI.server.Shutdown(ctx)
			if err != nil {
				t.Logf("Server shutdown error: %v", err)
			}

			// Check for start error
			select {
			case err := <-errChan:
				if tt.expectError {
					assert.Error(t, err)
				} else {
					// Server should have started successfully, but may get "Server closed" when shut down
					if err != nil && err.Error() != "http: Server closed" {
						assert.NoError(t, err)
					}
				}
			case <-time.After(1 * time.Second):
				t.Fatal("Server start timeout")
			}
		})
	}
}

func TestWebUI_Stop(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		port        int
		topologyURL string
		expectError bool
	}{
		{
			name:        "Stop without start",
			id:          "webui-1",
			port:        8080,
			topologyURL: "http://localhost:8080",
			expectError: false, // Should not error when stopping without starting
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webUI := NewWebUI(tt.id, tt.port, tt.topologyURL)

			err := webUI.Stop()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWebUI_StartStop(t *testing.T) {
	webUI := NewWebUI("test-webui", 0, "http://localhost:8080")

	// Start server
	errChan := make(chan error, 1)
	go func() {
		errChan <- webUI.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	err := webUI.Stop()
	assert.NoError(t, err)

	// Check for start error
	select {
	case err := <-errChan:
		// Server should have started successfully, but may get "Server closed" when shut down
		if err != nil && err.Error() != "http: Server closed" {
			assert.NoError(t, err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Server start timeout")
	}
}

func TestWebUI_SetupRoutes(t *testing.T) {
	webUI := NewWebUI("test-webui", 8080, "http://localhost:8080")
	router := webUI.setupRoutes()

	assert.NotNil(t, router)

	// Test that routes are properly configured
	testCases := []struct {
		method string
		path   string
	}{
		{"GET", "/"},
		{"GET", "/dashboard"},
		{"GET", "/topology"},
		{"GET", "/topology/containers"},
		{"GET", "/api/topology"},
		{"GET", "/api/topology/nodes"},
		{"GET", "/api/topology/edges"},
		{"GET", "/api/topology/search"},
		{"POST", "/api/topology/filter"},
		{"GET", "/api/views"},
		{"GET", "/api/metrics"},
		{"POST", "/api/containers/test-id/start"},
		{"POST", "/api/containers/test-id/stop"},
		{"POST", "/api/containers/test-id/restart"},
		{"POST", "/api/containers/test-id/pause"},
		{"POST", "/api/containers/test-id/unpause"},
		{"GET", "/api/containers/test-id/logs"},
		{"GET", "/ws"},
		{"GET", "/health"},
	}

	for _, tc := range testCases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			// Should not return 404 (route not found)
			assert.NotEqual(t, http.StatusNotFound, rr.Code, "Route %s %s should be found", tc.method, tc.path)
		})
	}
}

func TestWebUI_HandleDashboard(t *testing.T) {
	webUI := NewWebUI("test-webui", 8080, "http://localhost:8080")

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	webUI.handleDashboard(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/html", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Body.String(), "Mesos-Docker Orchestration Platform")
	assert.Contains(t, rr.Body.String(), "TopologyVisualizer")
}

func TestWebUI_HandleTopology(t *testing.T) {
	webUI := NewWebUI("test-webui", 8080, "http://localhost:8080")

	req := httptest.NewRequest("GET", "/topology", nil)
	rr := httptest.NewRecorder()

	webUI.handleTopology(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/html", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Body.String(), "Mesos-Docker Orchestration Platform")
}

func TestWebUI_HandleTopologyView(t *testing.T) {
	webUI := NewWebUI("test-webui", 8080, "http://localhost:8080")

	req := httptest.NewRequest("GET", "/topology/containers", nil)
	rr := httptest.NewRecorder()

	webUI.handleTopologyView(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/html", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Body.String(), "Mesos-Docker Orchestration Platform")
}

func TestWebUI_HandleAPIProxy(t *testing.T) {
	webUI := NewWebUI("test-webui", 8080, "http://localhost:8080")

	testCases := []struct {
		method string
		path   string
	}{
		{"GET", "/api/topology"},
		{"GET", "/api/topology/nodes"},
		{"GET", "/api/topology/edges"},
		{"GET", "/api/topology/search"},
		{"POST", "/api/topology/filter"},
		{"GET", "/api/views"},
		{"GET", "/api/metrics"},
	}

	for _, tc := range testCases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			webUI.handleAPIProxy(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

			// Should return mock data
			body := rr.Body.String()
			assert.Contains(t, body, "nodes")
			assert.Contains(t, body, "edges")
			assert.Contains(t, body, "metrics")
		})
	}
}

func TestWebUI_HandleWebSocketProxy(t *testing.T) {
	webUI := NewWebUI("test-webui", 8080, "http://localhost:8080")

	req := httptest.NewRequest("GET", "/ws", nil)
	rr := httptest.NewRecorder()

	webUI.handleWebSocketProxy(rr, req)

	assert.Equal(t, http.StatusNotImplemented, rr.Code)
	assert.Contains(t, rr.Body.String(), "WebSocket proxy not implemented")
}

func TestWebUI_HandleHealth(t *testing.T) {
	webUI := NewWebUI("test-webui", 8080, "http://localhost:8080")

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	webUI.handleHealth(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Body.String(), "healthy")
}

func TestWebUI_ConcurrentAccess(t *testing.T) {
	webUI := NewWebUI("test-webui", 8080, "http://localhost:8080")

	const numGoroutines = 10
	const numRequests = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numRequests; j++ {
				req := httptest.NewRequest("GET", "/health", nil)
				rr := httptest.NewRecorder()
				webUI.handleHealth(rr, req)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestWebUI_ErrorHandling(t *testing.T) {
	webUI := NewWebUI("test-webui", 8080, "http://localhost:8080")

	t.Run("Invalid method", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/health", nil)
		rr := httptest.NewRecorder()

		webUI.handleHealth(rr, req)

		// Should still work for POST to health endpoint
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Large request", func(t *testing.T) {
		// Create a large request body
		largeBody := make([]byte, 1024*1024) // 1MB
		req := httptest.NewRequest("POST", "/api/topology/filter", bytes.NewReader(largeBody))
		rr := httptest.NewRecorder()

		webUI.handleAPIProxy(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestWebUI_StaticAssets(t *testing.T) {
	webUI := NewWebUI("test-webui", 8080, "http://localhost:8080")
	router := webUI.setupRoutes()

	req := httptest.NewRequest("GET", "/static/test.css", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Static assets should return 404 if static directory doesn't exist
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestWebUI_ContainerControlEndpoints(t *testing.T) {
	webUI := NewWebUI("test-webui", 8080, "http://localhost:8080")

	testCases := []struct {
		method string
		path   string
	}{
		{"POST", "/api/containers/test-id/start"},
		{"POST", "/api/containers/test-id/stop"},
		{"POST", "/api/containers/test-id/restart"},
		{"POST", "/api/containers/test-id/pause"},
		{"POST", "/api/containers/test-id/unpause"},
		{"GET", "/api/containers/test-id/logs"},
	}

	for _, tc := range testCases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			webUI.handleAPIProxy(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
		})
	}
}

func TestWebUI_HTMLContent(t *testing.T) {
	webUI := NewWebUI("test-webui", 8080, "http://localhost:8080")

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	webUI.handleDashboard(rr, req)

	body := rr.Body.String()

	// Check for essential HTML elements
	assert.Contains(t, body, "<!DOCTYPE html>")
	assert.Contains(t, body, "<html")
	assert.Contains(t, body, "<head>")
	assert.Contains(t, body, "<body>")
	assert.Contains(t, body, "</html>")

	// Check for JavaScript libraries
	assert.Contains(t, body, "d3js.org")
	assert.Contains(t, body, "cytoscape")
	assert.Contains(t, body, "TopologyVisualizer")

	// Check for CSS classes
	assert.Contains(t, body, "header")
	assert.Contains(t, body, "nav")
	assert.Contains(t, body, "main")
	assert.Contains(t, body, "sidebar")
	assert.Contains(t, body, "content")
	assert.Contains(t, body, "graph-container")
}

func TestWebUI_WebSocketIntegration(t *testing.T) {
	webUI := NewWebUI("test-webui", 8080, "http://localhost:8080")

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	webUI.handleDashboard(rr, req)

	body := rr.Body.String()

	// Check for WebSocket connection code
	assert.Contains(t, body, "WebSocket")
	assert.Contains(t, body, "window.location.protocol")
	assert.Contains(t, body, "window.location.host")
	assert.Contains(t, body, "/ws")
}

func TestWebUI_ViewSelector(t *testing.T) {
	webUI := NewWebUI("test-webui", 8080, "http://localhost:8080")

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	webUI.handleDashboard(rr, req)

	body := rr.Body.String()

	// Check for view selector buttons
	assert.Contains(t, body, "data-view=\"processes\"")
	assert.Contains(t, body, "data-view=\"containers\"")
	assert.Contains(t, body, "data-view=\"hosts\"")
	assert.Contains(t, body, "data-view=\"pods\"")
	assert.Contains(t, body, "data-view=\"services\"")
}

func TestWebUI_SearchAndFilter(t *testing.T) {
	webUI := NewWebUI("test-webui", 8080, "http://localhost:8080")

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	webUI.handleDashboard(rr, req)

	body := rr.Body.String()

	// Check for search functionality
	assert.Contains(t, body, "searchInput")
	assert.Contains(t, body, "mainSearchInput")
	assert.Contains(t, body, "filterHealthy")
	assert.Contains(t, body, "filterWarning")
	assert.Contains(t, body, "filterCritical")
}

func BenchmarkWebUI_HandleDashboard(b *testing.B) {
	webUI := NewWebUI("test-webui", 8080, "http://localhost:8080")

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		webUI.handleDashboard(rr, req)
	}
}

func BenchmarkWebUI_HandleAPIProxy(b *testing.B) {
	webUI := NewWebUI("test-webui", 8080, "http://localhost:8080")

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/topology", nil)
		rr := httptest.NewRecorder()
		webUI.handleAPIProxy(rr, req)
	}
}

func BenchmarkWebUI_HandleHealth(b *testing.B) {
	webUI := NewWebUI("test-webui", 8080, "http://localhost:8080")

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()
		webUI.handleHealth(rr, req)
	}
}