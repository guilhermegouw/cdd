// Package session provides session management with persistence.
package session

import (
	"context"
	"time"
)

// Session represents a conversation session.
type Session struct {
	ID               string
	Title            string
	MessageCount     int
	SummaryMessageID string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// SessionWithPreview includes the first user message preview.
//
//nolint:revive // Name is clear and used across packages
type SessionWithPreview struct {
	Session
	FirstMessage string // Preview of the first user message
}

// Store defines the interface for session persistence.
type Store interface {
	// Create creates a new session with the given title.
	Create(ctx context.Context, id, title string) (*Session, error)

	// Get retrieves a session by ID.
	Get(ctx context.Context, id string) (*Session, error)

	// List returns all sessions ordered by updated_at descending.
	List(ctx context.Context) ([]*Session, error)

	// ListWithPreview returns all sessions with first message preview.
	ListWithPreview(ctx context.Context) ([]*SessionWithPreview, error)

	// Search searches sessions by title keyword.
	Search(ctx context.Context, keyword string) ([]*Session, error)

	// SearchWithPreview searches sessions by title with first message preview.
	SearchWithPreview(ctx context.Context, keyword string) ([]*SessionWithPreview, error)

	// UpdateTitle updates the title of a session.
	UpdateTitle(ctx context.Context, id, title string) error

	// IncrementMessageCount increments the message count for a session.
	IncrementMessageCount(ctx context.Context, id string) error

	// DecrementMessageCount decrements the message count for a session.
	DecrementMessageCount(ctx context.Context, id string) error

	// SetSummaryMessage sets the summary message ID for a session.
	SetSummaryMessage(ctx context.Context, sessionID, messageID string) error

	// Delete removes a session by ID.
	Delete(ctx context.Context, id string) error
}
