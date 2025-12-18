// Package debug provides development logging for CDD CLI.
package debug

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	enabled  bool
	logFile  *os.File
	mu       sync.Mutex
	logPath  string
)

// Enable turns on debug logging to the specified file.
func Enable(path string) error {
	mu.Lock()
	defer mu.Unlock()

	if enabled {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating log directory: %w", err)
	}

	// Open log file (append mode)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
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
	logFile.WriteString(header)
	logFile.Sync()

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
		logFile.Close()
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

// Log writes a debug message if logging is enabled.
func Log(format string, args ...any) {
	mu.Lock()
	defer mu.Unlock()

	if !enabled || logFile == nil {
		return
	}

	timestamp := time.Now().Format("15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("[%s] %s\n", timestamp, msg)

	logFile.WriteString(line)
	logFile.Sync() // Flush immediately for real-time viewing
}

// LogPath returns the path to the log file.
func LogPath() string {
	mu.Lock()
	defer mu.Unlock()
	return logPath
}

// Event logs a TUI event with component context.
func Event(component, eventType string, details string) {
	Log("[%s] %s: %s", component, eventType, details)
}

// Error logs an error with context.
func Error(component string, err error, context string) {
	Log("[%s] ERROR: %s - %v", component, context, err)
}
