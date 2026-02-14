package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config holds the configuration for the authentication plugin
type Config struct {
	ServerURL     string
	ClusterName   string
	VerifySSL     bool
	SecretStore   string
	Debug         bool
	SSOEnabled    bool
	SSOClientID   string
	SSOProvider   string
}

// KubernetesExecInfo represents the cluster info provided by kubectl
type KubernetesExecInfo struct {
	Spec struct {
		Cluster struct {
			Server string `json:"server"`
		} `json:"cluster"`
	} `json:"spec"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		ServerURL:   os.Getenv("OPENSHIFT_URL"),
		ClusterName: os.Getenv("CLUSTER_NAME"),
		VerifySSL:   getEnvBool("VERIFY_SSL", false),
		SecretStore: getEnvDefault("SECRET_STORE", "keychain"),
		Debug:       getEnvBool("DEBUG", false),
		SSOEnabled:  getEnvBool("SSO_ENABLED", true),
		SSOClientID: getEnvDefault("SSO_CLIENT_ID", "openshift-browser-client"),
		SSOProvider: os.Getenv("SSO_PROVIDER"),
	}

	// If OPENSHIFT_URL is not set, try to get it from KUBERNETES_EXEC_INFO
	if cfg.ServerURL == "" {
		execInfo := os.Getenv("KUBERNETES_EXEC_INFO")
		if execInfo != "" {
			var info KubernetesExecInfo
			if err := json.Unmarshal([]byte(execInfo), &info); err == nil {
				cfg.ServerURL = info.Spec.Cluster.Server
			}
		}
	}

	// Validate required fields
	if cfg.ServerURL == "" {
		return nil, fmt.Errorf("OPENSHIFT_URL is required (or enable provideClusterInfo in kubeconfig)")
	}

	if cfg.ClusterName == "" {
		return nil, fmt.Errorf("CLUSTER_NAME is required")
	}

	return cfg, nil
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes"
}

func getEnvDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
