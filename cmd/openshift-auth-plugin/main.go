package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/dinhkim/openshift-tools/internal/auth"
	"github.com/dinhkim/openshift-tools/internal/config"
	"github.com/dinhkim/openshift-tools/internal/credential"
	"github.com/dinhkim/openshift-tools/internal/log"
	"github.com/dinhkim/openshift-tools/internal/storage"
)

var version = "dev"

func main() {
	// Handle --version flag before normal flag parsing
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Fprintf(os.Stderr, "openshift-auth-plugin %s\n", version)
		os.Exit(0)
	}

	// Ensure flag.CommandLine.Parse is called if there are any flags (config.Load will call flag.Parse)
	_ = flag.NewFlagSet("", flag.ContinueOnError)

	// Load configuration from environment variables and flags
	cfg, err := config.Load()
	if err != nil {
		credential.OutputError(fmt.Sprintf("Configuration error: %v", err))
		os.Exit(1)
	}

	// Initialize logger
	logger := log.New(cfg.Debug)

	// Initialize storage backend
	store, err := storage.New(cfg.SecretStore)
	if err != nil {
		credential.OutputError(fmt.Sprintf("Storage initialization error: %v", err))
		os.Exit(1)
	}

	// Initialize authenticator
	authenticator := auth.NewAuthenticator(cfg.OpenShiftURL, cfg.VerifySSL, store, logger)

	// Try to run the authentication flow
	if err := run(cfg, store, authenticator, logger); err != nil {
		credential.OutputError(err.Error())
		os.Exit(1)
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

	// Method 2: SSO/PKCE authentication via browser
	logger.Debug("Attempting SSO authentication...")

	ssoTokenData, ssoErr := authenticator.AuthenticateWithSSO(cfg.ClusterName, cfg.SSOTimeout, cfg.IDPHint)
	if ssoErr == nil {
		logger.Debug("SSO authentication successful")
		if err := credential.OutputToken(ssoTokenData.Token, ssoTokenData.ExpirationTimestamp); err != nil {
			return fmt.Errorf("failed to output token: %w", err)
		}
		return nil
	}
	logger.Debug("SSO authentication failed: %v", ssoErr)

	// Method 3: Authenticate with username/password
	logger.Debug("Attempting to authenticate with stored credentials...")

	credStr, err := store.Get(cfg.ClusterName, "credentials")
	if err != nil || credStr == "" {
		return fmt.Errorf("no valid authentication method found. SSO failed: %v. No stored credentials found in %s. See README.md for more information", ssoErr, cfg.SecretStore)
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
