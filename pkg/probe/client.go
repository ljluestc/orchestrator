package probe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ReportData contains all collected data to be sent to the app component
type ReportData struct {
	HostInfo      *HostInfo      `json:"host_info,omitempty"`
	DockerInfo    *DockerInfo    `json:"docker_info,omitempty"`
	ProcessesInfo *ProcessesInfo `json:"processes_info,omitempty"`
	NetworkInfo   *NetworkInfo   `json:"network_info,omitempty"`
	Timestamp     time.Time      `json:"timestamp"`
	AgentID       string         `json:"agent_id"`
	Hostname      string         `json:"hostname"`
}

// Client handles communication with the app component
type Client struct {
	serverURL  string
	httpClient *http.Client
	agentID    string
	apiKey     string
}

// ClientConfig contains configuration for the client
type ClientConfig struct {
	ServerURL      string
	AgentID        string
	APIKey         string
	RequestTimeout time.Duration
	RetryAttempts  int
	RetryDelay     time.Duration
}

// NewClient creates a new client for communicating with the app component
func NewClient(config ClientConfig) *Client {
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}

	return &Client{
		serverURL: config.ServerURL,
		httpClient: &http.Client{
			Timeout: config.RequestTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		agentID: config.AgentID,
		apiKey:  config.APIKey,
	}
}

// SendReport sends collected data to the app component via HTTP
func (c *Client) SendReport(ctx context.Context, data *ReportData) error {
	// Set agent ID and timestamp
	data.AgentID = c.agentID
	data.Timestamp = time.Now()

	// Marshal data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal report data: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/reports", c.serverURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}
	req.Header.Set("X-Agent-ID", c.agentID)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned error status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendReportWithRetry sends a report with retry logic
func (c *Client) SendReportWithRetry(ctx context.Context, data *ReportData, attempts int, delay time.Duration) error {
	var lastErr error

	for i := 0; i < attempts; i++ {
		err := c.SendReport(ctx, data)
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Wait before retrying (except on last attempt)
		if i < attempts-1 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", attempts, lastErr)
}

// Ping checks connectivity with the app component
func (c *Client) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/v1/ping", c.serverURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create ping request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}
	req.Header.Set("X-Agent-ID", c.agentID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to ping server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ping failed with status: %d", resp.StatusCode)
	}

	return nil
}

// GetConfig retrieves configuration from the app component
func (c *Client) GetConfig(ctx context.Context) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v1/agents/config/%s", c.serverURL, c.agentID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create config request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}
	req.Header.Set("X-Agent-ID", c.agentID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get config, status %d: %s", resp.StatusCode, string(body))
	}

	var config map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return config, nil
}

// RegisterAgent registers the agent with the app component
func (c *Client) RegisterAgent(ctx context.Context, hostname string, metadata map[string]string) error {
	registration := map[string]interface{}{
		"agent_id":  c.agentID,
		"hostname":  hostname,
		"metadata":  metadata,
		"timestamp": time.Now(),
	}

	jsonData, err := json.Marshal(registration)
	if err != nil {
		return fmt.Errorf("failed to marshal registration: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/agents/register", c.serverURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create registration request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to register agent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Heartbeat sends a heartbeat to the app component
func (c *Client) Heartbeat(ctx context.Context) error {
	heartbeat := map[string]interface{}{
		"agent_id":  c.agentID,
		"timestamp": time.Now(),
	}

	jsonData, err := json.Marshal(heartbeat)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/agents/heartbeat/%s", c.serverURL, c.agentID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create heartbeat request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}
	req.Header.Set("X-Agent-ID", c.agentID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat failed with status: %d", resp.StatusCode)
	}

	return nil
}

// Close closes the HTTP client
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}
