package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dinhkim/openshift-tools/internal/log"
	"github.com/dinhkim/openshift-tools/internal/storage"
)

func TestValidateToken(t *testing.T) {
	tests := []struct {
		name                string
		token               string
		expirationTimestamp string
		apiResponseStatus   int
		wantErr             bool
		skipAPICall         bool
	}{
		{
			name:                "valid token not expired by timestamp",
			token:               "sha256~valid123",
			expirationTimestamp: time.Now().UTC().Add(1 * time.Hour).Format("2006-01-02T15:04:05Z"),
			apiResponseStatus:   http.StatusOK,
			wantErr:             false,
			skipAPICall:         true, // Should not make API call if not expired
		},
		{
			name:                "expired token but valid by API",
			token:               "sha256~expired123",
			expirationTimestamp: time.Now().UTC().Add(-1 * time.Hour).Format("2006-01-02T15:04:05Z"),
			apiResponseStatus:   http.StatusOK,
			wantErr:             false,
			skipAPICall:         false,
		},
		{
			name:                "expired token and invalid by API",
			token:               "sha256~invalid123",
			expirationTimestamp: time.Now().UTC().Add(-1 * time.Hour).Format("2006-01-02T15:04:05Z"),
			apiResponseStatus:   http.StatusUnauthorized,
			wantErr:             true,
			skipAPICall:         false,
		},
		{
			name:                "no expiration timestamp - valid by API",
			token:               "sha256~noexpiry123",
			expirationTimestamp: "",
			apiResponseStatus:   http.StatusOK,
			wantErr:             false,
			skipAPICall:         false,
		},
		{
			name:                "no expiration timestamp - invalid by API",
			token:               "sha256~noexpiry456",
			expirationTimestamp: "",
			apiResponseStatus:   http.StatusForbidden,
			wantErr:             true,
			skipAPICall:         false,
		},
		{
			name:                "invalid timestamp format - fallback to API",
			token:               "sha256~badformat",
			expirationTimestamp: "not-a-timestamp",
			apiResponseStatus:   http.StatusOK,
			wantErr:             false,
			skipAPICall:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiCalled := false

			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				apiCalled = true

				if r.URL.Path != "/apis/user.openshift.io/v1/users/~" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}

				// Check Authorization header
				authHeader := r.Header.Get("Authorization")
				expectedAuth := "Bearer " + tt.token
				if authHeader != expectedAuth {
					t.Errorf("Authorization header = %q, want %q", authHeader, expectedAuth)
				}

				// Check Accept header
				if r.Header.Get("Accept") != "application/json" {
					t.Error("missing or wrong Accept header")
				}

				w.WriteHeader(tt.apiResponseStatus)
				w.Write([]byte(`{"kind":"User","metadata":{"name":"testuser"}}`))
			}))
			defer server.Close()

			mockStorage := &storage.MockStorage{}
			logger := log.New(false)
			auth := NewAuthenticator(server.URL, false, mockStorage, logger)

			err := auth.ValidateToken(tt.token, tt.expirationTimestamp)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.skipAPICall && apiCalled {
				t.Error("API was called but should have been skipped (token not expired)")
			}

			if !tt.skipAPICall && !apiCalled {
				t.Error("API was not called but should have been")
			}
		})
	}
}

func TestValidateTokenWithDebugLogging(t *testing.T) {
	// Test that debug logging works during validation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"kind":"User"}`))
	}))
	defer server.Close()

	mockStorage := &storage.MockStorage{}
	logger := log.New(true) // Enable debug logging
	auth := NewAuthenticator(server.URL, false, mockStorage, logger)

	futureTime := time.Now().UTC().Add(1 * time.Hour).Format("2006-01-02T15:04:05Z")
	err := auth.ValidateToken("sha256~test", futureTime)

	if err != nil {
		t.Errorf("ValidateToken() error = %v", err)
	}
}
