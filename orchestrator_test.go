package main

import (
    "context"
    "testing"

    "github.com/docker/docker/api/types"
    "github.com/docker/docker/client"
    "github.com/stretchr/testify/assert"
)

// Unit Tests

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
    task := Task{ID: "test-task", Image: "alpine"}

    err := o.ScheduleTask(task)
    assert.NoError(t, err, "Should schedule task without error")
    assert.Equal(t, 1, len(o.nodes["node1"].Tasks), "Node should have one task")
    assert.Equal(t, task, o.tasks["test-task"], "Task should be in tasks map")

    // Clean up
    o.StopTask("test-task")
}

func TestScheduleTaskNoCapacity(t *testing.T) {
    o := NewOrchestrator()
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
    containers, err := o.client.ContainerList(ctx, types.ContainerListOptions{All: true})
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