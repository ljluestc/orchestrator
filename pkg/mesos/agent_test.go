package mesos

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewAgent tests creating a new agent
func TestNewAgent(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")

	assert.NotNil(t, agent)
	assert.Equal(t, "agent-1", agent.ID)
	assert.Equal(t, "localhost", agent.Hostname)
	assert.Equal(t, 5051, agent.Port)
	assert.Equal(t, "http://master:5050", agent.MasterURL)
	assert.Equal(t, "inactive", agent.Status)
	assert.NotNil(t, agent.Resources)
	assert.NotNil(t, agent.Tasks)
	assert.NotNil(t, agent.Executors)
	assert.Equal(t, 4.0, agent.Resources.CPUs)
	assert.Equal(t, 8192.0, agent.Resources.Memory)
	assert.Equal(t, 100000.0, agent.Resources.Disk)
}

// TestAgent_RegisterWithMaster tests agent registration
func TestAgent_RegisterWithMaster(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")

	err := agent.registerWithMaster()
	require.NoError(t, err)

	assert.Equal(t, "active", agent.Status)
	assert.False(t, agent.LastSeen.IsZero())
}

// TestAgent_LaunchTask tests launching a task
func TestAgent_LaunchTask(t *testing.T) {
	tests := []struct {
		name          string
		task          *Task
		expectedError bool
		errorContains string
	}{
		{
			name: "Successful task launch",
			task: &Task{
				ID:          "task-1",
				Name:        "test-task",
				FrameworkID: "framework-1",
				Resources: &Resources{
					CPUs:   1.0,
					Memory: 512.0,
					Disk:   1000.0,
				},
			},
			expectedError: false,
		},
		{
			name: "Launch with zero resources",
			task: &Task{
				ID:          "task-2",
				Name:        "test-task-2",
				FrameworkID: "framework-1",
				Resources: &Resources{
					CPUs:   0.0,
					Memory: 0.0,
					Disk:   0.0,
				},
			},
			expectedError: false,
		},
		{
			name: "Insufficient CPU",
			task: &Task{
				ID:          "task-3",
				Name:        "test-task-3",
				FrameworkID: "framework-1",
				Resources: &Resources{
					CPUs:   10.0, // More than available
					Memory: 512.0,
					Disk:   1000.0,
				},
			},
			expectedError: true,
			errorContains: "insufficient resources",
		},
		{
			name: "Insufficient memory",
			task: &Task{
				ID:          "task-4",
				Name:        "test-task-4",
				FrameworkID: "framework-1",
				Resources: &Resources{
					CPUs:   1.0,
					Memory: 20000.0, // More than available
					Disk:   1000.0,
				},
			},
			expectedError: true,
			errorContains: "insufficient resources",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")
			err := agent.LaunchTask(tt.task)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "agent-1", tt.task.AgentID)
				assert.Equal(t, "starting", tt.task.State)
				assert.Contains(t, agent.Tasks, tt.task.ID)

				// Check executor was created
				executorID := fmt.Sprintf("executor-%s", tt.task.FrameworkID)
				assert.Contains(t, agent.Executors, executorID)

				executor := agent.Executors[executorID]
				assert.Equal(t, tt.task.FrameworkID, executor.FrameworkID)
				assert.Equal(t, "agent-1", executor.AgentID)
				assert.Contains(t, executor.Tasks, tt.task.ID)
			}
		})
	}
}

// TestAgent_KillTask tests killing a task
func TestAgent_KillTask(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")

	// Launch a task first
	task := &Task{
		ID:          "task-1",
		Name:        "test-task",
		FrameworkID: "framework-1",
		Resources: &Resources{
			CPUs:   1.0,
			Memory: 512.0,
			Disk:   1000.0,
		},
	}
	err := agent.LaunchTask(task)
	require.NoError(t, err)

	// Kill the task
	err = agent.KillTask("task-1")
	assert.NoError(t, err)
	assert.NotContains(t, agent.Tasks, "task-1")

	// Try to kill non-existent task
	err = agent.KillTask("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestAgent_HasResources tests resource checking
func TestAgent_HasResources(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")

	tests := []struct {
		name      string
		resources *Resources
		expected  bool
	}{
		{
			name:      "Nil resources",
			resources: nil,
			expected:  true,
		},
		{
			name: "Available resources",
			resources: &Resources{
				CPUs:   2.0,
				Memory: 2048.0,
				Disk:   10000.0,
			},
			expected: true,
		},
		{
			name: "Insufficient CPU",
			resources: &Resources{
				CPUs:   10.0,
				Memory: 2048.0,
				Disk:   10000.0,
			},
			expected: false,
		},
		{
			name: "Insufficient memory",
			resources: &Resources{
				CPUs:   2.0,
				Memory: 20000.0,
				Disk:   10000.0,
			},
			expected: false,
		},
		{
			name: "Insufficient disk",
			resources: &Resources{
				CPUs:   2.0,
				Memory: 2048.0,
				Disk:   200000.0,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agent.hasResources(tt.resources)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAgent_CalculateAvailableResources tests resource calculation
func TestAgent_CalculateAvailableResources(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")

	// Initially, all resources should be available
	available := agent.calculateAvailableResources()
	assert.Equal(t, 4.0, available.CPUs)
	assert.Equal(t, 8192.0, available.Memory)
	assert.Equal(t, 100000.0, available.Disk)

	// Launch a task
	task := &Task{
		ID:          "task-1",
		Name:        "test-task",
		FrameworkID: "framework-1",
		State:       "running",
		Resources: &Resources{
			CPUs:   1.0,
			Memory: 1024.0,
			Disk:   5000.0,
		},
	}
	agent.Tasks["task-1"] = task

	// Calculate again
	available = agent.calculateAvailableResources()
	assert.Equal(t, 3.0, available.CPUs)
	assert.Equal(t, 7168.0, available.Memory)
	assert.Equal(t, 95000.0, available.Disk)

	// Add a starting task (should not count)
	task2 := &Task{
		ID:          "task-2",
		Name:        "test-task-2",
		FrameworkID: "framework-1",
		State:       "starting",
		Resources: &Resources{
			CPUs:   1.0,
			Memory: 1024.0,
			Disk:   5000.0,
		},
	}
	agent.Tasks["task-2"] = task2

	// Should be same as before (starting tasks don't count)
	available = agent.calculateAvailableResources()
	assert.Equal(t, 3.0, available.CPUs)
	assert.Equal(t, 7168.0, available.Memory)
	assert.Equal(t, 95000.0, available.Disk)
}

// TestAgent_AllocateResources tests resource allocation
func TestAgent_AllocateResources(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")

	resources := &Resources{
		CPUs:   1.0,
		Memory: 512.0,
		Disk:   1000.0,
	}

	// Should not panic
	agent.allocateResources(resources)
}

// TestAgent_ReleaseResources tests resource release
func TestAgent_ReleaseResources(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")

	resources := &Resources{
		CPUs:   1.0,
		Memory: 512.0,
		Disk:   1000.0,
	}

	// Should not panic
	agent.releaseResources(resources)
}

// TestAgent_SendHeartbeat tests heartbeat sending
func TestAgent_SendHeartbeat(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")

	initialLastSeen := agent.LastSeen
	time.Sleep(10 * time.Millisecond)

	agent.sendHeartbeat()

	assert.True(t, agent.LastSeen.After(initialLastSeen))
}

// TestAgent_MonitorTasks tests task monitoring
func TestAgent_MonitorTasks(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")

	// Add a starting task
	task := &Task{
		ID:          "task-1",
		Name:        "test-task",
		FrameworkID: "framework-1",
		State:       "starting",
	}
	agent.Tasks["task-1"] = task

	// Monitor tasks
	agent.monitorTasks()

	// Task should now be running
	assert.Equal(t, "running", task.State)
	assert.False(t, task.StartedAt.IsZero())
}

// TestAgent_HandleAgentInfo tests the agent info endpoint
func TestAgent_HandleAgentInfo(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")

	req := httptest.NewRequest("GET", "/api/v1/agent/info", nil)
	w := httptest.NewRecorder()

	agent.handleAgentInfo(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var info map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&info)
	require.NoError(t, err)

	assert.Equal(t, "agent-1", info["id"])
	assert.Equal(t, "localhost", info["hostname"])
	assert.Equal(t, float64(5051), info["port"])
	assert.Equal(t, "inactive", info["status"])
	assert.NotNil(t, info["resources"])
}

// TestAgent_HandleAgentState tests the agent state endpoint
func TestAgent_HandleAgentState(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")

	req := httptest.NewRequest("GET", "/api/v1/agent/state", nil)
	w := httptest.NewRecorder()

	agent.handleAgentState(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var state map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&state)
	require.NoError(t, err)

	assert.Equal(t, "agent-1", state["id"])
	assert.Equal(t, "localhost", state["hostname"])
	assert.NotNil(t, state["resources"])
	assert.NotNil(t, state["tasks"])
	assert.NotNil(t, state["executors"])
}

// TestAgent_HandleListTasks tests listing tasks
func TestAgent_HandleListTasks(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")

	// Add a task
	task := &Task{
		ID:          "task-1",
		Name:        "test-task",
		FrameworkID: "framework-1",
	}
	agent.Tasks["task-1"] = task

	req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	w := httptest.NewRecorder()

	agent.handleListTasks(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var tasks []*Task
	err := json.NewDecoder(w.Body).Decode(&tasks)
	require.NoError(t, err)

	assert.Len(t, tasks, 1)
	assert.Equal(t, "task-1", tasks[0].ID)
}

// TestAgent_HandleLaunchTask tests launching task via HTTP
func TestAgent_HandleLaunchTask(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")

	tests := []struct {
		name           string
		task           *Task
		expectedStatus int
	}{
		{
			name: "Valid task",
			task: &Task{
				ID:          "task-1",
				Name:        "test-task",
				FrameworkID: "framework-1",
				Resources: &Resources{
					CPUs:   1.0,
					Memory: 512.0,
					Disk:   1000.0,
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Insufficient resources",
			task: &Task{
				ID:          "task-2",
				Name:        "test-task-2",
				FrameworkID: "framework-1",
				Resources: &Resources{
					CPUs:   100.0,
					Memory: 512.0,
					Disk:   1000.0,
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.task)
			req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewReader(body))
			w := httptest.NewRecorder()

			agent.handleLaunchTask(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestAgent_HandleGetTask tests getting a task
func TestAgent_HandleGetTask(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")
	router := agent.setupRoutes()

	// Add a task
	task := &Task{
		ID:          "task-1",
		Name:        "test-task",
		FrameworkID: "framework-1",
	}
	agent.Tasks["task-1"] = task

	// Get existing task
	req := httptest.NewRequest("GET", "/api/v1/tasks/task-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var returnedTask Task
	err := json.NewDecoder(w.Body).Decode(&returnedTask)
	require.NoError(t, err)
	assert.Equal(t, "task-1", returnedTask.ID)

	// Get non-existent task
	req = httptest.NewRequest("GET", "/api/v1/tasks/non-existent", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAgent_HandleKillTask tests killing a task via HTTP
func TestAgent_HandleKillTask(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")
	router := agent.setupRoutes()

	// Add a task
	task := &Task{
		ID:          "task-1",
		Name:        "test-task",
		FrameworkID: "framework-1",
		Resources: &Resources{
			CPUs:   1.0,
			Memory: 512.0,
			Disk:   1000.0,
		},
	}
	agent.Tasks["task-1"] = task

	// Kill existing task
	req := httptest.NewRequest("POST", "/api/v1/tasks/task-1/kill", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, agent.Tasks, "task-1")

	// Try to kill non-existent task
	req = httptest.NewRequest("POST", "/api/v1/tasks/non-existent/kill", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestAgent_HandleTaskStatus tests getting task status
func TestAgent_HandleTaskStatus(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")
	router := agent.setupRoutes()

	// Add a task
	task := &Task{
		ID:        "task-1",
		Name:      "test-task",
		State:     "running",
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}
	agent.Tasks["task-1"] = task

	// Get task status
	req := httptest.NewRequest("GET", "/api/v1/tasks/task-1/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var status map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&status)
	require.NoError(t, err)
	assert.Equal(t, "task-1", status["id"])
	assert.Equal(t, "running", status["state"])

	// Get status for non-existent task
	req = httptest.NewRequest("GET", "/api/v1/tasks/non-existent/status", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAgent_HandleListExecutors tests listing executors
func TestAgent_HandleListExecutors(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")

	// Add an executor
	executor := &Executor{
		ID:          "executor-1",
		FrameworkID: "framework-1",
		AgentID:     "agent-1",
		Status:      "active",
		Tasks:       make(map[string]*Task),
	}
	agent.Executors["executor-1"] = executor

	req := httptest.NewRequest("GET", "/api/v1/executors", nil)
	w := httptest.NewRecorder()

	agent.handleListExecutors(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var executors []*Executor
	err := json.NewDecoder(w.Body).Decode(&executors)
	require.NoError(t, err)

	assert.Len(t, executors, 1)
	assert.Equal(t, "executor-1", executors[0].ID)
}

// TestAgent_HandleGetExecutor tests getting an executor
func TestAgent_HandleGetExecutor(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")
	router := agent.setupRoutes()

	// Add an executor
	executor := &Executor{
		ID:          "executor-1",
		FrameworkID: "framework-1",
		AgentID:     "agent-1",
		Status:      "active",
		Tasks:       make(map[string]*Task),
	}
	agent.Executors["executor-1"] = executor

	// Get existing executor
	req := httptest.NewRequest("GET", "/api/v1/executors/executor-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var returnedExecutor Executor
	err := json.NewDecoder(w.Body).Decode(&returnedExecutor)
	require.NoError(t, err)
	assert.Equal(t, "executor-1", returnedExecutor.ID)

	// Get non-existent executor
	req = httptest.NewRequest("GET", "/api/v1/executors/non-existent", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAgent_HandleGetResources tests getting resources
func TestAgent_HandleGetResources(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")

	req := httptest.NewRequest("GET", "/api/v1/resources", nil)
	w := httptest.NewRecorder()

	agent.handleGetResources(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resources map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resources)
	require.NoError(t, err)

	assert.NotNil(t, resources["total"])
	assert.NotNil(t, resources["available"])
}

// TestAgent_HandleHealth tests health check endpoint
func TestAgent_HandleHealth(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	agent.handleHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var health map[string]string
	err := json.NewDecoder(w.Body).Decode(&health)
	require.NoError(t, err)

	assert.Equal(t, "healthy", health["status"])
}

// TestAgent_SetupRoutes tests route setup
func TestAgent_SetupRoutes(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")
	router := agent.setupRoutes()

	assert.NotNil(t, router)

	// Test routes that don't require path parameters
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/agent/info"},
		{"GET", "/api/v1/agent/state"},
		{"GET", "/api/v1/tasks"},
		{"POST", "/api/v1/tasks"},
		{"GET", "/api/v1/executors"},
		{"GET", "/api/v1/resources"},
		{"GET", "/health"},
	}

	for _, route := range routes {
		req := httptest.NewRequest(route.method, route.path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should not be 404 (route exists)
		assert.NotEqual(t, http.StatusNotFound, w.Code,
			"Route %s %s should exist", route.method, route.path)
	}

	// Routes with path parameters are tested in individual handler tests
}

// TestAgent_HandleLaunchTask_InvalidJSON tests error handling for invalid JSON
func TestAgent_HandleLaunchTask_InvalidJSON(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")

	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	agent.handleLaunchTask(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAgent_MultipleTasksAndExecutors tests complex scenarios
func TestAgent_MultipleTasksAndExecutors(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")

	// Launch tasks from multiple frameworks
	for i := 0; i < 3; i++ {
		for j := 0; j < 2; j++ {
			task := &Task{
				ID:          fmt.Sprintf("task-%d-%d", i, j),
				Name:        fmt.Sprintf("test-task-%d-%d", i, j),
				FrameworkID: fmt.Sprintf("framework-%d", i),
				Resources: &Resources{
					CPUs:   0.5,
					Memory: 256.0,
					Disk:   500.0,
				},
			}
			err := agent.LaunchTask(task)
			require.NoError(t, err)
		}
	}

	// Should have 6 tasks
	assert.Len(t, agent.Tasks, 6)

	// Should have 3 executors (one per framework)
	assert.Len(t, agent.Executors, 3)

	// Each executor should have 2 tasks
	for i := 0; i < 3; i++ {
		executorID := fmt.Sprintf("executor-framework-%d", i)
		executor, exists := agent.Executors[executorID]
		assert.True(t, exists)
		assert.Len(t, executor.Tasks, 2)
	}

	// Mark tasks as running (they start as "starting")
	for _, task := range agent.Tasks {
		task.State = "running"
	}

	// Calculate available resources
	available := agent.calculateAvailableResources()
	assert.InDelta(t, 1.0, available.CPUs, 0.01)       // 4.0 - 3.0 = 1.0
	assert.InDelta(t, 6656.0, available.Memory, 1.0)   // 8192 - 1536 = 6656
	assert.InDelta(t, 97000.0, available.Disk, 1.0)    // 100000 - 3000 = 97000
}

// TestAgent_ResourceExhaustion tests resource exhaustion scenarios
func TestAgent_ResourceExhaustion(t *testing.T) {
	agent := NewAgent("agent-1", "localhost", 5051, "http://master:5050")

	// Try to launch tasks until resources are exhausted
	// Note: The agent only tracks resources for "running" tasks, not "starting" tasks
	// So we need to mark each task as running before launching the next one
	taskCount := 0
	for i := 0; i < 10; i++ {
		task := &Task{
			ID:          fmt.Sprintf("task-%d", i),
			Name:        fmt.Sprintf("test-task-%d", i),
			FrameworkID: "framework-1",
			Resources: &Resources{
				CPUs:   1.0,
				Memory: 2048.0,
				Disk:   10000.0,
			},
		}
		err := agent.LaunchTask(task)
		if err == nil {
			taskCount++
			// Mark as running so resources are tracked
			task.State = "running"
		} else {
			// Should fail due to insufficient resources
			assert.Contains(t, err.Error(), "insufficient resources")
			break
		}
	}

	// Should have launched 4 tasks (4 CPUs available)
	assert.Equal(t, 4, taskCount)
	assert.Len(t, agent.Tasks, 4)
}
