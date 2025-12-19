// Package tools provides agent tools for CDD.
package tools

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Context key types for type safety.
type (
	sessionIDContextKey  string
	messageIDContextKey  string
	workingDirContextKey string
)

// Context keys for tool execution.
const (
	SessionIDContextKey  sessionIDContextKey  = "session_id"
	MessageIDContextKey  messageIDContextKey  = "message_id"
	WorkingDirContextKey workingDirContextKey = "working_dir"
)

// WithSessionID adds a session ID to the context.
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, SessionIDContextKey, sessionID)
}

// SessionIDFromContext retrieves the session ID from the context.
func SessionIDFromContext(ctx context.Context) string {
	sessionID := ctx.Value(SessionIDContextKey)
	if sessionID == nil {
		return ""
	}
	s, ok := sessionID.(string)
	if !ok {
		return ""
	}
	return s
}

// WithMessageID adds a message ID to the context.
func WithMessageID(ctx context.Context, messageID string) context.Context {
	return context.WithValue(ctx, MessageIDContextKey, messageID)
}

// MessageIDFromContext retrieves the message ID from the context.
func MessageIDFromContext(ctx context.Context) string {
	messageID := ctx.Value(MessageIDContextKey)
	if messageID == nil {
		return ""
	}
	s, ok := messageID.(string)
	if !ok {
		return ""
	}
	return s
}

// WithWorkingDir adds a working directory to the context.
func WithWorkingDir(ctx context.Context, workingDir string) context.Context {
	return context.WithValue(ctx, WorkingDirContextKey, workingDir)
}

// WorkingDirFromContext retrieves the working directory from the context.
func WorkingDirFromContext(ctx context.Context) string {
	workingDir := ctx.Value(WorkingDirContextKey)
	if workingDir == nil {
		return ""
	}
	s, ok := workingDir.(string)
	if !ok {
		return ""
	}
	return s
}

// ResolvePath resolves a potentially relative path against the working directory.
func ResolvePath(workingDir, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(workingDir, path))
}

// IsPathWithinDir checks if a path is within the given directory.
func IsPathWithinDir(path, dir string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absDir, absPath)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}

// FileRecord tracks when files were read/written.
type FileRecord struct { //nolint:govet // fieldalignment: preserving logical field order
	Path      string
	ReadTime  time.Time
	WriteTime time.Time
}

var (
	fileRecords     = make(map[string]FileRecord)
	fileRecordMutex sync.RWMutex
)

// RecordFileRead records that a file was read.
func RecordFileRead(path string) {
	fileRecordMutex.Lock()
	defer fileRecordMutex.Unlock()

	record, exists := fileRecords[path]
	if !exists {
		record = FileRecord{Path: path}
	}
	record.ReadTime = time.Now()
	fileRecords[path] = record
}

// GetLastReadTime returns the last time a file was read.
func GetLastReadTime(path string) time.Time {
	fileRecordMutex.RLock()
	defer fileRecordMutex.RUnlock()

	record, exists := fileRecords[path]
	if !exists {
		return time.Time{}
	}
	return record.ReadTime
}

// RecordFileWrite records that a file was written.
func RecordFileWrite(path string) {
	fileRecordMutex.Lock()
	defer fileRecordMutex.Unlock()

	record, exists := fileRecords[path]
	if !exists {
		record = FileRecord{Path: path}
	}
	record.WriteTime = time.Now()
	fileRecords[path] = record
}

// GetLastWriteTime returns the last time a file was written.
func GetLastWriteTime(path string) time.Time {
	fileRecordMutex.RLock()
	defer fileRecordMutex.RUnlock()

	record, exists := fileRecords[path]
	if !exists {
		return time.Time{}
	}
	return record.WriteTime
}

// ClearFileRecords clears all file records.
func ClearFileRecords() {
	fileRecordMutex.Lock()
	defer fileRecordMutex.Unlock()
	fileRecords = make(map[string]FileRecord)
}
