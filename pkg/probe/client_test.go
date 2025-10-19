package probe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientPing(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "successful ping",
			statusCode: http.StatusOK,
			response:   `{"status":"ok"}`,
			wantErr:    false,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			response:   `{"error":"server error"}`,
			wantErr:    true,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			response:   `{"error":"not found"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/ping", r.URL.Path)
				assert.Equal(t, http.MethodGet, r.Method)
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient(ClientConfig{
				ServerURL: server.URL,
				AgentID:   "test-agent",
			})
			err := client.Ping(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClientGetConfig(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "successful config retrieval",
			statusCode: http.StatusOK,
			response: `{
				"server_url": "http://localhost:8080",
				"collection_interval": 30000000000,
				"heartbeat_interval": 60000000000
			}`,
			wantErr: false,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			response:   `{"error":"internal error"}`,
			wantErr:    true,
		},
		{
			name:       "invalid json response",
			statusCode: http.StatusOK,
			response:   `invalid json`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "/api/v1/agents/config/")
				assert.Equal(t, http.MethodGet, r.Method)
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient(ClientConfig{
				ServerURL: server.URL,
				AgentID:   "test-agent",
			})
			config, err := client.GetConfig(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
			}
		})
	}
}

func TestClientRegisterAgent(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "successful registration",
			statusCode: http.StatusOK,
			response:   `{"agent_id":"test-agent-123","status":"registered"}`,
			wantErr:    false,
		},
		{
			name:       "registration created",
			statusCode: http.StatusCreated,
			response:   `{"agent_id":"test-agent-123","status":"created"}`,
			wantErr:    false,
		},
		{
			name:       "registration conflict",
			statusCode: http.StatusConflict,
			response:   `{"error":"agent already registered"}`,
			wantErr:    true,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			response:   `{"error":"server error"}`,
			wantErr:    true,
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			response:   `{"error":"invalid request"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/agents/register", r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)

				// Verify request body
				var req map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&req)
				assert.NoError(t, err)
				assert.Contains(t, req, "hostname")

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient(ClientConfig{
				ServerURL: server.URL,
				AgentID:   "test-agent",
			})
			metadata := map[string]string{"env": "test"}
			err := client.RegisterAgent(context.Background(), "test-hostname", metadata)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClientHeartbeat(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "successful heartbeat",
			statusCode: http.StatusOK,
			response:   `{"status":"ok"}`,
			wantErr:    false,
		},
		{
			name:       "agent not found",
			statusCode: http.StatusNotFound,
			response:   `{"error":"agent not found"}`,
			wantErr:    true,
		},
		{
			name:       "server unavailable",
			statusCode: http.StatusServiceUnavailable,
			response:   `{"error":"service unavailable"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "/api/v1/agents/heartbeat/")
				assert.Equal(t, http.MethodPost, r.Method)
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient(ClientConfig{
				ServerURL: server.URL,
				AgentID:   "test-agent",
			})
			err := client.Heartbeat(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClientSendReport(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "successful report submission",
			statusCode: http.StatusOK,
			response:   `{"status":"received"}`,
			wantErr:    false,
		},
		{
			name:       "accepted status",
			statusCode: http.StatusAccepted,
			response:   `{"status":"accepted"}`,
			wantErr:    false,
		},
		{
			name:       "server error on report",
			statusCode: http.StatusInternalServerError,
			response:   `{"error":"failed to process report"}`,
			wantErr:    true,
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			response:   `{"error":"invalid report format"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/reports", r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient(ClientConfig{
				ServerURL: server.URL,
				AgentID:   "test-agent",
			})

			data := &ReportData{
				Hostname: "test-host",
			}
			err := client.SendReport(context.Background(), data)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClientSendReportWithRetry(t *testing.T) {
	t.Run("succeeds on first attempt", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		}))
		defer server.Close()

		client := NewClient(ClientConfig{
			ServerURL: server.URL,
			AgentID:   "test-agent",
		})
		data := &ReportData{Hostname: "test-host"}
		err := client.SendReportWithRetry(context.Background(), data, 3, 100*time.Millisecond)
		assert.NoError(t, err)
	})

	t.Run("retries on failure then succeeds", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 2 {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"temporary error"}`))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"ok"}`))
			}
		}))
		defer server.Close()

		client := NewClient(ClientConfig{
			ServerURL: server.URL,
			AgentID:   "test-agent",
		})
		data := &ReportData{Hostname: "test-host"}
		err := client.SendReportWithRetry(context.Background(), data, 3, 50*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 2, attempts)
	})

	t.Run("fails after max retries", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"persistent error"}`))
		}))
		defer server.Close()

		client := NewClient(ClientConfig{
			ServerURL: server.URL,
			AgentID:   "test-agent",
		})
		data := &ReportData{Hostname: "test-host"}
		err := client.SendReportWithRetry(context.Background(), data, 2, 50*time.Millisecond)
		assert.Error(t, err)
		assert.Equal(t, 2, attempts)
	})
}

func TestNewClient(t *testing.T) {
	client := NewClient(ClientConfig{
		ServerURL:      "http://localhost:8080",
		AgentID:        "test-agent",
		APIKey:         "test-key",
		RequestTimeout: 10 * time.Second,
	})

	require.NotNil(t, client)
	assert.NotNil(t, client.httpClient)
}
