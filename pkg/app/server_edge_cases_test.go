package app

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ljluestc/orchestrator/pkg/probe"
	"github.com/stretchr/testify/assert"
)

// TestServerCleanup tests the cleanup function
func TestServerCleanup(t *testing.T) {
	config := ServerConfig{
		Host:               "localhost",
		Port:               8081,
		MaxDataAge:         1 * time.Hour,
		CleanupInterval:    1 * time.Minute,
		StaleNodeThreshold: 2 * time.Minute,
	}

	server := NewServer(config)

	// Add some agents
	now := time.Now()
	server.mu.Lock()
	server.agents["fresh-agent"] = &AgentInfo{
		AgentID:      "fresh-agent",
		Hostname:     "host1",
		LastSeen:     now,
		RegisteredAt: now,
	}
	server.agents["stale-agent"] = &AgentInfo{
		AgentID:      "stale-agent",
		Hostname:     "host2",
		LastSeen:     now.Add(-5 * time.Minute), // 5 minutes ago, beyond threshold
		RegisteredAt: now.Add(-10 * time.Minute),
	}
	server.mu.Unlock()

	// Run cleanup
	server.cleanup()

	// Verify stale agent was removed
	server.mu.RLock()
	_, hasFresh := server.agents["fresh-agent"]
	_, hasStale := server.agents["stale-agent"]
	server.mu.RUnlock()

	assert.True(t, hasFresh, "Fresh agent should remain")
	assert.False(t, hasStale, "Stale agent should be removed")
}

// TestServerCleanup_EmptyAgents tests cleanup with no agents
func TestServerCleanup_EmptyAgents(t *testing.T) {
	config := ServerConfig{
		StaleNodeThreshold: 2 * time.Minute,
	}

	server := NewServer(config)

	// Run cleanup with no agents - should not panic
	server.cleanup()

	server.mu.RLock()
	agentCount := len(server.agents)
	server.mu.RUnlock()

	assert.Equal(t, 0, agentCount)
}

// TestServerCleanup_AllStaleAgents tests cleanup when all agents are stale
func TestServerCleanup_AllStaleAgents(t *testing.T) {
	config := ServerConfig{
		StaleNodeThreshold: 1 * time.Minute,
	}

	server := NewServer(config)

	// Add multiple stale agents
	now := time.Now()
	server.mu.Lock()
	for i := 0; i < 5; i++ {
		agentID := "stale-" + string(rune('a'+i))
		server.agents[agentID] = &AgentInfo{
			AgentID:      agentID,
			Hostname:     "host" + string(rune('1'+i)),
			LastSeen:     now.Add(-5 * time.Minute),
			RegisteredAt: now.Add(-10 * time.Minute),
		}
	}
	server.mu.Unlock()

	// Run cleanup
	server.cleanup()

	// All agents should be removed
	server.mu.RLock()
	agentCount := len(server.agents)
	server.mu.RUnlock()

	assert.Equal(t, 0, agentCount, "All stale agents should be removed")
}

// TestServerCleanupLoop tests the cleanup loop
func TestServerCleanupLoop(t *testing.T) {
	config := ServerConfig{
		Host:               "localhost",
		Port:               8082,
		CleanupInterval:    100 * time.Millisecond, // Very short interval for testing
		StaleNodeThreshold: 50 * time.Millisecond,
	}

	server := NewServer(config)
	ctx := context.Background()

	// Start server (which starts cleanup loop)
	err := server.Start(ctx)
	assert.NoError(t, err)

	// Add a stale agent
	now := time.Now()
	server.mu.Lock()
	server.agents["will-be-stale"] = &AgentInfo{
		AgentID:      "will-be-stale",
		Hostname:     "host1",
		LastSeen:     now.Add(-200 * time.Millisecond),
		RegisteredAt: now.Add(-500 * time.Millisecond),
	}
	server.mu.Unlock()

	// Wait for cleanup to run
	time.Sleep(250 * time.Millisecond)

	// Verify cleanup ran and removed stale agent
	server.mu.RLock()
	_, exists := server.agents["will-be-stale"]
	server.mu.RUnlock()

	assert.False(t, exists, "Cleanup loop should have removed stale agent")

	// Stop server
	err = server.Stop()
	assert.NoError(t, err)
}

// TestCORSMiddleware tests CORS middleware
func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware())

	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		checkHeaders   bool
	}{
		{
			name:           "OPTIONS request",
			method:         "OPTIONS",
			path:           "/test",
			expectedStatus: 204,
			checkHeaders:   true,
		},
		{
			name:           "GET request",
			method:         "GET",
			path:           "/test",
			expectedStatus: 200,
			checkHeaders:   true,
		},
		{
			name:           "POST request",
			method:         "POST",
			path:           "/test",
			expectedStatus: 404, // No POST handler defined
			checkHeaders:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkHeaders {
				assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
				assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
				assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
			}
		})
	}
}

// TestCORSMiddleware_AllMethods tests CORS with different HTTP methods
func TestCORSMiddleware_AllMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware())

	router.GET("/api", func(c *gin.Context) {
		c.JSON(200, gin.H{"method": "GET"})
	})
	router.POST("/api", func(c *gin.Context) {
		c.JSON(200, gin.H{"method": "POST"})
	})
	router.PUT("/api", func(c *gin.Context) {
		c.JSON(200, gin.H{"method": "PUT"})
	})
	router.DELETE("/api", func(c *gin.Context) {
		c.JSON(200, gin.H{"method": "DELETE"})
	})

	methods := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// All requests should have CORS headers
			assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Origin"))
			assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Methods"))

			if method == "OPTIONS" {
				assert.Equal(t, 204, w.Code)
			}
		})
	}
}

// TestCORSMiddleware_CustomHeaders tests CORS with custom headers
func TestCORSMiddleware_CustomHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware())

	router.POST("/api/submit", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("POST", "/api/submit", nil)
	req.Header.Set("X-Agent-ID", "test-agent")
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify CORS headers allow custom headers
	allowedHeaders := w.Header().Get("Access-Control-Allow-Headers")
	assert.Contains(t, allowedHeaders, "X-Agent-ID")
	assert.Contains(t, allowedHeaders, "Authorization")
	assert.Contains(t, allowedHeaders, "Content-Type")
}

// TestServerCleanup_ConcurrentAccess tests cleanup with concurrent access
func TestServerCleanup_ConcurrentAccess(t *testing.T) {
	config := ServerConfig{
		StaleNodeThreshold: 1 * time.Minute,
	}

	server := NewServer(config)

	// Add agents concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			now := time.Now()
			server.mu.Lock()
			agentID := "agent-" + string(rune('a'+id))
			server.agents[agentID] = &AgentInfo{
				AgentID:      agentID,
				Hostname:     "host",
				LastSeen:     now,
				RegisteredAt: now,
			}
			server.mu.Unlock()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Run cleanup concurrently with reads
	go server.cleanup()

	// Try to read agents while cleanup is running
	for i := 0; i < 5; i++ {
		go func() {
			server.mu.RLock()
			_ = len(server.agents)
			server.mu.RUnlock()
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	// Should not panic or deadlock
}

// TestServerStart_AlreadyRunning tests starting an already running server
func TestServerStart_AlreadyRunning(t *testing.T) {
	config := ServerConfig{
		Host: "localhost",
		Port: 8083,
	}

	server := NewServer(config)
	ctx := context.Background()

	// Start server
	err := server.Start(ctx)
	assert.NoError(t, err)

	// Try to start again
	err = server.Start(ctx)
	assert.Error(t, err, "Should error when starting already running server")
	assert.Contains(t, err.Error(), "already running")

	// Cleanup
	server.Stop()
}

// TestServerStop_NotRunning tests stopping a non-running server
func TestServerStop_NotRunning(t *testing.T) {
	config := ServerConfig{
		Host: "localhost",
		Port: 8084,
	}

	server := NewServer(config)

	// Try to stop server that was never started
	err := server.Stop()
	assert.Error(t, err, "Should error when stopping non-running server")
	assert.Contains(t, err.Error(), "not running")
}

// TestServerGetStats tests server statistics
func TestServerGetStats(t *testing.T) {
	config := ServerConfig{}
	server := NewServer(config)

	// Add some test data
	now := time.Now()
	server.mu.Lock()
	server.agents["test-agent"] = &AgentInfo{
		AgentID:      "test-agent",
		Hostname:     "test-host",
		LastSeen:     now,
		RegisteredAt: now,
	}
	server.mu.Unlock()

	stats := server.GetStats()

	assert.NotNil(t, stats)
	assert.Contains(t, stats, "uptime")
	assert.Contains(t, stats, "agents")
	assert.Contains(t, stats, "websocket_clients")
	assert.Contains(t, stats, "storage")
	assert.Contains(t, stats, "topology")

	agentCount := stats["agents"].(int)
	assert.Equal(t, 1, agentCount)
}

// TestServerAgentInfo tests agent information storage
func TestServerAgentInfo(t *testing.T) {
	config := ServerConfig{}
	server := NewServer(config)

	now := time.Now()
	agentInfo := &AgentInfo{
		AgentID:  "test-123",
		Hostname: "test-host",
		Metadata: map[string]string{
			"region": "us-west",
			"zone":   "a",
		},
		RegisteredAt: now,
		LastSeen:     now,
		LastReport: &probe.ReportData{
			AgentID:   "test-123",
			Timestamp: now,
		},
	}

	server.mu.Lock()
	server.agents[agentInfo.AgentID] = agentInfo
	server.mu.Unlock()

	server.mu.RLock()
	stored, exists := server.agents["test-123"]
	server.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, "test-123", stored.AgentID)
	assert.Equal(t, "test-host", stored.Hostname)
	assert.Equal(t, "us-west", stored.Metadata["region"])
	assert.NotNil(t, stored.LastReport)
}
