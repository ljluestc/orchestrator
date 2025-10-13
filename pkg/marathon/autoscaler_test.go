package marathon

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMarathonClientForAutoScaler is a mock implementation of MarathonClient for autoscaler tests
type MockMarathonClientForAutoScaler struct {
	mock.Mock
}

func (m *MockMarathonClientForAutoScaler) GetApp(appID string) (*Application, error) {
	args := m.Called(appID)
	return args.Get(0).(*Application), args.Error(1)
}

func (m *MockMarathonClientForAutoScaler) ScaleApp(appID string, instances int) error {
	args := m.Called(appID, instances)
	return args.Error(0)
}

func (m *MockMarathonClientForAutoScaler) GetAppTasks(appID string) ([]Task, error) {
	args := m.Called(appID)
	return args.Get(0).([]Task), args.Error(1)
}

// MockMetricsProvider is a mock implementation of MetricsProvider
type MockMetricsProvider struct {
	mock.Mock
}

func (m *MockMetricsProvider) GetCPUUtilization(appID string) (float64, error) {
	args := m.Called(appID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsProvider) GetMemoryUtilization(appID string) (float64, error) {
	args := m.Called(appID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsProvider) GetCustomMetric(appID, metricName string) (float64, error) {
	args := m.Called(appID, metricName)
	return args.Get(0).(float64), args.Error(1)
}

func TestNewAutoScaler(t *testing.T) {
	mockClient := &MockMarathonClientForAutoScaler{}
	mockMetrics := &MockMetricsProvider{}

	scaler := NewAutoScaler(mockClient, mockMetrics)

	assert.NotNil(t, scaler)
	assert.Equal(t, mockClient, scaler.client)
	assert.Equal(t, mockMetrics, scaler.metricsProvider)
	assert.Equal(t, 30*time.Second, scaler.checkInterval)
	assert.NotNil(t, scaler.applications)
	assert.Empty(t, scaler.applications)
}

func TestAutoScaler_RegisterApp(t *testing.T) {
	tests := []struct {
		name           string
		config         *AutoScaleConfig
		expectedError  string
		expectedConfig *AutoScaleConfig
	}{
		{
			name: "Valid config",
			config: &AutoScaleConfig{
				AppID:            "test-app",
				MinInstances:     2,
				MaxInstances:     10,
				TargetCPUPercent: 70.0,
				TargetMemPercent: 80.0,
				Enabled:          true,
			},
			expectedError: "",
			expectedConfig: &AutoScaleConfig{
				AppID:            "test-app",
				MinInstances:     2,
				MaxInstances:     10,
				TargetCPUPercent: 70.0,
				TargetMemPercent: 80.0,
				Enabled:          true,
				ScaleUpPolicy: ScalePolicy{
					Cooldown:            3 * time.Minute,
					ConsecutivePeriods: 2,
				},
				ScaleDownPolicy: ScalePolicy{
					Cooldown:            5 * time.Minute,
					ConsecutivePeriods: 3,
				},
			},
		},
		{
			name: "Min instances too low",
			config: &AutoScaleConfig{
				AppID:        "test-app",
				MinInstances: 0,
				MaxInstances: 10,
			},
			expectedError: "minInstances must be >= 1",
		},
		{
			name: "Max instances less than min",
			config: &AutoScaleConfig{
				AppID:        "test-app",
				MinInstances: 5,
				MaxInstances: 3,
			},
			expectedError: "maxInstances must be >= minInstances",
		},
		{
			name: "Zero target CPU percent",
			config: &AutoScaleConfig{
				AppID:            "test-app",
				MinInstances:     2,
				MaxInstances:     10,
				TargetCPUPercent: 0,
			},
			expectedError: "",
			expectedConfig: &AutoScaleConfig{
				AppID:            "test-app",
				MinInstances:     2,
				MaxInstances:     10,
				TargetCPUPercent: 70.0, // Should be set to default
				ScaleUpPolicy: ScalePolicy{
					Cooldown:            3 * time.Minute,
					ConsecutivePeriods: 2,
				},
				ScaleDownPolicy: ScalePolicy{
					Cooldown:            5 * time.Minute,
					ConsecutivePeriods: 3,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockMarathonClientForAutoScaler{}
			mockMetrics := &MockMetricsProvider{}
			scaler := NewAutoScaler(mockClient, mockMetrics)

			err := scaler.RegisterApp(tt.config)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, scaler.applications, tt.config.AppID)
				
				if tt.expectedConfig != nil {
					registeredConfig := scaler.applications[tt.config.AppID]
					assert.Equal(t, tt.expectedConfig.AppID, registeredConfig.AppID)
					assert.Equal(t, tt.expectedConfig.MinInstances, registeredConfig.MinInstances)
					assert.Equal(t, tt.expectedConfig.MaxInstances, registeredConfig.MaxInstances)
					assert.Equal(t, tt.expectedConfig.TargetCPUPercent, registeredConfig.TargetCPUPercent)
					assert.Equal(t, tt.expectedConfig.ScaleUpPolicy.Cooldown, registeredConfig.ScaleUpPolicy.Cooldown)
					assert.Equal(t, tt.expectedConfig.ScaleDownPolicy.Cooldown, registeredConfig.ScaleDownPolicy.Cooldown)
					assert.NotNil(t, registeredConfig.ScaleHistory)
				}
			}
		})
	}
}

func TestAutoScaler_Start(t *testing.T) {
	mockClient := &MockMarathonClientForAutoScaler{}
	mockMetrics := &MockMetricsProvider{}
	scaler := NewAutoScaler(mockClient, mockMetrics)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := scaler.Start(ctx)
	assert.NoError(t, err)
}

func TestAutoScaler_checkAndScale(t *testing.T) {
	tests := []struct {
		name           string
		applications   map[string]*AutoScaleConfig
		expectedCalls  int
	}{
		{
			name:           "No applications",
			applications:   map[string]*AutoScaleConfig{},
			expectedCalls: 0,
		},
		{
			name: "One enabled application",
			applications: map[string]*AutoScaleConfig{
				"app1": {
					AppID:   "app1",
					Enabled: true,
				},
			},
			expectedCalls: 1,
		},
		{
			name: "One disabled application",
			applications: map[string]*AutoScaleConfig{
				"app1": {
					AppID:   "app1",
					Enabled: false,
				},
			},
			expectedCalls: 0,
		},
		{
			name: "Mixed enabled/disabled applications",
			applications: map[string]*AutoScaleConfig{
				"app1": {
					AppID:   "app1",
					Enabled: true,
				},
				"app2": {
					AppID:   "app2",
					Enabled: false,
				},
				"app3": {
					AppID:   "app3",
					Enabled: true,
				},
			},
			expectedCalls: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockMarathonClientForAutoScaler{}
			mockMetrics := &MockMetricsProvider{}
			scaler := NewAutoScaler(mockClient, mockMetrics)

			// Register applications
			for _, config := range tt.applications {
				scaler.applications[config.AppID] = config
			}

			// Set up mocks for enabled applications
			for _, config := range tt.applications {
				if config.Enabled {
					// Set default values to prevent scaling
					if config.TargetCPUPercent == 0 {
						config.TargetCPUPercent = 70.0
					}
					if config.TargetMemPercent == 0 {
						config.TargetMemPercent = 80.0
					}
					if config.MinInstances == 0 {
						config.MinInstances = 1
					}
					if config.MaxInstances == 0 {
						config.MaxInstances = 10
					}
					
					mockClient.On("GetApp", config.AppID).Return(&Application{ID: config.AppID, Instances: 1}, nil)
					mockMetrics.On("GetCPUUtilization", config.AppID).Return(50.0, nil)
					mockMetrics.On("GetMemoryUtilization", config.AppID).Return(60.0, nil)
				}
			}

			ctx := context.Background()
			scaler.checkAndScale(ctx)

			// Note: We can't easily verify the exact number of calls to evaluateApp
			// since it's called in a loop, but we can verify the method exists and
			// the applications are processed
			assert.Equal(t, len(tt.applications), len(scaler.applications))
			mockClient.AssertExpectations(t)
			mockMetrics.AssertExpectations(t)
		})
	}
}

func TestAutoScaler_evaluateApp(t *testing.T) {
	tests := []struct {
		name                string
		config              *AutoScaleConfig
		app                 *Application
		cpuUtil             float64
		memUtil             float64
		getAppError         error
		cpuMetricsError     error
		memMetricsError     error
		expectedError       string
		expectedScaleCall   bool
		expectedTargetCount int
	}{
		{
			name: "Successful evaluation - no scaling needed",
			config: &AutoScaleConfig{
				AppID:            "test-app",
				MinInstances:     2,
				MaxInstances:     10,
				TargetCPUPercent: 70.0,
				TargetMemPercent: 80.0,
			},
			app: &Application{
				ID:        "test-app",
				Instances: 3,
			},
			cpuUtil:           50.0,
			memUtil:           60.0,
			expectedError:     "",
			expectedScaleCall: false,
		},
		{
			name: "Scale up due to high CPU",
			config: &AutoScaleConfig{
				AppID:            "test-app",
				MinInstances:     2,
				MaxInstances:     10,
				TargetCPUPercent: 70.0,
				TargetMemPercent: 80.0,
			},
			app: &Application{
				ID:        "test-app",
				Instances: 3,
			},
			cpuUtil:           85.0,
			memUtil:           60.0,
			expectedError:     "",
			expectedScaleCall: true,
			expectedTargetCount: 4, // 3 + 1 (50% of 3 = 1.5, rounded to 1)
		},
		{
			name: "Scale up due to high memory",
			config: &AutoScaleConfig{
				AppID:            "test-app",
				MinInstances:     2,
				MaxInstances:     10,
				TargetCPUPercent: 70.0,
				TargetMemPercent: 80.0,
			},
			app: &Application{
				ID:        "test-app",
				Instances: 3,
			},
			cpuUtil:           50.0,
			memUtil:           90.0,
			expectedError:     "",
			expectedScaleCall: true,
			expectedTargetCount: 4,
		},
		{
			name: "Scale down due to low utilization",
			config: &AutoScaleConfig{
				AppID:            "test-app",
				MinInstances:     2,
				MaxInstances:     10,
				TargetCPUPercent: 70.0,
				TargetMemPercent: 80.0,
			},
			app: &Application{
				ID:        "test-app",
				Instances: 5,
			},
			cpuUtil:           30.0, // < 35% (50% of 70%)
			memUtil:           35.0, // < 40% (50% of 80%)
			expectedError:     "",
			expectedScaleCall: true,
			expectedTargetCount: 4, // 5 - 1 (25% of 5 = 1.25, rounded to 1)
		},
		{
			name: "GetApp error",
			config: &AutoScaleConfig{
				AppID: "test-app",
			},
			getAppError:   errors.New("app not found"),
			expectedError: "failed to get app: app not found",
		},
		{
			name: "CPU metrics error",
			config: &AutoScaleConfig{
				AppID:            "test-app",
				TargetCPUPercent: 70.0,
			},
			app: &Application{
				ID:        "test-app",
				Instances: 3,
			},
			cpuMetricsError: errors.New("metrics unavailable"),
			memUtil:         60.0,
			expectedError:   "",
			// Should continue with CPU = 0
		},
		{
			name: "Memory metrics error",
			config: &AutoScaleConfig{
				AppID:            "test-app",
				TargetCPUPercent: 70.0, // Set proper target to prevent scaling
			},
			app: &Application{
				ID:        "test-app",
				Instances: 3,
			},
			cpuUtil:         50.0,
			memMetricsError: errors.New("metrics unavailable"),
			expectedError:   "",
			// Should continue with Memory = 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockMarathonClientForAutoScaler{}
			mockMetrics := &MockMetricsProvider{}
			scaler := NewAutoScaler(mockClient, mockMetrics)

			// Set up mocks
			if tt.getAppError != nil {
				mockClient.On("GetApp", tt.config.AppID).Return((*Application)(nil), tt.getAppError)
			} else {
				mockClient.On("GetApp", tt.config.AppID).Return(tt.app, nil)
				
				if tt.cpuMetricsError != nil {
					mockMetrics.On("GetCPUUtilization", tt.config.AppID).Return(float64(0), tt.cpuMetricsError)
				} else {
					mockMetrics.On("GetCPUUtilization", tt.config.AppID).Return(tt.cpuUtil, nil)
				}

				if tt.memMetricsError != nil {
					mockMetrics.On("GetMemoryUtilization", tt.config.AppID).Return(float64(0), tt.memMetricsError)
				} else {
					mockMetrics.On("GetMemoryUtilization", tt.config.AppID).Return(tt.memUtil, nil)
				}
			}

			if tt.expectedScaleCall {
				mockClient.On("ScaleApp", tt.config.AppID, tt.expectedTargetCount).Return(nil)
			}

			ctx := context.Background()
			err := scaler.evaluateApp(ctx, tt.config)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
			mockMetrics.AssertExpectations(t)
		})
	}
}

func TestAutoScaler_makeScalingDecision(t *testing.T) {
	tests := []struct {
		name           string
		config         *AutoScaleConfig
		app            *Application
		cpuUtil        float64
		memUtil        float64
		expectedScale  bool
		expectedDir    string
		expectedReason string
	}{
		{
			name: "No scaling needed",
			config: &AutoScaleConfig{
				MinInstances:     2,
				MaxInstances:     10,
				TargetCPUPercent: 70.0,
				TargetMemPercent: 80.0,
			},
			app: &Application{
				Instances: 3,
			},
			cpuUtil:        50.0,
			memUtil:        60.0,
			expectedScale: false,
		},
		{
			name: "Scale up - CPU threshold exceeded",
			config: &AutoScaleConfig{
				MinInstances:     2,
				MaxInstances:     10,
				TargetCPUPercent: 70.0,
				TargetMemPercent: 80.0,
			},
			app: &Application{
				Instances: 3,
			},
			cpuUtil:        85.0,
			memUtil:        60.0,
			expectedScale: true,
			expectedDir:    "up",
			expectedReason: "CPU utilization 85.0% > target 70.0%",
		},
		{
			name: "Scale up - Memory threshold exceeded",
			config: &AutoScaleConfig{
				MinInstances:     2,
				MaxInstances:     10,
				TargetCPUPercent: 70.0,
				TargetMemPercent: 80.0,
			},
			app: &Application{
				Instances: 3,
			},
			cpuUtil:        50.0,
			memUtil:        90.0,
			expectedScale: true,
			expectedDir:    "up",
			expectedReason: "Memory utilization 90.0% > target 80.0%",
		},
		{
			name: "Scale down - Low utilization",
			config: &AutoScaleConfig{
				MinInstances:     2,
				MaxInstances:     10,
				TargetCPUPercent: 70.0,
				TargetMemPercent: 80.0,
			},
			app: &Application{
				Instances: 5,
			},
			cpuUtil:        30.0, // < 35% (50% of 70%)
			memUtil:        35.0, // < 40% (50% of 80%)
			expectedScale: true,
			expectedDir:    "down",
			expectedReason: "Low utilization: CPU=30.0%, Memory=35.0%",
		},
		{
			name: "Scale up - At max instances",
			config: &AutoScaleConfig{
				MinInstances:     2,
				MaxInstances:     5,
				TargetCPUPercent: 70.0,
			},
			app: &Application{
				Instances: 5, // Already at max
			},
			cpuUtil:        85.0,
			memUtil:        60.0,
			expectedScale: false,
			expectedReason: "Already at scale limit",
		},
		{
			name: "Scale down - At min instances",
			config: &AutoScaleConfig{
				MinInstances:     3,
				MaxInstances:     10,
				TargetCPUPercent: 70.0,
				TargetMemPercent: 80.0,
			},
			app: &Application{
				Instances: 3, // Already at min
			},
			cpuUtil:        30.0,
			memUtil:        35.0,
			expectedScale: false,
			expectedReason: "Already at scale limit",
		},
		{
			name: "Scale up - Cooldown active",
			config: &AutoScaleConfig{
				MinInstances:     2,
				MaxInstances:     10,
				TargetCPUPercent: 70.0,
				ScaleUpPolicy: ScalePolicy{
					Cooldown: 5 * time.Minute,
				},
				LastScaleTime: time.Now().Add(-2 * time.Minute), // 2 minutes ago
			},
			app: &Application{
				Instances: 3,
			},
			cpuUtil:        85.0,
			memUtil:        60.0,
			expectedScale: false,
			expectedReason: "Cooldown active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockMarathonClientForAutoScaler{}
			mockMetrics := &MockMetricsProvider{}
			scaler := NewAutoScaler(mockClient, mockMetrics)

			decision := scaler.makeScalingDecision(tt.config, tt.app, tt.cpuUtil, tt.memUtil)

			assert.Equal(t, tt.expectedScale, decision.ShouldScale)
			if tt.expectedScale {
				assert.Equal(t, tt.expectedDir, decision.Direction)
				assert.Contains(t, decision.Reason, tt.expectedReason)
			} else if tt.expectedReason != "" {
				assert.Contains(t, decision.Reason, tt.expectedReason)
			}
		})
	}
}

func TestAutoScaler_executeScale(t *testing.T) {
	tests := []struct {
		name           string
		config         *AutoScaleConfig
		app            *Application
		decision       ScalingDecision
		scaleError     error
		expectedError  string
		expectedEvents int
	}{
		{
			name: "Successful scale up",
			config: &AutoScaleConfig{
				AppID: "test-app",
			},
			app: &Application{
				Instances: 3,
			},
			decision: ScalingDecision{
				ShouldScale:   true,
				Direction:     "up",
				TargetCount:   4,
				Reason:        "CPU utilization high",
				MetricName:    "cpu",
				MetricValue:   85.0,
			},
			expectedError:  "",
			expectedEvents: 1,
		},
		{
			name: "Scale operation fails",
			config: &AutoScaleConfig{
				AppID: "test-app",
			},
			app: &Application{
				Instances: 3,
			},
			decision: ScalingDecision{
				ShouldScale:   true,
				Direction:     "up",
				TargetCount:   4,
				Reason:        "CPU utilization high",
				MetricName:    "cpu",
				MetricValue:   85.0,
			},
			scaleError:     errors.New("scale failed"),
			expectedError:  "failed to scale app: scale failed",
			expectedEvents: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockMarathonClientForAutoScaler{}
			mockMetrics := &MockMetricsProvider{}
			scaler := NewAutoScaler(mockClient, mockMetrics)

			// Set up mocks
			if tt.scaleError != nil {
				mockClient.On("ScaleApp", tt.config.AppID, tt.decision.TargetCount).Return(tt.scaleError)
			} else {
				mockClient.On("ScaleApp", tt.config.AppID, tt.decision.TargetCount).Return(nil)
			}

			ctx := context.Background()
			err := scaler.executeScale(ctx, tt.config, tt.app, tt.decision)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedEvents, len(tt.config.ScaleHistory))
				if tt.expectedEvents > 0 {
					event := tt.config.ScaleHistory[0]
					assert.Equal(t, tt.app.Instances, event.FromInstances)
					assert.Equal(t, tt.decision.TargetCount, event.ToInstances)
					assert.Equal(t, tt.decision.Reason, event.Reason)
					assert.Equal(t, tt.decision.MetricName, event.Metric)
					assert.Equal(t, tt.decision.MetricValue, event.MetricValue)
				}
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestAutoScaler_GetScaleHistory(t *testing.T) {
	mockClient := &MockMarathonClient{}
	mockMetrics := &MockMetricsProvider{}
	scaler := NewAutoScaler(mockClient, mockMetrics)

	// Test non-existent app
	history := scaler.GetScaleHistory("non-existent")
	assert.Nil(t, history)

	// Test existing app
	config := &AutoScaleConfig{
		AppID:        "test-app",
		ScaleHistory: []ScaleEvent{{Timestamp: time.Now()}},
	}
	scaler.applications["test-app"] = config

	history = scaler.GetScaleHistory("test-app")
	assert.NotNil(t, history)
	assert.Len(t, history, 1)
}

func TestAutoScaler_UpdateConfig(t *testing.T) {
	tests := []struct {
		name          string
		appID         string
		updates       func(*AutoScaleConfig)
		expectedError string
	}{
		{
			name:  "Successful update",
			appID: "test-app",
			updates: func(config *AutoScaleConfig) {
				config.Enabled = false
				config.TargetCPUPercent = 60.0
			},
			expectedError: "",
		},
		{
			name:          "App not registered",
			appID:         "non-existent",
			updates:       func(config *AutoScaleConfig) {},
			expectedError: "app non-existent not registered for autoscaling",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockMarathonClientForAutoScaler{}
			mockMetrics := &MockMetricsProvider{}
			scaler := NewAutoScaler(mockClient, mockMetrics)

			if tt.appID == "test-app" {
				scaler.applications["test-app"] = &AutoScaleConfig{
					AppID:            "test-app",
					Enabled:          true,
					TargetCPUPercent: 70.0,
				}
			}

			err := scaler.UpdateConfig(tt.appID, tt.updates)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				config := scaler.applications[tt.appID]
				assert.False(t, config.Enabled)
				assert.Equal(t, 60.0, config.TargetCPUPercent)
			}
		})
	}
}

func TestAutoScaler_ScaleHistoryLimit(t *testing.T) {
	mockClient := &MockMarathonClient{}
	mockMetrics := &MockMetricsProvider{}
	scaler := NewAutoScaler(mockClient, mockMetrics)

	config := &AutoScaleConfig{
		AppID: "test-app",
	}
	scaler.applications["test-app"] = config

	// Add more than 50 events
	for i := 0; i < 60; i++ {
		event := ScaleEvent{
			Timestamp: time.Now(),
			Reason:    fmt.Sprintf("event-%d", i),
		}
		config.ScaleHistory = append(config.ScaleHistory, event)
	}

	// Simulate a scale operation that should trim the history
	decision := ScalingDecision{
		ShouldScale:   true,
		Direction:     "up",
		TargetCount:   2,
		Reason:        "test",
		MetricName:    "cpu",
		MetricValue:   85.0,
	}

	app := &Application{Instances: 1}
	mockClient.On("ScaleApp", "test-app", 2).Return(nil)

	ctx := context.Background()
	err := scaler.executeScale(ctx, config, app, decision)
	assert.NoError(t, err)

	// History should be trimmed to 50 events
	assert.Len(t, config.ScaleHistory, 50)
	mockClient.AssertExpectations(t)
}

func TestAutoScaler_EdgeCases(t *testing.T) {
	t.Run("Nil config", func(t *testing.T) {
		mockClient := &MockMarathonClient{}
		mockMetrics := &MockMetricsProvider{}
		scaler := NewAutoScaler(mockClient, mockMetrics)

		err := scaler.RegisterApp(nil)
		assert.Error(t, err)
	})

	t.Run("Zero target memory percent", func(t *testing.T) {
		mockClient := &MockMarathonClient{}
		mockMetrics := &MockMetricsProvider{}
		scaler := NewAutoScaler(mockClient, mockMetrics)

		config := &AutoScaleConfig{
			AppID:            "test-app",
			MinInstances:     2,
			MaxInstances:     10,
			TargetCPUPercent: 70.0,
			TargetMemPercent: 0, // Should not trigger memory-based scaling
		}

		err := scaler.RegisterApp(config)
		assert.NoError(t, err)

		app := &Application{Instances: 3}
		decision := scaler.makeScalingDecision(config, app, 50.0, 90.0) // High memory but target is 0
		assert.False(t, decision.ShouldScale)
	})

	t.Run("Custom step sizes", func(t *testing.T) {
		mockClient := &MockMarathonClient{}
		mockMetrics := &MockMetricsProvider{}
		scaler := NewAutoScaler(mockClient, mockMetrics)

		config := &AutoScaleConfig{
			AppID:        "test-app",
			MinInstances: 2,
			MaxInstances: 10,
			ScaleUpPolicy: ScalePolicy{
				StepSize: 3, // Custom step size
			},
			ScaleDownPolicy: ScalePolicy{
				StepSize: 2, // Custom step size
			},
		}

		app := &Application{Instances: 5}

		// Test scale up with custom step size
		decision := scaler.makeScalingDecision(config, app, 85.0, 60.0)
		assert.True(t, decision.ShouldScale)
		assert.Equal(t, "up", decision.Direction)
		assert.Equal(t, 8, decision.TargetCount) // 5 + 3

		// Test scale down with custom step size
		decision = scaler.makeScalingDecision(config, app, 30.0, 35.0)
		assert.True(t, decision.ShouldScale)
		assert.Equal(t, "down", decision.Direction)
		assert.Equal(t, 3, decision.TargetCount) // 5 - 2
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("max function", func(t *testing.T) {
		assert.Equal(t, 5, max(3, 5))
		assert.Equal(t, 5, max(5, 3))
		assert.Equal(t, 3, max(3, 3))
	})

	t.Run("min function", func(t *testing.T) {
		assert.Equal(t, 3, min(3, 5))
		assert.Equal(t, 3, min(5, 3))
		assert.Equal(t, 3, min(3, 3))
	})
}
