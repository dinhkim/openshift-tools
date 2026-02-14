package config

import (
	"flag"
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		args        []string
		wantErr     bool
		wantConfig  *Config
		errContains string
	}{
		{
			name: "valid config with env vars",
			envVars: map[string]string{
				"CLUSTER_NAME":  "test-cluster",
				"OPENSHIFT_URL": "https://api.test.com:6443",
				"VERIFY_SSL":    "true",
				"SECRET_STORE":  "gopass",
				"DEBUG":         "true",
			},
			args:    []string{},
			wantErr: false,
			wantConfig: &Config{
				ClusterName:  "test-cluster",
				OpenShiftURL: "https://api.test.com:6443",
				VerifySSL:    true,
				SecretStore:  "gopass",
				Debug:        true,
			},
		},
		{
			name: "flags override env vars",
			envVars: map[string]string{
				"CLUSTER_NAME":  "env-cluster",
				"OPENSHIFT_URL": "https://env.test.com:6443",
			},
			args: []string{
				"-cluster-name=flag-cluster",
				"-openshift-url=https://flag.test.com:6443",
			},
			wantErr: false,
			wantConfig: &Config{
				ClusterName:  "flag-cluster",
				OpenShiftURL: "https://flag.test.com:6443",
				VerifySSL:    false,
				SecretStore:  "keychain",
				Debug:        false,
			},
		},
		{
			name: "default values",
			envVars: map[string]string{
				"CLUSTER_NAME":  "test-cluster",
				"OPENSHIFT_URL": "https://api.test.com:6443",
			},
			args:    []string{},
			wantErr: false,
			wantConfig: &Config{
				ClusterName:  "test-cluster",
				OpenShiftURL: "https://api.test.com:6443",
				VerifySSL:    false,
				SecretStore:  "keychain",
				Debug:        false,
			},
		},
		{
			name: "missing cluster name",
			envVars: map[string]string{
				"OPENSHIFT_URL": "https://api.test.com:6443",
			},
			args:        []string{},
			wantErr:     true,
			errContains: "CLUSTER_NAME is required",
		},
		{
			name: "missing openshift url",
			envVars: map[string]string{
				"CLUSTER_NAME": "test-cluster",
			},
			args:        []string{},
			wantErr:     true,
			errContains: "OpenShift URL is required",
		},
		{
			name: "invalid secret store",
			envVars: map[string]string{
				"CLUSTER_NAME":  "test-cluster",
				"OPENSHIFT_URL": "https://api.test.com:6443",
				"SECRET_STORE":  "invalid-store",
			},
			args:        []string{},
			wantErr:     true,
			errContains: "invalid secret store",
		},
		{
			name: "KUBERNETES_EXEC_INFO provides URL",
			envVars: map[string]string{
				"CLUSTER_NAME":         "test-cluster",
				"KUBERNETES_EXEC_INFO": `{"spec":{"cluster":{"server":"https://exec-info.test.com:6443"}}}`,
			},
			args:    []string{},
			wantErr: false,
			wantConfig: &Config{
				ClusterName:  "test-cluster",
				OpenShiftURL: "https://exec-info.test.com:6443",
				VerifySSL:    false,
				SecretStore:  "keychain",
				Debug:        false,
			},
		},
		{
			name: "OPENSHIFT_URL overrides KUBERNETES_EXEC_INFO",
			envVars: map[string]string{
				"CLUSTER_NAME":         "test-cluster",
				"OPENSHIFT_URL":        "https://direct.test.com:6443",
				"KUBERNETES_EXEC_INFO": `{"spec":{"cluster":{"server":"https://exec-info.test.com:6443"}}}`,
			},
			args:    []string{},
			wantErr: false,
			wantConfig: &Config{
				ClusterName:  "test-cluster",
				OpenShiftURL: "https://direct.test.com:6443",
				VerifySSL:    false,
				SecretStore:  "keychain",
				Debug:        false,
			},
		},
		{
			name: "boolean env var variations",
			envVars: map[string]string{
				"CLUSTER_NAME":  "test-cluster",
				"OPENSHIFT_URL": "https://api.test.com:6443",
				"VERIFY_SSL":    "1",
				"DEBUG":         "yes",
			},
			args:    []string{},
			wantErr: false,
			wantConfig: &Config{
				ClusterName:  "test-cluster",
				OpenShiftURL: "https://api.test.com:6443",
				VerifySSL:    true,
				SecretStore:  "keychain",
				Debug:        true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Clearenv()

			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Reset flags for each test
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			os.Args = append([]string{"cmd"}, tt.args...)

			got, err := Load()

			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errContains != "" && err != nil {
					if !contains(err.Error(), tt.errContains) {
						t.Errorf("Load() error = %v, want error containing %q", err, tt.errContains)
					}
				}
				return
			}

			if got == nil {
				t.Fatal("Load() returned nil config")
			}

			if got.ClusterName != tt.wantConfig.ClusterName {
				t.Errorf("ClusterName = %q, want %q", got.ClusterName, tt.wantConfig.ClusterName)
			}
			if got.OpenShiftURL != tt.wantConfig.OpenShiftURL {
				t.Errorf("OpenShiftURL = %q, want %q", got.OpenShiftURL, tt.wantConfig.OpenShiftURL)
			}
			if got.VerifySSL != tt.wantConfig.VerifySSL {
				t.Errorf("VerifySSL = %v, want %v", got.VerifySSL, tt.wantConfig.VerifySSL)
			}
			if got.SecretStore != tt.wantConfig.SecretStore {
				t.Errorf("SecretStore = %q, want %q", got.SecretStore, tt.wantConfig.SecretStore)
			}
			if got.Debug != tt.wantConfig.Debug {
				t.Errorf("Debug = %v, want %v", got.Debug, tt.wantConfig.Debug)
			}
		})
	}
}

func TestGetEnvDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		want         string
	}{
		{
			name:         "env var set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "custom",
			want:         "custom",
		},
		{
			name:         "env var not set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			}

			got := getEnvDefault(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnvDefault() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		want         bool
	}{
		{
			name:         "true",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "true",
			want:         true,
		},
		{
			name:         "1",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "1",
			want:         true,
		},
		{
			name:         "yes",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "yes",
			want:         true,
		},
		{
			name:         "false",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "false",
			want:         false,
		},
		{
			name:         "empty uses default true",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "",
			want:         true,
		},
		{
			name:         "empty uses default false",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			}

			got := getEnvBool(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnvBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
