package auth

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/kim-truong/openshift-auth-plugin/pkg/config"
)

// OAuthFlow handles the OAuth authentication flow
type OAuthFlow struct {
	config    *config.Config
	oauthInfo *OAuthInfo
	logger    *config.Logger
}

// NewOAuthFlow creates a new OAuth flow handler
func NewOAuthFlow(cfg *config.Config, oauthInfo *OAuthInfo, logger *config.Logger) *OAuthFlow {
	return &OAuthFlow{
		config:    cfg,
		oauthInfo: oauthInfo,
		logger:    logger,
	}
}

// Authenticate performs the OAuth authentication flow
func (o *OAuthFlow) Authenticate() (string, error) {
	// Build authorization URL
	authURL := o.buildAuthorizationURL()

	// Display instructions to user
	o.displayInstructions(authURL)

	// Open browser
	if err := o.openBrowser(authURL); err != nil {
		o.logger.Debug("Failed to open browser: %v", err)
	}

	// Prompt user for token
	token, err := o.promptForToken()
	if err != nil {
		return "", err
	}

	return token, nil
}

// buildAuthorizationURL constructs the OAuth authorization URL
func (o *OAuthFlow) buildAuthorizationURL() string {
	params := url.Values{}
	params.Set("client_id", o.config.SSOClientID)
	params.Set("response_type", "token")
	params.Set("redirect_uri", fmt.Sprintf("%s/oauth/token/display", o.oauthInfo.Issuer))

	return fmt.Sprintf("%s?%s", o.oauthInfo.AuthorizationEndpoint, params.Encode())
}

// displayInstructions shows authentication instructions to the user
func (o *OAuthFlow) displayInstructions(authURL string) {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "==============================================================")
	fmt.Fprintln(os.Stderr, "OpenShift Authentication Required")
	fmt.Fprintln(os.Stderr, "==============================================================")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Opening browser to: %s\n", authURL)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Please complete authentication in your browser.")
	fmt.Fprintln(os.Stderr, "After authentication, you will receive an access token.")
	fmt.Fprintln(os.Stderr)
}

// openBrowser attempts to open the default browser
func (o *OAuthFlow) openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		// Try xdg-open first, then fallback to others
		if _, err := exec.LookPath("xdg-open"); err == nil {
			cmd = exec.Command("xdg-open", url)
		} else if _, err := exec.LookPath("wslview"); err == nil {
			cmd = exec.Command("wslview", url)
		}
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	}

	if cmd == nil {
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// promptForToken prompts the user to enter the token
func (o *OAuthFlow) promptForToken() (string, error) {
	fmt.Fprint(os.Stderr, "Enter access token: ")

	reader := bufio.NewReader(os.Stdin)
	token, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read token: %w", err)
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("no token provided")
	}

	return token, nil
}
