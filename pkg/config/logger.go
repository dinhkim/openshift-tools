package config

import (
	"fmt"
	"os"
	"time"
)

// Logger provides simple logging functionality
type Logger struct {
	debug bool
}

// NewLogger creates a new logger
func NewLogger(debug bool) *Logger {
	return &Logger{debug: debug}
}

// Debug logs a debug message if debug mode is enabled
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.debug {
		timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
		message := fmt.Sprintf(format, args...)
		fmt.Fprintf(os.Stderr, "%s %s\n", timestamp, message)
	}
}

// Info logs an informational message
func (l *Logger) Info(format string, args ...interface{}) {
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s %s\n", timestamp, message)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s ERROR: %s\n", timestamp, message)
}
