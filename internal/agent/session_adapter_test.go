package agent

import (
	"testing"
	"time"

	"github.com/guilhermegouw/cdd/internal/db"
	"github.com/guilhermegouw/cdd/internal/message"
	"github.com/guilhermegouw/cdd/internal/session"
)

// setupTestStore creates a PersistentSessionStore backed by an in-memory database.
func setupTestStore(t *testing.T) *PersistentSessionStore {
	t.Helper()

	tmpDir := t.TempDir()
	database, err := db.Open(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() }) //nolint:errcheck // Intentionally ignoring close error in test cleanup

	sessionStore := session.NewSQLiteStore(database.Conn())
	sessionSvc := session.NewService(sessionStore, nil) // nil broker for tests

	messageStore := message.NewSQLiteStore(database.Conn())
	messageSvc := message.NewService(messageStore, nil)

	return NewPersistentSessionStore(sessionSvc, messageSvc)
}

func TestPersistentSessionStore_Create(t *testing.T) {
	store := setupTestStore(t)

	sess := store.Create("Test Session")
	if sess == nil {
		t.Fatal("Create() returned nil")
	}

	if sess.ID == "" {
		t.Error("ID should not be empty")
	}
	if sess.Title != "Test Session" {
		t.Errorf("Title = %q, want %q", sess.Title, "Test Session")
	}
	if len(sess.Messages) != 0 {
		t.Errorf("Messages length = %d, want 0", len(sess.Messages))
	}
	if sess.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestPersistentSessionStore_Get(t *testing.T) {
	store := setupTestStore(t)

	t.Run("returns existing session", func(t *testing.T) {
		created := store.Create("Test")

		sess, ok := store.Get(created.ID)
		if !ok {
			t.Fatal("Get() returned ok=false")
		}
		if sess.ID != created.ID {
			t.Errorf("ID = %q, want %q", sess.ID, created.ID)
		}
	})

	t.Run("returns false for non-existent session", func(t *testing.T) {
		_, ok := store.Get("non-existent-id")
		if ok {
			t.Error("Get() returned ok=true for non-existent session")
		}
	})

	t.Run("loads messages from database", func(t *testing.T) {
		created := store.Create("With Messages")

		// Add messages
		store.AddMessage(created.ID, Message{Role: RoleUser, Content: "Hello"})
		store.AddMessage(created.ID, Message{Role: RoleAssistant, Content: "Hi there"})

		// Clear cache to force reload from DB
		store.mu.Lock()
		delete(store.cache, created.ID)
		store.mu.Unlock()

		// Get should load from DB
		sess, ok := store.Get(created.ID)
		if !ok {
			t.Fatal("Get() returned ok=false")
		}
		if len(sess.Messages) != 2 {
			t.Errorf("Messages length = %d, want 2", len(sess.Messages))
		}
	})
}

func TestPersistentSessionStore_Current(t *testing.T) {
	store := setupTestStore(t)

	t.Run("creates new session when none exists", func(t *testing.T) {
		sess := store.Current()
		if sess == nil {
			t.Fatal("Current() returned nil")
		}
		if sess.ID == "" {
			t.Error("ID should not be empty")
		}
	})

	t.Run("returns same session on subsequent calls", func(t *testing.T) {
		sess1 := store.Current()
		sess2 := store.Current()

		if sess1.ID != sess2.ID {
			t.Errorf("Current() returned different sessions: %q vs %q", sess1.ID, sess2.ID)
		}
	})
}

func TestPersistentSessionStore_SetCurrent(t *testing.T) {
	store := setupTestStore(t)

	sess1 := store.Create("Session 1")
	sess2 := store.Create("Session 2")

	// Current should be sess2 (most recently created)
	if store.Current().ID != sess2.ID {
		t.Errorf("Current() = %q, want %q", store.Current().ID, sess2.ID)
	}

	// Switch to sess1
	ok := store.SetCurrent(sess1.ID)
	if !ok {
		t.Fatal("SetCurrent() returned false")
	}

	if store.Current().ID != sess1.ID {
		t.Errorf("Current() after SetCurrent = %q, want %q", store.Current().ID, sess1.ID)
	}

	// SetCurrent with invalid ID should return false
	ok = store.SetCurrent("invalid-id")
	if ok {
		t.Error("SetCurrent() with invalid ID returned true")
	}
}

func TestPersistentSessionStore_List(t *testing.T) {
	store := setupTestStore(t)

	store.Create("First")
	time.Sleep(10 * time.Millisecond)
	store.Create("Second")
	time.Sleep(10 * time.Millisecond)
	store.Create("Third")

	sessions := store.List()
	if len(sessions) != 3 {
		t.Fatalf("List() returned %d sessions, want 3", len(sessions))
	}

	// Should be ordered by updated_at descending (most recent first)
	if sessions[0].Title != "Third" {
		t.Errorf("sessions[0].Title = %q, want %q", sessions[0].Title, "Third")
	}
}

func TestPersistentSessionStore_Delete(t *testing.T) {
	store := setupTestStore(t)

	sess := store.Create("To Delete")

	ok := store.Delete(sess.ID)
	if !ok {
		t.Fatal("Delete() returned false")
	}

	_, exists := store.Get(sess.ID)
	if exists {
		t.Error("session should not exist after delete")
	}
}

func TestPersistentSessionStore_AddMessage(t *testing.T) {
	store := setupTestStore(t)
	sess := store.Create("Test")

	t.Run("adds message to session", func(t *testing.T) {
		ok := store.AddMessage(sess.ID, Message{
			Role:    RoleUser,
			Content: "Hello",
		})
		if !ok {
			t.Fatal("AddMessage() returned false")
		}

		msgs := store.GetMessages(sess.ID)
		if len(msgs) != 1 {
			t.Fatalf("GetMessages() returned %d messages, want 1", len(msgs))
		}
		if msgs[0].Content != "Hello" { //nolint:goconst // Test literals are intentionally readable
			t.Errorf("Content = %q, want %q", msgs[0].Content, "Hello")
		}
	})

	t.Run("generates ID if empty", func(t *testing.T) {
		msg := Message{Role: RoleUser, Content: "No ID"}
		store.AddMessage(sess.ID, msg)

		msgs := store.GetMessages(sess.ID)
		lastMsg := msgs[len(msgs)-1]
		if lastMsg.ID == "" {
			t.Error("message ID should be generated")
		}
	})

	t.Run("handles tool calls", func(t *testing.T) {
		ok := store.AddMessage(sess.ID, Message{
			Role:    RoleAssistant,
			Content: "Using tool",
			ToolCalls: []ToolCall{
				{ID: "call-1", Name: "read_file", Input: `{"path": "/tmp"}`},
			},
		})
		if !ok {
			t.Fatal("AddMessage() with tool calls returned false")
		}
	})

	t.Run("handles tool results", func(t *testing.T) {
		ok := store.AddMessage(sess.ID, Message{
			Role: RoleTool,
			ToolResults: []ToolResult{
				{ToolCallID: "call-1", Name: "read_file", Content: "file contents", IsError: false},
			},
		})
		if !ok {
			t.Fatal("AddMessage() with tool results returned false")
		}
	})
}

func TestPersistentSessionStore_GetMessages(t *testing.T) {
	store := setupTestStore(t)
	sess := store.Create("Test")

	store.AddMessage(sess.ID, Message{Role: RoleUser, Content: "First"})
	store.AddMessage(sess.ID, Message{Role: RoleAssistant, Content: "Second"})

	t.Run("returns messages from cache", func(t *testing.T) {
		msgs := store.GetMessages(sess.ID)
		if len(msgs) != 2 {
			t.Errorf("GetMessages() returned %d messages, want 2", len(msgs))
		}
	})

	t.Run("returns empty for non-existent session", func(t *testing.T) {
		msgs := store.GetMessages("non-existent")
		if len(msgs) != 0 {
			t.Errorf("GetMessages() for non-existent session = %d messages, want 0", len(msgs))
		}
	})
}

func TestPersistentSessionStore_ClearMessages(t *testing.T) {
	store := setupTestStore(t)
	sess := store.Create("Test")

	store.AddMessage(sess.ID, Message{Role: RoleUser, Content: "Hello"})
	store.AddMessage(sess.ID, Message{Role: RoleAssistant, Content: "Hi"})

	ok := store.ClearMessages(sess.ID)
	if !ok {
		t.Fatal("ClearMessages() returned false")
	}

	msgs := store.GetMessages(sess.ID)
	if len(msgs) != 0 {
		t.Errorf("GetMessages() after clear = %d messages, want 0", len(msgs))
	}
}

func TestPersistentSessionStore_UpdateTitle(t *testing.T) {
	store := setupTestStore(t)
	sess := store.Create("Original Title")

	ok := store.UpdateTitle(sess.ID, "New Title")
	if !ok {
		t.Fatal("UpdateTitle() returned false")
	}

	updated, _ := store.Get(sess.ID)
	if updated.Title != "New Title" {
		t.Errorf("Title = %q, want %q", updated.Title, "New Title")
	}
}

func TestConvertFromMessagePkg(t *testing.T) {
	dbMsgs := []*message.Message{
		{
			ID:        "msg-1",
			Role:      message.RoleUser,
			Parts:     []message.Part{message.NewTextPart("Hello")},
			CreatedAt: time.Now(),
		},
		{
			ID:   "msg-2",
			Role: message.RoleAssistant,
			Parts: []message.Part{
				message.NewTextPart("Hi there"),
				message.NewReasoningPart("Thinking..."),
				message.NewToolCallPart("call-1", "read_file", `{"path": "/tmp"}`),
			},
			CreatedAt: time.Now(),
		},
		{
			ID:   "msg-3",
			Role: message.RoleTool,
			Parts: []message.Part{
				message.NewToolResultPart("call-1", "read_file", "contents", false),
			},
			CreatedAt: time.Now(),
		},
	}

	agentMsgs := convertFromMessagePkg(dbMsgs)

	if len(agentMsgs) != 3 {
		t.Fatalf("converted %d messages, want 3", len(agentMsgs))
	}

	// Check user message
	if agentMsgs[0].Content != "Hello" {
		t.Errorf("msg[0].Content = %q, want %q", agentMsgs[0].Content, "Hello")
	}
	if agentMsgs[0].Role != RoleUser {
		t.Errorf("msg[0].Role = %q, want %q", agentMsgs[0].Role, RoleUser)
	}

	// Check assistant message with tool calls
	if agentMsgs[1].Content != "Hi there" {
		t.Errorf("msg[1].Content = %q, want %q", agentMsgs[1].Content, "Hi there")
	}
	if agentMsgs[1].Reasoning != "Thinking..." {
		t.Errorf("msg[1].Reasoning = %q, want %q", agentMsgs[1].Reasoning, "Thinking...")
	}
	if len(agentMsgs[1].ToolCalls) != 1 {
		t.Errorf("msg[1].ToolCalls length = %d, want 1", len(agentMsgs[1].ToolCalls))
	}
	if agentMsgs[1].ToolCalls[0].Name != "read_file" {
		t.Errorf("msg[1].ToolCalls[0].Name = %q, want %q", agentMsgs[1].ToolCalls[0].Name, "read_file")
	}

	// Check tool result message
	if len(agentMsgs[2].ToolResults) != 1 {
		t.Errorf("msg[2].ToolResults length = %d, want 1", len(agentMsgs[2].ToolResults))
	}
	if agentMsgs[2].ToolResults[0].Content != "contents" {
		t.Errorf("msg[2].ToolResults[0].Content = %q, want %q", agentMsgs[2].ToolResults[0].Content, "contents")
	}
}

func TestConvertToMessageParts(t *testing.T) {
	msg := Message{
		Content:   "Hello",
		Reasoning: "Thinking...",
		ToolCalls: []ToolCall{
			{ID: "call-1", Name: "read_file", Input: `{"path": "/tmp"}`},
		},
		ToolResults: []ToolResult{
			{ToolCallID: "call-1", Name: "read_file", Content: "contents", IsError: false},
		},
	}

	parts := convertToMessageParts(msg)

	if len(parts) != 4 {
		t.Fatalf("converted %d parts, want 4", len(parts))
	}

	// Check text part
	if parts[0].Type != message.PartTypeText || parts[0].Text != "Hello" {
		t.Errorf("parts[0] = %+v, want text part with 'Hello'", parts[0])
	}

	// Check reasoning part
	if parts[1].Type != message.PartTypeReasoning || parts[1].Reasoning != "Thinking..." {
		t.Errorf("parts[1] = %+v, want reasoning part", parts[1])
	}

	// Check tool call part
	if parts[2].Type != message.PartTypeToolCall || parts[2].ToolCall.Name != "read_file" {
		t.Errorf("parts[2] = %+v, want tool call part", parts[2])
	}

	// Check tool result part
	if parts[3].Type != message.PartTypeToolResult || parts[3].ToolResult.Content != "contents" {
		t.Errorf("parts[3] = %+v, want tool result part", parts[3])
	}
}

func TestConvertToMessageParts_EmptyFields(t *testing.T) {
	msg := Message{} // All fields empty

	parts := convertToMessageParts(msg)

	if len(parts) != 0 {
		t.Errorf("converted %d parts for empty message, want 0", len(parts))
	}
}
