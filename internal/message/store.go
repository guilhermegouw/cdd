package message

import (
	"context"
)

// Store defines the interface for message persistence.
type Store interface {
	// Create creates a new message.
	Create(ctx context.Context, msg *Message) error

	// Get retrieves a message by ID.
	Get(ctx context.Context, id string) (*Message, error)

	// GetBySession returns all messages for a session ordered by created_at.
	GetBySession(ctx context.Context, sessionID string) ([]*Message, error)

	// GetBySessionWithLimit returns messages for a session with a limit.
	GetBySessionWithLimit(ctx context.Context, sessionID string, limit int) ([]*Message, error)

	// GetFromMessage returns messages from a specific message ID onwards.
	GetFromMessage(ctx context.Context, sessionID, messageID string) ([]*Message, error)

	// GetSummary returns the most recent summary message for a session.
	GetSummary(ctx context.Context, sessionID string) (*Message, error)

	// Count returns the number of messages in a session.
	Count(ctx context.Context, sessionID string) (int64, error)

	// Update updates a message's parts.
	Update(ctx context.Context, msg *Message) error

	// Delete removes a message by ID.
	Delete(ctx context.Context, id string) error

	// DeleteBySession removes all messages for a session.
	DeleteBySession(ctx context.Context, sessionID string) error

	// DeleteOldMessages removes old messages keeping only the most recent ones.
	DeleteOldMessages(ctx context.Context, sessionID string, keepCount int) error
}
