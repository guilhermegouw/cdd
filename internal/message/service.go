package message

import (
	"context"
	"errors"

	"github.com/guilhermegouw/cdd/internal/events"
	"github.com/guilhermegouw/cdd/internal/pubsub"
)

const (
	// MaxMessages is the threshold before we consider context too large.
	MaxMessages = 100

	// MessagesToKeep is how many recent messages to keep after trimming.
	MessagesToKeep = 50
)

// Service manages messages with pub/sub event publishing.
type Service struct {
	store  Store
	broker *pubsub.Broker[events.SessionEvent]
}

// NewService creates a new message service.
func NewService(store Store, broker *pubsub.Broker[events.SessionEvent]) *Service {
	return &Service{
		store:  store,
		broker: broker,
	}
}

// Add creates a new message.
func (s *Service) Add(ctx context.Context, msg *Message) error {
	if err := s.store.Create(ctx, msg); err != nil {
		return err
	}

	// Publish event
	if s.broker != nil {
		s.broker.Publish(pubsub.EventProgress,
			events.NewSessionMessageAddedEvent(msg.SessionID, string(msg.Role), msg.TextContent()))
	}

	return nil
}

// Get retrieves a message by ID.
func (s *Service) Get(ctx context.Context, id string) (*Message, error) {
	return s.store.Get(ctx, id)
}

// GetBySession returns all messages for a session.
func (s *Service) GetBySession(ctx context.Context, sessionID string) ([]*Message, error) {
	return s.store.GetBySession(ctx, sessionID)
}

// GetContext returns messages suitable for LLM context.
// If a summary exists, returns summary + messages after it.
// Otherwise returns all messages (up to MaxMessages).
func (s *Service) GetContext(ctx context.Context, sessionID string) ([]*Message, error) {
	// First, try to get summary
	summary, err := s.store.GetSummary(ctx, sessionID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	if summary != nil {
		// Get messages from summary onwards
		msgs, err := s.store.GetFromMessage(ctx, sessionID, summary.ID)
		if err != nil {
			return nil, err
		}
		return msgs, nil
	}

	// No summary, return all messages (limited)
	return s.store.GetBySessionWithLimit(ctx, sessionID, MaxMessages)
}

// Clear removes all messages from a session.
func (s *Service) Clear(ctx context.Context, sessionID string) error {
	err := s.store.DeleteBySession(ctx, sessionID)
	if err != nil {
		return err
	}

	if s.broker != nil {
		s.broker.Publish(pubsub.EventUpdated,
			events.NewSessionClearedEvent(sessionID))
	}

	return nil
}

// Count returns the number of messages in a session.
func (s *Service) Count(ctx context.Context, sessionID string) (int64, error) {
	return s.store.Count(ctx, sessionID)
}

// Delete removes a message by ID.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// TrimOldMessages removes old messages keeping only the most recent ones.
func (s *Service) TrimOldMessages(ctx context.Context, sessionID string, keepCount int) error {
	return s.store.DeleteOldMessages(ctx, sessionID, keepCount)
}

// ShouldSummarize returns true if the session has too many messages.
func (s *Service) ShouldSummarize(ctx context.Context, sessionID string) (bool, error) {
	count, err := s.store.Count(ctx, sessionID)
	if err != nil {
		return false, err
	}
	return count > MaxMessages, nil
}
