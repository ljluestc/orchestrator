package zkclient

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/go-zookeeper/zk"
)

// RetryConfig contains configuration for retry behavior
type RetryConfig struct {
	MaxAttempts int
	InitialDelay time.Duration
	MaxDelay time.Duration
	Multiplier float64
	EnableBackoff bool
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		Multiplier:    2.0,
		EnableBackoff: true,
	}
}

// RetryableOperation is a function that can be retried
type RetryableOperation func() error

// RetryWithBackoff executes an operation with exponential backoff retry logic
func RetryWithBackoff(ctx context.Context, config *RetryConfig, op RetryableOperation) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Execute the operation
		err := op()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return &RetryError{
				Op:       "operation",
				Attempts: attempt,
				Err:      err,
			}
		}

		// If this was the last attempt, don't wait
		if attempt == config.MaxAttempts {
			break
		}

		// Check context before sleeping
		select {
		case <-ctx.Done():
			return &RetryError{
				Op:       "operation",
				Attempts: attempt,
				Err:      ctx.Err(),
			}
		default:
		}

		// Calculate delay with exponential backoff
		if config.EnableBackoff {
			delay = calculateBackoffDelay(config.InitialDelay, config.MaxDelay, config.Multiplier, attempt)
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return &RetryError{
				Op:       "operation",
				Attempts: attempt,
				Err:      ctx.Err(),
			}
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return &RetryError{
		Op:       "operation",
		Attempts: config.MaxAttempts,
		Err:      lastErr,
	}
}

// calculateBackoffDelay calculates the delay for exponential backoff
func calculateBackoffDelay(initialDelay, maxDelay time.Duration, multiplier float64, attempt int) time.Duration {
	// Calculate exponential backoff: initialDelay * (multiplier ^ (attempt - 1))
	backoff := float64(initialDelay) * math.Pow(multiplier, float64(attempt-1))

	// Cap at max delay
	if backoff > float64(maxDelay) {
		return maxDelay
	}

	return time.Duration(backoff)
}

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for Zookeeper-specific errors
	switch err {
	case zk.ErrConnectionClosed:
		return true
	case zk.ErrSessionExpired:
		return true
	case zk.ErrSessionMoved:
		return true
	case zk.ErrNoServer:
		return true
	}

	// Check for custom error types
	switch err.(type) {
	case *ConnectionError:
		return true
	}

	// Non-retryable Zookeeper errors
	switch err {
	case zk.ErrNoNode:
		return false
	case zk.ErrNodeExists:
		return false
	case zk.ErrNotEmpty:
		return false
	case zk.ErrNoAuth:
		return false
	case zk.ErrBadVersion:
		return false
	case zk.ErrBadArguments:
		return false
	case zk.ErrInvalidACL:
		return false
	case zk.ErrAuthFailed:
		return false
	}

	return false
}

// attemptReconnect attempts to reconnect to Zookeeper with retry logic
func (c *Client) attemptReconnect(ctx context.Context) error {
	c.stateMu.Lock()
	if c.state == StateClosed {
		c.stateMu.Unlock()
		return ErrConnectionClosed
	}
	c.state = StateReconnecting
	c.stateMu.Unlock()

	c.logger.Info("Attempting to reconnect to Zookeeper")

	retryConfig := &RetryConfig{
		MaxAttempts:   c.config.MaxReconnectAttempts,
		InitialDelay:  c.config.ReconnectDelay,
		MaxDelay:      c.config.RetryMaxDelay,
		Multiplier:    2.0,
		EnableBackoff: c.config.EnableBackoff,
	}

	// If MaxReconnectAttempts is 0, retry indefinitely
	if retryConfig.MaxAttempts == 0 {
		retryConfig.MaxAttempts = math.MaxInt32
	}

	err := RetryWithBackoff(ctx, retryConfig, func() error {
		c.stateMu.RLock()
		if c.state == StateClosed {
			c.stateMu.RUnlock()
			return fmt.Errorf("client closed during reconnection")
		}
		c.stateMu.RUnlock()

		c.logger.Debug("Reconnection attempt")

		// Close existing connection
		if c.conn != nil {
			c.conn.Close()
		}

		// Create new connection
		return c.Connect(ctx)
	})

	if err != nil {
		c.logger.Error("Failed to reconnect after all attempts: %v", err)
		c.setState(StateDisconnected)
		return err
	}

	return nil
}

// WithRetry wraps a Zookeeper operation with retry logic
func (c *Client) WithRetry(ctx context.Context, op RetryableOperation) error {
	retryConfig := &RetryConfig{
		MaxAttempts:   c.config.RetryAttempts,
		InitialDelay:  c.config.RetryDelay,
		MaxDelay:      c.config.RetryMaxDelay,
		Multiplier:    2.0,
		EnableBackoff: c.config.EnableBackoff,
	}

	return RetryWithBackoff(ctx, retryConfig, op)
}
