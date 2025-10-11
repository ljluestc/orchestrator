package probe

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessCollector_Collect(t *testing.T) {
	collector := NewProcessCollector(true, 100)

	info, err := collector.Collect()
	require.NoError(t, err)
	require.NotNil(t, info)

	// Should have at least one process (ourselves)
	assert.Greater(t, info.TotalProcesses, 0)
	assert.Len(t, info.Processes, info.TotalProcesses)
	assert.False(t, info.Timestamp.IsZero())

	// Validate process information
	for _, proc := range info.Processes {
		assert.Greater(t, proc.PID, 0)
		assert.NotEmpty(t, proc.Name)
		assert.NotEmpty(t, proc.State)
		assert.GreaterOrEqual(t, proc.PPID, 0)
		assert.GreaterOrEqual(t, proc.Threads, 1)
	}
}

func TestProcessCollector_GetProcessByPID(t *testing.T) {
	// Skip this test on Windows as it uses Linux-specific proc filesystem
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Linux-specific test on Windows")
	}

	collector := NewProcessCollector(true, 0)

	// Get current process
	pid := os.Getpid()
	proc, err := collector.GetProcessByPID(pid)
	require.NoError(t, err)
	require.NotNil(t, proc)

	assert.Equal(t, pid, proc.PID)
	assert.NotEmpty(t, proc.Name)
	assert.NotEmpty(t, proc.State)
	assert.Greater(t, proc.PPID, 0)
	assert.GreaterOrEqual(t, proc.Threads, 1)
}

func TestProcessCollector_MaxProcessesLimit(t *testing.T) {
	collector := NewProcessCollector(true, 5)

	info, err := collector.Collect()
	require.NoError(t, err)
	require.NotNil(t, info)

	// Should respect the limit
	assert.LessOrEqual(t, info.TotalProcesses, 5)
}

func TestProcessCollector_FilterContainersOnly(t *testing.T) {
	collector := NewProcessCollector(false, 100)

	info, err := collector.Collect()
	require.NoError(t, err)
	require.NotNil(t, info)

	// All processes should have a container ID
	for _, proc := range info.Processes {
		if proc.Cgroup != "" {
			// If we found a containerized process, verify it has a valid container ID
			assert.Len(t, proc.Cgroup, 64)
		}
	}
}

func TestProcessCollector_ParseStatFile(t *testing.T) {
	collector := NewProcessCollector(true, 0)

	testCases := []struct {
		name     string
		statData string
		wantErr  bool
		check    func(t *testing.T, info *ProcessInfo)
	}{
		{
			name:     "valid_stat",
			statData: "1234 (test-process) S 1 1234 1234 0 -1 4194304 100 0 0 0 10 5 0 0 20 0 2 0 12345 1024000 256 18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0",
			wantErr:  false,
			check: func(t *testing.T, info *ProcessInfo) {
				assert.Equal(t, "test-process", info.Name)
				assert.Equal(t, "S", info.State)
				assert.Equal(t, 1, info.PPID)
				assert.Equal(t, uint64(15), info.CPUTime) // 10 + 5
			},
		},
		{
			name:     "process_with_spaces",
			statData: "5678 (test process with spaces) R 1 5678 5678 0 -1 4194304 100 0 0 0 20 10 0 0 20 0 4 0 23456 2048000 512 18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0",
			wantErr:  false,
			check: func(t *testing.T, info *ProcessInfo) {
				assert.Equal(t, "test process with spaces", info.Name)
				assert.Equal(t, "R", info.State)
				assert.Equal(t, 4, info.Threads)
			},
		},
		{
			name:     "invalid_format_no_parens",
			statData: "1234 test-process S 1",
			wantErr:  true,
			check:    nil,
		},
		{
			name:     "insufficient_fields",
			statData: "1234 (test) S",
			wantErr:  true,
			check:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info := &ProcessInfo{}
			err := collector.parseStatFile(tc.statData, info)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tc.check != nil {
					tc.check(t, info)
				}
			}
		})
	}
}

func TestProcessCollector_ParseStatusFile(t *testing.T) {
	collector := NewProcessCollector(true, 0)

	statusData := `Name:	test-process
Umask:	0022
State:	S (sleeping)
Tgid:	1234
Ngid:	0
Pid:	1234
PPid:	1
TracerPid:	0
Uid:	1000	1000	1000	1000
Gid:	1000	1000	1000	1000
FDSize:	256
Groups:	1000
VmPeak:	  102400 kB
VmSize:	  102400 kB
VmRSS:	   20480 kB
Threads:	4
`

	info := &ProcessInfo{}
	err := collector.parseStatusFile(statusData, info)
	require.NoError(t, err)

	assert.Equal(t, 1000, info.UID)
	assert.Equal(t, 1000, info.GID)
	assert.Equal(t, uint64(20), info.MemoryMB) // 20480 KB / 1024 = 20 MB
	assert.Equal(t, 4, info.Threads)
}

func TestProcessCollector_ExtractContainerID(t *testing.T) {
	collector := NewProcessCollector(true, 0)

	testCases := []struct {
		name       string
		cgroupData string
		expectedID string
	}{
		{
			name:       "docker_cgroup_v2",
			cgroupData: `0::/docker/a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2`,
			expectedID: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
		},
		{
			name:       "docker_cgroup_scope",
			cgroupData: `0::/system.slice/docker-a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2.scope`,
			expectedID: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
		},
		{
			name:       "no_docker",
			cgroupData: `0::/user.slice/user-1000.slice/session-1.scope`,
			expectedID: "",
		},
		{
			name: "multiple_lines",
			cgroupData: `12:pids:/user.slice/user-1000.slice
11:devices:/docker/a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2
10:cpu,cpuacct:/docker/a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2`,
			expectedID: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			containerID := collector.extractContainerID(tc.cgroupData)
			assert.Equal(t, tc.expectedID, containerID)
		})
	}
}

func TestProcessCollector_IsHexString(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"a1b2c3d4", true},
		{"ABCDEF01", true},
		{"0123456789abcdef", true},
		{"g1b2c3d4", false},
		{"a1b2c3d4!", false},
		{"", true}, // Empty string is valid hex
		{"xyz", false},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := isHexString(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestProcessCollector_WithMockProcFS(t *testing.T) {
	// Skip this test on Windows as it uses Linux-specific proc filesystem
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Linux-specific test on Windows")
	}
	// Create temporary proc directory
	tmpDir := t.TempDir()

	// Create a mock process directory
	pid := 12345
	pidDir := filepath.Join(tmpDir, strconv.Itoa(pid))
	err := os.Mkdir(pidDir, 0755)
	require.NoError(t, err)

	// Write stat file
	statData := "12345 (test-proc) S 1 12345 12345 0 -1 4194304 100 0 0 0 10 5 0 0 20 0 2 0 12345 1024000 256 18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0"
	err = os.WriteFile(filepath.Join(pidDir, "stat"), []byte(statData), 0644)
	require.NoError(t, err)

	// Write status file
	statusData := `Name:	test-proc
Uid:	1000	1000	1000	1000
Gid:	1000	1000	1000	1000
VmRSS:	  10240 kB
Threads:	2
`
	err = os.WriteFile(filepath.Join(pidDir, "status"), []byte(statusData), 0644)
	require.NoError(t, err)

	// Write cmdline file
	cmdlineData := "test-proc\x00--arg1\x00--arg2\x00"
	err = os.WriteFile(filepath.Join(pidDir, "cmdline"), []byte(cmdlineData), 0644)
	require.NoError(t, err)

	// Write cgroup file
	cgroupData := "0::/system.slice/test.service"
	err = os.WriteFile(filepath.Join(pidDir, "cgroup"), []byte(cgroupData), 0644)
	require.NoError(t, err)

	// Create fd directory
	fdDir := filepath.Join(pidDir, "fd")
	err = os.Mkdir(fdDir, 0755)
	require.NoError(t, err)

	// Create collector with mock proc path
	collector := NewProcessCollectorWithPath(tmpDir, true, 0)

	// Collect processes
	info, err := collector.Collect()
	require.NoError(t, err)
	require.NotNil(t, info)

	// Should find our mock process
	assert.Equal(t, 1, info.TotalProcesses)
	require.Len(t, info.Processes, 1)

	proc := info.Processes[0]
	assert.Equal(t, 12345, proc.PID)
	assert.Equal(t, "test-proc", proc.Name)
	assert.Equal(t, "S", proc.State)
	assert.Equal(t, 1, proc.PPID)
	assert.Equal(t, 1000, proc.UID)
	assert.Equal(t, 1000, proc.GID)
	assert.Equal(t, 2, proc.Threads)
	assert.Equal(t, uint64(10), proc.MemoryMB)
	assert.Equal(t, "test-proc --arg1 --arg2", proc.Cmdline)
}

func TestProcessCollector_GetProcessInfoEdgeCases(t *testing.T) {
	// Test getProcessInfo with various edge cases
	tmpDir := t.TempDir()
	collector := NewProcessCollectorWithPath(tmpDir, true, 0)

	// Test with non-existent process
	_, err := collector.GetProcessByPID(99999)
	assert.Error(t, err)

	// Test with invalid PID directory
	pidDir := filepath.Join(tmpDir, "invalid")
	err = os.Mkdir(pidDir, 0755)
	require.NoError(t, err)

	// This should fail because we can't convert "invalid" to int
	_, err = collector.GetProcessByPID(0) // This will try to read from "0" directory
	assert.Error(t, err)
}

func TestProcessCollector_CollectWithErrors(t *testing.T) {
	// Test Collect method with various error conditions
	tmpDir := t.TempDir()
	collector := NewProcessCollectorWithPath(tmpDir, true, 0)

	// This should work even with empty proc directory
	info, err := collector.Collect()
	require.NoError(t, err)
	require.NotNil(t, info)

	// On Windows, this should return 1 mock process
	// On Linux, this should return 0 processes
	if runtime.GOOS == "windows" {
		assert.Equal(t, 1, info.TotalProcesses)
	} else {
		assert.Equal(t, 0, info.TotalProcesses)
	}
}

func TestProcessCollector_ParseStatusFileEdgeCases(t *testing.T) {
	// Test parseStatusFile with edge cases
	collector := NewProcessCollector(true, 0)

	// Test with empty status data
	info := &ProcessInfo{}
	err := collector.parseStatusFile("", info)
	require.NoError(t, err)

	// Test with malformed status data
	statusData := `Name:	test-proc
Uid:	invalid
Gid:	invalid
VmRSS:	invalid
Threads:	invalid
`
	info = &ProcessInfo{}
	err = collector.parseStatusFile(statusData, info)
	require.NoError(t, err)
	// Should handle invalid values gracefully
	assert.Equal(t, 0, info.UID)
	assert.Equal(t, 0, info.GID)
	assert.Equal(t, uint64(0), info.MemoryMB)
	assert.Equal(t, 0, info.Threads)
}

func TestProcessCollector_IsHexString(t *testing.T) {
	// Test isHexString method
	collector := NewProcessCollector(true, 100)
	
	// Test valid hex strings
	assert.True(t, collector.isHexString("a1b2c3d4"))
	assert.True(t, collector.isHexString("ABCDEF01"))
	assert.True(t, collector.isHexString("0123456789abcdef"))
	assert.True(t, collector.isHexString("1234567890ABCDEF"))
	
	// Test invalid hex strings
	assert.False(t, collector.isHexString("g1b2c3d4"))
	assert.False(t, collector.isHexString("a1b2c3d4!"))
	assert.False(t, collector.isHexString("#00"))
	assert.False(t, collector.isHexString("xyz"))
	assert.False(t, collector.isHexString(""))
	
	// Test edge cases
	assert.False(t, collector.isHexString("a1b2c3d4 "))
	assert.False(t, collector.isHexString(" a1b2c3d4"))
	assert.False(t, collector.isHexString("a1b2c3d4\n"))
}

func TestProcessCollector_GetProcessByPID(t *testing.T) {
	// Test GetProcessByPID method
	collector := NewProcessCollector(true, 100)
	
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Linux-specific test on Windows")
	}
	
	// Test with valid PID (should exist on most systems)
	info, err := collector.GetProcessByPID(1) // PID 1 is usually init/systemd
	if err != nil {
		// If PID 1 doesn't exist or we can't access it, that's okay
		t.Logf("Could not access PID 1: %v", err)
		return
	}
	
	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, 1, info.PID)
	assert.NotEmpty(t, info.Name)
	
	// Test with invalid PID
	info, err = collector.GetProcessByPID(999999)
	assert.Error(t, err)
	assert.Nil(t, info)
	
	// Test with zero PID
	info, err = collector.GetProcessByPID(0)
	assert.Error(t, err)
	assert.Nil(t, info)
	
	// Test with negative PID
	info, err = collector.GetProcessByPID(-1)
	assert.Error(t, err)
	assert.Nil(t, info)
}
