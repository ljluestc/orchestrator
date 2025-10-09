package topology

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Manager manages the topology visualization system
type Manager struct {
	ID           string
	Nodes        map[string]*Node
	Edges        map[string]*Edge
	Views        map[string]*View
	Metrics      map[string]*Metrics
	Subscribers  map[string]*Subscriber
	mu           sync.RWMutex
	server       *http.Server
	upgrader     websocket.Upgrader
	updateTicker *time.Ticker
}

// Node represents a node in the topology graph
type Node struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // host, container, process, pod, service
	Name        string                 `json:"name"`
	Status      string                 `json:"status"` // healthy, warning, critical, unknown
	Metadata    map[string]interface{} `json:"metadata"`
	Metrics     *NodeMetrics           `json:"metrics"`
	Position    *Position              `json:"position,omitempty"`
	Size        *Size                  `json:"size,omitempty"`
	Color       string                 `json:"color,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	LastSeen    time.Time              `json:"last_seen"`
}

// Edge represents a connection between nodes
type Edge struct {
	ID          string                 `json:"id"`
	Source      string                 `json:"source"`
	Target      string                 `json:"target"`
	Type        string                 `json:"type"` // network, process, container, service
	Weight      float64                `json:"weight"`
	Metadata    map[string]interface{} `json:"metadata"`
	Metrics     *EdgeMetrics           `json:"metrics"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// View represents a specific topology view
type View struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	NodeTypes   []string `json:"node_types"`
	EdgeTypes   []string `json:"edge_types"`
	Filter      *Filter  `json:"filter,omitempty"`
	Layout      *Layout  `json:"layout,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Filter represents filtering criteria
type Filter struct {
	Search     string                 `json:"search,omitempty"`
	NodeTypes  []string               `json:"node_types,omitempty"`
	Status     []string               `json:"status,omitempty"`
	Labels     map[string]string      `json:"labels,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Metrics    *MetricsFilter         `json:"metrics,omitempty"`
}

// MetricsFilter represents metrics-based filtering
type MetricsFilter struct {
	CPUUsage    *RangeFilter `json:"cpu_usage,omitempty"`
	MemoryUsage *RangeFilter `json:"memory_usage,omitempty"`
	Connections *RangeFilter `json:"connections,omitempty"`
}

// RangeFilter represents a range filter
type RangeFilter struct {
	Min float64 `json:"min,omitempty"`
	Max float64 `json:"max,omitempty"`
}

// Layout represents graph layout configuration
type Layout struct {
	Type       string  `json:"type"` // force, hierarchical, circular, grid
	Width      int     `json:"width,omitempty"`
	Height     int     `json:"height,omitempty"`
	Iterations int     `json:"iterations,omitempty"`
	Strength   float64 `json:"strength,omitempty"`
}

// Position represents node position
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z,omitempty"`
}

// Size represents node size
type Size struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// NodeMetrics represents node metrics
type NodeMetrics struct {
	CPUUsage     *Sparkline `json:"cpu_usage,omitempty"`
	MemoryUsage  *Sparkline `json:"memory_usage,omitempty"`
	NetworkIn    *Sparkline `json:"network_in,omitempty"`
	NetworkOut   *Sparkline `json:"network_out,omitempty"`
	DiskUsage    *Sparkline `json:"disk_usage,omitempty"`
	Connections  *Sparkline `json:"connections,omitempty"`
	ProcessCount *Sparkline `json:"process_count,omitempty"`
}

// EdgeMetrics represents edge metrics
type EdgeMetrics struct {
	BytesIn     *Sparkline `json:"bytes_in,omitempty"`
	BytesOut    *Sparkline `json:"bytes_out,omitempty"`
	PacketsIn   *Sparkline `json:"packets_in,omitempty"`
	PacketsOut  *Sparkline `json:"packets_out,omitempty"`
	Latency     *Sparkline `json:"latency,omitempty"`
	ErrorRate   *Sparkline `json:"error_rate,omitempty"`
}

// Sparkline represents a time-series sparkline
type Sparkline struct {
	Values []float64 `json:"values"`
	Times  []int64   `json:"times"`
	Min    float64   `json:"min"`
	Max    float64   `json:"max"`
	Avg    float64   `json:"avg"`
	Current float64  `json:"current"`
}

// Metrics represents overall metrics
type Metrics struct {
	TotalNodes    int       `json:"total_nodes"`
	TotalEdges    int       `json:"total_edges"`
	HealthyNodes  int       `json:"healthy_nodes"`
	WarningNodes  int       `json:"warning_nodes"`
	CriticalNodes int       `json:"critical_nodes"`
	UnknownNodes  int       `json:"unknown_nodes"`
	LastUpdated   time.Time `json:"last_updated"`
}

// Subscriber represents a WebSocket subscriber
type Subscriber struct {
	ID       string
	Conn     *websocket.Conn
	Filter   *Filter
	LastPing time.Time
	Send     chan []byte
	Done     chan bool
}

// TopologyUpdate represents a topology update
type TopologyUpdate struct {
	Type    string      `json:"type"` // add, update, remove
	Node    *Node       `json:"node,omitempty"`
	Edge    *Edge       `json:"edge,omitempty"`
	Metrics *Metrics    `json:"metrics,omitempty"`
	View    *View       `json:"view,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// NewManager creates a new topology manager
func NewManager(id string) *Manager {
	return &Manager{
		ID:          id,
		Nodes:       make(map[string]*Node),
		Edges:       make(map[string]*Edge),
		Views:       make(map[string]*View),
		Metrics:     make(map[string]*Metrics),
		Subscribers: make(map[string]*Subscriber),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
	}
}

// Start starts the topology manager
func (m *Manager) Start() error {
	router := m.setupRoutes()
	
	m.server = &http.Server{
		Addr:    ":8082",
		Handler: router,
	}

	log.Printf("Starting topology manager on :8082")
	
	// Initialize default views
	m.initializeDefaultViews()
	
	// Start metrics collection
	go m.startMetricsCollection()
	
	// Start WebSocket cleanup
	go m.startWebSocketCleanup()
	
	return m.server.ListenAndServe()
}

// Stop stops the topology manager
func (m *Manager) Stop() error {
	if m.updateTicker != nil {
		m.updateTicker.Stop()
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return m.server.Shutdown(ctx)
}

// setupRoutes sets up HTTP routes
func (m *Manager) setupRoutes() *mux.Router {
	router := mux.NewRouter()
	
	// API v1 routes
	v1 := router.PathPrefix("/api/v1").Subrouter()
	
	// Topology endpoints
	v1.HandleFunc("/topology", m.handleGetTopology).Methods("GET")
	v1.HandleFunc("/topology/nodes", m.handleGetNodes).Methods("GET")
	v1.HandleFunc("/topology/nodes", m.handleAddNode).Methods("POST")
	v1.HandleFunc("/topology/nodes/{id}", m.handleGetNode).Methods("GET")
	v1.HandleFunc("/topology/edges", m.handleGetEdges).Methods("GET")
	v1.HandleFunc("/topology/edges", m.handleAddEdge).Methods("POST")
	v1.HandleFunc("/topology/edges/{id}", m.handleGetEdge).Methods("GET")
	v1.HandleFunc("/topology/search", m.handleSearch).Methods("GET")
	v1.HandleFunc("/topology/filter", m.handleFilter).Methods("POST")
	
	// Views endpoints
	v1.HandleFunc("/views", m.handleGetViews).Methods("GET")
	v1.HandleFunc("/views/{id}", m.handleGetView).Methods("GET")
	v1.HandleFunc("/views", m.handleCreateView).Methods("POST")
	v1.HandleFunc("/views/{id}", m.handleUpdateView).Methods("PUT")
	v1.HandleFunc("/views/{id}", m.handleDeleteView).Methods("DELETE")
	
	// Metrics endpoints
	v1.HandleFunc("/metrics", m.handleGetMetrics).Methods("GET")
	v1.HandleFunc("/metrics/nodes/{id}", m.handleGetNodeMetrics).Methods("GET")
	v1.HandleFunc("/metrics/edges/{id}", m.handleGetEdgeMetrics).Methods("GET")
	
	// Container control endpoints
	v1.HandleFunc("/containers/{id}/start", m.handleStartContainer).Methods("POST")
	v1.HandleFunc("/containers/{id}/stop", m.handleStopContainer).Methods("POST")
	v1.HandleFunc("/containers/{id}/restart", m.handleRestartContainer).Methods("POST")
	v1.HandleFunc("/containers/{id}/pause", m.handlePauseContainer).Methods("POST")
	v1.HandleFunc("/containers/{id}/unpause", m.handleUnpauseContainer).Methods("POST")
	v1.HandleFunc("/containers/{id}/logs", m.handleGetContainerLogs).Methods("GET")
	v1.HandleFunc("/containers/{id}/exec", m.handleContainerExec).Methods("POST")
	
	// WebSocket endpoint
	router.HandleFunc("/ws", m.handleWebSocket)
	
	// Health check
	router.HandleFunc("/health", m.handleHealth).Methods("GET")
	
	return router
}

// initializeDefaultViews initializes default topology views
func (m *Manager) initializeDefaultViews() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Processes View
	m.Views["processes"] = &View{
		ID:          "processes",
		Name:        "Processes",
		Description: "All processes and their relationships",
		NodeTypes:   []string{"process", "container", "host"},
		EdgeTypes:   []string{"process", "container"},
		Layout: &Layout{
			Type:       "force",
			Iterations: 300,
			Strength:   0.8,
		},
		CreatedAt: time.Now(),
	}
	
	// Containers View
	m.Views["containers"] = &View{
		ID:          "containers",
		Name:        "Containers",
		Description: "Container-level topology",
		NodeTypes:   []string{"container", "host"},
		EdgeTypes:   []string{"network", "container"},
		Layout: &Layout{
			Type:       "force",
			Iterations: 200,
			Strength:   0.6,
		},
		CreatedAt: time.Now(),
	}
	
	// Hosts View
	m.Views["hosts"] = &View{
		ID:          "hosts",
		Name:        "Hosts",
		Description: "Infrastructure visualization",
		NodeTypes:   []string{"host"},
		EdgeTypes:   []string{"network"},
		Layout: &Layout{
			Type:       "grid",
			Iterations: 100,
		},
		CreatedAt: time.Now(),
	}
	
	// Pods View
	m.Views["pods"] = &View{
		ID:          "pods",
		Name:        "Pods",
		Description: "Kubernetes pod topology",
		NodeTypes:   []string{"pod", "container", "host"},
		EdgeTypes:   []string{"network", "container"},
		Layout: &Layout{
			Type:       "hierarchical",
			Iterations: 150,
		},
		CreatedAt: time.Now(),
	}
	
	// Services View
	m.Views["services"] = &View{
		ID:          "services",
		Name:        "Services",
		Description: "Service mesh visualization",
		NodeTypes:   []string{"service", "pod", "container"},
		EdgeTypes:   []string{"network", "service"},
		Layout: &Layout{
			Type:       "circular",
			Iterations: 100,
		},
		CreatedAt: time.Now(),
	}
}

// startMetricsCollection starts collecting metrics
func (m *Manager) startMetricsCollection() {
	m.updateTicker = time.NewTicker(15 * time.Second)
	defer m.updateTicker.Stop()
	
	for {
		select {
		case <-m.updateTicker.C:
			m.updateMetrics()
			m.broadcastUpdate(&TopologyUpdate{
				Type: "metrics",
				Data: m.getMetrics(),
			})
		}
	}
}

// startWebSocketCleanup starts cleaning up WebSocket connections
func (m *Manager) startWebSocketCleanup() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			m.cleanupWebSocketConnections()
		}
	}
}

// updateMetrics updates the overall metrics
func (m *Manager) updateMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	metrics := &Metrics{
		TotalNodes:    len(m.Nodes),
		TotalEdges:    len(m.Edges),
		LastUpdated:   time.Now(),
	}
	
	for _, node := range m.Nodes {
		switch node.Status {
		case "healthy":
			metrics.HealthyNodes++
		case "warning":
			metrics.WarningNodes++
		case "critical":
			metrics.CriticalNodes++
		default:
			metrics.UnknownNodes++
		}
	}
	
	m.Metrics["overall"] = metrics
}

// getMetrics returns the current metrics
func (m *Manager) getMetrics() *Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if metrics, exists := m.Metrics["overall"]; exists {
		return metrics
	}
	return &Metrics{}
}

// AddNode adds a node to the topology
func (m *Manager) AddNode(node *Node) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	node.CreatedAt = time.Now()
	node.UpdatedAt = time.Now()
	node.LastSeen = time.Now()
	
	m.Nodes[node.ID] = node
	
	// Broadcast update
	go m.broadcastUpdate(&TopologyUpdate{
		Type: "add",
		Node: node,
	})
}

// UpdateNode updates a node in the topology
func (m *Manager) UpdateNode(node *Node) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if existing, exists := m.Nodes[node.ID]; exists {
		node.CreatedAt = existing.CreatedAt
		node.UpdatedAt = time.Now()
		node.LastSeen = time.Now()
		
		m.Nodes[node.ID] = node
		
		// Broadcast update
		go m.broadcastUpdate(&TopologyUpdate{
			Type: "update",
			Node: node,
		})
	}
}

// RemoveNode removes a node from the topology
func (m *Manager) RemoveNode(nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if node, exists := m.Nodes[nodeID]; exists {
		delete(m.Nodes, nodeID)
		
		// Remove associated edges
		for edgeID, edge := range m.Edges {
			if edge.Source == nodeID || edge.Target == nodeID {
				delete(m.Edges, edgeID)
			}
		}
		
		// Broadcast update
		go m.broadcastUpdate(&TopologyUpdate{
			Type: "remove",
			Node: node,
		})
	}
}

// AddEdge adds an edge to the topology
func (m *Manager) AddEdge(edge *Edge) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	edge.CreatedAt = time.Now()
	edge.UpdatedAt = time.Now()
	
	m.Edges[edge.ID] = edge
	
	// Broadcast update
	go m.broadcastUpdate(&TopologyUpdate{
		Type: "add",
		Edge: edge,
	})
}

// UpdateEdge updates an edge in the topology
func (m *Manager) UpdateEdge(edge *Edge) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if existing, exists := m.Edges[edge.ID]; exists {
		edge.CreatedAt = existing.CreatedAt
		edge.UpdatedAt = time.Now()
		
		m.Edges[edge.ID] = edge
		
		// Broadcast update
		go m.broadcastUpdate(&TopologyUpdate{
			Type: "update",
			Edge: edge,
		})
	}
}

// RemoveEdge removes an edge from the topology
func (m *Manager) RemoveEdge(edgeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if edge, exists := m.Edges[edgeID]; exists {
		delete(m.Edges, edgeID)
		
		// Broadcast update
		go m.broadcastUpdate(&TopologyUpdate{
			Type: "remove",
			Edge: edge,
		})
	}
}

// broadcastUpdate broadcasts an update to all subscribers
func (m *Manager) broadcastUpdate(update *TopologyUpdate) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	data, err := json.Marshal(update)
	if err != nil {
		log.Printf("Error marshaling update: %v", err)
		return
	}
	
	for _, subscriber := range m.Subscribers {
		select {
		case subscriber.Send <- data:
		default:
			// Channel is full, skip this subscriber
		}
	}
}

// cleanupWebSocketConnections cleans up stale WebSocket connections
func (m *Manager) cleanupWebSocketConnections() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	now := time.Now()
	for id, subscriber := range m.Subscribers {
		if now.Sub(subscriber.LastPing) > 2*time.Minute {
			subscriber.Done <- true
			subscriber.Conn.Close()
			delete(m.Subscribers, id)
		}
	}
}

// HTTP handlers
func (m *Manager) handleGetTopology(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	topology := map[string]interface{}{
		"nodes": m.Nodes,
		"edges": m.Edges,
		"views": m.Views,
		"metrics": m.getMetrics(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(topology)
}

func (m *Manager) handleGetNodes(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	nodes := make([]*Node, 0, len(m.Nodes))
	for _, node := range m.Nodes {
		nodes = append(nodes, node)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}

func (m *Manager) handleGetNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	node, exists := m.Nodes[nodeID]
	if !exists {
		http.NotFound(w, r)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(node)
}

func (m *Manager) handleGetEdges(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	edges := make([]*Edge, 0, len(m.Edges))
	for _, edge := range m.Edges {
		edges = append(edges, edge)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(edges)
}

func (m *Manager) handleGetEdge(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	edgeID := vars["id"]
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	edge, exists := m.Edges[edgeID]
	if !exists {
		http.NotFound(w, r)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(edge)
}

func (m *Manager) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	results := make([]*Node, 0)
	for _, node := range m.Nodes {
		if m.matchesSearch(node, query) {
			results = append(results, node)
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (m *Manager) handleFilter(w http.ResponseWriter, r *http.Request) {
	var filter Filter
	if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	results := make([]*Node, 0)
	for _, node := range m.Nodes {
		if m.matchesFilter(node, &filter) {
			results = append(results, node)
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (m *Manager) handleGetViews(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	views := make([]*View, 0, len(m.Views))
	for _, view := range m.Views {
		views = append(views, view)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(views)
}

func (m *Manager) handleGetView(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	viewID := vars["id"]
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	view, exists := m.Views[viewID]
	if !exists {
		http.NotFound(w, r)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(view)
}

func (m *Manager) handleCreateView(w http.ResponseWriter, r *http.Request) {
	var view View
	if err := json.NewDecoder(r.Body).Decode(&view); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	view.CreatedAt = time.Now()
	m.Views[view.ID] = &view
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(view)
}

func (m *Manager) handleUpdateView(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	viewID := vars["id"]
	
	var view View
	if err := json.NewDecoder(r.Body).Decode(&view); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if existing, exists := m.Views[viewID]; exists {
		view.CreatedAt = existing.CreatedAt
		m.Views[viewID] = &view
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(view)
	} else {
		http.NotFound(w, r)
	}
}

func (m *Manager) handleDeleteView(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	viewID := vars["id"]
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.Views, viewID)
	w.WriteHeader(http.StatusOK)
}

func (m *Manager) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m.getMetrics())
}

func (m *Manager) handleGetNodeMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	node, exists := m.Nodes[nodeID]
	if !exists {
		http.NotFound(w, r)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(node.Metrics)
}

func (m *Manager) handleGetEdgeMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	edgeID := vars["id"]
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	edge, exists := m.Edges[edgeID]
	if !exists {
		http.NotFound(w, r)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(edge.Metrics)
}

// Container control handlers
func (m *Manager) handleStartContainer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]
	
	// In a real implementation, this would call Docker API
	log.Printf("Starting container %s", containerID)
	
	w.WriteHeader(http.StatusOK)
}

func (m *Manager) handleStopContainer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]
	
	// In a real implementation, this would call Docker API
	log.Printf("Stopping container %s", containerID)
	
	w.WriteHeader(http.StatusOK)
}

func (m *Manager) handleRestartContainer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]
	
	// In a real implementation, this would call Docker API
	log.Printf("Restarting container %s", containerID)
	
	w.WriteHeader(http.StatusOK)
}

func (m *Manager) handlePauseContainer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]
	
	// In a real implementation, this would call Docker API
	log.Printf("Pausing container %s", containerID)
	
	w.WriteHeader(http.StatusOK)
}

func (m *Manager) handleUnpauseContainer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]
	
	// In a real implementation, this would call Docker API
	log.Printf("Unpausing container %s", containerID)
	
	w.WriteHeader(http.StatusOK)
}

func (m *Manager) handleGetContainerLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]
	
	// In a real implementation, this would stream container logs
	log.Printf("Getting logs for container %s", containerID)
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Container logs would be streamed here..."))
}

func (m *Manager) handleContainerExec(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]
	
	// In a real implementation, this would establish a WebSocket for terminal
	log.Printf("Executing command in container %s", containerID)
	
	w.WriteHeader(http.StatusOK)
}

func (m *Manager) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()
	
	subscriber := &Subscriber{
		ID:       fmt.Sprintf("subscriber-%d", time.Now().UnixNano()),
		Conn:     conn,
		LastPing: time.Now(),
		Send:     make(chan []byte, 256),
		Done:     make(chan bool),
	}
	
	m.mu.Lock()
	m.Subscribers[subscriber.ID] = subscriber
	m.mu.Unlock()
	
	// Send initial topology
	topology := m.getTopologyForSubscriber(subscriber)
	if data, err := json.Marshal(topology); err == nil {
		subscriber.Send <- data
	}
	
	// Handle WebSocket messages
	go m.handleWebSocketMessages(subscriber)
	
	// Send updates
	for {
		select {
		case data := <-subscriber.Send:
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		case <-subscriber.Done:
			return
		}
	}
}

func (m *Manager) handleWebSocketMessages(subscriber *Subscriber) {
	defer func() {
		m.mu.Lock()
		delete(m.Subscribers, subscriber.ID)
		m.mu.Unlock()
		subscriber.Conn.Close()
	}()
	
	subscriber.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	subscriber.Conn.SetPongHandler(func(string) error {
		subscriber.LastPing = time.Now()
		subscriber.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	
	for {
		_, _, err := subscriber.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

func (m *Manager) getTopologyForSubscriber(subscriber *Subscriber) map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	topology := map[string]interface{}{
		"nodes": m.Nodes,
		"edges": m.Edges,
		"views": m.Views,
		"metrics": m.getMetrics(),
	}
	
	return topology
}

func (m *Manager) matchesSearch(node *Node, query string) bool {
	query = strings.ToLower(query)
	
	// Search in name
	if strings.Contains(strings.ToLower(node.Name), query) {
		return true
	}
	
	// Search in metadata
	for _, value := range node.Metadata {
		if str, ok := value.(string); ok {
			if strings.Contains(strings.ToLower(str), query) {
				return true
			}
		}
	}
	
	return false
}

func (m *Manager) matchesFilter(node *Node, filter *Filter) bool {
	// Filter by node types
	if len(filter.NodeTypes) > 0 {
		found := false
		for _, nodeType := range filter.NodeTypes {
			if node.Type == nodeType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Filter by status
	if len(filter.Status) > 0 {
		found := false
		for _, status := range filter.Status {
			if node.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Filter by labels
	if len(filter.Labels) > 0 {
		for key, value := range filter.Labels {
			if node.Metadata[key] != value {
				return false
			}
		}
	}
	
	// Filter by metrics
	if filter.Metrics != nil {
		if !m.matchesMetricsFilter(node, filter.Metrics) {
			return false
		}
	}
	
	return true
}

func (m *Manager) matchesMetricsFilter(node *Node, metricsFilter *MetricsFilter) bool {
	if node.Metrics == nil {
		return false
	}
	
	// CPU usage filter
	if metricsFilter.CPUUsage != nil {
		if node.Metrics.CPUUsage != nil {
			cpu := node.Metrics.CPUUsage.Current
			if cpu < metricsFilter.CPUUsage.Min || cpu > metricsFilter.CPUUsage.Max {
				return false
			}
		}
	}
	
	// Memory usage filter
	if metricsFilter.MemoryUsage != nil {
		if node.Metrics.MemoryUsage != nil {
			memory := node.Metrics.MemoryUsage.Current
			if memory < metricsFilter.MemoryUsage.Min || memory > metricsFilter.MemoryUsage.Max {
				return false
			}
		}
	}
	
	// Connections filter
	if metricsFilter.Connections != nil {
		if node.Metrics.Connections != nil {
			connections := node.Metrics.Connections.Current
			if connections < metricsFilter.Connections.Min || connections > metricsFilter.Connections.Max {
				return false
			}
		}
	}
	
	return true
}

// handleAddNode handles POST requests to add a new node
func (m *Manager) handleAddNode(w http.ResponseWriter, r *http.Request) {
	var node Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Validate required fields
	if node.ID == "" {
		http.Error(w, "node ID is required", http.StatusBadRequest)
		return
	}
	
	if node.Type == "" {
		http.Error(w, "node type is required", http.StatusBadRequest)
		return
	}
	
	// Add the node
	m.AddNode(&node)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "created",
		"id":     node.ID,
	})
}

// handleAddEdge handles POST requests to add a new edge
func (m *Manager) handleAddEdge(w http.ResponseWriter, r *http.Request) {
	var edge Edge
	if err := json.NewDecoder(r.Body).Decode(&edge); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Validate required fields
	if edge.ID == "" {
		http.Error(w, "edge ID is required", http.StatusBadRequest)
		return
	}
	
	if edge.Source == "" {
		http.Error(w, "edge source is required", http.StatusBadRequest)
		return
	}
	
	if edge.Target == "" {
		http.Error(w, "edge target is required", http.StatusBadRequest)
		return
	}
	
	// Add the edge
	m.AddEdge(&edge)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "created",
		"id":     edge.ID,
	})
}

func (m *Manager) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
