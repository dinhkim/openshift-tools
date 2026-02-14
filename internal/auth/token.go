package auth

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// ValidateToken checks if a token is valid by checking expiry and making an API call
func (a *Authenticator) ValidateToken(token, expirationTimestamp string) error {
	a.logger.Debug("Validating token...")

	// First check if token is expired by timestamp
	if expirationTimestamp != "" {
		expiryTime, err := time.Parse("2006-01-02T15:04:05Z", expirationTimestamp)
		if err != nil {
			a.logger.Debug("Warning: failed to parse expiry timestamp: %v", err)
		} else {
			currentTime := time.Now().UTC()
			a.logger.Debug("Current timestamp: %s", currentTime.Format("2006-01-02T15:04:05Z"))
			a.logger.Debug("Expiry timestamp: %s", expirationTimestamp)

			if currentTime.Before(expiryTime) {
				a.logger.Debug("Token is not expired by timestamp, considering it valid")
				return nil
			}
			a.logger.Debug("Token is expired by timestamp")
		}
	}

	// If expired or no expiry timestamp, validate with API call
	userURL := fmt.Sprintf("%s/apis/user.openshift.io/v1/users/~", a.serverURL)
	a.logger.Debug("GET %s", userURL)

	req, err := http.NewRequest("GET", userURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	// Read and discard body to allow connection reuse
	_, _ = io.Copy(io.Discard, resp.Body)

	a.logger.Debug("Response %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token validation failed (HTTP %d)", resp.StatusCode)
	}

	return nil
}
