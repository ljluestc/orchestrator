package marathon

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

// Marathon represents the Marathon framework
type Marathon struct {
	ID           string
	Hostname     string
	Port         int
	MasterURL    string
	Applications map[string]*Application
	Deployments  map[string]*Deployment
	Tasks        map[string]*MarathonTask
	mu           sync.RWMutex
	server       *http.Server
}

// Application represents a Marathon application
type Application struct {
	ID              string            `json:"id"`
	Container       *Container        `json:"container,omitempty"`
	Instances       int               `json:"instances"`
	CPUs            float64           `json:"cpus"`
	Memory          float64           `json:"mem"`
	HealthChecks    []*HealthCheck    `json:"healthChecks,omitempty"`
	Constraints     [][]string        `json:"constraints,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	Env             map[string]string `json:"env,omitempty"`
	Tasks           []*MarathonTask   `json:"tasks,omitempty"`
	Deployments     []*Deployment     `json:"deployments,omitempty"`
	Version         string            `json:"version"`
	LastTaskFailure *TaskFailure      `json:"lastTaskFailure,omitempty"`
	TasksStaged     int               `json:"tasksStaged"`
	TasksRunning    int               `json:"tasksRunning"`
	TasksHealthy    int               `json:"tasksHealthy"`
	TasksUnhealthy  int               `json:"tasksUnhealthy"`
}

// MarathonTask represents a Marathon task
type MarathonTask struct {
	ID                 string               `json:"id"`
	AppID              string               `json:"appId"`
	Host               string               `json:"host"`
	Ports              []int                `json:"ports"`
	StartedAt          *time.Time           `json:"startedAt,omitempty"`
	StagedAt           *time.Time           `json:"stagedAt,omitempty"`
	Version            string               `json:"version"`
	State              string               `json:"state"`
	HealthCheckResults []*HealthCheckResult `json:"healthCheckResults,omitempty"`
	ServicePorts       []int                `json:"servicePorts,omitempty"`
	IPAddresses        []*IPAddress         `json:"ipAddresses,omitempty"`
}

// Deployment represents a Marathon deployment
type Deployment struct {
	ID                    string                  `json:"id"`
	Version               string                  `json:"version"`
	AffectedApps          []string                `json:"affectedApps"`
	Steps                 []*DeploymentStep       `json:"steps"`
	CurrentActions        []*DeploymentAction     `json:"currentActions"`
	CurrentStep           int                     `json:"currentStep"`
	TotalSteps            int                     `json:"totalSteps"`
	ReadinessCheckResults []*ReadinessCheckResult `json:"readinessCheckResults,omitempty"`
}

// Container represents a container specification
type Container struct {
	Type   string      `json:"type"`
	Docker *DockerSpec `json:"docker,omitempty"`
}

// DockerSpec represents Docker-specific configuration
type DockerSpec struct {
	Image          string         `json:"image"`
	Network        string         `json:"network,omitempty"`
	PortMappings   []*PortMapping `json:"portMappings,omitempty"`
	Parameters     []*Parameter   `json:"parameters,omitempty"`
	Privileged     bool           `json:"privileged,omitempty"`
	ForcePullImage bool           `json:"forcePullImage,omitempty"`
}

// PortMapping represents port mapping
type PortMapping struct {
	ContainerPort int    `json:"containerPort"`
	HostPort      int    `json:"hostPort"`
	ServicePort   int    `json:"servicePort,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
}

// Parameter represents a Docker parameter
type Parameter struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// HealthCheck represents a health check
type HealthCheck struct {
	Protocol               string `json:"protocol"`
	Path                   string `json:"path,omitempty"`
	PortIndex              int    `json:"portIndex,omitempty"`
	Port                   int    `json:"port,omitempty"`
	GracePeriodSeconds     int    `json:"gracePeriodSeconds,omitempty"`
	IntervalSeconds        int    `json:"intervalSeconds,omitempty"`
	TimeoutSeconds         int    `json:"timeoutSeconds,omitempty"`
	MaxConsecutiveFailures int    `json:"maxConsecutiveFailures,omitempty"`
	IgnoreHTTP1xx          bool   `json:"ignoreHttp1xx,omitempty"`
}

// HealthCheckResult represents a health check result
type HealthCheckResult struct {
	Alive               bool       `json:"alive"`
	ConsecutiveFailures int        `json:"consecutiveFailures"`
	FirstSuccess        *time.Time `json:"firstSuccess,omitempty"`
	LastFailure         *time.Time `json:"lastFailure,omitempty"`
	LastSuccess         *time.Time `json:"lastSuccess,omitempty"`
	LastFailureCause    string     `json:"lastFailureCause,omitempty"`
}

// IPAddress represents an IP address
type IPAddress struct {
	IPAddress string `json:"ipAddress"`
	Protocol  string `json:"protocol"`
}

// TaskFailure represents a task failure
type TaskFailure struct {
	AppID     string    `json:"appId"`
	TaskID    string    `json:"taskId"`
	State     string    `json:"state"`
	Message   string    `json:"message"`
	Host      string    `json:"host"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
}

// DeploymentStep represents a deployment step
type DeploymentStep struct {
	Action string `json:"action"`
	App    string `json:"app"`
}

// DeploymentAction represents a deployment action
type DeploymentAction struct {
	Action string `json:"action"`
	App    string `json:"app"`
}

// ReadinessCheckResult represents a readiness check result
type ReadinessCheckResult struct {
	TaskID       string                  `json:"taskId"`
	LastResponse *ReadinessCheckResponse `json:"lastResponse,omitempty"`
}

// ReadinessCheckResponse represents a readiness check response
type ReadinessCheckResponse struct {
	Status  int               `json:"status"`
	Body    string            `json:"body,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// NewMarathon creates a new Marathon framework
func NewMarathon(id, hostname string, port int, masterURL string) *Marathon {
	return &Marathon{
		ID:           id,
		Hostname:     hostname,
		Port:         port,
		MasterURL:    masterURL,
		Applications: make(map[string]*Application),
		Deployments:  make(map[string]*Deployment),
		Tasks:        make(map[string]*MarathonTask),
	}
}

// Start starts the Marathon framework
func (m *Marathon) Start() error {
	router := m.setupRoutes()

	m.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", m.Port),
		Handler: router,
	}

	log.Printf("Starting Marathon framework on %s:%d", m.Hostname, m.Port)

	// Start task monitoring
	go m.startTaskMonitoring()

	// Register with Mesos master
	go m.registerWithMaster()

	return m.server.ListenAndServe()
}

// Stop stops the Marathon framework
func (m *Marathon) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return m.server.Shutdown(ctx)
}

// setupRoutes sets up HTTP routes
func (m *Marathon) setupRoutes() *mux.Router {
	router := mux.NewRouter()

	// API v2 routes
	v2 := router.PathPrefix("/v2").Subrouter()

	// Applications
	v2.HandleFunc("/apps", m.handleListApps).Methods("GET")
	v2.HandleFunc("/apps", m.handleCreateApp).Methods("POST")
	v2.HandleFunc("/apps/{id}", m.handleGetApp).Methods("GET")
	v2.HandleFunc("/apps/{id}", m.handleUpdateApp).Methods("PUT")
	v2.HandleFunc("/apps/{id}", m.handleDeleteApp).Methods("DELETE")
	v2.HandleFunc("/apps/{id}/restart", m.handleRestartApp).Methods("POST")
	v2.HandleFunc("/apps/{id}/scale", m.handleScaleApp).Methods("PUT")

	// Tasks
	v2.HandleFunc("/tasks", m.handleListTasks).Methods("GET")
	v2.HandleFunc("/apps/{id}/tasks", m.handleListAppTasks).Methods("GET")
	v2.HandleFunc("/tasks/{id}", m.handleGetTask).Methods("GET")
	v2.HandleFunc("/tasks/{id}/kill", m.handleKillTask).Methods("DELETE")

	// Deployments
	v2.HandleFunc("/deployments", m.handleListDeployments).Methods("GET")
	v2.HandleFunc("/deployments/{id}", m.handleGetDeployment).Methods("GET")
	v2.HandleFunc("/deployments/{id}", m.handleDeleteDeployment).Methods("DELETE")

	// Health checks
	v2.HandleFunc("/apps/{id}/health", m.handleAppHealth).Methods("GET")

	// Health check
	router.HandleFunc("/ping", m.handlePing).Methods("GET")
	router.HandleFunc("/health", m.handleHealth).Methods("GET")

	return router
}

// startTaskMonitoring starts monitoring running tasks
func (m *Marathon) startTaskMonitoring() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.monitorTasks()
		}
	}
}

// registerWithMaster registers with Mesos master
func (m *Marathon) registerWithMaster() {
	// In a real implementation, this would register via Mesos scheduler API
	log.Printf("Registering Marathon framework %s with master %s", m.ID, m.MasterURL)
}

// monitorTasks monitors running tasks
func (m *Marathon) monitorTasks() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, task := range m.Tasks {
		// Simulate task health monitoring
		if task.State == "TASK_STAGING" {
			task.State = "TASK_RUNNING"
			now := time.Now()
			task.StartedAt = &now
		}
	}
}

// CreateApp creates a new application
func (m *Marathon) CreateApp(app *Application) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	app.Version = time.Now().Format("2006-01-02T15:04:05.000Z")
	app.Tasks = make([]*MarathonTask, 0)
	app.Deployments = make([]*Deployment, 0)
	app.TasksStaged = 0
	app.TasksRunning = 0
	app.TasksHealthy = 0
	app.TasksUnhealthy = 0

	m.Applications[app.ID] = app

	// Create initial deployment
	deployment := &Deployment{
		ID:             fmt.Sprintf("deployment-%d", time.Now().UnixNano()),
		Version:        app.Version,
		AffectedApps:   []string{app.ID},
		Steps:          []*DeploymentStep{{Action: "StartApplication", App: app.ID}},
		CurrentActions: []*DeploymentAction{{Action: "StartApplication", App: app.ID}},
		CurrentStep:    0,
		TotalSteps:     1,
	}

	m.Deployments[deployment.ID] = deployment
	app.Deployments = append(app.Deployments, deployment)

	// Launch tasks
	for i := 0; i < app.Instances; i++ {
		task := m.createTask(app, i)
		m.Tasks[task.ID] = task
		app.Tasks = append(app.Tasks, task)
		app.TasksStaged++
	}

	log.Printf("Created application %s with %d instances", app.ID, app.Instances)
	return nil
}

// createTask creates a task for an application
func (m *Marathon) createTask(app *Application, index int) *MarathonTask {
	taskID := fmt.Sprintf("%s.%d", app.ID, index)

	return &MarathonTask{
		ID:       taskID,
		AppID:    app.ID,
		Host:     "localhost", // In real implementation, this would be assigned by Mesos
		Ports:    []int{8080 + index},
		Version:  app.Version,
		State:    "TASK_STAGING",
		StagedAt: &[]time.Time{time.Now()}[0],
	}
}

// UpdateApp updates an existing application
func (m *Marathon) UpdateApp(appID string, app *Application) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.Applications[appID]
	if !exists {
		return fmt.Errorf("application %s not found", appID)
	}

	// Update application
	app.Version = time.Now().Format("2006-01-02T15:04:05.000Z")
	app.Tasks = existing.Tasks
	app.Deployments = existing.Deployments

	m.Applications[appID] = app

	// Create deployment for update
	deployment := &Deployment{
		ID:             fmt.Sprintf("deployment-%d", time.Now().UnixNano()),
		Version:        app.Version,
		AffectedApps:   []string{appID},
		Steps:          []*DeploymentStep{{Action: "RestartApplication", App: appID}},
		CurrentActions: []*DeploymentAction{{Action: "RestartApplication", App: appID}},
		CurrentStep:    0,
		TotalSteps:     1,
	}

	m.Deployments[deployment.ID] = deployment
	app.Deployments = append(app.Deployments, deployment)

	log.Printf("Updated application %s", appID)
	return nil
}

// DeleteApp deletes an application
func (m *Marathon) DeleteApp(appID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	app, exists := m.Applications[appID]
	if !exists {
		return fmt.Errorf("application %s not found", appID)
	}

	// Kill all tasks
	for _, task := range app.Tasks {
		delete(m.Tasks, task.ID)
	}

	// Remove application
	delete(m.Applications, appID)

	log.Printf("Deleted application %s", appID)
	return nil
}

// ScaleApp scales an application
func (m *Marathon) ScaleApp(appID string, instances int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	app, exists := m.Applications[appID]
	if !exists {
		return fmt.Errorf("application %s not found", appID)
	}

	oldInstances := app.Instances
	app.Instances = instances

	if instances > oldInstances {
		// Scale up - add new tasks
		for i := oldInstances; i < instances; i++ {
			task := m.createTask(app, i)
			m.Tasks[task.ID] = task
			app.Tasks = append(app.Tasks, task)
			app.TasksStaged++
		}
	} else if instances < oldInstances {
		// Scale down - remove tasks
		for i := instances; i < oldInstances; i++ {
			taskID := fmt.Sprintf("%s.%d", appID, i)
			if task, exists := m.Tasks[taskID]; exists {
				task.State = "TASK_KILLED"
				delete(m.Tasks, taskID)
				app.TasksRunning--
			}
		}
	}

	log.Printf("Scaled application %s from %d to %d instances", appID, oldInstances, instances)
	return nil
}

// HTTP handlers
func (m *Marathon) handleListApps(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	apps := make([]*Application, 0, len(m.Applications))
	for _, app := range m.Applications {
		apps = append(apps, app)
	}

	response := map[string]interface{}{
		"apps": apps,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *Marathon) handleCreateApp(w http.ResponseWriter, r *http.Request) {
	var app Application
	if err := json.NewDecoder(r.Body).Decode(&app); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := m.CreateApp(&app); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app)
}

func (m *Marathon) handleGetApp(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appID := vars["id"]

	m.mu.RLock()
	defer m.mu.RUnlock()

	app, exists := m.Applications[appID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app)
}

func (m *Marathon) handleUpdateApp(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appID := vars["id"]

	var app Application
	if err := json.NewDecoder(r.Body).Decode(&app); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := m.UpdateApp(appID, &app); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app)
}

func (m *Marathon) handleDeleteApp(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appID := vars["id"]

	if err := m.DeleteApp(appID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (m *Marathon) handleRestartApp(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appID := vars["id"]

	// In a real implementation, this would restart the application
	log.Printf("Restarting application %s", appID)

	w.WriteHeader(http.StatusOK)
}

func (m *Marathon) handleScaleApp(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appID := vars["id"]

	var request struct {
		Instances int `json:"instances"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := m.ScaleApp(appID, request.Instances); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (m *Marathon) handleListTasks(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]*MarathonTask, 0, len(m.Tasks))
	for _, task := range m.Tasks {
		tasks = append(tasks, task)
	}

	response := map[string]interface{}{
		"tasks": tasks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *Marathon) handleListAppTasks(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appID := vars["id"]

	m.mu.RLock()
	defer m.mu.RUnlock()

	app, exists := m.Applications[appID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app.Tasks)
}

func (m *Marathon) handleGetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.Tasks[taskID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (m *Marathon) handleKillTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	m.mu.Lock()
	defer m.mu.Unlock()

	if task, exists := m.Tasks[taskID]; exists {
		task.State = "TASK_KILLED"
	}

	w.WriteHeader(http.StatusOK)
}

func (m *Marathon) handleListDeployments(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	deployments := make([]*Deployment, 0, len(m.Deployments))
	for _, deployment := range m.Deployments {
		deployments = append(deployments, deployment)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deployments)
}

func (m *Marathon) handleGetDeployment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	m.mu.RLock()
	defer m.mu.RUnlock()

	deployment, exists := m.Deployments[deploymentID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deployment)
}

func (m *Marathon) handleDeleteDeployment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.Deployments, deploymentID)

	w.WriteHeader(http.StatusOK)
}

func (m *Marathon) handleAppHealth(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appID := vars["id"]

	m.mu.RLock()
	defer m.mu.RUnlock()

	app, exists := m.Applications[appID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	health := map[string]interface{}{
		"tasksRunning":   app.TasksRunning,
		"tasksHealthy":   app.TasksHealthy,
		"tasksUnhealthy": app.TasksUnhealthy,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (m *Marathon) handlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *Marathon) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
