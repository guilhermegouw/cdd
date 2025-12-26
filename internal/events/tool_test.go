//nolint:goconst,errorlint // Test files use literal strings and direct error comparison.
package events

import (
	"errors"
	"testing"
	"time"
)

func TestToolEventTypes(t *testing.T) {
	// Verify all event types are distinct
	types := []ToolEventType{
		ToolEventStarted,
		ToolEventProgress,
		ToolEventCompleted,
		ToolEventFailed,
	}

	seen := make(map[ToolEventType]bool)
	for _, typ := range types {
		if seen[typ] {
			t.Errorf("duplicate event type: %s", typ)
		}
		seen[typ] = true

		// Verify non-empty string value
		if string(typ) == "" {
			t.Error("event type should have non-empty string value")
		}
	}
}

func TestNewToolStartedEvent(t *testing.T) {
	t.Run("creates started event with correct fields", func(t *testing.T) {
		before := time.Now()
		event := NewToolStartedEvent("session-1", "tc-1", "read_file", `{"path": "/tmp"}`)
		after := time.Now()

		if event.SessionID != "session-1" {
			t.Errorf("expected SessionID 'session-1', got %q", event.SessionID)
		}
		if event.ToolCallID != "tc-1" {
			t.Errorf("expected ToolCallID 'tc-1', got %q", event.ToolCallID)
		}
		if event.ToolName != "read_file" {
			t.Errorf("expected ToolName 'read_file', got %q", event.ToolName)
		}
		if event.Type != ToolEventStarted {
			t.Errorf("expected Type ToolEventStarted, got %q", event.Type)
		}
		if event.Input != `{"path": "/tmp"}` {
			t.Errorf("unexpected Input: %q", event.Input)
		}
		if event.Timestamp.Before(before) || event.Timestamp.After(after) {
			t.Error("timestamp should be within test bounds")
		}

		// Other fields should be zero
		if event.Output != "" {
			t.Error("Output should be empty")
		}
		if event.Error != nil {
			t.Error("Error should be nil")
		}
		if event.Duration != 0 {
			t.Error("Duration should be zero")
		}
		if event.Progress != 0 {
			t.Error("Progress should be zero")
		}
	})

	t.Run("handles empty input", func(t *testing.T) {
		event := NewToolStartedEvent("s", "tc", "tool", "")
		if event.Input != "" {
			t.Error("expected empty Input")
		}
	})
}

func TestNewToolCompletedEvent(t *testing.T) {
	t.Run("creates completed event with correct fields", func(t *testing.T) {
		duration := 150 * time.Millisecond

		before := time.Now()
		event := NewToolCompletedEvent("session-1", "tc-1", "read_file", "file contents", duration)
		after := time.Now()

		if event.SessionID != "session-1" {
			t.Errorf("expected SessionID 'session-1', got %q", event.SessionID)
		}
		if event.ToolCallID != "tc-1" {
			t.Errorf("expected ToolCallID 'tc-1', got %q", event.ToolCallID)
		}
		if event.ToolName != "read_file" {
			t.Errorf("expected ToolName 'read_file', got %q", event.ToolName)
		}
		if event.Type != ToolEventCompleted {
			t.Errorf("expected Type ToolEventCompleted, got %q", event.Type)
		}
		if event.Output != "file contents" {
			t.Errorf("unexpected Output: %q", event.Output)
		}
		if event.Duration != duration {
			t.Errorf("expected Duration %v, got %v", duration, event.Duration)
		}
		if event.Timestamp.Before(before) || event.Timestamp.After(after) {
			t.Error("timestamp should be within test bounds")
		}

		// Other fields should be zero
		if event.Input != "" {
			t.Error("Input should be empty")
		}
		if event.Error != nil {
			t.Error("Error should be nil")
		}
	})

	t.Run("handles zero duration", func(t *testing.T) {
		event := NewToolCompletedEvent("s", "tc", "tool", "output", 0)
		if event.Duration != 0 {
			t.Error("expected zero Duration")
		}
	})
}

func TestNewToolFailedEvent(t *testing.T) {
	t.Run("creates failed event with correct fields", func(t *testing.T) {
		testErr := errors.New("file not found")
		duration := 50 * time.Millisecond

		before := time.Now()
		event := NewToolFailedEvent("session-1", "tc-1", "read_file", testErr, duration)
		after := time.Now()

		if event.SessionID != "session-1" {
			t.Errorf("expected SessionID 'session-1', got %q", event.SessionID)
		}
		if event.ToolCallID != "tc-1" {
			t.Errorf("expected ToolCallID 'tc-1', got %q", event.ToolCallID)
		}
		if event.ToolName != "read_file" {
			t.Errorf("expected ToolName 'read_file', got %q", event.ToolName)
		}
		if event.Type != ToolEventFailed {
			t.Errorf("expected Type ToolEventFailed, got %q", event.Type)
		}
		if event.Error != testErr {
			t.Errorf("expected Error to be testErr, got %v", event.Error)
		}
		if event.Duration != duration {
			t.Errorf("expected Duration %v, got %v", duration, event.Duration)
		}
		if event.Timestamp.Before(before) || event.Timestamp.After(after) {
			t.Error("timestamp should be within test bounds")
		}

		// Other fields should be zero
		if event.Input != "" {
			t.Error("Input should be empty")
		}
		if event.Output != "" {
			t.Error("Output should be empty")
		}
	})

	t.Run("handles nil error", func(t *testing.T) {
		event := NewToolFailedEvent("s", "tc", "tool", nil, 0)

		if event.Error != nil {
			t.Error("Error should be nil")
		}
		if event.Type != ToolEventFailed {
			t.Error("Type should still be ToolEventFailed")
		}
	})
}

func TestNewToolProgressEvent(t *testing.T) {
	t.Run("creates progress event with correct fields", func(t *testing.T) {
		before := time.Now()
		event := NewToolProgressEvent("session-1", "tc-1", "download", 0.75)
		after := time.Now()

		if event.SessionID != "session-1" {
			t.Errorf("expected SessionID 'session-1', got %q", event.SessionID)
		}
		if event.ToolCallID != "tc-1" {
			t.Errorf("expected ToolCallID 'tc-1', got %q", event.ToolCallID)
		}
		if event.ToolName != "download" {
			t.Errorf("expected ToolName 'download', got %q", event.ToolName)
		}
		if event.Type != ToolEventProgress {
			t.Errorf("expected Type ToolEventProgress, got %q", event.Type)
		}
		if event.Progress != 0.75 {
			t.Errorf("expected Progress 0.75, got %f", event.Progress)
		}
		if event.Timestamp.Before(before) || event.Timestamp.After(after) {
			t.Error("timestamp should be within test bounds")
		}

		// Other fields should be zero
		if event.Input != "" {
			t.Error("Input should be empty")
		}
		if event.Output != "" {
			t.Error("Output should be empty")
		}
		if event.Error != nil {
			t.Error("Error should be nil")
		}
		if event.Duration != 0 {
			t.Error("Duration should be zero")
		}
	})

	t.Run("handles boundary progress values", func(t *testing.T) {
		// 0%
		event := NewToolProgressEvent("s", "tc", "tool", 0.0)
		if event.Progress != 0.0 {
			t.Errorf("expected Progress 0.0, got %f", event.Progress)
		}

		// 100%
		event = NewToolProgressEvent("s", "tc", "tool", 1.0)
		if event.Progress != 1.0 {
			t.Errorf("expected Progress 1.0, got %f", event.Progress)
		}
	})
}

func TestToolEventStruct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		testErr := errors.New("test error")
		event := ToolEvent{
			SessionID:  "session-1",
			ToolCallID: "tc-1",
			ToolName:   "tool_name",
			Type:       ToolEventCompleted,
			Timestamp:  time.Now(),
			Input:      "input data",
			Output:     "output data",
			Error:      testErr,
			Duration:   100 * time.Millisecond,
			Progress:   0.5,
		}

		if event.SessionID != "session-1" {
			t.Error("SessionID mismatch")
		}
		if event.ToolCallID != "tc-1" {
			t.Error("ToolCallID mismatch")
		}
		if event.ToolName != "tool_name" {
			t.Error("ToolName mismatch")
		}
		if event.Type != ToolEventCompleted {
			t.Error("Type mismatch")
		}
		if event.Input != "input data" {
			t.Error("Input mismatch")
		}
		if event.Output != "output data" {
			t.Error("Output mismatch")
		}
		if event.Error != testErr {
			t.Error("Error mismatch")
		}
		if event.Duration != 100*time.Millisecond {
			t.Error("Duration mismatch")
		}
		if event.Progress != 0.5 {
			t.Error("Progress mismatch")
		}
		if event.Timestamp.IsZero() {
			t.Error("Timestamp should not be zero")
		}
	})
}
