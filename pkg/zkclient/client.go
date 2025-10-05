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

// Client is a wrapper around the Zookeeper client with connection pooling and retry logic
type Client struct {
	config *Config
	logger Logger

	// Connection pool
	pool     *connectionPool
	poolLock sync.RWMutex

	// Single connection mode
	conn     *zk.Conn
	connLock sync.RWMutex

	// Event channel for connection state changes
	eventChan <-chan zk.Event

	// Reconnection management
	reconnectCtx    context.Context
	reconnectCancel context.CancelFunc
	reconnectWg     sync.WaitGroup

	// Health check management
	healthCheckCtx    context.Context
	healthCheckCancel context.CancelFunc
	healthCheckWg     sync.WaitGroup

	// State
	state     ConnectionState
	stateLock sync.RWMutex

	closed     bool
	closedLock sync.RWMutex
}

// ConnectionState represents the state of the Zookeeper connection
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateReconnecting
	StateClosed
)

func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "Disconnected"
	case StateConnecting:
		return "Connecting"
	case StateConnected:
		return "Connected"
	case StateReconnecting:
		return "Reconnecting"
	case StateClosed:
		return "Closed"
	default:
		return "Unknown"
	}
}

// NewClient creates a new Zookeeper client
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	logger := config.Logger
	if logger == nil {
		logger = &defaultLogger{}
	}

	reconnectCtx, reconnectCancel := context.WithCancel(context.Background())
	healthCheckCtx, healthCheckCancel := context.WithCancel(context.Background())

	client := &Client{
		config:            config,
		logger:            logger,
		reconnectCtx:      reconnectCtx,
		reconnectCancel:   reconnectCancel,
		healthCheckCtx:    healthCheckCtx,
		healthCheckCancel: healthCheckCancel,
		state:             StateDisconnected,
	}

	return client, nil
}

// Connect establishes a connection to Zookeeper
func (c *Client) Connect() error {
	c.setState(StateConnecting)
	c.logger.Info("Connecting to Zookeeper servers: %v", c.config.Servers)

	if c.config.EnableConnectionPool {
		return c.connectWithPool()
	}
	return c.connectSingle()
}

// connectSingle establishes a single connection to Zookeeper
func (c *Client) connectSingle() error {
	conn, eventChan, err := c.createConnection()
	if err != nil {
		c.setState(StateDisconnected)
		return &ConnectionError{
			Op:      "connect",
			Servers: c.config.Servers,
			Err:     err,
		}
	}

	c.connLock.Lock()
	c.conn = conn
	c.eventChan = eventChan
	c.connLock.Unlock()

	// Apply authentication if configured
	if err := c.authenticate(conn); err != nil {
		conn.Close()
		c.setState(StateDisconnected)
		return err
	}

	c.setState(StateConnected)
	c.logger.Info("Successfully connected to Zookeeper")

	// Start background goroutines
	c.reconnectWg.Add(1)
	go c.handleEvents()

	c.healthCheckWg.Add(1)
	go c.runHealthChecks()

	return nil
}

// connectWithPool establishes a connection pool
func (c *Client) connectWithPool() error {
	pool, err := newConnectionPool(c.config, c.logger)
	if err != nil {
		c.setState(StateDisconnected)
		return err
	}

	c.poolLock.Lock()
	c.pool = pool
	c.poolLock.Unlock()

	c.setState(StateConnected)
	c.logger.Info("Successfully created connection pool with %d connections", c.config.PoolSize)

	// Start health checks for the pool
	c.healthCheckWg.Add(1)
	go c.runPoolHealthChecks()

	return nil
}

// createConnection creates a new Zookeeper connection
func (c *Client) createConnection() (*zk.Conn, <-chan zk.Event, error) {
	// Setup TLS if enabled
	var dialer zk.Dialer
	if c.config.TLSEnabled && c.config.TLSConfig != nil {
		tlsConfig, err := c.buildTLSConfig()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build TLS config: %w", err)
		}
		dialer = zk.WithDialer(&tls.Dialer{Config: tlsConfig})
	}

	// Create connection with timeout
	conn, eventChan, err := zk.Connect(
		c.config.Servers,
		c.config.SessionTimeout,
		dialer,
	)
	if err != nil {
		return nil, nil, err
	}

	// Wait for connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.config.ConnectionTimeout)
	defer cancel()

	select {
	case event := <-eventChan:
		if event.State == zk.StateHasSession {
			return conn, eventChan, nil
		}
		conn.Close()
		return nil, nil, fmt.Errorf("failed to establish session: %v", event.State)
	case <-ctx.Done():
		conn.Close()
		return nil, nil, fmt.Errorf("connection timeout: %w", ctx.Err())
	}
}

// buildTLSConfig builds TLS configuration from the config
func (c *Client) buildTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.config.TLSConfig.InsecureSkipVerify,
		ServerName:         c.config.TLSConfig.ServerName,
	}

	// Load client certificate if provided
	if c.config.TLSConfig.CertFile != "" && c.config.TLSConfig.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.config.TLSConfig.CertFile, c.config.TLSConfig.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
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

	return tlsConfig, nil
}

// authenticate applies authentication to the connection
func (c *Client) authenticate(conn *zk.Conn) error {
	switch c.config.AuthType {
	case AuthTypeNone:
		return nil
	case AuthTypeDigest:
		if c.config.AuthData == "" {
			return &AuthenticationError{
				AuthType: AuthTypeDigest,
				Err:      ErrMissingAuthCredentials,
			}
		}
		err := conn.AddAuth("digest", []byte(c.config.AuthData))
		if err != nil {
			return &AuthenticationError{
				AuthType: AuthTypeDigest,
				Err:      err,
			}
		}
		c.logger.Info("Applied digest authentication")
		return nil
	case AuthTypeSASL:
		if c.config.AuthData == "" || c.config.AuthPassword == "" {
			return &AuthenticationError{
				AuthType: AuthTypeSASL,
				Err:      ErrMissingAuthCredentials,
			}
		}
		authData := []byte(fmt.Sprintf("%s:%s", c.config.AuthData, c.config.AuthPassword))
		err := conn.AddAuth("sasl", authData)
		if err != nil {
			return &AuthenticationError{
				AuthType: AuthTypeSASL,
				Err:      err,
			}
		}
		c.logger.Info("Applied SASL authentication")
		return nil
	default:
		return fmt.Errorf("unsupported authentication type: %s", c.config.AuthType)
	}
}

// GetConnection returns a connection from the pool or the single connection
func (c *Client) GetConnection() (*zk.Conn, error) {
	if c.isClosed() {
		return nil, ErrConnectionClosed
	}

	if c.config.EnableConnectionPool {
		c.poolLock.RLock()
		pool := c.pool
		c.poolLock.RUnlock()

		if pool == nil {
			return nil, ErrNotConnected
		}
		return pool.Get()
	}

	c.connLock.RLock()
	conn := c.conn
	c.connLock.RUnlock()

	if conn == nil {
		return nil, ErrNotConnected
	}

	if conn.State() != zk.StateHasSession {
		return nil, ErrNotConnected
	}

	return conn, nil
}

// ReleaseConnection returns a connection to the pool (only for pooled connections)
func (c *Client) ReleaseConnection(conn *zk.Conn) {
	if c.config.EnableConnectionPool {
		c.poolLock.RLock()
		pool := c.pool
		c.poolLock.RUnlock()

		if pool != nil {
			pool.Put(conn)
		}
	}
}

// handleEvents handles connection events and triggers reconnection if needed
func (c *Client) handleEvents() {
	defer c.reconnectWg.Done()

	c.connLock.RLock()
	eventChan := c.eventChan
	c.connLock.RUnlock()

	for {
		select {
		case <-c.reconnectCtx.Done():
			return
		case event, ok := <-eventChan:
			if !ok {
				c.logger.Warn("Event channel closed")
				c.attemptReconnect()
				return
			}

			c.logger.Debug("Received event: %+v", event)

			switch event.State {
			case zk.StateDisconnected:
				c.setState(StateDisconnected)
				c.logger.Warn("Disconnected from Zookeeper")
				c.attemptReconnect()
			case zk.StateExpired:
				c.setState(StateDisconnected)
				c.logger.Error("Session expired")
				c.attemptReconnect()
			case zk.StateHasSession:
				c.setState(StateConnected)
				c.logger.Info("Session established")
			case zk.StateConnecting:
				c.setState(StateConnecting)
				c.logger.Debug("Connecting to Zookeeper")
			}
		}
	}
}

// Close closes the Zookeeper connection and cleans up resources
func (c *Client) Close() error {
	c.closedLock.Lock()
	if c.closed {
		c.closedLock.Unlock()
		return nil
	}
	c.closed = true
	c.closedLock.Unlock()

	c.logger.Info("Closing Zookeeper client")

	// Cancel background goroutines
	c.reconnectCancel()
	c.healthCheckCancel()

	// Wait for goroutines to finish
	c.reconnectWg.Wait()
	c.healthCheckWg.Wait()

	// Close connection or pool
	if c.config.EnableConnectionPool {
		c.poolLock.Lock()
		if c.pool != nil {
			c.pool.Close()
		}
		c.poolLock.Unlock()
	} else {
		c.connLock.Lock()
		if c.conn != nil {
			c.conn.Close()
		}
		c.connLock.Unlock()
	}

	c.setState(StateClosed)
	c.logger.Info("Zookeeper client closed")

	return nil
}

// GetState returns the current connection state
func (c *Client) GetState() ConnectionState {
	c.stateLock.RLock()
	defer c.stateLock.RUnlock()
	return c.state
}

// setState updates the connection state
func (c *Client) setState(state ConnectionState) {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	c.state = state
}

// isClosed checks if the client is closed
func (c *Client) isClosed() bool {
	c.closedLock.RLock()
	defer c.closedLock.RUnlock()
	return c.closed
}

// IsConnected returns true if the client is connected
func (c *Client) IsConnected() bool {
	state := c.GetState()
	return state == StateConnected
}
