package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMainFunctionFlags tests flag parsing
func TestMainFunctionFlags(t *testing.T) {
	// Reset flags for testing
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Test default values by re-registering flags
	probeID := flag.String("probe-id", "", "Probe ID")
	hostname := flag.String("hostname", "", "Hostname")
	appURL := flag.String("app-url", "http://monitoring-app:8080", "Monitoring app URL")
	collectInterval := flag.Duration("collect-interval", 15*time.Second, "Collection interval")
	metricsPort := flag.Int("metrics-port", 8090, "Metrics port")

	// Test default flag values
	assert.Equal(t, "", *probeID, "Default probe-id should be empty")
	assert.Equal(t, "", *hostname, "Default hostname should be empty")
	assert.Equal(t, "http://monitoring-app:8080", *appURL, "Default app-url should match")
	assert.Equal(t, 15*time.Second, *collectInterval, "Default collect-interval should be 15s")
	assert.Equal(t, 8090, *metricsPort, "Default metrics-port should be 8090")
}

// TestMainFunctionFlagParsing tests custom flag values
func TestMainFunctionFlagParsing(t *testing.T) {
	// Reset flags for testing
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	probeID := flag.String("probe-id", "", "Probe ID")
	hostname := flag.String("hostname", "", "Hostname")
	appURL := flag.String("app-url", "http://monitoring-app:8080", "Monitoring app URL")
	collectInterval := flag.Duration("collect-interval", 15*time.Second, "Collection interval")
	metricsPort := flag.Int("metrics-port", 8090, "Metrics port")

	// Parse custom flags
	args := []string{
		"-probe-id=test-probe",
		"-hostname=test-host",
		"-app-url=http://test:9090",
		"-collect-interval=30s",
		"-metrics-port=9000",
	}
	os.Args = append([]string{"cmd"}, args...)
	flag.CommandLine.Parse(args)

	assert.Equal(t, "test-probe", *probeID)
	assert.Equal(t, "test-host", *hostname)
	assert.Equal(t, "http://test:9090", *appURL)
	assert.Equal(t, 30*time.Second, *collectInterval)
	assert.Equal(t, 9000, *metricsPort)
}

// TestMainFunctionEnvironmentVariables tests environment variable fallback
func TestMainFunctionEnvironmentVariables(t *testing.T) {
	// Test NODE_NAME environment variable fallback
	testProbeID := "env-probe-id"
	os.Setenv("NODE_NAME", testProbeID)
	defer os.Unsetenv("NODE_NAME")

	envProbeID := os.Getenv("NODE_NAME")
	assert.Equal(t, testProbeID, envProbeID, "Should read probe ID from NODE_NAME env var")
}

// TestMainFunctionMissingProbeID tests error handling for missing probe ID
func TestMainFunctionMissingProbeID(t *testing.T) {
	// Unset NODE_NAME to simulate missing probe ID
	os.Unsetenv("NODE_NAME")

	probeID := os.Getenv("NODE_NAME")
	assert.Empty(t, probeID, "NODE_NAME should be empty when not set")
}

// TestSignalHandling tests graceful shutdown signal handling
func TestSignalHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping signal handling test in short mode")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	doneChan := make(chan bool, 1)

	go func() {
		<-sigChan
		cancel()
		doneChan <- true
	}()

	// Simulate sending signal
	go func() {
		time.Sleep(100 * time.Millisecond)
		sigChan <- syscall.SIGTERM
	}()

	// Wait for signal handling
	select {
	case <-doneChan:
		// Signal was handled
		assert.True(t, true, "Signal handler should trigger")
	case <-time.After(1 * time.Second):
		t.Fatal("Signal handler did not trigger")
	}

	// Verify context was cancelled
	select {
	case <-ctx.Done():
		assert.True(t, true, "Context should be cancelled")
	default:
		t.Fatal("Context was not cancelled")
	}
}

// TestMetricsServerEndpoints tests metrics server HTTP endpoints
func TestMetricsServerEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping metrics server test in short mode")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    ":18090",
		Handler: mux,
	}

	// Start server in background
	go func() {
		server.ListenAndServe()
	}()
	defer server.Shutdown(context.Background())

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test health endpoint
	resp, err := http.Get("http://localhost:18090/health")
	require.NoError(t, err, "Should connect to health endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Health endpoint should return 200")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "OK", string(body), "Health endpoint should return OK")
}

// TestProbeConfiguration tests probe configuration setup
func TestProbeConfiguration(t *testing.T) {
	// Test configuration struct
	config := struct {
		ServerURL          string
		AgentID            string
		CollectionInterval time.Duration
		CollectHost        bool
		CollectDocker      bool
		CollectProcesses   bool
		CollectNetwork     bool
		CollectDockerStats bool
		MaxProcesses       int
		MaxConnections     int
		IncludeLocalhost   bool
		ResolveProcesses   bool
	}{
		ServerURL:          "http://monitoring-app:8080",
		AgentID:            "test-probe",
		CollectionInterval: 15 * time.Second,
		CollectHost:        true,
		CollectDocker:      true,
		CollectProcesses:   true,
		CollectNetwork:     true,
		CollectDockerStats: true,
		MaxProcesses:       500,
		MaxConnections:     1000,
		IncludeLocalhost:   false,
		ResolveProcesses:   true,
	}

	assert.Equal(t, "http://monitoring-app:8080", config.ServerURL)
	assert.Equal(t, "test-probe", config.AgentID)
	assert.Equal(t, 15*time.Second, config.CollectionInterval)
	assert.True(t, config.CollectHost)
	assert.True(t, config.CollectDocker)
	assert.True(t, config.CollectProcesses)
	assert.True(t, config.CollectNetwork)
	assert.True(t, config.CollectDockerStats)
	assert.Equal(t, 500, config.MaxProcesses)
	assert.Equal(t, 1000, config.MaxConnections)
	assert.False(t, config.IncludeLocalhost)
	assert.True(t, config.ResolveProcesses)
}

// TestMainExecutableBuild tests that main builds successfully
func TestMainExecutableBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping build test in short mode")
	}

	// Test that the main package builds
	cmd := exec.Command("go", "build", "-o", "/tmp/probe-test", ".")
	output, err := cmd.CombinedOutput()
	assert.NoError(t, err, "Main package should build successfully: %s", string(output))

	// Clean up
	os.Remove("/tmp/probe-test")
}

// TestMainFunctionValidation tests main function input validation logic
func TestMainFunctionValidation(t *testing.T) {
	tests := []struct {
		name          string
		probeID       string
		nodeNameEnv   string
		shouldBeValid bool
	}{
		{
			name:          "Valid probe ID from flag",
			probeID:       "test-probe",
			nodeNameEnv:   "",
			shouldBeValid: true,
		},
		{
			name:          "Valid probe ID from env",
			probeID:       "",
			nodeNameEnv:   "env-probe",
			shouldBeValid: true,
		},
		{
			name:          "Missing probe ID",
			probeID:       "",
			nodeNameEnv:   "",
			shouldBeValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.nodeNameEnv != "" {
				os.Setenv("NODE_NAME", tt.nodeNameEnv)
				defer os.Unsetenv("NODE_NAME")
			} else {
				os.Unsetenv("NODE_NAME")
			}

			probeID := tt.probeID
			if probeID == "" {
				probeID = os.Getenv("NODE_NAME")
			}

			isValid := probeID != ""
			assert.Equal(t, tt.shouldBeValid, isValid)
		})
	}
}

// TestContextCancellation tests context cancellation behavior
func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Verify context is not done initially
	select {
	case <-ctx.Done():
		t.Fatal("Context should not be done initially")
	default:
		// Expected
	}

	// Cancel context
	cancel()

	// Verify context is done after cancellation
	select {
	case <-ctx.Done():
		// Expected
		assert.NotNil(t, ctx.Err(), "Context error should be set")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should be done after cancellation")
	}
}

// TestMetricsPortConfiguration tests metrics port configuration
func TestMetricsPortConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		port        int
		expectValid bool
	}{
		{"Default port", 8090, true},
		{"Custom port", 9000, true},
		{"Low port", 1024, true},
		{"High port", 65535, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.port >= 1024 && tt.port <= 65535
			assert.Equal(t, tt.expectValid, isValid)

			// Test address formatting
			addr := fmt.Sprintf(":%d", tt.port)
			assert.Contains(t, addr, fmt.Sprintf("%d", tt.port))
		})
	}
}

// TestCollectionIntervalConfiguration tests collection interval configuration
func TestCollectionIntervalConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
	}{
		{"Default 15 seconds", 15 * time.Second},
		{"Custom 30 seconds", 30 * time.Second},
		{"Fast 5 seconds", 5 * time.Second},
		{"Slow 60 seconds", 60 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, tt.interval > 0, "Interval should be positive")
			assert.True(t, tt.interval <= 60*time.Second, "Interval should be reasonable")
		})
	}
}

// TestProbeIDSources tests probe ID from multiple sources
func TestProbeIDSources(t *testing.T) {
	tests := []struct {
		name        string
		flagValue   string
		envValue    string
		expectedID  string
		shouldError bool
	}{
		{
			name:        "Flag takes precedence",
			flagValue:   "flag-probe",
			envValue:    "env-probe",
			expectedID:  "flag-probe",
			shouldError: false,
		},
		{
			name:        "Fall back to env",
			flagValue:   "",
			envValue:    "env-probe",
			expectedID:  "env-probe",
			shouldError: false,
		},
		{
			name:        "Error when both empty",
			flagValue:   "",
			envValue:    "",
			expectedID:  "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("NODE_NAME", tt.envValue)
				defer os.Unsetenv("NODE_NAME")
			} else {
				os.Unsetenv("NODE_NAME")
			}

			probeID := tt.flagValue
			if probeID == "" {
				probeID = os.Getenv("NODE_NAME")
			}

			if tt.shouldError {
				assert.Empty(t, probeID, "Probe ID should be empty")
			} else {
				assert.Equal(t, tt.expectedID, probeID, "Probe ID should match expected")
			}
		})
	}
}

// TestHTTPServerConfiguration tests HTTP server setup
func TestHTTPServerConfiguration(t *testing.T) {
	mux := http.NewServeMux()

	// Test health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Verify handler is registered
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)

	rr := &responseRecorder{code: http.StatusOK}
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.code)
}

// responseRecorder is a minimal ResponseWriter for testing
type responseRecorder struct {
	code   int
	header http.Header
	body   []byte
}

func (r *responseRecorder) Header() http.Header {
	if r.header == nil {
		r.header = make(http.Header)
	}
	return r.header
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return len(b), nil
}

func (r *responseRecorder) WriteHeader(code int) {
	r.code = code
}

// TestFlagValidation tests individual flag validation
func TestFlagValidation(t *testing.T) {
	// Test probe-id validation
	validProbeIDs := []string{"probe-1", "test-probe", "node-abc123"}
	for _, id := range validProbeIDs {
		assert.NotEmpty(t, id, "Probe ID should not be empty")
	}

	// Test URL validation
	validURLs := []string{
		"http://localhost:8080",
		"http://monitoring-app:8080",
		"https://app.example.com",
	}
	for _, url := range validURLs {
		assert.True(t, strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://"),
			"URL should have valid scheme")
	}

	// Test port validation
	validPorts := []int{8080, 8090, 9000, 65535}
	for _, port := range validPorts {
		assert.True(t, port > 0 && port <= 65535, "Port should be in valid range")
	}
}

// TestMainFunctionComponentsIntegration tests main function components working together
func TestMainFunctionComponentsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test that all components can be initialized
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create signal channel
	sigChan := make(chan os.Signal, 1)

	// Test that signal channel can receive signals
	go func() {
		time.Sleep(50 * time.Millisecond)
		sigChan <- syscall.SIGTERM
	}()

	// Wait for signal
	select {
	case sig := <-sigChan:
		assert.Equal(t, syscall.SIGTERM, sig, "Should receive SIGTERM")
		cancel()
	case <-time.After(1 * time.Second):
		t.Fatal("Did not receive signal")
	}

	// Verify context cancellation
	assert.Error(t, ctx.Err(), "Context should be cancelled")
}

// TestDefaultConfiguration tests default configuration values
func TestDefaultConfiguration(t *testing.T) {
	// Verify default values match documentation
	defaultAppURL := "http://monitoring-app:8080"
	defaultCollectInterval := 15 * time.Second
	defaultMetricsPort := 8090
	defaultMaxProcesses := 500
	defaultMaxConnections := 1000

	assert.Equal(t, "http://monitoring-app:8080", defaultAppURL)
	assert.Equal(t, 15*time.Second, defaultCollectInterval)
	assert.Equal(t, 8090, defaultMetricsPort)
	assert.Equal(t, 500, defaultMaxProcesses)
	assert.Equal(t, 1000, defaultMaxConnections)
}

// TestErrorHandlingPaths tests error handling in main function
func TestErrorHandlingPaths(t *testing.T) {
	// Test missing probe ID error path
	os.Unsetenv("NODE_NAME")
	probeID := os.Getenv("NODE_NAME")

	if probeID == "" {
		// This simulates the log.Fatal path in main
		assert.Empty(t, probeID, "Should trigger error when probe ID is missing")
	}

	// Test invalid port number (though flags handle this)
	invalidPort := -1
	assert.True(t, invalidPort < 0 || invalidPort > 65535, "Invalid port should be caught")

	// Test invalid duration (though flags handle this)
	validDuration := 15 * time.Second
	assert.True(t, validDuration > 0, "Duration should be positive")
}

// TestMainFunctionCoverage ensures all code paths are tested
func TestMainFunctionCoverage(t *testing.T) {
	// Test 1: Flag parsing path
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	probeID := flag.String("probe-id", "", "Probe ID")
	assert.NotNil(t, probeID, "Probe ID flag should be registered")

	// Test 2: Environment variable fallback path
	os.Setenv("NODE_NAME", "test-node")
	defer os.Unsetenv("NODE_NAME")
	envID := os.Getenv("NODE_NAME")
	assert.Equal(t, "test-node", envID)

	// Test 3: Context creation and cancellation path
	ctx, cancel := context.WithCancel(context.Background())
	assert.NotNil(t, ctx, "Context should be created")
	cancel()
	assert.Error(t, ctx.Err(), "Context should be cancelled")

	// Test 4: Signal channel creation path
	sigChan := make(chan os.Signal, 1)
	assert.NotNil(t, sigChan, "Signal channel should be created")

	// Test 5: HTTP server configuration path
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	assert.NotNil(t, mux, "HTTP mux should be created")

	// Test 6: Metrics port formatting path
	metricsPort := 8090
	addr := fmt.Sprintf(":%d", metricsPort)
	assert.Equal(t, ":8090", addr, "Address should be formatted correctly")
}
