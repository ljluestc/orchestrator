package zkclient

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-zookeeper/zk"
)

// connectionPool manages a pool of Zookeeper connections
type connectionPool struct {
	config *Config
	logger Logger

	connections chan *pooledConnection
	mu          sync.RWMutex

	stats poolStats

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	closed bool
}

// pooledConnection wraps a Zookeeper connection with metadata
type pooledConnection struct {
	conn         *zk.Conn
	eventChan    <-chan zk.Event
	createdAt    time.Time
	lastUsedAt   time.Time
	usageCount   int64
	mu           sync.RWMutex
	isHealthy    bool
	reconnecting bool
}

// poolStats tracks connection pool statistics
type poolStats struct {
	mu              sync.RWMutex
	activeConns     int
	idleConns       int
	totalConns      int
	totalRequests   int64
	failedRequests  int64
	reconnections   int64
	healthCheckFail int64
}

// newConnectionPool creates a new connection pool
func newConnectionPool(config *Config, logger Logger) (*connectionPool, error) {
	if config.PoolSize <= 0 {
		return nil, ErrInvalidPoolSize
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &connectionPool{
		config:      config,
		logger:      logger,
		connections: make(chan *pooledConnection, config.PoolSize),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Initialize the pool with minimum connections
	minConns := config.PoolSize
	if minConns <= 0 {
		minConns = 1
	}

	for i := 0; i < minConns; i++ {
		pooledConn, err := pool.createPooledConnection()
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to create initial connection %d: %w", i, err)
		}
		pool.connections <- pooledConn
		pool.stats.incrementTotalConns()
		pool.stats.incrementIdleConns()
	}

	logger.Info("Connection pool initialized with %d connections", minConns)

	// Start background maintenance
	pool.wg.Add(1)
	go pool.maintenanceLoop()

	return pool, nil
}

// createPooledConnection creates a new pooled connection
func (p *connectionPool) createPooledConnection() (*pooledConnection, error) {
	var dialer zk.Dialer
	if p.config.TLSEnabled && p.config.TLSConfig != nil {
		tlsConfig, err := buildTLSConfigFromConfig(p.config.TLSConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build TLS config: %w", err)
		}
		dialer = zk.WithDialer(&tlsDialer{config: tlsConfig})
	}

	conn, eventChan, err := zk.Connect(
		p.config.Servers,
		p.config.SessionTimeout,
		dialer,
	)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.config.ConnectionTimeout)
	defer cancel()

	select {
	case event := <-eventChan:
		if event.State == zk.StateHasSession {
			pooledConn := &pooledConnection{
				conn:       conn,
				eventChan:  eventChan,
				createdAt:  time.Now(),
				lastUsedAt: time.Now(),
				isHealthy:  true,
			}

			if err := p.authenticateConnection(conn); err != nil {
				conn.Close()
				return nil, err
			}

			p.wg.Add(1)
			go p.monitorConnection(pooledConn)

			return pooledConn, nil
		}
		conn.Close()
		return nil, fmt.Errorf("failed to establish session: %v", event.State)
	case <-ctx.Done():
		conn.Close()
		return nil, fmt.Errorf("connection timeout: %w", ctx.Err())
	}
}

// authenticateConnection applies authentication to a connection
func (p *connectionPool) authenticateConnection(conn *zk.Conn) error {
	switch p.config.AuthType {
	case AuthTypeNone:
		return nil
	case AuthTypeDigest:
		if p.config.AuthData == "" {
			return ErrMissingAuthCredentials
		}
		return conn.AddAuth("digest", []byte(p.config.AuthData))
	case AuthTypeSASL:
		if p.config.AuthData == "" || p.config.AuthPassword == "" {
			return ErrMissingAuthCredentials
		}
		authData := []byte(fmt.Sprintf("%s:%s", p.config.AuthData, p.config.AuthPassword))
		return conn.AddAuth("sasl", authData)
	default:
		return fmt.Errorf("unsupported authentication type: %s", p.config.AuthType)
	}
}

// Get retrieves a connection from the pool
func (p *connectionPool) Get() (*zk.Conn, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, ErrConnectionClosed
	}
	p.mu.RUnlock()

	p.stats.incrementTotalRequests()

	select {
	case pooledConn := <-p.connections:
		if !pooledConn.isHealthy || pooledConn.conn.State() != zk.StateHasSession {
			p.logger.Warn("Retrieved unhealthy connection from pool")
			pooledConn.Close()
			p.stats.decrementIdleConns()
			p.stats.decrementTotalConns()

			newConn, err := p.createPooledConnection()
			if err != nil {
				p.stats.incrementFailedRequests()
				return nil, fmt.Errorf("failed to create replacement connection: %w", err)
			}
			pooledConn = newConn
			p.stats.incrementTotalConns()
		}

		pooledConn.mu.Lock()
		pooledConn.lastUsedAt = time.Now()
		pooledConn.usageCount++
		pooledConn.mu.Unlock()

		p.stats.decrementIdleConns()
		p.stats.incrementActiveConns()

		return pooledConn.conn, nil

	case <-time.After(p.config.ConnectionTimeout):
		p.stats.incrementFailedRequests()
		return nil, ErrPoolExhausted
	}
}

// Put returns a connection to the pool
func (p *connectionPool) Put(conn *zk.Conn) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return
	}
	p.mu.RUnlock()

	select {
	case p.connections <- &pooledConnection{
		conn:       conn,
		lastUsedAt: time.Now(),
		isHealthy:  conn.State() == zk.StateHasSession,
	}:
		p.stats.incrementIdleConns()
		p.stats.decrementActiveConns()
	default:
		p.logger.Debug("Connection pool is full, closing excess connection")
		conn.Close()
	}
}

// monitorConnection monitors a pooled connection for events
func (p *connectionPool) monitorConnection(pooledConn *pooledConnection) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case event, ok := <-pooledConn.eventChan:
			if !ok {
				p.logger.Warn("Event channel closed for pooled connection")
				pooledConn.markUnhealthy()
				return
			}

			switch event.State {
			case zk.StateDisconnected, zk.StateExpired:
				p.logger.Warn("Pooled connection lost: %v", event.State)
				pooledConn.markUnhealthy()
			case zk.StateHasSession:
				pooledConn.markHealthy()
			}
		}
	}
}

// maintenanceLoop performs periodic maintenance on the pool
func (p *connectionPool) maintenanceLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.performMaintenance()
		}
	}
}

// performMaintenance performs maintenance tasks on the pool
func (p *connectionPool) performMaintenance() {
	p.logger.Debug("Performing pool maintenance")
	
	poolSize := len(p.connections)
	
	for i := 0; i < poolSize; i++ {
		select {
		case pooledConn := <-p.connections:
			pooledConn.mu.RLock()
			healthy := pooledConn.isHealthy
			pooledConn.mu.RUnlock()

			if !healthy || pooledConn.conn.State() != zk.StateHasSession {
				p.logger.Warn("Unhealthy connection detected during maintenance")
				pooledConn.Close()
				p.stats.decrementIdleConns()
				p.stats.decrementTotalConns()
				p.stats.incrementHealthCheckFail()

				newConn, err := p.createPooledConnection()
				if err != nil {
					p.logger.Error("Failed to create replacement connection: %v", err)
					continue
				}
				p.connections <- newConn
				p.stats.incrementTotalConns()
				p.stats.incrementIdleConns()
			} else {
				p.connections <- pooledConn
			}

		default:
			return
		}
	}
}

// Close closes all connections in the pool
func (p *connectionPool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()

	p.logger.Info("Closing connection pool")

	p.cancel()
	p.wg.Wait()

	close(p.connections)
	for pooledConn := range p.connections {
		pooledConn.Close()
	}

	p.logger.Info("Connection pool closed")
}

// GetStats returns pool statistics
func (p *connectionPool) GetStats() PoolStats {
	return PoolStats{
		ActiveConns:     p.stats.getActiveConns(),
		IdleConns:       p.stats.getIdleConns(),
		TotalConns:      p.stats.getTotalConns(),
		TotalRequests:   p.stats.getTotalRequests(),
		FailedRequests:  p.stats.getFailedRequests(),
		Reconnections:   p.stats.getReconnections(),
		HealthCheckFail: p.stats.getHealthCheckFail(),
	}
}

// PoolStats represents connection pool statistics
type PoolStats struct {
	ActiveConns     int
	IdleConns       int
	TotalConns      int
	TotalRequests   int64
	FailedRequests  int64
	Reconnections   int64
	HealthCheckFail int64
}

// pooledConnection helper methods
func (pc *pooledConnection) Close() {
	if pc.conn != nil {
		pc.conn.Close()
	}
}

func (pc *pooledConnection) markHealthy() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.isHealthy = true
}

func (pc *pooledConnection) markUnhealthy() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.isHealthy = false
}

// poolStats helper methods
func (ps *poolStats) incrementActiveConns() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.activeConns++
}

func (ps *poolStats) decrementActiveConns() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.activeConns--
}

func (ps *poolStats) incrementIdleConns() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.idleConns++
}

func (ps *poolStats) decrementIdleConns() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.idleConns--
}

func (ps *poolStats) incrementTotalConns() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.totalConns++
}

func (ps *poolStats) decrementTotalConns() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.totalConns--
}

func (ps *poolStats) incrementTotalRequests() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.totalRequests++
}

func (ps *poolStats) incrementFailedRequests() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.failedRequests++
}

func (ps *poolStats) incrementReconnections() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.reconnections++
}

func (ps *poolStats) incrementHealthCheckFail() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.healthCheckFail++
}

func (ps *poolStats) getActiveConns() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.activeConns
}

func (ps *poolStats) getIdleConns() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.idleConns
}

func (ps *poolStats) getTotalConns() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.totalConns
}

func (ps *poolStats) getTotalRequests() int64 {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.totalRequests
}

func (ps *poolStats) getFailedRequests() int64 {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.failedRequests
}

func (ps *poolStats) getReconnections() int64 {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.reconnections
}

func (ps *poolStats) getHealthCheckFail() int64 {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.healthCheckFail
}
