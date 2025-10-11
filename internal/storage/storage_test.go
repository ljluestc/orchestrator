package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStorage(t *testing.T) {
	// Test creating a new storage instance
	storage := NewStorage()
	assert.NotNil(t, storage)
}

func TestStorage_Store(t *testing.T) {
	// Test storing data
	storage := NewStorage()

	// Test storing valid data
	err := storage.Store("key1", "value1")
	assert.NoError(t, err)

	// Test storing empty key
	err = storage.Store("", "value2")
	assert.Error(t, err)

	// Test storing nil value
	err = storage.Store("key2", nil)
	assert.NoError(t, err)
}

func TestStorage_Get(t *testing.T) {
	// Test retrieving data
	storage := NewStorage()

	// Store some data first
	err := storage.Store("key1", "value1")
	require.NoError(t, err)

	// Test retrieving existing data
	value, err := storage.Get("key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", value)

	// Test retrieving non-existent data
	value, err = storage.Get("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, value)

	// Test retrieving empty key
	value, err = storage.Get("")
	assert.Error(t, err)
	assert.Nil(t, value)
}

func TestStorage_Delete(t *testing.T) {
	// Test deleting data
	storage := NewStorage()

	// Store some data first
	err := storage.Store("key1", "value1")
	require.NoError(t, err)

	// Test deleting existing data
	err = storage.Delete("key1")
	assert.NoError(t, err)

	// Verify data is deleted
	value, err := storage.Get("key1")
	assert.Error(t, err)
	assert.Nil(t, value)

	// Test deleting non-existent data
	err = storage.Delete("nonexistent")
	assert.Error(t, err)

	// Test deleting empty key
	err = storage.Delete("")
	assert.Error(t, err)
}

func TestStorage_List(t *testing.T) {
	// Test listing all data
	storage := NewStorage()

	// Store some data
	err := storage.Store("key1", "value1")
	require.NoError(t, err)
	err = storage.Store("key2", "value2")
	require.NoError(t, err)

	// Test listing all data
	keys, err := storage.List()
	assert.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, "key1")
	assert.Contains(t, keys, "key2")
}

func TestStorage_Exists(t *testing.T) {
	// Test checking if key exists
	storage := NewStorage()

	// Store some data
	err := storage.Store("key1", "value1")
	require.NoError(t, err)

	// Test existing key
	exists, err := storage.Exists("key1")
	assert.NoError(t, err)
	assert.True(t, exists)

	// Test non-existent key
	exists, err = storage.Exists("nonexistent")
	assert.NoError(t, err)
	assert.False(t, exists)

	// Test empty key
	exists, err = storage.Exists("")
	assert.Error(t, err)
	assert.False(t, exists)
}

func TestStorage_Clear(t *testing.T) {
	// Test clearing all data
	storage := NewStorage()

	// Store some data
	err := storage.Store("key1", "value1")
	require.NoError(t, err)
	err = storage.Store("key2", "value2")
	require.NoError(t, err)

	// Test clearing all data
	err = storage.Clear()
	assert.NoError(t, err)

	// Verify all data is cleared
	keys, err := storage.List()
	assert.NoError(t, err)
	assert.Len(t, keys, 0)
}

func TestStorage_Size(t *testing.T) {
	// Test getting storage size
	storage := NewStorage()

	// Test empty storage
	size, err := storage.Size()
	assert.NoError(t, err)
	assert.Equal(t, 0, size)

	// Store some data
	err = storage.Store("key1", "value1")
	require.NoError(t, err)

	// Test storage with data
	size, err = storage.Size()
	assert.NoError(t, err)
	assert.Equal(t, 1, size)
}

func TestStorage_Close(t *testing.T) {
	// Test closing storage
	storage := NewStorage()

	// Store some data first
	err := storage.Store("key1", "value1")
	require.NoError(t, err)

	// Test closing storage
	err = storage.Close()
	assert.NoError(t, err)

	// Test operations after closing
	err = storage.Store("key1", "value1")
	assert.Error(t, err)

	_, err = storage.Get("key1")
	assert.Error(t, err)
}

func TestStorage_ContextHandling(t *testing.T) {
	// Test storage with context
	storage := NewStorage()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test storing with context
	err := storage.StoreWithContext(ctx, "key1", "value1")
	assert.NoError(t, err)

	// Test retrieving with context
	value, err := storage.GetWithContext(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", value)

	// Test with cancelled context
	cancel()
	err = storage.StoreWithContext(ctx, "key2", "value2")
	assert.Error(t, err)
}

func TestStorage_ConcurrentAccess(t *testing.T) {
	// Test concurrent access to storage
	storage := NewStorage()

	// Test concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer func() { done <- true }()
			err := storage.Store("key"+string(rune(i)), "value"+string(rune(i)))
			assert.NoError(t, err)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all data was stored
	size, err := storage.Size()
	assert.NoError(t, err)
	assert.Equal(t, 10, size)
}

func TestStorage_ErrorHandling(t *testing.T) {
	// Test error handling
	storage := NewStorage()

	// Test with invalid operations
	err := storage.Store("", "value")
	assert.Error(t, err)

	_, err = storage.Get("")
	assert.Error(t, err)

	err = storage.Delete("")
	assert.Error(t, err)

	_, err = storage.Exists("")
	assert.Error(t, err)
}

func TestStorage_Performance(t *testing.T) {
	// Test storage performance
	storage := NewStorage()

	// Test storing large amount of data
	start := time.Now()
	for i := 0; i < 1000; i++ {
		err := storage.Store("key"+string(rune(i)), "value"+string(rune(i)))
		assert.NoError(t, err)
	}
	duration := time.Since(start)

	// Should complete within reasonable time
	assert.Less(t, duration, 1*time.Second)

	// Test retrieving large amount of data
	start = time.Now()
	for i := 0; i < 1000; i++ {
		_, err := storage.Get("key" + string(rune(i)))
		assert.NoError(t, err)
	}
	duration = time.Since(start)

	// Should complete within reasonable time
	assert.Less(t, duration, 1*time.Second)
}
