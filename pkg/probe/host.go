package probe

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// HostInfo contains collected host system information
type HostInfo struct {
	Hostname      string        `json:"hostname"`
	KernelVersion string        `json:"kernel_version"`
	Uptime        time.Duration `json:"uptime"`
	BootTime      time.Time     `json:"boot_time"`
	CPUInfo       CPUInfo       `json:"cpu_info"`
	MemoryInfo    MemoryInfo    `json:"memory_info"`
	LoadAverage   LoadAverage   `json:"load_average"`
	Timestamp     time.Time     `json:"timestamp"`
}

// CPUInfo contains CPU-related information
type CPUInfo struct {
	Model string  `json:"model"`
	Cores int     `json:"cores"`
	Usage float64 `json:"usage"` // percentage
}

// MemoryInfo contains memory-related information
type MemoryInfo struct {
	TotalMB     uint64  `json:"total_mb"`
	FreeMB      uint64  `json:"free_mb"`
	AvailableMB uint64  `json:"available_mb"`
	UsedMB      uint64  `json:"used_mb"`
	Usage       float64 `json:"usage"` // percentage
}

// LoadAverage contains system load averages
type LoadAverage struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

// HostCollector collects host information
type HostCollector struct {
	procPath string
}

// NewHostCollector creates a new host collector
func NewHostCollector() *HostCollector {
	return &HostCollector{
		procPath: "/proc",
	}
}

// NewHostCollectorWithPath creates a host collector with custom proc path (for testing)
func NewHostCollectorWithPath(procPath string) *HostCollector {
	return &HostCollector{
		procPath: procPath,
	}
}

// Collect gathers host information
func (h *HostCollector) Collect() (*HostInfo, error) {
	info := &HostInfo{
		Timestamp: time.Now(),
	}

	var err error

	// Get hostname
	info.Hostname, err = os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	// Get kernel version
	info.KernelVersion, err = h.getKernelVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get kernel version: %w", err)
	}

	// Get uptime and boot time
	info.Uptime, info.BootTime, err = h.getUptime()
	if err != nil {
		return nil, fmt.Errorf("failed to get uptime: %w", err)
	}

	// Get CPU info
	info.CPUInfo, err = h.getCPUInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU info: %w", err)
	}

	// Get memory info
	info.MemoryInfo, err = h.getMemoryInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}

	// Get load average
	info.LoadAverage, err = h.getLoadAverage()
	if err != nil {
		return nil, fmt.Errorf("failed to get load average: %w", err)
	}

	return info, nil
}

// getKernelVersion reads kernel version from /proc/version
func (h *HostCollector) getKernelVersion() (string, error) {
	data, err := os.ReadFile(fmt.Sprintf("%s/version", h.procPath))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// getUptime reads system uptime from /proc/uptime
func (h *HostCollector) getUptime() (time.Duration, time.Time, error) {
	data, err := os.ReadFile(fmt.Sprintf("%s/uptime", h.procPath))
	if err != nil {
		return 0, time.Time{}, err
	}

	parts := strings.Fields(string(data))
	if len(parts) < 1 {
		return 0, time.Time{}, fmt.Errorf("invalid uptime format")
	}

	uptimeSeconds, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, time.Time{}, err
	}

	uptime := time.Duration(uptimeSeconds * float64(time.Second))
	bootTime := time.Now().Add(-uptime)

	return uptime, bootTime, nil
}

// getCPUInfo reads CPU information from /proc/cpuinfo and /proc/stat
func (h *HostCollector) getCPUInfo() (CPUInfo, error) {
	info := CPUInfo{}

	// Read /proc/cpuinfo for model and core count
	file, err := os.Open(fmt.Sprintf("%s/cpuinfo", h.procPath))
	if err != nil {
		return info, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	cores := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			if info.Model == "" {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					info.Model = strings.TrimSpace(parts[1])
				}
			}
		} else if strings.HasPrefix(line, "processor") {
			cores++
		}
	}
	info.Cores = cores

	if err := scanner.Err(); err != nil {
		return info, err
	}

	// Calculate CPU usage from /proc/stat
	usage, err := h.getCPUUsage()
	if err != nil {
		// CPU usage is optional, don't fail on error
		info.Usage = 0
	} else {
		info.Usage = usage
	}

	return info, nil
}

// getCPUUsage calculates CPU usage percentage from /proc/stat
func (h *HostCollector) getCPUUsage() (float64, error) {
	data, err := os.ReadFile(fmt.Sprintf("%s/stat", h.procPath))
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) < 1 {
		return 0, fmt.Errorf("invalid stat format")
	}

	// First line is aggregate CPU stats
	fields := strings.Fields(lines[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, fmt.Errorf("invalid CPU line format")
	}

	user, _ := strconv.ParseUint(fields[1], 10, 64)
	nice, _ := strconv.ParseUint(fields[2], 10, 64)
	system, _ := strconv.ParseUint(fields[3], 10, 64)
	idle, _ := strconv.ParseUint(fields[4], 10, 64)

	total := user + nice + system + idle
	used := user + nice + system

	if total == 0 {
		return 0, nil
	}

	return float64(used) / float64(total) * 100, nil
}

// getMemoryInfo reads memory information from /proc/meminfo
func (h *HostCollector) getMemoryInfo() (MemoryInfo, error) {
	info := MemoryInfo{}

	file, err := os.Open(fmt.Sprintf("%s/meminfo", h.procPath))
	if err != nil {
		return info, err
	}
	defer file.Close()

	memData := make(map[string]uint64)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSuffix(parts[0], ":")
		value, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			continue
		}

		// Convert from KB to MB
		memData[key] = value / 1024
	}

	if err := scanner.Err(); err != nil {
		return info, err
	}

	info.TotalMB = memData["MemTotal"]
	info.FreeMB = memData["MemFree"]
	info.AvailableMB = memData["MemAvailable"]

	if info.AvailableMB > 0 {
		info.UsedMB = info.TotalMB - info.AvailableMB
	} else {
		info.UsedMB = info.TotalMB - info.FreeMB
	}

	if info.TotalMB > 0 {
		info.Usage = float64(info.UsedMB) / float64(info.TotalMB) * 100
	}

	return info, nil
}

// getLoadAverage reads load average from /proc/loadavg
func (h *HostCollector) getLoadAverage() (LoadAverage, error) {
	avg := LoadAverage{}

	data, err := os.ReadFile(fmt.Sprintf("%s/loadavg", h.procPath))
	if err != nil {
		return avg, err
	}

	parts := strings.Fields(string(data))
	if len(parts) < 3 {
		return avg, fmt.Errorf("invalid loadavg format")
	}

	avg.Load1, err = strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return avg, err
	}

	avg.Load5, err = strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return avg, err
	}

	avg.Load15, err = strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return avg, err
	}

	return avg, nil
}
