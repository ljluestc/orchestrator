package zkclient

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/go-zookeeper/zk"
)

// CreateMode represents the mode for creating a znode
type CreateMode int32

const (
	// ModePersistent creates a persistent znode
	ModePersistent CreateMode = 0
	// ModeEphemeral creates an ephemeral znode
	ModeEphemeral CreateMode = zk.FlagEphemeral
	// ModePersistentSequential creates a persistent sequential znode
	ModePersistentSequential CreateMode = zk.FlagSequence
	// ModeEphemeralSequential creates an ephemeral sequential znode
	ModeEphemeralSequential CreateMode = zk.FlagEphemeral | zk.FlagSequence
)

// Get retrieves data from a znode
func (c *Client) Get(ctx context.Context, path string) ([]byte, *zk.Stat, error) {
	if err := validatePath(path); err != nil {
		return nil, nil, err
	}

	if c.conn == nil {
		return nil, nil, ErrNotConnected
	}

	var data []byte
	var stat *zk.Stat
	var err error

	err = c.WithRetry(ctx, func() error {
		data, stat, err = c.conn.Get(path)
		return err
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get znode %s: %w", path, err)
	}

	return data, stat, nil
}

// Set updates data in a znode
func (c *Client) Set(ctx context.Context, path string, data []byte, version int32) (*zk.Stat, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}

	if c.conn == nil {
		return nil, ErrNotConnected
	}

	var stat *zk.Stat
	var err error

	err = c.WithRetry(ctx, func() error {
		stat, err = c.conn.Set(path, data, version)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to set znode %s: %w", path, err)
	}

	return stat, nil
}

// Create creates a new znode
func (c *Client) Create(ctx context.Context, path string, data []byte, mode CreateMode, acl []zk.ACL) (string, error) {
	if err := validatePath(path); err != nil {
		return "", err
	}

	if c.conn == nil {
		return "", ErrNotConnected
	}

	if acl == nil {
		acl = zk.WorldACL(zk.PermAll)
	}

	var createdPath string
	var err error

	err = c.WithRetry(ctx, func() error {
		createdPath, err = c.conn.Create(path, data, int32(mode), acl)
		return err
	})

	if err != nil {
		return "", fmt.Errorf("failed to create znode %s: %w", path, err)
	}

	return createdPath, nil
}

// CreateRecursive creates a znode and all parent znodes if they don't exist
func (c *Client) CreateRecursive(ctx context.Context, path string, data []byte, mode CreateMode, acl []zk.ACL) (string, error) {
	if err := validatePath(path); err != nil {
		return "", err
	}

	if acl == nil {
		acl = zk.WorldACL(zk.PermAll)
	}

	// Create parent directories first
	parts := strings.Split(strings.Trim(path, "/"), "/")
	currentPath := ""

	for i := 0; i < len(parts)-1; i++ {
		currentPath += "/" + parts[i]

		exists, err := c.Exists(ctx, currentPath)
		if err != nil {
			return "", fmt.Errorf("failed to check existence of %s: %w", currentPath, err)
		}

		if !exists {
			_, err = c.Create(ctx, currentPath, []byte{}, ModePersistent, acl)
			if err != nil && err != zk.ErrNodeExists {
				return "", fmt.Errorf("failed to create parent %s: %w", currentPath, err)
			}
		}
	}

	// Create the final node
	return c.Create(ctx, path, data, mode, acl)
}

// Delete deletes a znode
func (c *Client) Delete(ctx context.Context, path string, version int32) error {
	if err := validatePath(path); err != nil {
		return err
	}

	if c.conn == nil {
		return ErrNotConnected
	}

	err := c.WithRetry(ctx, func() error {
		return c.conn.Delete(path, version)
	})

	if err != nil {
		return fmt.Errorf("failed to delete znode %s: %w", path, err)
	}

	return nil
}

// DeleteRecursive deletes a znode and all its children
func (c *Client) DeleteRecursive(ctx context.Context, path string) error {
	if err := validatePath(path); err != nil {
		return err
	}

	// Get children first
	children, err := c.Children(ctx, path)
	if err != nil && err != zk.ErrNoNode {
		return fmt.Errorf("failed to get children of %s: %w", path, err)
	}

	// Delete all children recursively
	for _, child := range children {
		childPath := path + "/" + child
		if err := c.DeleteRecursive(ctx, childPath); err != nil {
			return err
		}
	}

	// Delete the node itself
	return c.Delete(ctx, path, -1)
}

// Exists checks if a znode exists
func (c *Client) Exists(ctx context.Context, path string) (bool, error) {
	if err := validatePath(path); err != nil {
		return false, err
	}

	if c.conn == nil {
		return false, ErrNotConnected
	}

	var exists bool
	var err error

	err = c.WithRetry(ctx, func() error {
		exists, _, err = c.conn.Exists(path)
		return err
	})

	if err != nil {
		return false, fmt.Errorf("failed to check existence of znode %s: %w", path, err)
	}

	return exists, nil
}

// Children returns the list of children for a znode
func (c *Client) Children(ctx context.Context, path string) ([]string, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}

	if c.conn == nil {
		return nil, ErrNotConnected
	}

	var children []string
	var err error

	err = c.WithRetry(ctx, func() error {
		children, _, err = c.conn.Children(path)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get children of znode %s: %w", path, err)
	}

	return children, nil
}

// GetACL retrieves the ACL for a znode
func (c *Client) GetACL(ctx context.Context, path string) ([]zk.ACL, *zk.Stat, error) {
	if err := validatePath(path); err != nil {
		return nil, nil, err
	}

	if c.conn == nil {
		return nil, nil, ErrNotConnected
	}

	var acl []zk.ACL
	var stat *zk.Stat
	var err error

	err = c.WithRetry(ctx, func() error {
		acl, stat, err = c.conn.GetACL(path)
		return err
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get ACL for znode %s: %w", path, err)
	}

	return acl, stat, nil
}

// SetACL sets the ACL for a znode
func (c *Client) SetACL(ctx context.Context, path string, acl []zk.ACL, version int32) (*zk.Stat, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}

	if c.conn == nil {
		return nil, ErrNotConnected
	}

	var stat *zk.Stat
	var err error

	err = c.WithRetry(ctx, func() error {
		stat, err = c.conn.SetACL(path, acl, version)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to set ACL for znode %s: %w", path, err)
	}

	return stat, nil
}

// Watch sets a watch on a znode and returns data and a channel for events
func (c *Client) Watch(ctx context.Context, path string) ([]byte, *zk.Stat, <-chan zk.Event, error) {
	if err := validatePath(path); err != nil {
		return nil, nil, nil, err
	}

	if c.conn == nil {
		return nil, nil, nil, ErrNotConnected
	}

	data, stat, eventChan, err := c.conn.GetW(path)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to watch znode %s: %w", path, err)
	}

	return data, stat, eventChan, nil
}

// WatchChildren sets a watch on a znode's children and returns a channel for events
func (c *Client) WatchChildren(ctx context.Context, path string) ([]string, *zk.Stat, <-chan zk.Event, error) {
	if err := validatePath(path); err != nil {
		return nil, nil, nil, err
	}

	if c.conn == nil {
		return nil, nil, nil, ErrNotConnected
	}

	children, stat, eventChan, err := c.conn.ChildrenW(path)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to watch children of znode %s: %w", path, err)
	}

	return children, stat, eventChan, nil
}

// ExistsWatch checks if a znode exists and sets a watch
func (c *Client) ExistsWatch(ctx context.Context, path string) (bool, *zk.Stat, <-chan zk.Event, error) {
	if err := validatePath(path); err != nil {
		return false, nil, nil, err
	}

	if c.conn == nil {
		return false, nil, nil, ErrNotConnected
	}

	exists, stat, eventChan, err := c.conn.ExistsW(path)
	if err != nil {
		return false, nil, nil, fmt.Errorf("failed to watch existence of znode %s: %w", path, err)
	}

	return exists, stat, eventChan, nil
}

// Multi executes multiple operations atomically
func (c *Client) Multi(ctx context.Context, ops ...interface{}) ([]zk.MultiResponse, error) {
	if len(ops) == 0 {
		return nil, fmt.Errorf("no operations provided")
	}

	if c.conn == nil {
		return nil, ErrNotConnected
	}

	var responses []zk.MultiResponse
	var err error

	err = c.WithRetry(ctx, func() error {
		responses, err = c.conn.Multi(ops...)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to execute multi operation: %w", err)
	}

	return responses, nil
}

// Sync forces the znode to be synchronized across the cluster
func (c *Client) Sync(ctx context.Context, path string) (string, error) {
	if err := validatePath(path); err != nil {
		return "", err
	}

	if c.conn == nil {
		return "", ErrNotConnected
	}

	var syncPath string
	var err error

	err = c.WithRetry(ctx, func() error {
		syncPath, err = c.conn.Sync(path)
		return err
	})

	if err != nil {
		return "", fmt.Errorf("failed to sync znode %s: %w", path, err)
	}

	return syncPath, nil
}

// validatePath validates a Zookeeper path
func validatePath(p string) error {
	if p == "" {
		return ErrInvalidPath
	}

	if p[0] != '/' {
		return fmt.Errorf("%w: path must start with /", ErrInvalidPath)
	}

	if len(p) > 1 && p[len(p)-1] == '/' {
		return fmt.Errorf("%w: path must not end with /", ErrInvalidPath)
	}

	// Check for invalid characters
	if strings.Contains(p, "//") {
		return fmt.Errorf("%w: path contains //", ErrInvalidPath)
	}

	return nil
}

// NormalizePath normalizes a Zookeeper path
func NormalizePath(p string) string {
	if p == "" {
		return "/"
	}

	// Clean the path
	cleaned := path.Clean(p)

	// Ensure it starts with /
	if cleaned[0] != '/' {
		cleaned = "/" + cleaned
	}

	return cleaned
}

// JoinPath joins multiple path segments into a valid Zookeeper path
func JoinPath(segments ...string) string {
	if len(segments) == 0 {
		return "/"
	}

	result := ""
	for _, segment := range segments {
		segment = strings.Trim(segment, "/")
		if segment != "" {
			result += "/" + segment
		}
	}

	if result == "" {
		return "/"
	}

	return result
}
