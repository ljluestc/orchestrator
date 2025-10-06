package probe

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostCollector_Collect(t *testing.T) {
	collector := NewHostCollector()

	info, err := collector.Collect()
	require.NoError(t, err)
	require.NotNil(t, info)

	// Validate hostname
	assert.NotEmpty(t, info.Hostname)

	// Validate kernel version
	assert.NotEmpty(t, info.KernelVersion)

	// Validate uptime
	assert.Greater(t, info.Uptime, time.Duration(0))
	assert.False(t, info.BootTime.IsZero())

	// Validate CPU info
	assert.NotEmpty(t, info.CPUInfo.Model)
	assert.Greater(t, info.CPUInfo.Cores, 0)
	assert.GreaterOrEqual(t, info.CPUInfo.Usage, 0.0)
	assert.LessOrEqual(t, info.CPUInfo.Usage, 100.0)

	// Validate memory info
	assert.Greater(t, info.MemoryInfo.TotalMB, uint64(0))
	assert.GreaterOrEqual(t, info.MemoryInfo.Usage, 0.0)
	assert.LessOrEqual(t, info.MemoryInfo.Usage, 100.0)

	// Validate load average
	assert.GreaterOrEqual(t, info.LoadAverage.Load1, 0.0)
	assert.GreaterOrEqual(t, info.LoadAverage.Load5, 0.0)
	assert.GreaterOrEqual(t, info.LoadAverage.Load15, 0.0)

	// Validate timestamp
	assert.False(t, info.Timestamp.IsZero())
}

func TestHostCollector_GetKernelVersion(t *testing.T) {
	// Create temporary proc directory
	tmpDir := t.TempDir()

	// Write test data
	versionData := "Linux version 5.10.0-test (gcc version 9.3.0)"
	err := os.WriteFile(filepath.Join(tmpDir, "version"), []byte(versionData), 0644)
	require.NoError(t, err)

	collector := NewHostCollectorWithPath(tmpDir)
	version, err := collector.getKernelVersion()
	require.NoError(t, err)
	assert.Equal(t, versionData, version)
}

func TestHostCollector_GetUptime(t *testing.T) {
	// Create temporary proc directory
	tmpDir := t.TempDir()

	// Write test data (100 seconds uptime)
	uptimeData := "100.50 200.00\n"
	err := os.WriteFile(filepath.Join(tmpDir, "uptime"), []byte(uptimeData), 0644)
	require.NoError(t, err)

	collector := NewHostCollectorWithPath(tmpDir)
	uptime, bootTime, err := collector.getUptime()
	require.NoError(t, err)

	// Check uptime is approximately 100 seconds
	assert.InDelta(t, 100.5*float64(time.Second), float64(uptime), float64(time.Second))

	// Boot time should be ~100 seconds ago
	expectedBootTime := time.Now().Add(-uptime)
	assert.InDelta(t, expectedBootTime.Unix(), bootTime.Unix(), 2)
}

func TestHostCollector_GetMemoryInfo(t *testing.T) {
	// Create temporary proc directory
	tmpDir := t.TempDir()

	// Write test data
	meminfoData := `MemTotal:       8192000 kB
MemFree:        2048000 kB
MemAvailable:   4096000 kB
Buffers:         512000 kB
Cached:         1024000 kB
`
	err := os.WriteFile(filepath.Join(tmpDir, "meminfo"), []byte(meminfoData), 0644)
	require.NoError(t, err)

	collector := NewHostCollectorWithPath(tmpDir)
	memInfo, err := collector.getMemoryInfo()
	require.NoError(t, err)

	assert.Equal(t, uint64(8000), memInfo.TotalMB)
	assert.Equal(t, uint64(2000), memInfo.FreeMB)
	assert.Equal(t, uint64(4000), memInfo.AvailableMB)
	assert.Equal(t, uint64(4000), memInfo.UsedMB)
	assert.InDelta(t, 50.0, memInfo.Usage, 0.1)
}

func TestHostCollector_GetLoadAverage(t *testing.T) {
	// Create temporary proc directory
	tmpDir := t.TempDir()

	// Write test data
	loadavgData := "1.23 2.34 3.45 2/100 12345\n"
	err := os.WriteFile(filepath.Join(tmpDir, "loadavg"), []byte(loadavgData), 0644)
	require.NoError(t, err)

	collector := NewHostCollectorWithPath(tmpDir)
	loadAvg, err := collector.getLoadAverage()
	require.NoError(t, err)

	assert.Equal(t, 1.23, loadAvg.Load1)
	assert.Equal(t, 2.34, loadAvg.Load5)
	assert.Equal(t, 3.45, loadAvg.Load15)
}

func TestHostCollector_GetCPUInfo(t *testing.T) {
	// Create temporary proc directory
	tmpDir := t.TempDir()

	// Write test cpuinfo
	cpuinfoData := `processor	: 0
model name	: Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz
cpu MHz		: 1800.000
cache size	: 8192 KB

processor	: 1
model name	: Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz
cpu MHz		: 1800.000
cache size	: 8192 KB
`
	err := os.WriteFile(filepath.Join(tmpDir, "cpuinfo"), []byte(cpuinfoData), 0644)
	require.NoError(t, err)

	// Write test stat
	statData := "cpu  1000 200 300 5000 100 0 50 0 0 0\n"
	err = os.WriteFile(filepath.Join(tmpDir, "stat"), []byte(statData), 0644)
	require.NoError(t, err)

	collector := NewHostCollectorWithPath(tmpDir)
	cpuInfo, err := collector.getCPUInfo()
	require.NoError(t, err)

	assert.Equal(t, "Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz", cpuInfo.Model)
	assert.Equal(t, 2, cpuInfo.Cores)
}

func TestHostCollector_GetCPUUsage(t *testing.T) {
	// Create temporary proc directory
	tmpDir := t.TempDir()

	// Write test data: user=1000, nice=200, system=300, idle=5000
	// Total = 6500, Used = 1500, Usage = 1500/6500 * 100 = 23.08%
	statData := "cpu  1000 200 300 5000 100 0 50 0 0 0\n"
	err := os.WriteFile(filepath.Join(tmpDir, "stat"), []byte(statData), 0644)
	require.NoError(t, err)

	collector := NewHostCollectorWithPath(tmpDir)
	usage, err := collector.getCPUUsage()
	require.NoError(t, err)

	assert.InDelta(t, 23.08, usage, 0.1)
}

func TestHostCollector_InvalidData(t *testing.T) {
	// Create temporary proc directory
	tmpDir := t.TempDir()

	collector := NewHostCollectorWithPath(tmpDir)

	t.Run("missing_version_file", func(t *testing.T) {
		_, err := collector.getKernelVersion()
		assert.Error(t, err)
	})

	t.Run("missing_uptime_file", func(t *testing.T) {
		_, _, err := collector.getUptime()
		assert.Error(t, err)
	})

	t.Run("invalid_uptime_format", func(t *testing.T) {
		err := os.WriteFile(filepath.Join(tmpDir, "uptime"), []byte("invalid"), 0644)
		require.NoError(t, err)
		_, _, err = collector.getUptime()
		assert.Error(t, err)
	})

	t.Run("missing_loadavg_file", func(t *testing.T) {
		_, err := collector.getLoadAverage()
		assert.Error(t, err)
	})

	t.Run("invalid_loadavg_format", func(t *testing.T) {
		err := os.WriteFile(filepath.Join(tmpDir, "loadavg"), []byte("invalid"), 0644)
		require.NoError(t, err)
		_, err = collector.getLoadAverage()
		assert.Error(t, err)
	})
}

func TestHostCollector_CollectWithErrors(t *testing.T) {
	// Test Collect method with various error conditions
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	// This should fail because we don't have the required files
	_, err := collector.Collect()
	assert.Error(t, err)
}

func TestHostCollector_GetCPUUsageEdgeCases(t *testing.T) {
	// Test CPU usage calculation edge cases
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	// Test with zero values
	statData := "cpu  0 0 0 0 0 0 0 0 0 0\n"
	err := os.WriteFile(filepath.Join(tmpDir, "stat"), []byte(statData), 0644)
	require.NoError(t, err)

	usage, err := collector.getCPUUsage()
	require.NoError(t, err)
	assert.Equal(t, 0.0, usage)

	// Test with invalid format
	err = os.WriteFile(filepath.Join(tmpDir, "stat"), []byte("invalid"), 0644)
	require.NoError(t, err)

	_, err = collector.getCPUUsage()
	assert.Error(t, err)
}

func TestHostCollector_GetMemoryInfoEdgeCases(t *testing.T) {
	// Test memory info edge cases
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	// Test with missing meminfo file
	_, err := collector.getMemoryInfo()
	assert.Error(t, err)

	// Test with empty meminfo file
	err = os.WriteFile(filepath.Join(tmpDir, "meminfo"), []byte(""), 0644)
	require.NoError(t, err)

	info, err := collector.getMemoryInfo()
	require.NoError(t, err)
	assert.Equal(t, uint64(0), info.TotalMB)
	assert.Equal(t, uint64(0), info.FreeMB)
	assert.Equal(t, uint64(0), info.AvailableMB)
	assert.Equal(t, uint64(0), info.UsedMB)
	assert.Equal(t, 0.0, info.Usage)
}

func TestHostCollector_GetCPUInfoEdgeCases(t *testing.T) {
	// Test CPU info edge cases
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	// Test with missing cpuinfo file
	_, err := collector.getCPUInfo()
	assert.Error(t, err)

	// Test with empty cpuinfo file
	err = os.WriteFile(filepath.Join(tmpDir, "cpuinfo"), []byte(""), 0644)
	require.NoError(t, err)

	info, err := collector.getCPUInfo()
	require.NoError(t, err)
	assert.Equal(t, "", info.Model)
	assert.Equal(t, 0, info.Cores)
	assert.Equal(t, 0.0, info.Usage)
}

func TestHostCollector_CollectWithPartialData(t *testing.T) {
	// Test collection with some missing files
	tmpDir := t.TempDir()

	// Create only some required files
	versionPath := filepath.Join(tmpDir, "version")
	err := os.WriteFile(versionPath, []byte("Linux version 5.4.0"), 0644)
	require.NoError(t, err)

	uptimePath := filepath.Join(tmpDir, "uptime")
	err = os.WriteFile(uptimePath, []byte("12345.67 9876.54"), 0644)
	require.NoError(t, err)

	// Don't create meminfo, loadavg, or cpuinfo files

	collector := NewHostCollectorWithPath(tmpDir)
	info, err := collector.Collect()

	// Should return partial data with errors for missing files
	assert.Error(t, err)
	if info != nil {
		assert.Equal(t, "Linux version 5.4.0", info.KernelVersion)
		assert.Equal(t, 12345.67, info.Uptime)
	}
}

func TestHostCollector_CollectWithCorruptedData(t *testing.T) {
	// Test collection with corrupted data files
	tmpDir := t.TempDir()

	// Create files with corrupted data
	versionPath := filepath.Join(tmpDir, "version")
	err := os.WriteFile(versionPath, []byte("Linux version 5.4.0"), 0644)
	require.NoError(t, err)

	uptimePath := filepath.Join(tmpDir, "uptime")
	err = os.WriteFile(uptimePath, []byte("invalid uptime"), 0644)
	require.NoError(t, err)

	meminfoPath := filepath.Join(tmpDir, "meminfo")
	err = os.WriteFile(meminfoPath, []byte("invalid meminfo"), 0644)
	require.NoError(t, err)

	loadavgPath := filepath.Join(tmpDir, "loadavg")
	err = os.WriteFile(loadavgPath, []byte("invalid loadavg"), 0644)
	require.NoError(t, err)

	cpuinfoPath := filepath.Join(tmpDir, "cpuinfo")
	err = os.WriteFile(cpuinfoPath, []byte("invalid cpuinfo"), 0644)
	require.NoError(t, err)

	collector := NewHostCollectorWithPath(tmpDir)
	info, err := collector.Collect()

	// Should return partial data with errors for corrupted files
	assert.Error(t, err)
	if info != nil {
		assert.Equal(t, "Linux version 5.4.0", info.KernelVersion)
	}
}

func TestHostCollector_CollectWithEmptyDirectory(t *testing.T) {
	// Test collection with completely empty directory
	tmpDir := t.TempDir()

	collector := NewHostCollectorWithPath(tmpDir)
	_, err := collector.Collect()

	// Should return error for missing files
	assert.Error(t, err)
}

func TestHostCollector_CollectWithPermissionDenied(t *testing.T) {
	// Test collection with permission denied files
	tmpDir := t.TempDir()

	// Create files with no read permission
	versionPath := filepath.Join(tmpDir, "version")
	err := os.WriteFile(versionPath, []byte("Linux version 5.4.0"), 0000) // No read permission
	require.NoError(t, err)

	collector := NewHostCollectorWithPath(tmpDir)
	_, err = collector.Collect()

	// Should return error for permission denied
	assert.Error(t, err)
}
