package auth

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kim-truong/openshift-auth-plugin/pkg/config"
)

// OAuthInfo represents OAuth server metadata
type OAuthInfo struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
}

// OpenshiftClient handles HTTP requests to OpenShift API
type OpenshiftClient struct {
	serverURL  string
	httpClient *http.Client
	logger     *config.Logger
}

// NewOpenshiftClient creates a new OpenShift API client
func NewOpenshiftClient(serverURL string, verifySSL bool, logger *config.Logger) *OpenshiftClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !verifySSL,
		},
	}

	return &OpenshiftClient{
		serverURL: serverURL,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		logger: logger,
	}
}

// GetOAuthInfo retrieves OAuth server metadata
func (c *OpenshiftClient) GetOAuthInfo() (*OAuthInfo, error) {
	url := fmt.Sprintf("%s/.well-known/oauth-authorization-server", c.serverURL)
	c.logger.Debug("Fetching OAuth info from: %s", url)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OAuth endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OAuth endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var oauthInfo OAuthInfo
	if err := json.Unmarshal(body, &oauthInfo); err != nil {
		return nil, fmt.Errorf("failed to parse OAuth info: %w", err)
	}

	return &oauthInfo, nil
}

// ValidateToken validates a token by making a request to the OpenShift API
func (c *OpenshiftClient) ValidateToken(token string) error {
	url := fmt.Sprintf("%s/apis/user.openshift.io/v1/users/~", c.serverURL)
	c.logger.Debug("Validating token with: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token validation failed with status %d", resp.StatusCode)
	}

	c.logger.Debug("Token validation successful")
	return nil
}
