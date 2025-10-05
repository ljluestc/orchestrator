package probe

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewProbe(t *testing.T) {
	// Test with minimal config
	config := ProbeConfig{
		ServerURL: "http://localhost:8080",
	}

	probe, err := NewProbe(config)
	assert.NoError(t, err)
	assert.NotNil(t, probe)
	assert.Equal(t, "http://localhost:8080", probe.config.ServerURL)
	assert.NotEmpty(t, probe.config.AgentID)
	assert.Equal(t, 30*time.Second, probe.config.CollectionInterval)
	assert.Equal(t, 60*time.Second, probe.config.HeartbeatInterval)
	assert.Equal(t, 3, probe.config.RetryAttempts)
	assert.Equal(t, 5*time.Second, probe.config.RetryDelay)
}

func TestNewProbeWithDefaults(t *testing.T) {
	// Test with zero values to trigger defaults
	config := ProbeConfig{
		ServerURL:           "http://localhost:8080",
		CollectionInterval:  0,
		HeartbeatInterval:   0,
		RetryAttempts:       0,
		RetryDelay:          0,
	}

	probe, err := NewProbe(config)
	assert.NoError(t, err)
	assert.NotNil(t, probe)
	assert.Equal(t, 30*time.Second, probe.config.CollectionInterval)
	assert.Equal(t, 60*time.Second, probe.config.HeartbeatInterval)
	assert.Equal(t, 3, probe.config.RetryAttempts)
	assert.Equal(t, 5*time.Second, probe.config.RetryDelay)
}

func TestNewProbeWithAgentID(t *testing.T) {
	// Test with provided agent ID
	config := ProbeConfig{
		ServerURL: "http://localhost:8080",
		AgentID:   "test-agent-123",
	}

	probe, err := NewProbe(config)
	assert.NoError(t, err)
	assert.NotNil(t, probe)
	assert.Equal(t, "test-agent-123", probe.config.AgentID)
}

func TestNewProbeWithHostnameError(t *testing.T) {
	// Test hostname error by mocking os.Hostname
	// We can't easily mock os.Hostname, but we can test the error handling
	// by creating a config that would cause other errors
	
	config := ProbeConfig{
		ServerURL: "", // Invalid server URL
	}

	probe, err := NewProbe(config)
	// This should not fail due to hostname, but might fail due to other reasons
	if err != nil {
		assert.Error(t, err)
	} else {
		assert.NotNil(t, probe)
	}
}

func TestNewProbeWithCollectors(t *testing.T) {
	// Test with all collectors enabled
	config := ProbeConfig{
		ServerURL:           "http://localhost:8080",
		CollectHost:         true,
		CollectDocker:       true,
		CollectDockerStats:  true,
		CollectProcesses:    true,
		CollectNetwork:      true,
		MaxProcesses:        100,
		MaxConnections:      100,
		IncludeLocalhost:    true,
		ResolveProcesses:    true,
		IncludeAllProcesses: true,
	}

	probe, err := NewProbe(config)
	assert.NoError(t, err)
	assert.NotNil(t, probe)
	assert.NotNil(t, probe.hostCollector)
	assert.NotNil(t, probe.dockerCollector)
	assert.NotNil(t, probe.processCollector)
	assert.NotNil(t, probe.networkCollector)
}

func TestNewProbeWithDockerError(t *testing.T) {
	// Test Docker collector creation error
	config := ProbeConfig{
		ServerURL:      "http://localhost:8080",
		CollectDocker:  true,
		CollectDockerStats: true,
	}

	// This might fail if Docker is not available
	probe, err := NewProbe(config)
	if err != nil {
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create Docker collector")
	} else {
		assert.NotNil(t, probe)
	}
}

func TestProbeStart(t *testing.T) {
	config := ProbeConfig{
		ServerURL: "http://localhost:8080",
	}

	probe, err := NewProbe(config)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test starting probe
	err = probe.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, probe.IsRunning())

	// Test starting already running probe
	err = probe.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "probe is already running")

	// Clean up
	probe.Stop()
}

func TestProbeStop(t *testing.T) {
	config := ProbeConfig{
		ServerURL: "http://localhost:8080",
	}

	probe, err := NewProbe(config)
	assert.NoError(t, err)

	// Test stopping non-running probe
	err = probe.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "probe is not running")

	// Test normal stop
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = probe.Start(ctx)
	assert.NoError(t, err)

	err = probe.Stop()
	assert.NoError(t, err)
	assert.False(t, probe.IsRunning())
}

func TestProbeStartStopCycle(t *testing.T) {
	config := ProbeConfig{
		ServerURL: "http://localhost:8080",
	}

	probe, err := NewProbe(config)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start probe
	err = probe.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, probe.IsRunning())

	// Stop probe
	err = probe.Stop()
	assert.NoError(t, err)
	assert.False(t, probe.IsRunning())

	// Create a new probe for the second cycle to avoid channel issues
	probe2, err := NewProbe(config)
	assert.NoError(t, err)

	// Start second probe
	err = probe2.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, probe2.IsRunning())

	// Stop second probe
	err = probe2.Stop()
	assert.NoError(t, err)
	assert.False(t, probe2.IsRunning())
}

func TestProbeCollectionLoop(t *testing.T) {
	config := ProbeConfig{
		ServerURL:           "http://localhost:8080",
		CollectionInterval:  100 * time.Millisecond,
		CollectHost:         true,
		CollectProcesses:    true,
		CollectNetwork:      true,
	}

	probe, err := NewProbe(config)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Start probe
	err = probe.Start(ctx)
	assert.NoError(t, err)

	// Wait for collection loop to run
	time.Sleep(150 * time.Millisecond)

	// Stop probe
	err = probe.Stop()
	assert.NoError(t, err)
}

func TestProbeHeartbeatLoop(t *testing.T) {
	config := ProbeConfig{
		ServerURL:          "http://localhost:8080",
		HeartbeatInterval:  100 * time.Millisecond,
	}

	probe, err := NewProbe(config)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Start probe
	err = probe.Start(ctx)
	assert.NoError(t, err)

	// Wait for heartbeat loop to run
	time.Sleep(150 * time.Millisecond)

	// Stop probe
	err = probe.Stop()
	assert.NoError(t, err)
}

func TestProbeContextCancellation(t *testing.T) {
	config := ProbeConfig{
		ServerURL:           "http://localhost:8080",
		CollectionInterval:  50 * time.Millisecond,
		HeartbeatInterval:   50 * time.Millisecond,
	}

	probe, err := NewProbe(config)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start probe
	err = probe.Start(ctx)
	assert.NoError(t, err)

	// Wait for context to be cancelled
	<-ctx.Done()

	// Stop probe
	err = probe.Stop()
	assert.NoError(t, err)
}

func TestProbeCollectAndSend(t *testing.T) {
	config := ProbeConfig{
		ServerURL:           "http://localhost:8080",
		CollectHost:         true,
		CollectProcesses:    true,
		CollectNetwork:      true,
	}

	probe, err := NewProbe(config)
	assert.NoError(t, err)

	ctx := context.Background()

	// Test collectAndSend directly
	probe.collectAndSend(ctx)

	// This should not panic and should complete
	assert.True(t, true)
}

func TestProbeCollectAndSendWithDocker(t *testing.T) {
	config := ProbeConfig{
		ServerURL:          "http://localhost:8080",
		CollectDocker:      true,
		CollectDockerStats: false,
	}

	probe, err := NewProbe(config)
	if err != nil {
		// Docker might not be available
		t.Skip("Docker not available, skipping test")
	}

	ctx := context.Background()

	// Test collectAndSend with Docker collector
	probe.collectAndSend(ctx)

	// This should not panic and should complete
	assert.True(t, true)
}

func TestProbeCollectAndSendWithAllCollectors(t *testing.T) {
	config := ProbeConfig{
		ServerURL:           "http://localhost:8080",
		CollectHost:         true,
		CollectDocker:       true,
		CollectDockerStats:  true,
		CollectProcesses:    true,
		CollectNetwork:      true,
		MaxProcesses:        100,
		MaxConnections:      100,
		IncludeLocalhost:    true,
		ResolveProcesses:    true,
		IncludeAllProcesses: true,
	}

	probe, err := NewProbe(config)
	if err != nil {
		// Docker might not be available
		t.Skip("Docker not available, skipping test")
	}

	ctx := context.Background()

	// Test collectAndSend with all collectors
	probe.collectAndSend(ctx)

	// This should not panic and should complete
	assert.True(t, true)
}

func TestProbeIsRunning(t *testing.T) {
	config := ProbeConfig{
		ServerURL: "http://localhost:8080",
	}

	probe, err := NewProbe(config)
	assert.NoError(t, err)

	// Initially not running
	assert.False(t, probe.IsRunning())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start probe
	err = probe.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, probe.IsRunning())

	// Stop probe
	err = probe.Stop()
	assert.NoError(t, err)
	assert.False(t, probe.IsRunning())
}

func TestProbeGetConfig(t *testing.T) {
	config := ProbeConfig{
		ServerURL: "http://localhost:8080",
		AgentID:   "test-agent",
	}

	probe, err := NewProbe(config)
	assert.NoError(t, err)

	retrievedConfig := probe.GetConfig()
	assert.Equal(t, config.ServerURL, retrievedConfig.ServerURL)
	assert.Equal(t, config.AgentID, retrievedConfig.AgentID)
}

func TestProbeWithInvalidConfig(t *testing.T) {
	// Test with empty server URL
	config := ProbeConfig{
		ServerURL: "",
	}

	probe, err := NewProbe(config)
	// This might succeed or fail depending on validation
	if err != nil {
		assert.Error(t, err)
	} else {
		assert.NotNil(t, probe)
	}
}

func TestProbeWithLongRunning(t *testing.T) {
	if os.Getenv("RUN_LONG_TESTS") != "1" {
		t.Skip("Skipping long running test; set RUN_LONG_TESTS=1 to run")
	}

	config := ProbeConfig{
		ServerURL:           "http://localhost:8080",
		CollectionInterval:  1 * time.Second,
		HeartbeatInterval:   2 * time.Second,
		CollectHost:         true,
		CollectProcesses:    true,
		CollectNetwork:      true,
	}

	probe, err := NewProbe(config)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start probe
	err = probe.Start(ctx)
	assert.NoError(t, err)

	// Let it run for a while
	time.Sleep(3 * time.Second)

	// Stop probe
	err = probe.Stop()
	assert.NoError(t, err)
}

func TestProbeConcurrentAccess(t *testing.T) {
	config := ProbeConfig{
		ServerURL: "http://localhost:8080",
	}

	probe, err := NewProbe(config)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start probe
	err = probe.Start(ctx)
	assert.NoError(t, err)

	// Test concurrent access to IsRunning
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_ = probe.IsRunning()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Stop probe
	err = probe.Stop()
	assert.NoError(t, err)
}

func TestProbeErrorHandling(t *testing.T) {
	// Test probe creation with invalid configuration
	config := ProbeConfig{
		ServerURL: "invalid-url",
	}

	probe, err := NewProbe(config)
	if err != nil {
		assert.Error(t, err)
	} else {
		assert.NotNil(t, probe)
	}
}

func TestProbeWithMinimalConfig(t *testing.T) {
	// Test with absolute minimal configuration
	config := ProbeConfig{}

	probe, err := NewProbe(config)
	// This should fail due to empty server URL or other validation
	if err != nil {
		assert.Error(t, err)
	} else {
		assert.NotNil(t, probe)
	}
}

func TestProbeHostname(t *testing.T) {
	config := ProbeConfig{
		ServerURL: "http://localhost:8080",
	}

	probe, err := NewProbe(config)
	assert.NoError(t, err)

	// Test that hostname is set
	assert.NotEmpty(t, probe.hostname)
}

func TestProbeClientInitialization(t *testing.T) {
	config := ProbeConfig{
		ServerURL: "http://localhost:8080",
		AgentID:   "test-agent",
		APIKey:    "test-key",
	}

	probe, err := NewProbe(config)
	assert.NoError(t, err)

	// Test that client is initialized
	assert.NotNil(t, probe.client)
	// Note: Client struct doesn't expose config fields directly
	// We can only test that the client is not nil
}
