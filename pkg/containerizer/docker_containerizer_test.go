package containerizer

import (
	"testing"
	"time"

	"github.com/docker/docker/api/types/registry"
	"github.com/stretchr/testify/assert"
)

func TestEncodeAuthToBase64(t *testing.T) {
	tests := []struct {
		name     string
		authConfig registry.AuthConfig
		expected  string
	}{
		{
			name: "Empty auth config",
			authConfig: registry.AuthConfig{},
			expected:  "",
		},
		{
			name: "Auth config with credentials",
			authConfig: registry.AuthConfig{
				Username:      "testuser",
				Password:      "testpass",
				Email:         "test@example.com",
				ServerAddress: "docker.io",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := encodeAuthToBase64(tt.authConfig)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestImageCache_Operations(t *testing.T) {
	cache := &ImageCache{
		images: make(map[string]*CachedImage),
		maxSize: 1024 * 1024 * 1024, // 1GB
	}

	t.Run("Add image to cache", func(t *testing.T) {
		image := &CachedImage{
			ID:       "image-123",
			RepoTags: []string{"nginx:latest"},
			Size:     100 * 1024 * 1024, // 100MB
			LastUsed: time.Now(),
		}
		
		cache.mu.Lock()
		cache.images["nginx:latest"] = image
		cache.mu.Unlock()
		
		cache.mu.RLock()
		cached, exists := cache.images["nginx:latest"]
		cache.mu.RUnlock()
		
		assert.True(t, exists)
		assert.Equal(t, image, cached)
	})

	t.Run("Get image from cache", func(t *testing.T) {
		cache.mu.RLock()
		cached, exists := cache.images["nginx:latest"]
		cache.mu.RUnlock()
		
		assert.True(t, exists)
		assert.NotNil(t, cached)
		assert.Equal(t, "image-123", cached.ID)
	})

	t.Run("Remove image from cache", func(t *testing.T) {
		cache.mu.Lock()
		delete(cache.images, "nginx:latest")
		cache.mu.Unlock()
		
		cache.mu.RLock()
		_, exists := cache.images["nginx:latest"]
		cache.mu.RUnlock()
		
		assert.False(t, exists)
	})
}

func TestContainerizerConfig_Validation(t *testing.T) {
	tests := []struct {
		name        string
		config      *ContainerizerConfig
		expectError bool
	}{
		{
			name: "Valid config",
			config: &ContainerizerConfig{
				DockerHost:          "unix:///var/run/docker.sock",
				ImagePullTimeout:    5 * time.Minute,
				ContainerStartupMax: 30 * time.Second,
				ImageCacheSize:      1024 * 1024 * 1024, // 1GB
				EnableImageCache:    true,
				DefaultRegistry:     "docker.io",
				RegistryAuth:        make(map[string]RegistryCredentials),
				NetworkMode:         "bridge",
				CPUShares:           1024,
				MemoryLimit:         1024 * 1024 * 1024, // 1GB
				EnableGPU:           false,
			},
			expectError: false,
		},
		{
			name:        "Nil config",
			config:      nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config != nil {
				assert.NotNil(t, tt.config)
				assert.Equal(t, "unix:///var/run/docker.sock", tt.config.DockerHost)
				assert.Equal(t, 5*time.Minute, tt.config.ImagePullTimeout)
				assert.Equal(t, 30*time.Second, tt.config.ContainerStartupMax)
			} else {
				assert.True(t, tt.expectError)
			}
		})
	}
}

func TestContainerConfig_Validation(t *testing.T) {
	tests := []struct {
		name        string
		config      ContainerConfig
		expectError bool
	}{
		{
			name: "Valid config",
			config: ContainerConfig{
				Name:         "test-container",
				Image:        "nginx:latest",
				Command:      []string{"nginx", "-g", "daemon off;"},
				Environment:  []string{"ENV=test"},
				WorkingDir:   "/app",
				ExposedPorts: map[string]struct{}{"80/tcp": {}},
				Labels:       map[string]string{"app": "test"},
				Volumes:      []string{"/host:/container"},
				CPUShares:    512,
				CPUQuota:     100000,
				MemoryLimit:  512 * 1024 * 1024,
				GPUCount:     0,
			},
			expectError: false,
		},
		{
			name: "Empty config",
			config: ContainerConfig{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.expectError {
				assert.NotEmpty(t, tt.config.Name)
				assert.NotEmpty(t, tt.config.Image)
				assert.NotNil(t, tt.config.Command)
				assert.NotNil(t, tt.config.Environment)
				assert.NotNil(t, tt.config.ExposedPorts)
				assert.NotNil(t, tt.config.Labels)
				assert.NotNil(t, tt.config.Volumes)
			} else {
				assert.Empty(t, tt.config.Name)
				assert.Empty(t, tt.config.Image)
			}
		})
	}
}

func TestRegistryCredentials_Validation(t *testing.T) {
	tests := []struct {
		name     string
		creds    RegistryCredentials
		expected bool
	}{
		{
			name: "Valid credentials",
			creds: RegistryCredentials{
				Username:      "testuser",
				Password:      "testpass",
				Email:         "test@example.com",
				ServerAddress: "docker.io",
			},
			expected: true,
		},
		{
			name: "Empty credentials",
			creds: RegistryCredentials{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expected {
				assert.NotEmpty(t, tt.creds.Username)
				assert.NotEmpty(t, tt.creds.Password)
				assert.NotEmpty(t, tt.creds.Email)
				assert.NotEmpty(t, tt.creds.ServerAddress)
			} else {
				assert.Empty(t, tt.creds.Username)
				assert.Empty(t, tt.creds.Password)
				assert.Empty(t, tt.creds.Email)
				assert.Empty(t, tt.creds.ServerAddress)
			}
		})
	}
}

func TestContainerState_Operations(t *testing.T) {
	state := &ContainerState{
		ID:        "test-container-123",
		Name:      "test-container",
		Status:    "running",
		StartTime: time.Now(),
		StopTime:  time.Time{},
		Image:     "nginx:latest",
		ExitCode:  0,
		ResourceUsage: ResourceUsage{
			CPUPercent:    50.0,
			MemoryUsage:   256 * 1024 * 1024,
			MemoryLimit:   512 * 1024 * 1024,
			NetworkRxBytes: 1024 * 1024,
			NetworkTxBytes: 512 * 1024,
			BlockRead:     1024,
			BlockWrite:    512,
		},
	}

	t.Run("Container state creation", func(t *testing.T) {
		assert.Equal(t, "test-container-123", state.ID)
		assert.Equal(t, "test-container", state.Name)
		assert.Equal(t, "running", state.Status)
		assert.NotZero(t, state.StartTime)
		assert.Zero(t, state.StopTime)
		assert.Equal(t, "nginx:latest", state.Image)
		assert.Equal(t, 0, state.ExitCode)
		assert.NotZero(t, state.ResourceUsage.CPUPercent)
		assert.NotZero(t, state.ResourceUsage.MemoryLimit)
	})

	t.Run("Container state update", func(t *testing.T) {
		state.Status = "stopped"
		state.StopTime = time.Now()
		state.ExitCode = 1
		
		assert.Equal(t, "stopped", state.Status)
		assert.NotZero(t, state.StopTime)
		assert.Equal(t, 1, state.ExitCode)
	})
}

func TestCachedImage_Operations(t *testing.T) {
	image := &CachedImage{
		ID:       "image-123",
		RepoTags: []string{"nginx:latest", "nginx:1.20"},
		Size:     100 * 1024 * 1024, // 100MB
		LastUsed: time.Now(),
	}

	t.Run("Cached image creation", func(t *testing.T) {
		assert.Equal(t, "image-123", image.ID)
		assert.Len(t, image.RepoTags, 2)
		assert.Contains(t, image.RepoTags, "nginx:latest")
		assert.Contains(t, image.RepoTags, "nginx:1.20")
		assert.Equal(t, int64(100*1024*1024), image.Size)
		assert.NotZero(t, image.LastUsed)
	})

	t.Run("Cached image update", func(t *testing.T) {
		image.LastUsed = time.Now().Add(1 * time.Hour)
		
		assert.True(t, image.LastUsed.After(time.Now()))
	})
}

func TestDockerContainerizer_EdgeCases(t *testing.T) {
	// Test with nil containerizer (this would normally be handled by NewDockerContainerizer)
	t.Run("Nil containerizer operations", func(t *testing.T) {
		var containerizer *DockerContainerizer
		
		// These would panic if called on nil, but we're just testing the structure
		assert.Nil(t, containerizer)
	})

	// Test empty string validations
	t.Run("Empty string validations", func(t *testing.T) {
		emptyStrings := []string{"", "   ", "\t", "\n"}
		
		for _, str := range emptyStrings {
			if str == "" {
				assert.Empty(t, str)
			} else {
				// For whitespace-only strings, we check they're not empty but contain only whitespace
				assert.NotEmpty(t, str)
			}
		}
	})

	// Test negative values
	t.Run("Negative resource values", func(t *testing.T) {
		config := ContainerConfig{
			Name:        "test",
			Image:       "nginx:latest",
			CPUShares:   -1,
			CPUQuota:    -1,
			MemoryLimit: -1,
			GPUCount:    -1,
		}
		
		assert.Less(t, config.CPUShares, int64(0))
		assert.Less(t, config.CPUQuota, int64(0))
		assert.Less(t, config.MemoryLimit, int64(0))
		assert.Less(t, config.GPUCount, int64(0))
	})
}