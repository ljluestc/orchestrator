package storage

import (
	"context"
	"errors"
	"sync"
)

// Storage represents a simple in-memory storage
type Storage struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

// NewStorage creates a new storage instance
func NewStorage() *Storage {
	return &Storage{
		data: make(map[string]interface{}),
	}
}

// Store stores a value with the given key
func (s *Storage) Store(key string, value interface{}) error {
	if key == "" {
		return errors.New("key cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data == nil {
		return errors.New("storage is closed")
	}

	s.data[key] = value
	return nil
}

// StoreWithContext stores a value with the given key using context
func (s *Storage) StoreWithContext(ctx context.Context, key string, value interface{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return s.Store(key, value)
	}
}

// Get retrieves a value by key
func (s *Storage) Get(key string) (interface{}, error) {
	if key == "" {
		return nil, errors.New("key cannot be empty")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.data == nil {
		return nil, errors.New("storage is closed")
	}

	value, exists := s.data[key]
	if !exists {
		return nil, errors.New("key not found")
	}

	return value, nil
}

// GetWithContext retrieves a value by key using context
func (s *Storage) GetWithContext(ctx context.Context, key string) (interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return s.Get(key)
	}
}

// Delete removes a value by key
func (s *Storage) Delete(key string) error {
	if key == "" {
		return errors.New("key cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data[key]; !exists {
		return errors.New("key not found")
	}

	delete(s.data, key)
	return nil
}

// List returns all keys in the storage
func (s *Storage) List() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.data))
	for key := range s.data {
		keys = append(keys, key)
	}

	return keys, nil
}

// Exists checks if a key exists in the storage
func (s *Storage) Exists(key string) (bool, error) {
	if key == "" {
		return false, errors.New("key cannot be empty")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.data[key]
	return exists, nil
}

// Clear removes all data from the storage
func (s *Storage) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[string]interface{})
	return nil
}

// Size returns the number of items in the storage
func (s *Storage) Size() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.data), nil
}

// Close closes the storage
func (s *Storage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = nil
	return nil
}
