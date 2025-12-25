// Package events defines domain-specific event types for the pub/sub system.
package events

import (
	"time"
)

// AgentEventType represents agent-specific event types.
type AgentEventType string

// Agent event type constants.
const (
	AgentEventTextDelta  AgentEventType = "text_delta"
	AgentEventToolCall   AgentEventType = "tool_call"
	AgentEventToolResult AgentEventType = "tool_result"
	AgentEventComplete   AgentEventType = "complete"
	AgentEventError      AgentEventType = "error"
	AgentEventCancelled  AgentEventType = "cancelled"
)

// AgentEvent represents an agent streaming event.
type AgentEvent struct { //nolint:govet // fieldalignment: preserving logical field order
	SessionID string
	MessageID string
	Type      AgentEventType
	Timestamp time.Time

	// Payload fields (only one populated per event type)
	TextDelta  string          // For TextDelta
	ToolCall   *ToolCallInfo   // For ToolCall
	ToolResult *ToolResultInfo // For ToolResult
	Error      error           // For Error
}

// ToolCallInfo contains tool call details.
type ToolCallInfo struct {
	ID    string
	Name  string
	Input string
}

// ToolResultInfo contains tool result details.
type ToolResultInfo struct {
	ToolCallID string
	Name       string
	Content    string
	IsError    bool
	Duration   time.Duration
}

// NewTextDeltaEvent creates a text delta event.
func NewTextDeltaEvent(sessionID, messageID, text string) AgentEvent {
	return AgentEvent{
		SessionID: sessionID,
		MessageID: messageID,
		Type:      AgentEventTextDelta,
		TextDelta: text,
		Timestamp: time.Now(),
	}
}

// NewToolCallEvent creates a tool call event.
func NewToolCallEvent(sessionID, messageID string, tc ToolCallInfo) AgentEvent {
	return AgentEvent{
		SessionID: sessionID,
		MessageID: messageID,
		Type:      AgentEventToolCall,
		ToolCall:  &tc,
		Timestamp: time.Now(),
	}
}

// NewToolResultEvent creates a tool result event.
func NewToolResultEvent(sessionID, messageID string, tr ToolResultInfo) AgentEvent {
	return AgentEvent{
		SessionID:  sessionID,
		MessageID:  messageID,
		Type:       AgentEventToolResult,
		ToolResult: &tr,
		Timestamp:  time.Now(),
	}
}

// NewCompleteEvent creates a completion event.
func NewCompleteEvent(sessionID, messageID string) AgentEvent {
	return AgentEvent{
		SessionID: sessionID,
		MessageID: messageID,
		Type:      AgentEventComplete,
		Timestamp: time.Now(),
	}
}

// NewErrorEvent creates an error event.
func NewErrorEvent(sessionID, messageID string, err error) AgentEvent {
	return AgentEvent{
		SessionID: sessionID,
		MessageID: messageID,
		Type:      AgentEventError,
		Error:     err,
		Timestamp: time.Now(),
	}
}

// NewCancelledEvent creates a cancelled event.
func NewCancelledEvent(sessionID, messageID string) AgentEvent {
	return AgentEvent{
		SessionID: sessionID,
		MessageID: messageID,
		Type:      AgentEventCancelled,
		Timestamp: time.Now(),
	}
}
