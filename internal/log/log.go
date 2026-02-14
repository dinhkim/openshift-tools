package log

import (
	"fmt"
	"os"
	"time"
)

// Logger provides debug logging to stderr with ISO 8601 timestamps
type Logger struct {
	enabled bool
}

// New creates a new Logger instance
func New(debug bool) *Logger {
	return &Logger{enabled: debug}
}

// Debug writes a debug message to stderr if logging is enabled
func (l *Logger) Debug(format string, args ...interface{}) {
	if !l.enabled {
		return
	}
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s %s\n", timestamp, message)
}

// IsEnabled returns whether debug logging is enabled
func (l *Logger) IsEnabled() bool {
	return l.enabled
}
