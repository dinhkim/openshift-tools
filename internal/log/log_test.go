package log

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		debug       bool
		wantEnabled bool
	}{
		{
			name:        "debug enabled",
			debug:       true,
			wantEnabled: true,
		},
		{
			name:        "debug disabled",
			debug:       false,
			wantEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.debug)
			if logger == nil {
				t.Fatal("New() returned nil")
			}
			if logger.IsEnabled() != tt.wantEnabled {
				t.Errorf("New(%v).IsEnabled() = %v, want %v", tt.debug, logger.IsEnabled(), tt.wantEnabled)
			}
		})
	}
}

func TestLogger_Debug(t *testing.T) {
	tests := []struct {
		name        string
		debug       bool
		format      string
		args        []interface{}
		wantOutput  bool
		wantContain string
	}{
		{
			name:        "debug enabled - simple message",
			debug:       true,
			format:      "test message",
			args:        nil,
			wantOutput:  true,
			wantContain: "test message",
		},
		{
			name:        "debug enabled - formatted message",
			debug:       true,
			format:      "user: %s, count: %d",
			args:        []interface{}{"alice", 42},
			wantOutput:  true,
			wantContain: "user: alice, count: 42",
		},
		{
			name:        "debug disabled - should not output",
			debug:       false,
			format:      "should not see this",
			args:        nil,
			wantOutput:  false,
			wantContain: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			logger := New(tt.debug)
			logger.Debug(tt.format, tt.args...)

			// Restore stderr and read output
			w.Close()
			os.Stderr = oldStderr
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if tt.wantOutput {
				if output == "" {
					t.Error("expected output but got none")
				}
				if !strings.Contains(output, tt.wantContain) {
					t.Errorf("output does not contain expected string.\nGot: %q\nWant to contain: %q", output, tt.wantContain)
				}
				// Check for ISO 8601 timestamp format (YYYY-MM-DDTHH:MM:SSZ)
				if !strings.Contains(output, "T") || !strings.Contains(output, "Z") {
					t.Errorf("output does not contain ISO 8601 timestamp.\nGot: %q", output)
				}
			} else {
				if output != "" {
					t.Errorf("expected no output but got: %q", output)
				}
			}
		})
	}
}

func TestLogger_IsEnabled(t *testing.T) {
	logger := New(true)
	if !logger.IsEnabled() {
		t.Error("IsEnabled() = false, want true")
	}

	logger = New(false)
	if logger.IsEnabled() {
		t.Error("IsEnabled() = true, want false")
	}
}
