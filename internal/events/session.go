package events

import "time"

// SessionEventType represents session-specific event types.
type SessionEventType string

// Session event type constants.
const (
	SessionEventCreated      SessionEventType = "created"
	SessionEventUpdated      SessionEventType = "updated"
	SessionEventDeleted      SessionEventType = "deleted"
	SessionEventSwitched     SessionEventType = "switched"
	SessionEventMessageAdded SessionEventType = "message_added"
	SessionEventCleared      SessionEventType = "cleared"
)

// SessionEvent represents a session lifecycle event.
type SessionEvent struct {
	SessionID string
	Title     string
	Type      SessionEventType
	Timestamp time.Time

	// Optional fields
	MessageRole string // For MessageAdded
	MessageText string // For MessageAdded
}

// NewSessionCreatedEvent creates a session created event.
func NewSessionCreatedEvent(id, title string) SessionEvent {
	return SessionEvent{
		SessionID: id,
		Title:     title,
		Type:      SessionEventCreated,
		Timestamp: time.Now(),
	}
}

// NewSessionSwitchedEvent creates a session switched event.
func NewSessionSwitchedEvent(id, title string) SessionEvent {
	return SessionEvent{
		SessionID: id,
		Title:     title,
		Type:      SessionEventSwitched,
		Timestamp: time.Now(),
	}
}

// NewSessionDeletedEvent creates a session deleted event.
func NewSessionDeletedEvent(id string) SessionEvent {
	return SessionEvent{
		SessionID: id,
		Type:      SessionEventDeleted,
		Timestamp: time.Now(),
	}
}

// NewSessionClearedEvent creates a session cleared event.
func NewSessionClearedEvent(id string) SessionEvent {
	return SessionEvent{
		SessionID: id,
		Type:      SessionEventCleared,
		Timestamp: time.Now(),
	}
}

// NewSessionMessageAddedEvent creates a message added event.
func NewSessionMessageAddedEvent(sessionID, role, text string) SessionEvent {
	return SessionEvent{
		SessionID:   sessionID,
		Type:        SessionEventMessageAdded,
		MessageRole: role,
		MessageText: text,
		Timestamp:   time.Now(),
	}
}
