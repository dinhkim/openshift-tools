package storage

import (
	"testing"
	"time"
)

func TestMarshalTokenData(t *testing.T) {
	token := "test-token-12345"
	expiry := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)

	jsonStr, err := marshalTokenData(token, expiry)
	if err != nil {
		t.Fatalf("marshalTokenData failed: %v", err)
	}

	if jsonStr == "" {
		t.Error("Expected non-empty JSON string")
	}

	// Verify it contains the token
	if !contains(jsonStr, token) {
		t.Errorf("JSON does not contain token: %s", jsonStr)
	}
}

func TestUnmarshalTokenData(t *testing.T) {
	jsonStr := `{"token":"test-token","expirationTimestamp":"2026-02-14T12:00:00Z"}`

	tokenData, err := unmarshalTokenData(jsonStr)
	if err != nil {
		t.Fatalf("unmarshalTokenData failed: %v", err)
	}

	if tokenData.Token != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", tokenData.Token)
	}

	expectedTime := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	if !tokenData.ExpirationTimestamp.Equal(expectedTime) {
		t.Errorf("Expected time %v, got %v", expectedTime, tokenData.ExpirationTimestamp)
	}
}

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	originalToken := "test-token-roundtrip"
	originalExpiry := time.Now().Add(24 * time.Hour).UTC()

	// Marshal
	jsonStr, err := marshalTokenData(originalToken, originalExpiry)
	if err != nil {
		t.Fatalf("marshalTokenData failed: %v", err)
	}

	// Unmarshal
	tokenData, err := unmarshalTokenData(jsonStr)
	if err != nil {
		t.Fatalf("unmarshalTokenData failed: %v", err)
	}

	// Verify
	if tokenData.Token != originalToken {
		t.Errorf("Token mismatch: expected '%s', got '%s'", originalToken, tokenData.Token)
	}

	// Time comparison with some tolerance (nanoseconds might differ in JSON)
	timeDiff := tokenData.ExpirationTimestamp.Sub(originalExpiry)
	if timeDiff > time.Second || timeDiff < -time.Second {
		t.Errorf("Time mismatch: expected %v, got %v (diff: %v)",
			originalExpiry, tokenData.ExpirationTimestamp, timeDiff)
	}
}

func TestUnmarshalTokenData_InvalidJSON(t *testing.T) {
	invalidJSON := `{"token":"test", invalid json}`

	_, err := unmarshalTokenData(invalidJSON)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestUnmarshalTokenData_EmptyString(t *testing.T) {
	_, err := unmarshalTokenData("")
	if err == nil {
		t.Error("Expected error for empty string, got nil")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr))))
}
