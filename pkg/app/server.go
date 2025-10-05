package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ljluestc/orchestrator/internal/storage"
	"github.com/ljluestc/orchestrator/pkg/probe"
)

// ServerConfig contains configuration for the app server
type ServerConfig struct {
	Host               string
	Port               int
	MaxDataAge         time.Duration
	CleanupInterval    time.Duration
	StaleNodeThreshold time.Duration
}

// AgentInfo stores information about registered agents
type AgentInfo struct {
	AgentID      string            `json:"agent_id"`
	Hostname     string            `json:"hostname"`
	Metadata     map[string]string `json:"metadata"`
	RegisteredAt time.Time         `json:"registered_at"`
	LastSeen     time.Time         `json:"last_seen"`
	LastReport   *probe.ReportData `json:"last_report,omitempty"`
}

// Server represents the app backend server
type Server struct {
	config     ServerConfig
	router     *gin.Engine
	httpServer *http.Server
	storage    *storage.TimeSeriesStore
	aggregator *Aggregator
	wsHub      *WSHub
	agents     map[string]*AgentInfo
	mu         sync.RWMutex
	running    bool
	startTime  time.Time
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewServer creates a new app backend server
func NewServer(config ServerConfig) *Server {
	// Set defaults
	if config.Host == "" {
		config.Host = "0.0.0.0"
	}
	if config.Port == 0 {
		config.Port = 8080
	}
	if config.MaxDataAge == 0 {
		config.MaxDataAge = 1 * time.Hour
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 5 * time.Minute
	}
	if config.StaleNodeThreshold == 0 {
		config.StaleNodeThreshold = 5 * time.Minute
	}

	// Set gin mode to release for production
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	server := &Server{
		config:     config,
		router:     router,
		storage:    storage.NewTimeSeriesStore(config.MaxDataAge),
		aggregator: NewAggregator(),
		wsHub:      NewWSHub(),
		agents:     make(map[string]*AgentInfo),
		stopCh:     make(chan struct{}),
		startTime:  time.Now(),
	}

	// Setup middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(CORSMiddleware())

	// Setup routes
	handlers := NewHandlers(server)
	handlers.SetupRoutes(router)

	return server
}

// Start starts the app server
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server is already running")
	}
	s.running = true
	s.mu.Unlock()

	log.Printf("Starting app server on %s:%d", s.config.Host, s.config.Port)

	// Start WebSocket hub
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.wsHub.Run()
	}()

	// Start cleanup loop
	s.wg.Add(1)
	go s.cleanupLoop()

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.config.Host, s.config.Port),
		Handler: s.router,
	}

	// Start HTTP server in goroutine
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	log.Printf("App server started successfully")
	log.Printf("REST API: http://%s:%d/api/v1", s.config.Host, s.config.Port)
	log.Printf("WebSocket: ws://%s:%d/api/v1/ws", s.config.Host, s.config.Port)
	log.Printf("Health check: http://%s:%d/health", s.config.Host, s.config.Port)

	return nil
}

// Stop stops the app server gracefully
func (s *Server) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return fmt.Errorf("server is not running")
	}
	s.running = false
	s.mu.Unlock()

	log.Printf("Stopping app server...")

	// Stop background tasks first
	close(s.stopCh)

	// Stop WebSocket hub
	s.wsHub.Stop()

	// Stop HTTP server
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}

	// Stop storage
	s.storage.Stop()

	// Wait for goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines finished
	case <-time.After(5 * time.Second):
		log.Printf("Warning: Some goroutines did not finish in time")
	}

	log.Printf("App server stopped successfully")

	return nil
}

// cleanupLoop periodically cleans up stale data
func (s *Server) cleanupLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanup()
		case <-s.stopCh:
			return
		}
	}
}

// cleanup removes stale nodes and agents
func (s *Server) cleanup() {
	log.Printf("Running cleanup...")

	// Clean stale nodes from topology
	s.aggregator.CleanStaleNodes(s.config.StaleNodeThreshold)

	// Clean stale agents
	s.mu.Lock()
	staleAgents := make([]string, 0)
	cutoff := time.Now().Add(-s.config.StaleNodeThreshold)

	for agentID, agent := range s.agents {
		if agent.LastSeen.Before(cutoff) {
			staleAgents = append(staleAgents, agentID)
		}
	}

	for _, agentID := range staleAgents {
		delete(s.agents, agentID)
		log.Printf("Removed stale agent: %s", agentID)
	}
	s.mu.Unlock()

	log.Printf("Cleanup completed. Removed %d stale agents", len(staleAgents))
}

// IsRunning returns whether the server is running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetConfig returns the server configuration
func (s *Server) GetConfig() ServerConfig {
	return s.config
}

// GetStats returns server statistics
func (s *Server) GetStats() map[string]interface{} {
	s.mu.RLock()
	agentCount := len(s.agents)
	s.mu.RUnlock()

	return map[string]interface{}{
		"uptime":            time.Since(s.startTime).String(),
		"agents":            agentCount,
		"websocket_clients": s.wsHub.GetClientCount(),
		"storage":           s.storage.GetStats(),
		"topology":          s.aggregator.GetStats(),
	}
}

// CORSMiddleware handles CORS headers
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Agent-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
