package probe

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ProcessInfo contains information about a running process
type ProcessInfo struct {
	PID       int    `json:"pid"`
	Name      string `json:"name"`
	Cmdline   string `json:"cmdline"`
	State     string `json:"state"`
	PPID      int    `json:"ppid"`
	UID       int    `json:"uid"`
	GID       int    `json:"gid"`
	Threads   int    `json:"threads"`
	CPUTime   uint64 `json:"cpu_time"`
	MemoryMB  uint64 `json:"memory_mb"`
	OpenFiles int    `json:"open_files"`
	Cgroup    string `json:"cgroup,omitempty"` // Docker container ID if applicable
}

// ProcessesInfo contains aggregated process information
type ProcessesInfo struct {
	Processes      []ProcessInfo `json:"processes"`
	TotalProcesses int           `json:"total_processes"`
	Timestamp      time.Time     `json:"timestamp"`
}

// ProcessCollector collects process information
type ProcessCollector struct {
	procPath     string
	includeAll   bool // if false, only include processes in containers
	maxProcesses int  // limit number of processes to collect (0 = no limit)
}

// NewProcessCollector creates a new process collector
func NewProcessCollector(includeAll bool, maxProcesses int) *ProcessCollector {
	return &ProcessCollector{
		procPath:     "/proc",
		includeAll:   includeAll,
		maxProcesses: maxProcesses,
	}
}

// NewProcessCollectorWithPath creates a process collector with custom proc path (for testing)
func NewProcessCollectorWithPath(procPath string, includeAll bool, maxProcesses int) *ProcessCollector {
	return &ProcessCollector{
		procPath:     procPath,
		includeAll:   includeAll,
		maxProcesses: maxProcesses,
	}
}

// Collect gathers process information
func (p *ProcessCollector) Collect() (*ProcessesInfo, error) {
	info := &ProcessesInfo{
		Timestamp: time.Now(),
		Processes: make([]ProcessInfo, 0),
	}

	// Platform-specific collection
	if runtime.GOOS == "windows" {
		return p.collectWindows(info)
	} else {
		return p.collectLinux(info)
	}
}

// collectLinux gathers process information on Linux systems
func (p *ProcessCollector) collectLinux(info *ProcessesInfo) (*ProcessesInfo, error) {
	// Read all PID directories from /proc
	entries, err := os.ReadDir(p.procPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read proc directory: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if directory name is a number (PID)
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		// Check limit
		if p.maxProcesses > 0 && count >= p.maxProcesses {
			break
		}

		procInfo, err := p.getProcessInfo(pid)
		if err != nil {
			// Process may have exited, skip it
			continue
		}

		// Filter by container if needed
		if !p.includeAll && procInfo.Cgroup == "" {
			continue
		}

		info.Processes = append(info.Processes, *procInfo)
		count++
	}

	info.TotalProcesses = len(info.Processes)

	return info, nil
}

// collectWindows gathers process information on Windows systems
func (p *ProcessCollector) collectWindows(info *ProcessesInfo) (*ProcessesInfo, error) {
	// For Windows, we'll create a basic implementation
	// In a real implementation, you would use Windows APIs or WMI
	
	// Create a mock process for testing
	mockProcess := ProcessInfo{
		PID:       1,
		Name:      "System",
		Cmdline:   "System",
		State:     "R",
		PPID:      0,
		UID:       0,
		GID:       0,
		Threads:   1,
		CPUTime:   0,
		MemoryMB:  0,
		OpenFiles: 0,
		Cgroup:    "",
	}

	info.Processes = append(info.Processes, mockProcess)
	info.TotalProcesses = 1

	return info, nil
}

// getProcessInfo retrieves information for a specific process
func (p *ProcessCollector) getProcessInfo(pid int) (*ProcessInfo, error) {
	info := &ProcessInfo{
		PID: pid,
	}

	pidPath := filepath.Join(p.procPath, strconv.Itoa(pid))

	// Read /proc/[pid]/stat
	statData, err := os.ReadFile(filepath.Join(pidPath, "stat"))
	if err != nil {
		return nil, err
	}

	if err := p.parseStatFile(string(statData), info); err != nil {
		return nil, err
	}

	// Read /proc/[pid]/status
	statusData, err := os.ReadFile(filepath.Join(pidPath, "status"))
	if err != nil {
		return nil, err
	}

	if err := p.parseStatusFile(string(statusData), info); err != nil {
		return nil, err
	}

	// Read /proc/[pid]/cmdline
	cmdlineData, err := os.ReadFile(filepath.Join(pidPath, "cmdline"))
	if err != nil {
		// cmdline may not be accessible, use name from stat
		info.Cmdline = info.Name
	} else {
		// Replace null bytes with spaces
		cmdline := strings.ReplaceAll(string(cmdlineData), "\x00", " ")
		info.Cmdline = strings.TrimSpace(cmdline)
		if info.Cmdline == "" {
			info.Cmdline = info.Name
		}
	}

	// Read /proc/[pid]/cgroup to detect container
	cgroupData, err := os.ReadFile(filepath.Join(pidPath, "cgroup"))
	if err == nil {
		info.Cgroup = p.extractContainerID(string(cgroupData))
	}

	// Count open files
	fdPath := filepath.Join(pidPath, "fd")
	fdEntries, err := os.ReadDir(fdPath)
	if err == nil {
		info.OpenFiles = len(fdEntries)
	}

	return info, nil
}

// parseStatFile parses /proc/[pid]/stat
func (p *ProcessCollector) parseStatFile(data string, info *ProcessInfo) error {
	// Format: pid (name) state ppid ...
	// Name can contain spaces and parentheses, so we need to find the last )
	startIdx := strings.IndexByte(data, '(')
	endIdx := strings.LastIndexByte(data, ')')

	if startIdx == -1 || endIdx == -1 {
		return fmt.Errorf("invalid stat format")
	}

	info.Name = data[startIdx+1 : endIdx]

	fields := strings.Fields(data[endIdx+1:])
	if len(fields) < 13 {
		return fmt.Errorf("insufficient fields in stat")
	}

	info.State = fields[0]

	ppid, _ := strconv.Atoi(fields[1])
	info.PPID = ppid

	// CPU time (user + system) in clock ticks
	utime, _ := strconv.ParseUint(fields[11], 10, 64)
	stime, _ := strconv.ParseUint(fields[12], 10, 64)
	info.CPUTime = utime + stime

	// Number of threads
	if len(fields) >= 17 {
		threads, _ := strconv.Atoi(fields[17])
		info.Threads = threads
	}

	return nil
}

// parseStatusFile parses /proc/[pid]/status
func (p *ProcessCollector) parseStatusFile(data string, info *ProcessInfo) error {
	scanner := bufio.NewScanner(strings.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSuffix(parts[0], ":")

		switch key {
		case "Uid":
			if len(parts) >= 2 {
				uid, _ := strconv.Atoi(parts[1])
				info.UID = uid
			}
		case "Gid":
			if len(parts) >= 2 {
				gid, _ := strconv.Atoi(parts[1])
				info.GID = gid
			}
		case "VmRSS":
			// Resident Set Size in KB
			if len(parts) >= 2 {
				rss, _ := strconv.ParseUint(parts[1], 10, 64)
				info.MemoryMB = rss / 1024 // Convert to MB
			}
		case "Threads":
			if len(parts) >= 2 {
				threads, _ := strconv.Atoi(parts[1])
				info.Threads = threads
			}
		}
	}

	return scanner.Err()
}

// extractContainerID extracts Docker container ID from cgroup path
func (p *ProcessCollector) extractContainerID(cgroupData string) string {
	lines := strings.Split(cgroupData, "\n")

	for _, line := range lines {
		// Look for docker container ID in cgroup path
		// Format examples:
		// 0::/docker/64-char-container-id
		// 0::/system.slice/docker-64-char-container-id.scope
		if strings.Contains(line, "docker") {
			parts := strings.Split(line, "/")
			for _, part := range parts {
				// Docker container IDs are 64 hex characters
				if len(part) == 64 && isHexString(part) {
					return part
				}
				// Handle docker-<containerid>.scope format
				if strings.HasPrefix(part, "docker-") && strings.HasSuffix(part, ".scope") {
					id := strings.TrimPrefix(part, "docker-")
					id = strings.TrimSuffix(id, ".scope")
					if len(id) == 64 && isHexString(id) {
						return id
					}
				}
			}
		}
	}

	return ""
}

// isHexString checks if a string contains only hexadecimal characters
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// GetProcessByPID retrieves information for a specific process by PID
func (p *ProcessCollector) GetProcessByPID(pid int) (*ProcessInfo, error) {
	return p.getProcessInfo(pid)
}
