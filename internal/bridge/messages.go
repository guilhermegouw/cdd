// Package bridge provides the connection between the pub/sub system and Bubble Tea.
package bridge

import (
	"github.com/guilhermegouw/cdd/internal/events"
	"github.com/guilhermegouw/cdd/internal/pubsub"
)

// AgentEventMsg wraps an agent event for the TUI.
type AgentEventMsg struct {
	Event pubsub.Event[events.AgentEvent]
}

// ToolEventMsg wraps a tool event for the TUI.
type ToolEventMsg struct {
	Event pubsub.Event[events.ToolEvent]
}

// SessionEventMsg wraps a session event for the TUI.
type SessionEventMsg struct {
	Event pubsub.Event[events.SessionEvent]
}

// AuthEventMsg wraps an auth event for the TUI.
type AuthEventMsg struct {
	Event pubsub.Event[events.AuthEvent]
}

// TodoEventMsg wraps a todo event for the TUI.
type TodoEventMsg struct {
	Event pubsub.Event[events.TodoEvent]
}

// ErrorMsg indicates an error in the bridge.
type ErrorMsg struct { //nolint:govet // fieldalignment: preserving logical field order
	Source string
	Error  error
}
