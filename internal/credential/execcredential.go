package credential

import (
	"encoding/json"
	"fmt"
	"os"
)

// ExecCredential represents the Kubernetes client.authentication.k8s.io/v1beta1 ExecCredential type
type ExecCredential struct {
	APIVersion string                `json:"apiVersion"`
	Kind       string                `json:"kind"`
	Spec       *ExecCredentialSpec   `json:"spec,omitempty"`
	Status     *ExecCredentialStatus `json:"status,omitempty"`
}

// ExecCredentialSpec contains the spec for ExecCredential
type ExecCredentialSpec struct {
	Interactive bool `json:"interactive"`
}

// ExecCredentialStatus contains the status for ExecCredential
type ExecCredentialStatus struct {
	Token               string `json:"token,omitempty"`
	ExpirationTimestamp string `json:"expirationTimestamp,omitempty"`
	Error               string `json:"error,omitempty"`
}

// OutputToken writes a successful ExecCredential with token to stdout
func OutputToken(token, expirationTimestamp string) error {
	cred := &ExecCredential{
		APIVersion: "client.authentication.k8s.io/v1beta1",
		Kind:       "ExecCredential",
		Spec: &ExecCredentialSpec{
			Interactive: false,
		},
		Status: &ExecCredentialStatus{
			Token:               token,
			ExpirationTimestamp: expirationTimestamp,
		},
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(cred)
}

// osExit is a variable to allow mocking os.Exit in tests
var osExit = os.Exit

// OutputError writes an ExecCredential with error to stderr and exits with code 1
func OutputError(message string) {
	cred := &ExecCredential{
		APIVersion: "client.authentication.k8s.io/v1beta1",
		Kind:       "ExecCredential",
		Status: &ExecCredentialStatus{
			Error: message,
		},
	}

	encoder := json.NewEncoder(os.Stderr)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cred); err != nil {
		// Fallback if JSON encoding fails
		fmt.Fprintf(os.Stderr, "Failed to encode error credential: %v\n", err)
	}
	osExit(1)
}
