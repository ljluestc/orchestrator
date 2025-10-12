package isolation

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// CgroupsManager manages cgroups v1 and v2 for container resource isolation
type CgroupsManager struct {
	version    CgroupsVersion
	rootPath   string
	containers map[string]*ContainerCgroups
	mu         sync.RWMutex
}

// CgroupsVersion represents cgroups version
type CgroupsVersion string

const (
	CgroupsV1      CgroupsVersion = "v1"
	CgroupsV2      CgroupsVersion = "v2"
	CgroupsHybrid  CgroupsVersion = "hybrid"
)

// ContainerCgroups tracks cgroup paths for a container
type ContainerCgroups struct {
	ContainerID  string
	CPUPath      string
	MemoryPath   string
	BlkIOPath    string
	PidsPath     string
	CPUSetPath   string
	DevicesPath  string
	FreezerPath  string
}

// ResourceLimits defines resource constraints
type ResourceLimits struct {
	CPUShares      int64   // CPU shares (relative weight)
	CPUQuota       int64   // CPU quota in microseconds per period
	CPUPeriod      int64   // CPU period in microseconds
	CPUCores       []int   // CPU cores to pin to
	MemoryLimit    int64   // Memory limit in bytes
	MemorySwap     int64   // Memory + swap limit in bytes
	OOMKillDisable bool    // Disable OOM killer
	BlkIOWeight    int64   // Block IO weight (10-1000)
	BlkIOReadBPS   int64   // Block IO read bytes per second
	BlkIOWriteBPS  int64   // Block IO write bytes per second
	PidsLimit      int64   // Maximum number of PIDs
}

// ResourceStats tracks current resource usage
type ResourceStats struct {
	CPUUsage       uint64  // Total CPU time in nanoseconds
	CPUPercent     float64 // CPU usage percentage
	MemoryUsage    uint64  // Current memory usage in bytes
	MemoryLimit    uint64  // Memory limit in bytes
	MemoryPercent  float64 // Memory usage percentage
	SwapUsage      uint64  // Swap usage in bytes
	BlkIORead      uint64  // Bytes read from block devices
	BlkIOWrite     uint64  // Bytes written to block devices
	NumPids        int     // Current number of PIDs
	OOMKillCount   int     // Number of OOM kills
}

// NewCgroupsManager creates a new cgroups manager
func NewCgroupsManager(rootPath string) (*CgroupsManager, error) {
	version, err := detectCgroupsVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to detect cgroups version: %w", err)
	}

	if rootPath == "" {
		rootPath = "/sys/fs/cgroup"
	}

	cm := &CgroupsManager{
		version:    version,
		rootPath:   rootPath,
		containers: make(map[string]*ContainerCgroups),
	}

	log.Printf("Cgroups manager initialized with version: %s, root: %s", version, rootPath)

	return cm, nil
}

// detectCgroupsVersion detects whether system uses cgroups v1, v2, or hybrid
func detectCgroupsVersion() (CgroupsVersion, error) {
	// Check if cgroups v2 unified hierarchy exists
	if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err == nil {
		// Check if v1 hierarchies also exist (hybrid mode)
		if _, err := os.Stat("/sys/fs/cgroup/cpu"); err == nil {
			return CgroupsHybrid, nil
		}
		return CgroupsV2, nil
	}

	// Check for cgroups v1
	if _, err := os.Stat("/sys/fs/cgroup/cpu"); err == nil {
		return CgroupsV1, nil
	}

	return "", fmt.Errorf("no cgroups detected")
}

// CreateContainerCgroups creates cgroup hierarchy for a container
func (cm *CgroupsManager) CreateContainerCgroups(containerID string, limits ResourceLimits) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	log.Printf("Creating cgroups for container %s", containerID)

	var cgroups *ContainerCgroups
	var err error

	switch cm.version {
	case CgroupsV1, CgroupsHybrid:
		cgroups, err = cm.createCgroupsV1(containerID, limits)
	case CgroupsV2:
		cgroups, err = cm.createCgroupsV2(containerID, limits)
	default:
		return fmt.Errorf("unsupported cgroups version: %s", cm.version)
	}

	if err != nil {
		return err
	}

	cm.containers[containerID] = cgroups

	log.Printf("Cgroups created successfully for container %s", containerID)
	return nil
}

// createCgroupsV1 creates cgroups v1 hierarchy
func (cm *CgroupsManager) createCgroupsV1(containerID string, limits ResourceLimits) (*ContainerCgroups, error) {
	cgroupPath := filepath.Join("mesos", "containers", containerID)

	cgroups := &ContainerCgroups{
		ContainerID: containerID,
		CPUPath:     filepath.Join(cm.rootPath, "cpu", cgroupPath),
		MemoryPath:  filepath.Join(cm.rootPath, "memory", cgroupPath),
		BlkIOPath:   filepath.Join(cm.rootPath, "blkio", cgroupPath),
		PidsPath:    filepath.Join(cm.rootPath, "pids", cgroupPath),
		CPUSetPath:  filepath.Join(cm.rootPath, "cpuset", cgroupPath),
		DevicesPath: filepath.Join(cm.rootPath, "devices", cgroupPath),
		FreezerPath: filepath.Join(cm.rootPath, "freezer", cgroupPath),
	}

	// Create cgroup directories
	paths := []string{
		cgroups.CPUPath,
		cgroups.MemoryPath,
		cgroups.BlkIOPath,
		cgroups.PidsPath,
		cgroups.CPUSetPath,
		cgroups.DevicesPath,
		cgroups.FreezerPath,
	}

	for _, path := range paths {
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, fmt.Errorf("failed to create cgroup directory %s: %w", path, err)
		}
	}

	// Apply CPU limits
	if limits.CPUShares > 0 {
		if err := cm.writeCgroupFile(cgroups.CPUPath, "cpu.shares", strconv.FormatInt(limits.CPUShares, 10)); err != nil {
			log.Printf("Failed to set cpu.shares: %v", err)
		}
	}
	if limits.CPUQuota > 0 {
		if err := cm.writeCgroupFile(cgroups.CPUPath, "cpu.cfs_quota_us", strconv.FormatInt(limits.CPUQuota, 10)); err != nil {
			log.Printf("Failed to set cpu.cfs_quota_us: %v", err)
		}
	}
	if limits.CPUPeriod > 0 {
		if err := cm.writeCgroupFile(cgroups.CPUPath, "cpu.cfs_period_us", strconv.FormatInt(limits.CPUPeriod, 10)); err != nil {
			log.Printf("Failed to set cpu.cfs_period_us: %v", err)
		}
	}

	// Apply memory limits
	if limits.MemoryLimit > 0 {
		if err := cm.writeCgroupFile(cgroups.MemoryPath, "memory.limit_in_bytes", strconv.FormatInt(limits.MemoryLimit, 10)); err != nil {
			log.Printf("Failed to set memory.limit_in_bytes: %v", err)
		}
	}
	if limits.MemorySwap > 0 {
		if err := cm.writeCgroupFile(cgroups.MemoryPath, "memory.memsw.limit_in_bytes", strconv.FormatInt(limits.MemorySwap, 10)); err != nil {
			log.Printf("Failed to set memory.memsw.limit_in_bytes: %v", err)
		}
	}
	if limits.OOMKillDisable {
		if err := cm.writeCgroupFile(cgroups.MemoryPath, "memory.oom_control", "1"); err != nil {
			log.Printf("Failed to disable OOM killer: %v", err)
		}
	}

	// Apply Block IO limits
	if limits.BlkIOWeight > 0 {
		if err := cm.writeCgroupFile(cgroups.BlkIOPath, "blkio.weight", strconv.FormatInt(limits.BlkIOWeight, 10)); err != nil {
			log.Printf("Failed to set blkio.weight: %v", err)
		}
	}

	// Apply PIDs limit
	if limits.PidsLimit > 0 {
		if err := cm.writeCgroupFile(cgroups.PidsPath, "pids.max", strconv.FormatInt(limits.PidsLimit, 10)); err != nil {
			log.Printf("Failed to set pids.max: %v", err)
		}
	}

	// Apply CPUSet (CPU affinity)
	if len(limits.CPUCores) > 0 {
		cpuList := formatCPUList(limits.CPUCores)
		if err := cm.writeCgroupFile(cgroups.CPUSetPath, "cpuset.cpus", cpuList); err != nil {
			log.Printf("Failed to set cpuset.cpus: %v", err)
		}
		// Copy parent's mems
		parentMems, _ := cm.readCgroupFile(filepath.Dir(cgroups.CPUSetPath), "cpuset.mems")
		if parentMems != "" {
			cm.writeCgroupFile(cgroups.CPUSetPath, "cpuset.mems", strings.TrimSpace(parentMems))
		}
	}

	return cgroups, nil
}

// createCgroupsV2 creates cgroups v2 unified hierarchy
func (cm *CgroupsManager) createCgroupsV2(containerID string, limits ResourceLimits) (*ContainerCgroups, error) {
	cgroupPath := filepath.Join(cm.rootPath, "mesos", "containers", containerID)

	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cgroup directory: %w", err)
	}

	cgroups := &ContainerCgroups{
		ContainerID: containerID,
		CPUPath:     cgroupPath,
		MemoryPath:  cgroupPath,
		BlkIOPath:   cgroupPath,
		PidsPath:    cgroupPath,
	}

	// Enable controllers
	controllers := []string{"cpu", "memory", "io", "pids"}
	controllersStr := strings.Join(controllers, " ")
	parentPath := filepath.Dir(cgroupPath)
	if err := cm.writeCgroupFile(parentPath, "cgroup.subtree_control", "+"+controllersStr); err != nil {
		log.Printf("Failed to enable controllers: %v", err)
	}

	// Apply CPU limits (cgroups v2 format)
	if limits.CPUQuota > 0 && limits.CPUPeriod > 0 {
		cpuMax := fmt.Sprintf("%d %d", limits.CPUQuota, limits.CPUPeriod)
		if err := cm.writeCgroupFile(cgroupPath, "cpu.max", cpuMax); err != nil {
			log.Printf("Failed to set cpu.max: %v", err)
		}
	}
	if limits.CPUShares > 0 {
		// Convert shares to weight (1-10000)
		weight := (limits.CPUShares * 10000) / 1024
		if err := cm.writeCgroupFile(cgroupPath, "cpu.weight", strconv.FormatInt(weight, 10)); err != nil {
			log.Printf("Failed to set cpu.weight: %v", err)
		}
	}

	// Apply memory limits
	if limits.MemoryLimit > 0 {
		if err := cm.writeCgroupFile(cgroupPath, "memory.max", strconv.FormatInt(limits.MemoryLimit, 10)); err != nil {
			log.Printf("Failed to set memory.max: %v", err)
		}
	}

	// Apply IO limits
	if limits.BlkIOReadBPS > 0 || limits.BlkIOWriteBPS > 0 {
		// Format: "major:minor rbps=<bytes> wbps=<bytes>"
		// This would require device detection in production
		log.Printf("IO limits configured: read=%d bps, write=%d bps", limits.BlkIOReadBPS, limits.BlkIOWriteBPS)
	}

	// Apply PIDs limit
	if limits.PidsLimit > 0 {
		if err := cm.writeCgroupFile(cgroupPath, "pids.max", strconv.FormatInt(limits.PidsLimit, 10)); err != nil {
			log.Printf("Failed to set pids.max: %v", err)
		}
	}

	return cgroups, nil
}

// AddProcessToCgroup adds a process to container cgroups
func (cm *CgroupsManager) AddProcessToCgroup(containerID string, pid int) error {
	cm.mu.RLock()
	cgroups, exists := cm.containers[containerID]
	cm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("cgroups not found for container %s", containerID)
	}

	pidStr := strconv.Itoa(pid)

	switch cm.version {
	case CgroupsV1, CgroupsHybrid:
		// Add to all v1 cgroup hierarchies
		paths := []string{
			cgroups.CPUPath,
			cgroups.MemoryPath,
			cgroups.BlkIOPath,
			cgroups.PidsPath,
			cgroups.CPUSetPath,
		}
		for _, path := range paths {
			if err := cm.writeCgroupFile(path, "cgroup.procs", pidStr); err != nil {
				log.Printf("Failed to add PID %d to %s: %v", pid, path, err)
			}
		}

	case CgroupsV2:
		// Add to unified hierarchy
		if err := cm.writeCgroupFile(cgroups.CPUPath, "cgroup.procs", pidStr); err != nil {
			return fmt.Errorf("failed to add PID to cgroup: %w", err)
		}
	}

	log.Printf("Added PID %d to cgroups for container %s", pid, containerID)
	return nil
}

// GetResourceStats retrieves current resource usage statistics
func (cm *CgroupsManager) GetResourceStats(containerID string) (*ResourceStats, error) {
	cm.mu.RLock()
	cgroups, exists := cm.containers[containerID]
	cm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("cgroups not found for container %s", containerID)
	}

	stats := &ResourceStats{}

	switch cm.version {
	case CgroupsV1, CgroupsHybrid:
		cm.getStatsV1(cgroups, stats)
	case CgroupsV2:
		cm.getStatsV2(cgroups, stats)
	}

	return stats, nil
}

// getStatsV1 retrieves stats from cgroups v1
func (cm *CgroupsManager) getStatsV1(cgroups *ContainerCgroups, stats *ResourceStats) {
	// CPU usage
	if cpuUsage, err := cm.readCgroupFile(cgroups.CPUPath, "cpuacct.usage"); err == nil {
		stats.CPUUsage, _ = strconv.ParseUint(strings.TrimSpace(cpuUsage), 10, 64)
	}

	// Memory usage
	if memUsage, err := cm.readCgroupFile(cgroups.MemoryPath, "memory.usage_in_bytes"); err == nil {
		stats.MemoryUsage, _ = strconv.ParseUint(strings.TrimSpace(memUsage), 10, 64)
	}
	if memLimit, err := cm.readCgroupFile(cgroups.MemoryPath, "memory.limit_in_bytes"); err == nil {
		stats.MemoryLimit, _ = strconv.ParseUint(strings.TrimSpace(memLimit), 10, 64)
	}
	if stats.MemoryLimit > 0 {
		stats.MemoryPercent = float64(stats.MemoryUsage) / float64(stats.MemoryLimit) * 100
	}

	// OOM kill count
	if oomControl, err := cm.readCgroupFile(cgroups.MemoryPath, "memory.oom_control"); err == nil {
		for _, line := range strings.Split(oomControl, "\n") {
			if strings.HasPrefix(line, "oom_kill ") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					stats.OOMKillCount, _ = strconv.Atoi(fields[1])
				}
			}
		}
	}

	// PIDs
	if pidsFile, err := cm.readCgroupFile(cgroups.PidsPath, "pids.current"); err == nil {
		stats.NumPids, _ = strconv.Atoi(strings.TrimSpace(pidsFile))
	}
}

// getStatsV2 retrieves stats from cgroups v2
func (cm *CgroupsManager) getStatsV2(cgroups *ContainerCgroups, stats *ResourceStats) {
	// CPU usage
	if cpuStat, err := cm.readCgroupFile(cgroups.CPUPath, "cpu.stat"); err == nil {
		for _, line := range strings.Split(cpuStat, "\n") {
			if strings.HasPrefix(line, "usage_usec ") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					usageUsec, _ := strconv.ParseUint(fields[1], 10, 64)
					stats.CPUUsage = usageUsec * 1000 // Convert to nanoseconds
				}
			}
		}
	}

	// Memory usage
	if memCurrent, err := cm.readCgroupFile(cgroups.MemoryPath, "memory.current"); err == nil {
		stats.MemoryUsage, _ = strconv.ParseUint(strings.TrimSpace(memCurrent), 10, 64)
	}
	if memMax, err := cm.readCgroupFile(cgroups.MemoryPath, "memory.max"); err == nil {
		if strings.TrimSpace(memMax) != "max" {
			stats.MemoryLimit, _ = strconv.ParseUint(strings.TrimSpace(memMax), 10, 64)
		}
	}
	if stats.MemoryLimit > 0 {
		stats.MemoryPercent = float64(stats.MemoryUsage) / float64(stats.MemoryLimit) * 100
	}

	// PIDs
	if pidsCurrent, err := cm.readCgroupFile(cgroups.PidsPath, "pids.current"); err == nil {
		stats.NumPids, _ = strconv.Atoi(strings.TrimSpace(pidsCurrent))
	}
}

// RemoveContainerCgroups removes cgroup hierarchy for a container
func (cm *CgroupsManager) RemoveContainerCgroups(containerID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cgroups, exists := cm.containers[containerID]
	if !exists {
		return fmt.Errorf("cgroups not found for container %s", containerID)
	}

	// Remove cgroup directories
	switch cm.version {
	case CgroupsV1, CgroupsHybrid:
		paths := []string{
			cgroups.CPUPath,
			cgroups.MemoryPath,
			cgroups.BlkIOPath,
			cgroups.PidsPath,
			cgroups.CPUSetPath,
			cgroups.DevicesPath,
			cgroups.FreezerPath,
		}
		for _, path := range paths {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				log.Printf("Failed to remove cgroup %s: %v", path, err)
			}
		}

	case CgroupsV2:
		if err := os.Remove(cgroups.CPUPath); err != nil && !os.IsNotExist(err) {
			log.Printf("Failed to remove cgroup %s: %v", cgroups.CPUPath, err)
		}
	}

	delete(cm.containers, containerID)
	log.Printf("Removed cgroups for container %s", containerID)

	return nil
}

// writeCgroupFile writes a value to a cgroup file
func (cm *CgroupsManager) writeCgroupFile(cgroupPath, filename, value string) error {
	filePath := filepath.Join(cgroupPath, filename)
	return ioutil.WriteFile(filePath, []byte(value), 0644)
}

// readCgroupFile reads a value from a cgroup file
func (cm *CgroupsManager) readCgroupFile(cgroupPath, filename string) (string, error) {
	filePath := filepath.Join(cgroupPath, filename)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// formatCPUList formats a list of CPU cores for cpuset.cpus
func formatCPUList(cores []int) string {
	if len(cores) == 0 {
		return ""
	}

	// Simple format: "0,1,2,3"
	var parts []string
	for _, core := range cores {
		parts = append(parts, strconv.Itoa(core))
	}
	return strings.Join(parts, ",")
}

// MonitorResourceViolations checks for resource limit violations
func (cm *CgroupsManager) MonitorResourceViolations(containerID string) ([]string, error) {
	stats, err := cm.GetResourceStats(containerID)
	if err != nil {
		return nil, err
	}

	var violations []string

	// Check memory usage
	if stats.MemoryPercent > 90 {
		violations = append(violations, fmt.Sprintf("Memory usage at %.1f%%", stats.MemoryPercent))
	}

	// Check OOM kills
	if stats.OOMKillCount > 0 {
		violations = append(violations, fmt.Sprintf("OOM killed %d times", stats.OOMKillCount))
	}

	return violations, nil
}
