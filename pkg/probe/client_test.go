package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClient_SendReportWithInvalidURL(t *testing.T) {
	client := NewClient(ClientConfig{
		ServerURL: "http://invalid-url",
		AgentID:   "test-agent",
	})

	ctx := context.Background()
	report := &ReportData{
		Timestamp: time.Now(),
		AgentID:   "test-agent",
		Hostname:  "test-host",
	}

	err := client.SendReport(ctx, report)
	assert.Error(t, err)
}

func TestClient_SendReportWithTimeout(t *testing.T) {
	// Create a server that takes too long to respond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Longer than client timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL:      server.URL,
		AgentID:        "test-agent",
		RequestTimeout: 100 * time.Millisecond, // Very short timeout
	})

	ctx := context.Background()
	report := &ReportData{
		Timestamp: time.Now(),
		AgentID:   "test-agent",
		Hostname:  "test-host",
	}

	err := client.SendReport(ctx, report)
	assert.Error(t, err)
}

func TestClient_SendReportWithRetryTimeout(t *testing.T) {
	// Create a server that always returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL: server.URL,
		AgentID:   "test-agent",
	})

	ctx := context.Background()
	report := &ReportData{
		Timestamp: time.Now(),
		AgentID:   "test-agent",
		Hostname:  "test-host",
	}

	err := client.SendReportWithRetry(ctx, report, 2, 10*time.Millisecond)
	assert.Error(t, err)
}

func TestClient_PingWithInvalidURL(t *testing.T) {
	client := NewClient(ClientConfig{
		ServerURL: "http://invalid-url",
		AgentID:   "test-agent",
	})

	ctx := context.Background()
	err := client.Ping(ctx)
	assert.Error(t, err)
}

func TestClient_PingWithTimeout(t *testing.T) {
	// Create a server that takes too long to respond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL:      server.URL,
		AgentID:        "test-agent",
		RequestTimeout: 100 * time.Millisecond,
	})

	ctx := context.Background()
	err := client.Ping(ctx)
	assert.Error(t, err)
}

func TestClient_GetConfigWithInvalidURL(t *testing.T) {
	client := NewClient(ClientConfig{
		ServerURL: "http://invalid-url",
		AgentID:   "test-agent",
	})

	ctx := context.Background()
	config, err := client.GetConfig(ctx)
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestClient_GetConfigWithTimeout(t *testing.T) {
	// Create a server that takes too long to respond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"collection_interval": 30}`))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL:      server.URL,
		AgentID:        "test-agent",
		RequestTimeout: 100 * time.Millisecond,
	})

	ctx := context.Background()
	config, err := client.GetConfig(ctx)
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestClient_RegisterAgentWithInvalidURL(t *testing.T) {
	client := NewClient(ClientConfig{
		ServerURL: "http://invalid-url",
		AgentID:   "test-agent",
	})

	ctx := context.Background()
	metadata := map[string]string{
		"version": "1.0.0",
	}

	err := client.RegisterAgent(ctx, "test-host", metadata)
	assert.Error(t, err)
}

func TestClient_RegisterAgentWithTimeout(t *testing.T) {
	// Create a server that takes too long to respond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL:      server.URL,
		AgentID:        "test-agent",
		RequestTimeout: 100 * time.Millisecond,
	})

	ctx := context.Background()
	metadata := map[string]string{
		"version": "1.0.0",
	}

	err := client.RegisterAgent(ctx, "test-host", metadata)
	assert.Error(t, err)
}

func TestClient_HeartbeatWithInvalidURL(t *testing.T) {
	client := NewClient(ClientConfig{
		ServerURL: "http://invalid-url",
		AgentID:   "test-agent",
	})

	ctx := context.Background()
	err := client.Heartbeat(ctx)
	assert.Error(t, err)
}

func TestClient_HeartbeatWithTimeout(t *testing.T) {
	// Create a server that takes too long to respond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL:      server.URL,
		AgentID:        "test-agent",
		RequestTimeout: 100 * time.Millisecond,
	})

	ctx := context.Background()
	err := client.Heartbeat(ctx)
	assert.Error(t, err)
}

func TestClient_SendReportWithInvalidJSON(t *testing.T) {
	// Create a server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL: server.URL,
		AgentID:   "test-agent",
	})

	ctx := context.Background()
	report := &ReportData{
		Timestamp: time.Now(),
		AgentID:   "test-agent",
		Hostname:  "test-host",
	}

	err := client.SendReport(ctx, report)
	// This should succeed since we're not parsing the response
	assert.NoError(t, err)
}

func TestClient_GetConfigWithInvalidJSON(t *testing.T) {
	// Create a server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL: server.URL,
		AgentID:   "test-agent",
	})

	ctx := context.Background()
	config, err := client.GetConfig(ctx)
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestClient_RegisterAgentWithInvalidJSON(t *testing.T) {
	// Create a server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL: server.URL,
		AgentID:   "test-agent",
	})

	ctx := context.Background()
	metadata := map[string]string{
		"version": "1.0.0",
	}

	err := client.RegisterAgent(ctx, "test-host", metadata)
	// This should succeed since we're not parsing the response
	assert.NoError(t, err)
}

func TestClient_HeartbeatWithInvalidJSON(t *testing.T) {
	// Create a server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL: server.URL,
		AgentID:   "test-agent",
	})

	ctx := context.Background()
	err := client.Heartbeat(ctx)
	// This should succeed since we're not parsing the response
	assert.NoError(t, err)
}

func TestClient_ContextCancellation(t *testing.T) {
	// Create a server that takes a long time to respond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL: server.URL,
		AgentID:   "test-agent",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	report := &ReportData{
		Timestamp: time.Now(),
		AgentID:   "test-agent",
		Hostname:  "test-host",
	}

	err := client.SendReport(ctx, report)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestClient_EmptyAgentID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL: server.URL,
		AgentID:   "", // Empty agent ID
	})

	ctx := context.Background()
	report := &ReportData{
		Timestamp: time.Now(),
		AgentID:   "",
		Hostname:  "test-host",
	}

	err := client.SendReport(ctx, report)
	assert.NoError(t, err) // Should still work with empty agent ID
}

func TestClient_EmptyHostname(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL: server.URL,
		AgentID:   "test-agent",
	})

	ctx := context.Background()
	report := &ReportData{
		Timestamp: time.Now(),
		AgentID:   "test-agent",
		Hostname:  "", // Empty hostname
	}

	err := client.SendReport(ctx, report)
	assert.NoError(t, err) // Should still work with empty hostname
}

func TestClient_DefaultTimeout(t *testing.T) {
	client := NewClient(ClientConfig{
		ServerURL: "http://invalid-url",
		AgentID:   "test-agent",
		// No RequestTimeout specified, should use default
	})

	// Verify default timeout is set
	assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
}

func TestClient_CustomTimeout(t *testing.T) {
	client := NewClient(ClientConfig{
		ServerURL:      "http://invalid-url",
		AgentID:        "test-agent",
		RequestTimeout: 5 * time.Second,
	})

	// Verify custom timeout is set
	assert.Equal(t, 5*time.Second, client.httpClient.Timeout)
}
