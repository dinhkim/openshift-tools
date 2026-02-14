package credential

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestOutputToken(t *testing.T) {
	tests := []struct {
		name                string
		token               string
		expirationTimestamp string
		wantErr             bool
	}{
		{
			name:                "valid token with expiration",
			token:               "sha256~test-token-12345",
			expirationTimestamp: "2024-12-31T23:59:59Z",
			wantErr:             false,
		},
		{
			name:                "valid token without expiration",
			token:               "sha256~test-token-67890",
			expirationTimestamp: "",
			wantErr:             false,
		},
		{
			name:                "empty token",
			token:               "",
			expirationTimestamp: "2024-12-31T23:59:59Z",
			wantErr:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := OutputToken(tt.token, tt.expirationTimestamp)

			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if (err != nil) != tt.wantErr {
				t.Errorf("OutputToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Validate JSON structure
			var cred ExecCredential
			if err := json.Unmarshal([]byte(output), &cred); err != nil {
				t.Fatalf("failed to unmarshal output: %v\nOutput: %s", err, output)
			}

			// Validate fields
			if cred.APIVersion != "client.authentication.k8s.io/v1beta1" {
				t.Errorf("APIVersion = %q, want %q", cred.APIVersion, "client.authentication.k8s.io/v1beta1")
			}
			if cred.Kind != "ExecCredential" {
				t.Errorf("Kind = %q, want %q", cred.Kind, "ExecCredential")
			}
			if cred.Spec == nil {
				t.Fatal("Spec is nil")
			}
			if cred.Spec.Interactive != false {
				t.Errorf("Spec.Interactive = %v, want false", cred.Spec.Interactive)
			}
			if cred.Status == nil {
				t.Fatal("Status is nil")
			}
			if cred.Status.Token != tt.token {
				t.Errorf("Status.Token = %q, want %q", cred.Status.Token, tt.token)
			}
			if cred.Status.ExpirationTimestamp != tt.expirationTimestamp {
				t.Errorf("Status.ExpirationTimestamp = %q, want %q", cred.Status.ExpirationTimestamp, tt.expirationTimestamp)
			}
			if cred.Status.Error != "" {
				t.Errorf("Status.Error = %q, want empty", cred.Status.Error)
			}
		})
	}
}

func TestOutputError(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "simple error message",
			message: "authentication failed",
		},
		{
			name:    "detailed error message",
			message: "failed to connect to OAuth endpoint: connection refused",
		},
		{
			name:    "empty error message",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// OutputError calls os.Exit(1), so we need to catch it
			// We'll use a separate goroutine and recover from panic if needed
			// For testing, we'll test the JSON encoding separately
			oldExit := osExit
			defer func() { osExit = oldExit }()

			exitCode := 0
			osExit = func(code int) {
				exitCode = code
			}

			OutputError(tt.message)

			// Restore stderr and read output
			w.Close()
			os.Stderr = oldStderr
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Validate exit code
			if exitCode != 1 {
				t.Errorf("exit code = %d, want 1", exitCode)
			}

			// Validate JSON structure
			var cred ExecCredential
			if err := json.Unmarshal([]byte(output), &cred); err != nil {
				t.Fatalf("failed to unmarshal output: %v\nOutput: %s", err, output)
			}

			// Validate fields
			if cred.APIVersion != "client.authentication.k8s.io/v1beta1" {
				t.Errorf("APIVersion = %q, want %q", cred.APIVersion, "client.authentication.k8s.io/v1beta1")
			}
			if cred.Kind != "ExecCredential" {
				t.Errorf("Kind = %q, want %q", cred.Kind, "ExecCredential")
			}
			if cred.Status == nil {
				t.Fatal("Status is nil")
			}
			if cred.Status.Error != tt.message {
				t.Errorf("Status.Error = %q, want %q", cred.Status.Error, tt.message)
			}
			// Token and ExpirationTimestamp should be empty for errors
			if cred.Status.Token != "" {
				t.Errorf("Status.Token = %q, want empty", cred.Status.Token)
			}
			if cred.Status.ExpirationTimestamp != "" {
				t.Errorf("Status.ExpirationTimestamp = %q, want empty", cred.Status.ExpirationTimestamp)
			}
		})
	}
}

func TestExecCredentialJSONFormat(t *testing.T) {
	// Test that the JSON output is properly formatted (indented)
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := OutputToken("test-token", "2024-12-31T23:59:59Z")
	if err != nil {
		t.Fatalf("OutputToken() error = %v", err)
	}

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check that output is indented (contains newlines and spaces)
	if !strings.Contains(output, "\n") {
		t.Error("output is not formatted with newlines")
	}
	if !strings.Contains(output, "  ") {
		t.Error("output is not indented")
	}
}
