package main

import (
    "context"
    "fmt"

    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/client"
)

// Node represents a compute node
type Node struct {
    ID       string
    Tasks    []Task
    Capacity int
}

// Task represents a task to be scheduled
type Task struct {
    ID    string
    Image string
}

// Orchestrator manages nodes and tasks
type Orchestrator struct {
    nodes  map[string]*Node
    tasks  map[string]Task
    client *client.Client
}

// NewOrchestrator initializes a new orchestrator
func NewOrchestrator() *Orchestrator {
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        fmt.Printf("Failed to create Docker client: %v\n", err)
        return nil
    }
    return &Orchestrator{
        nodes:  map[string]*Node{"node1": {ID: "node1", Capacity: 2}},
        tasks:  make(map[string]Task),
        client: cli,
    }
}

// ScheduleTask schedules a task on an available node
func (o *Orchestrator) ScheduleTask(task Task) error {
    for _, node := range o.nodes {
        if len(node.Tasks) < node.Capacity {
            node.Tasks = append(node.Tasks, task)
            o.tasks[task.ID] = task

            ctx := context.Background()
            config := &container.Config{
                Image: task.Image,
            }
            hostConfig := &container.HostConfig{}
            resp, err := o.client.ContainerCreate(ctx, config, hostConfig, nil, nil, task.ID)
            if err != nil {
                return fmt.Errorf("failed to create container: %v", err)
            }
            err = o.client.ContainerStart(ctx, resp.ID, container.StartOptions{})
            if err != nil {
                return fmt.Errorf("failed to start container: %v", err)
            }
            return nil
        }
    }
    return fmt.Errorf("no available nodes")
}

// StopTask stops and removes a task
func (o *Orchestrator) StopTask(taskID string) error {
    ctx := context.Background()
    err := o.client.ContainerStop(ctx, taskID, container.StopOptions{})
    if err != nil {
        return fmt.Errorf("failed to stop container: %v", err)
    }
    err = o.client.ContainerRemove(ctx, taskID, container.RemoveOptions{Force: true})
    if err != nil {
        return fmt.Errorf("failed to remove container: %v", err)
    }
    for _, node := range o.nodes {
        for i, t := range node.Tasks {
            if t.ID == taskID {
                node.Tasks = append(node.Tasks[:i], node.Tasks[i+1:]...)
                break
            }
        }
    }
    delete(o.tasks, taskID)
    return nil
}