package probe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostCollector_GetLoadAverageError(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	// Test missing loadavg file
	_, err := collector.getLoadAverage()
	assert.Error(t, err)
}

func TestHostCollector_GetLoadAverageInvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	// Create loadavg file with invalid format
	loadavgPath := filepath.Join(tmpDir, "loadavg")
	err := os.WriteFile(loadavgPath, []byte("invalid"), 0644)
	require.NoError(t, err)

	_, loadErr := collector.getLoadAverage()
	assert.Error(t, loadErr)
	assert.Contains(t, loadErr.Error(), "invalid loadavg format")
}

func TestHostCollector_GetLoadAverageInvalidFloats(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	tests := []struct {
		name    string
		content string
		errMsg  string
	}{
		{
			name:    "Invalid first field",
			content: "abc 0.5 0.3 1/100 1234",
			errMsg:  "invalid syntax",
		},
		{
			name:    "Invalid second field",
			content: "0.1 xyz 0.3 1/100 1234",
			errMsg:  "invalid syntax",
		},
		{
			name:    "Invalid third field",
			content: "0.1 0.5 bad 1/100 1234",
			errMsg:  "invalid syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loadavgPath := filepath.Join(tmpDir, "loadavg")
			err := os.WriteFile(loadavgPath, []byte(tt.content), 0644)
			require.NoError(t, err)

			_, loadErr := collector.getLoadAverage()
			assert.Error(t, loadErr)
			assert.Contains(t, loadErr.Error(), tt.errMsg)
		})
	}
}

func TestHostCollector_GetMemoryInfoError(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	// Test missing meminfo file
	_, err := collector.getMemoryInfo()
	assert.Error(t, err)
}

func TestHostCollector_GetCPUUsageError(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	// Test missing stat file
	_, err := collector.getCPUUsage()
	assert.Error(t, err)
}

func TestHostCollector_GetCPUUsageInvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	// Create stat file with invalid format (empty file)
	statPath := filepath.Join(tmpDir, "stat")
	err := os.WriteFile(statPath, []byte(""), 0644)
	require.NoError(t, err)

	_, cpuErr := collector.getCPUUsage()
	assert.Error(t, cpuErr)
	assert.Contains(t, cpuErr.Error(), "invalid")
}

func TestHostCollector_GetCPUUsageInvalidCPULine(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	// Create stat file with invalid CPU line
	statPath := filepath.Join(tmpDir, "stat")
	err := os.WriteFile(statPath, []byte("not_cpu 100 200 300"), 0644)
	require.NoError(t, err)

	_, cpuErr := collector.getCPUUsage()
	assert.Error(t, cpuErr)
	assert.Contains(t, cpuErr.Error(), "invalid CPU line format")
}

func TestHostCollector_GetCPUUsageInsufficientFields(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	// Create stat file with insufficient fields
	statPath := filepath.Join(tmpDir, "stat")
	err := os.WriteFile(statPath, []byte("cpu 100 200"), 0644)
	require.NoError(t, err)

	_, cpuErr := collector.getCPUUsage()
	assert.Error(t, cpuErr)
	assert.Contains(t, cpuErr.Error(), "invalid CPU line format")
}

func TestHostCollector_GetCPUInfoError(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	// Test missing cpuinfo file
	_, err := collector.getCPUInfo()
	assert.Error(t, err)
}

func TestHostCollector_GetKernelVersionError(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	// Test missing version file
	_, err := collector.getKernelVersion()
	assert.Error(t, err)
}

func TestHostCollector_GetUptimeError(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	// Test missing uptime file
	_, _, err := collector.getUptime()
	assert.Error(t, err)
}

func TestHostCollector_GetUptimeInvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	// Create uptime file with invalid format
	uptimePath := filepath.Join(tmpDir, "uptime")
	err := os.WriteFile(uptimePath, []byte(""), 0644)
	require.NoError(t, err)

	_, _, uptimeErr := collector.getUptime()
	assert.Error(t, uptimeErr)
	assert.Contains(t, uptimeErr.Error(), "invalid uptime format")
}

func TestHostCollector_GetUptimeInvalidNumber(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	// Create uptime file with invalid number
	uptimePath := filepath.Join(tmpDir, "uptime")
	err := os.WriteFile(uptimePath, []byte("not_a_number 1234"), 0644)
	require.NoError(t, err)

	_, _, uptimeErr := collector.getUptime()
	assert.Error(t, uptimeErr)
}

func TestHostCollector_GetCPUUsageZeroTotal(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	// Create stat file with zero values
	statPath := filepath.Join(tmpDir, "stat")
	err := os.WriteFile(statPath, []byte("cpu 0 0 0 0"), 0644)
	require.NoError(t, err)

	usage, err := collector.getCPUUsage()
	assert.NoError(t, err)
	assert.Equal(t, 0.0, usage)
}

func TestHostCollector_GetMemoryInfoZeroTotal(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	// Create meminfo with zero total
	meminfoPath := filepath.Join(tmpDir, "meminfo")
	content := `MemTotal:           0 kB
MemFree:            0 kB
MemAvailable:       0 kB
`
	err := os.WriteFile(meminfoPath, []byte(content), 0644)
	require.NoError(t, err)

	info, err := collector.getMemoryInfo()
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), info.TotalMB)
	assert.Equal(t, 0.0, info.Usage)
}

func TestHostCollector_GetMemoryInfoWithoutAvailable(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	// Create meminfo without MemAvailable field
	meminfoPath := filepath.Join(tmpDir, "meminfo")
	content := `MemTotal:        16384000 kB
MemFree:          8192000 kB
`
	err := os.WriteFile(meminfoPath, []byte(content), 0644)
	require.NoError(t, err)

	info, err := collector.getMemoryInfo()
	assert.NoError(t, err)
	assert.Equal(t, uint64(16000), info.TotalMB)
	assert.Equal(t, uint64(8000), info.FreeMB)
	assert.Equal(t, uint64(0), info.AvailableMB)
	// Should use FreeMB instead of AvailableMB for calculation
	assert.Equal(t, uint64(8000), info.UsedMB)
}

func TestHostCollector_CollectLinuxError(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewHostCollectorWithPath(tmpDir)

	info := &HostInfo{}

	// Should fail because kernel version file doesn't exist
	_, err := collector.collectLinux(info)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get kernel version")
}
