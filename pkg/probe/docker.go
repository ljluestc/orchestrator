package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// ContainerInfo contains information about a Docker container
type ContainerInfo struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	ImageID string            `json:"image_id"`
	Status  string            `json:"status"`
	State   string            `json:"state"`
	Created time.Time         `json:"created"`
	Ports   []PortMapping     `json:"ports"`
	Labels  map[string]string `json:"labels"`
	Stats   *ContainerStats   `json:"stats,omitempty"`
}

// PortMapping contains port mapping information
type PortMapping struct {
	PrivatePort uint16 `json:"private_port"`
	PublicPort  uint16 `json:"public_port"`
	Type        string `json:"type"`
	IP          string `json:"ip"`
}

// ContainerStats contains container resource usage statistics
type ContainerStats struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsageMB uint64  `json:"memory_usage_mb"`
	MemoryLimitMB uint64  `json:"memory_limit_mb"`
	MemoryPercent float64 `json:"memory_percent"`
	NetworkRxMB   float64 `json:"network_rx_mb"`
	NetworkTxMB   float64 `json:"network_tx_mb"`
}

// DockerInfo contains aggregated Docker information
type DockerInfo struct {
	Containers        []ContainerInfo `json:"containers"`
	TotalContainers   int             `json:"total_containers"`
	RunningContainers int             `json:"running_containers"`
	PausedContainers  int             `json:"paused_containers"`
	StoppedContainers int             `json:"stopped_containers"`
	Images            int             `json:"images"`
	DockerVersion     string          `json:"docker_version"`
	Timestamp         time.Time       `json:"timestamp"`
}

// DockerCollector collects Docker container information
type DockerCollector struct {
	client       *client.Client
	collectStats bool
}

// NewDockerCollector creates a new Docker collector
func NewDockerCollector(collectStats bool) (*DockerCollector, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &DockerCollector{
		client:       cli,
		collectStats: collectStats,
	}, nil
}

// NewDockerCollectorWithClient creates a Docker collector with custom client (for testing)
func NewDockerCollectorWithClient(cli *client.Client, collectStats bool) *DockerCollector {
	return &DockerCollector{
		client:       cli,
		collectStats: collectStats,
	}
}

// Collect gathers Docker container information
func (d *DockerCollector) Collect(ctx context.Context) (*DockerInfo, error) {
	if d.client == nil {
		return nil, fmt.Errorf("Docker client is nil")
	}

	info := &DockerInfo{
		Timestamp: time.Now(),
	}

	// Get Docker version
	version, err := d.client.ServerVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker version: %w", err)
	}
	info.DockerVersion = version.Version

	// List all containers
	containers, err := d.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	info.TotalContainers = len(containers)
	info.Containers = make([]ContainerInfo, 0, len(containers))

	for _, c := range containers {
		containerInfo := ContainerInfo{
			ID:      c.ID,
			Image:   c.Image,
			ImageID: c.ImageID,
			Status:  c.Status,
			State:   c.State,
			Created: time.Unix(c.Created, 0),
			Labels:  c.Labels,
		}

		// Extract container name (remove leading /)
		if len(c.Names) > 0 {
			containerInfo.Name = c.Names[0]
			if len(containerInfo.Name) > 0 && containerInfo.Name[0] == '/' {
				containerInfo.Name = containerInfo.Name[1:]
			}
		}

		// Extract port mappings
		containerInfo.Ports = make([]PortMapping, 0, len(c.Ports))
		for _, port := range c.Ports {
			containerInfo.Ports = append(containerInfo.Ports, PortMapping{
				PrivatePort: port.PrivatePort,
				PublicPort:  port.PublicPort,
				Type:        port.Type,
				IP:          port.IP,
			})
		}

		// Count containers by state
		switch c.State {
		case "running":
			info.RunningContainers++
		case "paused":
			info.PausedContainers++
		case "exited", "dead":
			info.StoppedContainers++
		}

		// Collect container stats if enabled
		if d.collectStats && c.State == "running" {
			stats, err := d.getContainerStats(ctx, c.ID)
			if err != nil {
				// Log error but don't fail the entire collection
				containerInfo.Stats = nil
			} else {
				containerInfo.Stats = stats
			}
		}

		info.Containers = append(info.Containers, containerInfo)
	}

	// Get image count
	images, err := d.client.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}
	info.Images = len(images)

	return info, nil
}

// getContainerStats retrieves resource usage statistics for a container
func (d *DockerCollector) getContainerStats(ctx context.Context, containerID string) (*ContainerStats, error) {
	// Use a timeout context for stats collection
	statsCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	stats, err := d.client.ContainerStats(statsCtx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer stats.Body.Close()

	var containerStats types.StatsJSON
	if err := json.NewDecoder(stats.Body).Decode(&containerStats); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	result := &ContainerStats{
		MemoryUsageMB: containerStats.MemoryStats.Usage / 1024 / 1024,
		MemoryLimitMB: containerStats.MemoryStats.Limit / 1024 / 1024,
	}

	// Calculate CPU percentage
	cpuDelta := float64(containerStats.CPUStats.CPUUsage.TotalUsage - containerStats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(containerStats.CPUStats.SystemUsage - containerStats.PreCPUStats.SystemUsage)
	onlineCPUs := float64(containerStats.CPUStats.OnlineCPUs)

	if onlineCPUs == 0 {
		onlineCPUs = float64(len(containerStats.CPUStats.CPUUsage.PercpuUsage))
	}

	if systemDelta > 0 && cpuDelta > 0 {
		result.CPUPercent = (cpuDelta / systemDelta) * onlineCPUs * 100.0
	}

	// Calculate memory percentage
	if result.MemoryLimitMB > 0 {
		result.MemoryPercent = float64(result.MemoryUsageMB) / float64(result.MemoryLimitMB) * 100.0
	}

	// Calculate network I/O
	var rxBytes, txBytes uint64
	for _, network := range containerStats.Networks {
		rxBytes += network.RxBytes
		txBytes += network.TxBytes
	}
	result.NetworkRxMB = float64(rxBytes) / 1024.0 / 1024.0
	result.NetworkTxMB = float64(txBytes) / 1024.0 / 1024.0

	return result, nil
}

// Close closes the Docker client connection
func (d *DockerCollector) Close() error {
	if d.client != nil {
		return d.client.Close()
	}
	return nil
}
