package auth

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/dinhkim/openshift-tools/internal/log"
	"github.com/dinhkim/openshift-tools/internal/storage"
)

// OAuthInfo represents the OAuth authorization server metadata
type OAuthInfo struct {
	AuthorizationEndpoint         string   `json:"authorization_endpoint"`
	TokenEndpoint                 string   `json:"token_endpoint"`
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
}

// TokenData represents stored token information
type TokenData struct {
	Token               string `json:"token"`
	ExpirationTimestamp string `json:"expirationTimestamp"`
}

// CredentialsData represents stored credentials
type CredentialsData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Authenticator handles OpenShift authentication
type Authenticator struct {
	serverURL  string
	verifySSL  bool
	storage    storage.Storage
	httpClient *http.Client
	logger     *log.Logger
}

// NewAuthenticator creates a new Authenticator instance
func NewAuthenticator(serverURL string, verifySSL bool, store storage.Storage, logger *log.Logger) *Authenticator {
	// Create HTTP client with custom transport for SSL verification
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !verifySSL,
		},
	}

	return &Authenticator{
		serverURL: serverURL,
		verifySSL: verifySSL,
		storage:   store,
		httpClient: &http.Client{
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Don't follow redirects - we need to extract the token from the Location header
				return http.ErrUseLastResponse
			},
		},
		logger: logger,
	}
}

// GetOAuthInfo fetches the OAuth authorization server metadata
func (a *Authenticator) GetOAuthInfo() (*OAuthInfo, error) {
	oauthURL := fmt.Sprintf("%s/.well-known/oauth-authorization-server", a.serverURL)
	a.logger.Debug("Fetching OAuth info from %s", oauthURL)

	resp, err := a.httpClient.Get(oauthURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OAuth endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OAuth response: %w", err)
	}

	var info OAuthInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("invalid JSON response from OAuth endpoint: %w", err)
	}

	if info.AuthorizationEndpoint == "" {
		return nil, fmt.Errorf("authorization endpoint not found in OAuth info")
	}

	return &info, nil
}

// AuthenticateWithCredentials performs OAuth authentication using username and password
func (a *Authenticator) AuthenticateWithCredentials(clusterName, username, password string) (*TokenData, error) {
	a.logger.Debug("Authenticating with credentials for cluster %s", clusterName)

	// Get OAuth information
	oauthInfo, err := a.GetOAuthInfo()
	if err != nil {
		return nil, err
	}

	a.logger.Debug("Authorization endpoint: %s", oauthInfo.AuthorizationEndpoint)

	// Build authorization URL
	authURL, err := url.Parse(oauthInfo.AuthorizationEndpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid authorization endpoint URL: %w", err)
	}

	query := authURL.Query()
	query.Set("client_id", "openshift-challenging-client")
	query.Set("response_type", "token")
	authURL.RawQuery = query.Encode()

	// Encode credentials for Basic Auth
	credentials := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))

	// Create request
	req, err := http.NewRequest("GET", authURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", credentials))
	req.Header.Set("X-CSRF-Token", "XXXXX")

	// Send request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to authorization endpoint: %w", err)
	}
	defer resp.Body.Close()

	// Extract access token from Location header
	locationHeader := resp.Header.Get("Location")
	if locationHeader == "" {
		return nil, fmt.Errorf("authentication failed: no Location header in response")
	}

	a.logger.Debug("Location header: %s", locationHeader)

	// Parse the fragment to extract access_token and expires_in
	accessToken, expiresIn, err := extractTokenFromFragment(locationHeader)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Calculate expiry timestamp
	var expiryTimestamp string
	if expiresIn > 0 {
		expiryTime := time.Now().UTC().Add(time.Duration(expiresIn) * time.Second)
		expiryTimestamp = expiryTime.Format("2006-01-02T15:04:05Z")
	} else {
		// Default to 24 hours if not provided
		expiryTime := time.Now().UTC().Add(24 * time.Hour)
		expiryTimestamp = expiryTime.Format("2006-01-02T15:04:05Z")
	}

	tokenData := &TokenData{
		Token:               accessToken,
		ExpirationTimestamp: expiryTimestamp,
	}

	// Store token in secret store
	tokenJSON, err := json.Marshal(tokenData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal token data: %w", err)
	}

	if err := a.storage.Store(clusterName, "token", string(tokenJSON)); err != nil {
		a.logger.Debug("Warning: failed to store token: %v", err)
		// Don't fail if storage fails - we can still return the token
	}

	a.logger.Debug("Authentication successful, token expires at %s", expiryTimestamp)

	return tokenData, nil
}

// AuthenticateWithSSO performs OAuth authentication using the PKCE flow
// (Authorization Code + Proof Key for Code Exchange) with openshift-cli-client.
// This opens a browser for the user to authenticate via SSO (Azure AD, Okta, etc.)
func (a *Authenticator) AuthenticateWithSSO(clusterName string, timeoutSeconds int) (*TokenData, error) {
	a.logger.Debug("Starting SSO authentication for cluster %s", clusterName)

	// Get OAuth information
	oauthInfo, err := a.GetOAuthInfo()
	if err != nil {
		return nil, fmt.Errorf("SSO authentication failed: %w", err)
	}

	if oauthInfo.TokenEndpoint == "" {
		return nil, fmt.Errorf("SSO authentication not available: token endpoint not found in OAuth metadata")
	}

	a.logger.Debug("Authorization endpoint: %s", oauthInfo.AuthorizationEndpoint)
	a.logger.Debug("Token endpoint: %s", oauthInfo.TokenEndpoint)

	// Generate PKCE parameters
	pkce, err := generatePKCE()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PKCE parameters: %w", err)
	}
	a.logger.Debug("Generated PKCE code challenge")

	// Start local callback server
	callbackSrv, err := startCallbackServer()
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}
	defer callbackSrv.shutdown()

	a.logger.Debug("Callback server listening on %s", callbackSrv.redirectURI())

	// Build authorization URL
	authURL, err := url.Parse(oauthInfo.AuthorizationEndpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid authorization endpoint URL: %w", err)
	}

	query := authURL.Query()
	query.Set("client_id", "openshift-cli-client")
	query.Set("response_type", "code")
	query.Set("redirect_uri", callbackSrv.redirectURI())
	query.Set("code_challenge", pkce.CodeChallenge)
	query.Set("code_challenge_method", "S256")
	authURL.RawQuery = query.Encode()

	authURLStr := authURL.String()

	// Open browser for authentication
	a.logger.Debug("Opening browser for SSO authentication...")
	if err := openBrowser(authURLStr); err != nil {
		a.logger.Debug("Failed to open browser: %v", err)
		fmt.Fprintf(os.Stderr, "Could not open browser automatically.\nOpen this URL in your browser to authenticate:\n\n%s\n\n", authURLStr)
	} else {
		fmt.Fprintf(os.Stderr, "Opening browser for SSO authentication...\nIf the browser does not open, visit:\n\n%s\n\n", authURLStr)
	}

	timeout := time.Duration(timeoutSeconds) * time.Second
	fmt.Fprintf(os.Stderr, "Waiting for authentication (timeout: %v)...\n", timeout)

	// Wait for the authorization code
	code, err := callbackSrv.waitForCallback(timeout)
	if err != nil {
		return nil, fmt.Errorf("SSO authentication failed: %w", err)
	}

	a.logger.Debug("Received authorization code, exchanging for token...")

	// Exchange authorization code for access token
	tokenData, err := a.exchangeCodeForToken(oauthInfo.TokenEndpoint, code, callbackSrv.redirectURI(), pkce.CodeVerifier)
	if err != nil {
		return nil, fmt.Errorf("SSO token exchange failed: %w", err)
	}

	// Store token in secret store
	tokenJSON, err := json.Marshal(tokenData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal token data: %w", err)
	}

	if err := a.storage.Store(clusterName, "token", string(tokenJSON)); err != nil {
		a.logger.Debug("Warning: failed to store token: %v", err)
	}

	a.logger.Debug("SSO authentication successful, token expires at %s", tokenData.ExpirationTimestamp)

	return tokenData, nil
}

// extractTokenFromFragment extracts access_token and expires_in from the OAuth redirect fragment
func extractTokenFromFragment(locationHeader string) (accessToken string, expiresIn int, err error) {
	// The token is in the URL fragment after #
	// Example: https://...#access_token=sha256~...&expires_in=86400&...

	// Extract fragment part
	fragmentIdx := regexp.MustCompile(`[#&]access_token=([^&]+)`)
	matches := fragmentIdx.FindStringSubmatch(locationHeader)
	if len(matches) < 2 {
		return "", 0, fmt.Errorf("access_token not found in response")
	}
	accessToken = matches[1]

	// Extract expires_in if present
	expiresIdx := regexp.MustCompile(`[#&]expires_in=([^&]+)`)
	expiresMatches := expiresIdx.FindStringSubmatch(locationHeader)
	if len(expiresMatches) >= 2 {
		expiresIn, _ = strconv.Atoi(expiresMatches[1])
	}

	return accessToken, expiresIn, nil
}
