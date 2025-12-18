package agent

import (
	"context"
	"sync"
	"testing"
	"time"

	"charm.land/fantasy"
)

// mockModel implements fantasy.LanguageModel for testing
type mockModel struct {
	generateFunc func(ctx context.Context, call fantasy.Call) (*fantasy.Response, error)
	streamFunc   func(ctx context.Context, call fantasy.Call) (fantasy.StreamResponse, error)
}

func (m *mockModel) Generate(ctx context.Context, call fantasy.Call) (*fantasy.Response, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, call)
	}
	return &fantasy.Response{}, nil
}

func (m *mockModel) Stream(ctx context.Context, call fantasy.Call) (fantasy.StreamResponse, error) {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, call)
	}
	// Return empty iterator
	return func(yield func(fantasy.StreamPart) bool) {}, nil
}

func (m *mockModel) GenerateObject(ctx context.Context, call fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
	return &fantasy.ObjectResponse{}, nil
}

func (m *mockModel) StreamObject(ctx context.Context, call fantasy.ObjectCall) (fantasy.ObjectStreamResponse, error) {
	return func(yield func(fantasy.ObjectStreamPart) bool) {}, nil
}

func (m *mockModel) Provider() string { return "mock" }
func (m *mockModel) Model() string    { return "mock-model" }

// Ensure mockModel implements the interface
var _ fantasy.LanguageModel = (*mockModel)(nil)

func TestAgentCreation(t *testing.T) {
	t.Run("create agent with config", func(t *testing.T) {
		cfg := Config{
			Model:        &mockModel{},
			SystemPrompt: "Test system prompt",
			WorkingDir:   "/tmp",
		}

		agent := New(cfg)

		if agent == nil {
			t.Fatal("Expected agent to be created")
		}
		if agent.systemPrompt != cfg.SystemPrompt {
			t.Errorf("Expected system prompt %q, got %q", cfg.SystemPrompt, agent.systemPrompt)
		}
		if agent.workingDir != cfg.WorkingDir {
			t.Errorf("Expected working dir %q, got %q", cfg.WorkingDir, agent.workingDir)
		}
		if agent.sessions == nil {
			t.Error("Expected sessions to be initialized")
		}
	})
}

func TestAgentSendErrors(t *testing.T) {
	t.Run("empty prompt returns error", func(t *testing.T) {
		agent := New(Config{
			Model: &mockModel{},
		})

		err := agent.Send(context.Background(), "", SendOptions{}, StreamCallbacks{})
		if err != ErrEmptyPrompt {
			t.Errorf("Expected ErrEmptyPrompt, got %v", err)
		}
	})
}

func TestAgentBusyState(t *testing.T) {
	t.Run("IsBusy returns false initially", func(t *testing.T) {
		agent := New(Config{
			Model: &mockModel{},
		})

		if agent.IsBusy("any-session") {
			t.Error("Expected IsBusy to return false initially")
		}
	})

	t.Run("IsBusy returns true during request", func(t *testing.T) {
		agent := New(Config{
			Model: &mockModel{},
		})

		sessionID := "test-session"

		// Manually set active request to simulate busy state
		agent.mu.Lock()
		ctx, cancel := context.WithCancel(context.Background())
		agent.activeRequests[sessionID] = cancel
		agent.mu.Unlock()

		if !agent.IsBusy(sessionID) {
			t.Error("Expected IsBusy to return true when request is active")
		}

		cancel()
		ctx.Done()
	})
}

func TestAgentCancel(t *testing.T) {
	t.Run("cancel clears active request", func(t *testing.T) {
		agent := New(Config{
			Model: &mockModel{},
		})

		sessionID := "test-session"

		// Set up active request
		agent.mu.Lock()
		_, cancel := context.WithCancel(context.Background())
		agent.activeRequests[sessionID] = cancel
		agent.mu.Unlock()

		// Should be busy
		if !agent.IsBusy(sessionID) {
			t.Error("Expected IsBusy before cancel")
		}

		// Cancel
		agent.Cancel(sessionID)

		// Should not be busy
		if agent.IsBusy(sessionID) {
			t.Error("Expected not busy after cancel")
		}
	})

	t.Run("cancel nonexistent session is safe", func(t *testing.T) {
		agent := New(Config{
			Model: &mockModel{},
		})

		// Should not panic
		agent.Cancel("nonexistent")
	})
}

func TestAgentSetSystemPrompt(t *testing.T) {
	t.Run("set system prompt", func(t *testing.T) {
		agent := New(Config{
			Model:        &mockModel{},
			SystemPrompt: "Original",
		})

		agent.SetSystemPrompt("Updated")

		agent.mu.RLock()
		prompt := agent.systemPrompt
		agent.mu.RUnlock()

		if prompt != "Updated" {
			t.Errorf("Expected 'Updated', got %q", prompt)
		}
	})
}

func TestAgentSetTools(t *testing.T) {
	t.Run("set tools", func(t *testing.T) {
		agent := New(Config{
			Model: &mockModel{},
		})

		tools := []fantasy.AgentTool{}

		agent.SetTools(tools)

		agent.mu.RLock()
		result := agent.tools
		agent.mu.RUnlock()

		if result == nil {
			t.Error("Expected tools to be set")
		}
	})
}

func TestAgentHistory(t *testing.T) {
	t.Run("history returns session messages", func(t *testing.T) {
		agent := New(Config{
			Model: &mockModel{},
		})

		session := agent.Sessions().Create("Test")
		agent.Sessions().AddMessage(session.ID, Message{
			Role:    RoleUser,
			Content: "Hello",
		})

		history := agent.History(session.ID)
		if len(history) != 1 {
			t.Errorf("Expected 1 message, got %d", len(history))
		}
	})

	t.Run("history for nonexistent session returns nil", func(t *testing.T) {
		agent := New(Config{
			Model: &mockModel{},
		})

		history := agent.History("nonexistent")
		if history != nil {
			t.Error("Expected nil for nonexistent session")
		}
	})
}

func TestAgentClear(t *testing.T) {
	t.Run("clear removes messages", func(t *testing.T) {
		agent := New(Config{
			Model: &mockModel{},
		})

		session := agent.Sessions().Create("Test")
		agent.Sessions().AddMessage(session.ID, Message{
			Role:    RoleUser,
			Content: "Hello",
		})

		agent.Clear(session.ID)

		history := agent.History(session.ID)
		if len(history) != 0 {
			t.Errorf("Expected 0 messages after clear, got %d", len(history))
		}
	})
}

func TestAgentSessions(t *testing.T) {
	t.Run("sessions returns store", func(t *testing.T) {
		agent := New(Config{
			Model: &mockModel{},
		})

		store := agent.Sessions()
		if store == nil {
			t.Error("Expected sessions store to be returned")
		}

		// Verify it's the same store
		session := store.Create("Test")
		if _, ok := agent.Sessions().Get(session.ID); !ok {
			t.Error("Expected same store instance")
		}
	})
}

func TestAgentConcurrency(t *testing.T) {
	agent := New(Config{
		Model: &mockModel{},
	})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			// Concurrent operations
			agent.SetSystemPrompt("Prompt")
			agent.SetTools(nil)
			agent.IsBusy("session")
			agent.Cancel("session")
		}(i)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Error("Concurrent operations timed out")
	}
}

func TestBuildHistory(t *testing.T) {
	t.Run("empty history", func(t *testing.T) {
		agent := New(Config{
			Model: &mockModel{},
		})

		session := agent.Sessions().Create("Test")
		history := agent.buildHistory(session.ID)

		if history != nil {
			t.Errorf("Expected nil for empty history, got %v", history)
		}
	})

	t.Run("builds user messages", func(t *testing.T) {
		agent := New(Config{
			Model: &mockModel{},
		})

		session := agent.Sessions().Create("Test")
		agent.Sessions().AddMessage(session.ID, Message{
			Role:    RoleUser,
			Content: "First message",
		})
		agent.Sessions().AddMessage(session.ID, Message{
			Role:    RoleAssistant,
			Content: "Response",
		})
		// Add current message (should be excluded from history)
		agent.Sessions().AddMessage(session.ID, Message{
			Role:    RoleUser,
			Content: "Current",
		})

		history := agent.buildHistory(session.ID)

		// Should have 2 messages (excluding current user input)
		if len(history) != 2 {
			t.Errorf("Expected 2 messages in history, got %d", len(history))
		}
	})

	t.Run("builds assistant messages with tool calls", func(t *testing.T) {
		agent := New(Config{
			Model: &mockModel{},
		})

		session := agent.Sessions().Create("Test")
		agent.Sessions().AddMessage(session.ID, Message{
			Role:    RoleAssistant,
			Content: "Let me help",
			ToolCalls: []ToolCall{
				{ID: "tc1", Name: "read", Input: "{}"},
			},
		})
		// Add another message so the first isn't excluded
		agent.Sessions().AddMessage(session.ID, Message{
			Role:    RoleUser,
			Content: "Current",
		})

		history := agent.buildHistory(session.ID)

		if len(history) != 1 {
			t.Errorf("Expected 1 message, got %d", len(history))
		}
	})

	t.Run("builds tool result messages", func(t *testing.T) {
		agent := New(Config{
			Model: &mockModel{},
		})

		session := agent.Sessions().Create("Test")
		agent.Sessions().AddMessage(session.ID, Message{
			Role: RoleTool,
			ToolResults: []ToolResult{
				{ToolCallID: "tc1", Name: "read", Content: "file content"},
			},
		})
		agent.Sessions().AddMessage(session.ID, Message{
			Role:    RoleUser,
			Content: "Current",
		})

		history := agent.buildHistory(session.ID)

		if len(history) != 1 {
			t.Errorf("Expected 1 message, got %d", len(history))
		}
	})

	t.Run("builds tool error results", func(t *testing.T) {
		agent := New(Config{
			Model: &mockModel{},
		})

		session := agent.Sessions().Create("Test")
		agent.Sessions().AddMessage(session.ID, Message{
			Role: RoleTool,
			ToolResults: []ToolResult{
				{ToolCallID: "tc1", Name: "read", Content: "file not found", IsError: true},
			},
		})
		agent.Sessions().AddMessage(session.ID, Message{
			Role:    RoleUser,
			Content: "Current",
		})

		history := agent.buildHistory(session.ID)

		if len(history) != 1 {
			t.Errorf("Expected 1 message, got %d", len(history))
		}
	})
}
