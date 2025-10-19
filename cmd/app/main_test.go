package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ljluestc/orchestrator/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestPrintConfig(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := app.ServerConfig{
		Host:               "127.0.0.1",
		Port:               9090,
		MaxDataAge:         2 * time.Hour,
		CleanupInterval:    10 * time.Minute,
		StaleNodeThreshold: 15 * time.Minute,
	}

	printConfig(config)

	// Restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains key information
	assert.Contains(t, output, "App Backend Server Configuration")
	assert.Contains(t, output, "Host:                  127.0.0.1")
	assert.Contains(t, output, "Port:                  9090")
	assert.Contains(t, output, "Max Data Age:          2h0m0s")
	assert.Contains(t, output, "Cleanup Interval:      10m0s")
	assert.Contains(t, output, "Stale Node Threshold:  15m0s")
	assert.Contains(t, output, "REST API:")
	assert.Contains(t, output, "WebSocket:")
	assert.Contains(t, output, "Health Check:")
	assert.Contains(t, output, "/api/v1/agents/register")
	assert.Contains(t, output, "/api/v1/query/topology")
}

func TestPrintConfigDefaultValues(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := app.ServerConfig{
		Host:               "0.0.0.0",
		Port:               8080,
		MaxDataAge:         1 * time.Hour,
		CleanupInterval:    5 * time.Minute,
		StaleNodeThreshold: 5 * time.Minute,
	}

	printConfig(config)

	// Restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify default values are printed correctly
	assert.Contains(t, output, "0.0.0.0")
	assert.Contains(t, output, "8080")
	assert.Contains(t, output, "1h0m0s")
	assert.Contains(t, output, "5m0s")
}

func TestPrintConfigEndpoints(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := app.ServerConfig{
		Host:               "localhost",
		Port:               3000,
		MaxDataAge:         30 * time.Minute,
		CleanupInterval:    2 * time.Minute,
		StaleNodeThreshold: 3 * time.Minute,
	}

	printConfig(config)

	// Restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify all endpoints are listed
	endpoints := []string{
		"/api/v1/agents/register",
		"/api/v1/agents/heartbeat/:agent_id",
		"/api/v1/agents/config/:agent_id",
		"/api/v1/agents/list",
		"/api/v1/reports",
		"/api/v1/query/topology",
		"/api/v1/query/agents/:agent_id/latest",
		"/api/v1/query/agents/:agent_id/timeseries",
		"/api/v1/query/stats",
		"/api/v1/ws",
	}

	for _, endpoint := range endpoints {
		assert.Contains(t, output, endpoint, "Output should contain endpoint: %s", endpoint)
	}
}

func TestPrintConfigURLFormats(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := app.ServerConfig{
		Host:               "192.168.1.100",
		Port:               5000,
		MaxDataAge:         45 * time.Minute,
		CleanupInterval:    3 * time.Minute,
		StaleNodeThreshold: 7 * time.Minute,
	}

	printConfig(config)

	// Restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify URLs are formatted correctly
	assert.Contains(t, output, "http://192.168.1.100:5000/api/v1")
	assert.Contains(t, output, "ws://192.168.1.100:5000/api/v1/ws")
	assert.Contains(t, output, "http://192.168.1.100:5000/health")
}

func TestPrintConfigStructure(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := app.ServerConfig{
		Host:               "test-host",
		Port:               7777,
		MaxDataAge:         10 * time.Minute,
		CleanupInterval:    1 * time.Minute,
		StaleNodeThreshold: 2 * time.Minute,
	}

	printConfig(config)

	// Restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify section headers are present
	sections := []string{
		"=== App Backend Server Configuration ===",
		"=== Endpoints ===",
		"=== Available Endpoints ===",
	}

	for _, section := range sections {
		assert.Contains(t, output, section, "Output should contain section: %s", section)
	}

	// Verify output contains separator lines
	separatorCount := strings.Count(output, "========")
	assert.GreaterOrEqual(t, separatorCount, 1, "Output should contain separator lines")
}
