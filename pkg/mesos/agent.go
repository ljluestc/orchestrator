package mesos

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// Agent represents a Mesos agent node
type Agent struct {
	ID              string
	Hostname        string
	Port            int
	MasterURL       string
	Resources       *Resources
	Tasks           map[string]*Task
	Executors       map[string]*Executor
	Status          string
	LastSeen        time.Time
	mu              sync.RWMutex
	server          *http.Server
	heartbeatTicker *time.Ticker
}

// Executor represents a task executor
type Executor struct {
	ID          string
	FrameworkID string
	AgentID     string
	Status      string
	Tasks       map[string]*Task
	CreatedAt   time.Time
}

// NewAgent creates a new Mesos agent
func NewAgent(id, hostname string, port int, masterURL string) *Agent {
	return &Agent{
		ID:        id,
		Hostname:  hostname,
		Port:      port,
		MasterURL: masterURL,
		Resources: &Resources{
			CPUs:   4.0,
			Memory: 8192.0,   // 8GB in MB
			Disk:   100000.0, // 100GB in MB
			Ports: []PortRange{
				{Begin: 31000, End: 32000},
			},
		},
		Tasks:     make(map[string]*Task),
		Executors: make(map[string]*Executor),
		Status:    "inactive",
	}
}

// Start starts the Mesos agent
func (a *Agent) Start() error {
	router := a.setupRoutes()

	a.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", a.Port),
		Handler: router,
	}

	log.Printf("Starting Mesos agent on %s:%d", a.Hostname, a.Port)

	// Start heartbeat to master
	go a.startHeartbeat()

	// Start task monitoring
	go a.startTaskMonitoring()

	// Register with master
	if err := a.registerWithMaster(); err != nil {
		log.Printf("Failed to register with master: %v", err)
	}

	return a.server.ListenAndServe()
}

// Stop stops the Mesos agent
func (a *Agent) Stop() error {
	if a.heartbeatTicker != nil {
		a.heartbeatTicker.Stop()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return a.server.Shutdown(ctx)
}

// setupRoutes sets up HTTP routes
func (a *Agent) setupRoutes() *mux.Router {
	router := mux.NewRouter()

	// API v1 routes
	v1 := router.PathPrefix("/api/v1").Subrouter()

	// Agent info
	v1.HandleFunc("/agent/info", a.handleAgentInfo).Methods("GET")
	v1.HandleFunc("/agent/state", a.handleAgentState).Methods("GET")

	// Task management
	v1.HandleFunc("/tasks", a.handleListTasks).Methods("GET")
	v1.HandleFunc("/tasks", a.handleLaunchTask).Methods("POST")
	v1.HandleFunc("/tasks/{id}", a.handleGetTask).Methods("GET")
	v1.HandleFunc("/tasks/{id}/kill", a.handleKillTask).Methods("POST")
	v1.HandleFunc("/tasks/{id}/status", a.handleTaskStatus).Methods("GET")

	// Executor management
	v1.HandleFunc("/executors", a.handleListExecutors).Methods("GET")
	v1.HandleFunc("/executors/{id}", a.handleGetExecutor).Methods("GET")

	// Resource management
	v1.HandleFunc("/resources", a.handleGetResources).Methods("GET")

	// Health check
	router.HandleFunc("/health", a.handleHealth).Methods("GET")

	return router
}

// startHeartbeat starts sending heartbeats to the master
func (a *Agent) startHeartbeat() {
	a.heartbeatTicker = time.NewTicker(30 * time.Second)
	defer a.heartbeatTicker.Stop()

	for {
		select {
		case <-a.heartbeatTicker.C:
			a.sendHeartbeat()
		}
	}
}

// startTaskMonitoring starts monitoring running tasks
func (a *Agent) startTaskMonitoring() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.monitorTasks()
		}
	}
}

// registerWithMaster registers the agent with the master
func (a *Agent) registerWithMaster() error {
	// In a real implementation, this would register via the master's API
	log.Printf("Registering agent %s with master %s", a.ID, a.MasterURL)

	a.mu.Lock()
	a.Status = "active"
	a.LastSeen = time.Now()
	a.mu.Unlock()

	return nil
}

// sendHeartbeat sends a heartbeat to the master
func (a *Agent) sendHeartbeat() {
	a.mu.Lock()
	a.LastSeen = time.Now()
	a.mu.Unlock()

	// In a real implementation, this would send heartbeat via HTTP
	log.Printf("Sending heartbeat from agent %s", a.ID)
}

// monitorTasks monitors running tasks
func (a *Agent) monitorTasks() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, task := range a.Tasks {
		// Simulate task monitoring
		if task.State == "starting" {
			task.State = "running"
			task.StartedAt = time.Now()
		}
	}
}

// LaunchTask launches a task on this agent
func (a *Agent) LaunchTask(task *Task) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if we have enough resources
	if !a.hasResources(task.Resources) {
		return fmt.Errorf("insufficient resources for task %s", task.ID)
	}

	// Allocate resources
	a.allocateResources(task.Resources)

	// Create executor if needed
	executorID := fmt.Sprintf("executor-%s", task.FrameworkID)
	executor, exists := a.Executors[executorID]
	if !exists {
		executor = &Executor{
			ID:          executorID,
			FrameworkID: task.FrameworkID,
			AgentID:     a.ID,
			Status:      "active",
			Tasks:       make(map[string]*Task),
			CreatedAt:   time.Now(),
		}
		a.Executors[executorID] = executor
	}

	// Add task to executor
	executor.Tasks[task.ID] = task

	// Add task to agent
	task.AgentID = a.ID
	task.State = "starting"
	task.CreatedAt = time.Now()
	a.Tasks[task.ID] = task

	log.Printf("Launched task %s on agent %s", task.ID, a.ID)
	return nil
}

// KillTask kills a running task
func (a *Agent) KillTask(taskID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	task, exists := a.Tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Release resources
	a.releaseResources(task.Resources)

	// Update task state
	task.State = "killed"

	// Remove from executor
	if executor, exists := a.Executors[fmt.Sprintf("executor-%s", task.FrameworkID)]; exists {
		delete(executor.Tasks, taskID)
	}

	// Remove from agent
	delete(a.Tasks, taskID)

	log.Printf("Killed task %s on agent %s", taskID, a.ID)
	return nil
}

// hasResources checks if the agent has enough resources
func (a *Agent) hasResources(required *Resources) bool {
	if required == nil {
		return true
	}

	// Calculate available resources
	available := a.calculateAvailableResources()

	return required.CPUs <= available.CPUs &&
		required.Memory <= available.Memory &&
		required.Disk <= available.Disk
}

// calculateAvailableResources calculates available resources
func (a *Agent) calculateAvailableResources() *Resources {
	used := &Resources{}

	for _, task := range a.Tasks {
		if task.Resources != nil && task.State == "running" {
			used.CPUs += task.Resources.CPUs
			used.Memory += task.Resources.Memory
			used.Disk += task.Resources.Disk
		}
	}

	return &Resources{
		CPUs:   a.Resources.CPUs - used.CPUs,
		Memory: a.Resources.Memory - used.Memory,
		Disk:   a.Resources.Disk - used.Disk,
		Ports:  a.Resources.Ports, // Ports are handled separately
	}
}

// allocateResources allocates resources for a task
func (a *Agent) allocateResources(resources *Resources) {
	// In a real implementation, this would update resource accounting
	log.Printf("Allocated resources: CPU=%.2f, Memory=%.2f, Disk=%.2f",
		resources.CPUs, resources.Memory, resources.Disk)
}

// releaseResources releases resources from a task
func (a *Agent) releaseResources(resources *Resources) {
	// In a real implementation, this would update resource accounting
	log.Printf("Released resources: CPU=%.2f, Memory=%.2f, Disk=%.2f",
		resources.CPUs, resources.Memory, resources.Disk)
}

// HTTP handlers
func (a *Agent) handleAgentInfo(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	info := map[string]interface{}{
		"id":        a.ID,
		"hostname":  a.Hostname,
		"port":      a.Port,
		"status":    a.Status,
		"resources": a.Resources,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (a *Agent) handleAgentState(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	state := map[string]interface{}{
		"id":        a.ID,
		"hostname":  a.Hostname,
		"status":    a.Status,
		"resources": a.Resources,
		"tasks":     a.Tasks,
		"executors": a.Executors,
		"last_seen": a.LastSeen,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

func (a *Agent) handleListTasks(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	tasks := make([]*Task, 0, len(a.Tasks))
	for _, task := range a.Tasks {
		tasks = append(tasks, task)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func (a *Agent) handleLaunchTask(w http.ResponseWriter, r *http.Request) {
	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := a.LaunchTask(&task); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (a *Agent) handleGetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	a.mu.RLock()
	defer a.mu.RUnlock()

	task, exists := a.Tasks[taskID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (a *Agent) handleKillTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	if err := a.KillTask(taskID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *Agent) handleTaskStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	a.mu.RLock()
	defer a.mu.RUnlock()

	task, exists := a.Tasks[taskID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	status := map[string]interface{}{
		"id":         task.ID,
		"state":      task.State,
		"created_at": task.CreatedAt,
		"started_at": task.StartedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (a *Agent) handleListExecutors(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	executors := make([]*Executor, 0, len(a.Executors))
	for _, executor := range a.Executors {
		executors = append(executors, executor)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(executors)
}

func (a *Agent) handleGetExecutor(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	executorID := vars["id"]

	a.mu.RLock()
	defer a.mu.RUnlock()

	executor, exists := a.Executors[executorID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(executor)
}

func (a *Agent) handleGetResources(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	resources := map[string]interface{}{
		"total":     a.Resources,
		"available": a.calculateAvailableResources(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resources)
}

func (a *Agent) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
