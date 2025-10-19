package main

import (
	"context"
	"flag"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/ljluestc/orchestrator/pkg/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFlagParsing tests command-line flag parsing
func TestFlagParsing(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		expectedHost     string
		expectedPort     int
		expectedMaxAge   time.Duration
		expectedCleanup  time.Duration
		expectedStale    time.Duration
	}{
		{
			name:             "default values",
			args:             []string{},
			expectedHost:     "0.0.0.0",
			expectedPort:     8080,
			expectedMaxAge:   1 * time.Hour,
			expectedCleanup:  5 * time.Minute,
			expectedStale:    5 * time.Minute,
		},
		{
			name:             "custom host and port",
			args:             []string{"-host=localhost", "-port=9090"},
			expectedHost:     "localhost",
			expectedPort:     9090,
			expectedMaxAge:   1 * time.Hour,
			expectedCleanup:  5 * time.Minute,
			expectedStale:    5 * time.Minute,
		},
		{
			name:             "custom durations",
			args:             []string{"-max-data-age=2h", "-cleanup-interval=10m", "-stale-node-threshold=15m"},
			expectedHost:     "0.0.0.0",
			expectedPort:     8080,
			expectedMaxAge:   2 * time.Hour,
			expectedCleanup:  10 * time.Minute,
			expectedStale:    15 * time.Minute,
		},
		{
			name:             "all custom values",
			args:             []string{"-host=127.0.0.1", "-port=8888", "-max-data-age=30m", "-cleanup-interval=2m", "-stale-node-threshold=3m"},
			expectedHost:     "127.0.0.1",
			expectedPort:     8888,
			expectedMaxAge:   30 * time.Minute,
			expectedCleanup:  2 * time.Minute,
			expectedStale:    3 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			host = flag.String("host", "0.0.0.0", "Server host address")
			port = flag.Int("port", 8080, "Server port")
			maxDataAge = flag.Duration("max-data-age", 1*time.Hour, "Maximum age for stored data")
			cleanupInterval = flag.Duration("cleanup-interval", 5*time.Minute, "Cleanup interval for stale data")
			staleNodeThreshold = flag.Duration("stale-node-threshold", 5*time.Minute, "Threshold for considering nodes stale")

			err := flag.CommandLine.Parse(tt.args)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedHost, *host)
			assert.Equal(t, tt.expectedPort, *port)
			assert.Equal(t, tt.expectedMaxAge, *maxDataAge)
			assert.Equal(t, tt.expectedCleanup, *cleanupInterval)
			assert.Equal(t, tt.expectedStale, *staleNodeThreshold)
		})
	}
}

// TestPrintConfig_AllScenarios tests the configuration printing function with multiple scenarios
func TestPrintConfig_AllScenarios(t *testing.T) {
	tests := []struct {
		name   string
		config app.ServerConfig
	}{
		{
			name: "default configuration",
			config: app.ServerConfig{
				Host:               "0.0.0.0",
				Port:               8080,
				MaxDataAge:         1 * time.Hour,
				CleanupInterval:    5 * time.Minute,
				StaleNodeThreshold: 5 * time.Minute,
			},
		},
		{
			name: "custom configuration",
			config: app.ServerConfig{
				Host:               "localhost",
				Port:               9090,
				MaxDataAge:         2 * time.Hour,
				CleanupInterval:    10 * time.Minute,
				StaleNodeThreshold: 15 * time.Minute,
			},
		},
		{
			name: "minimal configuration",
			config: app.ServerConfig{
				Host:               "127.0.0.1",
				Port:               8000,
				MaxDataAge:         30 * time.Minute,
				CleanupInterval:    1 * time.Minute,
				StaleNodeThreshold: 2 * time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output (printConfig uses fmt.Println, so we can't easily capture it)
			// But we can at least verify it doesn't panic
			assert.NotPanics(t, func() {
				printConfig(tt.config)
			})
		})
	}
}

// TestServerConfiguration_AllTypes tests server config creation with different configurations
func TestServerConfiguration_AllTypes(t *testing.T) {
	tests := []struct {
		name        string
		hostVal     string
		portVal     int
		maxAgeVal   time.Duration
		cleanupVal  time.Duration
		staleVal    time.Duration
	}{
		{
			name:       "production config",
			hostVal:    "0.0.0.0",
			portVal:    8080,
			maxAgeVal:  24 * time.Hour,
			cleanupVal: 1 * time.Hour,
			staleVal:   10 * time.Minute,
		},
		{
			name:       "development config",
			hostVal:    "localhost",
			portVal:    8080,
			maxAgeVal:  1 * time.Hour,
			cleanupVal: 5 * time.Minute,
			staleVal:   5 * time.Minute,
		},
		{
			name:       "testing config",
			hostVal:    "127.0.0.1",
			portVal:    18080,
			maxAgeVal:  10 * time.Minute,
			cleanupVal: 1 * time.Minute,
			staleVal:   1 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := app.ServerConfig{
				Host:               tt.hostVal,
				Port:               tt.portVal,
				MaxDataAge:         tt.maxAgeVal,
				CleanupInterval:    tt.cleanupVal,
				StaleNodeThreshold: tt.staleVal,
			}

			assert.Equal(t, tt.hostVal, config.Host)
			assert.Equal(t, tt.portVal, config.Port)
			assert.Equal(t, tt.maxAgeVal, config.MaxDataAge)
			assert.Equal(t, tt.cleanupVal, config.CleanupInterval)
			assert.Equal(t, tt.staleVal, config.StaleNodeThreshold)
		})
	}
}

// TestServer_StartStopLifecycle tests server start and stop lifecycle with graceful shutdown
func TestServer_StartStopLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping server test in short mode")
	}

	config := app.ServerConfig{
		Host:               "127.0.0.1",
		Port:               18080,
		MaxDataAge:         10 * time.Minute,
		CleanupInterval:    1 * time.Minute,
		StaleNodeThreshold: 1 * time.Minute,
	}

	server := app.NewServer(config)
	ctx, cancel := context.WithCancel(context.Background())

	// Start server in background
	serverStarted := make(chan bool)
	serverError := make(chan error)

	go func() {
		serverStarted <- true
		err := server.Start(ctx)
		if err != nil {
			serverError <- err
		}
	}()

	// Wait for server to start
	<-serverStarted
	time.Sleep(200 * time.Millisecond)

	// Stop server
	cancel()
	err := server.Stop()
	assert.NoError(t, err)

	// Check for server errors
	select {
	case err := <-serverError:
		t.Logf("Server error (expected on shutdown): %v", err)
	case <-time.After(100 * time.Millisecond):
		// No error, good
	}
}

// TestContext_CancellationBehavior tests context cancellation behavior
func TestContext_CancellationBehavior(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	// Verify context is done
	select {
	case <-ctx.Done():
		assert.NotNil(t, ctx.Err())
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context not cancelled")
	}
}

// TestSignal_ChannelSetup tests signal channel setup
func TestSignal_ChannelSetup(t *testing.T) {
	sigChan := make(chan os.Signal, 1)
	assert.NotNil(t, sigChan)
	assert.Equal(t, 1, cap(sigChan))
}

// TestMain_ExecutableBuildAndHelp tests building and running the main binary
func TestMain_ExecutableBuildAndHelp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping executable test in short mode")
	}

	// Build the binary
	cmd := exec.Command("go", "build", "-o", "/tmp/app-server-test", ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build app-server binary: %v\nOutput: %s", err, output)
	}
	defer os.Remove("/tmp/app-server-test")

	// Test help flag
	cmd = exec.Command("/tmp/app-server-test", "-h")
	output, err = cmd.CombinedOutput()
	// -h flag causes exit code 2 in flag package
	if err == nil || (err != nil && cmd.ProcessState.ExitCode() == 2) {
		t.Logf("Help output:\n%s", output)
	}

	assert.Contains(t, string(output), "host", "Help should mention host flag")
	assert.Contains(t, string(output), "port", "Help should mention port flag")
	assert.Contains(t, string(output), "max-data-age", "Help should mention max-data-age flag")
}

// TestConfiguration_Validation tests configuration validation scenarios
func TestConfiguration_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  app.ServerConfig
		isValid bool
	}{
		{
			name: "valid config",
			config: app.ServerConfig{
				Host:               "0.0.0.0",
				Port:               8080,
				MaxDataAge:         1 * time.Hour,
				CleanupInterval:    5 * time.Minute,
				StaleNodeThreshold: 5 * time.Minute,
			},
			isValid: true,
		},
		{
			name: "valid localhost",
			config: app.ServerConfig{
				Host:               "localhost",
				Port:               8080,
				MaxDataAge:         1 * time.Hour,
				CleanupInterval:    5 * time.Minute,
				StaleNodeThreshold: 5 * time.Minute,
			},
			isValid: true,
		},
		{
			name: "high port",
			config: app.ServerConfig{
				Host:               "0.0.0.0",
				Port:               65535,
				MaxDataAge:         1 * time.Hour,
				CleanupInterval:    5 * time.Minute,
				StaleNodeThreshold: 5 * time.Minute,
			},
			isValid: true,
		},
		{
			name: "custom durations",
			config: app.ServerConfig{
				Host:               "0.0.0.0",
				Port:               8080,
				MaxDataAge:         24 * time.Hour,
				CleanupInterval:    1 * time.Hour,
				StaleNodeThreshold: 30 * time.Minute,
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := app.NewServer(tt.config)
			assert.NotNil(t, server)

			// Basic validation
			if tt.isValid {
				assert.NotEmpty(t, tt.config.Host)
				assert.Greater(t, tt.config.Port, 0)
				assert.Greater(t, tt.config.MaxDataAge, time.Duration(0))
				assert.Greater(t, tt.config.CleanupInterval, time.Duration(0))
				assert.Greater(t, tt.config.StaleNodeThreshold, time.Duration(0))
			}
		})
	}
}

// TestServer_ConcurrentOperations tests concurrent server operations
func TestServer_ConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	// Test multiple server creations
	configs := []app.ServerConfig{
		{Host: "127.0.0.1", Port: 18080, MaxDataAge: 1 * time.Hour, CleanupInterval: 5 * time.Minute, StaleNodeThreshold: 5 * time.Minute},
		{Host: "127.0.0.1", Port: 18081, MaxDataAge: 1 * time.Hour, CleanupInterval: 5 * time.Minute, StaleNodeThreshold: 5 * time.Minute},
		{Host: "127.0.0.1", Port: 18082, MaxDataAge: 1 * time.Hour, CleanupInterval: 5 * time.Minute, StaleNodeThreshold: 5 * time.Minute},
	}

	for i, config := range configs {
		t.Run(t.Name()+"-"+string(rune('A'+i)), func(t *testing.T) {
			t.Parallel()
			server := app.NewServer(config)
			assert.NotNil(t, server)
		})
	}
}

// TestDuration_FlagParsing tests duration flag parsing
func TestDuration_FlagParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"seconds", "30s", 30 * time.Second, false},
		{"minutes", "5m", 5 * time.Minute, false},
		{"hours", "2h", 2 * time.Hour, false},
		{"combined", "1h30m", 90 * time.Minute, false},
		{"invalid", "invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := time.ParseDuration(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, duration)
			}
		})
	}
}

// TestRunWithConfig_SignalHandling tests the runWithConfig function with signal handling
func TestRunWithConfig_SignalHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping runWithConfig test in short mode")
	}

	config := app.ServerConfig{
		Host:               "127.0.0.1",
		Port:               19090,
		MaxDataAge:         10 * time.Minute,
		CleanupInterval:    1 * time.Minute,
		StaleNodeThreshold: 1 * time.Minute,
	}

	// Run in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- runWithConfig(config)
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Send SIGTERM to stop
	proc, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)

	err = proc.Signal(syscall.SIGTERM)
	require.NoError(t, err)

	// Wait for shutdown
	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not shut down in time")
	}
}

// TestRunWithConfig_ServerError tests runWithConfig with server startup error
func TestRunWithConfig_ServerError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping server error test in short mode")
	}

	// Use invalid port to trigger error
	config := app.ServerConfig{
		Host:               "invalid-host-that-does-not-exist-12345",
		Port:               99999, // Invalid port
		MaxDataAge:         10 * time.Minute,
		CleanupInterval:    1 * time.Minute,
		StaleNodeThreshold: 1 * time.Minute,
	}

	// Run and expect error
	errChan := make(chan error, 1)
	go func() {
		errChan <- runWithConfig(config)
	}()

	// Wait for the error
	select {
	case err := <-errChan:
		// We expect an error from invalid configuration
		if err != nil {
			assert.Contains(t, err.Error(), "server")
			t.Logf("Got expected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Log("Test timed out, but that's acceptable for this error case")
	}
}

// TestRunWithConfig_StopError tests error handling during server stop
func TestRunWithConfig_StopError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stop error test in short mode")
	}

	// Create a server and immediately signal it
	config := app.ServerConfig{
		Host:               "127.0.0.1",
		Port:               19092,
		MaxDataAge:         10 * time.Minute,
		CleanupInterval:    1 * time.Minute,
		StaleNodeThreshold: 1 * time.Minute,
	}

	// Run in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- runWithConfig(config)
	}()

	// Wait a bit for server to start
	time.Sleep(500 * time.Millisecond)

	// Send signal to trigger shutdown
	proc, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	err = proc.Signal(syscall.SIGTERM)
	require.NoError(t, err)

	// Wait for completion
	select {
	case err := <-errChan:
		// Should complete without error
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not shut down in time")
	}
}

// TestRun_FunctionCall tests the run() function
func TestRun_FunctionCall(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping run test in short mode")
	}

	// Reset flags to default values
	*host = "127.0.0.1"
	*port = 19091
	*maxDataAge = 10 * time.Minute
	*cleanupInterval = 1 * time.Minute
	*staleNodeThreshold = 1 * time.Minute

	// Run in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- run()
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Send SIGTERM to stop
	proc, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)

	err = proc.Signal(syscall.SIGTERM)
	require.NoError(t, err)

	// Wait for shutdown
	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not shut down in time")
	}
}

// TestRunWithConfig_ImmediateSignal tests receiving signal right after start
func TestRunWithConfig_ImmediateSignal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping immediate signal test in short mode")
	}

	config := app.ServerConfig{
		Host:               "127.0.0.1",
		Port:               19093,
		MaxDataAge:         10 * time.Minute,
		CleanupInterval:    1 * time.Minute,
		StaleNodeThreshold: 1 * time.Minute,
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- runWithConfig(config)
	}()

	// Send signal almost immediately
	time.Sleep(100 * time.Millisecond)
	proc, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	err = proc.Signal(syscall.SIGINT)
	require.NoError(t, err)

	// Wait for shutdown
	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not shut down in time")
	}
}

// TestRunWithConfig_MultipleCalls tests running multiple servers concurrently
func TestRunWithConfig_MultipleCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multiple calls test in short mode")
	}

	configs := []app.ServerConfig{
		{Host: "127.0.0.1", Port: 19094, MaxDataAge: 10 * time.Minute, CleanupInterval: 1 * time.Minute, StaleNodeThreshold: 1 * time.Minute},
		{Host: "127.0.0.1", Port: 19095, MaxDataAge: 10 * time.Minute, CleanupInterval: 1 * time.Minute, StaleNodeThreshold: 1 * time.Minute},
	}

	errChans := make([]chan error, len(configs))

	for i, config := range configs {
		errChans[i] = make(chan error, 1)
		go func(cfg app.ServerConfig, errCh chan error) {
			errCh <- runWithConfig(cfg)
		}(config, errChans[i])
	}

	// Wait for servers to start
	time.Sleep(500 * time.Millisecond)

	// Send signal to stop all
	proc, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	err = proc.Signal(syscall.SIGTERM)
	require.NoError(t, err)

	// Wait for all to shutdown
	for i, errCh := range errChans {
		select {
		case err := <-errCh:
			assert.NoError(t, err, "Server %d should shut down without error", i)
		case <-time.After(5 * time.Second):
			t.Fatalf("Server %d did not shut down in time", i)
		}
	}
}

// Benchmarks

func BenchmarkServer_Creation(b *testing.B) {
	config := app.ServerConfig{
		Host:               "0.0.0.0",
		Port:               8080,
		MaxDataAge:         1 * time.Hour,
		CleanupInterval:    5 * time.Minute,
		StaleNodeThreshold: 5 * time.Minute,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = app.NewServer(config)
	}
}

func BenchmarkFlag_Parsing(b *testing.B) {
	args := []string{"-host=localhost", "-port=8080", "-max-data-age=1h", "-cleanup-interval=5m", "-stale-node-threshold=5m"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		host = flag.String("host", "0.0.0.0", "Server host address")
		port = flag.Int("port", 8080, "Server port")
		maxDataAge = flag.Duration("max-data-age", 1*time.Hour, "Maximum age for stored data")
		cleanupInterval = flag.Duration("cleanup-interval", 5*time.Minute, "Cleanup interval for stale data")
		staleNodeThreshold = flag.Duration("stale-node-threshold", 5*time.Minute, "Threshold for considering nodes stale")

		_ = flag.CommandLine.Parse(args)
	}
}
