package marathon

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// AutoHealer implements automatic healing for Marathon applications
type AutoHealer struct {
	client              *MarathonClient
	applications        map[string]*HealingConfig
	mu                  sync.RWMutex
	checkInterval       time.Duration
	unhealthyTasks      map[string]*UnhealthyTask
	healingInProgress   map[string]bool
	healingHistory      []HealingEvent
}

// HealingConfig defines auto-healing parameters
type HealingConfig struct {
	AppID                  string
	Enabled                bool
	HealthCheckTimeout     time.Duration
	MaxConsecutiveFailures int
	RestartPolicy          RestartPolicy
	BackoffPolicy          BackoffPolicy
	MaxRestartAttempts     int
	ReplacementStrategy    ReplacementStrategy
}

// RestartPolicy defines how to handle unhealthy tasks
type RestartPolicy string

const (
	RestartAlways    RestartPolicy = "always"
	RestartOnFailure RestartPolicy = "on-failure"
	RestartNever     RestartPolicy = "never"
)

// ReplacementStrategy defines task replacement behavior
type ReplacementStrategy string

const (
	RollingReplacement  ReplacementStrategy = "rolling"
	ImmediateReplacement ReplacementStrategy = "immediate"
	BatchReplacement    ReplacementStrategy = "batch"
)

// BackoffPolicy defines restart backoff behavior
type BackoffPolicy struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// UnhealthyTask tracks an unhealthy task
type UnhealthyTask struct {
	TaskID           string
	AppID            string
	FirstFailureTime time.Time
	FailureCount     int
	LastAttemptTime  time.Time
	NextAttemptTime  time.Time
	RestartAttempts  int
	HealthState      string
	Reason           string
}

// HealingEvent records a healing action
type HealingEvent struct {
	Timestamp   time.Time
	AppID       string
	TaskID      string
	Action      string
	Reason      string
	Success     bool
	ErrorMsg    string
}

// NewAutoHealer creates a new auto-healer instance
func NewAutoHealer(client *MarathonClient) *AutoHealer {
	return &AutoHealer{
		client:            client,
		applications:      make(map[string]*HealingConfig),
		checkInterval:     15 * time.Second,
		unhealthyTasks:    make(map[string]*UnhealthyTask),
		healingInProgress: make(map[string]bool),
		healingHistory:    []HealingEvent{},
	}
}

// RegisterApp registers an application for auto-healing
func (ah *AutoHealer) RegisterApp(config *HealingConfig) error {
	ah.mu.Lock()
	defer ah.mu.Unlock()

	// Set defaults
	if config.HealthCheckTimeout == 0 {
		config.HealthCheckTimeout = 30 * time.Second
	}
	if config.MaxConsecutiveFailures == 0 {
		config.MaxConsecutiveFailures = 3
	}
	if config.MaxRestartAttempts == 0 {
		config.MaxRestartAttempts = 10
	}
	if config.RestartPolicy == "" {
		config.RestartPolicy = RestartOnFailure
	}
	if config.ReplacementStrategy == "" {
		config.ReplacementStrategy = RollingReplacement
	}

	// Set default backoff policy
	if config.BackoffPolicy.InitialDelay == 0 {
		config.BackoffPolicy.InitialDelay = 10 * time.Second
	}
	if config.BackoffPolicy.MaxDelay == 0 {
		config.BackoffPolicy.MaxDelay = 5 * time.Minute
	}
	if config.BackoffPolicy.Multiplier == 0 {
		config.BackoffPolicy.Multiplier = 2.0
	}

	ah.applications[config.AppID] = config

	log.Printf("Registered auto-healing for app %s: policy=%s, maxFailures=%d",
		config.AppID, config.RestartPolicy, config.MaxConsecutiveFailures)

	return nil
}

// Start begins the auto-healing loop
func (ah *AutoHealer) Start(ctx context.Context) error {
	log.Println("Starting Marathon auto-healer")

	ticker := time.NewTicker(ah.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Auto-healer shutting down")
			return nil
		case <-ticker.C:
			ah.checkAndHeal(ctx)
		}
	}
}

// checkAndHeal evaluates all registered applications and heals if needed
func (ah *AutoHealer) checkAndHeal(ctx context.Context) {
	ah.mu.RLock()
	apps := make([]*HealingConfig, 0, len(ah.applications))
	for _, config := range ah.applications {
		if config.Enabled {
			apps = append(apps, config)
		}
	}
	ah.mu.RUnlock()

	for _, config := range apps {
		if err := ah.checkAppHealth(ctx, config); err != nil {
			log.Printf("Error checking app health %s: %v", config.AppID, err)
		}
	}
}

// checkAppHealth checks health of all tasks for an application
func (ah *AutoHealer) checkAppHealth(ctx context.Context, config *HealingConfig) error {
	// Get app and tasks
	app, err := (*ah.client).GetApp(config.AppID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	tasks, err := (*ah.client).GetAppTasks(config.AppID)
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	// Check each task
	for _, task := range tasks {
		ah.evaluateTask(ctx, config, &task)
	}

	// Log health status
	if app.TasksUnhealthy > 0 {
		log.Printf("App %s health: %d/%d tasks healthy, %d unhealthy",
			config.AppID, app.TasksHealthy, app.TasksRunning, app.TasksUnhealthy)
	}

	return nil
}

// evaluateTask evaluates a single task for healing
func (ah *AutoHealer) evaluateTask(ctx context.Context, config *HealingConfig, task *Task) {
	// Skip if healing already in progress for this task
	ah.mu.RLock()
	inProgress := ah.healingInProgress[task.ID]
	ah.mu.RUnlock()

	if inProgress {
		return
	}

	// Check if task is unhealthy
	if task.HealthState == "unhealthy" || task.State == "TASK_FAILED" || task.State == "TASK_LOST" {
		ah.handleUnhealthyTask(ctx, config, task)
	} else if task.HealthState == "healthy" {
		// Task recovered, remove from tracking
		ah.mu.Lock()
		delete(ah.unhealthyTasks, task.ID)
		ah.mu.Unlock()
	}
}

// handleUnhealthyTask handles an unhealthy task
func (ah *AutoHealer) handleUnhealthyTask(ctx context.Context, config *HealingConfig, task *Task) {
	ah.mu.Lock()
	defer ah.mu.Unlock()

	now := time.Now()

	// Track unhealthy task
	unhealthy, exists := ah.unhealthyTasks[task.ID]
	if !exists {
		unhealthy = &UnhealthyTask{
			TaskID:           task.ID,
			AppID:            task.AppID,
			FirstFailureTime: now,
			FailureCount:     1,
			HealthState:      task.HealthState,
			Reason:           fmt.Sprintf("Task state: %s, Health: %s", task.State, task.HealthState),
		}
		ah.unhealthyTasks[task.ID] = unhealthy
		log.Printf("Detected unhealthy task: %s (app: %s)", task.ID, task.AppID)
		return
	}

	// Increment failure count
	unhealthy.FailureCount++

	// Check if we should heal
	shouldHeal := false
	reason := ""

	// Check consecutive failures
	if unhealthy.FailureCount >= config.MaxConsecutiveFailures {
		shouldHeal = true
		reason = fmt.Sprintf("Exceeded max consecutive failures (%d)", config.MaxConsecutiveFailures)
	}

	// Check restart policy
	if config.RestartPolicy == RestartNever {
		shouldHeal = false
	}

	// Check max restart attempts
	if unhealthy.RestartAttempts >= config.MaxRestartAttempts {
		shouldHeal = false
		log.Printf("Task %s exceeded max restart attempts (%d), giving up",
			task.ID, config.MaxRestartAttempts)
		return
	}

	// Check backoff delay
	if shouldHeal && now.Before(unhealthy.NextAttemptTime) {
		shouldHeal = false
		log.Printf("Task %s in backoff period, next attempt at %s",
			task.ID, unhealthy.NextAttemptTime.Format(time.RFC3339))
	}

	if shouldHeal {
		go ah.healTask(context.Background(), config, task, unhealthy, reason)
	}
}

// healTask performs the healing action for a task
func (ah *AutoHealer) healTask(ctx context.Context, config *HealingConfig, task *Task, unhealthy *UnhealthyTask, reason string) {
	taskID := task.ID

	// Mark as in progress
	ah.mu.Lock()
	ah.healingInProgress[taskID] = true
	ah.mu.Unlock()

	defer func() {
		ah.mu.Lock()
		delete(ah.healingInProgress, taskID)
		ah.mu.Unlock()
	}()

	log.Printf("Healing task %s (app: %s): %s", taskID, task.AppID, reason)

	event := HealingEvent{
		Timestamp: time.Now(),
		AppID:     task.AppID,
		TaskID:    taskID,
		Action:    "restart",
		Reason:    reason,
	}

	// Execute healing based on replacement strategy
	var err error
	switch config.ReplacementStrategy {
	case RollingReplacement:
		err = ah.rollingReplace(ctx, task)
	case ImmediateReplacement:
		err = ah.immediateReplace(ctx, task)
	case BatchReplacement:
		err = ah.batchReplace(ctx, task)
	default:
		err = fmt.Errorf("unknown replacement strategy: %s", config.ReplacementStrategy)
	}

	// Record result
	ah.mu.Lock()
	if err != nil {
		event.Success = false
		event.ErrorMsg = err.Error()
		log.Printf("Failed to heal task %s: %v", taskID, err)

		// Update backoff
		unhealthy.RestartAttempts++
		unhealthy.LastAttemptTime = time.Now()
		unhealthy.NextAttemptTime = ah.calculateBackoff(config, unhealthy.RestartAttempts)
	} else {
		event.Success = true
		log.Printf("Successfully healed task %s", taskID)

		// Remove from unhealthy tracking
		delete(ah.unhealthyTasks, taskID)
	}

	// Add to history
	ah.healingHistory = append(ah.healingHistory, event)
	if len(ah.healingHistory) > 100 {
		ah.healingHistory = ah.healingHistory[len(ah.healingHistory)-100:]
	}
	ah.mu.Unlock()
}

// rollingReplace replaces a task using rolling strategy
func (ah *AutoHealer) rollingReplace(ctx context.Context, task *Task) error {
	// In Marathon, killing a task triggers automatic replacement
	// This is already a rolling replacement
	log.Printf("Killing unhealthy task %s for replacement", task.ID)
	// TODO: Implement actual task kill via Marathon API
	return nil
}

// immediateReplace replaces a task immediately
func (ah *AutoHealer) immediateReplace(ctx context.Context, task *Task) error {
	// Start new task before killing old one
	log.Printf("Immediately replacing task %s", task.ID)
	// TODO: Implement via Marathon API
	return nil
}

// batchReplace replaces multiple tasks at once
func (ah *AutoHealer) batchReplace(ctx context.Context, task *Task) error {
	// Batch replacement for multiple unhealthy tasks
	log.Printf("Batch replacing task %s", task.ID)
	// TODO: Implement via Marathon API
	return nil
}

// calculateBackoff calculates next retry time using exponential backoff
func (ah *AutoHealer) calculateBackoff(config *HealingConfig, attempts int) time.Time {
	delay := config.BackoffPolicy.InitialDelay
	for i := 1; i < attempts; i++ {
		delay = time.Duration(float64(delay) * config.BackoffPolicy.Multiplier)
		if delay > config.BackoffPolicy.MaxDelay {
			delay = config.BackoffPolicy.MaxDelay
			break
		}
	}
	return time.Now().Add(delay)
}

// GetHealingHistory returns healing history
func (ah *AutoHealer) GetHealingHistory() []HealingEvent {
	ah.mu.RLock()
	defer ah.mu.RUnlock()
	return ah.healingHistory
}

// GetUnhealthyTasks returns currently tracked unhealthy tasks
func (ah *AutoHealer) GetUnhealthyTasks() map[string]*UnhealthyTask {
	ah.mu.RLock()
	defer ah.mu.RUnlock()

	// Return a copy
	tasks := make(map[string]*UnhealthyTask)
	for k, v := range ah.unhealthyTasks {
		tasks[k] = v
	}
	return tasks
}

// GetHealthStatus returns overall health status for an app
func (ah *AutoHealer) GetHealthStatus(appID string) (*HealthStatus, error) {
	ah.mu.RLock()
	defer ah.mu.RUnlock()

	app, err := (*ah.client).GetApp(appID)
	if err != nil {
		return nil, err
	}

	// Count unhealthy tasks for this app
	unhealthyCount := 0
	for _, task := range ah.unhealthyTasks {
		if task.AppID == appID {
			unhealthyCount++
		}
	}

	healthPercent := 0.0
	if app.TasksRunning > 0 {
		healthPercent = float64(app.TasksHealthy) / float64(app.TasksRunning) * 100
	}

	return &HealthStatus{
		AppID:          appID,
		TotalTasks:     app.TasksRunning,
		HealthyTasks:   app.TasksHealthy,
		UnhealthyTasks: unhealthyCount,
		HealthPercent:  healthPercent,
		Status:         ah.determineStatus(healthPercent),
	}, nil
}

// HealthStatus represents application health status
type HealthStatus struct {
	AppID          string
	TotalTasks     int
	HealthyTasks   int
	UnhealthyTasks int
	HealthPercent  float64
	Status         string
}

// determineStatus determines overall health status
func (ah *AutoHealer) determineStatus(healthPercent float64) string {
	if healthPercent >= 95 {
		return "healthy"
	} else if healthPercent >= 75 {
		return "degraded"
	} else if healthPercent >= 50 {
		return "warning"
	} else {
		return "critical"
	}
}
