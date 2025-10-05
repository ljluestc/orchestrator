package app

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// WSClient represents a WebSocket client connection
type WSClient struct {
	ID       string
	conn     *websocket.Conn
	send     chan []byte
	hub      *WSHub
	mu       sync.Mutex
	isClosed bool
}

// WSHub manages WebSocket connections
type WSHub struct {
	clients    map[*WSClient]bool
	broadcast  chan []byte
	register   chan *WSClient
	unregister chan *WSClient
	mu         sync.RWMutex
	stopCh     chan struct{}
}

// NewWSHub creates a new WebSocket hub
func NewWSHub() *WSHub {
	return &WSHub{
		clients:    make(map[*WSClient]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
		stopCh:     make(chan struct{}),
	}
}

// Run starts the WebSocket hub
func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client registered: %s (total: %d)", client.ID, len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.close()
				log.Printf("WebSocket client unregistered: %s (total: %d)", client.ID, len(h.clients))
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client's send buffer is full, close the connection
					h.mu.RUnlock()
					h.unregister <- client
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()

		case <-h.stopCh:
			// Close all clients
			h.mu.Lock()
			for client := range h.clients {
				client.close()
			}
			h.clients = make(map[*WSClient]bool)
			h.mu.Unlock()
			return
		}
	}
}

// Stop stops the WebSocket hub
func (h *WSHub) Stop() {
	close(h.stopCh)
}

// Broadcast sends a message to all connected clients
func (h *WSHub) Broadcast(msgType string, payload interface{}) error {
	msg := WSMessage{
		Type:    msgType,
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	h.broadcast <- data
	return nil
}

// BroadcastTopologyUpdate sends topology updates to all clients
func (h *WSHub) BroadcastTopologyUpdate(topology *TopologyView) {
	if err := h.Broadcast("topology_update", topology); err != nil {
		log.Printf("Failed to broadcast topology update: %v", err)
	}
}

// BroadcastReportUpdate sends report updates to all clients
func (h *WSHub) BroadcastReportUpdate(agentID string, report interface{}) {
	payload := map[string]interface{}{
		"agent_id": agentID,
		"report":   report,
	}

	if err := h.Broadcast("report_update", payload); err != nil {
		log.Printf("Failed to broadcast report update: %v", err)
	}
}

// GetClientCount returns the number of connected clients
func (h *WSHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// close closes the client connection
func (c *WSClient) close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isClosed {
		close(c.send)
		c.conn.Close()
		c.isClosed = true
	}
}

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512 KB
)

// readPump reads messages from the WebSocket connection
func (c *WSClient) readPump() {
	defer func() {
		c.hub.unregister <- c
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error for client %s: %v", c.ID, err)
			}
			break
		}

		// Handle incoming messages (e.g., subscriptions, filters)
		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to unmarshal WebSocket message from client %s: %v", c.ID, err)
			continue
		}

		// Handle different message types
		switch msg.Type {
		case "ping":
			c.sendMessage("pong", map[string]interface{}{"timestamp": time.Now()})
		case "subscribe":
			// Handle subscription requests
			log.Printf("Client %s subscribed to: %v", c.ID, msg.Payload)
		default:
			log.Printf("Unknown message type from client %s: %s", c.ID, msg.Type)
		}
	}
}

// writePump writes messages to the WebSocket connection
func (c *WSClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// sendMessage sends a message to the client
func (c *WSClient) sendMessage(msgType string, payload interface{}) error {
	msg := WSMessage{
		Type:    msgType,
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isClosed {
		return nil
	}

	select {
	case c.send <- data:
	default:
		// Send buffer is full
		return nil
	}

	return nil
}

// NewWSClient creates a new WebSocket client
func NewWSClient(id string, conn *websocket.Conn, hub *WSHub) *WSClient {
	return &WSClient{
		ID:   id,
		conn: conn,
		send: make(chan []byte, 256),
		hub:  hub,
	}
}

// Start starts the client's read and write pumps
func (c *WSClient) Start() {
	go c.writePump()
	go c.readPump()
}
