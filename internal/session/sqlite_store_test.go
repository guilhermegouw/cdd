package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/guilhermegouw/cdd/internal/db"
)

// setupTestDB creates an in-memory database for testing.
func setupTestDB(t *testing.T) *db.DB {
	t.Helper()

	tmpDir := t.TempDir()
	database, err := db.Open(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() }) //nolint:errcheck // Intentionally ignoring close error in test cleanup

	return database
}

func TestSQLiteStore_Create(t *testing.T) {
	database := setupTestDB(t)
	store := NewSQLiteStore(database.Conn())
	ctx := context.Background()

	t.Run("creates session with ID and title", func(t *testing.T) {
		session, err := store.Create(ctx, "test-id", "Test Session")
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if session.ID != "test-id" {
			t.Errorf("ID = %q, want %q", session.ID, "test-id")
		}
		if session.Title != "Test Session" {
			t.Errorf("Title = %q, want %q", session.Title, "Test Session")
		}
		if session.MessageCount != 0 {
			t.Errorf("MessageCount = %d, want 0", session.MessageCount)
		}
		if session.CreatedAt.IsZero() {
			t.Error("CreatedAt should not be zero")
		}
		if session.UpdatedAt.IsZero() {
			t.Error("UpdatedAt should not be zero")
		}
	})

	t.Run("fails on duplicate ID", func(t *testing.T) {
		_, err := store.Create(ctx, "dup-id", "First")
		if err != nil {
			t.Fatalf("first Create() error = %v", err)
		}

		_, err = store.Create(ctx, "dup-id", "Second")
		if err == nil {
			t.Error("expected error for duplicate ID, got nil")
		}
	})
}

func TestSQLiteStore_Get(t *testing.T) {
	database := setupTestDB(t)
	store := NewSQLiteStore(database.Conn())
	ctx := context.Background()

	t.Run("returns existing session", func(t *testing.T) {
		created, err := store.Create(ctx, "get-test", "Test Session")
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		session, err := store.Get(ctx, "get-test")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if session.ID != created.ID {
			t.Errorf("ID = %q, want %q", session.ID, created.ID)
		}
		if session.Title != created.Title {
			t.Errorf("Title = %q, want %q", session.Title, created.Title)
		}
	})

	t.Run("returns ErrNotFound for missing session", func(t *testing.T) {
		_, err := store.Get(ctx, "non-existent")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Get() error = %v, want ErrNotFound", err)
		}
	})
}

func TestSQLiteStore_List(t *testing.T) {
	database := setupTestDB(t)
	store := NewSQLiteStore(database.Conn())
	ctx := context.Background()

	t.Run("returns empty list when no sessions", func(t *testing.T) {
		sessions, err := store.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(sessions) != 0 {
			t.Errorf("List() returned %d sessions, want 0", len(sessions))
		}
	})

	t.Run("returns sessions ordered by updated_at desc", func(t *testing.T) {
		if _, err := store.Create(ctx, "list-1", "First"); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		time.Sleep(10 * time.Millisecond)
		if _, err := store.Create(ctx, "list-2", "Second"); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		time.Sleep(10 * time.Millisecond)
		if _, err := store.Create(ctx, "list-3", "Third"); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		sessions, err := store.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(sessions) != 3 {
			t.Fatalf("List() returned %d sessions, want 3", len(sessions))
		}

		// Most recent first
		if sessions[0].ID != "list-3" {
			t.Errorf("sessions[0].ID = %q, want %q", sessions[0].ID, "list-3")
		}
		if sessions[2].ID != "list-1" {
			t.Errorf("sessions[2].ID = %q, want %q", sessions[2].ID, "list-1")
		}
	})
}

//nolint:dupl // Test structure is intentionally similar to TestSQLiteStore_SetSummaryMessage
func TestSQLiteStore_UpdateTitle(t *testing.T) {
	database := setupTestDB(t)
	store := NewSQLiteStore(database.Conn())
	ctx := context.Background()

	if _, err := store.Create(ctx, "update-title", "Original Title"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	err := store.UpdateTitle(ctx, "update-title", "New Title")
	if err != nil {
		t.Fatalf("UpdateTitle() error = %v", err)
	}

	session, err := store.Get(ctx, "update-title")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if session.Title != "New Title" {
		t.Errorf("Title = %q, want %q", session.Title, "New Title")
	}
}

func TestSQLiteStore_MessageCount(t *testing.T) {
	database := setupTestDB(t)
	store := NewSQLiteStore(database.Conn())
	ctx := context.Background()

	if _, err := store.Create(ctx, "msg-count", "Test"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("increment increases count", func(t *testing.T) {
		err := store.IncrementMessageCount(ctx, "msg-count")
		if err != nil {
			t.Fatalf("IncrementMessageCount() error = %v", err)
		}

		session, err := store.Get(ctx, "msg-count")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if session.MessageCount != 1 {
			t.Errorf("MessageCount = %d, want 1", session.MessageCount)
		}
	})

	t.Run("decrement decreases count", func(t *testing.T) {
		if err := store.IncrementMessageCount(ctx, "msg-count"); err != nil {
			t.Fatalf("IncrementMessageCount() error = %v", err)
		}

		err := store.DecrementMessageCount(ctx, "msg-count")
		if err != nil {
			t.Fatalf("DecrementMessageCount() error = %v", err)
		}

		session, err := store.Get(ctx, "msg-count")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if session.MessageCount != 1 {
			t.Errorf("MessageCount = %d, want 1", session.MessageCount)
		}
	})
}

//nolint:dupl // Test structure is intentionally similar to TestSQLiteStore_UpdateTitle
func TestSQLiteStore_SetSummaryMessage(t *testing.T) {
	database := setupTestDB(t)
	store := NewSQLiteStore(database.Conn())
	ctx := context.Background()

	if _, err := store.Create(ctx, "summary-test", "Test"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	err := store.SetSummaryMessage(ctx, "summary-test", "msg-123")
	if err != nil {
		t.Fatalf("SetSummaryMessage() error = %v", err)
	}

	session, err := store.Get(ctx, "summary-test")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if session.SummaryMessageID != "msg-123" {
		t.Errorf("SummaryMessageID = %q, want %q", session.SummaryMessageID, "msg-123")
	}
}

func TestSQLiteStore_Delete(t *testing.T) {
	database := setupTestDB(t)
	store := NewSQLiteStore(database.Conn())
	ctx := context.Background()

	if _, err := store.Create(ctx, "delete-test", "Test"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	err := store.Delete(ctx, "delete-test")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = store.Get(ctx, "delete-test")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() after delete error = %v, want ErrNotFound", err)
	}
}

func TestSQLiteStore_Search(t *testing.T) {
	database := setupTestDB(t)
	store := NewSQLiteStore(database.Conn())
	ctx := context.Background()

	if _, err := store.Create(ctx, "s1", "Authentication Bug Fix"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := store.Create(ctx, "s2", "Add Login Feature"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := store.Create(ctx, "s3", "Database Migration"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("finds sessions by keyword", func(t *testing.T) {
		sessions, err := store.Search(ctx, "Bug")
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if len(sessions) != 1 {
			t.Errorf("Search() returned %d sessions, want 1", len(sessions))
		}
		if sessions[0].ID != "s1" {
			t.Errorf("sessions[0].ID = %q, want %q", sessions[0].ID, "s1")
		}
	})

	t.Run("case insensitive search", func(t *testing.T) {
		sessions, err := store.Search(ctx, "bug")
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if len(sessions) != 1 {
			t.Errorf("Search() returned %d sessions, want 1", len(sessions))
		}
	})

	t.Run("multi-word search", func(t *testing.T) {
		sessions, err := store.Search(ctx, "auth bug")
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if len(sessions) != 1 {
			t.Errorf("Search() returned %d sessions, want 1", len(sessions))
		}
	})

	t.Run("returns empty for no match", func(t *testing.T) {
		sessions, err := store.Search(ctx, "xyz")
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if len(sessions) != 0 {
			t.Errorf("Search() returned %d sessions, want 0", len(sessions))
		}
	})
}

func TestPrepareSearchTerm(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"bug", "bug"},
		{"bug fix", "bug%fix"},
		{"  bug   fix  ", "bug%fix"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := prepareSearchTerm(tt.input)
			if got != tt.want {
				t.Errorf("prepareSearchTerm(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractTextFromParts(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "extracts text from parts",
			input: `[{"type":"text","text":"Hello world"}]`,
			want:  "Hello world",
		},
		{
			name:  "returns empty for no text part",
			input: `[{"type":"reasoning","reasoning":"thinking..."}]`,
			want:  "",
		},
		{
			name:  "returns empty for invalid JSON",
			input: `invalid json`,
			want:  "",
		},
		{
			name:  "returns empty for empty string",
			input: "",
			want:  "",
		},
		{
			name:  "returns first text part",
			input: `[{"type":"reasoning","reasoning":"thinking"},{"type":"text","text":"result"}]`,
			want:  "result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTextFromParts(tt.input)
			if got != tt.want {
				t.Errorf("extractTextFromParts() = %q, want %q", got, tt.want)
			}
		})
	}
}
