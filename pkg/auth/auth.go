package auth

import (
	"fmt"
	"time"

	"github.com/kim-truong/openshift-auth-plugin/pkg/config"
	"github.com/kim-truong/openshift-auth-plugin/pkg/storage"
)

// Authenticator handles OpenShift authentication
type Authenticator struct {
	config  *config.Config
	storage storage.Storage
	logger  *config.Logger
	client  *OpenshiftClient
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator(cfg *config.Config, store storage.Storage, logger *config.Logger) *Authenticator {
	client := NewOpenshiftClient(cfg.ServerURL, cfg.VerifySSL, logger)
	return &Authenticator{
		config:  cfg,
		storage: store,
		logger:  logger,
		client:  client,
	}
}

// GetToken retrieves a valid token using various authentication methods
func (a *Authenticator) GetToken() (string, time.Time, error) {
	// Method 1: Try cached token
	if token, expiry, err := a.tryStoredToken(); err == nil {
		a.logger.Debug("Using cached token")
		return token, expiry, nil
	}

	// Method 2: Try SSO authentication
	if a.config.SSOEnabled {
		a.logger.Debug("Attempting SSO authentication")
		if token, expiry, err := a.authenticateSSO(); err == nil {
			return token, expiry, nil
		} else {
			a.logger.Debug("SSO authentication failed: %v", err)
		}
	}

	return "", time.Time{}, fmt.Errorf("no valid authentication method found")
}

// tryStoredToken attempts to use a stored token
func (a *Authenticator) tryStoredToken() (string, time.Time, error) {
	tokenData, err := a.storage.GetToken()
	if err != nil {
		return "", time.Time{}, err
	}

	if tokenData == nil {
		return "", time.Time{}, fmt.Errorf("no stored token")
	}

	// Check if token is expired
	if time.Now().After(tokenData.ExpirationTimestamp) {
		a.logger.Debug("Stored token is expired")
		return "", time.Time{}, fmt.Errorf("token expired")
	}

	// Validate token with OpenShift API
	if err := a.client.ValidateToken(tokenData.Token); err != nil {
		a.logger.Debug("Token validation failed: %v", err)
		return "", time.Time{}, err
	}

	return tokenData.Token, tokenData.ExpirationTimestamp, nil
}

// authenticateSSO performs browser-based SSO authentication
func (a *Authenticator) authenticateSSO() (string, time.Time, error) {
	// Get OAuth metadata
	oauthInfo, err := a.client.GetOAuthInfo()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to get OAuth info: %w", err)
	}

	// Perform OAuth flow
	flow := NewOAuthFlow(a.config, oauthInfo, a.logger)
	token, err := flow.Authenticate()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("OAuth authentication failed: %w", err)
	}

	// Default expiry to 24 hours
	expiry := time.Now().Add(24 * time.Hour)

	// Store token
	if err := a.storage.StoreToken(token, expiry); err != nil {
		a.logger.Error("Failed to store token: %v", err)
		// Don't fail authentication if storage fails
	}

	return token, expiry, nil
}
