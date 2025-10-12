package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/ljluestc/orchestrator/pkg/probe"
)

// App is the central monitoring application backend
type App struct {
	ID             string
	Port           int
	router         *mux.Router
	server         *http.Server
	upgrader       websocket.Upgrader
	reports        map[string]*probe.ReportData
	reportsMux     sync.RWMutex
	subscribers    map[*websocket.Conn]bool
	subscribersMux sync.RWMutex
	topology       *Topology
}

// Topology represents the aggregated topology from all probes
type Topology struct {
	Timestamp   time.Time                    `json:"timestamp"`
	Hosts       map[string]*probe.HostInfo   `json:"hosts"`
	Containers  map[string]*probe.ContainerInfo `json:"containers"`
	Processes   map[string]*probe.ProcessInfo  `json:"processes"`
	Networks    []probe.NetworkConnection     `json:"networks"`
	ViewModes   []string                      `json:"view_modes"`
	TotalProbes int                           `json:"total_probes"`
}

// NewApp creates a new monitoring app instance
func NewApp(id string, port int) *App {
	app := &App{
		ID:          id,
		Port:        port,
		router:      mux.NewRouter(),
		reports:     make(map[string]*probe.ReportData),
		subscribers: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
		topology: &Topology{
			Hosts:      make(map[string]*probe.HostInfo),
			Containers: make(map[string]*probe.ContainerInfo),
			Processes:  make(map[string]*probe.ProcessInfo),
			Networks:   []probe.NetworkConnection{},
			ViewModes:  []string{"processes", "containers", "hosts", "pods", "services"},
		},
	}

	app.setupRoutes()

	app.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      app.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return app
}

// setupRoutes configures HTTP routes
func (a *App) setupRoutes() {
	// API routes
	api := a.router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/reports", a.handleReport).Methods("POST")
	api.HandleFunc("/topology", a.handleGetTopology).Methods("GET")
	api.HandleFunc("/topology/ws", a.handleWebSocket)
	api.HandleFunc("/containers", a.handleListContainers).Methods("GET")
	api.HandleFunc("/containers/{id}/stop", a.handleContainerStop).Methods("POST")
	api.HandleFunc("/containers/{id}/start", a.handleContainerStart).Methods("POST")
	api.HandleFunc("/containers/{id}/restart", a.handleContainerRestart).Methods("POST")
	api.HandleFunc("/containers/{id}/logs", a.handleContainerLogs).Methods("GET")
	api.HandleFunc("/containers/{id}/exec", a.handleContainerExec).Methods("POST")
	api.HandleFunc("/search", a.handleSearch).Methods("GET")

	// Health check
	a.router.HandleFunc("/health", a.handleHealth).Methods("GET")
	a.router.HandleFunc("/metrics", a.handleMetrics).Methods("GET")
}

// Start starts the monitoring app server
func (a *App) Start(ctx context.Context) error {
	log.Printf("Starting monitoring app %s on port %d", a.ID, a.Port)

	// Start topology aggregation goroutine
	go a.aggregateTopology(ctx)

	// Start broadcast goroutine
	go a.broadcastUpdates(ctx)

	// Start HTTP server
	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	<-ctx.Done()

	// Shutdown gracefully
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return a.server.Shutdown(shutdownCtx)
}

// handleReport receives reports from probes
func (a *App) handleReport(w http.ResponseWriter, r *http.Request) {
	var report probe.ReportData
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		http.Error(w, "Invalid report format", http.StatusBadRequest)
		return
	}

	a.reportsMux.Lock()
	a.reports[report.AgentID] = &report
	a.reportsMux.Unlock()

	log.Printf("Received report from probe %s with %d containers", report.AgentID, len(report.DockerInfo.Containers))

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

// handleGetTopology returns the current topology
func (a *App) handleGetTopology(w http.ResponseWriter, r *http.Request) {
	view := r.URL.Query().Get("view")
	if view == "" {
		view = "containers"
	}

	a.reportsMux.RLock()
	topology := a.buildTopologyView(view)
	a.reportsMux.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(topology)
}

// handleWebSocket handles WebSocket connections for real-time updates
func (a *App) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	a.subscribersMux.Lock()
	a.subscribers[conn] = true
	a.subscribersMux.Unlock()

	log.Printf("New WebSocket connection from %s", r.RemoteAddr)

	// Send initial topology
	a.reportsMux.RLock()
	topology := a.buildTopologyView("containers")
	a.reportsMux.RUnlock()

	if err := conn.WriteJSON(topology); err != nil {
		log.Printf("Error sending initial topology: %v", err)
	}

	// Keep connection alive
	go func() {
		defer func() {
			a.subscribersMux.Lock()
			delete(a.subscribers, conn)
			a.subscribersMux.Unlock()
			conn.Close()
		}()

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

// handleListContainers returns list of all containers
func (a *App) handleListContainers(w http.ResponseWriter, r *http.Request) {
	a.reportsMux.RLock()
	defer a.reportsMux.RUnlock()

	var containers []probe.ContainerInfo
	for _, report := range a.reports {
		containers = append(containers, report.DockerInfo.Containers...)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(containers)
}

// handleContainerStop stops a container
func (a *App) handleContainerStop(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]

	log.Printf("Stopping container %s", containerID)
	// TODO: Implement actual container stop via Docker API

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"action": "stop",
		"container_id": containerID,
	})
}

// handleContainerStart starts a container
func (a *App) handleContainerStart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]

	log.Printf("Starting container %s", containerID)
	// TODO: Implement actual container start via Docker API

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"action": "start",
		"container_id": containerID,
	})
}

// handleContainerRestart restarts a container
func (a *App) handleContainerRestart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]

	log.Printf("Restarting container %s", containerID)
	// TODO: Implement actual container restart via Docker API

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"action": "restart",
		"container_id": containerID,
	})
}

// handleContainerLogs streams container logs
func (a *App) handleContainerLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]

	log.Printf("Streaming logs for container %s", containerID)
	// TODO: Implement actual log streaming via Docker API

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "Logs for container %s\n", containerID)
}

// handleContainerExec executes a command in a container
func (a *App) handleContainerExec(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]

	var req struct {
		Command []string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	log.Printf("Executing command in container %s: %v", containerID, req.Command)
	// TODO: Implement actual exec via Docker API

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"container_id": containerID,
		"command": req.Command,
		"output": "Command executed successfully",
	})
}

// handleSearch performs full-text search across topology
func (a *App) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual search functionality
	log.Printf("Search query: %s", query)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query": query,
		"results": []interface{}{},
	})
}

// handleHealth returns health status
func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	a.reportsMux.RLock()
	probeCount := len(a.reports)
	a.reportsMux.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"app_id": a.ID,
		"probes_connected": probeCount,
		"timestamp": time.Now(),
	})
}

// handleMetrics returns Prometheus metrics
func (a *App) handleMetrics(w http.ResponseWriter, r *http.Request) {
	a.reportsMux.RLock()
	probeCount := len(a.reports)
	containerCount := 0
	for _, report := range a.reports {
		containerCount += len(report.DockerInfo.Containers)
	}
	a.reportsMux.RUnlock()

	fmt.Fprintf(w, "# HELP monitoring_app_probes_connected Number of connected probes\n")
	fmt.Fprintf(w, "# TYPE monitoring_app_probes_connected gauge\n")
	fmt.Fprintf(w, "monitoring_app_probes_connected %d\n", probeCount)

	fmt.Fprintf(w, "# HELP monitoring_app_containers_total Total number of containers\n")
	fmt.Fprintf(w, "# TYPE monitoring_app_containers_total gauge\n")
	fmt.Fprintf(w, "monitoring_app_containers_total %d\n", containerCount)
}

// aggregateTopology periodically aggregates reports into topology
func (a *App) aggregateTopology(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.updateTopology()
		}
	}
}

// updateTopology updates the aggregated topology
func (a *App) updateTopology() {
	a.reportsMux.RLock()
	defer a.reportsMux.RUnlock()

	topology := &Topology{
		Timestamp:  time.Now(),
		Hosts:      make(map[string]*probe.HostInfo),
		Containers: make(map[string]*probe.ContainerInfo),
		Processes:  make(map[string]*probe.ProcessInfo),
		Networks:   []probe.NetworkConnection{},
		ViewModes:  []string{"processes", "containers", "hosts", "pods", "services"},
		TotalProbes: len(a.reports),
	}

	for _, report := range a.reports {
		if report.HostInfo != nil {
			topology.Hosts[report.Hostname] = report.HostInfo
		}
		for i := range report.DockerInfo.Containers {
			topology.Containers[report.DockerInfo.Containers[i].ID] = &report.DockerInfo.Containers[i]
		}
		for i := range report.ProcessesInfo.Processes {
			topology.Processes[fmt.Sprintf("%d", report.ProcessesInfo.Processes[i].PID)] = &report.ProcessesInfo.Processes[i]
		}
		topology.Networks = append(topology.Networks, report.NetworkInfo.Connections...)
	}

	a.topology = topology
}

// broadcastUpdates broadcasts topology updates to WebSocket subscribers
func (a *App) broadcastUpdates(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.broadcastTopology()
		}
	}
}

// broadcastTopology sends topology to all WebSocket subscribers
func (a *App) broadcastTopology() {
	a.reportsMux.RLock()
	topology := a.topology
	a.reportsMux.RUnlock()

	a.subscribersMux.RLock()
	defer a.subscribersMux.RUnlock()

	for conn := range a.subscribers {
		if err := conn.WriteJSON(topology); err != nil {
			log.Printf("Error broadcasting to subscriber: %v", err)
		}
	}
}

// buildTopologyView builds a view-specific topology representation
func (a *App) buildTopologyView(view string) *Topology {
	// Return the aggregated topology
	// TODO: Filter based on view mode
	return a.topology
}
