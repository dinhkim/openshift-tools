package auth

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestStartCallbackServer(t *testing.T) {
	cs, err := startCallbackServer()
	if err != nil {
		t.Fatalf("startCallbackServer() error = %v", err)
	}
	defer cs.shutdown()

	if cs.port == 0 {
		t.Error("port should not be 0")
	}

	if cs.listener == nil {
		t.Error("listener is nil")
	}

	if cs.server == nil {
		t.Error("server is nil")
	}

	if cs.resultCh == nil {
		t.Error("resultCh is nil")
	}
}

func TestCallbackServer_RedirectURI(t *testing.T) {
	cs, err := startCallbackServer()
	if err != nil {
		t.Fatalf("startCallbackServer() error = %v", err)
	}
	defer cs.shutdown()

	uri := cs.redirectURI()
	expected := fmt.Sprintf("http://127.0.0.1:%d/callback", cs.port)
	if uri != expected {
		t.Errorf("redirectURI() = %q, want %q", uri, expected)
	}
}

func TestCallbackServer_ReceivesCode(t *testing.T) {
	cs, err := startCallbackServer()
	if err != nil {
		t.Fatalf("startCallbackServer() error = %v", err)
	}
	defer cs.shutdown()

	// Simulate the OAuth redirect by making an HTTP request to the callback
	go func() {
		callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?code=test-auth-code-123", cs.port)
		resp, err := http.Get(callbackURL)
		if err != nil {
			t.Errorf("failed to make callback request: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("callback response status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
	}()

	code, err := cs.waitForCallback(5 * time.Second)
	if err != nil {
		t.Fatalf("waitForCallback() error = %v", err)
	}

	if code != "test-auth-code-123" {
		t.Errorf("code = %q, want %q", code, "test-auth-code-123")
	}
}

func TestCallbackServer_MissingCode(t *testing.T) {
	cs, err := startCallbackServer()
	if err != nil {
		t.Fatalf("startCallbackServer() error = %v", err)
	}
	defer cs.shutdown()

	// Make callback without code parameter
	go func() {
		callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback", cs.port)
		resp, err := http.Get(callbackURL)
		if err != nil {
			t.Errorf("failed to make callback request: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("callback response status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	}()

	_, err = cs.waitForCallback(5 * time.Second)
	if err == nil {
		t.Error("waitForCallback() expected error for missing code, got nil")
	}
}

func TestCallbackServer_OAuthError(t *testing.T) {
	cs, err := startCallbackServer()
	if err != nil {
		t.Fatalf("startCallbackServer() error = %v", err)
	}
	defer cs.shutdown()

	// Simulate an OAuth error response
	go func() {
		callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?error=access_denied&error_description=user+denied+access", cs.port)
		resp, err := http.Get(callbackURL)
		if err != nil {
			t.Errorf("failed to make callback request: %v", err)
			return
		}
		defer resp.Body.Close()
	}()

	_, err = cs.waitForCallback(5 * time.Second)
	if err == nil {
		t.Error("waitForCallback() expected error for OAuth error, got nil")
	}
	// Verify error message contains the OAuth error
	if err != nil && !containsString(err.Error(), "access_denied") {
		t.Errorf("error should contain 'access_denied', got: %v", err)
	}
}

func TestCallbackServer_Timeout(t *testing.T) {
	cs, err := startCallbackServer()
	if err != nil {
		t.Fatalf("startCallbackServer() error = %v", err)
	}
	defer cs.shutdown()

	// Use a very short timeout - no callback will arrive
	_, err = cs.waitForCallback(100 * time.Millisecond)
	if err == nil {
		t.Error("waitForCallback() expected timeout error, got nil")
	}
	if err != nil && !containsString(err.Error(), "timed out") {
		t.Errorf("error should contain 'timed out', got: %v", err)
	}
}

func TestCallbackServer_Shutdown(t *testing.T) {
	cs, err := startCallbackServer()
	if err != nil {
		t.Fatalf("startCallbackServer() error = %v", err)
	}

	port := cs.port
	cs.shutdown()

	// After shutdown, the server should not accept new connections
	// Give it a moment to shut down
	time.Sleep(100 * time.Millisecond)

	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?code=test", port)
	_, err = http.Get(callbackURL)
	if err == nil {
		t.Error("expected connection error after shutdown, got nil")
	}
}

func TestCallbackServer_MultiplePorts(t *testing.T) {
	// Start multiple servers to ensure they get different ports
	cs1, err := startCallbackServer()
	if err != nil {
		t.Fatalf("startCallbackServer() 1 error = %v", err)
	}
	defer cs1.shutdown()

	cs2, err := startCallbackServer()
	if err != nil {
		t.Fatalf("startCallbackServer() 2 error = %v", err)
	}
	defer cs2.shutdown()

	if cs1.port == cs2.port {
		t.Errorf("two servers got same port: %d", cs1.port)
	}
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
