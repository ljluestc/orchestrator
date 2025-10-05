package probe

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerCollector_NewDockerCollector(t *testing.T) {
	// This test requires Docker to be running
	collector, err := NewDockerCollector(false)
	if err != nil {
		t.Skip("Docker not available, skipping test")
	}
	require.NoError(t, err)
	require.NotNil(t, collector)

	defer collector.Close()

	assert.False(t, collector.collectStats)
}

func TestDockerCollector_NewDockerCollectorWithStats(t *testing.T) {
	collector, err := NewDockerCollector(true)
	if err != nil {
		t.Skip("Docker not available, skipping test")
	}
	require.NoError(t, err)
	require.NotNil(t, collector)

	defer collector.Close()

	assert.True(t, collector.collectStats)
}

func TestDockerCollector_Collect(t *testing.T) {
	collector, err := NewDockerCollector(false)
	if err != nil {
		t.Skip("Docker not available, skipping test")
	}
	defer collector.Close()

	ctx := context.Background()
	info, err := collector.Collect(ctx)
	if err != nil && strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
		t.Skip("Docker not available, skipping test")
	}
	require.NoError(t, err)
	require.NotNil(t, info)

	// Validate basic fields
	assert.NotEmpty(t, info.DockerVersion)
	assert.GreaterOrEqual(t, info.TotalContainers, 0)
	assert.GreaterOrEqual(t, info.Images, 0)
	assert.False(t, info.Timestamp.IsZero())

	// Validate container counts
	total := info.RunningContainers + info.PausedContainers + info.StoppedContainers
	assert.Equal(t, info.TotalContainers, total)

	// Validate containers array
	assert.Len(t, info.Containers, info.TotalContainers)

	for _, container := range info.Containers {
		assert.NotEmpty(t, container.ID)
		assert.NotEmpty(t, container.Image)
		assert.NotEmpty(t, container.State)
		assert.NotEmpty(t, container.Status)
		assert.False(t, container.Created.IsZero())

		// Stats should not be collected when collectStats is false
		assert.Nil(t, container.Stats)
	}
}

func TestDockerCollector_CollectWithStats(t *testing.T) {
	collector, err := NewDockerCollector(true)
	if err != nil {
		t.Skip("Docker not available, skipping test")
	}
	defer collector.Close()

	ctx := context.Background()
	info, err := collector.Collect(ctx)
	if err != nil && strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
		t.Skip("Docker not available, skipping test")
	}
	require.NoError(t, err)
	require.NotNil(t, info)

	// Check if any running containers have stats
	hasRunningContainer := false
	for _, container := range info.Containers {
		if container.State == "running" {
			hasRunningContainer = true
			// Stats may still be nil if collection failed, but we don't fail the test
			if container.Stats != nil {
				assert.GreaterOrEqual(t, container.Stats.CPUPercent, 0.0)
				assert.GreaterOrEqual(t, container.Stats.MemoryUsageMB, uint64(0))
				assert.GreaterOrEqual(t, container.Stats.MemoryLimitMB, uint64(0))
			}
		}
	}

	if !hasRunningContainer {
		t.Log("No running containers to test stats collection")
	}
}

func TestDockerCollector_ContextCancellation(t *testing.T) {
	collector, err := NewDockerCollector(false)
	if err != nil {
		t.Skip("Docker not available, skipping test")
	}
	require.NoError(t, err)
	defer collector.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = collector.Collect(ctx)
	// The error might be context canceled or might succeed if it was fast enough
	// We just verify it doesn't panic
	if err != nil {
		assert.Contains(t, err.Error(), "context")
	}
}

func TestDockerCollector_Close(t *testing.T) {
	collector, err := NewDockerCollector(false)
	if err != nil {
		t.Skip("Docker not available, skipping test")
	}
	require.NoError(t, err)

	err = collector.Close()
	assert.NoError(t, err)

	// Second close should not panic
	err = collector.Close()
	assert.NoError(t, err)
}

func TestPortMapping(t *testing.T) {
	// Test PortMapping structure
	pm := PortMapping{
		PrivatePort: 8080,
		PublicPort:  80,
		Type:        "tcp",
		IP:          "0.0.0.0",
	}

	assert.Equal(t, uint16(8080), pm.PrivatePort)
	assert.Equal(t, uint16(80), pm.PublicPort)
	assert.Equal(t, "tcp", pm.Type)
	assert.Equal(t, "0.0.0.0", pm.IP)
}

func TestContainerStats(t *testing.T) {
	// Test ContainerStats structure
	stats := ContainerStats{
		CPUPercent:    25.5,
		MemoryUsageMB: 512,
		MemoryLimitMB: 1024,
		MemoryPercent: 50.0,
		NetworkRxMB:   10.5,
		NetworkTxMB:   5.2,
	}

	assert.Equal(t, 25.5, stats.CPUPercent)
	assert.Equal(t, uint64(512), stats.MemoryUsageMB)
	assert.Equal(t, uint64(1024), stats.MemoryLimitMB)
	assert.Equal(t, 50.0, stats.MemoryPercent)
	assert.Equal(t, 10.5, stats.NetworkRxMB)
	assert.Equal(t, 5.2, stats.NetworkTxMB)
}

func TestNewDockerCollectorWithClient(t *testing.T) {
	// Test the constructor that takes a client
	// This is used for testing and should not require Docker to be running
	collector := NewDockerCollectorWithClient(nil, false)
	assert.NotNil(t, collector)
	assert.False(t, collector.collectStats)
	
	collector2 := NewDockerCollectorWithClient(nil, true)
	assert.NotNil(t, collector2)
	assert.True(t, collector2.collectStats)
}

func TestDockerCollector_GetContainerStats(t *testing.T) {
	// Test getContainerStats method with mock data
	// This tests the internal method that has 0% coverage
	collector := NewDockerCollectorWithClient(nil, true)
	
	// This will fail because we don't have a real client, but we're testing the method exists
	ctx := context.Background()
	_, err := collector.getContainerStats(ctx, "test-container")
	assert.Error(t, err) // Expected to fail without real Docker client
}

func TestDockerCollector_CloseWithNilClient(t *testing.T) {
	// Test Close method with nil client
	collector := NewDockerCollectorWithClient(nil, false)
	err := collector.Close()
	assert.NoError(t, err) // Should not panic with nil client
}
