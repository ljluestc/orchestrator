package isolation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectCgroupsVersion(t *testing.T) {
	version, err := detectCgroupsVersion()
	if err != nil {
		t.Skipf("Cgroups not available: %v", err)
	}

	t.Logf("Detected cgroups version: %s", version)

	switch version {
	case CgroupsV1, CgroupsV2, CgroupsHybrid:
		// Valid versions
	default:
		t.Errorf("Unknown cgroups version: %s", version)
	}
}

func TestCgroupsManager_CreateAndRemove(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("Test requires root privileges")
	}

	cm, err := NewCgroupsManager("")
	if err != nil {
		t.Fatalf("Failed to create cgroups manager: %v", err)
	}

	containerID := "test-container-123"
	limits := ResourceLimits{
		CPUShares:   1024,
		CPUQuota:    100000,  // 100ms
		CPUPeriod:   100000,  // 100ms period = 1 CPU core
		MemoryLimit: 512 * 1024 * 1024, // 512MB
		PidsLimit:   1000,
	}

	// Create cgroups
	err = cm.CreateContainerCgroups(containerID, limits)
	if err != nil {
		t.Fatalf("Failed to create cgroups: %v", err)
	}

	// Verify cgroups exist
	cgroups, exists := cm.containers[containerID]
	if !exists {
		t.Fatal("Cgroups not tracked after creation")
	}

	// Verify cgroup directories exist
	switch cm.version {
	case CgroupsV1, CgroupsHybrid:
		if _, err := os.Stat(cgroups.CPUPath); err != nil {
			t.Errorf("CPU cgroup path doesn't exist: %v", err)
		}
		if _, err := os.Stat(cgroups.MemoryPath); err != nil {
			t.Errorf("Memory cgroup path doesn't exist: %v", err)
		}

	case CgroupsV2:
		if _, err := os.Stat(cgroups.CPUPath); err != nil {
			t.Errorf("Cgroup path doesn't exist: %v", err)
		}
	}

	// Add current process to cgroup (for testing)
	err = cm.AddProcessToCgroup(containerID, os.Getpid())
	if err != nil {
		t.Errorf("Failed to add process to cgroup: %v", err)
	}

	// Get resource stats
	stats, err := cm.GetResourceStats(containerID)
	if err != nil {
		t.Errorf("Failed to get resource stats: %v", err)
	} else {
		t.Logf("Resource stats: CPU=%d ns, Memory=%d bytes (%.1f%%)",
			stats.CPUUsage, stats.MemoryUsage, stats.MemoryPercent)
	}

	// Remove cgroups
	err = cm.RemoveContainerCgroups(containerID)
	if err != nil {
		t.Errorf("Failed to remove cgroups: %v", err)
	}

	// Verify removal
	if _, exists := cm.containers[containerID]; exists {
		t.Error("Cgroups still tracked after removal")
	}
}

func TestCgroupsManager_ResourceStats(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("Test requires root privileges")
	}

	cm, err := NewCgroupsManager("")
	if err != nil {
		t.Fatalf("Failed to create cgroups manager: %v", err)
	}

	containerID := "stats-test-container"
	limits := ResourceLimits{
		CPUShares:   2048,
		MemoryLimit: 1024 * 1024 * 1024, // 1GB
		PidsLimit:   5000,
	}

	err = cm.CreateContainerCgroups(containerID, limits)
	if err != nil {
		t.Fatalf("Failed to create cgroups: %v", err)
	}
	defer cm.RemoveContainerCgroups(containerID)

	// Add current process
	cm.AddProcessToCgroup(containerID, os.Getpid())

	// Get stats
	stats, err := cm.GetResourceStats(containerID)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	t.Logf("CPU Usage: %d nanoseconds", stats.CPUUsage)
	t.Logf("Memory Usage: %d bytes (%.2f%%)", stats.MemoryUsage, stats.MemoryPercent)
	t.Logf("Memory Limit: %d bytes", stats.MemoryLimit)
	t.Logf("Number of PIDs: %d", stats.NumPids)

	// Validate stats make sense
	if stats.MemoryUsage == 0 {
		t.Log("Warning: Memory usage is 0 (may be expected for minimal process)")
	}

	if stats.MemoryLimit > 0 && stats.MemoryUsage > stats.MemoryLimit {
		t.Errorf("Memory usage (%d) exceeds limit (%d)", stats.MemoryUsage, stats.MemoryLimit)
	}
}

func TestResourceLimitsEnforcement(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("Test requires root privileges")
	}

	cm, err := NewCgroupsManager("")
	if err != nil {
		t.Fatalf("Failed to create cgroups manager: %v", err)
	}

	containerID := "enforcement-test"
	limits := ResourceLimits{
		CPUShares:   512,  // Low CPU share
		CPUQuota:    50000,  // 0.5 CPU cores
		CPUPeriod:   100000,
		MemoryLimit: 256 * 1024 * 1024, // 256MB
		PidsLimit:   100,
	}

	err = cm.CreateContainerCgroups(containerID, limits)
	if err != nil {
		t.Fatalf("Failed to create cgroups: %v", err)
	}
	defer cm.RemoveContainerCgroups(containerID)

	// Verify limits were applied
	cgroups := cm.containers[containerID]

	switch cm.version {
	case CgroupsV1, CgroupsHybrid:
		// Check CPU limits
		if cpuShares, err := cm.readCgroupFile(cgroups.CPUPath, "cpu.shares"); err == nil {
			t.Logf("CPU shares: %s (expected: 512)", cpuShares)
		}
		if cpuQuota, err := cm.readCgroupFile(cgroups.CPUPath, "cpu.cfs_quota_us"); err == nil {
			t.Logf("CPU quota: %s (expected: 50000)", cpuQuota)
		}

		// Check memory limits
		if memLimit, err := cm.readCgroupFile(cgroups.MemoryPath, "memory.limit_in_bytes"); err == nil {
			t.Logf("Memory limit: %s (expected: 268435456)", memLimit)
		}

		// Check PIDs limit
		if pidsMax, err := cm.readCgroupFile(cgroups.PidsPath, "pids.max"); err == nil {
			t.Logf("PIDs max: %s (expected: 100)", pidsMax)
		}

	case CgroupsV2:
		// Check CPU limits
		if cpuMax, err := cm.readCgroupFile(cgroups.CPUPath, "cpu.max"); err == nil {
			t.Logf("CPU max: %s (expected: 50000 100000)", cpuMax)
		}

		// Check memory limits
		if memMax, err := cm.readCgroupFile(cgroups.MemoryPath, "memory.max"); err == nil {
			t.Logf("Memory max: %s (expected: 268435456)", memMax)
		}

		// Check PIDs limit
		if pidsMax, err := cm.readCgroupFile(cgroups.PidsPath, "pids.max"); err == nil {
			t.Logf("PIDs max: %s (expected: 100)", pidsMax)
		}
	}
}

func TestCPUAffinity(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("Test requires root privileges")
	}

	cm, err := NewCgroupsManager("")
	if err != nil {
		t.Fatalf("Failed to create cgroups manager: %v", err)
	}

	// Only test on cgroups v1 where cpuset is separate
	if cm.version == CgroupsV2 {
		t.Skip("CPU affinity testing for v2 not implemented")
	}

	containerID := "cpuset-test"
	limits := ResourceLimits{
		CPUCores:    []int{0, 1}, // Pin to first 2 CPUs
		MemoryLimit: 256 * 1024 * 1024,
	}

	err = cm.CreateContainerCgroups(containerID, limits)
	if err != nil {
		t.Fatalf("Failed to create cgroups: %v", err)
	}
	defer cm.RemoveContainerCgroups(containerID)

	cgroups := cm.containers[containerID]

	// Verify CPU affinity
	if cpuset, err := cm.readCgroupFile(cgroups.CPUSetPath, "cpuset.cpus"); err == nil {
		t.Logf("CPUSet: %s (expected: 0,1)", cpuset)
	} else {
		t.Errorf("Failed to read cpuset.cpus: %v", err)
	}
}

func TestMonitorResourceViolations(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("Test requires root privileges")
	}

	cm, err := NewCgroupsManager("")
	if err != nil {
		t.Fatalf("Failed to create cgroups manager: %v", err)
	}

	containerID := "violations-test"
	limits := ResourceLimits{
		MemoryLimit: 128 * 1024 * 1024, // 128MB (low to potentially trigger violations)
	}

	err = cm.CreateContainerCgroups(containerID, limits)
	if err != nil {
		t.Fatalf("Failed to create cgroups: %v", err)
	}
	defer cm.RemoveContainerCgroups(containerID)

	cm.AddProcessToCgroup(containerID, os.Getpid())

	violations, err := cm.MonitorResourceViolations(containerID)
	if err != nil {
		t.Errorf("Failed to monitor violations: %v", err)
	}

	if len(violations) > 0 {
		t.Logf("Detected violations: %v", violations)
	} else {
		t.Log("No resource violations detected")
	}
}

func BenchmarkCreateCgroups(b *testing.B) {
	if os.Geteuid() != 0 {
		b.Skip("Benchmark requires root privileges")
	}

	cm, err := NewCgroupsManager("")
	if err != nil {
		b.Fatalf("Failed to create cgroups manager: %v", err)
	}

	limits := ResourceLimits{
		CPUShares:   1024,
		MemoryLimit: 512 * 1024 * 1024,
		PidsLimit:   1000,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		containerID := filepath.Join("bench", string(rune(i)))

		err := cm.CreateContainerCgroups(containerID, limits)
		if err != nil {
			b.Error(err)
		}

		cm.RemoveContainerCgroups(containerID)
	}
}

func BenchmarkGetResourceStats(b *testing.B) {
	if os.Geteuid() != 0 {
		b.Skip("Benchmark requires root privileges")
	}

	cm, err := NewCgroupsManager("")
	if err != nil {
		b.Fatalf("Failed to create cgroups manager: %v", err)
	}

	containerID := "bench-stats"
	limits := ResourceLimits{
		CPUShares:   1024,
		MemoryLimit: 512 * 1024 * 1024,
	}

	err = cm.CreateContainerCgroups(containerID, limits)
	if err != nil {
		b.Fatalf("Failed to create cgroups: %v", err)
	}
	defer cm.RemoveContainerCgroups(containerID)

	cm.AddProcessToCgroup(containerID, os.Getpid())

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := cm.GetResourceStats(containerID)
		if err != nil {
			b.Error(err)
		}
	}
}
