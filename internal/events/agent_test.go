//nolint:dupl,goconst,errorlint // Test files use literal strings and direct error comparison.
package events

import (
	"errors"
	"testing"
	"time"
)

func TestAgentEventTypes(t *testing.T) {
	// Verify all event types are distinct
	types := []AgentEventType{
		AgentEventTextDelta,
		AgentEventToolCall,
		AgentEventToolResult,
		AgentEventComplete,
		AgentEventError,
		AgentEventCancelled,
	}

	seen := make(map[AgentEventType]bool)
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

func TestNewTextDeltaEvent(t *testing.T) {
	t.Run("creates text delta event with correct fields", func(t *testing.T) {
		before := time.Now()
		event := NewTextDeltaEvent("session-1", "msg-1", "Hello, world!")
		after := time.Now()

		if event.SessionID != "session-1" {
			t.Errorf("expected SessionID 'session-1', got %q", event.SessionID)
		}
		if event.MessageID != "msg-1" {
			t.Errorf("expected MessageID 'msg-1', got %q", event.MessageID)
		}
		if event.Type != AgentEventTextDelta {
			t.Errorf("expected Type AgentEventTextDelta, got %q", event.Type)
		}
		if event.TextDelta != "Hello, world!" {
			t.Errorf("expected TextDelta 'Hello, world!', got %q", event.TextDelta)
		}
		if event.Timestamp.Before(before) || event.Timestamp.After(after) {
			t.Error("timestamp should be within test bounds")
		}

		// Other fields should be zero
		if event.ToolCall != nil {
			t.Error("ToolCall should be nil")
		}
		if event.ToolResult != nil {
			t.Error("ToolResult should be nil")
		}
		if event.Error != nil {
			t.Error("Error should be nil")
		}
	})

	t.Run("handles empty text delta", func(t *testing.T) {
		event := NewTextDeltaEvent("s", "m", "")
		if event.TextDelta != "" {
			t.Error("expected empty TextDelta")
		}
	})
}

func TestNewToolCallEvent(t *testing.T) {
	t.Run("creates tool call event with correct fields", func(t *testing.T) {
		tc := ToolCallInfo{
			ID:    "tc-1",
			Name:  "read_file",
			Input: `{"path": "/tmp/test.txt"}`,
		}

		before := time.Now()
		event := NewToolCallEvent("session-1", "msg-1", tc)
		after := time.Now()

		if event.SessionID != "session-1" {
			t.Errorf("expected SessionID 'session-1', got %q", event.SessionID)
		}
		if event.MessageID != "msg-1" {
			t.Errorf("expected MessageID 'msg-1', got %q", event.MessageID)
		}
		if event.Type != AgentEventToolCall {
			t.Errorf("expected Type AgentEventToolCall, got %q", event.Type)
		}
		if event.ToolCall == nil {
			t.Fatal("ToolCall should not be nil")
		}
		if event.ToolCall.ID != "tc-1" {
			t.Errorf("expected ToolCall.ID 'tc-1', got %q", event.ToolCall.ID)
		}
		if event.ToolCall.Name != "read_file" {
			t.Errorf("expected ToolCall.Name 'read_file', got %q", event.ToolCall.Name)
		}
		if event.ToolCall.Input != `{"path": "/tmp/test.txt"}` {
			t.Errorf("unexpected ToolCall.Input: %q", event.ToolCall.Input)
		}
		if event.Timestamp.Before(before) || event.Timestamp.After(after) {
			t.Error("timestamp should be within test bounds")
		}

		// Other fields should be zero
		if event.TextDelta != "" {
			t.Error("TextDelta should be empty")
		}
		if event.ToolResult != nil {
			t.Error("ToolResult should be nil")
		}
		if event.Error != nil {
			t.Error("Error should be nil")
		}
	})
}

func TestNewToolResultEvent(t *testing.T) {
	t.Run("creates tool result event with correct fields", func(t *testing.T) {
		tr := ToolResultInfo{
			ToolCallID: "tc-1",
			Name:       "read_file",
			Content:    "file contents here",
			IsError:    false,
			Duration:   100 * time.Millisecond,
		}

		before := time.Now()
		event := NewToolResultEvent("session-1", "msg-1", tr)
		after := time.Now()

		if event.SessionID != "session-1" {
			t.Errorf("expected SessionID 'session-1', got %q", event.SessionID)
		}
		if event.Type != AgentEventToolResult {
			t.Errorf("expected Type AgentEventToolResult, got %q", event.Type)
		}
		if event.ToolResult == nil {
			t.Fatal("ToolResult should not be nil")
		}
		if event.ToolResult.ToolCallID != "tc-1" {
			t.Errorf("expected ToolResult.ToolCallID 'tc-1', got %q", event.ToolResult.ToolCallID)
		}
		if event.ToolResult.Name != "read_file" {
			t.Errorf("expected ToolResult.Name 'read_file', got %q", event.ToolResult.Name)
		}
		if event.ToolResult.Content != "file contents here" {
			t.Errorf("unexpected ToolResult.Content: %q", event.ToolResult.Content)
		}
		if event.ToolResult.IsError {
			t.Error("ToolResult.IsError should be false")
		}
		if event.ToolResult.Duration != 100*time.Millisecond {
			t.Errorf("expected Duration 100ms, got %v", event.ToolResult.Duration)
		}
		if event.Timestamp.Before(before) || event.Timestamp.After(after) {
			t.Error("timestamp should be within test bounds")
		}
	})

	t.Run("handles error result", func(t *testing.T) {
		tr := ToolResultInfo{
			ToolCallID: "tc-1",
			Name:       "read_file",
			Content:    "file not found",
			IsError:    true,
		}

		event := NewToolResultEvent("s", "m", tr)

		if !event.ToolResult.IsError {
			t.Error("ToolResult.IsError should be true")
		}
	})
}

func TestNewCompleteEvent(t *testing.T) {
	t.Run("creates complete event with correct fields", func(t *testing.T) {
		before := time.Now()
		event := NewCompleteEvent("session-1", "msg-1")
		after := time.Now()

		if event.SessionID != "session-1" {
			t.Errorf("expected SessionID 'session-1', got %q", event.SessionID)
		}
		if event.MessageID != "msg-1" {
			t.Errorf("expected MessageID 'msg-1', got %q", event.MessageID)
		}
		if event.Type != AgentEventComplete {
			t.Errorf("expected Type AgentEventComplete, got %q", event.Type)
		}
		if event.Timestamp.Before(before) || event.Timestamp.After(after) {
			t.Error("timestamp should be within test bounds")
		}

		// All payload fields should be zero
		if event.TextDelta != "" {
			t.Error("TextDelta should be empty")
		}
		if event.ToolCall != nil {
			t.Error("ToolCall should be nil")
		}
		if event.ToolResult != nil {
			t.Error("ToolResult should be nil")
		}
		if event.Error != nil {
			t.Error("Error should be nil")
		}
	})
}

func TestNewErrorEvent(t *testing.T) {
	t.Run("creates error event with correct fields", func(t *testing.T) {
		testErr := errors.New("something went wrong")

		before := time.Now()
		event := NewErrorEvent("session-1", "msg-1", testErr)
		after := time.Now()

		if event.SessionID != "session-1" {
			t.Errorf("expected SessionID 'session-1', got %q", event.SessionID)
		}
		if event.MessageID != "msg-1" {
			t.Errorf("expected MessageID 'msg-1', got %q", event.MessageID)
		}
		if event.Type != AgentEventError {
			t.Errorf("expected Type AgentEventError, got %q", event.Type)
		}
		if event.Error != testErr {
			t.Errorf("expected Error to be testErr, got %v", event.Error)
		}
		if event.Timestamp.Before(before) || event.Timestamp.After(after) {
			t.Error("timestamp should be within test bounds")
		}
	})

	t.Run("handles nil error", func(t *testing.T) {
		event := NewErrorEvent("s", "m", nil)

		if event.Error != nil {
			t.Error("Error should be nil")
		}
		if event.Type != AgentEventError {
			t.Error("Type should still be AgentEventError")
		}
	})
}

func TestNewCancelledEvent(t *testing.T) {
	t.Run("creates cancelled event with correct fields", func(t *testing.T) {
		before := time.Now()
		event := NewCancelledEvent("session-1", "msg-1")
		after := time.Now()

		if event.SessionID != "session-1" {
			t.Errorf("expected SessionID 'session-1', got %q", event.SessionID)
		}
		if event.MessageID != "msg-1" {
			t.Errorf("expected MessageID 'msg-1', got %q", event.MessageID)
		}
		if event.Type != AgentEventCancelled {
			t.Errorf("expected Type AgentEventCancelled, got %q", event.Type)
		}
		if event.Timestamp.Before(before) || event.Timestamp.After(after) {
			t.Error("timestamp should be within test bounds")
		}

		// All payload fields should be zero
		if event.TextDelta != "" {
			t.Error("TextDelta should be empty")
		}
		if event.ToolCall != nil {
			t.Error("ToolCall should be nil")
		}
		if event.ToolResult != nil {
			t.Error("ToolResult should be nil")
		}
		if event.Error != nil {
			t.Error("Error should be nil")
		}
	})
}

func TestToolCallInfo(t *testing.T) {
	t.Run("struct fields", func(t *testing.T) {
		tc := ToolCallInfo{
			ID:    "id-1",
			Name:  "tool_name",
			Input: `{"key": "value"}`,
		}

		if tc.ID != "id-1" {
			t.Errorf("expected ID 'id-1', got %q", tc.ID)
		}
		if tc.Name != "tool_name" {
			t.Errorf("expected Name 'tool_name', got %q", tc.Name)
		}
		if tc.Input != `{"key": "value"}` {
			t.Errorf("unexpected Input: %q", tc.Input)
		}
	})
}

func TestToolResultInfo(t *testing.T) {
	t.Run("struct fields", func(t *testing.T) {
		tr := ToolResultInfo{
			ToolCallID: "tc-1",
			Name:       "tool_name",
			Content:    "result content",
			IsError:    true,
			Duration:   500 * time.Millisecond,
		}

		if tr.ToolCallID != "tc-1" {
			t.Errorf("expected ToolCallID 'tc-1', got %q", tr.ToolCallID)
		}
		if tr.Name != "tool_name" {
			t.Errorf("expected Name 'tool_name', got %q", tr.Name)
		}
		if tr.Content != "result content" {
			t.Errorf("unexpected Content: %q", tr.Content)
		}
		if !tr.IsError {
			t.Error("expected IsError to be true")
		}
		if tr.Duration != 500*time.Millisecond {
			t.Errorf("expected Duration 500ms, got %v", tr.Duration)
		}
	})
}
