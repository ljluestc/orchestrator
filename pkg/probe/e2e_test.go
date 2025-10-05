package probe

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_FullProbeLifecycle tests the complete probe lifecycle
func TestE2E_FullProbeLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Setup mock server
	server := newMockServer()
	defer server.Close()

	// Create probe configuration
	config := ProbeConfig{
		ServerURL:           server.server.URL,
		AgentID:             "e2e-test-probe",
		CollectionInterval:  200 * time.Millisecond,
		HeartbeatInterval:   300 * time.Millisecond,
		CollectHost:         true,
		CollectProcesses:    true,
		CollectNetwork:      true,
		MaxProcesses:        20,
		MaxConnections:      20,
		IncludeAllProcesses: true,
		IncludeLocalhost:    true,
		ResolveProcesses:    true,
		RetryAttempts:       3,
		RetryDelay:          100 * time.Millisecond,
	}

	// Try to add Docker if available
	_, err := NewDockerCollector(false)
	if err == nil {
		config.CollectDocker = true
		config.CollectDockerStats = true
	}

	// Create probe
	probe, err := NewProbe(config)
	require.NoError(t, err)
	assert.NotNil(t, probe)

	// Verify initial state
	assert.False(t, probe.IsRunning())

	// Start probe
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = probe.Start(ctx)
	require.NoError(t, err)
	assert.True(t, probe.IsRunning())

	// Wait for multiple collection cycles
	time.Sleep(700 * time.Millisecond)

	// Verify data collection
	reports := server.GetReports()
	t.Logf("Collected %d reports", len(reports))
	assert.GreaterOrEqual(t, len(reports), 2, "Should have collected at least 2 reports")

	// Verify registration
	registrations := server.GetRegistrations()
	assert.Len(t, registrations, 1, "Should have registered exactly once")
	assert.Equal(t, "e2e-test-probe", registrations[0]["agent_id"])

	// Validate report structure
	for i, report := range reports {
		t.Logf("Validating report %d", i)

		assert.Equal(t, "e2e-test-probe", report.AgentID)
		assert.NotEmpty(t, report.Hostname)
		assert.False(t, report.Timestamp.IsZero())

		// Validate host info
		if config.CollectHost {
			require.NotNil(t, report.HostInfo, "Report %d missing HostInfo", i)
			assert.NotEmpty(t, report.HostInfo.Hostname)
			assert.NotEmpty(t, report.HostInfo.KernelVersion)
			assert.Greater(t, report.HostInfo.Uptime, time.Duration(0))
			assert.Greater(t, report.HostInfo.CPUInfo.Cores, 0)
			assert.Greater(t, report.HostInfo.MemoryInfo.TotalMB, uint64(0))
		}

		// Validate process info
		if config.CollectProcesses {
			require.NotNil(t, report.ProcessesInfo, "Report %d missing ProcessesInfo", i)
			assert.Greater(t, report.ProcessesInfo.TotalProcesses, 0)

			for _, proc := range report.ProcessesInfo.Processes {
				assert.Greater(t, proc.PID, 0)
				assert.NotEmpty(t, proc.Name)
				assert.NotEmpty(t, proc.State)
			}
		}

		// Validate network info
		if config.CollectNetwork {
			require.NotNil(t, report.NetworkInfo, "Report %d missing NetworkInfo", i)
			assert.GreaterOrEqual(t, report.NetworkInfo.TotalConnections, 0)
			assert.Equal(t, report.NetworkInfo.TotalConnections,
				report.NetworkInfo.TCPConnections+report.NetworkInfo.UDPConnections)
		}

		// Validate Docker info if enabled and available
		if config.CollectDocker && report.DockerInfo != nil {
			require.NotNil(t, report.DockerInfo, "Report %d missing DockerInfo", i)
			assert.NotEmpty(t, report.DockerInfo.DockerVersion)
			assert.GreaterOrEqual(t, report.DockerInfo.TotalContainers, 0)
		}
	}

	// Stop probe
	err = probe.Stop()
	require.NoError(t, err)
	assert.False(t, probe.IsRunning())

	// Verify clean shutdown
	time.Sleep(100 * time.Millisecond)

	// Attempt to stop again should fail
	err = probe.Stop()
	assert.Error(t, err)
}

// TestE2E_ProbeResilience tests probe resilience to errors
func TestE2E_ProbeResilience(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := newMockServer()
	defer server.Close()

	config := ProbeConfig{
		ServerURL:          server.server.URL,
		AgentID:            "resilience-test",
		CollectionInterval: 100 * time.Millisecond,
		CollectHost:        true,
		RetryAttempts:      3,
		RetryDelay:         50 * time.Millisecond,
	}

	probe, err := NewProbe(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = probe.Start(ctx)
	require.NoError(t, err)

	// Let it collect successfully once
	time.Sleep(150 * time.Millisecond)
	initialReports := len(server.GetReports())
	assert.Greater(t, initialReports, 0)

	// Introduce server errors
	server.SetErrorResponse(true)
	time.Sleep(200 * time.Millisecond)

	// Fix server
	server.SetErrorResponse(false)
	time.Sleep(200 * time.Millisecond)

	// Should recover and send more reports
	finalReports := len(server.GetReports())
	assert.GreaterOrEqual(t, finalReports, initialReports)

	err = probe.Stop()
	require.NoError(t, err)
}

// TestE2E_MultipleProbes tests multiple probes running concurrently
func TestE2E_MultipleProbes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := newMockServer()
	defer server.Close()

	probeCount := 3
	probes := make([]*Probe, probeCount)

	ctx := context.Background()

	// Create and start multiple probes
	for i := 0; i < probeCount; i++ {
		config := ProbeConfig{
			ServerURL:          server.server.URL,
			AgentID:            string(rune('A' + i)),
			CollectionInterval: 150 * time.Millisecond,
			CollectHost:        true,
		}

		probe, err := NewProbe(config)
		require.NoError(t, err)

		err = probe.Start(ctx)
		require.NoError(t, err)

		probes[i] = probe
	}

	// Wait for collections
	time.Sleep(500 * time.Millisecond)

	// Verify all probes registered
	registrations := server.GetRegistrations()
	assert.GreaterOrEqual(t, len(registrations), probeCount)

	// Verify reports from multiple agents
	reports := server.GetReports()
	assert.GreaterOrEqual(t, len(reports), probeCount)

	agentIDs := make(map[string]bool)
	for _, report := range reports {
		agentIDs[report.AgentID] = true
	}
	assert.GreaterOrEqual(t, len(agentIDs), probeCount)

	// Stop all probes
	for _, probe := range probes {
		err := probe.Stop()
		assert.NoError(t, err)
	}
}

// TestE2E_LongRunning tests probe stability over longer duration
func TestE2E_LongRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running E2E test in short mode")
	}

	// Only run if explicitly requested
	if os.Getenv("RUN_LONG_TESTS") != "1" {
		t.Skip("Set RUN_LONG_TESTS=1 to run long-running tests")
	}

	server := newMockServer()
	defer server.Close()

	config := ProbeConfig{
		ServerURL:          server.server.URL,
		AgentID:            "long-running-test",
		CollectionInterval: 500 * time.Millisecond,
		HeartbeatInterval:  1 * time.Second,
		CollectHost:        true,
		CollectProcesses:   true,
		CollectNetwork:     true,
		MaxProcesses:       50,
		MaxConnections:     50,
	}

	probe, err := NewProbe(config)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = probe.Start(ctx)
	require.NoError(t, err)

	// Monitor for 10 seconds
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	startReports := 0
	for i := 0; i < 10; i++ {
		<-ticker.C
		currentReports := len(server.GetReports())
		t.Logf("After %d seconds: %d reports", i+1, currentReports)

		if i == 0 {
			startReports = currentReports
		}
	}

	finalReports := len(server.GetReports())

	// Should have collected multiple reports
	assert.Greater(t, finalReports-startReports, 10)

	err = probe.Stop()
	require.NoError(t, err)

	t.Logf("Long-running test completed: %d total reports", finalReports)
}

// TestE2E_ConfigValidation tests configuration validation
func TestE2E_ConfigValidation(t *testing.T) {
	testCases := []struct {
		name    string
		config  ProbeConfig
		wantErr bool
	}{
		{
			name: "valid_minimal_config",
			config: ProbeConfig{
				ServerURL:   "http://localhost:8080",
				CollectHost: true,
			},
			wantErr: false,
		},
		{
			name: "valid_full_config",
			config: ProbeConfig{
				ServerURL:           "http://localhost:8080",
				AgentID:             "test-agent",
				CollectionInterval:  1 * time.Second,
				HeartbeatInterval:   1 * time.Minute,
				CollectHost:         true,
				CollectDocker:       false, // Don't require Docker
				CollectProcesses:    true,
				CollectNetwork:      true,
				MaxProcesses:        100,
				MaxConnections:      100,
				IncludeAllProcesses: true,
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			probe, err := NewProbe(tc.config)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, probe)

				// Verify defaults were set
				cfg := probe.GetConfig()
				if tc.config.CollectionInterval == 0 {
					assert.Greater(t, cfg.CollectionInterval, time.Duration(0))
				}
				if tc.config.HeartbeatInterval == 0 {
					assert.Greater(t, cfg.HeartbeatInterval, time.Duration(0))
				}
			}
		})
	}
}
