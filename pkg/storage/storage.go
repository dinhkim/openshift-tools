package storage

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kim-truong/openshift-auth-plugin/pkg/config"
)

// TokenData represents stored token information
type TokenData struct {
	Token               string    `json:"token"`
	ExpirationTimestamp time.Time `json:"expirationTimestamp"`
}

// Storage interface for credential storage
type Storage interface {
	GetToken() (*TokenData, error)
	StoreToken(token string, expiry time.Time) error
}

// NewStorage creates a new storage backend based on the type
func NewStorage(storeType, clusterName string, logger *config.Logger) (Storage, error) {
	switch storeType {
	case "keychain":
		return NewKeychainStorage(clusterName, logger)
	case "gopass":
		return NewGopassStorage(clusterName, logger)
	default:
		return nil, fmt.Errorf("unsupported secret store: %s", storeType)
	}
}

// marshalTokenData converts token data to JSON string
func marshalTokenData(token string, expiry time.Time) (string, error) {
	data := TokenData{
		Token:               token,
		ExpirationTimestamp: expiry,
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// unmarshalTokenData parses JSON string to token data
func unmarshalTokenData(data string) (*TokenData, error) {
	var tokenData TokenData
	if err := json.Unmarshal([]byte(data), &tokenData); err != nil {
		return nil, err
	}
	return &tokenData, nil
}
