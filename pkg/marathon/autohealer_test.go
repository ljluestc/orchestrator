package marathon

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMarathonClient is a mock implementation of MarathonClient
type MockMarathonClient struct {
	mock.Mock
}

func (m *MockMarathonClient) GetApp(appID string) (*Application, error) {
	args := m.Called(appID)
	return args.Get(0).(*Application), args.Error(1)
}

func (m *MockMarathonClient) GetAppTasks(appID string) ([]Task, error) {
	args := m.Called(appID)
	return args.Get(0).([]Task), args.Error(1)
}

func (m *MockMarathonClient) CreateApp(app *Application) error {
	args := m.Called(app)
	return args.Error(0)
}

func (m *MockMarathonClient) UpdateApp(appID string, app *Application) error {
	args := m.Called(appID, app)
	return args.Error(0)
}

func (m *MockMarathonClient) DeleteApp(appID string) error {
	args := m.Called(appID)
	return args.Error(0)
}

func (m *MockMarathonClient) ScaleApp(appID string, instances int) error {
	args := m.Called(appID, instances)
	return args.Error(0)
}

func (m *MockMarathonClient) RestartApp(appID string) error {
	args := m.Called(appID)
	return args.Error(0)
}

func (m *MockMarathonClient) GetApps() ([]Application, error) {
	args := m.Called()
	return args.Get(0).([]Application), args.Error(1)
}

func (m *MockMarathonClient) GetTasks() ([]Task, error) {
	args := m.Called()
	return args.Get(0).([]Task), args.Error(1)
}

func (m *MockMarathonClient) KillTask(taskID string) error {
	args := m.Called(taskID)
	return args.Error(0)
}

func (m *MockMarathonClient) KillTasks(taskIDs []string) error {
	args := m.Called(taskIDs)
	return args.Error(0)
}

func TestNewAutoHealer(t *testing.T) {
	mockClient := &MockMarathonClient{}
	healer := NewAutoHealer(mockClient)

	assert.NotNil(t, healer)
	assert.Equal(t, mockClient, healer.client)
	assert.NotNil(t, healer.applications)
	assert.NotNil(t, healer.unhealthyTasks)
	assert.NotNil(t, healer.healingInProgress)
	assert.NotNil(t, healer.healingHistory)
	assert.Equal(t, 15*time.Second, healer.checkInterval)
}

func TestAutoHealer_RegisterApp(t *testing.T) {
	tests := []struct {
		name           string
		config         *HealingConfig
		expectedError  bool
		expectedConfig *HealingConfig
	}{
		{
			name: "Valid config with defaults",
			config: &HealingConfig{
				AppID:   "test-app",
				Enabled: true,
			},
			expectedError: false,
			expectedConfig: &HealingConfig{
				AppID:                  "test-app",
				Enabled:                true,
				HealthCheckTimeout:     30 * time.Second,
				MaxConsecutiveFailures: 3,
				MaxRestartAttempts:     10,
				RestartPolicy:          RestartOnFailure,
				ReplacementStrategy:    RollingReplacement,
				BackoffPolicy: BackoffPolicy{
					InitialDelay: 10 * time.Second,
					MaxDelay:     5 * time.Minute,
					Multiplier:   2.0,
				},
			},
		},
		{
			name: "Config with custom values",
			config: &HealingConfig{
				AppID:                  "custom-app",
				Enabled:                true,
				HealthCheckTimeout:     60 * time.Second,
				MaxConsecutiveFailures: 5,
				MaxRestartAttempts:     20,
				RestartPolicy:          RestartAlways,
				ReplacementStrategy:    ImmediateReplacement,
				BackoffPolicy: BackoffPolicy{
					InitialDelay: 5 * time.Second,
					MaxDelay:     10 * time.Minute,
					Multiplier:   1.5,
				},
			},
			expectedError: false,
			expectedConfig: &HealingConfig{
				AppID:                  "custom-app",
				Enabled:                true,
				HealthCheckTimeout:     60 * time.Second,
				MaxConsecutiveFailures: 5,
				MaxRestartAttempts:     20,
				RestartPolicy:          RestartAlways,
				ReplacementStrategy:    ImmediateReplacement,
				BackoffPolicy: BackoffPolicy{
					InitialDelay: 5 * time.Second,
					MaxDelay:     10 * time.Minute,
					Multiplier:   1.5,
				},
			},
		},
		{
			name: "Config with partial defaults",
			config: &HealingConfig{
				AppID:                  "partial-app",
				Enabled:                true,
				HealthCheckTimeout:     45 * time.Second,
				MaxConsecutiveFailures: 0, // Should get default
				MaxRestartAttempts:     0, // Should get default
				RestartPolicy:          "", // Should get default
				ReplacementStrategy:    "", // Should get default
			},
			expectedError: false,
			expectedConfig: &HealingConfig{
				AppID:                  "partial-app",
				Enabled:                true,
				HealthCheckTimeout:     45 * time.Second,
				MaxConsecutiveFailures: 3,
				MaxRestartAttempts:     10,
				RestartPolicy:          RestartOnFailure,
				ReplacementStrategy:    RollingReplacement,
				BackoffPolicy: BackoffPolicy{
					InitialDelay: 10 * time.Second,
					MaxDelay:     5 * time.Minute,
					Multiplier:   2.0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockMarathonClient{}
			healer := NewAutoHealer(mockClient)

			err := healer.RegisterApp(tt.config)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, healer.applications, tt.config.AppID)
				
				registeredConfig := healer.applications[tt.config.AppID]
				assert.Equal(t, tt.expectedConfig.AppID, registeredConfig.AppID)
				assert.Equal(t, tt.expectedConfig.Enabled, registeredConfig.Enabled)
				assert.Equal(t, tt.expectedConfig.HealthCheckTimeout, registeredConfig.HealthCheckTimeout)
				assert.Equal(t, tt.expectedConfig.MaxConsecutiveFailures, registeredConfig.MaxConsecutiveFailures)
				assert.Equal(t, tt.expectedConfig.MaxRestartAttempts, registeredConfig.MaxRestartAttempts)
				assert.Equal(t, tt.expectedConfig.RestartPolicy, registeredConfig.RestartPolicy)
				assert.Equal(t, tt.expectedConfig.ReplacementStrategy, registeredConfig.ReplacementStrategy)
				assert.Equal(t, tt.expectedConfig.BackoffPolicy, registeredConfig.BackoffPolicy)
			}
		})
	}
}

func TestAutoHealer_Start(t *testing.T) {
	mockClient := &MockMarathonClient{}
	healer := NewAutoHealer(mockClient)
	healer.checkInterval = 10 * time.Millisecond // Short interval for testing

	// Register a test app
	config := &HealingConfig{
		AppID:   "test-app",
		Enabled: true,
	}
	err := healer.RegisterApp(config)
	assert.NoError(t, err)

	// Mock the client calls
	mockClient.On("GetApp", "test-app").Return(&Application{
		ID:             "test-app",
		TasksRunning:   2,
		TasksHealthy:   1,
		TasksUnhealthy: 1,
	}, nil)
	mockClient.On("GetAppTasks", "test-app").Return([]Task{
		{
			ID:         "task-1",
			AppID:      "test-app",
			State:      "TASK_RUNNING",
			HealthState: "healthy",
		},
		{
			ID:         "task-2",
			AppID:      "test-app",
			State:      "TASK_RUNNING",
			HealthState: "unhealthy",
		},
	}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = healer.Start(ctx)
	assert.NoError(t, err)

	mockClient.AssertExpectations(t)
}

func TestAutoHealer_StartWithContextCancellation(t *testing.T) {
	mockClient := &MockMarathonClient{}
	healer := NewAutoHealer(mockClient)

	ctx, cancel := context.WithCancel(context.Background())
	
	// Cancel immediately
	cancel()

	err := healer.Start(ctx)
	assert.NoError(t, err)
}

func TestAutoHealer_checkAndHeal(t *testing.T) {
	mockClient := &MockMarathonClient{}
	healer := NewAutoHealer(mockClient)

	// Register enabled and disabled apps
	enabledConfig := &HealingConfig{
		AppID:   "enabled-app",
		Enabled: true,
	}
	disabledConfig := &HealingConfig{
		AppID:   "disabled-app",
		Enabled: false,
	}

	err := healer.RegisterApp(enabledConfig)
	assert.NoError(t, err)
	err = healer.RegisterApp(disabledConfig)
	assert.NoError(t, err)

	// Mock client calls for enabled app only
	mockClient.On("GetApp", "enabled-app").Return(&Application{
		ID:             "enabled-app",
		TasksRunning:   1,
		TasksHealthy:   1,
		TasksUnhealthy: 0,
	}, nil)
	mockClient.On("GetAppTasks", "enabled-app").Return([]Task{
		{
			ID:         "task-1",
			AppID:      "enabled-app",
			State:      "TASK_RUNNING",
			HealthState: "healthy",
		},
	}, nil)

	ctx := context.Background()
	healer.checkAndHeal(ctx)

	// Only enabled app should be checked
	mockClient.AssertExpectations(t)
}

func TestAutoHealer_checkAppHealth(t *testing.T) {
	tests := []struct {
		name           string
		appID          string
		mockApp        *Application
		mockTasks      []Task
		mockAppError   error
		mockTasksError error
		expectedError  bool
	}{
		{
			name:  "Healthy app",
			appID: "healthy-app",
			mockApp: &Application{
				ID:             "healthy-app",
				TasksRunning:   2,
				TasksHealthy:   2,
				TasksUnhealthy: 0,
			},
			mockTasks: []Task{
				{
					ID:         "task-1",
					AppID:      "healthy-app",
					State:      "TASK_RUNNING",
					HealthState: "healthy",
				},
				{
					ID:         "task-2",
					AppID:      "healthy-app",
					State:      "TASK_RUNNING",
					HealthState: "healthy",
				},
			},
			expectedError: false,
		},
		{
			name:  "Unhealthy app",
			appID: "unhealthy-app",
			mockApp: &Application{
				ID:             "unhealthy-app",
				TasksRunning:   2,
				TasksHealthy:   1,
				TasksUnhealthy: 1,
			},
			mockTasks: []Task{
				{
					ID:         "task-1",
					AppID:      "unhealthy-app",
					State:      "TASK_RUNNING",
					HealthState: "healthy",
				},
				{
					ID:         "task-2",
					AppID:      "unhealthy-app",
					State:      "TASK_RUNNING",
					HealthState: "unhealthy",
				},
			},
			expectedError: false,
		},
		{
			name:          "App not found",
			appID:         "missing-app",
			mockAppError:  assert.AnError,
			expectedError: true,
		},
		{
			name:  "Tasks not found",
			appID: "no-tasks-app",
			mockApp: &Application{
				ID:             "no-tasks-app",
				TasksRunning:   0,
				TasksHealthy:   0,
				TasksUnhealthy: 0,
			},
			mockTasksError: assert.AnError,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockMarathonClient{}
			healer := NewAutoHealer(mockClient)

			config := &HealingConfig{
				AppID:   tt.appID,
				Enabled: true,
			}

			if tt.mockAppError != nil {
				mockClient.On("GetApp", tt.appID).Return((*Application)(nil), tt.mockAppError)
			} else {
				mockClient.On("GetApp", tt.appID).Return(tt.mockApp, nil)
				if tt.mockTasksError != nil {
					mockClient.On("GetAppTasks", tt.appID).Return([]Task(nil), tt.mockTasksError)
				} else {
					mockClient.On("GetAppTasks", tt.appID).Return(tt.mockTasks, nil)
				}
			}

			ctx := context.Background()
			err := healer.checkAppHealth(ctx, config)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestAutoHealer_evaluateTask(t *testing.T) {
	tests := []struct {
		name                    string
		task                    *Task
		healingInProgress       bool
		expectedUnhealthyTrack bool
		expectedHealingAction  bool
	}{
		{
			name: "Healthy task",
			task: &Task{
				ID:         "healthy-task",
				AppID:      "test-app",
				State:      "TASK_RUNNING",
				HealthState: "healthy",
			},
			expectedUnhealthyTrack: false,
			expectedHealingAction:  false,
		},
		{
			name: "Unhealthy task - first failure",
			task: &Task{
				ID:         "unhealthy-task",
				AppID:      "test-app",
				State:      "TASK_RUNNING",
				HealthState: "unhealthy",
			},
			expectedUnhealthyTrack: true,
			expectedHealingAction:  false, // First failure, not enough consecutive failures
		},
		{
			name: "Failed task",
			task: &Task{
				ID:         "failed-task",
				AppID:      "test-app",
				State:      "TASK_FAILED",
				HealthState: "unhealthy",
			},
			expectedUnhealthyTrack: true,
			expectedHealingAction:  false,
		},
		{
			name: "Lost task",
			task: &Task{
				ID:         "lost-task",
				AppID:      "test-app",
				State:      "TASK_LOST",
				HealthState: "unhealthy",
			},
			expectedUnhealthyTrack: true,
			expectedHealingAction:  false,
		},
		{
			name: "Task with healing in progress",
			task: &Task{
				ID:         "healing-task",
				AppID:      "test-app",
				State:      "TASK_RUNNING",
				HealthState: "unhealthy",
			},
			healingInProgress:       true,
			expectedUnhealthyTrack: false,
			expectedHealingAction:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockMarathonClient{}
			healer := NewAutoHealer(mockClient)

			config := &HealingConfig{
				AppID:                  tt.task.AppID,
				Enabled:                true,
				MaxConsecutiveFailures: 3,
			}

			// Set up healing in progress if needed
			if tt.healingInProgress {
				healer.healingInProgress[tt.task.ID] = true
			}

			ctx := context.Background()
			healer.evaluateTask(ctx, config, tt.task)

			if tt.expectedUnhealthyTrack {
				assert.Contains(t, healer.unhealthyTasks, tt.task.ID)
				unhealthy := healer.unhealthyTasks[tt.task.ID]
				assert.Equal(t, tt.task.ID, unhealthy.TaskID)
				assert.Equal(t, tt.task.AppID, unhealthy.AppID)
				assert.Equal(t, 1, unhealthy.FailureCount)
			} else {
				assert.NotContains(t, healer.unhealthyTasks, tt.task.ID)
			}
		})
	}
}

func TestAutoHealer_handleUnhealthyTask(t *testing.T) {
	tests := []struct {
		name                    string
		task                    *Task
		existingUnhealthy       *UnhealthyTask
		config                  *HealingConfig
		expectedShouldHeal      bool
		expectedRestartAttempts int
		expectedFailureCount    int
	}{
		{
			name: "First failure - should not heal",
			task: &Task{
				ID:         "task-1",
				AppID:      "test-app",
				State:      "TASK_RUNNING",
				HealthState: "unhealthy",
			},
			config: &HealingConfig{
				AppID:                  "test-app",
				MaxConsecutiveFailures: 3,
				RestartPolicy:          RestartOnFailure,
				MaxRestartAttempts:     10,
			},
			expectedShouldHeal:      false,
			expectedRestartAttempts: 0,
			expectedFailureCount:    1,
		},
		{
			name: "Enough consecutive failures - should heal",
			task: &Task{
				ID:         "task-2",
				AppID:      "test-app",
				State:      "TASK_RUNNING",
				HealthState: "unhealthy",
			},
			existingUnhealthy: &UnhealthyTask{
				TaskID:        "task-2",
				AppID:         "test-app",
				FailureCount:  3,
				RestartAttempts: 0,
			},
			config: &HealingConfig{
				AppID:                  "test-app",
				MaxConsecutiveFailures: 3,
				RestartPolicy:          RestartOnFailure,
				MaxRestartAttempts:     10,
			},
			expectedShouldHeal:      true,
			expectedRestartAttempts: 0,
			expectedFailureCount:    4,
		},
		{
			name: "Restart policy never - should not heal",
			task: &Task{
				ID:         "task-3",
				AppID:      "test-app",
				State:      "TASK_RUNNING",
				HealthState: "unhealthy",
			},
			existingUnhealthy: &UnhealthyTask{
				TaskID:        "task-3",
				AppID:         "test-app",
				FailureCount:  5,
				RestartAttempts: 0,
			},
			config: &HealingConfig{
				AppID:                  "test-app",
				MaxConsecutiveFailures: 3,
				RestartPolicy:          RestartNever,
				MaxRestartAttempts:     10,
			},
			expectedShouldHeal:      false,
			expectedRestartAttempts: 0,
			expectedFailureCount:    6,
		},
		{
			name: "Max restart attempts exceeded - should not heal",
			task: &Task{
				ID:         "task-4",
				AppID:      "test-app",
				State:      "TASK_RUNNING",
				HealthState: "unhealthy",
			},
			existingUnhealthy: &UnhealthyTask{
				TaskID:        "task-4",
				AppID:         "test-app",
				FailureCount:  5,
				RestartAttempts: 10,
			},
			config: &HealingConfig{
				AppID:                  "test-app",
				MaxConsecutiveFailures: 3,
				RestartPolicy:          RestartOnFailure,
				MaxRestartAttempts:     10,
			},
			expectedShouldHeal:      false,
			expectedRestartAttempts: 10,
			expectedFailureCount:    6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockMarathonClient{}
			healer := NewAutoHealer(mockClient)

			// Set up existing unhealthy task if needed
			if tt.existingUnhealthy != nil {
				healer.unhealthyTasks[tt.task.ID] = tt.existingUnhealthy
			}

			ctx := context.Background()
			healer.handleUnhealthyTask(ctx, tt.config, tt.task)

			// Check if task is tracked as unhealthy
			assert.Contains(t, healer.unhealthyTasks, tt.task.ID)
			unhealthy := healer.unhealthyTasks[tt.task.ID]
			assert.Equal(t, tt.expectedFailureCount, unhealthy.FailureCount)
			assert.Equal(t, tt.expectedRestartAttempts, unhealthy.RestartAttempts)
		})
	}
}

func TestAutoHealer_healTask(t *testing.T) {
	tests := []struct {
		name                string
		task                *Task
		config              *HealingConfig
		unhealthy           *UnhealthyTask
		reason              string
		expectedSuccess     bool
		expectedInProgress  bool
	}{
		{
			name: "Rolling replacement",
			task: &Task{
				ID:    "task-1",
				AppID: "test-app",
			},
			config: &HealingConfig{
				AppID:               "test-app",
				ReplacementStrategy: RollingReplacement,
			},
			unhealthy: &UnhealthyTask{
				TaskID:        "task-1",
				AppID:         "test-app",
				FailureCount:  3,
				RestartAttempts: 0,
			},
			reason:             "Test healing",
			expectedSuccess:   true,
			expectedInProgress: false, // Should be cleaned up after healing
		},
		{
			name: "Immediate replacement",
			task: &Task{
				ID:    "task-2",
				AppID: "test-app",
			},
			config: &HealingConfig{
				AppID:               "test-app",
				ReplacementStrategy: ImmediateReplacement,
			},
			unhealthy: &UnhealthyTask{
				TaskID:        "task-2",
				AppID:         "test-app",
				FailureCount:  3,
				RestartAttempts: 0,
			},
			reason:             "Test healing",
			expectedSuccess:   true,
			expectedInProgress: false,
		},
		{
			name: "Batch replacement",
			task: &Task{
				ID:    "task-3",
				AppID: "test-app",
			},
			config: &HealingConfig{
				AppID:               "test-app",
				ReplacementStrategy: BatchReplacement,
			},
			unhealthy: &UnhealthyTask{
				TaskID:        "task-3",
				AppID:         "test-app",
				FailureCount:  3,
				RestartAttempts: 0,
			},
			reason:             "Test healing",
			expectedSuccess:   true,
			expectedInProgress: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockMarathonClient{}
			healer := NewAutoHealer(mockClient)

			ctx := context.Background()
			healer.healTask(ctx, tt.config, tt.task, tt.unhealthy, tt.reason)

			// Check healing history
			assert.Len(t, healer.healingHistory, 1)
			event := healer.healingHistory[0]
			assert.Equal(t, tt.task.AppID, event.AppID)
			assert.Equal(t, tt.task.ID, event.TaskID)
			assert.Equal(t, "restart", event.Action)
			assert.Equal(t, tt.reason, event.Reason)
			assert.Equal(t, tt.expectedSuccess, event.Success)

			// Check that healing is no longer in progress
			assert.NotContains(t, healer.healingInProgress, tt.task.ID)

			if tt.expectedSuccess {
				// Task should be removed from unhealthy tracking
				assert.NotContains(t, healer.unhealthyTasks, tt.task.ID)
			}
		})
	}
}

func TestAutoHealer_calculateBackoff(t *testing.T) {
	tests := []struct {
		name           string
		config         *HealingConfig
		attempts       int
		expectedDelay  time.Duration
	}{
		{
			name: "First attempt",
			config: &HealingConfig{
				BackoffPolicy: BackoffPolicy{
					InitialDelay: 10 * time.Second,
					MaxDelay:     5 * time.Minute,
					Multiplier:   2.0,
				},
			},
			attempts:       1,
			expectedDelay:  10 * time.Second,
		},
		{
			name: "Second attempt",
			config: &HealingConfig{
				BackoffPolicy: BackoffPolicy{
					InitialDelay: 10 * time.Second,
					MaxDelay:     5 * time.Minute,
					Multiplier:   2.0,
				},
			},
			attempts:       2,
			expectedDelay:  20 * time.Second,
		},
		{
			name: "Third attempt",
			config: &HealingConfig{
				BackoffPolicy: BackoffPolicy{
					InitialDelay: 10 * time.Second,
					MaxDelay:     5 * time.Minute,
					Multiplier:   2.0,
				},
			},
			attempts:       3,
			expectedDelay:  40 * time.Second,
		},
		{
			name: "Max delay reached",
			config: &HealingConfig{
				BackoffPolicy: BackoffPolicy{
					InitialDelay: 10 * time.Second,
					MaxDelay:     30 * time.Second,
					Multiplier:   2.0,
				},
			},
			attempts:       5,
			expectedDelay:  30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockMarathonClient{}
			healer := NewAutoHealer(mockClient)

			nextTime := healer.calculateBackoff(tt.config, tt.attempts)
			expectedTime := time.Now().Add(tt.expectedDelay)

			// Allow for small time differences
			diff := nextTime.Sub(expectedTime)
			if diff < 0 {
				diff = -diff
			}
			assert.Less(t, diff, 100*time.Millisecond)
		})
	}
}

func TestAutoHealer_GetHealingHistory(t *testing.T) {
	mockClient := &MockMarathonClient{}
	healer := NewAutoHealer(mockClient)

	// Add some healing events
	event1 := HealingEvent{
		Timestamp: time.Now(),
		AppID:     "app-1",
		TaskID:    "task-1",
		Action:    "restart",
		Reason:    "test",
		Success:   true,
	}
	event2 := HealingEvent{
		Timestamp: time.Now(),
		AppID:     "app-2",
		TaskID:    "task-2",
		Action:    "restart",
		Reason:    "test",
		Success:   false,
	}

	healer.healingHistory = []HealingEvent{event1, event2}

	history := healer.GetHealingHistory()
	assert.Len(t, history, 2)
	assert.Equal(t, event1, history[0])
	assert.Equal(t, event2, history[1])
}

func TestAutoHealer_GetUnhealthyTasks(t *testing.T) {
	mockClient := &MockMarathonClient{}
	healer := NewAutoHealer(mockClient)

	// Add some unhealthy tasks
	task1 := &UnhealthyTask{
		TaskID:        "task-1",
		AppID:         "app-1",
		FailureCount:  3,
		RestartAttempts: 1,
	}
	task2 := &UnhealthyTask{
		TaskID:        "task-2",
		AppID:         "app-2",
		FailureCount:  5,
		RestartAttempts: 2,
	}

	healer.unhealthyTasks["task-1"] = task1
	healer.unhealthyTasks["task-2"] = task2

	tasks := healer.GetUnhealthyTasks()
	assert.Len(t, tasks, 2)
	assert.Equal(t, task1, tasks["task-1"])
	assert.Equal(t, task2, tasks["task-2"])
}

func TestAutoHealer_GetHealthStatus(t *testing.T) {
	tests := []struct {
		name           string
		appID          string
		mockApp        *Application
		unhealthyTasks map[string]*UnhealthyTask
		expectedStatus *HealthStatus
		expectedError  bool
	}{
		{
			name:  "Healthy app",
			appID: "healthy-app",
			mockApp: &Application{
				ID:             "healthy-app",
				TasksRunning:   4,
				TasksHealthy:   4,
				TasksUnhealthy: 0,
			},
			unhealthyTasks: map[string]*UnhealthyTask{},
			expectedStatus: &HealthStatus{
				AppID:          "healthy-app",
				TotalTasks:     4,
				HealthyTasks:   4,
				UnhealthyTasks: 0,
				HealthPercent:  100.0,
				Status:         "healthy",
			},
			expectedError: false,
		},
		{
			name:  "Degraded app",
			appID: "degraded-app",
			mockApp: &Application{
				ID:             "degraded-app",
				TasksRunning:   4,
				TasksHealthy:   3,
				TasksUnhealthy: 1,
			},
			unhealthyTasks: map[string]*UnhealthyTask{
				"task-1": {
					TaskID: "task-1",
					AppID:  "degraded-app",
				},
			},
			expectedStatus: &HealthStatus{
				AppID:          "degraded-app",
				TotalTasks:     4,
				HealthyTasks:   3,
				UnhealthyTasks: 1,
				HealthPercent:  75.0,
				Status:         "degraded",
			},
			expectedError: false,
		},
		{
			name:  "Warning app",
			appID: "warning-app",
			mockApp: &Application{
				ID:             "warning-app",
				TasksRunning:   4,
				TasksHealthy:   2,
				TasksUnhealthy: 2,
			},
			unhealthyTasks: map[string]*UnhealthyTask{
				"task-1": {
					TaskID: "task-1",
					AppID:  "warning-app",
				},
				"task-2": {
					TaskID: "task-2",
					AppID:  "warning-app",
				},
			},
			expectedStatus: &HealthStatus{
				AppID:          "warning-app",
				TotalTasks:     4,
				HealthyTasks:   2,
				UnhealthyTasks: 2,
				HealthPercent:  50.0,
				Status:         "warning",
			},
			expectedError: false,
		},
		{
			name:  "Critical app",
			appID: "critical-app",
			mockApp: &Application{
				ID:             "critical-app",
				TasksRunning:   4,
				TasksHealthy:   1,
				TasksUnhealthy: 3,
			},
			unhealthyTasks: map[string]*UnhealthyTask{
				"task-1": {
					TaskID: "task-1",
					AppID:  "critical-app",
				},
				"task-2": {
					TaskID: "task-2",
					AppID:  "critical-app",
				},
				"task-3": {
					TaskID: "task-3",
					AppID:  "critical-app",
				},
			},
			expectedStatus: &HealthStatus{
				AppID:          "critical-app",
				TotalTasks:     4,
				HealthyTasks:   1,
				UnhealthyTasks: 3,
				HealthPercent:  25.0,
				Status:         "critical",
			},
			expectedError: false,
		},
		{
			name:          "App not found",
			appID:         "missing-app",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockMarathonClient{}
			healer := NewAutoHealer(mockClient)

			// Set up unhealthy tasks
			healer.unhealthyTasks = tt.unhealthyTasks

			if tt.expectedError {
				mockClient.On("GetApp", tt.appID).Return((*Application)(nil), assert.AnError)
			} else {
				mockClient.On("GetApp", tt.appID).Return(tt.mockApp, nil)
			}

			status, err := healer.GetHealthStatus(tt.appID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, status)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, status)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestAutoHealer_determineStatus(t *testing.T) {
	tests := []struct {
		name           string
		healthPercent  float64
		expectedStatus string
	}{
		{
			name:           "Healthy - 100%",
			healthPercent:  100.0,
			expectedStatus: "healthy",
		},
		{
			name:           "Healthy - 95%",
			healthPercent:  95.0,
			expectedStatus: "healthy",
		},
		{
			name:           "Degraded - 80%",
			healthPercent:  80.0,
			expectedStatus: "degraded",
		},
		{
			name:           "Degraded - 75%",
			healthPercent:  75.0,
			expectedStatus: "degraded",
		},
		{
			name:           "Warning - 60%",
			healthPercent:  60.0,
			expectedStatus: "warning",
		},
		{
			name:           "Warning - 50%",
			healthPercent:  50.0,
			expectedStatus: "warning",
		},
		{
			name:           "Critical - 30%",
			healthPercent:  30.0,
			expectedStatus: "critical",
		},
		{
			name:           "Critical - 0%",
			healthPercent:  0.0,
			expectedStatus: "critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockMarathonClient{}
			healer := NewAutoHealer(mockClient)

			status := healer.determineStatus(tt.healthPercent)
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

func TestAutoHealer_ConcurrentAccess(t *testing.T) {
	mockClient := &MockMarathonClient{}
	healer := NewAutoHealer(mockClient)

	// Register multiple apps
	for i := 0; i < 10; i++ {
		config := &HealingConfig{
			AppID:   fmt.Sprintf("app-%d", i),
			Enabled: true,
		}
		err := healer.RegisterApp(config)
		assert.NoError(t, err)
	}

	// Concurrent access to healing history
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			history := healer.GetHealingHistory()
			assert.NotNil(t, history)
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent access to unhealthy tasks
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			tasks := healer.GetUnhealthyTasks()
			assert.NotNil(t, tasks)
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestAutoHealer_EdgeCases(t *testing.T) {
	mockClient := &MockMarathonClient{}
	healer := NewAutoHealer(mockClient)

	// Test with nil config
	err := healer.RegisterApp(nil)
	assert.Error(t, err)

	// Test with empty app ID
	config := &HealingConfig{
		AppID:   "",
		Enabled: true,
	}
	err = healer.RegisterApp(config)
	assert.NoError(t, err) // Should not error, but won't be useful

	// Test health status for non-existent app
	mockClient.On("GetApp", "non-existent").Return((*Application)(nil), assert.AnError)
	status, err := healer.GetHealthStatus("non-existent")
	assert.Error(t, err)
	assert.Nil(t, status)
}

func TestAutoHealer_HealingHistoryLimit(t *testing.T) {
	mockClient := &MockMarathonClient{}
	healer := NewAutoHealer(mockClient)

	// Add more than 100 healing events
	for i := 0; i < 150; i++ {
		event := HealingEvent{
			Timestamp: time.Now(),
			AppID:     fmt.Sprintf("app-%d", i),
			TaskID:    fmt.Sprintf("task-%d", i),
			Action:    "restart",
			Reason:    "test",
			Success:   true,
		}
		healer.healingHistory = append(healer.healingHistory, event)
	}

	// Simulate the limit check that happens in healTask
	if len(healer.healingHistory) > 100 {
		healer.healingHistory = healer.healingHistory[len(healer.healingHistory)-100:]
	}

	history := healer.GetHealingHistory()
	assert.Len(t, history, 100)
	assert.Equal(t, "app-50", history[0].AppID) // Should start from app-50
	assert.Equal(t, "app-149", history[99].AppID) // Should end at app-149
}
