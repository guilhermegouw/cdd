// Package debug provides development logging for CDD CLI.
package debug

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	enabled bool
	logFile *os.File
	mu      sync.Mutex
	logPath string
)

// noisePatterns are log patterns to filter out (too frequent, not useful for debugging).
var noisePatterns = []string{
	"cursor.BlinkMsg",
	"cursor.blinkCanceled",
	"cursor.initialBlinkMsg",
	"ViewportBefore:",
	"ViewportAfter:",
	"spinner.TickMsg",
}

// Enable turns on debug logging to the specified file.
func Enable(path string) error {
	mu.Lock()
	defer mu.Unlock()

	if enabled {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil { //nolint:gosec // Debug log dir needs to be readable
		return fmt.Errorf("creating log directory: %w", err)
	}

	// Open log file (append mode)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) //nolint:gosec // G304: path is user-controlled debug config
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}

	logFile = f
	logPath = path
	enabled = true

	// Write session header directly (can't call Log() - would deadlock)
	timestamp := time.Now().Format("15:04:05.000")
	header := fmt.Sprintf("[%s] === CDD Debug Session Started ===\n", timestamp)
	header += fmt.Sprintf("[%s] Time: %s\n", timestamp, time.Now().Format(time.RFC3339))
	header += fmt.Sprintf("[%s] Log file: %s\n", timestamp, path)
	header += fmt.Sprintf("[%s] ================================\n", timestamp)
	_, _ = logFile.WriteString(header) //nolint:errcheck // Intentionally ignore errors in debug logging
	_ = logFile.Sync()                 //nolint:errcheck // Flush immediately

	return nil
}

// Disable turns off debug logging and closes the file.
func Disable() {
	mu.Lock()
	defer mu.Unlock()

	if !enabled {
		return
	}

	if logFile != nil {
		_ = logFile.Close() //nolint:errcheck // Intentionally ignore error on close
		logFile = nil
	}
	enabled = false
}

// IsEnabled returns whether debug logging is enabled.
func IsEnabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return enabled
}

// isNoise checks if a message matches a noise pattern.
func isNoise(msg string) bool {
	for _, pattern := range noisePatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}

// Log writes a debug message if logging is enabled.
func Log(format string, args ...any) {
	mu.Lock()
	defer mu.Unlock()

	if !enabled || logFile == nil {
		return
	}

	msg := fmt.Sprintf(format, args...)

	// Filter out noisy messages
	if isNoise(msg) {
		return
	}

	timestamp := time.Now().Format("15:04:05.000")
	line := fmt.Sprintf("[%s] %s\n", timestamp, msg)

	_, _ = logFile.WriteString(line) //nolint:errcheck // Intentionally ignore errors in debug logging
	_ = logFile.Sync()               //nolint:errcheck // Flush immediately for real-time viewing
}

// LogPath returns the path to the log file.
func LogPath() string {
	mu.Lock()
	defer mu.Unlock()
	return logPath
}

// Event logs a TUI event with component context.
func Event(component, eventType, details string) {
	Log("[%s] %s: %s", component, eventType, details)
}

// Error logs an error with context.
func Error(component string, err error, context string) {
	Log("[%s] ERROR: %s - %v", component, context, err)
}

// API logs an API request/response (high priority, never filtered).
func API(method, url string, statusCode int, details string) {
	mu.Lock()
	defer mu.Unlock()

	if !enabled || logFile == nil {
		return
	}

	timestamp := time.Now().Format("15:04:05.000")
	line := fmt.Sprintf("[%s] [API] %s %s -> %d: %s\n", timestamp, method, url, statusCode, details)

	_, _ = logFile.WriteString(line) //nolint:errcheck // Intentionally ignore errors in debug logging
	_ = logFile.Sync()               //nolint:errcheck // Flush immediately
}

// Auth logs authentication-related events (high priority, never filtered).
func Auth(event, details string) {
	mu.Lock()
	defer mu.Unlock()

	if !enabled || logFile == nil {
		return
	}

	timestamp := time.Now().Format("15:04:05.000")
	line := fmt.Sprintf("[%s] [AUTH] %s: %s\n", timestamp, event, details)

	_, _ = logFile.WriteString(line) //nolint:errcheck // Intentionally ignore errors in debug logging
	_ = logFile.Sync()               //nolint:errcheck // Flush immediately
}

// Token logs token-related events (refresh, expiry, etc).
func Token(event, details string) {
	mu.Lock()
	defer mu.Unlock()

	if !enabled || logFile == nil {
		return
	}

	timestamp := time.Now().Format("15:04:05.000")
	line := fmt.Sprintf("[%s] [TOKEN] %s: %s\n", timestamp, event, details)

	_, _ = logFile.WriteString(line) //nolint:errcheck // Intentionally ignore errors in debug logging
	_ = logFile.Sync()               //nolint:errcheck // Flush immediately
}
