package storage

import (
	"fmt"
	"os/exec"
	"strings"
)

const (
	keychainAccount = "openshift-auth-plugin"
)

// KeychainStorage implements Storage using macOS Keychain
type KeychainStorage struct{}

// NewKeychainStorage creates a new KeychainStorage instance
func NewKeychainStorage() *KeychainStorage {
	return &KeychainStorage{}
}

// Get retrieves a value from the macOS Keychain
func (k *KeychainStorage) Get(cluster, key string) (string, error) {
	service := fmt.Sprintf("%s-%s", cluster, key)

	cmd := exec.Command("security", "find-generic-password",
		"-a", keychainAccount,
		"-s", service,
		"-w")

	output, err := cmd.Output()
	if err != nil {
		// Not found or other error - return empty string, no error
		// This matches the shell script behavior (returns empty on error)
		return "", nil
	}

	return strings.TrimSpace(string(output)), nil
}

// Store saves a value to the macOS Keychain
func (k *KeychainStorage) Store(cluster, key, value string) error {
	service := fmt.Sprintf("%s-%s", cluster, key)
	label := fmt.Sprintf("%s %s", cluster, key)

	// The -U flag allows updating an existing item
	cmd := exec.Command("security", "add-generic-password",
		"-a", keychainAccount,
		"-s", service,
		"-l", label,
		"-w", value,
		"-U")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to store in keychain: %w", err)
	}

	return nil
}
