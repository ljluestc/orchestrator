package isolation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCgroupsManager(t *testing.T) {
	tests := []struct {
		name        string
		rootPath    string
		expectError bool
	}{
		{
			name:        "Default root path",
			rootPath:    "",
			expectError: false,
		},
		{
			name:        "Custom root path",
			rootPath:    "/tmp/test-cgroups",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir := t.TempDir()
			if tt.rootPath == "" {
				tt.rootPath = tempDir
			}

			// Create mock cgroups structure
			setupMockCgroups(t, tt.rootPath)

			manager, err := NewCgroupsManager(tt.rootPath)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, manager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
				assert.Equal(t, tt.rootPath, manager.rootPath)
			}
		})
	}
}

func TestCgroupsManager_CreateContainerCgroups(t *testing.T) {
	tempDir := t.TempDir()
	setupMockCgroups(t, tempDir)

	manager, err := NewCgroupsManager(tempDir)
	require.NoError(t, err)

	tests := []struct {
		name        string
		containerID string
		limits      ResourceLimits
		expectError bool
	}{
		{
			name:        "Basic container with CPU limits",
			containerID: "test-container-1",
			limits: ResourceLimits{
				CPUShares:   512,
				CPUQuota:    100000,
				CPUPeriod:   100000,
				MemoryLimit: 1024 * 1024 * 1024, // 1GB
			},
			expectError: false,
		},
		{
			name:        "Container with all limits",
			containerID: "test-container-2",
			limits: ResourceLimits{
				CPUShares:      1024,
				CPUQuota:       200000,
				CPUPeriod:      100000,
				CPUCores:       []int{0, 1},
				MemoryLimit:    2048 * 1024 * 1024, // 2GB
				MemorySwap:     4096 * 1024 * 1024, // 4GB
				OOMKillDisable: true,
				BlkIOWeight:    500,
				BlkIOReadBPS:   1024 * 1024, // 1MB/s
				BlkIOWriteBPS:  1024 * 1024, // 1MB/s
				PidsLimit:      1000,
			},
			expectError: false,
		},
		{
			name:        "Container with minimal limits",
			containerID: "test-container-3",
			limits: ResourceLimits{
				CPUShares: 256,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.CreateContainerCgroups(tt.containerID, tt.limits)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// Verify container was registered
				manager.mu.RLock()
				_, exists := manager.containers[tt.containerID]
				manager.mu.RUnlock()
				assert.True(t, exists)
			}
		})
	}
}

func TestCgroupsManager_AddProcessToCgroup(t *testing.T) {
	tempDir := t.TempDir()
	setupMockCgroups(t, tempDir)

	manager, err := NewCgroupsManager(tempDir)
	require.NoError(t, err)

	// Create a container first
	containerID := "test-container"
	limits := ResourceLimits{
		CPUShares:   512,
		MemoryLimit: 1024 * 1024 * 1024,
	}
	err = manager.CreateContainerCgroups(containerID, limits)
	require.NoError(t, err)

	tests := []struct {
		name        string
		containerID string
		pid         int
		expectError bool
	}{
		{
			name:        "Valid container and PID",
			containerID: containerID,
			pid:         1234,
			expectError: false,
		},
		{
			name:        "Non-existent container",
			containerID: "non-existent",
			pid:         1234,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.AddProcessToCgroup(tt.containerID, tt.pid)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCgroupsManager_GetResourceStats(t *testing.T) {
	tempDir := t.TempDir()
	setupMockCgroups(t, tempDir)

	manager, err := NewCgroupsManager(tempDir)
	require.NoError(t, err)

	// Create a container first
	containerID := "test-container"
	limits := ResourceLimits{
		CPUShares:   512,
		MemoryLimit: 1024 * 1024 * 1024,
	}
	err = manager.CreateContainerCgroups(containerID, limits)
	require.NoError(t, err)

	tests := []struct {
		name        string
		containerID string
		expectError bool
	}{
		{
			name:        "Valid container",
			containerID: containerID,
			expectError: false,
		},
		{
			name:        "Non-existent container",
			containerID: "non-existent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := manager.GetResourceStats(tt.containerID)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, stats)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, stats)
				// Stats values depend on mock files, so we just check structure
				assert.GreaterOrEqual(t, stats.MemoryPercent, 0.0)
			}
		})
	}
}

func TestCgroupsManager_RemoveContainerCgroups(t *testing.T) {
	tempDir := t.TempDir()
	setupMockCgroups(t, tempDir)

	manager, err := NewCgroupsManager(tempDir)
	require.NoError(t, err)

	// Create a container first
	containerID := "test-container"
	limits := ResourceLimits{
		CPUShares:   512,
		MemoryLimit: 1024 * 1024 * 1024,
	}
	err = manager.CreateContainerCgroups(containerID, limits)
	require.NoError(t, err)

	tests := []struct {
		name        string
		containerID string
		expectError bool
	}{
		{
			name:        "Valid container",
			containerID: containerID,
			expectError: false,
		},
		{
			name:        "Non-existent container",
			containerID: "non-existent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.RemoveContainerCgroups(tt.containerID)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// Verify container was removed
				manager.mu.RLock()
				_, exists := manager.containers[tt.containerID]
				manager.mu.RUnlock()
				assert.False(t, exists)
			}
		})
	}
}

func TestCgroupsManager_MonitorResourceViolations(t *testing.T) {
	tempDir := t.TempDir()
	setupMockCgroups(t, tempDir)

	manager, err := NewCgroupsManager(tempDir)
	require.NoError(t, err)

	// Create a container first
	containerID := "test-container"
	limits := ResourceLimits{
		CPUShares:   512,
		MemoryLimit: 1024 * 1024 * 1024,
	}
	err = manager.CreateContainerCgroups(containerID, limits)
	require.NoError(t, err)

	tests := []struct {
		name        string
		containerID string
		expectError bool
	}{
		{
			name:        "Valid container",
			containerID: containerID,
			expectError: false,
		},
		{
			name:        "Non-existent container",
			containerID: "non-existent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations, err := manager.MonitorResourceViolations(tt.containerID)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, violations)
			} else {
				assert.NoError(t, err)
				// Violations can be nil if no violations found, so we just check it's a slice when not nil
				if violations != nil {
					assert.IsType(t, []string{}, violations)
				}
			}
		})
	}
}

func TestFormatCPUList(t *testing.T) {
	tests := []struct {
		name     string
		cores    []int
		expected string
	}{
		{
			name:     "Single core",
			cores:    []int{0},
			expected: "0",
		},
		{
			name:     "Multiple cores",
			cores:    []int{0, 1, 2, 3},
			expected: "0,1,2,3",
		},
		{
			name:     "Non-sequential cores",
			cores:    []int{0, 2, 4, 6},
			expected: "0,2,4,6",
		},
		{
			name:     "Empty cores",
			cores:    []int{},
			expected: "",
		},
		{
			name:     "Nil cores",
			cores:    nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCPUList(tt.cores)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCgroupsManager_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	setupMockCgroups(t, tempDir)

	manager, err := NewCgroupsManager(tempDir)
	require.NoError(t, err)

	t.Run("CreateContainerCgroups with empty container ID", func(t *testing.T) {
		err := manager.CreateContainerCgroups("", ResourceLimits{})
		assert.Error(t, err)
	})

	t.Run("AddProcessToCgroup with invalid PID", func(t *testing.T) {
		// Create a container first
		containerID := "test-container"
		limits := ResourceLimits{CPUShares: 512}
		err := manager.CreateContainerCgroups(containerID, limits)
		require.NoError(t, err)

		// Test with invalid PID
		err = manager.AddProcessToCgroup(containerID, -1)
		assert.NoError(t, err) // Should not error, just log
	})

	t.Run("GetResourceStats for non-existent container", func(t *testing.T) {
		stats, err := manager.GetResourceStats("non-existent")
		assert.Error(t, err)
		assert.Nil(t, stats)
	})

	t.Run("RemoveContainerCgroups for non-existent container", func(t *testing.T) {
		err := manager.RemoveContainerCgroups("non-existent")
		assert.Error(t, err)
	})

	t.Run("MonitorResourceViolations for non-existent container", func(t *testing.T) {
		violations, err := manager.MonitorResourceViolations("non-existent")
		assert.Error(t, err)
		assert.Nil(t, violations)
	})
}

// setupMockCgroups creates a mock cgroups directory structure for testing
func setupMockCgroups(t *testing.T, rootPath string) {
	// Create cgroups v1 structure
	cgroupsV1Dirs := []string{
		"cpu", "memory", "blkio", "pids", "cpuset", "devices", "freezer",
	}

	for _, dir := range cgroupsV1Dirs {
		dirPath := filepath.Join(rootPath, dir)
		err := os.MkdirAll(dirPath, 0755)
		require.NoError(t, err)

		// Create some mock files
		switch dir {
		case "cpu":
			createMockFile(t, filepath.Join(dirPath, "cpu.shares"), "1024")
			createMockFile(t, filepath.Join(dirPath, "cpu.cfs_quota_us"), "100000")
			createMockFile(t, filepath.Join(dirPath, "cpu.cfs_period_us"), "100000")
			createMockFile(t, filepath.Join(dirPath, "cpuacct.usage"), "1000000000")
		case "memory":
			createMockFile(t, filepath.Join(dirPath, "memory.limit_in_bytes"), "1073741824")
			createMockFile(t, filepath.Join(dirPath, "memory.memsw.limit_in_bytes"), "2147483648")
			createMockFile(t, filepath.Join(dirPath, "memory.usage_in_bytes"), "536870912")
			createMockFile(t, filepath.Join(dirPath, "memory.oom_control"), "oom_kill 0")
		case "pids":
			createMockFile(t, filepath.Join(dirPath, "pids.max"), "1000")
			createMockFile(t, filepath.Join(dirPath, "pids.current"), "10")
		case "cpuset":
			createMockFile(t, filepath.Join(dirPath, "cpuset.cpus"), "0-3")
			createMockFile(t, filepath.Join(dirPath, "cpuset.mems"), "0")
		}
	}

	// Create cgroups v2 structure
	cgroupsV2File := filepath.Join(rootPath, "cgroup.controllers")
	createMockFile(t, cgroupsV2File, "cpu memory io pids")

	// Create some mock v2 files
	createMockFile(t, filepath.Join(rootPath, "cpu.stat"), "usage_usec 1000000")
	createMockFile(t, filepath.Join(rootPath, "memory.current"), "536870912")
	createMockFile(t, filepath.Join(rootPath, "memory.max"), "1073741824")
	createMockFile(t, filepath.Join(rootPath, "pids.current"), "10")
}

// createMockFile creates a mock file with the given content
func createMockFile(t *testing.T, path, content string) {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	require.NoError(t, err)
	
	err = os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
}