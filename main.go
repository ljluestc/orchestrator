package main

import (
    "fmt"
    "log"
    "os"
)

func main() {
    o := NewOrchestrator()
    if err := o.Start(); err != nil {
        log.Fatalf("Failed to start orchestrator: %v", err)
    }

    // Example task
    task := Task{
        ID:    "task1",
        Image: "nginx:latest",
    }
    if err := o.ScheduleTask(task); err != nil {
        log.Printf("Failed to schedule task: %v", err)
    }

    fmt.Println("Orchestrator running. Press Ctrl+C to stop.")
    select {} // Keep running
}