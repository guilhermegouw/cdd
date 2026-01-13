package message

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

// createTestSession creates a session for message tests.
func createTestSession(t *testing.T, database *db.DB, id string) {
	t.Helper()
	ctx := context.Background()
	_, err := database.ExecContext(ctx,
		"INSERT INTO sessions (id, title, created_at, updated_at) VALUES (?, 'Test', ?, ?)",
		id, time.Now().UnixMilli(), time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
}

func TestSQLiteStore_Create(t *testing.T) {
	database := setupTestDB(t)
	store := NewSQLiteStore(database.Conn())
	ctx := context.Background()
	createTestSession(t, database, "sess-1")

	t.Run("creates message with all fields", func(t *testing.T) {
		msg := &Message{
			ID:        "msg-1",
			SessionID: "sess-1",
			Role:      RoleUser,
			Parts:     []Part{NewTextPart("Hello")},
			Model:     "gpt-4",
			Provider:  "openai",
		}

		err := store.Create(ctx, msg)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Verify timestamps were set
		if msg.CreatedAt.IsZero() {
			t.Error("CreatedAt should not be zero")
		}
		if msg.UpdatedAt.IsZero() {
			t.Error("UpdatedAt should not be zero")
		}
	})

	t.Run("generates ID if empty", func(t *testing.T) {
		msg := &Message{
			SessionID: "sess-1",
			Role:      RoleUser,
			Parts:     []Part{NewTextPart("Hello")},
		}

		err := store.Create(ctx, msg)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if msg.ID == "" {
			t.Error("ID should be generated")
		}
	})

	t.Run("creates summary message", func(t *testing.T) {
		msg := &Message{
			ID:        "msg-summary",
			SessionID: "sess-1",
			Role:      RoleSystem,
			Parts:     []Part{NewTextPart("Summary of conversation")},
			IsSummary: true,
		}

		err := store.Create(ctx, msg)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		retrieved, err := store.Get(ctx, "msg-summary")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if !retrieved.IsSummary {
			t.Error("IsSummary = false, want true")
		}
	})
}

func TestSQLiteStore_Get(t *testing.T) {
	database := setupTestDB(t)
	store := NewSQLiteStore(database.Conn())
	ctx := context.Background()
	createTestSession(t, database, "sess-1")

	t.Run("returns existing message", func(t *testing.T) {
		original := &Message{
			ID:        "get-test",
			SessionID: "sess-1",
			Role:      RoleAssistant,
			Parts: []Part{
				NewTextPart("Hello"),
				NewToolCallPart("call-1", "read_file", `{"path": "/tmp"}`),
			},
			Model:    "claude-3",
			Provider: "anthropic",
		}
		if err := store.Create(ctx, original); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		msg, err := store.Get(ctx, "get-test")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if msg.ID != original.ID {
			t.Errorf("ID = %q, want %q", msg.ID, original.ID)
		}
		if msg.Role != original.Role {
			t.Errorf("Role = %q, want %q", msg.Role, original.Role)
		}
		if len(msg.Parts) != 2 {
			t.Errorf("Parts length = %d, want 2", len(msg.Parts))
		}
		if msg.Model != "claude-3" {
			t.Errorf("Model = %q, want %q", msg.Model, "claude-3")
		}
	})

	t.Run("returns ErrNotFound for missing message", func(t *testing.T) {
		_, err := store.Get(ctx, "non-existent")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Get() error = %v, want ErrNotFound", err)
		}
	})
}

func TestSQLiteStore_GetBySession(t *testing.T) {
	database := setupTestDB(t)
	store := NewSQLiteStore(database.Conn())
	ctx := context.Background()
	createTestSession(t, database, "sess-1")
	createTestSession(t, database, "sess-2")

	// Create messages for sess-1
	if err := store.Create(ctx, &Message{ID: "m1", SessionID: "sess-1", Role: RoleUser, Parts: []Part{NewTextPart("First")}}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := store.Create(ctx, &Message{ID: "m2", SessionID: "sess-1", Role: RoleAssistant, Parts: []Part{NewTextPart("Second")}}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Create message for sess-2
	if err := store.Create(ctx, &Message{ID: "m3", SessionID: "sess-2", Role: RoleUser, Parts: []Part{NewTextPart("Other")}}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("returns messages for session ordered by created_at", func(t *testing.T) {
		msgs, err := store.GetBySession(ctx, "sess-1")
		if err != nil {
			t.Fatalf("GetBySession() error = %v", err)
		}

		if len(msgs) != 2 {
			t.Fatalf("GetBySession() returned %d messages, want 2", len(msgs))
		}

		if msgs[0].ID != "m1" {
			t.Errorf("msgs[0].ID = %q, want %q", msgs[0].ID, "m1")
		}
		if msgs[1].ID != "m2" {
			t.Errorf("msgs[1].ID = %q, want %q", msgs[1].ID, "m2")
		}
	})

	t.Run("returns empty for session with no messages", func(t *testing.T) {
		createTestSession(t, database, "sess-empty")
		msgs, err := store.GetBySession(ctx, "sess-empty")
		if err != nil {
			t.Fatalf("GetBySession() error = %v", err)
		}
		if len(msgs) != 0 {
			t.Errorf("GetBySession() returned %d messages, want 0", len(msgs))
		}
	})
}

func TestSQLiteStore_GetBySessionWithLimit(t *testing.T) {
	database := setupTestDB(t)
	store := NewSQLiteStore(database.Conn())
	ctx := context.Background()
	createTestSession(t, database, "sess-1")

	// Create 5 messages
	for i := 1; i <= 5; i++ {
		if err := store.Create(ctx, &Message{
			SessionID: "sess-1",
			Role:      RoleUser,
			Parts:     []Part{NewTextPart("Message")},
		}); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}

	msgs, err := store.GetBySessionWithLimit(ctx, "sess-1", 3)
	if err != nil {
		t.Fatalf("GetBySessionWithLimit() error = %v", err)
	}

	if len(msgs) != 3 {
		t.Errorf("GetBySessionWithLimit() returned %d messages, want 3", len(msgs))
	}
}

func TestSQLiteStore_GetSummary(t *testing.T) {
	database := setupTestDB(t)
	store := NewSQLiteStore(database.Conn())
	ctx := context.Background()
	createTestSession(t, database, "sess-1")

	// Create regular message
	if err := store.Create(ctx, &Message{ID: "reg-1", SessionID: "sess-1", Role: RoleUser, Parts: []Part{NewTextPart("Hello")}}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Create summary message
	if err := store.Create(ctx, &Message{
		ID:        "sum-1",
		SessionID: "sess-1",
		Role:      RoleSystem,
		Parts:     []Part{NewTextPart("Summary")},
		IsSummary: true,
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("returns summary message", func(t *testing.T) {
		msg, err := store.GetSummary(ctx, "sess-1")
		if err != nil {
			t.Fatalf("GetSummary() error = %v", err)
		}

		if msg.ID != "sum-1" {
			t.Errorf("ID = %q, want %q", msg.ID, "sum-1")
		}
		if !msg.IsSummary {
			t.Error("IsSummary = false, want true")
		}
	})

	t.Run("returns ErrNotFound when no summary", func(t *testing.T) {
		createTestSession(t, database, "sess-no-summary")
		if err := store.Create(ctx, &Message{SessionID: "sess-no-summary", Role: RoleUser, Parts: []Part{NewTextPart("Hi")}}); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		_, err := store.GetSummary(ctx, "sess-no-summary")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("GetSummary() error = %v, want ErrNotFound", err)
		}
	})
}

func TestSQLiteStore_Count(t *testing.T) {
	database := setupTestDB(t)
	store := NewSQLiteStore(database.Conn())
	ctx := context.Background()
	createTestSession(t, database, "sess-1")

	t.Run("returns zero for empty session", func(t *testing.T) {
		count, err := store.Count(ctx, "sess-1")
		if err != nil {
			t.Fatalf("Count() error = %v", err)
		}
		if count != 0 {
			t.Errorf("Count() = %d, want 0", count)
		}
	})

	t.Run("returns correct count", func(t *testing.T) {
		for _, content := range []string{"1", "2", "3"} {
			if err := store.Create(ctx, &Message{SessionID: "sess-1", Role: RoleUser, Parts: []Part{NewTextPart(content)}}); err != nil {
				t.Fatalf("Create() error = %v", err)
			}
		}

		count, err := store.Count(ctx, "sess-1")
		if err != nil {
			t.Fatalf("Count() error = %v", err)
		}
		if count != 3 {
			t.Errorf("Count() = %d, want 3", count)
		}
	})
}

func TestSQLiteStore_Update(t *testing.T) {
	database := setupTestDB(t)
	store := NewSQLiteStore(database.Conn())
	ctx := context.Background()
	createTestSession(t, database, "sess-1")

	original := &Message{
		ID:        "update-test",
		SessionID: "sess-1",
		Role:      RoleAssistant,
		Parts:     []Part{NewTextPart("Original")},
	}
	if err := store.Create(ctx, original); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update parts
	original.Parts = []Part{NewTextPart("Updated content")}
	err := store.Update(ctx, original)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	updated, err := store.Get(ctx, "update-test")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if updated.TextContent() != "Updated content" {
		t.Errorf("TextContent() = %q, want %q", updated.TextContent(), "Updated content")
	}
}

func TestSQLiteStore_Delete(t *testing.T) {
	database := setupTestDB(t)
	store := NewSQLiteStore(database.Conn())
	ctx := context.Background()
	createTestSession(t, database, "sess-1")

	if err := store.Create(ctx, &Message{ID: "del-test", SessionID: "sess-1", Role: RoleUser, Parts: []Part{NewTextPart("Delete me")}}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	err := store.Delete(ctx, "del-test")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = store.Get(ctx, "del-test")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() after delete error = %v, want ErrNotFound", err)
	}
}

func TestSQLiteStore_DeleteBySession(t *testing.T) {
	database := setupTestDB(t)
	store := NewSQLiteStore(database.Conn())
	ctx := context.Background()
	createTestSession(t, database, "sess-1")

	for _, content := range []string{"1", "2"} {
		if err := store.Create(ctx, &Message{SessionID: "sess-1", Role: RoleUser, Parts: []Part{NewTextPart(content)}}); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	err := store.DeleteBySession(ctx, "sess-1")
	if err != nil {
		t.Fatalf("DeleteBySession() error = %v", err)
	}

	count, err := store.Count(ctx, "sess-1")
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 0 {
		t.Errorf("Count() after delete = %d, want 0", count)
	}
}

func TestSQLiteStore_DeleteOldMessages(t *testing.T) {
	database := setupTestDB(t)
	store := NewSQLiteStore(database.Conn())
	ctx := context.Background()
	createTestSession(t, database, "sess-1")

	// Create 5 messages with distinct timestamps
	for i := 1; i <= 5; i++ {
		if err := store.Create(ctx, &Message{
			SessionID: "sess-1",
			Role:      RoleUser,
			Parts:     []Part{NewTextPart("Message")},
		}); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Keep only 2 most recent
	err := store.DeleteOldMessages(ctx, "sess-1", 2)
	if err != nil {
		t.Fatalf("DeleteOldMessages() error = %v", err)
	}

	count, err := store.Count(ctx, "sess-1")
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 2 {
		t.Errorf("Count() after DeleteOldMessages = %d, want 2", count)
	}
}

func TestSQLiteStore_PartsRoundTrip(t *testing.T) {
	database := setupTestDB(t)
	store := NewSQLiteStore(database.Conn())
	ctx := context.Background()
	createTestSession(t, database, "sess-1")

	// Create message with all part types
	original := &Message{
		ID:        "roundtrip",
		SessionID: "sess-1",
		Role:      RoleAssistant,
		Parts: []Part{
			NewTextPart("Hello"),
			NewReasoningPart("Thinking about the problem"),
			NewToolCallPart("call-1", "read_file", `{"path": "/tmp/test.txt"}`),
			NewToolResultPart("call-1", "read_file", "file contents here", false),
		},
	}

	err := store.Create(ctx, original)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := store.Get(ctx, "roundtrip")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(retrieved.Parts) != 4 {
		t.Fatalf("Parts length = %d, want 4", len(retrieved.Parts))
	}

	// Verify text part
	if retrieved.Parts[0].Type != PartTypeText || retrieved.Parts[0].Text != "Hello" {
		t.Errorf("text part mismatch: %+v", retrieved.Parts[0])
	}

	// Verify reasoning part
	if retrieved.Parts[1].Type != PartTypeReasoning || retrieved.Parts[1].Reasoning != "Thinking about the problem" {
		t.Errorf("reasoning part mismatch: %+v", retrieved.Parts[1])
	}

	// Verify tool call part
	tc := retrieved.Parts[2].ToolCall
	if tc == nil || tc.ID != "call-1" || tc.Name != "read_file" { //nolint:goconst // Test literals are intentionally readable
		t.Errorf("tool call part mismatch: %+v", retrieved.Parts[2])
	}

	// Verify tool result part
	tr := retrieved.Parts[3].ToolResult
	if tr == nil || tr.ToolCallID != "call-1" || tr.Content != "file contents here" { //nolint:goconst // Test literals are intentionally readable
		t.Errorf("tool result part mismatch: %+v", retrieved.Parts[3])
	}
}
