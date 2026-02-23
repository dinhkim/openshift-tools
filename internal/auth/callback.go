package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

const callbackPath = "/callback"

// callbackResult holds the result from the OAuth callback
type callbackResult struct {
	Code string
	Err  error
}

// callbackServer manages the local HTTP server that receives the OAuth callback
type callbackServer struct {
	listener      net.Listener
	server        *http.Server
	resultCh      chan callbackResult
	port          int
	expectedState string
}

// startCallbackServer starts a local HTTP server on a random port to receive
// the OAuth authorization code callback
func startCallbackServer(expectedState string) (*callbackServer, error) {
	// Listen on loopback address with random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	resultCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	srv := &http.Server{
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    30 * time.Second,
		MaxHeaderBytes: 4096,
	}

	cs := &callbackServer{
		listener:      listener,
		server:        srv,
		resultCh:      resultCh,
		port:          port,
		expectedState: expectedState,
	}

	mux.HandleFunc(callbackPath, cs.handleCallback)

	// Start serving in background
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			resultCh <- callbackResult{Err: fmt.Errorf("callback server error: %w", err)}
		}
	}()

	return cs, nil
}

// handleCallback handles the OAuth callback request
func (cs *callbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Validate state parameter if expected
	if cs.expectedState != "" {
		state := r.URL.Query().Get("state")
		if state == "" {
			cs.resultCh <- callbackResult{
				Err: fmt.Errorf("state parameter missing in callback"),
			}
			http.Error(w, "State parameter missing", http.StatusBadRequest)
			return
		}
		if state != cs.expectedState {
			cs.resultCh <- callbackResult{
				Err: fmt.Errorf("state parameter mismatch - possible CSRF attack"),
			}
			http.Error(w, "State parameter mismatch", http.StatusBadRequest)
			return
		}
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		errMsg := r.URL.Query().Get("error")
		errDesc := r.URL.Query().Get("error_description")
		if errMsg != "" {
			cs.resultCh <- callbackResult{
				Err: fmt.Errorf("OAuth error: %s - %s", errMsg, errDesc),
			}
			http.Error(w, fmt.Sprintf("Authentication failed: %s", errMsg), http.StatusBadRequest)
			return
		}
		cs.resultCh <- callbackResult{
			Err: fmt.Errorf("no authorization code received"),
		}
		http.Error(w, "No authorization code received", http.StatusBadRequest)
		return
	}

	cs.resultCh <- callbackResult{Code: code}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head><title>OpenShift Authentication</title></head>
<body>
<h2>Login successful</h2>
<p>You have been authenticated successfully. You may close this window and return to your terminal.</p>
</body>
</html>`)
}

// redirectURI returns the full redirect URI for this callback server
func (cs *callbackServer) redirectURI() string {
	return fmt.Sprintf("http://127.0.0.1:%d%s", cs.port, callbackPath)
}

// waitForCallback blocks until an authorization code is received or the timeout expires
func (cs *callbackServer) waitForCallback(timeout time.Duration) (string, error) {
	select {
	case result := <-cs.resultCh:
		return result.Code, result.Err
	case <-time.After(timeout):
		return "", fmt.Errorf("SSO authentication timed out after %v — no response received from browser", timeout)
	}
}

// shutdown gracefully shuts down the callback server
func (cs *callbackServer) shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cs.server.Shutdown(ctx)
}

// openBrowser opens the given URL in the user's default browser.
// If the browser cannot be opened, it returns an error.
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
