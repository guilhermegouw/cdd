//nolint:goconst // Test files use literal strings for clarity.
package events

import (
	"testing"
	"time"
)

func TestSessionEventTypes(t *testing.T) {
	// Verify all event types are distinct
	types := []SessionEventType{
		SessionEventCreated,
		SessionEventSwitched,
		SessionEventDeleted,
	}

	seen := make(map[SessionEventType]bool)
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

func TestNewSessionCreatedEvent(t *testing.T) {
	t.Run("creates created event with correct fields", func(t *testing.T) {
		before := time.Now()
		event := NewSessionCreatedEvent("session-123", "My Session")
		after := time.Now()

		if event.SessionID != "session-123" {
			t.Errorf("expected SessionID 'session-123', got %q", event.SessionID)
		}
		if event.Title != "My Session" {
			t.Errorf("expected Title 'My Session', got %q", event.Title)
		}
		if event.Type != SessionEventCreated {
			t.Errorf("expected Type SessionEventCreated, got %q", event.Type)
		}
		if event.Timestamp.Before(before) || event.Timestamp.After(after) {
			t.Error("timestamp should be within test bounds")
		}
	})

	t.Run("handles empty title", func(t *testing.T) {
		event := NewSessionCreatedEvent("s", "")
		if event.Title != "" {
			t.Error("expected empty Title")
		}
	})

	t.Run("handles special characters in title", func(t *testing.T) {
		title := "Session with 日本語 and émojis 🎉"
		event := NewSessionCreatedEvent("s", title)
		if event.Title != title {
			t.Errorf("expected Title %q, got %q", title, event.Title)
		}
	})
}

func TestNewSessionSwitchedEvent(t *testing.T) {
	t.Run("creates switched event with correct fields", func(t *testing.T) {
		before := time.Now()
		event := NewSessionSwitchedEvent("session-456", "Another Session")
		after := time.Now()

		if event.SessionID != "session-456" {
			t.Errorf("expected SessionID 'session-456', got %q", event.SessionID)
		}
		if event.Title != "Another Session" {
			t.Errorf("expected Title 'Another Session', got %q", event.Title)
		}
		if event.Type != SessionEventSwitched {
			t.Errorf("expected Type SessionEventSwitched, got %q", event.Type)
		}
		if event.Timestamp.Before(before) || event.Timestamp.After(after) {
			t.Error("timestamp should be within test bounds")
		}
	})
}

func TestNewSessionDeletedEvent(t *testing.T) {
	t.Run("creates deleted event with correct fields", func(t *testing.T) {
		before := time.Now()
		event := NewSessionDeletedEvent("session-789")
		after := time.Now()

		if event.SessionID != "session-789" {
			t.Errorf("expected SessionID 'session-789', got %q", event.SessionID)
		}
		if event.Type != SessionEventDeleted {
			t.Errorf("expected Type SessionEventDeleted, got %q", event.Type)
		}
		if event.Timestamp.Before(before) || event.Timestamp.After(after) {
			t.Error("timestamp should be within test bounds")
		}

		// Title should be empty for deleted events
		if event.Title != "" {
			t.Error("Title should be empty for deleted events")
		}
	})
}

func TestSessionEventStruct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		now := time.Now()
		event := SessionEvent{
			SessionID: "session-1",
			Title:     "Test Session",
			Type:      SessionEventCreated,
			Timestamp: now,
		}

		if event.SessionID != "session-1" {
			t.Error("SessionID mismatch")
		}
		if event.Title != "Test Session" {
			t.Error("Title mismatch")
		}
		if event.Type != SessionEventCreated {
			t.Error("Type mismatch")
		}
		if event.Timestamp != now {
			t.Error("Timestamp mismatch")
		}
	})
}
