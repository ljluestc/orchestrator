package zkclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-zookeeper/zk"
)

// ConnectionState represents the current state of the connection
type ConnectionState int

const (
	// StateDisconnected indicates no connection
	StateDisconnected ConnectionState = iota
	// StateConnecting indicates connection attempt in progress
	StateConnecting
	// StateConnected indicates active connection
	StateConnected
	// StateReconnecting indicates reconnection attempt in progress
	StateReconnecting
	// StateClosed indicates client is closed
	StateClosed
)

// String returns string representation of connection state
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateReconnecting:
		return "reconnecting"
	case StateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// Client is a robust Zookeeper client wrapper
type Client struct {
	config *Config
	conn   *zk.Conn

	state          ConnectionState
	stateMu        sync.RWMutex

	eventCh        <-chan zk.Event
	stopCh         chan struct{}
	reconnectCh    chan struct{}

	reconnectCount int
	lastConnTime   time.Time

	healthStatus   *HealthStatus
	healthMu       sync.RWMutex

	wg             sync.WaitGroup

	logger         Logger
}

// HealthStatus represents the health status of the Zookeeper connection
type HealthStatus struct {
	IsHealthy      bool
	LastCheckTime  time.Time
	QuorumHealthy  bool
	SessionActive  bool
	LastError      error
}

// NewClient creates a new Zookeeper client
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	client := &Client{
		config:       config.Clone(),
		state:        StateDisconnected,
		stopCh:       make(chan struct{}),
		reconnectCh:  make(chan struct{}, 1),
		logger:       config.Logger,
		healthStatus: &HealthStatus{},
	}

	return client, nil
}

// Connect establishes a connection to the Zookeeper cluster
func (c *Client) Connect(ctx context.Context) error {
	c.stateMu.Lock()
	if c.state == StateClosed {
		c.stateMu.Unlock()
		return ErrConnectionClosed
	}
	if c.state == StateConnected || c.state == StateConnecting {
		c.stateMu.Unlock()
		return nil
	}
	c.state = StateConnecting
	c.stateMu.Unlock()

	c.logger.Info("Connecting to Zookeeper cluster: %v", c.config.Servers)

	// Set up connection options (go-zookeeper doesn't use ConnOption anymore)
	// Connection is created directly with zk.Connect

	// Configure TLS if enabled
	if c.config.TLSEnabled {
		tlsConfig, err := c.buildTLSConfig()
		if err != nil {
			c.setState(StateDisconnected)
			return fmt.Errorf("failed to build TLS config: %w", err)
		}
		// Note: go-zookeeper doesn't directly support TLS in the standard way
		// This would require custom dialer implementation
		c.logger.Warn("TLS support requires custom dialer implementation")
		_ = tlsConfig // Placeholder for future TLS implementation
	}

	// Create connection
	conn, eventCh, err := zk.Connect(
		c.config.Servers,
		c.config.SessionTimeout,
	)
	if err != nil {
		c.setState(StateDisconnected)
		return &ConnectionError{
			Op:      "connect",
			Servers: c.config.Servers,
			Err:     err,
		}
	}

	c.conn = conn
	c.eventCh = eventCh
	c.lastConnTime = time.Now()

	// Wait for connection with timeout
	connectTimeout := c.config.ConnectionTimeout
	if connectTimeout == 0 {
		connectTimeout = 10 * time.Second
	}

	connectedCh := make(chan bool, 1)
	go func() {
		for event := range eventCh {
			if event.State == zk.StateConnected || event.State == zk.StateHasSession {
				connectedCh <- true
				return
			}
		}
		connectedCh <- false
	}()

	select {
	case <-ctx.Done():
		c.closeConnection()
		c.setState(StateDisconnected)
		return ctx.Err()
	case connected := <-connectedCh:
		if !connected {
			c.closeConnection()
			c.setState(StateDisconnected)
			return &ConnectionError{
				Op:      "connect",
				Servers: c.config.Servers,
				Err:     fmt.Errorf("failed to establish connection"),
			}
		}
	case <-time.After(connectTimeout):
		c.closeConnection()
		c.setState(StateDisconnected)
		return &ConnectionError{
			Op:      "connect",
			Servers: c.config.Servers,
			Err:     fmt.Errorf("connection timeout after %v", connectTimeout),
		}
	}

	// Apply authentication if configured
	if err := c.authenticate(); err != nil {
		c.closeConnection()
		c.setState(StateDisconnected)
		return err
	}

	c.setState(StateConnected)
	c.logger.Info("Successfully connected to Zookeeper cluster")

	// Start event monitoring and health checking
	c.wg.Add(2)
	go c.eventLoop()
	go c.healthCheckLoop()

	return nil
}

// Close closes the Zookeeper connection
func (c *Client) Close() error {
	c.stateMu.Lock()
	if c.state == StateClosed {
		c.stateMu.Unlock()
		return nil
	}
	c.state = StateClosed
	c.stateMu.Unlock()

	c.logger.Info("Closing Zookeeper client")

	// Signal goroutines to stop
	select {
	case <-c.stopCh:
		// Already closed
	default:
		close(c.stopCh)
	}

	// Wait for goroutines to finish
	c.wg.Wait()

	// Close connection
	c.closeConnection()

	c.logger.Info("Zookeeper client closed")

	return nil
}

// closeConnection closes the underlying connection
func (c *Client) closeConnection() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

// GetState returns the current connection state
func (c *Client) GetState() ConnectionState {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state
}

// setState sets the connection state
func (c *Client) setState(state ConnectionState) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	c.state = state
	c.logger.Debug("Connection state changed to: %s", state)
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	return c.GetState() == StateConnected
}

// GetConn returns the underlying Zookeeper connection
// Returns error if not connected
func (c *Client) GetConn() (*zk.Conn, error) {
	if !c.IsConnected() {
		return nil, ErrNotConnected
	}
	if c.conn == nil {
		return nil, ErrNotConnected
	}
	return c.conn, nil
}

// buildTLSConfig builds TLS configuration from config
func (c *Client) buildTLSConfig() (*tls.Config, error) {
	if c.config.TLSConfig == nil {
		return nil, ErrMissingTLSConfig
	}

	tlsConfig := &tls.Config{
		ServerName:         c.config.TLSConfig.ServerName,
		InsecureSkipVerify: c.config.TLSConfig.InsecureSkipVerify,
	}

	// Load CA certificate if provided
	if c.config.TLSConfig.CAFile != "" {
		caCert, err := os.ReadFile(c.config.TLSConfig.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Load client certificate if provided
	if c.config.TLSConfig.CertFile != "" && c.config.TLSConfig.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.config.TLSConfig.CertFile, c.config.TLSConfig.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// authenticate performs authentication based on config
func (c *Client) authenticate() error {
	if c.config.AuthType == AuthTypeNone {
		return nil
	}

	if c.conn == nil {
		return ErrNotConnected
	}

	switch c.config.AuthType {
	case AuthTypeDigest:
		err := c.conn.AddAuth("digest", []byte(c.config.AuthData))
		if err != nil {
			return &AuthenticationError{
				AuthType: AuthTypeDigest,
				Err:      err,
			}
		}
		c.logger.Info("Digest authentication successful")

	case AuthTypeSASL:
		// Note: SASL authentication in go-zookeeper requires additional setup
		// This is a placeholder for SASL implementation
		c.logger.Warn("SASL authentication requires additional implementation")
		return &AuthenticationError{
			AuthType: AuthTypeSASL,
			Err:      fmt.Errorf("SASL not fully implemented"),
		}

	default:
		return &AuthenticationError{
			AuthType: c.config.AuthType,
			Err:      fmt.Errorf("unsupported auth type: %s", c.config.AuthType),
		}
	}

	return nil
}

// eventLoop monitors connection events
func (c *Client) eventLoop() {
	defer c.wg.Done()

	for {
		select {
		case <-c.stopCh:
			c.logger.Debug("Event loop stopping")
			return
		case event, ok := <-c.eventCh:
			if !ok {
				c.logger.Warn("Event channel closed")
				// Trigger reconnection
				select {
				case c.reconnectCh <- struct{}{}:
				default:
				}
				return
			}

			c.logger.Debug("Received event: %v", event)

			switch event.State {
			case zk.StateDisconnected:
				c.setState(StateDisconnected)
				c.logger.Warn("Disconnected from Zookeeper")
				// Trigger reconnection
				select {
				case c.reconnectCh <- struct{}{}:
				default:
				}
			case zk.StateExpired:
				c.setState(StateDisconnected)
				c.logger.Error("Session expired")
				// Trigger reconnection
				select {
				case c.reconnectCh <- struct{}{}:
				default:
				}
			case zk.StateHasSession, zk.StateConnected:
				c.setState(StateConnected)
				c.logger.Info("Session established")
			case zk.StateConnecting:
				c.setState(StateConnecting)
				c.logger.Debug("Connecting to Zookeeper")
			}
		}
	}
}
