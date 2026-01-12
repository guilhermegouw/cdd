package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/guilhermegouw/cdd/internal/db/sqlc"
)

// ErrNotFound is returned when a session is not found.
var ErrNotFound = errors.New("session not found")

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	queries *sqlc.Queries
}

// NewSQLiteStore creates a new SQLite-backed session store.
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{
		queries: sqlc.New(db),
	}
}

// Create creates a new session with the given ID and title.
func (s *SQLiteStore) Create(ctx context.Context, id, title string) (*Session, error) {
	now := time.Now().UnixMilli()

	dbSession, err := s.queries.CreateSession(ctx, sqlc.CreateSessionParams{
		ID:        id,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}

	return sessionFromDB(dbSession), nil
}

// Get retrieves a session by ID.
func (s *SQLiteStore) Get(ctx context.Context, id string) (*Session, error) {
	dbSession, err := s.queries.GetSession(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting session: %w", err)
	}

	return sessionFromDB(dbSession), nil
}

// List returns all sessions ordered by updated_at descending.
func (s *SQLiteStore) List(ctx context.Context) ([]*Session, error) {
	dbSessions, err := s.queries.ListSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}

	sessions := make([]*Session, len(dbSessions))
	for i, dbs := range dbSessions {
		sessions[i] = sessionFromDB(dbs)
	}

	return sessions, nil
}

// UpdateTitle updates the title of a session.
func (s *SQLiteStore) UpdateTitle(ctx context.Context, id, title string) error {
	now := time.Now().UnixMilli()

	err := s.queries.UpdateSessionTitle(ctx, sqlc.UpdateSessionTitleParams{
		Title:     title,
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return fmt.Errorf("updating session title: %w", err)
	}

	return nil
}

// IncrementMessageCount increments the message count for a session.
func (s *SQLiteStore) IncrementMessageCount(ctx context.Context, id string) error {
	now := time.Now().UnixMilli()

	err := s.queries.UpdateSessionMessageCount(ctx, sqlc.UpdateSessionMessageCountParams{
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return fmt.Errorf("incrementing message count: %w", err)
	}

	return nil
}

// DecrementMessageCount decrements the message count for a session.
func (s *SQLiteStore) DecrementMessageCount(ctx context.Context, id string) error {
	now := time.Now().UnixMilli()

	err := s.queries.DecrementSessionMessageCount(ctx, sqlc.DecrementSessionMessageCountParams{
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return fmt.Errorf("decrementing message count: %w", err)
	}

	return nil
}

// SetSummaryMessage sets the summary message ID for a session.
func (s *SQLiteStore) SetSummaryMessage(ctx context.Context, sessionID, messageID string) error {
	now := time.Now().UnixMilli()

	err := s.queries.SetSessionSummary(ctx, sqlc.SetSessionSummaryParams{
		SummaryMessageID: sql.NullString{String: messageID, Valid: messageID != ""},
		UpdatedAt:        now,
		ID:               sessionID,
	})
	if err != nil {
		return fmt.Errorf("setting summary message: %w", err)
	}

	return nil
}

// Delete removes a session by ID.
func (s *SQLiteStore) Delete(ctx context.Context, id string) error {
	err := s.queries.DeleteSession(ctx, id)
	if err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}

	return nil
}

// ListWithPreview returns all sessions with first message preview.
func (s *SQLiteStore) ListWithPreview(ctx context.Context) ([]*SessionWithPreview, error) {
	dbSessions, err := s.queries.ListSessionsWithPreview(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing sessions with preview: %w", err)
	}

	sessions := make([]*SessionWithPreview, len(dbSessions))
	for i, dbs := range dbSessions {
		sessions[i] = sessionWithPreviewFromDB(dbs)
	}

	return sessions, nil
}

// Search searches sessions by title keyword.
// Supports multi-word search: "bug auth" matches "Authentication Bug Fix".
func (s *SQLiteStore) Search(ctx context.Context, keyword string) ([]*Session, error) {
	// Preprocess keyword for multi-word search
	searchTerm := prepareSearchTerm(keyword)
	dbSessions, err := s.queries.SearchSessions(ctx, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("searching sessions: %w", err)
	}

	sessions := make([]*Session, len(dbSessions))
	for i, dbs := range dbSessions {
		sessions[i] = sessionFromDB(dbs)
	}

	return sessions, nil
}

// SearchWithPreview searches sessions by title with first message preview.
// Supports multi-word search: "bug auth" matches "Authentication Bug Fix".
func (s *SQLiteStore) SearchWithPreview(ctx context.Context, keyword string) ([]*SessionWithPreview, error) {
	// Preprocess keyword for multi-word search
	searchTerm := prepareSearchTerm(keyword)
	dbSessions, err := s.queries.SearchSessionsWithPreview(ctx, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("searching sessions with preview: %w", err)
	}

	sessions := make([]*SessionWithPreview, len(dbSessions))
	for i, dbs := range dbSessions {
		sessions[i] = searchSessionWithPreviewFromDB(dbs)
	}

	return sessions, nil
}

// prepareSearchTerm converts a search keyword for multi-word matching.
// "bug auth" becomes "bug%auth" to match titles containing both words.
func prepareSearchTerm(keyword string) string {
	// Trim whitespace and replace spaces with % for multi-word matching
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return ""
	}
	// Replace multiple spaces with single % and trim
	parts := strings.Fields(keyword)
	return strings.Join(parts, "%")
}

// sessionFromDB converts a database session to a domain session.
func sessionFromDB(dbs sqlc.Session) *Session {
	var summaryID string
	if dbs.SummaryMessageID.Valid {
		summaryID = dbs.SummaryMessageID.String
	}

	return &Session{
		ID:               dbs.ID,
		Title:            dbs.Title,
		MessageCount:     int(dbs.MessageCount),
		SummaryMessageID: summaryID,
		CreatedAt:        time.UnixMilli(dbs.CreatedAt),
		UpdatedAt:        time.UnixMilli(dbs.UpdatedAt),
	}
}

// sessionPreviewData holds common fields for session with preview conversion.
type sessionPreviewData struct {
	ID               string
	Title            string
	MessageCount     int64
	SummaryMessageID sql.NullString
	CreatedAt        int64
	UpdatedAt        int64
	FirstMessage     any
}

// buildSessionWithPreview creates a SessionWithPreview from common data.
func buildSessionWithPreview(data sessionPreviewData) *SessionWithPreview {
	var summaryID string
	if data.SummaryMessageID.Valid {
		summaryID = data.SummaryMessageID.String
	}

	// Handle interface{} type from SQLC - convert to string
	firstMsgStr := ""
	if data.FirstMessage != nil {
		if s, ok := data.FirstMessage.(string); ok {
			firstMsgStr = s
		}
	}
	firstMsg := extractTextFromParts(firstMsgStr)

	return &SessionWithPreview{
		Session: Session{
			ID:               data.ID,
			Title:            data.Title,
			MessageCount:     int(data.MessageCount),
			SummaryMessageID: summaryID,
			CreatedAt:        time.UnixMilli(data.CreatedAt),
			UpdatedAt:        time.UnixMilli(data.UpdatedAt),
		},
		FirstMessage: firstMsg,
	}
}

// sessionWithPreviewFromDB converts a database session with preview to domain type.
func sessionWithPreviewFromDB(dbs sqlc.ListSessionsWithPreviewRow) *SessionWithPreview {
	return buildSessionWithPreview(sessionPreviewData{
		ID:               dbs.ID,
		Title:            dbs.Title,
		MessageCount:     dbs.MessageCount,
		SummaryMessageID: dbs.SummaryMessageID,
		CreatedAt:        dbs.CreatedAt,
		UpdatedAt:        dbs.UpdatedAt,
		FirstMessage:     dbs.FirstMessage,
	})
}

// searchSessionWithPreviewFromDB converts a search result with preview to domain type.
func searchSessionWithPreviewFromDB(dbs sqlc.SearchSessionsWithPreviewRow) *SessionWithPreview {
	return buildSessionWithPreview(sessionPreviewData{
		ID:               dbs.ID,
		Title:            dbs.Title,
		MessageCount:     dbs.MessageCount,
		SummaryMessageID: dbs.SummaryMessageID,
		CreatedAt:        dbs.CreatedAt,
		UpdatedAt:        dbs.UpdatedAt,
		FirstMessage:     dbs.FirstMessage,
	})
}

// extractTextFromParts extracts text content from JSON parts array.
func extractTextFromParts(partsJSON string) string {
	if partsJSON == "" {
		return ""
	}

	// Simple extraction - look for "text" field in first text part
	// Format: [{"type":"text","text":"..."}]
	type part struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	var parts []part
	if err := json.Unmarshal([]byte(partsJSON), &parts); err != nil {
		return ""
	}
	for _, p := range parts {
		if p.Type == "text" && p.Text != "" {
			return p.Text
		}
	}
	return ""
}
