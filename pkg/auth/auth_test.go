package auth

import (
	"testing"
	"time"

	"github.com/kim-truong/openshift-auth-plugin/pkg/config"
	"github.com/kim-truong/openshift-auth-plugin/pkg/storage"
)

// MockStorage implements the Storage interface for testing
type MockStorage struct {
	token  *storage.TokenData
	stored bool
}

func (m *MockStorage) GetToken() (*storage.TokenData, error) {
	return m.token, nil
}

func (m *MockStorage) StoreToken(token string, expiry time.Time) error {
	m.token = &storage.TokenData{
		Token:               token,
		ExpirationTimestamp: expiry,
	}
	m.stored = true
	return nil
}

func TestAuthenticator_tryStoredToken_NoToken(t *testing.T) {
	cfg := &config.Config{
		ServerURL:   "https://api.test.example.com:6443",
		ClusterName: "test-cluster",
		VerifySSL:   false,
		Debug:       false,
	}

	mockStorage := &MockStorage{token: nil}
	logger := config.NewLogger(false)

	auth := NewAuthenticator(cfg, mockStorage, logger)

	_, _, err := auth.tryStoredToken()
	if err == nil {
		t.Error("Expected error when no token is stored, got nil")
	}
}

func TestAuthenticator_tryStoredToken_ExpiredToken(t *testing.T) {
	cfg := &config.Config{
		ServerURL:   "https://api.test.example.com:6443",
		ClusterName: "test-cluster",
		VerifySSL:   false,
		Debug:       false,
	}

	// Create an expired token (1 hour ago)
	expiredTime := time.Now().Add(-1 * time.Hour)
	mockStorage := &MockStorage{
		token: &storage.TokenData{
			Token:               "expired-token",
			ExpirationTimestamp: expiredTime,
		},
	}
	logger := config.NewLogger(false)

	auth := NewAuthenticator(cfg, mockStorage, logger)

	_, _, err := auth.tryStoredToken()
	if err == nil {
		t.Error("Expected error for expired token, got nil")
	}
}

func TestMockStorage_StoreToken(t *testing.T) {
	mockStorage := &MockStorage{}

	token := "test-token"
	expiry := time.Now().Add(24 * time.Hour)

	err := mockStorage.StoreToken(token, expiry)
	if err != nil {
		t.Errorf("StoreToken failed: %v", err)
	}

	if !mockStorage.stored {
		t.Error("Token was not marked as stored")
	}

	if mockStorage.token.Token != token {
		t.Errorf("Expected token %s, got %s", token, mockStorage.token.Token)
	}
}

func TestMockStorage_GetToken(t *testing.T) {
	expectedToken := "test-token"
	expectedExpiry := time.Now().Add(24 * time.Hour)

	mockStorage := &MockStorage{
		token: &storage.TokenData{
			Token:               expectedToken,
			ExpirationTimestamp: expectedExpiry,
		},
	}

	tokenData, err := mockStorage.GetToken()
	if err != nil {
		t.Errorf("GetToken failed: %v", err)
	}

	if tokenData.Token != expectedToken {
		t.Errorf("Expected token %s, got %s", expectedToken, tokenData.Token)
	}

	if !tokenData.ExpirationTimestamp.Equal(expectedExpiry) {
		t.Errorf("Expected expiry %v, got %v", expectedExpiry, tokenData.ExpirationTimestamp)
	}
}
