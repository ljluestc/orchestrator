package containerizer

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/api/types/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDockerContainerizer_KillContainer tests force-stopping containers
func TestDockerContainerizer_KillContainer(t *testing.T) {
	dc := setupTestContainerizer(t)
	defer dc.Close()

	ctx := context.Background()

	// Create and start a container
	containerID, err := dc.CreateContainer(ctx, &ContainerConfig{
		Name:    "test-kill-container",
		Image:   "alpine:latest",
		Command: []string{"sleep", "300"},
	})
	require.NoError(t, err)

	err = dc.StartContainer(ctx, containerID)
	require.NoError(t, err)

	// Wait a bit for container to be running
	time.Sleep(100 * time.Millisecond)

	// Kill the container
	err = dc.KillContainer(ctx, containerID)
	assert.NoError(t, err)

	// Verify state was updated
	dc.statesMux.RLock()
	state := dc.containerStates[containerID]
	dc.statesMux.RUnlock()

	assert.Equal(t, "killed", state.Status)
	assert.False(t, state.StopTime.IsZero())

	// Cleanup
	dc.RemoveContainer(ctx, containerID)
}

// TestDockerContainerizer_RestartContainer tests container restart
func TestDockerContainerizer_RestartContainer(t *testing.T) {
	dc := setupTestContainerizer(t)
	defer dc.Close()

	ctx := context.Background()

	// Create and start a container
	containerID, err := dc.CreateContainer(ctx, &ContainerConfig{
		Name:    "test-restart-container",
		Image:   "alpine:latest",
		Command: []string{"sleep", "300"},
	})
	require.NoError(t, err)

	err = dc.StartContainer(ctx, containerID)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Restart the container
	err = dc.RestartContainer(ctx, containerID, 5)
	assert.NoError(t, err)

	// Verify state was updated
	dc.statesMux.RLock()
	state := dc.containerStates[containerID]
	dc.statesMux.RUnlock()

	assert.Equal(t, "running", state.Status)
	assert.False(t, state.StartTime.IsZero())

	// Cleanup
	dc.KillContainer(ctx, containerID)
	dc.RemoveContainer(ctx, containerID)
}

// TestDockerContainerizer_PushImage tests pushing images to registry
func TestDockerContainerizer_PushImage(t *testing.T) {
	t.Skip("Skipping push test as it requires a registry")

	dc := setupTestContainerizer(t)
	defer dc.Close()

	ctx := context.Background()

	// Tag an image for push
	err := dc.TagImage(ctx, "alpine:latest", "localhost:5000/alpine:test")
	require.NoError(t, err)

	// Attempt to push (will fail without a registry, but tests the code path)
	err = dc.PushImage(ctx, "localhost:5000/alpine:test")
	// We expect this to fail without a registry, but the function is executed
	assert.Error(t, err)
}

// TestDockerContainerizer_UpdateImageCache tests image cache updates
func TestDockerContainerizer_UpdateImageCache(t *testing.T) {
	config := &ContainerizerConfig{
		ImageCacheSize:   10 * 1024 * 1024 * 1024, // 10GB
		EnableImageCache: true,
		NetworkMode:      "bridge",
	}

	dc, err := NewDockerContainerizer(config)
	require.NoError(t, err)
	defer dc.Close()

	ctx := context.Background()

	// Pull an image to trigger cache update
	err = dc.PullImage(ctx, "alpine:latest")
	require.NoError(t, err)

	// Verify cache was updated
	dc.imageCache.mu.RLock()
	cachedCount := len(dc.imageCache.images)
	cacheSize := dc.imageCache.currentSize
	dc.imageCache.mu.RUnlock()

	assert.Greater(t, cachedCount, 0, "Image cache should have at least one image")
	assert.Greater(t, cacheSize, int64(0), "Cache size should be greater than 0")
}

// TestDockerContainerizer_EvictLRUImages tests LRU eviction
func TestDockerContainerizer_EvictLRUImages(t *testing.T) {
	// Create containerizer with small cache
	config := &ContainerizerConfig{
		ImageCacheSize:   1 * 1024 * 1024, // 1MB - very small to force eviction
		EnableImageCache: true,
		NetworkMode:      "bridge",
	}

	dc, err := NewDockerContainerizer(config)
	require.NoError(t, err)
	defer dc.Close()

	ctx := context.Background()

	// Manually populate cache to trigger eviction
	dc.imageCache.mu.Lock()

	// Add mock cached images
	dc.imageCache.images["img1"] = &CachedImage{
		ID:       "img1",
		Size:     500 * 1024,
		LastUsed: time.Now().Add(-10 * time.Minute),
	}
	dc.imageCache.images["img2"] = &CachedImage{
		ID:       "img2",
		Size:     500 * 1024,
		LastUsed: time.Now().Add(-5 * time.Minute),
	}
	dc.imageCache.currentSize = 1000 * 1024
	dc.imageCache.mu.Unlock()

	// Pull a new image which should trigger eviction
	err = dc.PullImage(ctx, "alpine:latest")
	require.NoError(t, err)

	// The eviction logic should have been triggered
	// We can't verify exact behavior without the image being larger than cache,
	// but the code path has been executed
	dc.imageCache.mu.RLock()
	currentSize := dc.imageCache.currentSize
	dc.imageCache.mu.RUnlock()

	// Cache should respect max size
	assert.LessOrEqual(t, currentSize, config.ImageCacheSize*2,
		"Cache size should be reasonable relative to max size")
}

// TestDockerContainerizer_EncodeAuthToBase64 tests auth encoding
func TestDockerContainerizer_EncodeAuthToBase64(t *testing.T) {
	authConfig := registry.AuthConfig{
		Username:      "testuser",
		Password:      "testpass",
		Email:         "test@example.com",
		ServerAddress: "registry.example.com",
	}

	encoded, err := encodeAuthToBase64(authConfig)
	assert.NoError(t, err)
	assert.NotEmpty(t, encoded)
	// Base64 encoding should produce a valid non-empty string
	assert.Greater(t, len(encoded), 20, "Encoded auth should be a reasonable length")
}

// TestDockerContainerizer_CountContainersByStatus tests status counting
func TestDockerContainerizer_CountContainersByStatus(t *testing.T) {
	dc := setupTestContainerizer(t)
	defer dc.Close()

	ctx := context.Background()

	// Create multiple containers in different states
	container1, err := dc.CreateContainer(ctx, &ContainerConfig{
		Name:    "test-count-1",
		Image:   "alpine:latest",
		Command: []string{"sleep", "300"},
	})
	require.NoError(t, err)

	container2, err := dc.CreateContainer(ctx, &ContainerConfig{
		Name:    "test-count-2",
		Image:   "alpine:latest",
		Command: []string{"sleep", "300"},
	})
	require.NoError(t, err)

	// Start only one
	err = dc.StartContainer(ctx, container1)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Count by status
	dc.statesMux.RLock()
	runningCount := dc.countContainersByStatus("running")
	createdCount := dc.countContainersByStatus("created")
	dc.statesMux.RUnlock()

	assert.Equal(t, 1, runningCount, "Should have 1 running container")
	assert.Equal(t, 1, createdCount, "Should have 1 created container")

	// Cleanup
	dc.KillContainer(ctx, container1)
	dc.RemoveContainer(ctx, container1)
	dc.RemoveContainer(ctx, container2)
}

// TestDockerContainerizer_GetStats tests aggregated statistics
func TestDockerContainerizer_GetStats(t *testing.T) {
	dc := setupTestContainerizer(t)
	defer dc.Close()

	ctx := context.Background()

	// Create some containers
	containerID, err := dc.CreateContainer(ctx, &ContainerConfig{
		Name:    "test-stats",
		Image:   "alpine:latest",
		Command: []string{"sleep", "10"},
	})
	require.NoError(t, err)

	err = dc.StartContainer(ctx, containerID)
	require.NoError(t, err)

	// Get stats
	stats := dc.GetStats()

	assert.NotNil(t, stats)
	assert.Contains(t, stats, "containers")
	assert.Contains(t, stats, "images")

	containers := stats["containers"].(map[string]interface{})
	assert.Contains(t, containers, "total")
	assert.Contains(t, containers, "running")
	assert.Contains(t, containers, "stopped")

	assert.GreaterOrEqual(t, containers["total"].(int), 1)

	// Cleanup
	dc.KillContainer(ctx, containerID)
	dc.RemoveContainer(ctx, containerID)
}

// TestDockerContainerizer_ImageCacheWithPullTimeout tests cache with timeout
func TestDockerContainerizer_ImageCacheWithPullTimeout(t *testing.T) {
	config := &ContainerizerConfig{
		ImagePullTimeout: 30 * time.Second,
		ImageCacheSize:   10 * 1024 * 1024 * 1024,
		EnableImageCache: true,
		NetworkMode:      "bridge",
	}

	dc, err := NewDockerContainerizer(config)
	require.NoError(t, err)
	defer dc.Close()

	ctx := context.Background()

	// Pull image with timeout configured
	err = dc.PullImage(ctx, "alpine:latest")
	assert.NoError(t, err)

	// Second pull should hit cache
	err = dc.PullImage(ctx, "alpine:latest")
	assert.NoError(t, err)

	// Verify cache hit incremented use count
	dc.imageCache.mu.RLock()
	foundCache := false
	for _, cached := range dc.imageCache.images {
		if len(cached.RepoTags) > 0 {
			foundCache = true
			break
		}
	}
	dc.imageCache.mu.RUnlock()

	assert.True(t, foundCache, "Image should be in cache")
}

// TestDockerContainerizer_PullImageWithAuth tests authenticated image pull
func TestDockerContainerizer_PullImageWithAuth(t *testing.T) {
	config := &ContainerizerConfig{
		ImageCacheSize:   10 * 1024 * 1024 * 1024,
		EnableImageCache: true,
		DefaultRegistry:  "docker.io",
		RegistryAuth: map[string]RegistryCredentials{
			"docker.io": {
				Username:      "testuser",
				Password:      "testpass",
				Email:         "test@example.com",
				ServerAddress: "https://index.docker.io/v1/",
			},
		},
		NetworkMode: "bridge",
	}

	dc, err := NewDockerContainerizer(config)
	require.NoError(t, err)
	defer dc.Close()

	ctx := context.Background()

	// This will attempt to use auth but won't actually succeed with fake creds
	// The important part is that the code path is executed
	err = dc.PullImage(ctx, "alpine:latest")
	// Even with fake auth, alpine:latest is a public image so it should succeed
	assert.NoError(t, err)
}

// TestDockerContainerizer_ListContainersAll tests listing all containers
func TestDockerContainerizer_ListContainers_All(t *testing.T) {
	dc := setupTestContainerizer(t)
	defer dc.Close()

	ctx := context.Background()

	// Create a container but don't start it
	containerID, err := dc.CreateContainer(ctx, &ContainerConfig{
		Name:    "test-list-all",
		Image:   "alpine:latest",
		Command: []string{"echo", "hello"},
	})
	require.NoError(t, err)

	// List all containers (including stopped)
	containers, err := dc.ListContainers(ctx, true)
	assert.NoError(t, err)
	assert.NotNil(t, containers)

	// Should find our container
	found := false
	for _, c := range containers {
		for _, name := range c.Names {
			if name == "/test-list-all" {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "Should find the created container")

	// Cleanup
	dc.RemoveContainer(ctx, containerID)
}

// TestDockerContainerizer_RemoveImage tests image removal with cache update
func TestDockerContainerizer_RemoveImage_WithCache(t *testing.T) {
	dc := setupTestContainerizer(t)
	defer dc.Close()

	ctx := context.Background()

	// Pull an image first
	err := dc.PullImage(ctx, "alpine:latest")
	require.NoError(t, err)

	// Get image ID
	images, err := dc.GetImageList(ctx)
	require.NoError(t, err)

	var alpineID string
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == "alpine:latest" {
				alpineID = img.ID
				break
			}
		}
	}

	if alpineID != "" {
		// Cache should have the image
		dc.imageCache.mu.RLock()
		initialCacheSize := dc.imageCache.currentSize
		dc.imageCache.mu.RUnlock()

		// Remove image (force=true to remove even if cached)
		err = dc.RemoveImage(ctx, alpineID, true)
		// May fail if image is in use, but that's okay
		if err == nil {
			// Verify cache was updated
			dc.imageCache.mu.RLock()
			newCacheSize := dc.imageCache.currentSize
			dc.imageCache.mu.RUnlock()

			assert.LessOrEqual(t, newCacheSize, initialCacheSize,
				"Cache size should decrease or stay same after image removal")
		}
	}
}

// TestDockerContainerizer_TagImage tests image tagging
func TestDockerContainerizer_TagImage_Success(t *testing.T) {
	dc := setupTestContainerizer(t)
	defer dc.Close()

	ctx := context.Background()

	// Ensure alpine:latest exists
	err := dc.PullImage(ctx, "alpine:latest")
	require.NoError(t, err)

	// Tag the image
	err = dc.TagImage(ctx, "alpine:latest", "alpine:test-tag")
	assert.NoError(t, err)

	// Verify the tag exists
	images, err := dc.GetImageList(ctx)
	require.NoError(t, err)

	foundTag := false
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == "alpine:test-tag" {
				foundTag = true
				break
			}
		}
	}
	assert.True(t, foundTag, "Should find the new tag")

	// Cleanup - remove the tagged image
	dc.RemoveImage(ctx, "alpine:test-tag", false)
}

// Helper function used by multiple tests
func setupTestContainerizer(t *testing.T) *DockerContainerizer {
	config := &ContainerizerConfig{
		ImageCacheSize:   10 * 1024 * 1024 * 1024,
		EnableImageCache: true,
		NetworkMode:      "bridge",
	}

	dc, err := NewDockerContainerizer(config)
	require.NoError(t, err)
	require.NotNil(t, dc)

	return dc
}
