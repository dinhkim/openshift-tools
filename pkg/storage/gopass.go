package storage

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/kim-truong/openshift-auth-plugin/pkg/config"
)

// GopassStorage implements Storage using gopass
type GopassStorage struct {
	clusterName string
	logger      *config.Logger
}

// NewGopassStorage creates a new gopass storage backend
func NewGopassStorage(clusterName string, logger *config.Logger) (*GopassStorage, error) {
	// Check if gopass is available
	if _, err := exec.LookPath("gopass"); err != nil {
		return nil, fmt.Errorf("gopass not found in PATH")
	}

	return &GopassStorage{
		clusterName: clusterName,
		logger:      logger,
	}, nil
}

// GetToken retrieves the token from gopass
func (g *GopassStorage) GetToken() (*TokenData, error) {
	g.logger.Debug("Retrieving token from gopass")

	cmd := exec.Command("gopass", "show", "-o", g.clusterName, "token")
	output, err := cmd.Output()
	if err != nil {
		g.logger.Debug("No token found in gopass")
		return nil, nil
	}

	data := strings.TrimSpace(string(output))
	if data == "" {
		return nil, nil
	}

	tokenData, err := unmarshalTokenData(data)
	if err != nil {
		g.logger.Debug("Failed to parse token data: %v", err)
		return nil, nil
	}

	return tokenData, nil
}

// StoreToken stores the token in gopass
func (g *GopassStorage) StoreToken(token string, expiry time.Time) error {
	g.logger.Debug("Storing token in gopass")

	data, err := marshalTokenData(token, expiry)
	if err != nil {
		return fmt.Errorf("failed to marshal token data: %w", err)
	}

	cmd := exec.Command("gopass", "insert", "-f", g.clusterName, "token")
	cmd.Stdin = bytes.NewBufferString(data)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to store token in gopass: %w", err)
	}

	g.logger.Debug("Token stored successfully")
	return nil
}
