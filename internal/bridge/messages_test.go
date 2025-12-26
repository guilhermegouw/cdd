//nolint:errorlint // Test files use direct error comparison.
package bridge

import (
	"errors"
	"testing"
	"time"

	"github.com/guilhermegouw/cdd/internal/events"
	"github.com/guilhermegouw/cdd/internal/pubsub"
)

func TestAgentEventMsg(t *testing.T) {
	t.Run("wraps agent event correctly", func(t *testing.T) {
		agentEvent := events.NewTextDeltaEvent("session-1", "msg-1", "Hello")
		event := pubsub.Event[events.AgentEvent]{
			Type:      pubsub.EventProgress,
			Payload:   agentEvent,
			Timestamp: time.Now(),
		}

		msg := AgentEventMsg{Event: event}

		if msg.Event.Type != pubsub.EventProgress {
			t.Errorf("expected Type EventProgress, got %q", msg.Event.Type)
		}
		if msg.Event.Payload.TextDelta != "Hello" {
			t.Errorf("expected TextDelta 'Hello', got %q", msg.Event.Payload.TextDelta)
		}
		if msg.Event.Payload.SessionID != "session-1" {
			t.Errorf("expected SessionID 'session-1', got %q", msg.Event.Payload.SessionID)
		}
	})
}

func TestToolEventMsg(t *testing.T) {
	t.Run("wraps tool event correctly", func(t *testing.T) {
		toolEvent := events.NewToolStartedEvent("session-1", "tc-1", "read_file", "{}")
		event := pubsub.Event[events.ToolEvent]{
			Type:      pubsub.EventStarted,
			Payload:   toolEvent,
			Timestamp: time.Now(),
		}

		msg := ToolEventMsg{Event: event}

		if msg.Event.Type != pubsub.EventStarted {
			t.Errorf("expected Type EventStarted, got %q", msg.Event.Type)
		}
		if msg.Event.Payload.ToolName != "read_file" {
			t.Errorf("expected ToolName 'read_file', got %q", msg.Event.Payload.ToolName)
		}
		if msg.Event.Payload.ToolCallID != "tc-1" {
			t.Errorf("expected ToolCallID 'tc-1', got %q", msg.Event.Payload.ToolCallID)
		}
	})
}

func TestSessionEventMsg(t *testing.T) {
	t.Run("wraps session event correctly", func(t *testing.T) {
		sessionEvent := events.NewSessionCreatedEvent("session-1", "My Session")
		event := pubsub.Event[events.SessionEvent]{
			Type:      pubsub.EventCreated,
			Payload:   sessionEvent,
			Timestamp: time.Now(),
		}

		msg := SessionEventMsg{Event: event}

		if msg.Event.Type != pubsub.EventCreated {
			t.Errorf("expected Type EventCreated, got %q", msg.Event.Type)
		}
		if msg.Event.Payload.Title != "My Session" {
			t.Errorf("expected Title 'My Session', got %q", msg.Event.Payload.Title)
		}
		if msg.Event.Payload.SessionID != "session-1" {
			t.Errorf("expected SessionID 'session-1', got %q", msg.Event.Payload.SessionID)
		}
	})
}

func TestAuthEventMsg(t *testing.T) {
	t.Run("wraps auth event correctly", func(t *testing.T) {
		expiresAt := time.Now().Add(1 * time.Hour)
		authEvent := events.NewTokenRefreshedEvent("anthropic", expiresAt)
		event := pubsub.Event[events.AuthEvent]{
			Type:      pubsub.EventCompleted,
			Payload:   authEvent,
			Timestamp: time.Now(),
		}

		msg := AuthEventMsg{Event: event}

		if msg.Event.Type != pubsub.EventCompleted {
			t.Errorf("expected Type EventCompleted, got %q", msg.Event.Type)
		}
		if msg.Event.Payload.ProviderID != "anthropic" {
			t.Errorf("expected ProviderID 'anthropic', got %q", msg.Event.Payload.ProviderID)
		}
		if msg.Event.Payload.ExpiresAt != expiresAt {
			t.Errorf("expected ExpiresAt %v, got %v", expiresAt, msg.Event.Payload.ExpiresAt)
		}
	})
}

func TestErrorMsg(t *testing.T) {
	t.Run("creates error message with source and error", func(t *testing.T) {
		testErr := errors.New("connection failed")
		msg := ErrorMsg{
			Source: "agent_broker",
			Error:  testErr,
		}

		if msg.Source != "agent_broker" {
			t.Errorf("expected Source 'agent_broker', got %q", msg.Source)
		}
		if msg.Error != testErr {
			t.Errorf("expected Error to be testErr, got %v", msg.Error)
		}
	})

	t.Run("handles nil error", func(t *testing.T) {
		msg := ErrorMsg{
			Source: "tool_broker",
			Error:  nil,
		}

		if msg.Error != nil {
			t.Error("expected Error to be nil")
		}
	})
}

func TestMessageTypesAreDistinct(t *testing.T) {
	// This test ensures that Go's type system correctly distinguishes
	// between the different message types (useful for switch statements)

	t.Run("all message types can be type-asserted", func(t *testing.T) {
		// Create instances of each message type
		agentMsg := AgentEventMsg{}
		toolMsg := ToolEventMsg{}
		sessionMsg := SessionEventMsg{}
		authMsg := AuthEventMsg{}
		errorMsg := ErrorMsg{}

		// Store them as interface{}
		messages := []interface{}{agentMsg, toolMsg, sessionMsg, authMsg, errorMsg}

		// Verify each can be type-asserted correctly
		for i, msg := range messages {
			switch i {
			case 0:
				if _, ok := msg.(AgentEventMsg); !ok {
					t.Error("expected AgentEventMsg")
				}
			case 1:
				if _, ok := msg.(ToolEventMsg); !ok {
					t.Error("expected ToolEventMsg")
				}
			case 2:
				if _, ok := msg.(SessionEventMsg); !ok {
					t.Error("expected SessionEventMsg")
				}
			case 3:
				if _, ok := msg.(AuthEventMsg); !ok {
					t.Error("expected AuthEventMsg")
				}
			case 4:
				if _, ok := msg.(ErrorMsg); !ok {
					t.Error("expected ErrorMsg")
				}
			}
		}
	})
}
