package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/kim-truong/openshift-auth-plugin/pkg/auth"
	"github.com/kim-truong/openshift-auth-plugin/pkg/config"
	"github.com/kim-truong/openshift-auth-plugin/pkg/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthv1 "k8s.io/client-go/pkg/apis/clientauthentication/v1"
)

func main() {
	if err := run(); err != nil {
		outputError(err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration from environment
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger
	logger := config.NewLogger(cfg.Debug)
	logger.Debug("Starting OpenShift authentication plugin")
	logger.Debug("Cluster: %s, Server: %s", cfg.ClusterName, cfg.ServerURL)

	// Initialize storage backend
	store, err := storage.NewStorage(cfg.SecretStore, cfg.ClusterName, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize authenticator
	authenticator := auth.NewAuthenticator(cfg, store, logger)

	// Try to get a valid token
	token, expiry, err := authenticator.GetToken()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Output the credential in ExecCredential format
	outputCredential(token, expiry)
	return nil
}

func outputCredential(token string, expiry time.Time) {
	cred := &clientauthv1.ExecCredential{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "client.authentication.k8s.io/v1",
			Kind:       "ExecCredential",
		},
		Status: &clientauthv1.ExecCredentialStatus{
			Token:               token,
			ExpirationTimestamp: &metav1.Time{Time: expiry},
		},
	}

	output, err := json.MarshalIndent(cred, "", "  ")
	if err != nil {
		outputError(fmt.Errorf("failed to marshal credential: %w", err))
		os.Exit(1)
	}

	fmt.Println(string(output))
}

func outputError(err error) {
	cred := &clientauthv1.ExecCredential{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "client.authentication.k8s.io/v1",
			Kind:       "ExecCredential",
		},
		Status: &clientauthv1.ExecCredentialStatus{},
	}

	// Set error message
	errMsg := err.Error()
	cred.Status.ExpirationTimestamp = &metav1.Time{Time: time.Now()}

	output, _ := json.MarshalIndent(cred, "", "  ")
	fmt.Fprintf(os.Stderr, "%s\nError: %s\n", string(output), errMsg)
}
