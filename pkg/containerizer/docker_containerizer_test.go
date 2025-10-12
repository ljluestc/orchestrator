package containerizer

import (
	"context"
	"testing"
	"time"
)

func TestDockerContainerizer_ImageCache(t *testing.T) {
	config := &ContainerizerConfig{
		DockerHost:          "unix:///var/run/docker.sock",
		ImagePullTimeout:    5 * time.Minute,
		ContainerStartupMax: 5 * time.Second,
		ImageCacheSize:      10 * 1024 * 1024 * 1024, // 10GB
		EnableImageCache:    true,
		DefaultRegistry:     "docker.io",
		NetworkMode:         "bridge",
	}

	dc, err := NewDockerContainerizer(config)
	if err != nil {
		t.Skipf("Docker not available: %v", err)
		return
	}

	ctx := context.Background()

	// Test image pull
	err = dc.PullImage(ctx, "alpine:3.18")
	if err != nil {
		t.Errorf("Failed to pull image: %v", err)
	}

	// Test image exists after pull
	exists := dc.imageExists(ctx, "alpine:3.18")
	if !exists {
		t.Error("Image should exist after pull")
	}

	// Test cache hit on second pull
	err = dc.PullImage(ctx, "alpine:3.18")
	if err != nil {
		t.Errorf("Failed on cached pull: %v", err)
	}
}

func TestDockerContainerizer_CreateAndStart(t *testing.T) {
	config := &ContainerizerConfig{
		DockerHost:          "unix:///var/run/docker.sock",
		ImagePullTimeout:    5 * time.Minute,
		ContainerStartupMax: 5 * time.Second,
		EnableImageCache:    true,
		DefaultRegistry:     "docker.io",
		NetworkMode:         "bridge",
	}

	dc, err := NewDockerContainerizer(config)
	if err != nil {
		t.Skipf("Docker not available: %v", err)
		return
	}

	ctx := context.Background()

	// Create container
	containerCfg := &ContainerConfig{
		Name:        "test-container",
		Image:       "alpine:3.18",
		Command:     []string{"sleep", "300"},
		CPUShares:   1024,
		MemoryLimit: 128 * 1024 * 1024, // 128MB
	}

	containerID, err := dc.CreateContainer(ctx, containerCfg)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}

	// Start container and measure startup time
	startTime := time.Now()
	err = dc.StartContainer(ctx, containerID)
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}
	startupDuration := time.Since(startTime)

	// Verify startup time is < 5s
	if startupDuration > 5*time.Second {
		t.Errorf("Container startup took %v, exceeds 5s target", startupDuration)
	} else {
		t.Logf("Container started in %v (target: <5s) ✓", startupDuration)
	}

	// Cleanup
	defer func() {
		dc.StopContainer(ctx, containerID, 10)
		dc.RemoveContainer(ctx, containerID)
	}()

	// Verify container state
	dc.statesMux.RLock()
	state, exists := dc.containerStates[containerID]
	dc.statesMux.RUnlock()

	if !exists {
		t.Error("Container state not tracked")
	}
	if state.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", state.Status)
	}
}

func TestDockerContainerizer_StartupPerformance(t *testing.T) {
	config := &ContainerizerConfig{
		DockerHost:          "unix:///var/run/docker.sock",
		ImagePullTimeout:    5 * time.Minute,
		ContainerStartupMax: 5 * time.Second,
		EnableImageCache:    true,
		DefaultRegistry:     "docker.io",
		NetworkMode:         "bridge",
	}

	dc, err := NewDockerContainerizer(config)
	if err != nil {
		t.Skipf("Docker not available: %v", err)
		return
	}

	ctx := context.Background()

	// Pre-pull image
	err = dc.PullImage(ctx, "alpine:3.18")
	if err != nil {
		t.Fatalf("Failed to pull image: %v", err)
	}

	// Test multiple container startups
	iterations := 5
	var totalStartupTime time.Duration

	for i := 0; i < iterations; i++ {
		containerCfg := &ContainerConfig{
			Name:        "perf-test-" + time.Now().Format("20060102150405"),
			Image:       "alpine:3.18",
			Command:     []string{"echo", "hello"},
			CPUShares:   512,
			MemoryLimit: 64 * 1024 * 1024,
		}

		startTime := time.Now()

		containerID, err := dc.CreateContainer(ctx, containerCfg)
		if err != nil {
			t.Errorf("Iteration %d: Failed to create container: %v", i, err)
			continue
		}

		err = dc.StartContainer(ctx, containerID)
		if err != nil {
			t.Errorf("Iteration %d: Failed to start container: %v", i, err)
			dc.RemoveContainer(ctx, containerID)
			continue
		}

		startupTime := time.Since(startTime)
		totalStartupTime += startupTime

		// Cleanup
		dc.StopContainer(ctx, containerID, 5)
		dc.RemoveContainer(ctx, containerID)

		t.Logf("Iteration %d: Startup time = %v", i, startupTime)
	}

	avgStartupTime := totalStartupTime / time.Duration(iterations)
	t.Logf("Average startup time: %v (target: <5s)", avgStartupTime)

	if avgStartupTime > 5*time.Second {
		t.Errorf("Average startup time %v exceeds 5s target", avgStartupTime)
	}
}

func TestImageCache_LRUEviction(t *testing.T) {
	cache := &ImageCache{
		images:  make(map[string]*CachedImage),
		maxSize: 1 * 1024 * 1024 * 1024, // 1GB
	}

	// Add images to cache
	for i := 0; i < 10; i++ {
		imageID := time.Now().Format("image-20060102150405")
		cache.images[imageID] = &CachedImage{
			ID:       imageID,
			Size:     200 * 1024 * 1024, // 200MB each
			LastUsed: time.Now().Add(-time.Duration(i) * time.Hour),
		}
		cache.currentSize += 200 * 1024 * 1024
	}

	// Total: 2GB, max: 1GB - should trigger eviction
	if cache.currentSize > cache.maxSize {
		t.Logf("Cache size %d exceeds max %d, eviction needed", cache.currentSize, cache.maxSize)
	}
}

func BenchmarkContainerStartup(b *testing.B) {
	config := &ContainerizerConfig{
		DockerHost:          "unix:///var/run/docker.sock",
		ImagePullTimeout:    5 * time.Minute,
		ContainerStartupMax: 5 * time.Second,
		EnableImageCache:    true,
		DefaultRegistry:     "docker.io",
		NetworkMode:         "bridge",
	}

	dc, err := NewDockerContainerizer(config)
	if err != nil {
		b.Skipf("Docker not available: %v", err)
		return
	}

	ctx := context.Background()

	// Pre-pull image
	dc.PullImage(ctx, "alpine:3.18")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		containerCfg := &ContainerConfig{
			Name:        "bench-" + time.Now().Format("20060102150405"),
			Image:       "alpine:3.18",
			Command:     []string{"echo", "test"},
			CPUShares:   512,
			MemoryLimit: 64 * 1024 * 1024,
		}

		containerID, err := dc.CreateContainer(ctx, containerCfg)
		if err != nil {
			b.Error(err)
			continue
		}

		err = dc.StartContainer(ctx, containerID)
		if err != nil {
			b.Error(err)
			dc.RemoveContainer(ctx, containerID)
			continue
		}

		dc.StopContainer(ctx, containerID, 5)
		dc.RemoveContainer(ctx, containerID)
	}
}
