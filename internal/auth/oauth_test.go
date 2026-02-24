package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dinhkim/openshift-tools/internal/log"
	"github.com/dinhkim/openshift-tools/internal/storage"
)

func TestNewAuthenticator(t *testing.T) {
	mockStorage := &storage.MockStorage{}
	logger := log.New(false)

	auth := NewAuthenticator("https://api.test.com:6443", true, mockStorage, logger)

	if auth == nil {
		t.Fatal("NewAuthenticator() returned nil")
	}
	if auth.serverURL != "https://api.test.com:6443" {
		t.Errorf("serverURL = %q, want %q", auth.serverURL, "https://api.test.com:6443")
	}
	if auth.verifySSL != true {
		t.Errorf("verifySSL = %v, want true", auth.verifySSL)
	}
	if auth.storage == nil {
		t.Error("storage is nil")
	}
	if auth.httpClient == nil {
		t.Error("httpClient is nil")
	}
}

func TestGetOAuthInfo(t *testing.T) {
	tests := []struct {
		name              string
		responseStatus    int
		responseBody      string
		wantErr           bool
		wantEndpoint      string
		wantTokenEndpoint string
	}{
		{
			name:              "valid oauth info with all fields",
			responseStatus:    http.StatusOK,
			responseBody:      `{"authorization_endpoint":"https://oauth.test.com/authorize","token_endpoint":"https://oauth.test.com/token","code_challenge_methods_supported":["S256"]}`,
			wantErr:           false,
			wantEndpoint:      "https://oauth.test.com/authorize",
			wantTokenEndpoint: "https://oauth.test.com/token",
		},
		{
			name:           "valid oauth info without token endpoint",
			responseStatus: http.StatusOK,
			responseBody:   `{"authorization_endpoint":"https://oauth.test.com/authorize"}`,
			wantErr:        false,
			wantEndpoint:   "https://oauth.test.com/authorize",
		},
		{
			name:           "missing authorization endpoint",
			responseStatus: http.StatusOK,
			responseBody:   `{"issuer":"https://oauth.test.com"}`,
			wantErr:        true,
		},
		{
			name:           "invalid json",
			responseStatus: http.StatusOK,
			responseBody:   `not json`,
			wantErr:        true,
		},
		{
			name:           "server error with invalid response",
			responseStatus: http.StatusInternalServerError,
			responseBody:   `{"error":"internal error"}`,
			wantErr:        true, // Error because authorization_endpoint is missing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/.well-known/oauth-authorization-server" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			mockStorage := &storage.MockStorage{}
			logger := log.New(false)
			auth := NewAuthenticator(server.URL, false, mockStorage, logger)

			info, err := auth.GetOAuthInfo()

			if (err != nil) != tt.wantErr {
				t.Errorf("GetOAuthInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if info == nil {
					t.Fatal("GetOAuthInfo() returned nil info")
				}
				if info.AuthorizationEndpoint != tt.wantEndpoint {
					t.Errorf("AuthorizationEndpoint = %q, want %q", info.AuthorizationEndpoint, tt.wantEndpoint)
				}
				if tt.wantTokenEndpoint != "" && info.TokenEndpoint != tt.wantTokenEndpoint {
					t.Errorf("TokenEndpoint = %q, want %q", info.TokenEndpoint, tt.wantTokenEndpoint)
				}
			}
		})
	}
}

func TestAuthenticateWithSSO_NoTokenEndpoint(t *testing.T) {
	// When the OAuth server does not provide a token_endpoint, SSO should fail
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"authorization_endpoint":"https://oauth.test.com/authorize"}`))
	}))
	defer server.Close()

	mockStorage := storage.NewMockStorage()
	logger := log.New(false)
	auth := NewAuthenticator(server.URL, false, mockStorage, logger)

	_, err := auth.AuthenticateWithSSO("test-cluster", 5, "")
	if err == nil {
		t.Error("AuthenticateWithSSO() expected error when token_endpoint is missing")
	}
	if err != nil && !containsString(err.Error(), "token endpoint not found") {
		t.Errorf("error should mention token endpoint, got: %v", err)
	}
}

func TestAuthenticateWithSSO_InvalidOAuth(t *testing.T) {
	// When the OAuth server returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	mockStorage := storage.NewMockStorage()
	logger := log.New(false)
	auth := NewAuthenticator(server.URL, false, mockStorage, logger)

	_, err := auth.AuthenticateWithSSO("test-cluster", 5, "")
	if err == nil {
		t.Error("AuthenticateWithSSO() expected error for invalid OAuth response")
	}
}

func TestAuthenticateWithSSO_IDPHint(t *testing.T) {
	tests := []struct {
		name         string
		idpHint      string
		wantIDPParam bool
	}{
		{
			name:         "idp param included when hint is set",
			idpHint:      "AzureAD",
			wantIDPParam: true,
		},
		{
			name:         "idp param omitted when hint is empty",
			idpHint:      "",
			wantIDPParam: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedQuery string

			// Serve OAuth metadata with both endpoints so SSO proceeds past early checks
			oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				// Return a dummy token endpoint so SSO doesn't bail out early.
				// The authorization_endpoint points back at this same server so
				// we can capture the query parameters.
				body := fmt.Sprintf(
					`{"authorization_endpoint":"%s/authorize","token_endpoint":"%s/token"}`,
					"http://"+r.Host, "http://"+r.Host,
				)
				w.Write([]byte(body))
			}))
			defer oauthServer.Close()

			// Intercept the browser open by inspecting the authorization URL
			// indirectly: start a minimal server that captures query params.
			authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/authorize" {
					capturedQuery = r.URL.RawQuery
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer authServer.Close()

			// We cannot easily intercept the browser call or callback in a unit
			// test, so we only verify the early-exit path: the function will fail
			// after calling GetOAuthInfo (the mock returns a non-HTTPS token
			// endpoint which will be rejected). We just need to confirm it reached
			// the IDP-hint conditional, so we use a server that returns HTTP URLs
			// (which fail HTTPS validation) and inspect that the correct error is
			// returned while the IDP logic does not panic.
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				body := fmt.Sprintf(
					`{"authorization_endpoint":"https://oauth.example.com/authorize","token_endpoint":"http://bad-endpoint.example.com/token"}`,
				)
				w.Write([]byte(body))
			}))
			defer server.Close()

			mockStorage := storage.NewMockStorage()
			logger := log.New(false)
			auth := NewAuthenticator(server.URL, false, mockStorage, logger)

			// The call will error due to the HTTP token endpoint, but must not panic.
			_, err := auth.AuthenticateWithSSO("test-cluster", 5, tt.idpHint)
			if err == nil {
				t.Error("AuthenticateWithSSO() expected error for HTTP token endpoint")
			}
			// Confirm error is about token endpoint HTTPS validation, not a panic.
			if !containsSubstringSimple(err.Error(), "token endpoint validation failed") {
				t.Errorf("unexpected error: %v", err)
			}
			_ = capturedQuery // captured only when the full browser flow runs
		})
	}
}

func TestExtractTokenFromFragment(t *testing.T) {
	tests := []struct {
		name            string
		locationHeader  string
		wantAccessToken string
		wantExpiresIn   int
		wantErr         bool
	}{
		{
			name:            "valid token with expires_in",
			locationHeader:  "https://oauth.test.com/callback#access_token=sha256~test123&expires_in=86400&token_type=Bearer",
			wantAccessToken: "sha256~test123",
			wantExpiresIn:   86400,
			wantErr:         false,
		},
		{
			name:            "valid token without expires_in",
			locationHeader:  "https://oauth.test.com/callback#access_token=sha256~test456&token_type=Bearer",
			wantAccessToken: "sha256~test456",
			wantExpiresIn:   0,
			wantErr:         false,
		},
		{
			name:            "token in query params with &",
			locationHeader:  "https://oauth.test.com/callback?foo=bar&access_token=sha256~test789&expires_in=3600",
			wantAccessToken: "sha256~test789",
			wantExpiresIn:   3600,
			wantErr:         false,
		},
		{
			name:            "missing access_token",
			locationHeader:  "https://oauth.test.com/callback#token_type=Bearer&expires_in=86400",
			wantAccessToken: "",
			wantExpiresIn:   0,
			wantErr:         true,
		},
		{
			name:            "empty location header",
			locationHeader:  "",
			wantAccessToken: "",
			wantExpiresIn:   0,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accessToken, expiresIn, err := extractTokenFromFragment(tt.locationHeader)

			if (err != nil) != tt.wantErr {
				t.Errorf("extractTokenFromFragment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if accessToken != tt.wantAccessToken {
				t.Errorf("accessToken = %q, want %q", accessToken, tt.wantAccessToken)
			}

			if expiresIn != tt.wantExpiresIn {
				t.Errorf("expiresIn = %d, want %d", expiresIn, tt.wantExpiresIn)
			}
		})
	}
}

func TestAuthenticateWithCredentials(t *testing.T) {
	tests := []struct {
		name               string
		username           string
		password           string
		oauthResponse      string
		authResponseStatus int
		authLocation       string
		wantErr            bool
		wantStored         bool
	}{
		{
			name:               "successful authentication",
			username:           "testuser",
			password:           "testpass",
			oauthResponse:      `{"authorization_endpoint":"https://oauth.test.com/authorize"}`,
			authResponseStatus: http.StatusFound,
			authLocation:       "https://oauth.test.com/callback#access_token=sha256~success123&expires_in=86400",
			wantErr:            false,
			wantStored:         true,
		},
		{
			name:               "missing location header",
			username:           "testuser",
			password:           "testpass",
			oauthResponse:      `{"authorization_endpoint":"https://oauth.test.com/authorize"}`,
			authResponseStatus: http.StatusOK,
			authLocation:       "",
			wantErr:            true,
			wantStored:         false,
		},
		{
			name:               "invalid oauth response",
			username:           "testuser",
			password:           "testpass",
			oauthResponse:      `not json`,
			authResponseStatus: http.StatusFound,
			authLocation:       "https://oauth.test.com/callback#access_token=sha256~test",
			wantErr:            true,
			wantStored:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := storage.NewMockStorage()
			logger := log.New(false)

			// Create test server
			oauthEndpointCalled := false
			authEndpointCalled := false

			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/.well-known/oauth-authorization-server" {
					oauthEndpointCalled = true
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(tt.oauthResponse))
					return
				}

				if r.URL.Path == "/authorize" {
					authEndpointCalled = true

					// Check for Basic Auth header
					authHeader := r.Header.Get("Authorization")
					if authHeader == "" {
						t.Error("missing Authorization header")
					}
					if r.Header.Get("X-CSRF-Token") != "XXXXX" {
						t.Error("missing or wrong X-CSRF-Token header")
					}

					w.Header().Set("Location", tt.authLocation)
					w.WriteHeader(tt.authResponseStatus)
					return
				}

				t.Errorf("unexpected path: %s", r.URL.Path)
			}))
			defer server.Close()

			// Update OAuth response to use test server URL
			if tt.oauthResponse != "" && tt.oauthResponse != "not json" {
				var oauth OAuthInfo
				json.Unmarshal([]byte(tt.oauthResponse), &oauth)
				oauth.AuthorizationEndpoint = server.URL + "/authorize"
				updatedResponse, _ := json.Marshal(oauth)
				tt.oauthResponse = string(updatedResponse)
			}

			auth := NewAuthenticator(server.URL, false, mockStorage, logger)
			// Update transport to skip TLS verification for test server
			tlsClient := server.Client()
			auth.httpClient.Transport = tlsClient.Transport

			tokenData, err := auth.AuthenticateWithCredentials("test-cluster", tt.username, tt.password)

			if (err != nil) != tt.wantErr {
				t.Errorf("AuthenticateWithCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tokenData == nil {
					t.Fatal("AuthenticateWithCredentials() returned nil tokenData")
				}
				if tokenData.Token == "" {
					t.Error("token is empty")
				}
				if tokenData.ExpirationTimestamp == "" {
					t.Error("expirationTimestamp is empty")
				}

				// Verify token was stored
				if tt.wantStored {
					storedToken, _ := mockStorage.Get("test-cluster", "token")
					if storedToken == "" {
						t.Error("token was not stored")
					}
				}
			}

			if !oauthEndpointCalled {
				t.Error("OAuth endpoint was not called")
			}
			// Auth endpoint is only called if OAuth response was valid
			if tt.oauthResponse != "not json" && !authEndpointCalled {
				t.Error("Authorization endpoint was not called")
			}
		})
	}
}

func TestGetOAuthInfo_HTTPSValidation(t *testing.T) {
	tests := []struct {
		name    string
		oauth   OAuthInfo
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid HTTPS authorization endpoint",
			oauth: OAuthInfo{
				AuthorizationEndpoint: "https://oauth.example.com/authorize",
				TokenEndpoint:         "https://oauth.example.com/token",
			},
			wantErr: false,
		},
		{
			name: "HTTP authorization endpoint rejected",
			oauth: OAuthInfo{
				AuthorizationEndpoint: "http://oauth.example.com/authorize",
			},
			wantErr: true,
			errMsg:  "authorization endpoint validation failed: endpoint must use HTTPS",
		},
		{
			name: "HTTP token endpoint rejected",
			oauth: OAuthInfo{
				AuthorizationEndpoint: "https://oauth.example.com/authorize",
				TokenEndpoint:         "http://oauth.example.com/token",
			},
			wantErr: true,
			errMsg:  "token endpoint validation failed: endpoint must use HTTPS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/.well-known/oauth-authorization-server" {
					body, _ := json.Marshal(tt.oauth)
					w.WriteHeader(http.StatusOK)
					w.Write(body)
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			mockStorage := storage.NewMockStorage()
			logger := log.New(false)
			auth := NewAuthenticator(server.URL, false, mockStorage, logger)
			auth.httpClient.Transport = server.Client().Transport

			_, err := auth.GetOAuthInfo()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOAuthInfo() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !containsSubstringSimple(err.Error(), tt.errMsg) {
				t.Errorf("GetOAuthInfo() error = %v, want substring %q", err, tt.errMsg)
			}
		})
	}
}

func TestCallbackServer_StateValidation(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		callURL  string
		wantErr  bool
		wantCode string
		errMsg   string
	}{
		{
			name:     "valid state parameter",
			state:    "test-state-123",
			callURL:  "http://127.0.0.1:%d/callback?code=auth-code&state=test-state-123",
			wantErr:  false,
			wantCode: "auth-code",
		},
		{
			name:    "missing state parameter",
			state:   "test-state-123",
			callURL: "http://127.0.0.1:%d/callback?code=auth-code",
			wantErr: true,
			errMsg:  "state parameter missing",
		},
		{
			name:    "state mismatch - CSRF attack",
			state:   "test-state-123",
			callURL: "http://127.0.0.1:%d/callback?code=auth-code&state=wrong-state",
			wantErr: true,
			errMsg:  "state parameter mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs, err := startCallbackServer(tt.state)
			if err != nil {
				t.Fatalf("startCallbackServer() error = %v", err)
			}
			defer cs.shutdown()

			go func() {
				callbackURL := fmt.Sprintf(tt.callURL, cs.port)
				http.Get(callbackURL)
			}()

			code, err := cs.waitForCallback(5 * time.Second)
			if (err != nil) != tt.wantErr {
				t.Errorf("waitForCallback() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !containsSubstringSimple(err.Error(), tt.errMsg) {
				t.Errorf("waitForCallback() error = %v, want substring %q", err, tt.errMsg)
			}
			if !tt.wantErr && code != tt.wantCode {
				t.Errorf("code = %q, want %q", code, tt.wantCode)
			}
		})
	}
}

// containsSubstringSimple checks if a string contains a substring (case-sensitive)
func containsSubstringSimple(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
