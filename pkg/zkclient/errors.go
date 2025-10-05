package zkclient

import (
	"errors"
	"fmt"
)

// Common errors
var (
	ErrNoServers                = errors.New("no servers specified")
	ErrInvalidSessionTimeout    = errors.New("invalid session timeout")
	ErrInvalidConnectionTimeout = errors.New("invalid connection timeout")
	ErrInvalidPoolSize          = errors.New("invalid pool size")
	ErrMissingTLSConfig         = errors.New("TLS enabled but no TLS config provided")
	ErrMissingAuthCredentials   = errors.New("missing authentication credentials")
	ErrNotConnected             = errors.New("not connected to Zookeeper")
	ErrConnectionClosed         = errors.New("connection is closed")
	ErrMaxReconnectAttempts     = errors.New("maximum reconnection attempts reached")
	ErrPoolExhausted            = errors.New("connection pool exhausted")
	ErrInvalidPath              = errors.New("invalid znode path")
	ErrHealthCheckFailed        = errors.New("health check failed")
	ErrQuorumLost               = errors.New("Zookeeper quorum lost")
)

// ConnectionError represents a connection-related error
type ConnectionError struct {
	Op      string
	Servers []string
	Err     error
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("connection error during %s (servers: %v): %v", e.Op, e.Servers, e.Err)
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// RetryError represents an error after exhausting all retries
type RetryError struct {
	Op       string
	Attempts int
	Err      error
}

func (e *RetryError) Error() string {
	return fmt.Sprintf("operation %s failed after %d attempts: %v", e.Op, e.Attempts, e.Err)
}

func (e *RetryError) Unwrap() error {
	return e.Err
}

// AuthenticationError represents an authentication failure
type AuthenticationError struct {
	AuthType AuthType
	Err      error
}

func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("authentication failed (%s): %v", e.AuthType, e.Err)
}

func (e *AuthenticationError) Unwrap() error {
	return e.Err
}

// HealthCheckError represents a health check failure
type HealthCheckError struct {
	Check  string
	Reason string
	Err    error
}

func (e *HealthCheckError) Error() string {
	return fmt.Sprintf("health check %s failed: %s: %v", e.Check, e.Reason, e.Err)
}

func (e *HealthCheckError) Unwrap() error {
	return e.Err
}
