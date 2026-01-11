package message

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/guilhermegouw/cdd/internal/db/sqlc"
)

// ErrNotFound is returned when a message is not found.
var ErrNotFound = errors.New("message not found")

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	queries *sqlc.Queries
}

// NewSQLiteStore creates a new SQLite-backed message store.
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{
		queries: sqlc.New(db),
	}
}

// Create creates a new message.
func (s *SQLiteStore) Create(ctx context.Context, msg *Message) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}

	now := time.Now()
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = now
	}
	msg.UpdatedAt = now

	partsJSON, err := json.Marshal(msg.Parts)
	if err != nil {
		return fmt.Errorf("marshaling parts: %w", err)
	}

	isSummary := int64(0)
	if msg.IsSummary {
		isSummary = 1
	}

	_, err = s.queries.CreateMessage(ctx, sqlc.CreateMessageParams{
		ID:        msg.ID,
		SessionID: msg.SessionID,
		Role:      string(msg.Role),
		Parts:     string(partsJSON),
		Model:     sql.NullString{String: msg.Model, Valid: msg.Model != ""},
		Provider:  sql.NullString{String: msg.Provider, Valid: msg.Provider != ""},
		IsSummary: isSummary,
		CreatedAt: msg.CreatedAt.UnixMilli(),
		UpdatedAt: msg.UpdatedAt.UnixMilli(),
	})
	if err != nil {
		return fmt.Errorf("creating message: %w", err)
	}

	return nil
}

// Get retrieves a message by ID.
func (s *SQLiteStore) Get(ctx context.Context, id string) (*Message, error) {
	dbMsg, err := s.queries.GetMessage(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting message: %w", err)
	}

	return messageFromDB(dbMsg)
}

// GetBySession returns all messages for a session.
func (s *SQLiteStore) GetBySession(ctx context.Context, sessionID string) ([]*Message, error) {
	dbMsgs, err := s.queries.GetSessionMessages(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("getting session messages: %w", err)
	}

	return messagesFromDB(dbMsgs)
}

// GetBySessionWithLimit returns messages for a session with a limit.
func (s *SQLiteStore) GetBySessionWithLimit(ctx context.Context, sessionID string, limit int) ([]*Message, error) {
	dbMsgs, err := s.queries.GetSessionMessagesWithLimit(ctx, sqlc.GetSessionMessagesWithLimitParams{
		SessionID: sessionID,
		Limit:     int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("getting session messages with limit: %w", err)
	}

	return messagesFromDB(dbMsgs)
}

// GetFromMessage returns messages from a specific message ID onwards.
func (s *SQLiteStore) GetFromMessage(ctx context.Context, sessionID, messageID string) ([]*Message, error) {
	dbMsgs, err := s.queries.GetMessagesFromID(ctx, sqlc.GetMessagesFromIDParams{
		SessionID: sessionID,
		ID:        messageID,
	})
	if err != nil {
		return nil, fmt.Errorf("getting messages from ID: %w", err)
	}

	return messagesFromDB(dbMsgs)
}

// GetSummary returns the most recent summary message for a session.
func (s *SQLiteStore) GetSummary(ctx context.Context, sessionID string) (*Message, error) {
	dbMsg, err := s.queries.GetSummaryMessage(ctx, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting summary message: %w", err)
	}

	return messageFromDB(dbMsg)
}

// Count returns the number of messages in a session.
func (s *SQLiteStore) Count(ctx context.Context, sessionID string) (int64, error) {
	count, err := s.queries.CountSessionMessages(ctx, sessionID)
	if err != nil {
		return 0, fmt.Errorf("counting messages: %w", err)
	}

	return count, nil
}

// Update updates a message's parts.
func (s *SQLiteStore) Update(ctx context.Context, msg *Message) error {
	msg.UpdatedAt = time.Now()

	partsJSON, err := json.Marshal(msg.Parts)
	if err != nil {
		return fmt.Errorf("marshaling parts: %w", err)
	}

	err = s.queries.UpdateMessageParts(ctx, sqlc.UpdateMessagePartsParams{
		Parts:     string(partsJSON),
		UpdatedAt: msg.UpdatedAt.UnixMilli(),
		ID:        msg.ID,
	})
	if err != nil {
		return fmt.Errorf("updating message: %w", err)
	}

	return nil
}

// Delete removes a message by ID.
func (s *SQLiteStore) Delete(ctx context.Context, id string) error {
	err := s.queries.DeleteMessage(ctx, id)
	if err != nil {
		return fmt.Errorf("deleting message: %w", err)
	}

	return nil
}

// DeleteBySession removes all messages for a session.
func (s *SQLiteStore) DeleteBySession(ctx context.Context, sessionID string) error {
	err := s.queries.DeleteSessionMessages(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("deleting session messages: %w", err)
	}

	return nil
}

// DeleteOldMessages removes old messages keeping only the most recent ones.
func (s *SQLiteStore) DeleteOldMessages(ctx context.Context, sessionID string, keepCount int) error {
	err := s.queries.DeleteOldMessages(ctx, sqlc.DeleteOldMessagesParams{
		SessionID: sessionID,
		Limit:     int64(keepCount),
	})
	if err != nil {
		return fmt.Errorf("deleting old messages: %w", err)
	}

	return nil
}

// messageFromDB converts a database message to a domain message.
func messageFromDB(dbMsg sqlc.Message) (*Message, error) {
	var parts []Part
	if err := json.Unmarshal([]byte(dbMsg.Parts), &parts); err != nil {
		return nil, fmt.Errorf("unmarshaling parts: %w", err)
	}

	var model, provider string
	if dbMsg.Model.Valid {
		model = dbMsg.Model.String
	}
	if dbMsg.Provider.Valid {
		provider = dbMsg.Provider.String
	}

	return &Message{
		ID:        dbMsg.ID,
		SessionID: dbMsg.SessionID,
		Role:      Role(dbMsg.Role),
		Parts:     parts,
		Model:     model,
		Provider:  provider,
		IsSummary: dbMsg.IsSummary == 1,
		CreatedAt: time.UnixMilli(dbMsg.CreatedAt),
		UpdatedAt: time.UnixMilli(dbMsg.UpdatedAt),
	}, nil
}

// messagesFromDB converts database messages to domain messages.
func messagesFromDB(dbMsgs []sqlc.Message) ([]*Message, error) {
	msgs := make([]*Message, len(dbMsgs))
	for i, dbm := range dbMsgs {
		msg, err := messageFromDB(dbm)
		if err != nil {
			return nil, err
		}
		msgs[i] = msg
	}
	return msgs, nil
}
