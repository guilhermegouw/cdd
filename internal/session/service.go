package session

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/guilhermegouw/cdd/internal/events"
	"github.com/guilhermegouw/cdd/internal/pubsub"
)

// Service manages sessions with pub/sub event publishing.
type Service struct {
	store   Store
	broker  *pubsub.Broker[events.SessionEvent]
	current string
	mu      sync.RWMutex
}

// NewService creates a new session service.
func NewService(store Store, broker *pubsub.Broker[events.SessionEvent]) *Service {
	return &Service{
		store:  store,
		broker: broker,
	}
}

// Create creates a new session with the given title.
func (s *Service) Create(ctx context.Context, title string) (*Session, error) {
	id := uuid.New().String()

	session, err := s.store.Create(ctx, id, title)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.current = session.ID
	s.mu.Unlock()

	if s.broker != nil {
		s.broker.Publish(pubsub.EventCreated,
			events.NewSessionCreatedEvent(session.ID, session.Title))
	}

	return session, nil
}

// Get retrieves a session by ID.
func (s *Service) Get(ctx context.Context, id string) (*Session, error) {
	return s.store.Get(ctx, id)
}

// List returns all sessions.
func (s *Service) List(ctx context.Context) ([]*Session, error) {
	return s.store.List(ctx)
}

// ListWithPreview returns all sessions with first message preview.
func (s *Service) ListWithPreview(ctx context.Context) ([]*SessionWithPreview, error) {
	return s.store.ListWithPreview(ctx)
}

// Search searches sessions by title keyword.
func (s *Service) Search(ctx context.Context, keyword string) ([]*Session, error) {
	return s.store.Search(ctx, keyword)
}

// SearchWithPreview searches sessions with first message preview.
func (s *Service) SearchWithPreview(ctx context.Context, keyword string) ([]*SessionWithPreview, error) {
	return s.store.SearchWithPreview(ctx, keyword)
}

// Current returns the current session, creating one if none exists.
func (s *Service) Current(ctx context.Context) (*Session, error) {
	s.mu.RLock()
	currentID := s.current
	s.mu.RUnlock()

	if currentID != "" {
		session, err := s.store.Get(ctx, currentID)
		if err == nil {
			return session, nil
		}
		// Session not found, fall through to create new one
	}

	// No current session or it doesn't exist, create one
	return s.Create(ctx, "New Session")
}

// SetCurrent sets the current session ID.
func (s *Service) SetCurrent(id string) {
	s.mu.Lock()
	s.current = id
	s.mu.Unlock()

	if s.broker != nil {
		s.broker.Publish(pubsub.EventUpdated,
			events.NewSessionSwitchedEvent(id, ""))
	}
}

// CurrentID returns the current session ID.
func (s *Service) CurrentID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

// UpdateTitle updates the title of a session.
func (s *Service) UpdateTitle(ctx context.Context, id, title string) error {
	return s.store.UpdateTitle(ctx, id, title)
}

// Delete removes a session by ID.
func (s *Service) Delete(ctx context.Context, id string) error {
	err := s.store.Delete(ctx, id)
	if err != nil {
		return err
	}

	s.mu.Lock()
	if s.current == id {
		s.current = ""
	}
	s.mu.Unlock()

	if s.broker != nil {
		s.broker.Publish(pubsub.EventDeleted,
			events.NewSessionDeletedEvent(id))
	}

	return nil
}

// IncrementMessageCount increments the message count for a session.
func (s *Service) IncrementMessageCount(ctx context.Context, id string) error {
	return s.store.IncrementMessageCount(ctx, id)
}

// SetSummaryMessage sets the summary message ID for a session.
func (s *Service) SetSummaryMessage(ctx context.Context, sessionID, messageID string) error {
	return s.store.SetSummaryMessage(ctx, sessionID, messageID)
}
