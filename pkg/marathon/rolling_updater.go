package marathon

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// RollingUpdater implements rolling update strategies for Marathon applications
type RollingUpdater struct {
	client         *MarathonClient
	activeUpdates  map[string]*UpdateState
	updateHistory  []UpdateEvent
	mu             sync.RWMutex
}

// UpdateStrategy defines the update strategy
type UpdateStrategy string

const (
	RollingUpdate  UpdateStrategy = "rolling"
	BlueGreenUpdate UpdateStrategy = "blue-green"
	CanaryUpdate   UpdateStrategy = "canary"
	RecreateUpdate UpdateStrategy = "recreate"
)

// UpdateConfig defines rolling update parameters
type UpdateConfig struct {
	AppID            string
	Strategy         UpdateStrategy
	NewVersion       string
	NewImage         string
	NewConfig        map[string]string
	RollingConfig    *RollingConfig
	CanaryConfig     *CanaryConfig
	BlueGreenConfig  *BlueGreenConfig
	HealthCheckDelay time.Duration
	MaxUnavailable   int
	MaxSurge         int
}

// RollingConfig defines rolling update behavior
type RollingConfig struct {
	BatchSize         int           // Number of instances to update at once
	MinHealthyPercent float64       // Minimum healthy instances during update
	PauseTime         time.Duration // Pause between batches
	AutoRollback      bool          // Automatically rollback on failure
	HealthCheckGrace  time.Duration // Grace period for health checks
}

// CanaryConfig defines canary deployment behavior
type CanaryConfig struct {
	Stages           []CanaryStage // Canary deployment stages
	TrafficShiftMode string        // "manual" or "automatic"
	AnalysisInterval time.Duration // Time to analyze each stage
	SuccessThreshold float64       // Success rate threshold (e.g., 0.99)
	MetricsQuery     string        // Prometheus query for success rate
}

// CanaryStage defines a single canary stage
type CanaryStage struct {
	Name            string
	Weight          int           // Percentage of traffic (0-100)
	Duration        time.Duration // Duration of this stage
	PauseBeforeNext bool          // Require manual approval before next stage
}

// BlueGreenConfig defines blue-green deployment behavior
type BlueGreenConfig struct {
	AutoPromote       bool          // Automatically promote green to blue
	PromotionDelay    time.Duration // Delay before auto-promotion
	KeepOldVersion    bool          // Keep old version running
	TestTrafficWeight int           // Percentage of test traffic to green
}

// UpdateState tracks the state of an ongoing update
type UpdateState struct {
	AppID          string
	Strategy       UpdateStrategy
	StartTime      time.Time
	CurrentStage   string
	Progress       float64 // 0.0 to 1.0
	Status         UpdateStatus
	OldVersion     string
	NewVersion     string
	UpdatedTasks   int
	TotalTasks     int
	FailedTasks    []string
	HealthyTasks   []string
	ErrorMessage   string
}

// UpdateStatus represents update status
type UpdateStatus string

const (
	UpdateInProgress UpdateStatus = "in-progress"
	UpdatePaused     UpdateStatus = "paused"
	UpdateCompleted  UpdateStatus = "completed"
	UpdateFailed     UpdateStatus = "failed"
	UpdateRollingBack UpdateStatus = "rolling-back"
)

// UpdateEvent records an update event
type UpdateEvent struct {
	Timestamp  time.Time
	AppID      string
	Strategy   UpdateStrategy
	Stage      string
	Action     string
	Success    bool
	Message    string
}

// NewRollingUpdater creates a new rolling updater
func NewRollingUpdater(client *MarathonClient) *RollingUpdater {
	return &RollingUpdater{
		client:        client,
		activeUpdates: make(map[string]*UpdateState),
		updateHistory: []UpdateEvent{},
	}
}

// StartUpdate initiates a rolling update
func (ru *RollingUpdater) StartUpdate(ctx context.Context, config *UpdateConfig) error {
	ru.mu.Lock()
	defer ru.mu.Unlock()

	// Check if update already in progress
	if state, exists := ru.activeUpdates[config.AppID]; exists {
		if state.Status == UpdateInProgress {
			return fmt.Errorf("update already in progress for app %s", config.AppID)
		}
	}

	log.Printf("Starting %s update for app %s to version %s",
		config.Strategy, config.AppID, config.NewVersion)

	// Get current application state
	app, err := (*ru.client).GetApp(config.AppID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	// Initialize update state
	state := &UpdateState{
		AppID:        config.AppID,
		Strategy:     config.Strategy,
		StartTime:    time.Now(),
		CurrentStage: "initializing",
		Progress:     0.0,
		Status:       UpdateInProgress,
		OldVersion:   "", // Get from app labels
		NewVersion:   config.NewVersion,
		TotalTasks:   app.Instances,
	}

	ru.activeUpdates[config.AppID] = state

	// Execute update asynchronously
	go ru.executeUpdate(context.Background(), config, state)

	return nil
}

// executeUpdate performs the actual update
func (ru *RollingUpdater) executeUpdate(ctx context.Context, config *UpdateConfig, state *UpdateState) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Update panic for app %s: %v", config.AppID, r)
			ru.updateStatus(config.AppID, UpdateFailed, fmt.Sprintf("panic: %v", r))
		}
	}()

	var err error

	switch config.Strategy {
	case RollingUpdate:
		err = ru.rollingUpdate(ctx, config, state)
	case CanaryUpdate:
		err = ru.canaryUpdate(ctx, config, state)
	case BlueGreenUpdate:
		err = ru.blueGreenUpdate(ctx, config, state)
	case RecreateUpdate:
		err = ru.recreateUpdate(ctx, config, state)
	default:
		err = fmt.Errorf("unsupported update strategy: %s", config.Strategy)
	}

	if err != nil {
		log.Printf("Update failed for app %s: %v", config.AppID, err)
		ru.updateStatus(config.AppID, UpdateFailed, err.Error())

		// Auto-rollback if configured
		if config.RollingConfig != nil && config.RollingConfig.AutoRollback {
			log.Printf("Auto-rollback triggered for app %s", config.AppID)
			ru.rollback(ctx, config.AppID, state)
		}
	} else {
		log.Printf("Update completed successfully for app %s", config.AppID)
		ru.updateStatus(config.AppID, UpdateCompleted, "")
	}
}

// rollingUpdate performs a rolling update
func (ru *RollingUpdater) rollingUpdate(ctx context.Context, config *UpdateConfig, state *UpdateState) error {
	log.Printf("Executing rolling update for app %s", config.AppID)

	app, err := (*ru.client).GetApp(config.AppID)
	if err != nil {
		return err
	}

	batchSize := config.RollingConfig.BatchSize
	if batchSize == 0 {
		batchSize = max(1, app.Instances/10) // Default: 10% at a time
	}

	totalBatches := (app.Instances + batchSize - 1) / batchSize

	for batch := 0; batch < totalBatches; batch++ {
		state.CurrentStage = fmt.Sprintf("batch %d/%d", batch+1, totalBatches)
		state.Progress = float64(batch) / float64(totalBatches)

		log.Printf("App %s: Processing batch %d/%d", config.AppID, batch+1, totalBatches)

		// Update batch of tasks
		// In production, this would:
		// 1. Kill old tasks
		// 2. Wait for new tasks to start
		// 3. Wait for health checks
		// 4. Verify new tasks are healthy

		// Simulate batch update
		time.Sleep(config.RollingConfig.PauseTime)

		// Check health of new tasks
		healthy, err := ru.checkBatchHealth(config.AppID, batchSize)
		if err != nil || !healthy {
			return fmt.Errorf("batch %d health check failed", batch+1)
		}

		state.UpdatedTasks += batchSize
		if state.UpdatedTasks > app.Instances {
			state.UpdatedTasks = app.Instances
		}

		ru.recordEvent(UpdateEvent{
			Timestamp: time.Now(),
			AppID:     config.AppID,
			Strategy:  config.Strategy,
			Stage:     state.CurrentStage,
			Action:    "batch-completed",
			Success:   true,
			Message:   fmt.Sprintf("Updated %d tasks", batchSize),
		})
	}

	state.Progress = 1.0
	return nil
}

// canaryUpdate performs a canary deployment
func (ru *RollingUpdater) canaryUpdate(ctx context.Context, config *UpdateConfig, state *UpdateState) error {
	log.Printf("Executing canary update for app %s", config.AppID)

	if config.CanaryConfig == nil || len(config.CanaryConfig.Stages) == 0 {
		return fmt.Errorf("canary config not specified")
	}

	app, err := (*ru.client).GetApp(config.AppID)
	if err != nil {
		return err
	}

	totalStages := len(config.CanaryConfig.Stages)

	for i, stage := range config.CanaryConfig.Stages {
		state.CurrentStage = stage.Name
		state.Progress = float64(i) / float64(totalStages)

		log.Printf("App %s: Canary stage '%s' - %d%% traffic", config.AppID, stage.Name, stage.Weight)

		// Calculate number of canary instances
		canaryInstances := (app.Instances * stage.Weight) / 100
		if canaryInstances == 0 {
			canaryInstances = 1
		}

		// Deploy canary instances
		// In production:
		// 1. Create canary deployment with NewImage
		// 2. Route traffic according to weight
		// 3. Monitor metrics

		// Wait for analysis
		log.Printf("Analyzing canary stage '%s' for %v", stage.Name, config.CanaryConfig.AnalysisInterval)
		time.Sleep(config.CanaryConfig.AnalysisInterval)

		// Check metrics
		success, err := ru.analyzeCanaryMetrics(config, stage)
		if err != nil || !success {
			return fmt.Errorf("canary analysis failed at stage '%s'", stage.Name)
		}

		// Pause if required
		if stage.PauseBeforeNext && i < totalStages-1 {
			state.Status = UpdatePaused
			log.Printf("Canary paused at stage '%s', waiting for manual approval", stage.Name)
			// In production, wait for approval via API
			time.Sleep(30 * time.Second)
			state.Status = UpdateInProgress
		}

		ru.recordEvent(UpdateEvent{
			Timestamp: time.Now(),
			AppID:     config.AppID,
			Strategy:  config.Strategy,
			Stage:     stage.Name,
			Action:    "stage-completed",
			Success:   true,
			Message:   fmt.Sprintf("Canary stage %d%% traffic successful", stage.Weight),
		})
	}

	// Promote canary to full deployment
	log.Printf("Promoting canary to full deployment for app %s", config.AppID)
	state.CurrentStage = "promotion"
	state.Progress = 1.0

	return nil
}

// blueGreenUpdate performs a blue-green deployment
func (ru *RollingUpdater) blueGreenUpdate(ctx context.Context, config *UpdateConfig, state *UpdateState) error {
	log.Printf("Executing blue-green update for app %s", config.AppID)

	if config.BlueGreenConfig == nil {
		return fmt.Errorf("blue-green config not specified")
	}

	// Stage 1: Deploy green environment
	state.CurrentStage = "deploying-green"
	state.Progress = 0.25

	log.Printf("Deploying green environment for app %s", config.AppID)
	// In production:
	// 1. Create parallel deployment with NewImage
	// 2. Wait for all instances to be healthy

	time.Sleep(10 * time.Second)

	// Stage 2: Test green environment
	state.CurrentStage = "testing-green"
	state.Progress = 0.5

	if config.BlueGreenConfig.TestTrafficWeight > 0 {
		log.Printf("Routing %d%% test traffic to green", config.BlueGreenConfig.TestTrafficWeight)
		time.Sleep(config.BlueGreenConfig.PromotionDelay)
	}

	// Stage 3: Promote green to production
	state.CurrentStage = "promoting-green"
	state.Progress = 0.75

	if config.BlueGreenConfig.AutoPromote {
		log.Printf("Auto-promoting green to production for app %s", config.AppID)
		// Switch traffic from blue to green
		time.Sleep(5 * time.Second)
	} else {
		// Wait for manual promotion
		state.Status = UpdatePaused
		log.Printf("Waiting for manual promotion of app %s", config.AppID)
		// In production, wait for API call
		return fmt.Errorf("manual promotion required")
	}

	// Stage 4: Cleanup old version
	state.CurrentStage = "cleanup"
	state.Progress = 1.0

	if !config.BlueGreenConfig.KeepOldVersion {
		log.Printf("Cleaning up blue environment for app %s", config.AppID)
		// Remove old deployment
	}

	return nil
}

// recreateUpdate performs a recreate update (stop all, then start all)
func (ru *RollingUpdater) recreateUpdate(ctx context.Context, config *UpdateConfig, state *UpdateState) error {
	log.Printf("Executing recreate update for app %s", config.AppID)

	// Stage 1: Stop all old instances
	state.CurrentStage = "stopping-old"
	state.Progress = 0.3

	log.Printf("Stopping all instances of app %s", config.AppID)
	// Scale to 0
	time.Sleep(5 * time.Second)

	// Stage 2: Start new instances
	state.CurrentStage = "starting-new"
	state.Progress = 0.7

	log.Printf("Starting new instances of app %s", config.AppID)
	// Scale up with new version
	time.Sleep(10 * time.Second)

	// Stage 3: Wait for health
	state.CurrentStage = "health-check"
	state.Progress = 1.0

	healthy, err := ru.checkBatchHealth(config.AppID, 0)
	if err != nil || !healthy {
		return fmt.Errorf("health check failed after recreate")
	}

	return nil
}

// checkBatchHealth checks if a batch of tasks is healthy
func (ru *RollingUpdater) checkBatchHealth(appID string, batchSize int) (bool, error) {
	// In production, query Marathon API for task health
	app, err := (*ru.client).GetApp(appID)
	if err != nil {
		return false, err
	}

	healthPercent := float64(app.TasksHealthy) / float64(app.TasksRunning)
	return healthPercent >= 0.9, nil
}

// analyzeCanaryMetrics analyzes metrics for a canary stage
func (ru *RollingUpdater) analyzeCanaryMetrics(config *UpdateConfig, stage CanaryStage) (bool, error) {
	// In production, query Prometheus for success rate, error rate, latency
	// Compare canary metrics vs baseline
	log.Printf("Analyzing metrics for canary stage '%s'", stage.Name)

	// Simulated analysis
	successRate := 0.995 // 99.5% success rate

	threshold := config.CanaryConfig.SuccessThreshold
	if threshold == 0 {
		threshold = 0.99
	}

	return successRate >= threshold, nil
}

// updateStatus updates the status of an update
func (ru *RollingUpdater) updateStatus(appID string, status UpdateStatus, errorMsg string) {
	ru.mu.Lock()
	defer ru.mu.Unlock()

	if state, exists := ru.activeUpdates[appID]; exists {
		state.Status = status
		state.ErrorMessage = errorMsg

		if status == UpdateCompleted || status == UpdateFailed {
			// Move to history after completion
			delete(ru.activeUpdates, appID)
		}
	}
}

// rollback rolls back an update
func (ru *RollingUpdater) rollback(ctx context.Context, appID string, state *UpdateState) error {
	log.Printf("Rolling back update for app %s", appID)

	state.Status = UpdateRollingBack
	state.CurrentStage = "rollback"

	// In production:
	// 1. Revert to old version
	// 2. Scale back to original instance count
	// 3. Wait for health checks

	time.Sleep(10 * time.Second)

	ru.recordEvent(UpdateEvent{
		Timestamp: time.Now(),
		AppID:     appID,
		Strategy:  state.Strategy,
		Stage:     "rollback",
		Action:    "rollback-completed",
		Success:   true,
		Message:   "Rolled back to previous version",
	})

	return nil
}

// recordEvent records an update event
func (ru *RollingUpdater) recordEvent(event UpdateEvent) {
	ru.mu.Lock()
	defer ru.mu.Unlock()

	ru.updateHistory = append(ru.updateHistory, event)
	if len(ru.updateHistory) > 200 {
		ru.updateHistory = ru.updateHistory[len(ru.updateHistory)-200:]
	}
}

// GetUpdateState returns the current state of an update
func (ru *RollingUpdater) GetUpdateState(appID string) *UpdateState {
	ru.mu.RLock()
	defer ru.mu.RUnlock()

	if state, exists := ru.activeUpdates[appID]; exists {
		return state
	}
	return nil
}

// GetUpdateHistory returns update history
func (ru *RollingUpdater) GetUpdateHistory() []UpdateEvent {
	ru.mu.RLock()
	defer ru.mu.RUnlock()
	return ru.updateHistory
}

// PauseUpdate pauses an ongoing update
func (ru *RollingUpdater) PauseUpdate(appID string) error {
	ru.mu.Lock()
	defer ru.mu.Unlock()

	if state, exists := ru.activeUpdates[appID]; exists {
		if state.Status == UpdateInProgress {
			state.Status = UpdatePaused
			log.Printf("Paused update for app %s", appID)
			return nil
		}
	}
	return fmt.Errorf("no active update for app %s", appID)
}

// ResumeUpdate resumes a paused update
func (ru *RollingUpdater) ResumeUpdate(appID string) error {
	ru.mu.Lock()
	defer ru.mu.Unlock()

	if state, exists := ru.activeUpdates[appID]; exists {
		if state.Status == UpdatePaused {
			state.Status = UpdateInProgress
			log.Printf("Resumed update for app %s", appID)
			return nil
		}
	}
	return fmt.Errorf("no paused update for app %s", appID)
}
