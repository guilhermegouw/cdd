package agent

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/guilhermegouw/cdd/internal/message"
	"github.com/guilhermegouw/cdd/internal/session"
)

// PersistentSessionStore adapts the session and message services to the SessionStore interface.
// This enables database persistence while maintaining backward compatibility.
type PersistentSessionStore struct {
	sessionSvc *session.Service
	messageSvc *message.Service
	cache      map[string]*Session // Local cache for current session messages
	mu         sync.RWMutex
}

// NewPersistentSessionStore creates a new database-backed session store.
func NewPersistentSessionStore(ss *session.Service, ms *message.Service) *PersistentSessionStore {
	return &PersistentSessionStore{
		sessionSvc: ss,
		messageSvc: ms,
		cache:      make(map[string]*Session),
	}
}

// Create creates a new session.
func (s *PersistentSessionStore) Create(title string) *Session {
	ctx := context.Background()
	sess, err := s.sessionSvc.Create(ctx, title)
	if err != nil {
		// Fallback to in-memory session on error
		return s.createInMemory(title)
	}

	agentSession := &Session{
		ID:        sess.ID,
		Title:     sess.Title,
		Messages:  []Message{},
		CreatedAt: sess.CreatedAt,
		UpdatedAt: sess.UpdatedAt,
	}

	s.mu.Lock()
	s.cache[sess.ID] = agentSession
	s.mu.Unlock()

	return agentSession
}

// Get returns a session by ID.
func (s *PersistentSessionStore) Get(id string) (*Session, bool) {
	// Check cache first
	s.mu.RLock()
	if sess, ok := s.cache[id]; ok {
		s.mu.RUnlock()
		return sess, true
	}
	s.mu.RUnlock()

	// Load from database
	ctx := context.Background()
	dbSess, err := s.sessionSvc.Get(ctx, id)
	if err != nil {
		return nil, false
	}

	// Load messages
	msgs, err := s.messageSvc.GetContext(ctx, id)
	if err != nil {
		msgs = []*message.Message{}
	}

	agentSession := &Session{
		ID:        dbSess.ID,
		Title:     dbSess.Title,
		Messages:  convertFromMessagePkg(msgs),
		CreatedAt: dbSess.CreatedAt,
		UpdatedAt: dbSess.UpdatedAt,
	}

	s.mu.Lock()
	s.cache[id] = agentSession
	s.mu.Unlock()

	return agentSession, true
}

// Current returns the current session, creating a new one if none is set.
// By default, creates a fresh session on startup. Use SetCurrent() to resume
// a previous session (e.g., via the /sessions modal).
func (s *PersistentSessionStore) Current() *Session {
	ctx := context.Background()
	currentID := s.sessionSvc.CurrentID()

	// If we have a current session set, use it
	if currentID != "" {
		if sess, ok := s.Get(currentID); ok {
			return sess
		}
	}

	// Create a new session (fresh start by default)
	sess, err := s.sessionSvc.Create(ctx, "New Session")
	if err != nil {
		return s.createInMemory("New Session")
	}

	agentSession := &Session{
		ID:        sess.ID,
		Title:     sess.Title,
		Messages:  []Message{},
		CreatedAt: sess.CreatedAt,
		UpdatedAt: sess.UpdatedAt,
	}

	s.mu.Lock()
	s.cache[sess.ID] = agentSession
	s.mu.Unlock()

	return agentSession
}

// SetCurrent sets the current session.
func (s *PersistentSessionStore) SetCurrent(id string) bool {
	// Verify session exists
	if _, ok := s.Get(id); !ok {
		return false
	}

	s.sessionSvc.SetCurrent(id)
	return true
}

// List returns all sessions.
func (s *PersistentSessionStore) List() []*Session {
	ctx := context.Background()
	dbSessions, err := s.sessionSvc.List(ctx)
	if err != nil {
		return nil
	}

	sessions := make([]*Session, len(dbSessions))
	for i, dbs := range dbSessions {
		sessions[i] = &Session{
			ID:        dbs.ID,
			Title:     dbs.Title,
			CreatedAt: dbs.CreatedAt,
			UpdatedAt: dbs.UpdatedAt,
		}
	}
	return sessions
}

// Delete removes a session.
func (s *PersistentSessionStore) Delete(id string) bool {
	ctx := context.Background()
	if err := s.sessionSvc.Delete(ctx, id); err != nil {
		return false
	}

	s.mu.Lock()
	delete(s.cache, id)
	s.mu.Unlock()

	return true
}

// AddMessage adds a message to a session.
func (s *PersistentSessionStore) AddMessage(sessionID string, msg Message) bool {
	ctx := context.Background()

	// Convert to message.Message
	dbMsg := &message.Message{
		ID:        msg.ID,
		SessionID: sessionID,
		Role:      message.Role(msg.Role),
		Parts:     convertToMessageParts(msg),
		CreatedAt: msg.CreatedAt,
	}

	if dbMsg.ID == "" {
		dbMsg.ID = uuid.New().String()
		msg.ID = dbMsg.ID
	}

	if err := s.messageSvc.Add(ctx, dbMsg); err != nil {
		return false
	}

	// Increment message count in session
	_ = s.sessionSvc.IncrementMessageCount(ctx, sessionID)

	// Update cache
	s.mu.Lock()
	if sess, ok := s.cache[sessionID]; ok {
		sess.Messages = append(sess.Messages, msg)
		sess.UpdatedAt = time.Now()

		// Apply MaxSessionMessages limit to cache
		if len(sess.Messages) > MaxSessionMessages {
			excess := len(sess.Messages) - MaxSessionMessages
			sess.Messages = sess.Messages[excess:]
		}
	}
	s.mu.Unlock()

	return true
}

// GetMessages returns all messages for a session.
func (s *PersistentSessionStore) GetMessages(sessionID string) []Message {
	// Return from cache if available
	s.mu.RLock()
	if sess, ok := s.cache[sessionID]; ok {
		msgs := make([]Message, len(sess.Messages))
		copy(msgs, sess.Messages)
		s.mu.RUnlock()
		return msgs
	}
	s.mu.RUnlock()

	// Load from database
	ctx := context.Background()
	dbMsgs, err := s.messageSvc.GetContext(ctx, sessionID)
	if err != nil {
		return nil
	}

	return convertFromMessagePkg(dbMsgs)
}

// ClearMessages clears all messages from a session.
func (s *PersistentSessionStore) ClearMessages(sessionID string) bool {
	ctx := context.Background()
	if err := s.messageSvc.Clear(ctx, sessionID); err != nil {
		return false
	}

	s.mu.Lock()
	if sess, ok := s.cache[sessionID]; ok {
		sess.Messages = []Message{}
		sess.UpdatedAt = time.Now()
	}
	s.mu.Unlock()

	return true
}

// UpdateTitle updates a session's title.
func (s *PersistentSessionStore) UpdateTitle(sessionID, title string) bool {
	ctx := context.Background()
	if err := s.sessionSvc.UpdateTitle(ctx, sessionID, title); err != nil {
		return false
	}

	s.mu.Lock()
	if sess, ok := s.cache[sessionID]; ok {
		sess.Title = title
		sess.UpdatedAt = time.Now()
	}
	s.mu.Unlock()

	return true
}

// createInMemory creates an in-memory session as fallback.
func (s *PersistentSessionStore) createInMemory(title string) *Session {
	id := uuid.New().String()
	now := time.Now()
	sess := &Session{
		ID:        id,
		Title:     title,
		Messages:  []Message{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mu.Lock()
	s.cache[id] = sess
	s.mu.Unlock()

	return sess
}

// convertFromMessagePkg converts message.Message slice to agent.Message slice.
func convertFromMessagePkg(dbMsgs []*message.Message) []Message {
	msgs := make([]Message, len(dbMsgs))
	for i, dbm := range dbMsgs {
		msgs[i] = Message{
			ID:        dbm.ID,
			Role:      Role(dbm.Role),
			Content:   dbm.TextContent(),
			Reasoning: dbm.ReasoningContent(),
			CreatedAt: dbm.CreatedAt,
		}

		// Convert tool calls from parts
		for _, tc := range dbm.ToolCalls() {
			msgs[i].ToolCalls = append(msgs[i].ToolCalls, ToolCall{
				ID:    tc.ID,
				Name:  tc.Name,
				Input: tc.Input,
			})
		}

		// Convert tool results from parts
		for _, tr := range dbm.ToolResults() {
			msgs[i].ToolResults = append(msgs[i].ToolResults, ToolResult{
				ToolCallID: tr.ToolCallID,
				Name:       tr.Name,
				Content:    tr.Content,
				IsError:    tr.IsError,
			})
		}
	}
	return msgs
}

// convertToMessageParts converts an agent.Message to message.Part slice.
func convertToMessageParts(msg Message) []message.Part {
	var parts []message.Part

	if msg.Content != "" {
		parts = append(parts, message.NewTextPart(msg.Content))
	}

	if msg.Reasoning != "" {
		parts = append(parts, message.NewReasoningPart(msg.Reasoning))
	}

	for _, tc := range msg.ToolCalls {
		parts = append(parts, message.NewToolCallPart(tc.ID, tc.Name, tc.Input))
	}

	for _, tr := range msg.ToolResults {
		parts = append(parts, message.NewToolResultPart(tr.ToolCallID, tr.Name, tr.Content, tr.IsError))
	}

	return parts
}
