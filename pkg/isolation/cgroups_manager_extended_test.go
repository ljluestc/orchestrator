package isolation

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetectCgroupsVersion tests version detection behavior
func TestDetectCgroupsVersion(t *testing.T) {
	// Test that detectCgroupsVersion is called during NewCgroupsManager
	// The actual behavior depends on the host system, so we just verify it doesn't error
	// when a valid cgroup structure exists

	t.Run("NewCgroupsManager uses detected version", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create v2 structure
		os.WriteFile(filepath.Join(tempDir, "cgroup.controllers"), []byte("cpu memory io"), 0644)

		manager, err := NewCgroupsManager(tempDir)
		require.NoError(t, err)
		require.NotNil(t, manager)

		// Version should be one of the valid types
		validVersions := []CgroupsVersion{CgroupsV1, CgroupsV2, CgroupsHybrid}
		assert.Contains(t, validVersions, manager.version)
	})

	t.Run("NewCgroupsManager with v1 structure", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create only v1 structure (no cgroup.controllers)
		os.MkdirAll(filepath.Join(tempDir, "cpu"), 0755)

		manager, err := NewCgroupsManager(tempDir)
		require.NoError(t, err)
		require.NotNil(t, manager)

		// Should detect some version
		assert.NotEmpty(t, manager.version)
	})
}

// TestCgroupsManager_CgroupsV1_CreateContainer tests v1-specific creation
func TestCgroupsManager_CgroupsV1_CreateContainer(t *testing.T) {
	tempDir := t.TempDir()

	// Create v1-only structure
	setupCgroupsV1Only(t, tempDir)

	// Force v1 mode
	manager := &CgroupsManager{
		version:    CgroupsV1,
		rootPath:   tempDir,
		containers: make(map[string]*ContainerCgroups),
	}

	tests := []struct {
		name        string
		containerID string
		limits      ResourceLimits
		expectError bool
	}{
		{
			name:        "Create v1 container with CPU limits",
			containerID: "v1-container-cpu",
			limits: ResourceLimits{
				CPUShares:   512,
				CPUQuota:    100000,
				CPUPeriod:   100000,
			},
			expectError: false,
		},
		{
			name:        "Create v1 container with memory limits",
			containerID: "v1-container-mem",
			limits: ResourceLimits{
				MemoryLimit:    1024 * 1024 * 1024, // 1GB
				MemorySwap:     2048 * 1024 * 1024, // 2GB
				OOMKillDisable: true,
			},
			expectError: false,
		},
		{
			name:        "Create v1 container with Block IO limits",
			containerID: "v1-container-blkio",
			limits: ResourceLimits{
				BlkIOWeight: 500,
			},
			expectError: false,
		},
		{
			name:        "Create v1 container with PIDs limit",
			containerID: "v1-container-pids",
			limits: ResourceLimits{
				PidsLimit: 1000,
			},
			expectError: false,
		},
		{
			name:        "Create v1 container with CPU affinity",
			containerID: "v1-container-cpuset",
			limits: ResourceLimits{
				CPUCores: []int{0, 1, 2},
			},
			expectError: false,
		},
		{
			name:        "Create v1 container with all limits",
			containerID: "v1-container-all",
			limits: ResourceLimits{
				CPUShares:      1024,
				CPUQuota:       200000,
				CPUPeriod:      100000,
				CPUCores:       []int{0, 1},
				MemoryLimit:    2048 * 1024 * 1024,
				MemorySwap:     4096 * 1024 * 1024,
				OOMKillDisable: true,
				BlkIOWeight:    500,
				BlkIOReadBPS:   1024 * 1024,
				BlkIOWriteBPS:  1024 * 1024,
				PidsLimit:      1000,
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
				cgroups, exists := manager.containers[tt.containerID]
				manager.mu.RUnlock()

				assert.True(t, exists)
				assert.NotNil(t, cgroups)

				// Verify v1 paths were created
				assert.Contains(t, cgroups.CPUPath, "cpu")
				assert.Contains(t, cgroups.MemoryPath, "memory")
				assert.Contains(t, cgroups.BlkIOPath, "blkio")
				assert.Contains(t, cgroups.PidsPath, "pids")
				assert.Contains(t, cgroups.CPUSetPath, "cpuset")
				assert.Contains(t, cgroups.DevicesPath, "devices")
				assert.Contains(t, cgroups.FreezerPath, "freezer")

				// Verify directories were created
				assert.DirExists(t, cgroups.CPUPath)
				assert.DirExists(t, cgroups.MemoryPath)

				// Verify limit files were written (if they were set)
				if tt.limits.CPUShares > 0 {
					content, _ := os.ReadFile(filepath.Join(cgroups.CPUPath, "cpu.shares"))
					expectedShares := strconv.FormatInt(tt.limits.CPUShares, 10)
					assert.Equal(t, expectedShares, strings.TrimSpace(string(content)))
				}
			}
		})
	}
}

// TestCgroupsManager_CgroupsV1_GetStats tests v1-specific stats retrieval
func TestCgroupsManager_CgroupsV1_GetStats(t *testing.T) {
	tempDir := t.TempDir()
	setupCgroupsV1WithStats(t, tempDir)

	manager := &CgroupsManager{
		version:    CgroupsV1,
		rootPath:   tempDir,
		containers: make(map[string]*ContainerCgroups),
	}

	// Create a container
	containerID := "test-v1-stats"
	limits := ResourceLimits{
		CPUShares:   512,
		MemoryLimit: 1024 * 1024 * 1024,
	}
	err := manager.CreateContainerCgroups(containerID, limits)
	require.NoError(t, err)

	// Write mock stat files for this container
	cgroups := manager.containers[containerID]

	// CPU stats
	os.WriteFile(filepath.Join(cgroups.CPUPath, "cpuacct.usage"), []byte("5000000000"), 0644)

	// Memory stats
	os.WriteFile(filepath.Join(cgroups.MemoryPath, "memory.usage_in_bytes"), []byte("536870912"), 0644)
	os.WriteFile(filepath.Join(cgroups.MemoryPath, "memory.limit_in_bytes"), []byte("1073741824"), 0644)
	os.WriteFile(filepath.Join(cgroups.MemoryPath, "memory.oom_control"), []byte("oom_kill 3\nother_field 123"), 0644)

	// PIDs stats
	os.WriteFile(filepath.Join(cgroups.PidsPath, "pids.current"), []byte("42"), 0644)

	// Get stats
	stats, err := manager.GetResourceStats(containerID)
	require.NoError(t, err)
	require.NotNil(t, stats)

	// Verify v1 stats were read correctly
	assert.Equal(t, uint64(5000000000), stats.CPUUsage)
	assert.Equal(t, uint64(536870912), stats.MemoryUsage)
	assert.Equal(t, uint64(1073741824), stats.MemoryLimit)
	assert.InDelta(t, 50.0, stats.MemoryPercent, 0.1) // 536870912 / 1073741824 * 100 = 50%
	assert.Equal(t, 3, stats.OOMKillCount)
	assert.Equal(t, 42, stats.NumPids)
}

// TestCgroupsManager_CgroupsV2_GetStats tests v2-specific stats retrieval
func TestCgroupsManager_CgroupsV2_GetStats(t *testing.T) {
	tempDir := t.TempDir()

	manager := &CgroupsManager{
		version:    CgroupsV2,
		rootPath:   tempDir,
		containers: make(map[string]*ContainerCgroups),
	}

	// Create a container
	containerID := "test-v2-stats"
	limits := ResourceLimits{
		MemoryLimit: 1024 * 1024 * 1024,
	}
	err := manager.CreateContainerCgroups(containerID, limits)
	require.NoError(t, err)

	cgroups := manager.containers[containerID]

	// Write v2 stat files
	os.WriteFile(filepath.Join(cgroups.CPUPath, "cpu.stat"), []byte("usage_usec 2000000\nother_stat 123"), 0644)
	os.WriteFile(filepath.Join(cgroups.MemoryPath, "memory.current"), []byte("268435456"), 0644)
	os.WriteFile(filepath.Join(cgroups.MemoryPath, "memory.max"), []byte("1073741824"), 0644)
	os.WriteFile(filepath.Join(cgroups.PidsPath, "pids.current"), []byte("25"), 0644)

	// Get stats
	stats, err := manager.GetResourceStats(containerID)
	require.NoError(t, err)
	require.NotNil(t, stats)

	// Verify v2 stats were read correctly
	assert.Equal(t, uint64(2000000000), stats.CPUUsage) // 2000000 * 1000 = 2000000000 ns
	assert.Equal(t, uint64(268435456), stats.MemoryUsage)
	assert.Equal(t, uint64(1073741824), stats.MemoryLimit)
	assert.InDelta(t, 25.0, stats.MemoryPercent, 0.1) // 268435456 / 1073741824 * 100 = 25%
	assert.Equal(t, 25, stats.NumPids)
}

// TestCgroupsManager_CgroupsV2_GetStats_UnlimitedMemory tests v2 stats with "max" memory
func TestCgroupsManager_CgroupsV2_GetStats_UnlimitedMemory(t *testing.T) {
	tempDir := t.TempDir()

	manager := &CgroupsManager{
		version:    CgroupsV2,
		rootPath:   tempDir,
		containers: make(map[string]*ContainerCgroups),
	}

	containerID := "test-v2-unlimited"
	err := manager.CreateContainerCgroups(containerID, ResourceLimits{})
	require.NoError(t, err)

	cgroups := manager.containers[containerID]

	// Write v2 stat files with "max" for memory limit
	os.WriteFile(filepath.Join(cgroups.MemoryPath, "memory.current"), []byte("268435456"), 0644)
	os.WriteFile(filepath.Join(cgroups.MemoryPath, "memory.max"), []byte("max"), 0644)

	stats, err := manager.GetResourceStats(containerID)
	require.NoError(t, err)
	require.NotNil(t, stats)

	// When memory limit is "max", it should be 0 and percent should be 0
	assert.Equal(t, uint64(0), stats.MemoryLimit)
	assert.Equal(t, 0.0, stats.MemoryPercent)
}

// TestCgroupsManager_AddProcessToCgroup_V1 tests adding process to v1 cgroups
func TestCgroupsManager_AddProcessToCgroup_V1(t *testing.T) {
	tempDir := t.TempDir()
	setupCgroupsV1Only(t, tempDir)

	manager := &CgroupsManager{
		version:    CgroupsV1,
		rootPath:   tempDir,
		containers: make(map[string]*ContainerCgroups),
	}

	// Create container
	containerID := "test-v1-process"
	err := manager.CreateContainerCgroups(containerID, ResourceLimits{CPUShares: 512})
	require.NoError(t, err)

	// Add process
	err = manager.AddProcessToCgroup(containerID, 12345)
	assert.NoError(t, err)

	// Verify PID was written to all v1 cgroup hierarchies
	cgroups := manager.containers[containerID]
	paths := []string{
		cgroups.CPUPath,
		cgroups.MemoryPath,
		cgroups.BlkIOPath,
		cgroups.PidsPath,
		cgroups.CPUSetPath,
	}

	for _, path := range paths {
		procFile := filepath.Join(path, "cgroup.procs")
		if _, err := os.Stat(procFile); err == nil {
			content, _ := os.ReadFile(procFile)
			assert.Contains(t, string(content), "12345")
		}
	}
}

// TestCgroupsManager_AddProcessToCgroup_V2 tests adding process to v2 cgroups
func TestCgroupsManager_AddProcessToCgroup_V2(t *testing.T) {
	tempDir := t.TempDir()

	manager := &CgroupsManager{
		version:    CgroupsV2,
		rootPath:   tempDir,
		containers: make(map[string]*ContainerCgroups),
	}

	// Create container
	containerID := "test-v2-process"
	err := manager.CreateContainerCgroups(containerID, ResourceLimits{})
	require.NoError(t, err)

	// Add process
	err = manager.AddProcessToCgroup(containerID, 67890)
	assert.NoError(t, err)

	// Verify PID was written to unified hierarchy
	cgroups := manager.containers[containerID]
	procFile := filepath.Join(cgroups.CPUPath, "cgroup.procs")
	if _, err := os.Stat(procFile); err == nil {
		content, _ := os.ReadFile(procFile)
		assert.Contains(t, string(content), "67890")
	}
}

// TestCgroupsManager_RemoveContainerCgroups_V1 tests v1 cleanup
func TestCgroupsManager_RemoveContainerCgroups_V1(t *testing.T) {
	tempDir := t.TempDir()
	setupCgroupsV1Only(t, tempDir)

	manager := &CgroupsManager{
		version:    CgroupsV1,
		rootPath:   tempDir,
		containers: make(map[string]*ContainerCgroups),
	}

	// Create container
	containerID := "test-v1-remove"
	err := manager.CreateContainerCgroups(containerID, ResourceLimits{CPUShares: 512})
	require.NoError(t, err)

	cgroups := manager.containers[containerID]

	// Verify directories exist
	assert.DirExists(t, cgroups.CPUPath)
	assert.DirExists(t, cgroups.MemoryPath)

	// Remove container
	err = manager.RemoveContainerCgroups(containerID)
	assert.NoError(t, err)

	// Verify container was removed from map
	_, exists := manager.containers[containerID]
	assert.False(t, exists)
}

// TestCgroupsManager_RemoveContainerCgroups_V2 tests v2 cleanup
func TestCgroupsManager_RemoveContainerCgroups_V2(t *testing.T) {
	tempDir := t.TempDir()

	manager := &CgroupsManager{
		version:    CgroupsV2,
		rootPath:   tempDir,
		containers: make(map[string]*ContainerCgroups),
	}

	// Create container
	containerID := "test-v2-remove"
	err := manager.CreateContainerCgroups(containerID, ResourceLimits{})
	require.NoError(t, err)

	cgroups := manager.containers[containerID]
	assert.DirExists(t, cgroups.CPUPath)

	// Remove container
	err = manager.RemoveContainerCgroups(containerID)
	assert.NoError(t, err)

	// Verify container was removed from map
	_, exists := manager.containers[containerID]
	assert.False(t, exists)
}

// TestCgroupsManager_MonitorResourceViolations_WithViolations tests violation detection
func TestCgroupsManager_MonitorResourceViolations_WithViolations(t *testing.T) {
	tempDir := t.TempDir()

	manager := &CgroupsManager{
		version:    CgroupsV2,
		rootPath:   tempDir,
		containers: make(map[string]*ContainerCgroups),
	}

	containerID := "test-violations"
	err := manager.CreateContainerCgroups(containerID, ResourceLimits{
		MemoryLimit: 1024 * 1024 * 1024,
	})
	require.NoError(t, err)

	cgroups := manager.containers[containerID]

	// Write stats that will trigger violations
	// Memory at 95% (above 90% threshold)
	os.WriteFile(filepath.Join(cgroups.MemoryPath, "memory.current"), []byte("1023410176"), 0644) // ~95% of 1GB
	os.WriteFile(filepath.Join(cgroups.MemoryPath, "memory.max"), []byte("1073741824"), 0644)

	// Check violations
	violations, err := manager.MonitorResourceViolations(containerID)
	require.NoError(t, err)
	require.NotNil(t, violations)

	// Should have memory violation
	assert.NotEmpty(t, violations)
	assert.Contains(t, violations[0], "Memory usage at")
}

// TestCgroupsManager_MonitorResourceViolations_OOMKills tests OOM kill detection
func TestCgroupsManager_MonitorResourceViolations_OOMKills(t *testing.T) {
	tempDir := t.TempDir()
	setupCgroupsV1Only(t, tempDir)

	manager := &CgroupsManager{
		version:    CgroupsV1,
		rootPath:   tempDir,
		containers: make(map[string]*ContainerCgroups),
	}

	containerID := "test-oom"
	err := manager.CreateContainerCgroups(containerID, ResourceLimits{
		MemoryLimit: 1024 * 1024 * 1024,
	})
	require.NoError(t, err)

	cgroups := manager.containers[containerID]

	// Write stats with OOM kills
	os.WriteFile(filepath.Join(cgroups.MemoryPath, "memory.usage_in_bytes"), []byte("536870912"), 0644)
	os.WriteFile(filepath.Join(cgroups.MemoryPath, "memory.limit_in_bytes"), []byte("1073741824"), 0644)
	os.WriteFile(filepath.Join(cgroups.MemoryPath, "memory.oom_control"), []byte("oom_kill 5"), 0644)
	os.WriteFile(filepath.Join(cgroups.PidsPath, "pids.current"), []byte("10"), 0644)

	// Check violations
	violations, err := manager.MonitorResourceViolations(containerID)
	require.NoError(t, err)
	require.NotNil(t, violations)

	// Should have OOM kill violation
	assert.NotEmpty(t, violations)
	found := false
	for _, v := range violations {
		if strings.Contains(v, "OOM killed") {
			found = true
			assert.Contains(t, v, "5 times")
		}
	}
	assert.True(t, found, "Should find OOM kill violation")
}

// TestCgroupsManager_ConcurrentOperations tests thread safety
func TestCgroupsManager_ConcurrentOperations(t *testing.T) {
	tempDir := t.TempDir()
	setupCgroupsV1Only(t, tempDir)

	manager := &CgroupsManager{
		version:    CgroupsV1,
		rootPath:   tempDir,
		containers: make(map[string]*ContainerCgroups),
	}

	// Run concurrent operations
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			containerID := string(rune('a' + id))

			// Create
			err := manager.CreateContainerCgroups(containerID, ResourceLimits{CPUShares: 512})
			assert.NoError(t, err)

			// Add process
			err = manager.AddProcessToCgroup(containerID, 1000+id)
			assert.NoError(t, err)

			// Get stats
			_, err = manager.GetResourceStats(containerID)
			assert.NoError(t, err)

			// Monitor violations
			_, err = manager.MonitorResourceViolations(containerID)
			assert.NoError(t, err)

			// Remove
			err = manager.RemoveContainerCgroups(containerID)
			assert.NoError(t, err)

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all containers were removed
	assert.Empty(t, manager.containers)
}

// setupCgroupsV1Only creates a v1-only cgroups structure
func setupCgroupsV1Only(t *testing.T, rootPath string) {
	cgroupsV1Dirs := []string{
		"cpu", "memory", "blkio", "pids", "cpuset", "devices", "freezer",
	}

	for _, dir := range cgroupsV1Dirs {
		dirPath := filepath.Join(rootPath, dir)
		err := os.MkdirAll(dirPath, 0755)
		require.NoError(t, err)

		// Create parent cpuset.mems for cpuset tests
		if dir == "cpuset" {
			os.WriteFile(filepath.Join(dirPath, "cpuset.mems"), []byte("0"), 0644)
		}
	}
}

// setupCgroupsV1WithStats creates a v1 structure with stat files
func setupCgroupsV1WithStats(t *testing.T, rootPath string) {
	setupCgroupsV1Only(t, rootPath)

	// Add stat files at root level for testing
	os.WriteFile(filepath.Join(rootPath, "cpu", "cpuacct.usage"), []byte("1000000000"), 0644)
	os.WriteFile(filepath.Join(rootPath, "memory", "memory.usage_in_bytes"), []byte("536870912"), 0644)
	os.WriteFile(filepath.Join(rootPath, "memory", "memory.limit_in_bytes"), []byte("1073741824"), 0644)
	os.WriteFile(filepath.Join(rootPath, "memory", "memory.oom_control"), []byte("oom_kill 0"), 0644)
	os.WriteFile(filepath.Join(rootPath, "pids", "pids.current"), []byte("10"), 0644)
}
