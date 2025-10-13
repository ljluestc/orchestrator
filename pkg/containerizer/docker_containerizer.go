package containerizer

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// DockerContainerizer implements Docker-based container management
type DockerContainerizer struct {
	client          *client.Client
	imageCache      *ImageCache
	containerStates map[string]*ContainerState
	statesMux       sync.RWMutex
	config          *ContainerizerConfig
}

// ContainerizerConfig holds configuration for the containerizer
type ContainerizerConfig struct {
	DockerHost          string
	ImagePullTimeout    time.Duration
	ContainerStartupMax time.Duration // Target: <5s
	ImageCacheSize      int64         // bytes
	EnableImageCache    bool
	DefaultRegistry     string
	RegistryAuth        map[string]RegistryCredentials
	NetworkMode         string
	CPUShares           int64
	MemoryLimit         int64
	EnableGPU           bool
}

// RegistryCredentials holds Docker registry authentication
type RegistryCredentials struct {
	Username      string
	Password      string
	Email         string
	ServerAddress string
}

// ContainerState tracks container lifecycle state
type ContainerState struct {
	ID            string
	Name          string
	Image         string
	Status        string
	StartTime     time.Time
	StopTime      time.Time
	ExitCode      int
	ResourceUsage ResourceUsage
}

// ResourceUsage tracks container resource consumption
type ResourceUsage struct {
	CPUPercent    float64
	MemoryUsage   uint64
	MemoryLimit   uint64
	NetworkRxBytes uint64
	NetworkTxBytes uint64
	BlockRead     uint64
	BlockWrite    uint64
}

// ImageCache implements intelligent image caching for fast startup
type ImageCache struct {
	images      map[string]*CachedImage
	mu          sync.RWMutex
	maxSize     int64
	currentSize int64
}

// CachedImage represents a cached Docker image
type CachedImage struct {
	ID           string
	RepoTags     []string
	Size         int64
	LastUsed     time.Time
	PullTime     time.Duration
	UseCount     int
	CachedLayers int
}

// NewDockerContainerizer creates a new Docker containerizer
func NewDockerContainerizer(config *ContainerizerConfig) (*DockerContainerizer, error) {
	// Connect to Docker daemon
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker daemon: %w", err)
	}

	dc := &DockerContainerizer{
		client:          cli,
		containerStates: make(map[string]*ContainerState),
		config:          config,
		imageCache: &ImageCache{
			images:  make(map[string]*CachedImage),
			maxSize: config.ImageCacheSize,
		},
	}

	// Initialize image cache
	if config.EnableImageCache {
		if err := dc.initializeImageCache(context.Background()); err != nil {
			log.Printf("Warning: Failed to initialize image cache: %v", err)
		}
	}

	log.Printf("Docker containerizer initialized successfully")
	return dc, nil
}

// initializeImageCache populates the image cache with existing images
func (dc *DockerContainerizer) initializeImageCache(ctx context.Context) error {
	images, err := dc.client.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return err
	}

	dc.imageCache.mu.Lock()
	defer dc.imageCache.mu.Unlock()

	for _, img := range images {
		cached := &CachedImage{
			ID:       img.ID,
			RepoTags: img.RepoTags,
			Size:     img.Size,
			LastUsed: time.Now(),
		}
		dc.imageCache.images[img.ID] = cached
		dc.imageCache.currentSize += img.Size
	}

	log.Printf("Image cache initialized with %d images (%.2f GB)",
		len(images), float64(dc.imageCache.currentSize)/(1024*1024*1024))

	return nil
}

// PullImage pulls a Docker image with caching and optimizations
func (dc *DockerContainerizer) PullImage(ctx context.Context, imageName string) error {
	startTime := time.Now()

	// Check if image exists in cache
	if dc.config.EnableImageCache {
		if dc.imageExists(ctx, imageName) {
			dc.imageCache.mu.Lock()
			if cached, exists := dc.imageCache.images[imageName]; exists {
				cached.LastUsed = time.Now()
				cached.UseCount++
			}
			dc.imageCache.mu.Unlock()
			log.Printf("Image %s found in cache, skipping pull", imageName)
			return nil
		}
	}

	// Set timeout for image pull
	pullCtx := ctx
	if dc.config.ImagePullTimeout > 0 {
		var cancel context.CancelFunc
		pullCtx, cancel = context.WithTimeout(ctx, dc.config.ImagePullTimeout)
		defer cancel()
	}

	log.Printf("Pulling image: %s", imageName)

	// Get registry credentials if configured
	options := types.ImagePullOptions{}
	if creds, exists := dc.config.RegistryAuth[dc.config.DefaultRegistry]; exists {
		authConfig := registry.AuthConfig{
			Username:      creds.Username,
			Password:      creds.Password,
			Email:         creds.Email,
			ServerAddress: creds.ServerAddress,
		}
		encodedAuth, err := encodeAuthToBase64(authConfig)
		if err == nil {
			options.RegistryAuth = encodedAuth
		}
	}

	// Pull image
	reader, err := dc.client.ImagePull(pullCtx, imageName, options)
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Stream pull output
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("error reading pull output: %w", err)
	}

	pullDuration := time.Since(startTime)
	log.Printf("Image %s pulled successfully in %v", imageName, pullDuration)

	// Update cache
	if dc.config.EnableImageCache {
		dc.updateImageCache(ctx, imageName, pullDuration)
	}

	return nil
}

// imageExists checks if an image exists locally
func (dc *DockerContainerizer) imageExists(ctx context.Context, imageName string) bool {
	_, _, err := dc.client.ImageInspectWithRaw(ctx, imageName)
	return err == nil
}

// updateImageCache updates the image cache after a pull
func (dc *DockerContainerizer) updateImageCache(ctx context.Context, imageName string, pullDuration time.Duration) {
	img, _, err := dc.client.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		log.Printf("Failed to inspect image for cache: %v", err)
		return
	}

	dc.imageCache.mu.Lock()
	defer dc.imageCache.mu.Unlock()

	cached := &CachedImage{
		ID:       img.ID,
		RepoTags: img.RepoTags,
		Size:     img.Size,
		LastUsed: time.Now(),
		PullTime: pullDuration,
		UseCount: 1,
	}

	// Enforce cache size limit
	dc.imageCache.currentSize += img.Size
	if dc.imageCache.currentSize > dc.imageCache.maxSize {
		dc.evictLRUImages(img.Size)
	}

	dc.imageCache.images[img.ID] = cached
}

// evictLRUImages evicts least recently used images to free space
func (dc *DockerContainerizer) evictLRUImages(neededSpace int64) {
	// Find LRU images
	type imageLRU struct {
		id       string
		lastUsed time.Time
		size     int64
	}

	var lruList []imageLRU
	for id, cached := range dc.imageCache.images {
		lruList = append(lruList, imageLRU{
			id:       id,
			lastUsed: cached.LastUsed,
			size:     cached.Size,
		})
	}

	// Sort by last used (oldest first)
	// In production, use sort.Slice here

	// Evict oldest images until we have enough space
	freedSpace := int64(0)
	for _, img := range lruList {
		if dc.imageCache.currentSize-freedSpace+neededSpace <= dc.imageCache.maxSize {
			break
		}

		log.Printf("Evicting image %s from cache (LRU)", img.id)
		delete(dc.imageCache.images, img.id)
		freedSpace += img.size
	}

	dc.imageCache.currentSize -= freedSpace
}

// CreateContainer creates a new container with resource constraints
func (dc *DockerContainerizer) CreateContainer(ctx context.Context, config *ContainerConfig) (string, error) {
	// Pull image if needed
	if err := dc.PullImage(ctx, config.Image); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Convert ExposedPorts to nat.PortSet
	exposedPorts := make(nat.PortSet)
	for port := range config.ExposedPorts {
		exposedPorts[nat.Port(port)] = struct{}{}
	}

	// Build container configuration
	containerConfig := &container.Config{
		Image:        config.Image,
		Cmd:          config.Command,
		Env:          config.Environment,
		WorkingDir:   config.WorkingDir,
		ExposedPorts: exposedPorts,
		Labels:       config.Labels,
	}

	// Build host configuration with resource limits
	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			CPUShares:  config.CPUShares,
			Memory:     config.MemoryLimit,
			MemorySwap: config.MemoryLimit, // Disable swap
			NanoCPUs:   config.CPUQuota,
		},
		NetworkMode: container.NetworkMode(dc.config.NetworkMode),
		Binds:       config.Volumes,
	}

	// GPU support
	if dc.config.EnableGPU && config.GPUCount > 0 {
		hostConfig.DeviceRequests = []container.DeviceRequest{
			{
				Count:        int(config.GPUCount),
				Capabilities: [][]string{{"gpu"}},
			},
		}
	}

	// Create container
	resp, err := dc.client.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		&network.NetworkingConfig{},
		nil,
		config.Name,
	)

	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	log.Printf("Container created: %s (ID: %s)", config.Name, resp.ID)

	// Track state
	dc.statesMux.Lock()
	dc.containerStates[resp.ID] = &ContainerState{
		ID:     resp.ID,
		Name:   config.Name,
		Image:  config.Image,
		Status: "created",
	}
	dc.statesMux.Unlock()

	return resp.ID, nil
}

// StartContainer starts a container with startup time tracking
func (dc *DockerContainerizer) StartContainer(ctx context.Context, containerID string) error {
	startTime := time.Now()

	err := dc.client.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	startupDuration := time.Since(startTime)

	// Update state
	dc.statesMux.Lock()
	if state, exists := dc.containerStates[containerID]; exists {
		state.Status = "running"
		state.StartTime = startTime
	}
	dc.statesMux.Unlock()

	log.Printf("Container %s started in %v", containerID[:12], startupDuration)

	// Check if we met the <5s startup target
	if startupDuration > 5*time.Second {
		log.Printf("WARNING: Container startup exceeded 5s target: %v", startupDuration)
	}

	return nil
}

// StopContainer stops a running container
func (dc *DockerContainerizer) StopContainer(ctx context.Context, containerID string, timeout int) error {
	stopTimeout := timeout
	err := dc.client.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &stopTimeout,
	})
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Update state
	dc.statesMux.Lock()
	if state, exists := dc.containerStates[containerID]; exists {
		state.Status = "stopped"
		state.StopTime = time.Now()
	}
	dc.statesMux.Unlock()

	log.Printf("Container %s stopped", containerID[:12])
	return nil
}

// RemoveContainer removes a container
func (dc *DockerContainerizer) RemoveContainer(ctx context.Context, containerID string) error {
	err := dc.client.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	// Remove from state tracking
	dc.statesMux.Lock()
	delete(dc.containerStates, containerID)
	dc.statesMux.Unlock()

	log.Printf("Container %s removed", containerID[:12])
	return nil
}

// ContainerConfig defines container creation parameters
type ContainerConfig struct {
	Name          string
	Image         string
	Command       []string
	Environment   []string
	WorkingDir    string
	ExposedPorts  map[string]struct{}
	Labels        map[string]string
	Volumes       []string
	CPUShares     int64
	CPUQuota      int64
	MemoryLimit   int64
	GPUCount      int64
}

// Helper function to encode auth config (stub)
func encodeAuthToBase64(authConfig registry.AuthConfig) (string, error) {
	// In production, implement proper base64 encoding of auth config
	return "", nil
}
