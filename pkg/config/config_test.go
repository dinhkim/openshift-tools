package config

import (
	"os"
	"testing"
)

func TestLoadConfig_MissingClusterName(t *testing.T) {
	// Clear environment
	os.Unsetenv("CLUSTER_NAME")
	os.Unsetenv("OPENSHIFT_URL")
	os.Setenv("OPENSHIFT_URL", "https://api.test.example.com:6443")

	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error when CLUSTER_NAME is missing, got nil")
	}

	// Cleanup
	os.Unsetenv("OPENSHIFT_URL")
}

func TestLoadConfig_MissingServerURL(t *testing.T) {
	// Clear environment
	os.Unsetenv("CLUSTER_NAME")
	os.Unsetenv("OPENSHIFT_URL")
	os.Unsetenv("KUBERNETES_EXEC_INFO")
	os.Setenv("CLUSTER_NAME", "test-cluster")

	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error when OPENSHIFT_URL is missing, got nil")
	}

	// Cleanup
	os.Unsetenv("CLUSTER_NAME")
}

func TestLoadConfig_ValidConfig(t *testing.T) {
	// Set required environment variables
	os.Setenv("CLUSTER_NAME", "test-cluster")
	os.Setenv("OPENSHIFT_URL", "https://api.test.example.com:6443")
	os.Setenv("DEBUG", "true")
	os.Setenv("VERIFY_SSL", "true")
	os.Setenv("SECRET_STORE", "gopass")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.ClusterName != "test-cluster" {
		t.Errorf("Expected ClusterName 'test-cluster', got '%s'", cfg.ClusterName)
	}

	if cfg.ServerURL != "https://api.test.example.com:6443" {
		t.Errorf("Expected ServerURL 'https://api.test.example.com:6443', got '%s'", cfg.ServerURL)
	}

	if !cfg.Debug {
		t.Error("Expected Debug to be true")
	}

	if !cfg.VerifySSL {
		t.Error("Expected VerifySSL to be true")
	}

	if cfg.SecretStore != "gopass" {
		t.Errorf("Expected SecretStore 'gopass', got '%s'", cfg.SecretStore)
	}

	// Cleanup
	os.Unsetenv("CLUSTER_NAME")
	os.Unsetenv("OPENSHIFT_URL")
	os.Unsetenv("DEBUG")
	os.Unsetenv("VERIFY_SSL")
	os.Unsetenv("SECRET_STORE")
}

func TestLoadConfig_Defaults(t *testing.T) {
	// Set only required variables
	os.Setenv("CLUSTER_NAME", "test-cluster")
	os.Setenv("OPENSHIFT_URL", "https://api.test.example.com:6443")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Check defaults
	if cfg.Debug {
		t.Error("Expected Debug to be false by default")
	}

	if cfg.VerifySSL {
		t.Error("Expected VerifySSL to be false by default")
	}

	if cfg.SecretStore != "keychain" {
		t.Errorf("Expected SecretStore 'keychain' by default, got '%s'", cfg.SecretStore)
	}

	if !cfg.SSOEnabled {
		t.Error("Expected SSOEnabled to be true by default")
	}

	if cfg.SSOClientID != "openshift-browser-client" {
		t.Errorf("Expected SSOClientID 'openshift-browser-client' by default, got '%s'", cfg.SSOClientID)
	}

	// Cleanup
	os.Unsetenv("CLUSTER_NAME")
	os.Unsetenv("OPENSHIFT_URL")
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{"empty string with default true", "", true, true},
		{"empty string with default false", "", false, false},
		{"true string", "true", false, true},
		{"1 string", "1", false, true},
		{"yes string", "yes", false, true},
		{"false string", "false", true, false},
		{"0 string", "0", true, false},
		{"no string", "no", true, false},
		{"random string", "random", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("TEST_BOOL", tt.envValue)
			result := getEnvBool("TEST_BOOL", tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
			os.Unsetenv("TEST_BOOL")
		})
	}
}

func TestGetEnvDefault(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue string
		expected     string
	}{
		{"empty string", "", "default", "default"},
		{"set value", "custom", "default", "custom"},
		{"empty default", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("TEST_STRING", tt.envValue)
			} else {
				os.Unsetenv("TEST_STRING")
			}
			result := getEnvDefault("TEST_STRING", tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
			os.Unsetenv("TEST_STRING")
		})
	}
}
