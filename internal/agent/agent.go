// Package agent provides the AI agent implementation for CDD.
package agent

import (
	"context"
	"time"

	"charm.land/fantasy"
)

// Role represents the role of a message.
type Role string

// Role constants for message roles.
const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// Message represents a conversation message.
type Message struct { //nolint:govet // fieldalignment: preserving logical field order
	ID          string
	Content     string
	ToolCalls   []ToolCall
	ToolResults []ToolResult
	CreatedAt   time.Time
	Role        Role
}

// ToolCall represents a tool call made by the assistant.
type ToolCall struct {
	ID    string
	Name  string
	Input string
}

// ToolResult represents the result of a tool call.
type ToolResult struct {
	ToolCallID string
	Name       string
	Content    string
	IsError    bool
}

// StreamCallbacks contains callbacks for streaming responses.
type StreamCallbacks struct {
	OnTextDelta  func(text string) error
	OnToolCall   func(tc ToolCall) error
	OnToolResult func(tr ToolResult) error
	OnComplete   func() error
	OnError      func(err error)
}

// SendOptions contains options for sending a message.
type SendOptions struct { //nolint:govet // fieldalignment: preserving logical field order
	SessionID   string
	Temperature *float64
	MaxTokens   int64
}

// Agent is the interface for an AI agent.
type Agent interface {
	// Send sends a prompt and streams the response.
	Send(ctx context.Context, prompt string, opts SendOptions, callbacks StreamCallbacks) error

	// SetSystemPrompt sets the system prompt.
	SetSystemPrompt(prompt string)

	// SetTools sets the available tools.
	SetTools(tools []fantasy.AgentTool)

	// History returns the conversation history.
	History(sessionID string) []Message

	// Clear clears the conversation history.
	Clear(sessionID string)

	// Cancel cancels any ongoing request for a session.
	Cancel(sessionID string)

	// IsBusy returns true if the agent is processing a request for the session.
	IsBusy(sessionID string) bool
}

// Config contains agent configuration.
type Config struct { //nolint:govet // fieldalignment: preserving logical field order
	Model        fantasy.LanguageModel
	SystemPrompt string
	Tools        []fantasy.AgentTool
	WorkingDir   string
}

// ErrSessionBusy is returned when a session is already processing a request.
var ErrSessionBusy = NewError("session is busy")

// ErrEmptyPrompt is returned when an empty prompt is provided.
var ErrEmptyPrompt = NewError("prompt cannot be empty")

// Error represents an agent-specific error.
type Error struct {
	message string
}

// NewError creates a new agent error with the given message.
func NewError(message string) *Error {
	return &Error{message: message}
}

func (e *Error) Error() string {
	return e.message
}
