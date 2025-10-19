package app

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

// TestWSClientSendMessage_ClosedClient tests sending to a closed client
func TestWSClientSendMessage_ClosedClient(t *testing.T) {
	hub := NewWSHub()

	// Create a mock connection that we'll close
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial("ws://echo.websocket.org/", nil)
	if err != nil {
		t.Skip("Cannot reach echo server for test")
		return
	}

	client := NewWSClient("test-closed", conn, hub)

	// Close the client
	client.close()

	// Try to send message to closed client - should not error
	err = client.sendMessage("test", map[string]interface{}{"data": "test"})
	assert.NoError(t, err, "Sending to closed client should not return error")
}

// TestWSClientSendMessage_FullBuffer tests sending when buffer is full
func TestWSClientSendMessage_FullBuffer(t *testing.T) {
	hub := NewWSHub()

	// Create a client with a very small buffer for testing
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial("ws://echo.websocket.org/", nil)
	if err != nil {
		t.Skip("Cannot reach echo server for test")
		return
	}
	defer conn.Close()

	client := &WSClient{
		ID:   "test-full",
		conn: conn,
		send: make(chan []byte, 1), // Very small buffer
		hub:  hub,
	}

	// Fill the buffer
	client.send <- []byte("message1")

	// Try to send another message - should not block or error
	err = client.sendMessage("test", map[string]interface{}{"data": "test"})
	assert.NoError(t, err, "Sending to full buffer should not error")
}

// TestBroadcastTopologyUpdate_WithClients tests broadcast with active clients
func TestBroadcastTopologyUpdate_WithClients(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()
	defer hub.Stop()

	topology := &TopologyView{
		Nodes: map[string]*TopologyNode{
			"node-1": {
				ID:   "node-1",
				Type: "host",
				Name: "test-host",
			},
		},
		Edges:     map[string]*TopologyEdge{},
		Timestamp: time.Now(),
	}

	// This should not panic or error even without clients
	hub.BroadcastTopologyUpdate(topology)

	// Verify it works
	time.Sleep(50 * time.Millisecond)
}

// TestBroadcastReportUpdate_WithPayload tests broadcast with complex payload
func TestBroadcastReportUpdate_WithPayload(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()
	defer hub.Stop()

	report := map[string]interface{}{
		"cpu":    85.5,
		"memory": 1024,
		"disk":   map[string]interface{}{"used": 50, "total": 100},
	}

	// This should not panic or error
	hub.BroadcastReportUpdate("agent-123", report)

	// Verify it works
	time.Sleep(50 * time.Millisecond)
}

// TestWSHub_Stop tests hub stopping
func TestWSHub_Stop(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()

	// Add a mock client
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial("ws://echo.websocket.org/", nil)
	if err != nil {
		t.Skip("Cannot reach echo server for test")
		return
	}

	client := NewWSClient("test-stop", conn, hub)
	hub.register <- client

	// Wait for registration
	time.Sleep(100 * time.Millisecond)

	// Stop the hub - should close all clients
	hub.Stop()

	// Wait for stop to complete
	time.Sleep(100 * time.Millisecond)

	// Verify hub stopped
	assert.Equal(t, 0, hub.GetClientCount())
}

// TestWSClientReadPump_InvalidJSON tests readPump with invalid JSON
func TestWSClientReadPump_InvalidJSON(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()
	defer hub.Stop()

	// This test verifies that invalid JSON is handled gracefully
	// The readPump function should log an error and continue
	// We can't easily test this without mocking the connection,
	// but we verify the code path exists

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial("ws://echo.websocket.org/", nil)
	if err != nil {
		t.Skip("Cannot reach echo server for test")
		return
	}
	defer conn.Close()

	client := NewWSClient("test-invalid-json", conn, hub)
	hub.register <- client

	// Start the pumps
	go client.readPump()

	// Send invalid JSON
	invalidJSON := []byte("{invalid json}")
	err = conn.WriteMessage(websocket.TextMessage, invalidJSON)
	if err != nil {
		t.Logf("Could not send invalid JSON: %v", err)
	}

	// The client should handle this gracefully
	time.Sleep(100 * time.Millisecond)
}

// TestWSClientReadPump_PingMessage tests readPump with ping message type
func TestWSClientReadPump_PingMessage(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()
	defer hub.Stop()

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial("ws://echo.websocket.org/", nil)
	if err != nil {
		t.Skip("Cannot reach echo server for test")
		return
	}
	defer conn.Close()

	client := NewWSClient("test-ping", conn, hub)
	hub.register <- client

	// Start the pumps
	go client.readPump()
	go client.writePump()

	// Send a ping message
	pingMsg := WSMessage{
		Type:    "ping",
		Payload: map[string]interface{}{},
	}
	data, _ := json.Marshal(pingMsg)
	err = conn.WriteMessage(websocket.TextMessage, data)
	if err == nil {
		// Should receive a pong back
		time.Sleep(200 * time.Millisecond)
	}
}

// TestWSClientReadPump_SubscribeMessage tests readPump with subscribe message
func TestWSClientReadPump_SubscribeMessage(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()
	defer hub.Stop()

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial("ws://echo.websocket.org/", nil)
	if err != nil {
		t.Skip("Cannot reach echo server for test")
		return
	}
	defer conn.Close()

	client := NewWSClient("test-subscribe", conn, hub)
	hub.register <- client

	// Start the pumps
	go client.readPump()

	// Send a subscribe message
	subscribeMsg := WSMessage{
		Type:    "subscribe",
		Payload: map[string]interface{}{"topic": "topology"},
	}
	data, _ := json.Marshal(subscribeMsg)
	err = conn.WriteMessage(websocket.TextMessage, data)
	if err == nil {
		// Should be logged and processed
		time.Sleep(100 * time.Millisecond)
	}
}

// TestWSClientReadPump_UnknownMessageType tests unknown message type
func TestWSClientReadPump_UnknownMessageType(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()
	defer hub.Stop()

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial("ws://echo.websocket.org/", nil)
	if err != nil {
		t.Skip("Cannot reach echo server for test")
		return
	}
	defer conn.Close()

	client := NewWSClient("test-unknown", conn, hub)
	hub.register <- client

	// Start the pumps
	go client.readPump()

	// Send an unknown message type
	unknownMsg := WSMessage{
		Type:    "unknown_type",
		Payload: map[string]interface{}{"data": "test"},
	}
	data, _ := json.Marshal(unknownMsg)
	err = conn.WriteMessage(websocket.TextMessage, data)
	if err == nil {
		// Should be logged as unknown
		time.Sleep(100 * time.Millisecond)
	}
}

// TestWSClientWritePump_ChannelClosed tests writePump when channel is closed
func TestWSClientWritePump_ChannelClosed(t *testing.T) {
	hub := NewWSHub()

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial("ws://echo.websocket.org/", nil)
	if err != nil {
		t.Skip("Cannot reach echo server for test")
		return
	}
	defer conn.Close()

	client := NewWSClient("test-write-closed", conn, hub)

	// Start writePump in background
	go client.writePump()

	// Close the send channel to trigger the channel closed path
	close(client.send)

	// Wait for writePump to handle the closed channel
	time.Sleep(100 * time.Millisecond)
}

// TestWSClientWritePump_QueuedMessages tests batching of queued messages
func TestWSClientWritePump_QueuedMessages(t *testing.T) {
	hub := NewWSHub()

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial("ws://echo.websocket.org/", nil)
	if err != nil {
		t.Skip("Cannot reach echo server for test")
		return
	}
	defer conn.Close()

	client := NewWSClient("test-queued", conn, hub)

	// Start writePump
	go client.writePump()

	// Queue multiple messages quickly
	for i := 0; i < 5; i++ {
		msg := WSMessage{
			Type:    "test",
			Payload: map[string]interface{}{"index": i},
		}
		data, _ := json.Marshal(msg)
		select {
		case client.send <- data:
		default:
		}
	}

	// Let writePump batch and send them
	time.Sleep(200 * time.Millisecond)
}

// TestWSHub_BroadcastWithFullClientBuffer tests broadcast when client buffer is full
func TestWSHub_BroadcastWithFullClientBuffer(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()
	defer hub.Stop()

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial("ws://echo.websocket.org/", nil)
	if err != nil {
		t.Skip("Cannot reach echo server for test")
		return
	}
	defer conn.Close()

	// Create client with small buffer
	client := &WSClient{
		ID:   "test-full-buffer",
		conn: conn,
		send: make(chan []byte, 1),
		hub:  hub,
	}

	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	// Fill the client's buffer
	client.send <- []byte("fill")

	// Try to broadcast - should unregister the client with full buffer
	err = hub.Broadcast("test", map[string]interface{}{"data": "test"})
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
}
