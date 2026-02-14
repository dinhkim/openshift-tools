package main

import (
	"encoding/json"
	"fmt"

	"github.com/dinhkim/openshift-tools/internal/auth"
	"github.com/dinhkim/openshift-tools/internal/config"
	"github.com/dinhkim/openshift-tools/internal/credential"
	"github.com/dinhkim/openshift-tools/internal/log"
	"github.com/dinhkim/openshift-tools/internal/storage"
)

func main() {
	// Load configuration from environment variables and flags
	cfg, err := config.Load()
	if err != nil {
		credential.OutputError(fmt.Sprintf("Configuration error: %v", err))
		return
	}

	// Initialize logger
	logger := log.New(cfg.Debug)

	// Initialize storage backend
	store, err := storage.New(cfg.SecretStore)
	if err != nil {
		credential.OutputError(fmt.Sprintf("Storage initialization error: %v", err))
		return
	}

	// Initialize authenticator
	authenticator := auth.NewAuthenticator(cfg.OpenShiftURL, cfg.VerifySSL, store, logger)

	// Try to run the authentication flow
	if err := run(cfg, store, authenticator, logger); err != nil {
		credential.OutputError(err.Error())
		return
	}
}

func run(cfg *config.Config, store storage.Storage, authenticator *auth.Authenticator, logger *log.Logger) error {
	// Method 1: Try cached token
	logger.Debug("Attempting to use cached token...")

	tokenStr, err := store.Get(cfg.ClusterName, "token")
	if err == nil && tokenStr != "" {
		var tokenData auth.TokenData
		if err := json.Unmarshal([]byte(tokenStr), &tokenData); err == nil {
			logger.Debug("Found cached token, validating...")

			if err := authenticator.ValidateToken(tokenData.Token, tokenData.ExpirationTimestamp); err == nil {
				logger.Debug("Cached token is valid, using it")

				if err := credential.OutputToken(tokenData.Token, tokenData.ExpirationTimestamp); err != nil {
					return fmt.Errorf("failed to output token: %w", err)
				}
				return nil
			}

			logger.Debug("Cached token validation failed: %v", err)
		} else {
			logger.Debug("Failed to parse cached token: %v", err)
		}
	} else {
		logger.Debug("No cached token found")
	}

	// Method 2: Authenticate with username/password
	logger.Debug("Attempting to authenticate with stored credentials...")

	credStr, err := store.Get(cfg.ClusterName, "credentials")
	if err != nil || credStr == "" {
		return fmt.Errorf("no valid authentication method found. Please store credentials in %s. See README.md for more information", cfg.SecretStore)
	}

	var creds auth.CredentialsData
	if err := json.Unmarshal([]byte(credStr), &creds); err != nil {
		return fmt.Errorf("failed to parse stored credentials: %w", err)
	}

	if creds.Username == "" || creds.Password == "" {
		return fmt.Errorf("stored credentials are incomplete (missing username or password)")
	}

	logger.Debug("Found credentials for user: %s", creds.Username)

	tokenData, err := authenticator.AuthenticateWithCredentials(cfg.ClusterName, creds.Username, creds.Password)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if err := credential.OutputToken(tokenData.Token, tokenData.ExpirationTimestamp); err != nil {
		return fmt.Errorf("failed to output token: %w", err)
	}

	return nil
}
