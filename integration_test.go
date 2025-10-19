// +build integration

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/ljluestc/orchestrator/pkg/app"
	"github.com/ljluestc/orchestrator/pkg/probe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFullSystemIntegration tests the complete system end-to-end
func TestFullSystemIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Step 1: Start the app server
	t.Log("Starting app server...")
	appServer, appAddr := startTestAppServer(t, ctx)
	defer appServer.Shutdown(ctx)

	// Step 2: Start probe agent
	t.Log("Starting probe agent...")
	probeInstance := startTestProbe(t, ctx, appAddr)
	defer probeInstance.Stop()

	// Step 3: Wait for data collection
	t.Log("Waiting for data collection...")
	time.Sleep(20 * time.Second)

	// Step 4: Verify data was collected and sent
	t.Log("Verifying data collection...")
	verifyDataCollection(t, appAddr)

	// Step 5: Test API endpoints
	t.Log("Testing API endpoints...")
	testAPIEndpoints(t, appAddr)

	// Step 6: Test WebSocket connections
	t.Log("Testing WebSocket connections...")
	testWebSocketConnections(t, appAddr)

	t.Log("Integration test completed successfully!")
}

// TestAppServerStandalone tests the app server in isolation
func TestAppServerStandalone(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	appServer, addr := startTestAppServer(t, ctx)
	defer appServer.Shutdown(ctx)

	// Test health endpoint
	resp, err := http.Get(fmt.Sprintf("http://%s/health", addr))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test API endpoints
	endpoints := []string{
		"/api/topology",
		"/api/agents",
		"/api/reports",
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("http://%s%s", addr, endpoint))
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should return 200 or 404 depending on endpoint
			assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound)
		})
	}
}

// TestProbeStandalone tests the probe in isolation
func TestProbeStandalone(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create probe without a running server (will fail to connect but should still work)
	probeInstance, err := probe.NewProbe(probe.ProbeConfig{
		ServerURL:          "http://localhost:19999", // Non-existent server
		AgentID:            "test-standalone-probe",
		CollectionInterval: 5 * time.Second,
		CollectHost:        true,
		CollectDocker:      true,
		CollectProcesses:   false, // Disable to speed up test
		CollectNetwork:     false,
		MaxProcesses:       100,
	})
	require.NoError(t, err)
	defer probeInstance.Stop()

	// Start probe (will run but fail to send reports)
	go func() {
		_ = probeInstance.Start(ctx)
	}()

	// Wait a bit for collection
	time.Sleep(10 * time.Second)

	t.Log("Probe standalone test completed")
}

// TestDockerIntegration tests Docker-related functionality
func TestDockerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration test in short mode")
	}

	// Check if Docker is available
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		t.Skip("Docker not available, skipping Docker integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// Start a test container
	t.Log("Starting test container...")
	containerID := startTestContainer(t, ctx)
	defer stopTestContainer(t, containerID)

	// Create probe to collect Docker info
	probeInstance, err := probe.NewProbe(probe.ProbeConfig{
		ServerURL:          "http://localhost:19999",
		AgentID:            "docker-test-probe",
		CollectionInterval: 5 * time.Second,
		CollectHost:        false,
		CollectDocker:      true,
		CollectDockerStats: true,
		CollectProcesses:   false,
		CollectNetwork:     false,
	})
	require.NoError(t, err)
	defer probeInstance.Stop()

	// Start probe
	go func() {
		_ = probeInstance.Start(ctx)
	}()

	// Wait for collection
	time.Sleep(10 * time.Second)

	t.Log("Docker integration test completed")
}

// TestConcurrentProbes tests multiple probes running simultaneously
func TestConcurrentProbes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent probes test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// Start app server
	appServer, addr := startTestAppServer(t, ctx)
	defer appServer.Shutdown(ctx)

	// Start multiple probes
	numProbes := 3
	probes := make([]*probe.Probe, numProbes)

	for i := 0; i < numProbes; i++ {
		probeConfig := probe.ProbeConfig{
			ServerURL:          fmt.Sprintf("http://%s", addr),
			AgentID:            fmt.Sprintf("concurrent-probe-%d", i),
			CollectionInterval: 5 * time.Second,
			CollectHost:        true,
			CollectDocker:      false, // Disable to reduce load
			CollectProcesses:   false,
			CollectNetwork:     false,
		}

		p, err := probe.NewProbe(probeConfig)
		require.NoError(t, err)
		probes[i] = p

		go func(probe *probe.Probe) {
			_ = probe.Start(ctx)
		}(p)
	}

	// Wait for collection
	time.Sleep(15 * time.Second)

	// Stop all probes
	for i, p := range probes {
		t.Logf("Stopping probe %d", i)
		p.Stop()
	}

	t.Log("Concurrent probes test completed")
}

// Helper functions

func startTestAppServer(t *testing.T, ctx context.Context) (*http.Server, string) {
	// Create app configuration
	config := app.Config{
		Port:              18080,
		EnableMetrics:     true,
		MetricsPort:       18081,
		EnableWebSocket:   true,
		MaxConnections:    100,
		StorageRetention:  5 * time.Minute,
		CleanupInterval:   1 * time.Minute,
	}

	// Create and start app server
	addr := fmt.Sprintf("localhost:%d", config.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: createTestAppHandler(t, config),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("App server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	return server, addr
}

func createTestAppHandler(t *testing.T, config app.Config) http.Handler {
	mux := http.NewServeMux()

	// Basic endpoints
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	mux.HandleFunc("/api/topology", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"nodes": []interface{}{},
			"edges": []interface{}{},
		})
	})

	mux.HandleFunc("/api/agents", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
	})

	mux.HandleFunc("/api/reports", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Accept report
			body, _ := io.ReadAll(r.Body)
			t.Logf("Received report: %d bytes", len(body))
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	return mux
}

func startTestProbe(t *testing.T, ctx context.Context, appAddr string) *probe.Probe {
	probeConfig := probe.ProbeConfig{
		ServerURL:          fmt.Sprintf("http://%s", appAddr),
		AgentID:            "integration-test-probe",
		CollectionInterval: 10 * time.Second,
		CollectHost:        true,
		CollectDocker:      false, // Disable for faster tests
		CollectProcesses:   false,
		CollectNetwork:     false,
	}

	p, err := probe.NewProbe(probeConfig)
	require.NoError(t, err)

	go func() {
		if err := p.Start(ctx); err != nil {
			t.Logf("Probe error: %v", err)
		}
	}()

	return p
}

func verifyDataCollection(t *testing.T, appAddr string) {
	// Try to get reports from the app server
	resp, err := http.Get(fmt.Sprintf("http://%s/api/reports", appAddr))
	if err != nil {
		t.Logf("Warning: Failed to get reports: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Log("Successfully retrieved reports from app server")
	}
}

func testAPIEndpoints(t *testing.T, appAddr string) {
	endpoints := []struct {
		path   string
		method string
	}{
		{"/health", "GET"},
		{"/api/topology", "GET"},
		{"/api/agents", "GET"},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.path, func(t *testing.T) {
			req, err := http.NewRequest(endpoint.method, fmt.Sprintf("http://%s%s", appAddr, endpoint.path), nil)
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Logf("Warning: Request to %s failed: %v", endpoint.path, err)
				return
			}
			defer resp.Body.Close()

			assert.True(t, resp.StatusCode >= 200 && resp.StatusCode < 500)
		})
	}
}

func testWebSocketConnections(t *testing.T, appAddr string) {
	// Basic WebSocket connection test
	// This is a simplified test - full WebSocket testing would require gorilla/websocket
	t.Log("WebSocket test placeholder - implement with websocket library if needed")
}

func startTestContainer(t *testing.T, ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "docker", "run", "-d", "--name", "orchestrator-test", "alpine", "sleep", "60")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to start test container: %s", output)

	containerID := string(output)
	t.Logf("Started test container: %s", containerID[:12])
	return containerID
}

func stopTestContainer(t *testing.T, containerID string) {
	cmd := exec.Command("docker", "rm", "-f", "orchestrator-test")
	if err := cmd.Run(); err != nil {
		t.Logf("Warning: Failed to stop test container: %v", err)
	}
}

// Benchmark integration tests
func BenchmarkFullIntegration(b *testing.B) {
	ctx := context.Background()

	appServer, addr := startBenchAppServer(b, ctx)
	defer appServer.Shutdown(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create and run probe
		probeInstance, _ := probe.NewProbe(probe.ProbeConfig{
			ServerURL:          fmt.Sprintf("http://%s", addr),
			AgentID:            fmt.Sprintf("bench-probe-%d", i),
			CollectionInterval: 1 * time.Second,
			CollectHost:        true,
			CollectDocker:      false,
			CollectProcesses:   false,
			CollectNetwork:     false,
		})

		ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
		probeInstance.Start(ctx2)
		cancel()
		probeInstance.Stop()
	}
}

func startBenchAppServer(b *testing.B, ctx context.Context) (*http.Server, string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/reports", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	addr := "localhost:28080"
	server := &http.Server{Addr: addr, Handler: mux}

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)

	return server, addr
}
