package main

import (
    "context"
    "errors"
    "fmt"
    "sync"

    "github.com/docker/docker/api/types"
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/client"
)

// Task represents a container task
type Task struct {
    ID    string
    Image string
}

// Node represents a worker node
type Node struct {
    ID       string
    Capacity int
    Tasks    map[string]Task
}

// Orchestrator manages tasks and nodes
type Orchestrator struct {
    nodes  map[string]*Node
    tasks  map[string]Task
    client *client.Client
    mu     sync.Mutex
}

// NewOrchestrator creates a new orchestrator instance
func NewOrchestrator() *Orchestrator {
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        panic(fmt.Sprintf("Failed to create Docker client: %v", err))
    }

    return &Orchestrator{
        nodes: map[string]*Node{
            "node1": {ID: "node1", Capacity: 2, Tasks: make(map[string]Task)},
        },
        tasks:  make(map[string]Task),
        client: cli,
    }
}

// Start initializes the orchestrator
func (o *Orchestrator) Start() error {
    o.mu.Lock()
    defer o.mu.Unlock()

    // Basic health check
    ctx := context.Background()
    _, err := o.client.Ping(ctx)
    if err != nil {
        return fmt.Errorf("Docker daemon not reachable: %v", err)
    }
    return nil
}

// ScheduleTask schedules a task to a node
func (o *Orchestrator) ScheduleTask(task Task) error {
    o.mu.Lock()
    defer o.mu.Unlock()

    // Simple scheduling: find a node with capacity
    for _, node := range o.nodes {
        if len(node.Tasks) < node.Capacity {
            if err := o.startContainer(task); err != nil {
                return err
            }
            node.Tasks[task.ID] = task
            o.tasks[task.ID] = task
            return nil
        }
    }
    return errors.New("no available nodes")
}

// startContainer starts a Docker container for the task
func (o *Orchestrator) startContainer(task Task) error {
    ctx := context.Background()
    config := &container.Config{
        Image: task.Image,
    }
    resp, err := o.client.ContainerCreate(ctx, config, nil, nil, nil, task.ID)
    if err != nil {
        return fmt.Errorf("failed to create container: %v", err)
    }

    if err := o.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
        return fmt.Errorf("failed to start container: %v", err)
    }
    return nil
}

// StopTask stops and removes a task
func (o *Orchestrator) StopTask(taskID string) error {
    o.mu.Lock()
    defer o.mu.Unlock()

    ctx := context.Background()
    if err := o.client.ContainerStop(ctx, taskID, container.StopOptions{}); err != nil {
        return fmt.Errorf("failed to stop container: %v", err)
    }
    if err := o.client.ContainerRemove(ctx, taskID, types.ContainerRemoveOptions{}); err != nil {
        return fmt.Errorf("failed to remove container: %v", err)
    }

    for _, node := range o.nodes {
        delete(node.Tasks, taskID)
    }
    delete(o.tasks, taskID)
    return nil
}