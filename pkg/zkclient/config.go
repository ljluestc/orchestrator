package zkclient

import (
	"time"
)

// AuthType represents the authentication type for Zookeeper
type AuthType string

const (
	// AuthTypeNone indicates no authentication
	AuthTypeNone AuthType = "none"
	// AuthTypeSASL indicates SASL authentication
	AuthTypeSASL AuthType = "sasl"
	// AuthTypeDigest indicates digest authentication
	AuthTypeDigest AuthType = "digest"
)

// Config contains configuration for Zookeeper client
type Config struct {
	// Servers is a list of Zookeeper server endpoints (host:port)
	Servers []string

	// SessionTimeout is the session timeout for Zookeeper connections
	SessionTimeout time.Duration

	// ConnectionTimeout is the timeout for establishing connections
	ConnectionTimeout time.Duration

	// AuthType specifies the authentication method
	AuthType AuthType

	// AuthData contains the authentication credentials
	// For Digest: "username:password"
	// For SASL: username
	AuthData string

	// AuthPassword is used for SASL authentication
	AuthPassword string

	// TLSEnabled indicates whether to use TLS for connections
	TLSEnabled bool

	// TLSConfig contains TLS configuration
	TLSConfig *TLSConfig

	// RetryAttempts is the number of retry attempts for failed operations
	RetryAttempts int

	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration

	// RetryMaxDelay is the maximum delay between retry attempts (for exponential backoff)
	RetryMaxDelay time.Duration

	// EnableBackoff enables exponential backoff for retries
	EnableBackoff bool

	// ReconnectDelay is the delay before attempting to reconnect after connection loss
	ReconnectDelay time.Duration

	// MaxReconnectAttempts is the maximum number of reconnection attempts (0 = unlimited)
	MaxReconnectAttempts int

	// HealthCheckInterval is the interval for health checks
	HealthCheckInterval time.Duration

	// EnableConnectionPool enables connection pooling
	EnableConnectionPool bool

	// PoolSize is the size of the connection pool (if enabled)
	PoolSize int

	// Logger allows custom logging
	Logger Logger
}

// TLSConfig contains TLS configuration for Zookeeper connections
type TLSConfig struct {
	// CertFile is the path to the client certificate file
	CertFile string

	// KeyFile is the path to the client key file
	KeyFile string

	// CAFile is the path to the CA certificate file
	CAFile string

	// ServerName is the server name for TLS verification
	ServerName string

	// InsecureSkipVerify skips TLS verification (not recommended for production)
	InsecureSkipVerify bool
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		SessionTimeout:       30 * time.Second,
		ConnectionTimeout:    10 * time.Second,
		AuthType:             AuthTypeNone,
		TLSEnabled:           false,
		RetryAttempts:        3,
		RetryDelay:           1 * time.Second,
		RetryMaxDelay:        30 * time.Second,
		EnableBackoff:        true,
		ReconnectDelay:       5 * time.Second,
		MaxReconnectAttempts: 10,
		HealthCheckInterval:  30 * time.Second,
		EnableConnectionPool: false,
		PoolSize:             5,
		Logger:               &defaultLogger{},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if len(c.Servers) == 0 {
		return ErrNoServers
	}

	if c.SessionTimeout <= 0 {
		return ErrInvalidSessionTimeout
	}

	if c.ConnectionTimeout <= 0 {
		return ErrInvalidConnectionTimeout
	}

	if c.EnableConnectionPool && c.PoolSize <= 0 {
		return ErrInvalidPoolSize
	}

	if c.TLSEnabled && c.TLSConfig == nil {
		return ErrMissingTLSConfig
	}

	if c.AuthType == AuthTypeSASL && (c.AuthData == "" || c.AuthPassword == "") {
		return ErrMissingAuthCredentials
	}

	if c.AuthType == AuthTypeDigest && c.AuthData == "" {
		return ErrMissingAuthCredentials
	}

	return nil
}

// Clone creates a deep copy of the configuration
func (c *Config) Clone() *Config {
	clone := *c
	clone.Servers = make([]string, len(c.Servers))
	copy(clone.Servers, c.Servers)

	if c.TLSConfig != nil {
		tlsConfig := *c.TLSConfig
		clone.TLSConfig = &tlsConfig
	}

	return &clone
}
