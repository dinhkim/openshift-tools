package storage

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// GopassStorage implements Storage using gopass
type GopassStorage struct{}

// NewGopassStorage creates a new GopassStorage instance
func NewGopassStorage() *GopassStorage {
	return &GopassStorage{}
}

// Get retrieves a value from gopass
func (g *GopassStorage) Get(cluster, key string) (string, error) {
	cmd := exec.Command("gopass", "show", "-o", cluster, key)

	output, err := cmd.Output()
	if err != nil {
		// Not found or other error - return empty string, no error
		// This matches the shell script behavior (returns empty on error)
		return "", nil
	}

	return strings.TrimSpace(string(output)), nil
}

// Store saves a value to gopass
func (g *GopassStorage) Store(cluster, key, value string) error {
	cmd := exec.Command("gopass", "insert", "-f", cluster, key)
	cmd.Stdin = bytes.NewBufferString(value)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to store in gopass: %w", err)
	}

	return nil
}
