package storage

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/kim-truong/openshift-auth-plugin/pkg/config"
)

// KeychainStorage implements Storage using macOS Keychain
type KeychainStorage struct {
	clusterName string
	logger      *config.Logger
}

// NewKeychainStorage creates a new keychain storage backend
func NewKeychainStorage(clusterName string, logger *config.Logger) (*KeychainStorage, error) {
	return &KeychainStorage{
		clusterName: clusterName,
		logger:      logger,
	}, nil
}

// GetToken retrieves the token from keychain
func (k *KeychainStorage) GetToken() (*TokenData, error) {
	k.logger.Debug("Retrieving token from keychain")

	serviceName := fmt.Sprintf("%s-token", k.clusterName)
	cmd := exec.Command("security", "find-generic-password",
		"-a", "openshift-auth-plugin",
		"-s", serviceName,
		"-w")

	output, err := cmd.Output()
	if err != nil {
		k.logger.Debug("No token found in keychain")
		return nil, nil
	}

	data := strings.TrimSpace(string(output))
	if data == "" {
		return nil, nil
	}

	tokenData, err := unmarshalTokenData(data)
	if err != nil {
		k.logger.Debug("Failed to parse token data: %v", err)
		return nil, nil
	}

	return tokenData, nil
}

// StoreToken stores the token in keychain
func (k *KeychainStorage) StoreToken(token string, expiry time.Time) error {
	k.logger.Debug("Storing token in keychain")

	data, err := marshalTokenData(token, expiry)
	if err != nil {
		return fmt.Errorf("failed to marshal token data: %w", err)
	}

	serviceName := fmt.Sprintf("%s-token", k.clusterName)
	label := fmt.Sprintf("%s token", k.clusterName)

	cmd := exec.Command("security", "add-generic-password",
		"-a", "openshift-auth-plugin",
		"-s", serviceName,
		"-l", label,
		"-w", data,
		"-U")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to store token in keychain: %w", err)
	}

	k.logger.Debug("Token stored successfully")
	return nil
}
