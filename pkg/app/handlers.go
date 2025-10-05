package app

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ljluestc/orchestrator/pkg/probe"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		// In production, you should validate the origin
		return true
	},
}

// Handlers contains all HTTP handlers
type Handlers struct {
	server *Server
}

// NewHandlers creates a new handlers instance
func NewHandlers(server *Server) *Handlers {
	return &Handlers{
		server: server,
	}
}

// SetupRoutes sets up all routes for the server
func (h *Handlers) SetupRoutes(router *gin.Engine) {
	// Health check
	router.GET("/health", h.HealthCheck)
	router.GET("/api/v1/ping", h.Ping)

	// Agent management
	agentGroup := router.Group("/api/v1/agents")
	{
		agentGroup.POST("/register", h.RegisterAgent)
		agentGroup.POST("/heartbeat/:agent_id", h.Heartbeat)
		agentGroup.GET("/config/:agent_id", h.GetAgentConfig)
		agentGroup.GET("/list", h.ListAgents)
	}

	// Report submission
	router.POST("/api/v1/reports", h.SubmitReport)

	// Data queries
	queryGroup := router.Group("/api/v1/query")
	{
		queryGroup.GET("/topology", h.GetTopology)
		queryGroup.GET("/agents/:agent_id/latest", h.GetLatestReport)
		queryGroup.GET("/agents/:agent_id/timeseries", h.GetTimeSeries)
		queryGroup.GET("/stats", h.GetStats)
	}

	// WebSocket endpoint
	router.GET("/api/v1/ws", h.WebSocketHandler)
}

// HealthCheck returns the health status of the server
func (h *Handlers) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
	})
}

// Ping handles ping requests from agents
func (h *Handlers) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"pong":      true,
		"timestamp": time.Now(),
	})
}

// RegisterAgent handles agent registration
func (h *Handlers) RegisterAgent(c *gin.Context) {
	var req struct {
		AgentID   string            `json:"agent_id" binding:"required"`
		Hostname  string            `json:"hostname" binding:"required"`
		Metadata  map[string]string `json:"metadata"`
		Timestamp time.Time         `json:"timestamp"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Register agent
	h.server.mu.Lock()
	h.server.agents[req.AgentID] = &AgentInfo{
		AgentID:      req.AgentID,
		Hostname:     req.Hostname,
		Metadata:     req.Metadata,
		RegisteredAt: time.Now(),
		LastSeen:     time.Now(),
	}
	h.server.mu.Unlock()

	log.Printf("Agent registered: %s (hostname: %s)", req.AgentID, req.Hostname)

	c.JSON(http.StatusCreated, gin.H{
		"status":  "registered",
		"message": fmt.Sprintf("Agent %s registered successfully", req.AgentID),
	})
}

// Heartbeat handles heartbeat requests from agents
func (h *Handlers) Heartbeat(c *gin.Context) {
	agentID := c.Param("agent_id")

	h.server.mu.Lock()
	if agent, exists := h.server.agents[agentID]; exists {
		agent.LastSeen = time.Now()
	}
	h.server.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now(),
	})
}

// GetAgentConfig returns configuration for an agent
func (h *Handlers) GetAgentConfig(c *gin.Context) {
	agentID := c.Param("agent_id")

	// Return default configuration
	// In a real implementation, this could be per-agent configuration
	config := map[string]interface{}{
		"collection_interval": "30s",
		"heartbeat_interval":  "60s",
		"enabled_collectors":  []string{"host", "docker", "process", "network"},
	}

	log.Printf("Config requested for agent: %s", agentID)

	c.JSON(http.StatusOK, config)
}

// ListAgents returns a list of all registered agents
func (h *Handlers) ListAgents(c *gin.Context) {
	h.server.mu.RLock()
	agents := make([]*AgentInfo, 0, len(h.server.agents))
	for _, agent := range h.server.agents {
		agents = append(agents, agent)
	}
	h.server.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"agents": agents,
		"count":  len(agents),
	})
}

// SubmitReport handles report submissions from probes
func (h *Handlers) SubmitReport(c *gin.Context) {
	var report probe.ReportData

	if err := c.ShouldBindJSON(&report); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate report
	if report.AgentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id is required"})
		return
	}

	// Update last seen time
	h.server.mu.Lock()
	if agent, exists := h.server.agents[report.AgentID]; exists {
		agent.LastSeen = time.Now()
		agent.LastReport = &report
	}
	h.server.mu.Unlock()

	// Store in time-series database
	h.server.storage.AddReport(&report)

	// Process report for aggregation
	h.server.aggregator.ProcessReport(&report)

	// Broadcast update to WebSocket clients
	h.server.wsHub.BroadcastReportUpdate(report.AgentID, &report)

	log.Printf("Report received from agent: %s (hostname: %s)", report.AgentID, report.Hostname)

	c.JSON(http.StatusAccepted, gin.H{
		"status":  "accepted",
		"message": "Report processed successfully",
	})
}

// GetTopology returns the current topology view
func (h *Handlers) GetTopology(c *gin.Context) {
	topology := h.server.aggregator.GetTopology()

	c.JSON(http.StatusOK, gin.H{
		"topology":  topology,
		"timestamp": topology.Timestamp,
	})
}

// GetLatestReport returns the latest report for an agent
func (h *Handlers) GetLatestReport(c *gin.Context) {
	agentID := c.Param("agent_id")

	report := h.server.storage.GetLatestReport(agentID)
	if report == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No reports found for this agent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"agent_id": agentID,
		"report":   report,
	})
}

// GetTimeSeries returns time-series data for an agent
func (h *Handlers) GetTimeSeries(c *gin.Context) {
	agentID := c.Param("agent_id")

	// Parse duration parameter (default: 1 hour)
	durationStr := c.DefaultQuery("duration", "1h")
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid duration format"})
		return
	}

	points := h.server.storage.GetRecentPoints(agentID, duration)

	c.JSON(http.StatusOK, gin.H{
		"agent_id": agentID,
		"duration": durationStr,
		"points":   points,
		"count":    len(points),
	})
}

// GetStats returns statistics about the server
func (h *Handlers) GetStats(c *gin.Context) {
	h.server.mu.RLock()
	agentCount := len(h.server.agents)
	h.server.mu.RUnlock()

	stats := gin.H{
		"agents":           agentCount,
		"websocket_clients": h.server.wsHub.GetClientCount(),
		"storage":          h.server.storage.GetStats(),
		"topology":         h.server.aggregator.GetStats(),
		"uptime":           time.Since(h.server.startTime).String(),
	}

	c.JSON(http.StatusOK, stats)
}

// WebSocketHandler handles WebSocket connections
func (h *Handlers) WebSocketHandler(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket connection: %v", err)
		return
	}

	// Create new client
	clientID := fmt.Sprintf("client-%d", time.Now().UnixNano())
	client := NewWSClient(clientID, conn, h.server.wsHub)

	// Register client
	h.server.wsHub.register <- client

	// Start client pumps
	client.Start()

	// Send initial topology
	topology := h.server.aggregator.GetTopology()
	client.sendMessage("topology_update", topology)
}
