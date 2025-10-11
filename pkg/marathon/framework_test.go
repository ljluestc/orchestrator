package marathon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMarathon(t *testing.T) {
	// Test creating a new Marathon instance
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")
	assert.NotNil(t, marathon)
	assert.Equal(t, "test-marathon", marathon.ID)
	assert.Equal(t, "localhost", marathon.Hostname)
	assert.Equal(t, 8080, marathon.Port)
	assert.Equal(t, "http://localhost:5050", marathon.MasterURL)
}

func TestMarathon_Start(t *testing.T) {
	// Test starting Marathon
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")
	
	err := marathon.Start()
	assert.NoError(t, err)
	
	// Test stopping Marathon
	err = marathon.Stop()
	assert.NoError(t, err)
}

func TestMarathon_Stop(t *testing.T) {
	// Test stopping Marathon
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")
	
	err := marathon.Stop()
	assert.NoError(t, err)
}

func TestMarathon_RegisterFramework(t *testing.T) {
	// Test registering framework
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")
	
	err := marathon.RegisterFramework()
	assert.NoError(t, err)
}

func TestMarathon_DeployApp(t *testing.T) {
	// Test deploying application
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")
	
	app := Application{
		ID:   "test-app",
		Name: "Test Application",
		Container: &Container{
			Type: "DOCKER",
			Docker: &DockerInfo{
				Image: "nginx:latest",
			},
		},
		Instances: 1,
		CPUs:      0.5,
		Memory:    512,
	}
	
	err := marathon.DeployApp(app)
	assert.NoError(t, err)
	
	// Verify app was deployed
	apps, err := marathon.ListApps()
	assert.NoError(t, err)
	assert.Len(t, apps, 1)
	assert.Equal(t, "test-app", apps[0].ID)
}

func TestMarathon_UpdateApp(t *testing.T) {
	// Test updating application
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")
	
	// Deploy initial app
	app := Application{
		ID:        "test-app",
		Name:      "Test Application",
		Instances: 1,
		CPUs:      0.5,
		Memory:    512,
	}
	
	err := marathon.DeployApp(app)
	assert.NoError(t, err)
	
	// Update app
	app.Instances = 2
	app.Memory = 1024
	
	err = marathon.UpdateApp("test-app", &app)
	assert.NoError(t, err)
	
	// Verify app was updated
	updatedApp, err := marathon.GetApp("test-app")
	assert.NoError(t, err)
	assert.Equal(t, 2, updatedApp.Instances)
	assert.Equal(t, 1024, updatedApp.Memory)
}

func TestMarathon_DeleteApp(t *testing.T) {
	// Test deleting application
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")
	
	// Deploy app first
	app := Application{
		ID:        "test-app",
		Name:      "Test Application",
		Instances: 1,
		CPUs:      0.5,
		Memory:    512,
	}
	
	err := marathon.DeployApp(app)
	assert.NoError(t, err)
	
	// Delete app
	err = marathon.DeleteApp("test-app")
	assert.NoError(t, err)
	
	// Verify app was deleted
	apps, err := marathon.ListApps()
	assert.NoError(t, err)
	assert.Len(t, apps, 0)
}

func TestMarathon_ListApps(t *testing.T) {
	// Test listing applications
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")
	
	// Deploy some apps
	app1 := Application{
		ID:        "app1",
		Name:      "Application 1",
		Instances: 1,
		CPUs:      0.5,
		Memory:    512,
	}
	
	app2 := Application{
		ID:        "app2",
		Name:      "Application 2",
		Instances: 2,
		CPUs:      1.0,
		Memory:    1024,
	}
	
	err := marathon.DeployApp(app1)
	assert.NoError(t, err)
	
	err = marathon.DeployApp(app2)
	assert.NoError(t, err)
	
	// List apps
	apps, err := marathon.ListApps()
	assert.NoError(t, err)
	assert.Len(t, apps, 2)
	
	// Verify app IDs
	appIDs := make([]string, len(apps))
	for i, app := range apps {
		appIDs[i] = app.ID
	}
	assert.Contains(t, appIDs, "app1")
	assert.Contains(t, appIDs, "app2")
}

func TestMarathon_GetApp(t *testing.T) {
	// Test getting application
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")
	
	// Deploy app
	app := Application{
		ID:        "test-app",
		Name:      "Test Application",
		Instances: 1,
		CPUs:      0.5,
		Memory:    512,
	}
	
	err := marathon.DeployApp(app)
	assert.NoError(t, err)
	
	// Get app
	retrievedApp, err := marathon.GetApp("test-app")
	assert.NoError(t, err)
	assert.Equal(t, "test-app", retrievedApp.ID)
	assert.Equal(t, "Test Application", retrievedApp.Name)
	assert.Equal(t, 1, retrievedApp.Instances)
	assert.Equal(t, 0.5, retrievedApp.CPUs)
	assert.Equal(t, 512, retrievedApp.Memory)
	
	// Test getting non-existent app
	_, err = marathon.GetApp("nonexistent")
	assert.Error(t, err)
}

func TestMarathon_ListTasks(t *testing.T) {
	// Test listing tasks
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")
	
	// Deploy app
	app := Application{
		ID:        "test-app",
		Name:      "Test Application",
		Instances: 2,
		CPUs:      0.5,
		Memory:    512,
	}
	
	err := marathon.DeployApp(app)
	assert.NoError(t, err)
	
	// List tasks
	tasks, err := marathon.ListTasks("test-app")
	assert.NoError(t, err)
	assert.Len(t, tasks, 2)
	
	// Verify task properties
	for _, task := range tasks {
		assert.Equal(t, "test-app", task.AppID)
		assert.NotEmpty(t, task.ID)
		assert.NotEmpty(t, task.Host)
	}
}

func TestMarathon_GetTask(t *testing.T) {
	// Test getting task
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")
	
	// Deploy app
	app := Application{
		ID:        "test-app",
		Name:      "Test Application",
		Instances: 1,
		CPUs:      0.5,
		Memory:    512,
	}
	
	err := marathon.DeployApp(app)
	assert.NoError(t, err)
	
	// Get tasks
	tasks, err := marathon.ListTasks("test-app")
	assert.NoError(t, err)
	assert.Len(t, tasks, 1)
	
	// Get specific task
	task, err := marathon.GetTask(tasks[0].ID)
	assert.NoError(t, err)
	assert.Equal(t, tasks[0].ID, task.ID)
	assert.Equal(t, "test-app", task.AppID)
	
	// Test getting non-existent task
	_, err = marathon.GetTask("nonexistent")
	assert.Error(t, err)
}

func TestMarathon_ScaleApplication(t *testing.T) {
	// Test scaling application
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")
	
	// Deploy app
	app := Application{
		ID:        "test-app",
		Name:      "Test Application",
		Instances: 1,
		CPUs:      0.5,
		Memory:    512,
	}
	
	err := marathon.DeployApp(app)
	assert.NoError(t, err)
	
	// Scale app
	err = marathon.ScaleApplication("test-app", 3)
	assert.NoError(t, err)
	
	// Verify scaling
	scaledApp, err := marathon.GetApp("test-app")
	assert.NoError(t, err)
	assert.Equal(t, 3, scaledApp.Instances)
	
	// Test scaling non-existent app
	err = marathon.ScaleApplication("nonexistent", 2)
	assert.Error(t, err)
}

func TestMarathon_HealthChecks(t *testing.T) {
	// Test health checks
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")
	
	// Deploy app with health check
	app := Application{
		ID:        "test-app",
		Name:      "Test Application",
		Instances: 1,
		CPUs:      0.5,
		Memory:    512,
		HealthCheck: &HealthCheck{
			Path:     "/health",
			Port:     8080,
			Protocol: "HTTP",
		},
	}
	
	err := marathon.DeployApp(app)
	assert.NoError(t, err)
	
	// Run health checks
	err = marathon.RunHealthChecks()
	assert.NoError(t, err)
}

func TestMarathon_ErrorHandling(t *testing.T) {
	// Test error handling
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")
	
	// Test invalid operations
	err := marathon.DeleteApp("nonexistent")
	assert.Error(t, err)
	
	_, err = marathon.GetApp("nonexistent")
	assert.Error(t, err)
	
	_, err = marathon.GetTask("nonexistent")
	assert.Error(t, err)
	
	err = marathon.ScaleApplication("nonexistent", 2)
	assert.Error(t, err)
}

func TestMarathon_ConcurrentAccess(t *testing.T) {
	// Test concurrent access
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")
	
	// Test concurrent app deployments
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(i int) {
			defer func() { done <- true }()
			app := Application{
				ID:        "app" + string(rune(i)),
				Name:      "Application " + string(rune(i)),
				Instances: 1,
				CPUs:      0.5,
				Memory:    512,
			}
			err := marathon.DeployApp(app)
			assert.NoError(t, err)
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}
	
	// Verify all apps were deployed
	apps, err := marathon.ListApps()
	assert.NoError(t, err)
	assert.Len(t, apps, 5)
}

func TestMarathon_Performance(t *testing.T) {
	// Test performance
	marathon := NewMarathon("test-marathon", "localhost", 8080, "http://localhost:5050")
	
	// Test deploying many apps
	for i := 0; i < 100; i++ {
		app := Application{
			ID:        "app" + string(rune(i)),
			Name:      "Application " + string(rune(i)),
			Instances: 1,
			CPUs:      0.5,
			Memory:    512,
		}
		err := marathon.DeployApp(app)
		assert.NoError(t, err)
	}
	
	// Verify all apps were deployed
	apps, err := marathon.ListApps()
	assert.NoError(t, err)
	assert.Len(t, apps, 100)
}
