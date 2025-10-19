package app

import (
	"testing"
	"time"
)

// TestBroadcastTopologyUpdateWithMarshalError tests error handling in BroadcastTopologyUpdate
func TestBroadcastTopologyUpdateWithMarshalError(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()
	defer hub.Stop()

	// Create a topology with an unmarshalable field (channel)
	// Channels cannot be marshaled to JSON, which will trigger the error path
	type BadTopology struct {
		Nodes     map[string]*TopologyNode
		Edges     map[string]*TopologyEdge
		Timestamp time.Time
		BadField  chan int // This cannot be marshaled to JSON
	}

	badTopology := &BadTopology{
		Nodes: map[string]*TopologyNode{
			"node-1": {
				ID:   "node-1",
				Type: "host",
				Name: "test-host",
			},
		},
		Edges:     map[string]*TopologyEdge{},
		Timestamp: time.Now(),
		BadField:  make(chan int), // This will cause json.Marshal to fail
	}

	// This should trigger the error log path in Broadcast
	// We can't directly test the log output, but we exercise the error path
	_ = hub.Broadcast("test_bad_topology", badTopology)

	// Give time for processing
	time.Sleep(100 * time.Millisecond)
}

// TestBroadcastReportUpdateWithMarshalError tests error handling in BroadcastReportUpdate
func TestBroadcastReportUpdateWithMarshalError(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()
	defer hub.Stop()

	// Create a report with an unmarshalable field
	type BadReport struct {
		Data      string
		BadField  chan int // This cannot be marshaled to JSON
	}

	badReport := BadReport{
		Data:     "test-data",
		BadField: make(chan int),
	}

	// This should trigger the error log path in BroadcastReportUpdate
	hub.BroadcastReportUpdate("test-agent", badReport)

	// Give time for processing
	time.Sleep(100 * time.Millisecond)
}

// TestBroadcastWithChannelError tests the error path when Broadcast fails
func TestBroadcastWithChannelError(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()
	defer hub.Stop()

	// Create payload with unmarshalable data
	badPayload := make(chan int)

	// This should fail during json.Marshal and return an error
	err := hub.Broadcast("test_message", badPayload)
	if err == nil {
		t.Error("Expected error from Broadcast with unmarshalable data, got nil")
	}
}
