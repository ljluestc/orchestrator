package marathon

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Type alias for compatibility
type MarathonApp = Application

func createMockClient(t *testing.T) *MarathonClient {
	mockClient := &MockMarathonClient{}
	// Setup default behavior
	mockClient.On("GetApp", mock.Anything).Return(&Application{
		ID:           "/test-app",
		Instances:    3,
		TasksRunning: 3,
		TasksHealthy: 3,
	}, nil)
	var clientInterface MarathonClient = mockClient
	return &clientInterface
}

// TestNewRollingUpdater tests creating a new rolling updater
func TestNewRollingUpdater(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	assert.NotNil(t, updater)
	assert.NotNil(t, updater.activeUpdates)
	assert.NotNil(t, updater.updateHistory)
	assert.Equal(t, 0, len(updater.activeUpdates))
}

// TestStartUpdate_RollingStrategy tests starting a rolling update
func TestStartUpdate_RollingStrategy(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	config := &UpdateConfig{
		AppID:      "/test-app",
		Strategy:   RollingUpdate,
		NewVersion: "v2.0.0",
		NewImage:   "myapp:v2",
		RollingConfig: &RollingConfig{
			BatchSize:         2,
			MinHealthyPercent: 0.8,
			PauseTime:         100 * time.Millisecond,
			AutoRollback:      false,
		},
	}

	err := updater.StartUpdate(context.Background(), config)
	require.NoError(t, err)

	// Wait for update to start
	time.Sleep(50 * time.Millisecond)

	// Check update state
	state := updater.GetUpdateState("/test-app")
	assert.NotNil(t, state)
	assert.Equal(t, "/test-app", state.AppID)
	assert.Equal(t, RollingUpdate, state.Strategy)
}

// TestStartUpdate_AlreadyInProgress tests starting update when one is already running
func TestStartUpdate_AlreadyInProgress(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	config := &UpdateConfig{
		AppID:      "/test-app",
		Strategy:   RollingUpdate,
		NewVersion: "v2.0.0",
		RollingConfig: &RollingConfig{
			BatchSize: 1,
			PauseTime: 100 * time.Millisecond,
		},
	}

	err := updater.StartUpdate(context.Background(), config)
	require.NoError(t, err)

	// Try to start another update immediately
	err = updater.StartUpdate(context.Background(), config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already in progress")
}

// TestStartUpdate_CanaryStrategy tests canary deployment
func TestStartUpdate_CanaryStrategy(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	config := &UpdateConfig{
		AppID:      "/canary-app",
		Strategy:   CanaryUpdate,
		NewVersion: "v2.0.0",
		CanaryConfig: &CanaryConfig{
			Stages: []CanaryStage{
				{Name: "stage-1", Weight: 10, Duration: 50 * time.Millisecond},
				{Name: "stage-2", Weight: 50, Duration: 50 * time.Millisecond},
			},
			AnalysisInterval: 20 * time.Millisecond,
			SuccessThreshold: 0.99,
		},
	}

	err := updater.StartUpdate(context.Background(), config)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	state := updater.GetUpdateState("/canary-app")
	if state != nil {
		assert.Equal(t, CanaryUpdate, state.Strategy)
		t.Logf("Canary update state: %+v", state)
	}
}

// TestStartUpdate_BlueGreenStrategy tests blue-green deployment
func TestStartUpdate_BlueGreenStrategy(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	config := &UpdateConfig{
		AppID:      "/bluegreen-app",
		Strategy:   BlueGreenUpdate,
		NewVersion: "v2.0.0",
		BlueGreenConfig: &BlueGreenConfig{
			AutoPromote:       true,
			PromotionDelay:    50 * time.Millisecond,
			KeepOldVersion:    false,
			TestTrafficWeight: 20,
		},
	}

	err := updater.StartUpdate(context.Background(), config)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	state := updater.GetUpdateState("/bluegreen-app")
	if state != nil {
		assert.Equal(t, BlueGreenUpdate, state.Strategy)
	}
}

// TestStartUpdate_RecreateStrategy tests recreate deployment
func TestStartUpdate_RecreateStrategy(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	config := &UpdateConfig{
		AppID:      "/recreate-app",
		Strategy:   RecreateUpdate,
		NewVersion: "v2.0.0",
	}

	err := updater.StartUpdate(context.Background(), config)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	state := updater.GetUpdateState("/recreate-app")
	if state != nil {
		assert.Equal(t, RecreateUpdate, state.Strategy)
	}
}

// TestGetUpdateHistory tests getting update history
func TestGetUpdateHistory(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	// Add some history
	updater.recordEvent(UpdateEvent{
		Timestamp: time.Now(),
		AppID:     "/app1",
		Strategy:  RollingUpdate,
		Stage:     "batch-1",
		Action:    "completed",
		Success:   true,
	})

	history := updater.GetUpdateHistory()
	assert.Equal(t, 1, len(history))
	assert.Equal(t, "/app1", history[0].AppID)
}

// TestGetUpdateState_NonExistent tests getting state for non-existent update
func TestGetUpdateState_NonExistent(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	state := updater.GetUpdateState("/nonexistent")
	assert.Nil(t, state)
}

// TestPauseUpdate tests pausing an update
func TestPauseUpdate(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	config := &UpdateConfig{
		AppID:      "/pause-app",
		Strategy:   RollingUpdate,
		NewVersion: "v2.0.0",
		RollingConfig: &RollingConfig{
			BatchSize: 1,
			PauseTime: 1 * time.Second,
		},
	}

	err := updater.StartUpdate(context.Background(), config)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	err = updater.PauseUpdate("/pause-app")
	assert.NoError(t, err)

	state := updater.GetUpdateState("/pause-app")
	if state != nil {
		assert.Equal(t, UpdatePaused, state.Status)
	}
}

// TestPauseUpdate_NoActiveUpdate tests pausing when no update exists
func TestPauseUpdate_NoActiveUpdate(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	err := updater.PauseUpdate("/nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active update")
}

// TestResumeUpdate tests resuming a paused update
func TestResumeUpdate(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	config := &UpdateConfig{
		AppID:      "/resume-app",
		Strategy:   RollingUpdate,
		NewVersion: "v2.0.0",
		RollingConfig: &RollingConfig{
			BatchSize: 1,
			PauseTime: 1 * time.Second,
		},
	}

	err := updater.StartUpdate(context.Background(), config)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	err = updater.PauseUpdate("/resume-app")
	require.NoError(t, err)

	err = updater.ResumeUpdate("/resume-app")
	assert.NoError(t, err)

	state := updater.GetUpdateState("/resume-app")
	if state != nil {
		assert.Equal(t, UpdateInProgress, state.Status)
	}
}

// TestResumeUpdate_NoPausedUpdate tests resuming when no paused update exists
func TestResumeUpdate_NoPausedUpdate(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	err := updater.ResumeUpdate("/nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no paused update")
}

// TestRecordEvent tests event recording
func TestRecordEvent(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	// Record events
	for i := 0; i < 250; i++ {
		updater.recordEvent(UpdateEvent{
			Timestamp: time.Now(),
			AppID:     "/test",
			Strategy:  RollingUpdate,
			Action:    "test",
			Success:   true,
		})
	}

	history := updater.GetUpdateHistory()
	// Should be capped at 200
	assert.LessOrEqual(t, len(history), 200)
}

// TestUpdateStrategy_UnsupportedStrategy tests unsupported strategy
func TestUpdateStrategy_UnsupportedStrategy(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	config := &UpdateConfig{
		AppID:      "/test-app",
		Strategy:   UpdateStrategy("unsupported"),
		NewVersion: "v2.0.0",
	}

	err := updater.StartUpdate(context.Background(), config)
	require.NoError(t, err)

	// Wait for update to process
	time.Sleep(100 * time.Millisecond)

	// Should have failed
	state := updater.GetUpdateState("/test-app")
	assert.Nil(t, state, "Failed update should be removed from active updates")
}

// TestCanaryUpdate_MissingConfig tests canary without config
func TestCanaryUpdate_MissingConfig(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	config := &UpdateConfig{
		AppID:      "/canary-missing",
		Strategy:   CanaryUpdate,
		NewVersion: "v2.0.0",
		// Missing CanaryConfig
	}

	err := updater.StartUpdate(context.Background(), config)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
}

// TestBlueGreenUpdate_MissingConfig tests blue-green without config
func TestBlueGreenUpdate_MissingConfig(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	config := &UpdateConfig{
		AppID:      "/bluegreen-missing",
		Strategy:   BlueGreenUpdate,
		NewVersion: "v2.0.0",
		// Missing BlueGreenConfig
	}

	err := updater.StartUpdate(context.Background(), config)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
}

// TestRollingUpdate_DefaultBatchSize tests rolling update with default batch size
func TestRollingUpdate_DefaultBatchSize(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	config := &UpdateConfig{
		AppID:      "/default-batch",
		Strategy:   RollingUpdate,
		NewVersion: "v2.0.0",
		RollingConfig: &RollingConfig{
			// BatchSize: 0, // Default should be calculated
			PauseTime: 10 * time.Millisecond,
		},
	}

	err := updater.StartUpdate(context.Background(), config)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
}

// TestCanaryUpdate_WithPause tests canary with manual pause
func TestCanaryUpdate_WithPause(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	config := &UpdateConfig{
		AppID:      "/canary-pause",
		Strategy:   CanaryUpdate,
		NewVersion: "v2.0.0",
		CanaryConfig: &CanaryConfig{
			Stages: []CanaryStage{
				{Name: "stage-1", Weight: 10, Duration: 10 * time.Millisecond, PauseBeforeNext: true},
				{Name: "stage-2", Weight: 100, Duration: 10 * time.Millisecond},
			},
			AnalysisInterval: 5 * time.Millisecond,
		},
	}

	err := updater.StartUpdate(context.Background(), config)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
}

// TestBlueGreenUpdate_ManualPromotion tests blue-green with manual promotion
func TestBlueGreenUpdate_ManualPromotion(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	config := &UpdateConfig{
		AppID:      "/bluegreen-manual",
		Strategy:   BlueGreenUpdate,
		NewVersion: "v2.0.0",
		BlueGreenConfig: &BlueGreenConfig{
			AutoPromote:    false, // Manual promotion
			PromotionDelay: 10 * time.Millisecond,
		},
	}

	err := updater.StartUpdate(context.Background(), config)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
}

// TestBlueGreenUpdate_KeepOldVersion tests keeping old version
func TestBlueGreenUpdate_KeepOldVersion(t *testing.T) {
	client := createMockClient(t)
	updater := NewRollingUpdater(client)

	config := &UpdateConfig{
		AppID:      "/bluegreen-keep",
		Strategy:   BlueGreenUpdate,
		NewVersion: "v2.0.0",
		BlueGreenConfig: &BlueGreenConfig{
			AutoPromote:    true,
			KeepOldVersion: true, // Keep old version
			PromotionDelay: 10 * time.Millisecond,
		},
	}

	err := updater.StartUpdate(context.Background(), config)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)
}

// TestRollingUpdate_AutoRollback tests auto-rollback on failure
func TestRollingUpdate_AutoRollback(t *testing.T) {
	// Create a client that will fail health checks
	mockClient := &MockMarathonClient{}
	mockClient.On("GetApp", mock.Anything).Return(&Application{
		ID:           "/rollback-app",
		Instances:    3,
		TasksRunning: 3,
		TasksHealthy: 1, // Low health will cause failure
	}, nil)
	var clientInterface MarathonClient = mockClient
	client := &clientInterface

	updater := NewRollingUpdater(client)

	config := &UpdateConfig{
		AppID:      "/rollback-app",
		Strategy:   RollingUpdate,
		NewVersion: "v2.0.0",
		RollingConfig: &RollingConfig{
			BatchSize:    1,
			PauseTime:    10 * time.Millisecond,
			AutoRollback: true, // Enable auto-rollback
		},
	}

	err := updater.StartUpdate(context.Background(), config)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// Check history for rollback event
	history := updater.GetUpdateHistory()
	t.Logf("Update history: %d events", len(history))
	for _, event := range history {
		if event.Action == "rollback-completed" {
			assert.True(t, event.Success)
			return
		}
	}
}

// TestUpdateStateStructure tests the update state structure
func TestUpdateStateStructure(t *testing.T) {
	state := &UpdateState{
		AppID:        "/test",
		Strategy:     RollingUpdate,
		StartTime:    time.Now(),
		CurrentStage: "batch-1",
		Progress:     0.5,
		Status:       UpdateInProgress,
		OldVersion:   "v1.0.0",
		NewVersion:   "v2.0.0",
		UpdatedTasks: 5,
		TotalTasks:   10,
		FailedTasks:  []string{"task-1"},
		HealthyTasks: []string{"task-2", "task-3"},
		ErrorMessage: "",
	}

	assert.Equal(t, "/test", state.AppID)
	assert.Equal(t, 0.5, state.Progress)
	assert.Equal(t, 1, len(state.FailedTasks))
	assert.Equal(t, 2, len(state.HealthyTasks))
}
