package marathon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMarathon(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		hostname  string
		port      int
		masterURL string
	}{
		{
			name:      "Valid Marathon",
			id:        "marathon-1",
			hostname:  "localhost",
			port:      8080,
			masterURL: "http://localhost:5050",
		},
		{
			name:      "Empty ID",
			id:        "",
			hostname:  "localhost",
			port:      8080,
			masterURL: "http://localhost:5050",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marathon := NewMarathon(tt.id, tt.hostname, tt.port, tt.masterURL)

			assert.Equal(t, tt.id, marathon.ID)
			assert.Equal(t, tt.hostname, marathon.Hostname)
			assert.Equal(t, tt.port, marathon.Port)
			assert.Equal(t, tt.masterURL, marathon.MasterURL)
			assert.NotNil(t, marathon.Applications)
			assert.NotNil(t, marathon.Deployments)
			assert.NotNil(t, marathon.Tasks)
		})
	}
}

func TestMarathon_CreateApp(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	tests := []struct {
		name        string
		app         *Application
		expectError bool
	}{
		{
			name: "Valid application",
			app: &Application{
				ID:        "/test/app",
				Instances: 3,
				CPUs:      1.0,
				Memory:    1024.0,
				Container: &Container{
					Type: "DOCKER",
					Docker: &DockerSpec{
						Image: "nginx:latest",
					},
				},
			},
			expectError: false,
		},
		{
			name: "Application with health checks",
			app: &Application{
				ID:        "/test/app-with-health",
				Instances: 2,
				CPUs:      0.5,
				Memory:    512.0,
				HealthChecks: []*HealthCheck{
					{
						Protocol:        "HTTP",
						Path:            "/health",
						PortIndex:       0,
						IntervalSeconds: 30,
						TimeoutSeconds:  10,
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := marathon.CreateApp(tt.app)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, tt.app.Version)
				assert.Len(t, tt.app.Tasks, tt.app.Instances)
				assert.Len(t, tt.app.Deployments, 1)
				assert.Equal(t, tt.app.TasksStaged, tt.app.Instances)
				assert.Equal(t, tt.app.TasksRunning, 0)
				assert.Equal(t, tt.app.TasksHealthy, 0)
				assert.Equal(t, tt.app.TasksUnhealthy, 0)

				// Verify app is stored
				storedApp, exists := marathon.Applications[tt.app.ID]
				assert.True(t, exists)
				assert.Equal(t, tt.app, storedApp)

				// Verify tasks are created
				for i, task := range tt.app.Tasks {
					expectedTaskID := tt.app.ID + "." + string(rune(i))
					assert.Equal(t, expectedTaskID, task.ID)
					assert.Equal(t, tt.app.ID, task.AppID)
					assert.Equal(t, tt.app.Version, task.Version)
					assert.Equal(t, "TASK_STAGING", task.State)
					assert.NotNil(t, task.StagedAt)
				}

				// Verify deployment is created
				deployment := tt.app.Deployments[0]
				assert.NotEmpty(t, deployment.ID)
				assert.Equal(t, tt.app.Version, deployment.Version)
				assert.Contains(t, deployment.AffectedApps, tt.app.ID)
				assert.Len(t, deployment.Steps, 1)
				assert.Equal(t, "StartApplication", deployment.Steps[0].Action)
				assert.Equal(t, tt.app.ID, deployment.Steps[0].App)
			}
		})
	}
}

func TestMarathon_CreateAppDuplicate(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	app := &Application{
		ID:        "/test/app",
		Instances: 1,
		CPUs:      1.0,
		Memory:    1024.0,
	}

	// Create app first time
	err := marathon.CreateApp(app)
	assert.NoError(t, err)

	// Try to create same app again
	err = marathon.CreateApp(app)
	assert.NoError(t, err) // Should overwrite existing app
}

func TestMarathon_UpdateApp(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	// Create initial app
	originalApp := &Application{
		ID:        "/test/app",
		Instances: 2,
		CPUs:      1.0,
		Memory:    1024.0,
	}

	err := marathon.CreateApp(originalApp)
	assert.NoError(t, err)

	// Update app
	updatedApp := &Application{
		ID:        "/test/app",
		Instances: 3,
		CPUs:      2.0,
		Memory:    2048.0,
	}

	err = marathon.UpdateApp("/test/app", updatedApp)
	assert.NoError(t, err)

	// Verify update
	storedApp := marathon.Applications["/test/app"]
	assert.Equal(t, 3, storedApp.Instances)
	assert.Equal(t, 2.0, storedApp.CPUs)
	assert.Equal(t, 2048.0, storedApp.Memory)
	assert.NotEqual(t, originalApp.Version, storedApp.Version)
	assert.Len(t, storedApp.Deployments, 2) // Original + update deployment
}

func TestMarathon_UpdateAppNotFound(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	app := &Application{
		ID:        "/nonexistent/app",
		Instances: 1,
		CPUs:      1.0,
		Memory:    1024.0,
	}

	err := marathon.UpdateApp("/nonexistent/app", app)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMarathon_DeleteApp(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	// Create app
	app := &Application{
		ID:        "/test/app",
		Instances: 2,
		CPUs:      1.0,
		Memory:    1024.0,
	}

	err := marathon.CreateApp(app)
	assert.NoError(t, err)

	// Verify app exists
	_, exists := marathon.Applications["/test/app"]
	assert.True(t, exists)

	// Delete app
	err = marathon.DeleteApp("/test/app")
	assert.NoError(t, err)

	// Verify app is deleted
	_, exists = marathon.Applications["/test/app"]
	assert.False(t, exists)

	// Verify tasks are deleted
	for _, task := range app.Tasks {
		_, exists := marathon.Tasks[task.ID]
		assert.False(t, exists)
	}
}

func TestMarathon_DeleteAppNotFound(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	err := marathon.DeleteApp("/nonexistent/app")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMarathon_ScaleApp(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	// Create app with 2 instances
	app := &Application{
		ID:        "/test/app",
		Instances: 2,
		CPUs:      1.0,
		Memory:    1024.0,
	}

	err := marathon.CreateApp(app)
	assert.NoError(t, err)

	_ = len(app.Tasks)

	// Scale up to 5 instances
	err = marathon.ScaleApp("/test/app", 5)
	assert.NoError(t, err)

	// Verify scaling up
	storedApp := marathon.Applications["/test/app"]
	assert.Equal(t, 5, storedApp.Instances)
	assert.Len(t, storedApp.Tasks, 5)
	assert.Equal(t, 5, storedApp.TasksStaged)

	// Scale down to 3 instances
	err = marathon.ScaleApp("/test/app", 3)
	assert.NoError(t, err)

	// Verify scaling down
	storedApp = marathon.Applications["/test/app"]
	assert.Equal(t, 3, storedApp.Instances)
	assert.Len(t, storedApp.Tasks, 3)

	// Verify tasks were killed
	killedTasks := 0
	for _, task := range marathon.Tasks {
		if task.State == "TASK_KILLED" {
			killedTasks++
		}
	}
	assert.Equal(t, 2, killedTasks) // 5 - 3 = 2 tasks killed
}

func TestMarathon_ScaleAppNotFound(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	err := marathon.ScaleApp("/nonexistent/app", 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMarathon_CreateTask(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	app := &Application{
		ID:        "/test/app",
		Instances: 1,
		CPUs:      1.0,
		Memory:    1024.0,
	}

	err := marathon.CreateApp(app)
	assert.NoError(t, err)

	task := marathon.createTask(app, 0)

	assert.Equal(t, "/test/app.0", task.ID)
	assert.Equal(t, "/test/app", task.AppID)
	assert.Equal(t, "localhost", task.Host)
	assert.Equal(t, []int{8080}, task.Ports)
	assert.Equal(t, app.Version, task.Version)
	assert.Equal(t, "TASK_STAGING", task.State)
	assert.NotNil(t, task.StagedAt)
}

func TestMarathon_MonitorTasks(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	// Create app with tasks
	app := &Application{
		ID:        "/test/app",
		Instances: 2,
		CPUs:      1.0,
		Memory:    1024.0,
	}

	err := marathon.CreateApp(app)
	assert.NoError(t, err)

	// Verify tasks are in staging
	for _, task := range app.Tasks {
		assert.Equal(t, "TASK_STAGING", task.State)
		assert.Nil(t, task.StartedAt)
	}

	// Monitor tasks
	marathon.monitorTasks()

	// Verify tasks transitioned to running
	for _, task := range app.Tasks {
		assert.Equal(t, "TASK_RUNNING", task.State)
		assert.NotNil(t, task.StartedAt)
	}
}

func TestMarathon_HTTPHandlers(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	// Create test app
	app := &Application{
		ID:        "/test/app",
		Instances: 1,
		CPUs:      1.0,
		Memory:    1024.0,
	}

	err := marathon.CreateApp(app)
	assert.NoError(t, err)

	testCases := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{"List Apps", "GET", "/v2/apps", http.StatusOK},
		{"Get App", "GET", "/v2/apps/test/app", http.StatusOK},
		{"List Tasks", "GET", "/v2/tasks", http.StatusOK},
		{"Get App Tasks", "GET", "/v2/apps/test/app/tasks", http.StatusOK},
		{"Get Task", "GET", "/v2/tasks/test/app.0", http.StatusOK},
		{"List Deployments", "GET", "/v2/deployments", http.StatusOK},
		{"Get Deployment", "GET", "/v2/deployments/" + app.Deployments[0].ID, http.StatusOK},
		{"App Health", "GET", "/v2/apps/test/app/health", http.StatusOK},
		{"Ping", "GET", "/ping", http.StatusOK},
		{"Health", "GET", "/health", http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			router := marathon.setupRoutes()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.status, rr.Code)
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
		})
	}
}

func TestMarathon_HandleListApps(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	// Create test apps
	app1 := &Application{ID: "/app1", Instances: 1, CPUs: 1.0, Memory: 1024.0}
	app2 := &Application{ID: "/app2", Instances: 2, CPUs: 2.0, Memory: 2048.0}

	marathon.CreateApp(app1)
	marathon.CreateApp(app2)

	req := httptest.NewRequest("GET", "/v2/apps", nil)
	rr := httptest.NewRecorder()

	marathon.handleListApps(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)

	apps := response["apps"].([]interface{})
	assert.Len(t, apps, 2)
}

func TestMarathon_HandleCreateApp(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	app := &Application{ID: "test-app", Instances: 1, CPUs: 1.0, Memory: 1024.0}
	
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(app)
	
	req := httptest.NewRequest("POST", "/v2/apps", &buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Use the router to handle the request
	router := marathon.setupRoutes()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify app was created
	_, exists := marathon.Applications["test-app"]
	assert.True(t, exists)
}

func TestMarathon_HandleGetApp(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	// Create test app
	app := &Application{ID: "/test/app", Instances: 1, CPUs: 1.0, Memory: 1024.0}
	marathon.CreateApp(app)

	req := httptest.NewRequest("GET", "/v2/apps/test/app", nil)
	rr := httptest.NewRecorder()

	marathon.handleGetApp(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response Application
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "/test/app", response.ID)
}

func TestMarathon_HandleGetAppNotFound(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	req := httptest.NewRequest("GET", "/v2/apps/nonexistent", nil)
	rr := httptest.NewRecorder()

	marathon.handleGetApp(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestMarathon_HandleUpdateApp(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	// Create initial app
	app := &Application{ID: "test-app", Instances: 1, CPUs: 1.0, Memory: 1024.0}
	marathon.CreateApp(app)

	// Update app
	updatedApp := &Application{ID: "test-app", Instances: 2, CPUs: 2.0, Memory: 2048.0}
	
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(updatedApp)
	
	req := httptest.NewRequest("PUT", "/v2/apps/test-app", &buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Use the router to handle the request
	router := marathon.setupRoutes()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Logf("Response body: %s", rr.Body.String())
	}
	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify app was updated
	storedApp := marathon.Applications["test-app"]
	assert.Equal(t, 2, storedApp.Instances)
	assert.Equal(t, 2.0, storedApp.CPUs)
	assert.Equal(t, 2048.0, storedApp.Memory)
}

func TestMarathon_HandleDeleteApp(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	// Create test app
	app := &Application{ID: "/test/app", Instances: 1, CPUs: 1.0, Memory: 1024.0}
	marathon.CreateApp(app)

	req := httptest.NewRequest("DELETE", "/v2/apps/test/app", nil)
	rr := httptest.NewRecorder()

	marathon.handleDeleteApp(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify app was deleted
	_, exists := marathon.Applications["/test/app"]
	assert.False(t, exists)
}

func TestMarathon_HandleRestartApp(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	req := httptest.NewRequest("POST", "/v2/apps/test/app/restart", nil)
	rr := httptest.NewRecorder()

	marathon.handleRestartApp(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMarathon_HandleScaleApp(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	// Create test app
	app := &Application{ID: "test-app", Instances: 1, CPUs: 1.0, Memory: 1024.0}
	marathon.CreateApp(app)

	scaleRequest := map[string]int{"instances": 3}
	
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(scaleRequest)
	
	req := httptest.NewRequest("PUT", "/v2/apps/test-app/scale", &buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Use the router to handle the request
	router := marathon.setupRoutes()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify app was scaled
	storedApp := marathon.Applications["test-app"]
	assert.Equal(t, 3, storedApp.Instances)
}

func TestMarathon_HandleListTasks(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	// Create test app with tasks
	app := &Application{ID: "/test/app", Instances: 2, CPUs: 1.0, Memory: 1024.0}
	marathon.CreateApp(app)

	req := httptest.NewRequest("GET", "/v2/tasks", nil)
	rr := httptest.NewRecorder()

	marathon.handleListTasks(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)

	tasks := response["tasks"].([]interface{})
	assert.Len(t, tasks, 2)
}

func TestMarathon_HandleListAppTasks(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	// Create test app
	app := &Application{ID: "/test/app", Instances: 2, CPUs: 1.0, Memory: 1024.0}
	marathon.CreateApp(app)

	req := httptest.NewRequest("GET", "/v2/apps/test/app/tasks", nil)
	rr := httptest.NewRecorder()

	marathon.handleListAppTasks(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var tasks []MarathonTask
	err := json.Unmarshal(rr.Body.Bytes(), &tasks)
	assert.NoError(t, err)
	assert.Len(t, tasks, 2)
}

func TestMarathon_HandleGetTask(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	// Create test app
	app := &Application{ID: "/test/app", Instances: 1, CPUs: 1.0, Memory: 1024.0}
	marathon.CreateApp(app)

	taskID := "/test/app.0"
	req := httptest.NewRequest("GET", "/v2/tasks/"+taskID, nil)
	rr := httptest.NewRecorder()

	marathon.handleGetTask(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var task MarathonTask
	err := json.Unmarshal(rr.Body.Bytes(), &task)
	assert.NoError(t, err)
	assert.Equal(t, taskID, task.ID)
}

func TestMarathon_HandleGetTaskNotFound(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	req := httptest.NewRequest("GET", "/v2/tasks/nonexistent", nil)
	rr := httptest.NewRecorder()

	marathon.handleGetTask(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestMarathon_HandleKillTask(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	// Create test app
	app := &Application{ID: "/test/app", Instances: 1, CPUs: 1.0, Memory: 1024.0}
	marathon.CreateApp(app)

	taskID := "/test/app.0"
	req := httptest.NewRequest("DELETE", "/v2/tasks/"+taskID+"/kill", nil)
	rr := httptest.NewRecorder()

	marathon.handleKillTask(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify task was killed
	task := marathon.Tasks[taskID]
	assert.Equal(t, "TASK_KILLED", task.State)
}

func TestMarathon_HandleListDeployments(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	// Create test app (creates deployment)
	app := &Application{ID: "/test/app", Instances: 1, CPUs: 1.0, Memory: 1024.0}
	marathon.CreateApp(app)

	req := httptest.NewRequest("GET", "/v2/deployments", nil)
	rr := httptest.NewRecorder()

	marathon.handleListDeployments(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var deployments []Deployment
	err := json.Unmarshal(rr.Body.Bytes(), &deployments)
	assert.NoError(t, err)
	assert.Len(t, deployments, 1)
}

func TestMarathon_HandleGetDeployment(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	// Create test app (creates deployment)
	app := &Application{ID: "/test/app", Instances: 1, CPUs: 1.0, Memory: 1024.0}
	marathon.CreateApp(app)

	deploymentID := app.Deployments[0].ID
	req := httptest.NewRequest("GET", "/v2/deployments/"+deploymentID, nil)
	rr := httptest.NewRecorder()

	marathon.handleGetDeployment(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var deployment Deployment
	err := json.Unmarshal(rr.Body.Bytes(), &deployment)
	assert.NoError(t, err)
	assert.Equal(t, deploymentID, deployment.ID)
}

func TestMarathon_HandleGetDeploymentNotFound(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	req := httptest.NewRequest("GET", "/v2/deployments/nonexistent", nil)
	rr := httptest.NewRecorder()

	marathon.handleGetDeployment(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestMarathon_HandleDeleteDeployment(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	// Create test app (creates deployment)
	app := &Application{ID: "/test/app", Instances: 1, CPUs: 1.0, Memory: 1024.0}
	marathon.CreateApp(app)

	deploymentID := app.Deployments[0].ID
	req := httptest.NewRequest("DELETE", "/v2/deployments/"+deploymentID, nil)
	rr := httptest.NewRecorder()

	marathon.handleDeleteDeployment(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify deployment was deleted
	_, exists := marathon.Deployments[deploymentID]
	assert.False(t, exists)
}

func TestMarathon_HandleAppHealth(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	// Create test app
	app := &Application{ID: "/test/app", Instances: 2, CPUs: 1.0, Memory: 1024.0}
	marathon.CreateApp(app)

	req := httptest.NewRequest("GET", "/v2/apps/test/app/health", nil)
	rr := httptest.NewRecorder()

	marathon.handleAppHealth(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var health map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &health)
	assert.NoError(t, err)

	assert.Equal(t, float64(0), health["tasksRunning"])
	assert.Equal(t, float64(0), health["tasksHealthy"])
	assert.Equal(t, float64(0), health["tasksUnhealthy"])
}

func TestMarathon_HandleAppHealthNotFound(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	req := httptest.NewRequest("GET", "/v2/apps/nonexistent/health", nil)
	rr := httptest.NewRecorder()

	marathon.handleAppHealth(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestMarathon_HandlePing(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	req := httptest.NewRequest("GET", "/ping", nil)
	rr := httptest.NewRecorder()

	marathon.handlePing(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMarathon_HandleHealth(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	marathon.handleHealth(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var health map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &health)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", health["status"])
}

func TestMarathon_SetupRoutes(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	router := marathon.setupRoutes()

	assert.NotNil(t, router)

	// Test that routes are properly configured
	testCases := []struct {
		method string
		path   string
	}{
		{"GET", "/v2/apps"},
		{"POST", "/v2/apps"},
		{"GET", "/v2/apps/test-app"},
		{"PUT", "/v2/apps/test-app"},
		{"DELETE", "/v2/apps/test-app"},
		{"POST", "/v2/apps/test-app/restart"},
		{"PUT", "/v2/apps/test-app/scale"},
		{"GET", "/v2/tasks"},
		{"GET", "/v2/apps/test-app/tasks"},
		{"GET", "/v2/tasks/task-id"},
		{"DELETE", "/v2/tasks/task-id/kill"},
		{"GET", "/v2/deployments"},
		{"GET", "/v2/deployments/deployment-id"},
		{"DELETE", "/v2/deployments/deployment-id"},
		{"GET", "/v2/apps/test-app/health"},
		{"GET", "/ping"},
		{"GET", "/health"},
	}

	for _, tc := range testCases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			// Should not return 404 (route not found)
			assert.NotEqual(t, http.StatusNotFound, rr.Code, "Route %s %s should be found", tc.method, tc.path)
		})
	}
}

func TestMarathon_StartStop(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 0, "http://localhost:5050")

	// Start server
	errChan := make(chan error, 1)
	go func() {
		errChan <- marathon.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := marathon.server.Shutdown(ctx)
	if err != nil {
		t.Logf("Server shutdown error: %v", err)
	}

	// Check for start error
	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("Server start timeout")
	}
}

func TestMarathon_ConcurrentAccess(t *testing.T) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	const numGoroutines = 10
	const numOperations = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numOperations; j++ {
				app := &Application{
					ID:        fmt.Sprintf("/app-%d", j),
					Instances: 1,
					CPUs:      1.0,
					Memory:    1024.0,
				}
				marathon.CreateApp(app)
				marathon.ScaleApp(app.ID, 2)
				marathon.DeleteApp(app.ID)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestMarathon_ApplicationStructures(t *testing.T) {
	// Test Application structure
	app := &Application{
		ID:        "/test/app",
		Instances: 3,
		CPUs:      1.5,
		Memory:    2048.0,
		Container: &Container{
			Type: "DOCKER",
			Docker: &DockerSpec{
				Image: "nginx:latest",
				Network: "BRIDGE",
				PortMappings: []*PortMapping{
					{ContainerPort: 80, HostPort: 0, ServicePort: 10000, Protocol: "tcp"},
				},
				Parameters: []*Parameter{
					{Key: "env", Value: "production"},
				},
				Privileged:     false,
				ForcePullImage: true,
			},
		},
		HealthChecks: []*HealthCheck{
			{
				Protocol:               "HTTP",
				Path:                   "/health",
				PortIndex:              0,
				GracePeriodSeconds:     300,
				IntervalSeconds:        30,
				TimeoutSeconds:         10,
				MaxConsecutiveFailures: 3,
				IgnoreHTTP1xx:          false,
			},
		},
		Constraints: [][]string{
			{"hostname", "UNIQUE"},
			{"rack", "GROUP_BY", "rack-1"},
		},
		Labels: map[string]string{
			"environment": "production",
			"team":        "platform",
		},
		Env: map[string]string{
			"LOG_LEVEL": "info",
			"PORT":      "8080",
		},
	}

	assert.Equal(t, "/test/app", app.ID)
	assert.Equal(t, 3, app.Instances)
	assert.Equal(t, 1.5, app.CPUs)
	assert.Equal(t, 2048.0, app.Memory)
	assert.Equal(t, "DOCKER", app.Container.Type)
	assert.Equal(t, "nginx:latest", app.Container.Docker.Image)
	assert.Equal(t, "BRIDGE", app.Container.Docker.Network)
	assert.Len(t, app.Container.Docker.PortMappings, 1)
	assert.Len(t, app.Container.Docker.Parameters, 1)
	assert.False(t, app.Container.Docker.Privileged)
	assert.True(t, app.Container.Docker.ForcePullImage)
	assert.Len(t, app.HealthChecks, 1)
	assert.Equal(t, "HTTP", app.HealthChecks[0].Protocol)
	assert.Equal(t, "/health", app.HealthChecks[0].Path)
	assert.Equal(t, 0, app.HealthChecks[0].PortIndex)
	assert.Equal(t, 300, app.HealthChecks[0].GracePeriodSeconds)
	assert.Equal(t, 30, app.HealthChecks[0].IntervalSeconds)
	assert.Equal(t, 10, app.HealthChecks[0].TimeoutSeconds)
	assert.Equal(t, 3, app.HealthChecks[0].MaxConsecutiveFailures)
	assert.False(t, app.HealthChecks[0].IgnoreHTTP1xx)
	assert.Len(t, app.Constraints, 2)
	assert.Equal(t, "hostname", app.Constraints[0][0])
	assert.Equal(t, "UNIQUE", app.Constraints[0][1])
	assert.Equal(t, "rack", app.Constraints[1][0])
	assert.Equal(t, "GROUP_BY", app.Constraints[1][1])
	assert.Equal(t, "rack-1", app.Constraints[1][2])
	assert.Equal(t, "production", app.Labels["environment"])
	assert.Equal(t, "platform", app.Labels["team"])
	assert.Equal(t, "info", app.Env["LOG_LEVEL"])
	assert.Equal(t, "8080", app.Env["PORT"])
}

func TestMarathon_TaskStructures(t *testing.T) {
	now := time.Now()
	task := &MarathonTask{
		ID:    "/test/app.0",
		AppID: "/test/app",
		Host:  "localhost",
		Ports: []int{8080, 8081},
		StartedAt: &now,
		StagedAt: &now,
		Version: "2023-01-01T00:00:00.000Z",
		State: "TASK_RUNNING",
		HealthCheckResults: []*HealthCheckResult{
			{
				Alive:               true,
				ConsecutiveFailures: 0,
				FirstSuccess:        &now,
				LastSuccess:         &now,
			},
		},
		ServicePorts: []int{10000, 10001},
		IPAddresses: []*IPAddress{
			{IPAddress: "192.168.1.100", Protocol: "IPv4"},
		},
	}

	assert.Equal(t, "/test/app.0", task.ID)
	assert.Equal(t, "/test/app", task.AppID)
	assert.Equal(t, "localhost", task.Host)
	assert.Equal(t, []int{8080, 8081}, task.Ports)
	assert.Equal(t, now, *task.StartedAt)
	assert.Equal(t, now, *task.StagedAt)
	assert.Equal(t, "2023-01-01T00:00:00.000Z", task.Version)
	assert.Equal(t, "TASK_RUNNING", task.State)
	assert.Len(t, task.HealthCheckResults, 1)
	assert.True(t, task.HealthCheckResults[0].Alive)
	assert.Equal(t, 0, task.HealthCheckResults[0].ConsecutiveFailures)
	assert.Equal(t, now, *task.HealthCheckResults[0].FirstSuccess)
	assert.Equal(t, now, *task.HealthCheckResults[0].LastSuccess)
	assert.Equal(t, []int{10000, 10001}, task.ServicePorts)
	assert.Len(t, task.IPAddresses, 1)
	assert.Equal(t, "192.168.1.100", task.IPAddresses[0].IPAddress)
	assert.Equal(t, "IPv4", task.IPAddresses[0].Protocol)
}

func TestMarathon_DeploymentStructures(t *testing.T) {
	deployment := &Deployment{
		ID:           "deployment-123",
		Version:      "2023-01-01T00:00:00.000Z",
		AffectedApps: []string{"/test/app"},
		Steps: []*DeploymentStep{
			{Action: "StartApplication", App: "/test/app"},
		},
		CurrentActions: []*DeploymentAction{
			{Action: "StartApplication", App: "/test/app"},
		},
		CurrentStep: 0,
		TotalSteps:  1,
		ReadinessCheckResults: []*ReadinessCheckResult{
			{
				TaskID: "/test/app.0",
				LastResponse: &ReadinessCheckResponse{
					Status: 200,
					Body:   "OK",
					Headers: map[string]string{
						"Content-Type": "text/plain",
					},
				},
			},
		},
	}

	assert.Equal(t, "deployment-123", deployment.ID)
	assert.Equal(t, "2023-01-01T00:00:00.000Z", deployment.Version)
	assert.Equal(t, []string{"/test/app"}, deployment.AffectedApps)
	assert.Len(t, deployment.Steps, 1)
	assert.Equal(t, "StartApplication", deployment.Steps[0].Action)
	assert.Equal(t, "/test/app", deployment.Steps[0].App)
	assert.Len(t, deployment.CurrentActions, 1)
	assert.Equal(t, "StartApplication", deployment.CurrentActions[0].Action)
	assert.Equal(t, "/test/app", deployment.CurrentActions[0].App)
	assert.Equal(t, 0, deployment.CurrentStep)
	assert.Equal(t, 1, deployment.TotalSteps)
	assert.Len(t, deployment.ReadinessCheckResults, 1)
	assert.Equal(t, "/test/app.0", deployment.ReadinessCheckResults[0].TaskID)
	assert.Equal(t, 200, deployment.ReadinessCheckResults[0].LastResponse.Status)
	assert.Equal(t, "OK", deployment.ReadinessCheckResults[0].LastResponse.Body)
	assert.Equal(t, "text/plain", deployment.ReadinessCheckResults[0].LastResponse.Headers["Content-Type"])
}

func BenchmarkMarathon_CreateApp(b *testing.B) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	for i := 0; i < b.N; i++ {
		app := &Application{
			ID:        fmt.Sprintf("/app-%d", i),
			Instances: 1,
			CPUs:      1.0,
			Memory:    1024.0,
		}
		marathon.CreateApp(app)
	}
}

func BenchmarkMarathon_ScaleApp(b *testing.B) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	app := &Application{ID: "/test/app", Instances: 1, CPUs: 1.0, Memory: 1024.0}
	marathon.CreateApp(app)

	for i := 0; i < b.N; i++ {
		marathon.ScaleApp("/test/app", i%10+1)
	}
}

func BenchmarkMarathon_HandleListApps(b *testing.B) {
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")

	// Create test apps
	for i := 0; i < 100; i++ {
		app := &Application{
			ID:        fmt.Sprintf("/app-%d", i),
			Instances: 1,
			CPUs:      1.0,
			Memory:    1024.0,
		}
		marathon.CreateApp(app)
	}

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/v2/apps", nil)
		rr := httptest.NewRecorder()
		marathon.handleListApps(rr, req)
	}
}