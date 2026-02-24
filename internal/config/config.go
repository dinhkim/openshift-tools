package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the auth plugin
type Config struct {
	ClusterName  string
	OpenShiftURL string
	VerifySSL    bool
	SecretStore  string
	Debug        bool
	SSOTimeout   int    // seconds to wait for SSO callback
	IDPHint      string // optional IdP identifier to pass in the SSO authorization URL
}

// KubernetesExecInfo represents the cluster info passed via KUBERNETES_EXEC_INFO env var
type KubernetesExecInfo struct {
	Spec struct {
		Cluster struct {
			Server string `json:"server"`
		} `json:"cluster"`
	} `json:"spec"`
}

// Load loads configuration from environment variables and command-line flags
// Flags take precedence over environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Define flags
	flag.StringVar(&cfg.ClusterName, "cluster-name", os.Getenv("CLUSTER_NAME"), "Cluster identifier for credential storage")
	flag.StringVar(&cfg.OpenShiftURL, "openshift-url", os.Getenv("OPENSHIFT_URL"), "OpenShift API URL")
	flag.BoolVar(&cfg.VerifySSL, "verify-ssl", getEnvBool("VERIFY_SSL", false), "Verify SSL certificates")
	flag.StringVar(&cfg.SecretStore, "secret-store", getEnvDefault("SECRET_STORE", "keychain"), "Secret store backend: keychain or gopass")
	flag.BoolVar(&cfg.Debug, "debug", getEnvBool("DEBUG", false), "Enable debug logging")
	flag.IntVar(&cfg.SSOTimeout, "sso-timeout", getEnvInt("SSO_TIMEOUT", 120), "Seconds to wait for SSO authentication (default: 120)")
	flag.StringVar(&cfg.IDPHint, "idp", getEnvDefault("IDP", ""), "Identity provider hint to pass in the SSO authorization URL (optional)")

	flag.Parse()

	// Validate cluster name is required
	if cfg.ClusterName == "" {
		return nil, fmt.Errorf("CLUSTER_NAME is required (set via --cluster-name flag or CLUSTER_NAME environment variable)")
	}

	// If OpenShiftURL is not set, try to get it from KUBERNETES_EXEC_INFO
	if cfg.OpenShiftURL == "" {
		execInfo := os.Getenv("KUBERNETES_EXEC_INFO")
		if execInfo != "" {
			var info KubernetesExecInfo
			if err := json.Unmarshal([]byte(execInfo), &info); err != nil {
				return nil, fmt.Errorf("failed to parse KUBERNETES_EXEC_INFO: %w", err)
			}
			cfg.OpenShiftURL = info.Spec.Cluster.Server
		}
	}

	// Validate OpenShift URL is set
	if cfg.OpenShiftURL == "" {
		return nil, fmt.Errorf("OpenShift URL is required (set via --openshift-url flag, OPENSHIFT_URL environment variable, or enable 'provideClusterInfo' in kubeconfig)")
	}

	// Validate secret store value
	if cfg.SecretStore != "keychain" && cfg.SecretStore != "gopass" {
		return nil, fmt.Errorf("invalid secret store '%s': must be 'keychain' or 'gopass'", cfg.SecretStore)
	}

	return cfg, nil
}

// getEnvDefault returns the environment variable value or a default if not set
func getEnvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool returns the boolean value of an environment variable
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes"
}

// getEnvInt returns the integer value of an environment variable
func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}
