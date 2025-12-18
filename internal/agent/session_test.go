package agent

import (
	"testing"
	"time"
)

//nolint:gocyclo // Test functions naturally have high complexity
func TestSessionStore(t *testing.T) {
	t.Run("create session", func(t *testing.T) {
		store := NewSessionStore()
		session := store.Create("Test Session")

		if session.ID == "" {
			t.Error("Expected session ID to be set")
		}
		if session.Title != "Test Session" {
			t.Errorf("Expected title 'Test Session', got %q", session.Title)
		}
		if len(session.Messages) != 0 {
			t.Errorf("Expected empty messages, got %d", len(session.Messages))
		}
		if session.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
	})

	t.Run("get session", func(t *testing.T) {
		store := NewSessionStore()
		created := store.Create("Test")

		retrieved, ok := store.Get(created.ID)
		if !ok {
			t.Error("Expected to find session")
		}
		if retrieved.ID != created.ID {
			t.Errorf("Expected ID %q, got %q", created.ID, retrieved.ID)
		}
	})

	t.Run("get nonexistent session", func(t *testing.T) {
		store := NewSessionStore()

		_, ok := store.Get("nonexistent")
		if ok {
			t.Error("Expected not to find nonexistent session")
		}
	})

	t.Run("current session creates if none", func(t *testing.T) {
		store := NewSessionStore()

		current := store.Current()
		if current == nil {
			t.Fatal("Expected current session to be created")
		}
		if current.Title != "New Session" {
			t.Errorf("Expected default title 'New Session', got %q", current.Title)
		}
	})

	t.Run("current session returns existing", func(t *testing.T) {
		store := NewSessionStore()
		created := store.Create("First Session")

		current := store.Current()
		if current.ID != created.ID {
			t.Errorf("Expected current to be created session")
		}
	})

	t.Run("set current session", func(t *testing.T) {
		store := NewSessionStore()
		first := store.Create("First")
		second := store.Create("Second")

		// Second should be current now
		if store.Current().ID != second.ID {
			t.Error("Expected second to be current after creation")
		}

		// Set first as current
		if !store.SetCurrent(first.ID) {
			t.Error("Expected SetCurrent to succeed")
		}
		if store.Current().ID != first.ID {
			t.Error("Expected first to be current after SetCurrent")
		}
	})

	t.Run("set current nonexistent fails", func(t *testing.T) {
		store := NewSessionStore()

		if store.SetCurrent("nonexistent") {
			t.Error("Expected SetCurrent to fail for nonexistent session")
		}
	})

	t.Run("list sessions", func(t *testing.T) {
		store := NewSessionStore()
		store.Create("First")
		store.Create("Second")
		store.Create("Third")

		sessions := store.List()
		if len(sessions) != 3 {
			t.Errorf("Expected 3 sessions, got %d", len(sessions))
		}
	})

	t.Run("delete session", func(t *testing.T) {
		store := NewSessionStore()
		session := store.Create("To Delete")

		if !store.Delete(session.ID) {
			t.Error("Expected Delete to succeed")
		}

		_, ok := store.Get(session.ID)
		if ok {
			t.Error("Expected session to be deleted")
		}
	})

	t.Run("delete current session clears current", func(t *testing.T) {
		store := NewSessionStore()
		session := store.Create("Current")

		store.Delete(session.ID)

		// Getting current should create a new session
		newCurrent := store.Current()
		if newCurrent.ID == session.ID {
			t.Error("Expected new current session after deleting old current")
		}
	})

	t.Run("delete nonexistent fails", func(t *testing.T) {
		store := NewSessionStore()

		if store.Delete("nonexistent") {
			t.Error("Expected Delete to fail for nonexistent session")
		}
	})
}

//nolint:gocyclo // Test functions naturally have high complexity
func TestSessionStoreMessages(t *testing.T) {
	t.Run("add message", func(t *testing.T) {
		store := NewSessionStore()
		session := store.Create("Test")

		msg := Message{
			Role:    RoleUser,
			Content: "Hello",
		}

		if !store.AddMessage(session.ID, msg) {
			t.Error("Expected AddMessage to succeed")
		}

		messages := store.GetMessages(session.ID)
		if len(messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(messages))
		}
		if messages[0].Content != "Hello" {
			t.Errorf("Expected content 'Hello', got %q", messages[0].Content)
		}
		if messages[0].ID == "" {
			t.Error("Expected message ID to be auto-generated")
		}
	})

	t.Run("add message to nonexistent session fails", func(t *testing.T) {
		store := NewSessionStore()

		msg := Message{Role: RoleUser, Content: "Hello"}
		if store.AddMessage("nonexistent", msg) {
			t.Error("Expected AddMessage to fail for nonexistent session")
		}
	})

	t.Run("get messages returns copy", func(t *testing.T) {
		store := NewSessionStore()
		session := store.Create("Test")
		store.AddMessage(session.ID, Message{Role: RoleUser, Content: "Hello"})

		messages := store.GetMessages(session.ID)
		messages[0].Content = "Modified"

		// Original should be unchanged
		originalMessages := store.GetMessages(session.ID)
		if originalMessages[0].Content == "Modified" {
			t.Error("Expected GetMessages to return a copy")
		}
	})

	t.Run("get messages from nonexistent session returns nil", func(t *testing.T) {
		store := NewSessionStore()

		messages := store.GetMessages("nonexistent")
		if messages != nil {
			t.Error("Expected nil for nonexistent session")
		}
	})

	t.Run("clear messages", func(t *testing.T) {
		store := NewSessionStore()
		session := store.Create("Test")
		store.AddMessage(session.ID, Message{Role: RoleUser, Content: "Hello"})

		if !store.ClearMessages(session.ID) {
			t.Error("Expected ClearMessages to succeed")
		}

		messages := store.GetMessages(session.ID)
		if len(messages) != 0 {
			t.Errorf("Expected 0 messages after clear, got %d", len(messages))
		}
	})

	t.Run("clear messages nonexistent fails", func(t *testing.T) {
		store := NewSessionStore()

		if store.ClearMessages("nonexistent") {
			t.Error("Expected ClearMessages to fail for nonexistent session")
		}
	})

	t.Run("update title", func(t *testing.T) {
		store := NewSessionStore()
		session := store.Create("Original")

		if !store.UpdateTitle(session.ID, "Updated") {
			t.Error("Expected UpdateTitle to succeed")
		}

		retrieved, _ := store.Get(session.ID)
		if retrieved.Title != "Updated" {
			t.Errorf("Expected title 'Updated', got %q", retrieved.Title)
		}
	})

	t.Run("update title nonexistent fails", func(t *testing.T) {
		store := NewSessionStore()

		if store.UpdateTitle("nonexistent", "New Title") {
			t.Error("Expected UpdateTitle to fail for nonexistent session")
		}
	})

	t.Run("message timestamps", func(t *testing.T) {
		store := NewSessionStore()
		session := store.Create("Test")

		before := time.Now()
		store.AddMessage(session.ID, Message{Role: RoleUser, Content: "Hello"})
		after := time.Now()

		messages := store.GetMessages(session.ID)
		if messages[0].CreatedAt.Before(before) || messages[0].CreatedAt.After(after) {
			t.Error("Expected message CreatedAt to be within test bounds")
		}
	})
}

func TestSessionStoreConcurrency(t *testing.T) {
	store := NewSessionStore()
	session := store.Create("Concurrent Test")

	// Run concurrent operations
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			store.AddMessage(session.ID, Message{
				Role:    RoleUser,
				Content: "Message",
			})
			store.GetMessages(session.ID)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	messages := store.GetMessages(session.ID)
	if len(messages) != 10 {
		t.Errorf("Expected 10 messages, got %d", len(messages))
	}
}
