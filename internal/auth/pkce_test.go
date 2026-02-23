package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dinhkim/openshift-tools/internal/log"
	"github.com/dinhkim/openshift-tools/internal/storage"
)

func TestGeneratePKCE(t *testing.T) {
	pkce, err := generatePKCE()
	if err != nil {
		t.Fatalf("generatePKCE() error = %v", err)
	}

	if pkce == nil {
		t.Fatal("generatePKCE() returned nil")
	}

	// code_verifier should be 43 characters (base64url of 32 bytes, no padding)
	if len(pkce.CodeVerifier) != 43 {
		t.Errorf("CodeVerifier length = %d, want 43", len(pkce.CodeVerifier))
	}

	// code_challenge should be 43 characters (base64url of SHA-256 hash, no padding)
	if len(pkce.CodeChallenge) != 43 {
		t.Errorf("CodeChallenge length = %d, want 43", len(pkce.CodeChallenge))
	}

	// Verify that code_challenge = BASE64URL(SHA256(code_verifier))
	hash := sha256.Sum256([]byte(pkce.CodeVerifier))
	expectedChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
	if pkce.CodeChallenge != expectedChallenge {
		t.Errorf("CodeChallenge does not match SHA256 of CodeVerifier.\nGot:  %s\nWant: %s", pkce.CodeChallenge, expectedChallenge)
	}
}

func TestGeneratePKCE_Uniqueness(t *testing.T) {
	// Generate multiple PKCE pairs and verify they are unique
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		pkce, err := generatePKCE()
		if err != nil {
			t.Fatalf("generatePKCE() iteration %d error = %v", i, err)
		}
		if seen[pkce.CodeVerifier] {
			t.Fatalf("duplicate CodeVerifier generated at iteration %d", i)
		}
		seen[pkce.CodeVerifier] = true
	}
}

func TestGeneratePKCE_Base64URLCharacters(t *testing.T) {
	pkce, err := generatePKCE()
	if err != nil {
		t.Fatalf("generatePKCE() error = %v", err)
	}

	// Verify code_verifier only contains base64url characters (no padding)
	for _, c := range pkce.CodeVerifier {
		if !isBase64URLChar(c) {
			t.Errorf("CodeVerifier contains invalid character: %c", c)
		}
	}

	// Verify code_challenge only contains base64url characters (no padding)
	for _, c := range pkce.CodeChallenge {
		if !isBase64URLChar(c) {
			t.Errorf("CodeChallenge contains invalid character: %c", c)
		}
	}
}

func isBase64URLChar(c rune) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') || c == '-' || c == '_'
}

func TestExchangeCodeForToken(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		wantErr        bool
		wantToken      string
	}{
		{
			name:           "successful token exchange",
			responseStatus: http.StatusOK,
			responseBody:   `{"access_token":"sha256~sso-token-123","token_type":"Bearer","expires_in":86400}`,
			wantErr:        false,
			wantToken:      "sha256~sso-token-123",
		},
		{
			name:           "successful exchange without expires_in",
			responseStatus: http.StatusOK,
			responseBody:   `{"access_token":"sha256~no-expiry","token_type":"Bearer"}`,
			wantErr:        false,
			wantToken:      "sha256~no-expiry",
		},
		{
			name:           "server error",
			responseStatus: http.StatusBadRequest,
			responseBody:   `{"error":"invalid_grant","error_description":"code expired"}`,
			wantErr:        true,
		},
		{
			name:           "empty access token",
			responseStatus: http.StatusOK,
			responseBody:   `{"access_token":"","token_type":"Bearer"}`,
			wantErr:        true,
		},
		{
			name:           "invalid json response",
			responseStatus: http.StatusOK,
			responseBody:   `not json`,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and content type
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
					t.Errorf("expected Content-Type application/x-www-form-urlencoded, got %s", r.Header.Get("Content-Type"))
				}

				// Verify form values
				r.ParseForm()
				if r.FormValue("grant_type") != "authorization_code" {
					t.Errorf("grant_type = %q, want authorization_code", r.FormValue("grant_type"))
				}
				if r.FormValue("code") != "test-auth-code" {
					t.Errorf("code = %q, want test-auth-code", r.FormValue("code"))
				}
				if r.FormValue("client_id") != "openshift-cli-client" {
					t.Errorf("client_id = %q, want openshift-cli-client", r.FormValue("client_id"))
				}
				if r.FormValue("code_verifier") != "test-verifier" {
					t.Errorf("code_verifier = %q, want test-verifier", r.FormValue("code_verifier"))
				}
				if r.FormValue("redirect_uri") != "http://127.0.0.1:12345/callback" {
					t.Errorf("redirect_uri = %q, want http://127.0.0.1:12345/callback", r.FormValue("redirect_uri"))
				}

				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			mockStorage := &storage.MockStorage{}
			logger := log.New(false)
			auth := NewAuthenticator(server.URL, false, mockStorage, logger)

			tokenData, err := auth.exchangeCodeForToken(
				server.URL,
				"test-auth-code",
				"http://127.0.0.1:12345/callback",
				"test-verifier",
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("exchangeCodeForToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tokenData == nil {
					t.Fatal("exchangeCodeForToken() returned nil tokenData")
				}
				if tokenData.Token != tt.wantToken {
					t.Errorf("Token = %q, want %q", tokenData.Token, tt.wantToken)
				}
				if tokenData.ExpirationTimestamp == "" {
					t.Error("ExpirationTimestamp is empty")
				}
			}
		})
	}
}

func TestExchangeCodeForToken_ExpiryCalculation(t *testing.T) {
	tests := []struct {
		name        string
		expiresIn   int
		wantDefault bool
	}{
		{
			name:        "with expires_in",
			expiresIn:   3600,
			wantDefault: false,
		},
		{
			name:        "without expires_in defaults to 24h",
			expiresIn:   0,
			wantDefault: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tokenResponse{
				AccessToken: "sha256~test",
				ExpiresIn:   tt.expiresIn,
				TokenType:   "Bearer",
			}
			body, _ := json.Marshal(resp)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write(body)
			}))
			defer server.Close()

			mockStorage := &storage.MockStorage{}
			logger := log.New(false)
			auth := NewAuthenticator(server.URL, false, mockStorage, logger)

			tokenData, err := auth.exchangeCodeForToken(server.URL, "code", "http://localhost/cb", "verifier")
			if err != nil {
				t.Fatalf("exchangeCodeForToken() error = %v", err)
			}

			if tokenData.ExpirationTimestamp == "" {
				t.Error("ExpirationTimestamp should be set")
			}
		})
	}
}

func TestExchangeCodeForToken_TokenTypeValidation(t *testing.T) {
	tests := []struct {
		name      string
		tokenType string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid Bearer token type",
			tokenType: "Bearer",
			wantErr:   false,
		},
		{
			name:      "unsupported token type",
			tokenType: "DPoP",
			wantErr:   true,
			errMsg:    "unsupported token type",
		},
		{
			name:      "missing token type",
			tokenType: "",
			wantErr:   true,
			errMsg:    "unsupported token type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tokenResponse{
				AccessToken: "sha256~test",
				ExpiresIn:   3600,
				TokenType:   tt.tokenType,
			}
			body, _ := json.Marshal(resp)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write(body)
			}))
			defer server.Close()

			mockStorage := &storage.MockStorage{}
			logger := log.New(false)
			auth := NewAuthenticator(server.URL, false, mockStorage, logger)

			tokenData, err := auth.exchangeCodeForToken(server.URL, "code", "http://localhost/cb", "verifier")
			if (err != nil) != tt.wantErr {
				t.Errorf("exchangeCodeForToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !containsTokenTypeSubstring(err.Error(), tt.errMsg) {
				t.Errorf("exchangeCodeForToken() error = %v, want substring %q", err, tt.errMsg)
			}
			if !tt.wantErr && tokenData == nil {
				t.Error("tokenData should not be nil when wantErr is false")
			}
		})
	}
}

// containsTokenTypeSubstring checks if a string contains a substring
func containsTokenTypeSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
