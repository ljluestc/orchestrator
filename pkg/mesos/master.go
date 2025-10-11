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

// Master represents a Mesos master node
type Master struct {
	ID           string
	Hostname     string
	Port         int
	ZookeeperURL string
	IsLeader     bool
	Agents       map[string]*AgentInfo
	Frameworks   map[string]*Framework
	Resources    *ResourcePool
	Offers       []*ResourceOffer
	State        *ClusterState
	mu           sync.RWMutex
	server       *http.Server
}

// AgentInfo represents agent information in the master
type AgentInfo struct {
	ID        string
	Hostname  string
	Port      int
	Resources *Resources
	Tasks     map[string]*Task
	Status    string
	LastSeen  time.Time
}

// Framework represents a registered framework
type Framework struct {
	ID           string
	Name         string
	Principal    string
	Role         string
	Hostname     string
	Port         int
	Status       string
	Tasks        map[string]*Task
	Offers       []*ResourceOffer
	RegisteredAt time.Time
}

// Resources represents available resources on an agent
type Resources struct {
	CPUs   float64
	Memory float64
	Disk   float64
	Ports  []PortRange
}

// PortRange represents a range of ports
type PortRange struct {
	Begin uint32
	End   uint32
}

// ResourceOffer represents a resource offer to a framework
type ResourceOffer struct {
	ID          string
	AgentID     string
	Resources   *Resources
	FrameworkID string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

// Task represents a running task
type Task struct {
	ID          string
	Name        string
	FrameworkID string
	AgentID     string
	State       string
	Resources   *Resources
	Command     *Command
	Container   *Container
	CreatedAt   time.Time
	StartedAt   time.Time
}

// Command represents a task command
type Command struct {
	Value string
	Shell bool
	User  string
}

// Container represents a container specification
type Container struct {
	Type   string
	Docker *DockerContainer
}

// DockerContainer represents Docker-specific container config
type DockerContainer struct {
	Image        string
	Network      string
	PortMappings []PortMapping
	Volumes      []Volume
}

// PortMapping represents port mapping
type PortMapping struct {
	ContainerPort int
	HostPort      int
	Protocol      string
}

// Volume represents a volume mount
type Volume struct {
	ContainerPath string
	HostPath      string
	Mode          string
}

// ResourcePool manages cluster resources
type ResourcePool struct {
	TotalCPUs       float64
	TotalMemory     float64
	TotalDisk       float64
	AvailableCPUs   float64
	AvailableMemory float64
	AvailableDisk   float64
	ReservedCPUs    float64
	ReservedMemory  float64
	ReservedDisk    float64
}

// ClusterState represents the current cluster state
type ClusterState struct {
	Version     string
	Leader      string
	Agents      map[string]*AgentInfo
	Frameworks  map[string]*Framework
	Tasks       map[string]*Task
	Offers      []*ResourceOffer
	LastUpdated time.Time
}

// NewMaster creates a new Mesos master
func NewMaster(id, hostname string, port int, zookeeperURL string) *Master {
	return &Master{
		ID:           id,
		Hostname:     hostname,
		Port:         port,
		ZookeeperURL: zookeeperURL,
		IsLeader:     false,
		Agents:       make(map[string]*AgentInfo),
		Frameworks:   make(map[string]*Framework),
		Resources:    &ResourcePool{},
		Offers:       make([]*ResourceOffer, 0),
		State: &ClusterState{
			Version:     "1.0.0",
			Agents:      make(map[string]*AgentInfo),
			Frameworks:  make(map[string]*Framework),
			Tasks:       make(map[string]*Task),
			Offers:      make([]*ResourceOffer, 0),
			LastUpdated: time.Now(),
		},
	}
}

// Start starts the Mesos master
func (m *Master) Start() error {
	router := m.setupRoutes()

	m.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", m.Port),
		Handler: router,
	}

	log.Printf("Starting Mesos master on %s:%d", m.Hostname, m.Port)

	// Start leader election process
	go m.startLeaderElection()

	// Start resource offer generation
	go m.startResourceOffering()

	// Start agent health monitoring
	go m.startAgentMonitoring()

	return m.server.ListenAndServe()
}

// Stop stops the Mesos master
func (m *Master) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return m.server.Shutdown(ctx)
}

// setupRoutes sets up HTTP routes
func (m *Master) setupRoutes() *mux.Router {
	router := mux.NewRouter()

	// API v1 routes
	v1 := router.PathPrefix("/api/v1").Subrouter()

	// Master info
	v1.HandleFunc("/master/info", m.handleMasterInfo).Methods("GET")
	v1.HandleFunc("/master/state", m.handleMasterState).Methods("GET")

	// Agent management
	v1.HandleFunc("/agents", m.handleListAgents).Methods("GET")
	v1.HandleFunc("/agents/{id}", m.handleGetAgent).Methods("GET")
	v1.HandleFunc("/agents/{id}/tasks", m.handleGetAgentTasks).Methods("GET")

	// Framework management
	v1.HandleFunc("/frameworks", m.handleListFrameworks).Methods("GET")
	v1.HandleFunc("/frameworks", m.handleRegisterFramework).Methods("POST")
	v1.HandleFunc("/frameworks/{id}", m.handleGetFramework).Methods("GET")
	v1.HandleFunc("/frameworks/{id}/tasks", m.handleGetFrameworkTasks).Methods("GET")

	// Task management
	v1.HandleFunc("/tasks", m.handleListTasks).Methods("GET")
	v1.HandleFunc("/tasks/{id}", m.handleGetTask).Methods("GET")
	v1.HandleFunc("/tasks/{id}/kill", m.handleKillTask).Methods("POST")

	// Resource offers
	v1.HandleFunc("/offers", m.handleListOffers).Methods("GET")
	v1.HandleFunc("/offers/{id}/accept", m.handleAcceptOffer).Methods("POST")
	v1.HandleFunc("/offers/{id}/decline", m.handleDeclineOffer).Methods("POST")

	// Health check
	router.HandleFunc("/health", m.handleHealth).Methods("GET")

	return router
}

// startLeaderElection starts the leader election process
func (m *Master) startLeaderElection() {
	// Simulate leader election via Zookeeper
	// In a real implementation, this would use Zookeeper for leader election
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// For now, assume this master is always the leader
			// In production, this would check Zookeeper for leadership
			m.mu.Lock()
			m.IsLeader = true
			m.State.Leader = m.ID
			m.mu.Unlock()

			log.Printf("Master %s is the leader", m.ID)
		}
	}
}

// startResourceOffering starts the resource offering process
func (m *Master) startResourceOffering() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.generateResourceOffers()
		}
	}
}

// startAgentMonitoring starts monitoring agent health
func (m *Master) startAgentMonitoring() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkAgentHealth()
		}
	}
}

// generateResourceOffers generates resource offers for frameworks
func (m *Master) generateResourceOffers() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate offers from available agent resources
	for _, agent := range m.Agents {
		if agent.Status == "active" && agent.Resources != nil {
			offer := &ResourceOffer{
				ID:        fmt.Sprintf("offer-%d", time.Now().UnixNano()),
				AgentID:   agent.ID,
				Resources: agent.Resources,
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(5 * time.Minute),
			}

			m.Offers = append(m.Offers, offer)
			m.State.Offers = append(m.State.Offers, offer)
		}
	}

	// Send offers to frameworks
	for _, framework := range m.Frameworks {
		if framework.Status == "active" {
			m.sendOffersToFramework(framework)
		}
	}
}

// sendOffersToFramework sends available offers to a framework
func (m *Master) sendOffersToFramework(framework *Framework) {
	// In a real implementation, this would send offers via the scheduler API
	log.Printf("Sending %d offers to framework %s", len(m.Offers), framework.ID)
}

// checkAgentHealth checks the health of all agents
func (m *Master) checkAgentHealth() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for id, agent := range m.Agents {
		if now.Sub(agent.LastSeen) > 2*time.Minute {
			agent.Status = "inactive"
			log.Printf("Agent %s is inactive (last seen: %v)", id, agent.LastSeen)
		}
	}
}

// RegisterAgent registers a new agent
func (m *Master) RegisterAgent(agent *AgentInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent.Status = "active"
	agent.LastSeen = time.Now()
	m.Agents[agent.ID] = agent
	m.State.Agents[agent.ID] = agent

	// Update resource pool
	if agent.Resources != nil {
		m.Resources.TotalCPUs += agent.Resources.CPUs
		m.Resources.TotalMemory += agent.Resources.Memory
		m.Resources.TotalDisk += agent.Resources.Disk
		m.Resources.AvailableCPUs += agent.Resources.CPUs
		m.Resources.AvailableMemory += agent.Resources.Memory
		m.Resources.AvailableDisk += agent.Resources.Disk
	}

	log.Printf("Registered agent %s (%s:%d)", agent.ID, agent.Hostname, agent.Port)
	return nil
}

// RegisterFramework registers a new framework
func (m *Master) RegisterFramework(framework *Framework) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	framework.Status = "active"
	framework.RegisteredAt = time.Now()
	framework.Tasks = make(map[string]*Task)
	framework.Offers = make([]*ResourceOffer, 0)

	m.Frameworks[framework.ID] = framework
	m.State.Frameworks[framework.ID] = framework

	log.Printf("Registered framework %s (%s)", framework.ID, framework.Name)
	return nil
}

// LaunchTask launches a task on an agent
func (m *Master) LaunchTask(task *Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find the agent
	agent, exists := m.Agents[task.AgentID]
	if !exists {
		return fmt.Errorf("agent %s not found", task.AgentID)
	}

	// Update task state
	task.State = "starting"
	task.CreatedAt = time.Now()

	// Add to agent
	agent.Tasks[task.ID] = task

	// Add to framework
	if framework, exists := m.Frameworks[task.FrameworkID]; exists {
		framework.Tasks[task.ID] = task
	}

	// Add to global state
	m.State.Tasks[task.ID] = task

	log.Printf("Launched task %s on agent %s", task.ID, task.AgentID)
	return nil
}

// KillTask kills a running task
func (m *Master) KillTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.State.Tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Update task state
	task.State = "killed"

	// Remove from agent
	if agent, exists := m.Agents[task.AgentID]; exists {
		delete(agent.Tasks, taskID)
	}

	// Remove from framework
	if framework, exists := m.Frameworks[task.FrameworkID]; exists {
		delete(framework.Tasks, taskID)
	}

	// Remove from global state
	delete(m.State.Tasks, taskID)

	log.Printf("Killed task %s", taskID)
	return nil
}

// HTTP handlers
func (m *Master) handleMasterInfo(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info := map[string]interface{}{
		"id":       m.ID,
		"hostname": m.Hostname,
		"port":     m.Port,
		"leader":   m.IsLeader,
		"version":  m.State.Version,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (m *Master) handleMasterState(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m.State)
}

func (m *Master) handleListAgents(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agents := make([]*AgentInfo, 0, len(m.Agents))
	for _, agent := range m.Agents {
		agents = append(agents, agent)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

func (m *Master) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, exists := m.Agents[agentID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)
}

func (m *Master) handleGetAgentTasks(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, exists := m.Agents[agentID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	tasks := make([]*Task, 0, len(agent.Tasks))
	for _, task := range agent.Tasks {
		tasks = append(tasks, task)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func (m *Master) handleListFrameworks(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	frameworks := make([]*Framework, 0, len(m.Frameworks))
	for _, framework := range m.Frameworks {
		frameworks = append(frameworks, framework)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(frameworks)
}

func (m *Master) handleRegisterFramework(w http.ResponseWriter, r *http.Request) {
	var framework Framework
	if err := json.NewDecoder(r.Body).Decode(&framework); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := m.RegisterFramework(&framework); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(framework)
}

func (m *Master) handleGetFramework(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	frameworkID := vars["id"]

	m.mu.RLock()
	defer m.mu.RUnlock()

	framework, exists := m.Frameworks[frameworkID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(framework)
}

func (m *Master) handleGetFrameworkTasks(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	frameworkID := vars["id"]

	m.mu.RLock()
	defer m.mu.RUnlock()

	framework, exists := m.Frameworks[frameworkID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	tasks := make([]*Task, 0, len(framework.Tasks))
	for _, task := range framework.Tasks {
		tasks = append(tasks, task)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func (m *Master) handleListTasks(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]*Task, 0, len(m.State.Tasks))
	for _, task := range m.State.Tasks {
		tasks = append(tasks, task)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func (m *Master) handleGetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.State.Tasks[taskID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (m *Master) handleKillTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	if err := m.KillTask(taskID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (m *Master) handleListOffers(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m.Offers)
}

func (m *Master) handleAcceptOffer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	offerID := vars["id"]

	// In a real implementation, this would accept the offer and launch tasks
	log.Printf("Accepting offer %s", offerID)

	w.WriteHeader(http.StatusOK)
}

func (m *Master) handleDeclineOffer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	offerID := vars["id"]

	// In a real implementation, this would decline the offer
	log.Printf("Declining offer %s", offerID)

	w.WriteHeader(http.StatusOK)
}

func (m *Master) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
