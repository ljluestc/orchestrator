package marathon

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// AutoScaler implements horizontal pod autoscaling for Marathon applications
type AutoScaler struct {
	client          *MarathonClient
	applications    map[string]*AutoScaleConfig
	mu              sync.RWMutex
	checkInterval   time.Duration
	metricsProvider MetricsProvider
}

// AutoScaleConfig defines autoscaling parameters for an application
type AutoScaleConfig struct {
	AppID            string
	MinInstances     int
	MaxInstances     int
	TargetCPUPercent float64
	TargetMemPercent float64
	ScaleUpPolicy    ScalePolicy
	ScaleDownPolicy  ScalePolicy
	Enabled          bool
	LastScaleTime    time.Time
	ScaleHistory     []ScaleEvent
}

// ScalePolicy defines scaling behavior
type ScalePolicy struct {
	Threshold         float64       // Metric threshold to trigger scaling
	ConsecutivePeriods int          // Number of consecutive periods before scaling
	Cooldown          time.Duration // Minimum time between scale operations
	StepSize          int           // Number of instances to add/remove
	StepPercentage    float64       // Percentage of current instances to add/remove
}

// ScaleEvent records a scaling event
type ScaleEvent struct {
	Timestamp     time.Time
	FromInstances int
	ToInstances   int
	Reason        string
	Metric        string
	MetricValue   float64
}

// MetricsProvider interface for retrieving application metrics
type MetricsProvider interface {
	GetCPUUtilization(appID string) (float64, error)
	GetMemoryUtilization(appID string) (float64, error)
	GetCustomMetric(appID, metricName string) (float64, error)
}

// MarathonClient interface for Marathon API operations
type MarathonClient interface {
	GetApp(appID string) (*Application, error)
	ScaleApp(appID string, instances int) error
	GetAppTasks(appID string) ([]Task, error)
}

// ApplicationMetrics represents metrics for a Marathon application
type ApplicationMetrics struct {
	ID        string
	Instances int
	TasksRunning int
	TasksHealthy int
	TasksUnhealthy int
	CPUUsage  float64
	MemUsage  float64
}

// Task represents a Marathon task
type Task struct {
	ID          string
	AppID       string
	State       string
	HealthState string
	Host        string
	StartedAt   time.Time
}

// NewAutoScaler creates a new autoscaler instance
func NewAutoScaler(client *MarathonClient, metricsProvider MetricsProvider) *AutoScaler {
	return &AutoScaler{
		client:          client,
		applications:    make(map[string]*AutoScaleConfig),
		checkInterval:   30 * time.Second,
		metricsProvider: metricsProvider,
	}
}

// RegisterApp registers an application for autoscaling
func (as *AutoScaler) RegisterApp(config *AutoScaleConfig) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	// Validate config
	if config.MinInstances < 1 {
		return fmt.Errorf("minInstances must be >= 1")
	}
	if config.MaxInstances < config.MinInstances {
		return fmt.Errorf("maxInstances must be >= minInstances")
	}
	if config.TargetCPUPercent <= 0 {
		config.TargetCPUPercent = 70.0
	}

	// Set default scale policies if not configured
	if config.ScaleUpPolicy.Cooldown == 0 {
		config.ScaleUpPolicy.Cooldown = 3 * time.Minute
	}
	if config.ScaleDownPolicy.Cooldown == 0 {
		config.ScaleDownPolicy.Cooldown = 5 * time.Minute
	}
	if config.ScaleUpPolicy.ConsecutivePeriods == 0 {
		config.ScaleUpPolicy.ConsecutivePeriods = 2
	}
	if config.ScaleDownPolicy.ConsecutivePeriods == 0 {
		config.ScaleDownPolicy.ConsecutivePeriods = 3
	}

	config.ScaleHistory = []ScaleEvent{}
	as.applications[config.AppID] = config

	log.Printf("Registered autoscaling for app %s: min=%d, max=%d, targetCPU=%.1f%%",
		config.AppID, config.MinInstances, config.MaxInstances, config.TargetCPUPercent)

	return nil
}

// Start begins the autoscaling loop
func (as *AutoScaler) Start(ctx context.Context) error {
	log.Println("Starting Marathon autoscaler")

	ticker := time.NewTicker(as.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Autoscaler shutting down")
			return nil
		case <-ticker.C:
			as.checkAndScale(ctx)
		}
	}
}

// checkAndScale evaluates all registered applications and scales if needed
func (as *AutoScaler) checkAndScale(ctx context.Context) {
	as.mu.RLock()
	apps := make([]*AutoScaleConfig, 0, len(as.applications))
	for _, config := range as.applications {
		if config.Enabled {
			apps = append(apps, config)
		}
	}
	as.mu.RUnlock()

	for _, config := range apps {
		if err := as.evaluateApp(ctx, config); err != nil {
			log.Printf("Error evaluating app %s: %v", config.AppID, err)
		}
	}
}

// evaluateApp evaluates a single application for scaling
func (as *AutoScaler) evaluateApp(ctx context.Context, config *AutoScaleConfig) error {
	// Get current application state
	app, err := (*as.client).GetApp(config.AppID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	// Get metrics
	cpuUtil, err := as.metricsProvider.GetCPUUtilization(config.AppID)
	if err != nil {
		log.Printf("Failed to get CPU metrics for %s: %v", config.AppID, err)
		cpuUtil = 0
	}

	memUtil, err := as.metricsProvider.GetMemoryUtilization(config.AppID)
	if err != nil {
		log.Printf("Failed to get memory metrics for %s: %v", config.AppID, err)
		memUtil = 0
	}

	log.Printf("App %s: instances=%d, CPU=%.1f%%, Memory=%.1f%%",
		config.AppID, app.Instances, cpuUtil, memUtil)

	// Determine scaling action
	decision := as.makeScalingDecision(config, app, cpuUtil, memUtil)

	if decision.ShouldScale {
		return as.executeScale(ctx, config, app, decision)
	}

	return nil
}

// ScalingDecision represents a scaling decision
type ScalingDecision struct {
	ShouldScale   bool
	Direction     string // "up" or "down"
	TargetCount   int
	Reason        string
	MetricName    string
	MetricValue   float64
}

// makeScalingDecision determines if and how to scale
func (as *AutoScaler) makeScalingDecision(config *AutoScaleConfig, app *Application, cpuUtil, memUtil float64) ScalingDecision {
	decision := ScalingDecision{
		ShouldScale: false,
	}

	currentInstances := app.Instances

	// Check scale-up conditions
	if cpuUtil > config.TargetCPUPercent {
		decision.ShouldScale = true
		decision.Direction = "up"
		decision.MetricName = "cpu"
		decision.MetricValue = cpuUtil
		decision.Reason = fmt.Sprintf("CPU utilization %.1f%% > target %.1f%%", cpuUtil, config.TargetCPUPercent)
	} else if memUtil > config.TargetMemPercent && config.TargetMemPercent > 0 {
		decision.ShouldScale = true
		decision.Direction = "up"
		decision.MetricName = "memory"
		decision.MetricValue = memUtil
		decision.Reason = fmt.Sprintf("Memory utilization %.1f%% > target %.1f%%", memUtil, config.TargetMemPercent)
	}

	// Check scale-down conditions
	if cpuUtil < config.TargetCPUPercent*0.5 && memUtil < config.TargetMemPercent*0.5 {
		decision.ShouldScale = true
		decision.Direction = "down"
		decision.MetricName = "cpu_memory"
		decision.MetricValue = (cpuUtil + memUtil) / 2
		decision.Reason = fmt.Sprintf("Low utilization: CPU=%.1f%%, Memory=%.1f%%", cpuUtil, memUtil)
	}

	// Check cooldown period
	if decision.ShouldScale {
		cooldown := config.ScaleUpPolicy.Cooldown
		if decision.Direction == "down" {
			cooldown = config.ScaleDownPolicy.Cooldown
		}

		if time.Since(config.LastScaleTime) < cooldown {
			decision.ShouldScale = false
			decision.Reason = fmt.Sprintf("Cooldown active (%.0f seconds remaining)",
				(cooldown - time.Since(config.LastScaleTime)).Seconds())
			return decision
		}
	}

	// Calculate target instance count
	if decision.ShouldScale {
		if decision.Direction == "up" {
			stepSize := config.ScaleUpPolicy.StepSize
			if stepSize == 0 {
				stepSize = max(1, int(float64(currentInstances)*0.5)) // 50% increase
			}
			decision.TargetCount = min(currentInstances+stepSize, config.MaxInstances)
		} else {
			stepSize := config.ScaleDownPolicy.StepSize
			if stepSize == 0 {
				stepSize = max(1, int(float64(currentInstances)*0.25)) // 25% decrease
			}
			decision.TargetCount = max(currentInstances-stepSize, config.MinInstances)
		}

		// Don't scale if already at limit
		if decision.TargetCount == currentInstances {
			decision.ShouldScale = false
			decision.Reason = "Already at scale limit"
		}
	}

	return decision
}

// executeScale performs the scaling operation
func (as *AutoScaler) executeScale(ctx context.Context, config *AutoScaleConfig, app *Application, decision ScalingDecision) error {
	log.Printf("Scaling %s from %d to %d instances: %s",
		config.AppID, app.Instances, decision.TargetCount, decision.Reason)

	// Execute scale operation
	err := (*as.client).ScaleApp(config.AppID, decision.TargetCount)
	if err != nil {
		return fmt.Errorf("failed to scale app: %w", err)
	}

	// Record scale event
	event := ScaleEvent{
		Timestamp:     time.Now(),
		FromInstances: app.Instances,
		ToInstances:   decision.TargetCount,
		Reason:        decision.Reason,
		Metric:        decision.MetricName,
		MetricValue:   decision.MetricValue,
	}

	as.mu.Lock()
	config.LastScaleTime = time.Now()
	config.ScaleHistory = append(config.ScaleHistory, event)
	// Keep only last 50 events
	if len(config.ScaleHistory) > 50 {
		config.ScaleHistory = config.ScaleHistory[len(config.ScaleHistory)-50:]
	}
	as.mu.Unlock()

	log.Printf("Successfully scaled %s to %d instances", config.AppID, decision.TargetCount)

	return nil
}

// GetScaleHistory returns scaling history for an application
func (as *AutoScaler) GetScaleHistory(appID string) []ScaleEvent {
	as.mu.RLock()
	defer as.mu.RUnlock()

	if config, exists := as.applications[appID]; exists {
		return config.ScaleHistory
	}
	return nil
}

// UpdateConfig updates autoscaling configuration for an application
func (as *AutoScaler) UpdateConfig(appID string, updates func(*AutoScaleConfig)) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	config, exists := as.applications[appID]
	if !exists {
		return fmt.Errorf("app %s not registered for autoscaling", appID)
	}

	updates(config)
	log.Printf("Updated autoscaling config for %s", appID)

	return nil
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
