package storage

import (
	"fmt"
	"sync"
)

// MockStorage is a mock implementation of Storage for testing
type MockStorage struct {
	mu         sync.RWMutex
	data       map[string]string
	getCalls   int
	storeCalls int
}

// NewMockStorage creates a new MockStorage instance
func NewMockStorage() *MockStorage {
	return &MockStorage{
		data: make(map[string]string),
	}
}

// Get retrieves a value from the mock storage
func (m *MockStorage) Get(cluster, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.getCalls++

	storageKey := fmt.Sprintf("%s-%s", cluster, key)
	value, exists := m.data[storageKey]
	if !exists {
		return "", nil
	}
	return value, nil
}

// Store saves a value to the mock storage
func (m *MockStorage) Store(cluster, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.storeCalls++

	storageKey := fmt.Sprintf("%s-%s", cluster, key)
	m.data[storageKey] = value
	return nil
}

// GetCallCount returns the number of times Get was called
func (m *MockStorage) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.getCalls
}

// StoreCallCount returns the number of times Store was called
func (m *MockStorage) StoreCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.storeCalls
}

// Clear clears all stored data
func (m *MockStorage) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]string)
	m.getCalls = 0
	m.storeCalls = 0
}
