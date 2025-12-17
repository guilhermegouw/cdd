package agent

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session represents a conversation session.
type Session struct {
	ID        string
	Title     string
	Messages  []Message
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SessionStore manages conversation sessions in memory.
type SessionStore struct {
	sessions map[string]*Session
	current  string
	mu       sync.RWMutex
}

// NewSessionStore creates a new in-memory session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
	}
}

// Create creates a new session.
func (s *SessionStore) Create(title string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()
	now := time.Now()
	session := &Session{
		ID:        id,
		Title:     title,
		Messages:  []Message{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.sessions[id] = session
	s.current = id

	return session
}

// Get returns a session by ID.
func (s *SessionStore) Get(id string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[id]
	return session, ok
}

// Current returns the current session, creating one if none exists.
func (s *SessionStore) Current() *Session {
	s.mu.RLock()
	if s.current != "" {
		if session, ok := s.sessions[s.current]; ok {
			s.mu.RUnlock()
			return session
		}
	}
	s.mu.RUnlock()

	return s.Create("New Session")
}

// SetCurrent sets the current session.
func (s *SessionStore) SetCurrent(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[id]; ok {
		s.current = id
		return true
	}
	return false
}

// List returns all sessions.
func (s *SessionStore) List() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// Delete removes a session.
func (s *SessionStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[id]; !ok {
		return false
	}

	delete(s.sessions, id)

	if s.current == id {
		s.current = ""
	}

	return true
}

// AddMessage adds a message to a session.
func (s *SessionStore) AddMessage(sessionID string, msg Message) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return false
	}

	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	session.Messages = append(session.Messages, msg)
	session.UpdatedAt = time.Now()

	return true
}

// GetMessages returns all messages for a session.
func (s *SessionStore) GetMessages(sessionID string) []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil
	}

	// Return a copy
	messages := make([]Message, len(session.Messages))
	copy(messages, session.Messages)
	return messages
}

// ClearMessages clears all messages from a session.
func (s *SessionStore) ClearMessages(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return false
	}

	session.Messages = []Message{}
	session.UpdatedAt = time.Now()

	return true
}

// UpdateTitle updates a session's title.
func (s *SessionStore) UpdateTitle(sessionID, title string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return false
	}

	session.Title = title
	session.UpdatedAt = time.Now()

	return true
}
