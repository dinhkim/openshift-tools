package storage

import (
	"fmt"
)

// Storage defines the interface for credential storage backends
type Storage interface {
	// Get retrieves a value from storage
	Get(cluster, key string) (string, error)

	// Store saves a value to storage
	Store(cluster, key, value string) error
}

// New creates a new storage backend based on the provided type
func New(storageType string) (Storage, error) {
	switch storageType {
	case "keychain":
		return NewKeychainStorage(), nil
	case "gopass":
		return NewGopassStorage(), nil
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}
}
