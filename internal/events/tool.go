package events

import (
	"time"
)

// ToolEventType represents tool-specific event types.
type ToolEventType string

// Tool event type constants.
const (
	ToolEventStarted   ToolEventType = "started"
	ToolEventProgress  ToolEventType = "progress"
	ToolEventCompleted ToolEventType = "completed"
	ToolEventFailed    ToolEventType = "failed"
)

// ToolEvent represents a tool execution event.
type ToolEvent struct { //nolint:govet // fieldalignment: preserving logical field order
	SessionID  string
	ToolCallID string
	ToolName   string
	Type       ToolEventType
	Timestamp  time.Time

	// Optional fields
	Input    string        // For Started
	Output   string        // For Completed
	Error    error         // For Failed
	Duration time.Duration // For Completed/Failed
	Progress float64       // For Progress (0.0-1.0)
}

// NewToolStartedEvent creates a tool started event.
func NewToolStartedEvent(sessionID, toolCallID, toolName, input string) ToolEvent {
	return ToolEvent{
		SessionID:  sessionID,
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Type:       ToolEventStarted,
		Input:      input,
		Timestamp:  time.Now(),
	}
}

// NewToolCompletedEvent creates a tool completed event.
func NewToolCompletedEvent(sessionID, toolCallID, toolName, output string, duration time.Duration) ToolEvent {
	return ToolEvent{
		SessionID:  sessionID,
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Type:       ToolEventCompleted,
		Output:     output,
		Duration:   duration,
		Timestamp:  time.Now(),
	}
}

// NewToolFailedEvent creates a tool failed event.
func NewToolFailedEvent(sessionID, toolCallID, toolName string, err error, duration time.Duration) ToolEvent {
	return ToolEvent{
		SessionID:  sessionID,
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Type:       ToolEventFailed,
		Error:      err,
		Duration:   duration,
		Timestamp:  time.Now(),
	}
}

// NewToolProgressEvent creates a tool progress event.
func NewToolProgressEvent(sessionID, toolCallID, toolName string, progress float64) ToolEvent {
	return ToolEvent{
		SessionID:  sessionID,
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Type:       ToolEventProgress,
		Progress:   progress,
		Timestamp:  time.Now(),
	}
}
