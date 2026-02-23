package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// pkceParams holds the PKCE code verifier and challenge pair
type pkceParams struct {
	CodeVerifier  string
	CodeChallenge string
}

// generatePKCE generates a PKCE code verifier and code challenge (S256)
// per RFC 7636
func generatePKCE() (*pkceParams, error) {
	// Generate 32 cryptographically random bytes for code_verifier
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes for PKCE: %w", err)
	}

	// Base64url-encode without padding to create code_verifier (43 chars)
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// Compute code_challenge = BASE64URL(SHA256(ASCII(code_verifier)))
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return &pkceParams{
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
	}, nil
}

// tokenResponse represents the JSON response from the OAuth token endpoint
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// exchangeCodeForToken exchanges an authorization code for an access token
// using the OAuth token endpoint with PKCE verification
func (a *Authenticator) exchangeCodeForToken(tokenEndpoint, code, redirectURI, codeVerifier string) (*TokenData, error) {
	a.logger.Debug("Exchanging authorization code for access token...")

	// Build form-encoded body
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", "openshift-cli-client")
	data.Set("code_verifier", codeVerifier)

	req, err := http.NewRequest("POST", tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token exchange request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Use a separate client that follows redirects for the token exchange
	tokenClient := &http.Client{
		Transport: a.httpClient.Transport,
	}

	resp, err := tokenClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		a.logger.Debug("Token exchange failed with status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("token exchange failed (HTTP %d)", resp.StatusCode)
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("token response did not contain an access token")
	}

	// Calculate expiry timestamp
	var expiryTimestamp string
	if tokenResp.ExpiresIn > 0 {
		expiryTime := time.Now().UTC().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		expiryTimestamp = expiryTime.Format("2006-01-02T15:04:05Z")
	} else {
		// Default to 24 hours if not provided
		expiryTime := time.Now().UTC().Add(24 * time.Hour)
		expiryTimestamp = expiryTime.Format("2006-01-02T15:04:05Z")
	}

	a.logger.Debug("Token exchange successful, expires at %s", expiryTimestamp)

	return &TokenData{
		Token:               tokenResp.AccessToken,
		ExpirationTimestamp: expiryTimestamp,
	}, nil
}
