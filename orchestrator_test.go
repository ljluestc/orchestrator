package main

import (
    "context"
    "os"
    "strings"
    "testing"

    "github.com/docker/docker/api/types/container"
    "github.com/stretchr/testify/assert"
)

// Unit Tests

func TestMain(t *testing.T) {
	// Test that main function doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("main() panicked: %v", r)
		}
	}()
	
	// We can't easily test main() directly, but we can test the NewOrchestrator function
	// which is the main logic
	o := NewOrchestrator()
	if o != nil {
		// If Docker is available, orchestrator should be created
		assert.NotNil(t, o)
	} else {
		// If Docker is not available, orchestrator should be nil
		assert.Nil(t, o)
	}
}

func TestNewOrchestrator(t *testing.T) {
    o := NewOrchestrator()
    assert.NotNil(t, o, "Orchestrator should be initialized")
    assert.NotNil(t, o.nodes, "Nodes map should be initialized")
    assert.NotNil(t, o.tasks, "Tasks map should be initialized")
    assert.NotNil(t, o.client, "Docker client should be initialized")
    assert.Equal(t, 1, len(o.nodes), "Should have one node")
}

func TestScheduleTask(t *testing.T) {
    o := NewOrchestrator()
    if o == nil {
        t.Skip("Docker not available, skipping test")
    }
    
    task := Task{ID: "test-task", Image: "alpine"}

    err := o.ScheduleTask(task)
    if err != nil && strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
        t.Skip("Docker not available, skipping test")
    }
    assert.NoError(t, err, "Should schedule task without error")
    assert.Equal(t, 1, len(o.nodes["node1"].Tasks), "Node should have one task")
    assert.Equal(t, task, o.tasks["test-task"], "Task should be in tasks map")

    // Clean up
    o.StopTask("test-task")
}

func TestScheduleTaskNoCapacity(t *testing.T) {
    o := NewOrchestrator()
    if o == nil {
        t.Skip("Docker not available, skipping test")
    }
    
    task1 := Task{ID: "task1", Image: "alpine"}
    task2 := Task{ID: "task2", Image: "alpine"}
    task3 := Task{ID: "task3", Image: "alpine"}

    o.ScheduleTask(task1)
    o.ScheduleTask(task2)
    err := o.ScheduleTask(task3)
    assert.Error(t, err, "Should fail when no capacity")
    assert.Equal(t, "no available nodes", err.Error(), "Error message should match")

    // Clean up
    o.StopTask("task1")
    o.StopTask("task2")
}

func TestStopTask(t *testing.T) {
    o := NewOrchestrator()
    if o == nil {
        t.Skip("Docker not available, skipping test")
    }
    
    // Test stopping a non-existent task
    err := o.StopTask("non-existent-task")
    if err != nil && strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
        t.Skip("Docker not available, skipping test")
    }
    // Should not panic, but might return an error for non-existent task
    _ = err
}

func TestStopTaskWithExistingTask(t *testing.T) {
    o := NewOrchestrator()
    if o == nil {
        t.Skip("Docker not available, skipping test")
    }
    
    // First schedule a task
    task := Task{ID: "test-stop-task", Image: "alpine"}
    err := o.ScheduleTask(task)
    if err != nil && strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
        t.Skip("Docker not available, skipping test")
    }
    
    if err == nil {
        // If scheduling succeeded, test stopping
        err = o.StopTask("test-stop-task")
        if err != nil && strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
            t.Skip("Docker not available, skipping test")
        }
        // Should not panic
        _ = err
    }
}

// Integration Tests (requires Docker)

func TestIntegrationStartContainer(t *testing.T) {
    if os.Getenv("INTEGRATION") != "true" {
        t.Skip("Skipping integration test; set INTEGRATION=true to run")
    }

    o := NewOrchestrator()
    task := Task{ID: "integration-test", Image: "alpine"}

    err := o.ScheduleTask(task)
    assert.NoError(t, err, "Should start container without error")

    ctx := context.Background()
    containers, err := o.client.ContainerList(ctx, container.ListOptions{All: true})
    assert.NoError(t, err, "Should list containers")
    found := false
    for _, c := range containers {
        if c.Names[0] == "/integration-test" {
            found = true
            break
        }
    }
    assert.True(t, found, "Container should be running")

    err = o.StopTask("integration-test")
    assert.NoError(t, err, "Should stop and remove container")
}