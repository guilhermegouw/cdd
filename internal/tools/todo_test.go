//nolint:goconst,errcheck // Test files use literal strings for clarity and intentionally ignore some errors.
package tools

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"charm.land/fantasy"

	"github.com/guilhermegouw/cdd/internal/pubsub"
)

func TestTodoStatus_IsValid(t *testing.T) {
	tests := []struct {
		status TodoStatus
		want   bool
	}{
		{TodoStatusPending, true},
		{TodoStatusInProgress, true},
		{TodoStatusCompleted, true},
		{TodoStatus("invalid"), false},
		{TodoStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTodoStore_GetSet(t *testing.T) {
	store := NewTodoStore()
	sessionID := "test-session"

	// Initially empty
	if got := store.Get(sessionID); got != nil {
		t.Errorf("Get() on empty store = %v, want nil", got)
	}

	// Set some todos
	todos := []TodoItem{
		{Content: "Task 1", ActiveForm: "Doing task 1", Status: TodoStatusPending},
		{Content: "Task 2", ActiveForm: "Doing task 2", Status: TodoStatusInProgress},
	}
	store.Set(sessionID, todos)

	// Get should return the todos
	got := store.Get(sessionID)
	if len(got) != 2 {
		t.Fatalf("Get() returned %d items, want 2", len(got))
	}
	if got[0].Content != "Task 1" {
		t.Errorf("Get()[0].Content = %q, want %q", got[0].Content, "Task 1")
	}
	if got[1].Status != TodoStatusInProgress {
		t.Errorf("Get()[1].Status = %q, want %q", got[1].Status, TodoStatusInProgress)
	}
}

func TestTodoStore_GetReturnsCopy(t *testing.T) {
	store := NewTodoStore()
	sessionID := "test-session"

	todos := []TodoItem{
		{Content: "Task 1", ActiveForm: "Doing task 1", Status: TodoStatusPending},
	}
	store.Set(sessionID, todos)

	// Modify the returned slice
	got := store.Get(sessionID)
	got[0].Content = "Modified"

	// Original should be unchanged
	got2 := store.Get(sessionID)
	if got2[0].Content != "Task 1" {
		t.Errorf("Store was modified through Get() return value")
	}
}

func TestTodoStore_SetStoresCopy(t *testing.T) {
	store := NewTodoStore()
	sessionID := "test-session"

	todos := []TodoItem{
		{Content: "Task 1", ActiveForm: "Doing task 1", Status: TodoStatusPending},
	}
	store.Set(sessionID, todos)

	// Modify the original slice
	todos[0].Content = "Modified"

	// Stored value should be unchanged
	got := store.Get(sessionID)
	if got[0].Content != "Task 1" {
		t.Errorf("Store was modified through original slice")
	}
}

func TestTodoStore_Clear(t *testing.T) {
	store := NewTodoStore()
	sessionID := "test-session"

	todos := []TodoItem{
		{Content: "Task 1", ActiveForm: "Doing task 1", Status: TodoStatusPending},
	}
	store.Set(sessionID, todos)

	// Clear the session
	store.Clear(sessionID)

	// Should be empty now
	if got := store.Get(sessionID); got != nil {
		t.Errorf("Get() after Clear() = %v, want nil", got)
	}
}

func TestTodoStore_ClearAll(t *testing.T) {
	store := NewTodoStore()

	// Set todos for multiple sessions
	store.Set("session1", []TodoItem{{Content: "Task 1", ActiveForm: "Task 1", Status: TodoStatusPending}})
	store.Set("session2", []TodoItem{{Content: "Task 2", ActiveForm: "Task 2", Status: TodoStatusPending}})

	// Clear all
	store.ClearAll()

	// Both should be empty
	if store.HasTodos("session1") || store.HasTodos("session2") {
		t.Errorf("Sessions still have todos after ClearAll()")
	}
}

func TestTodoStore_SetEmptyClears(t *testing.T) {
	store := NewTodoStore()
	sessionID := "test-session"

	todos := []TodoItem{
		{Content: "Task 1", ActiveForm: "Doing task 1", Status: TodoStatusPending},
	}
	store.Set(sessionID, todos)

	// Set empty slice should clear
	store.Set(sessionID, []TodoItem{})

	if store.HasTodos(sessionID) {
		t.Errorf("HasTodos() after Set([]) = true, want false")
	}

	// Set nil should also clear
	store.Set(sessionID, todos)
	store.Set(sessionID, nil)

	if store.HasTodos(sessionID) {
		t.Errorf("HasTodos() after Set(nil) = true, want false")
	}
}

func TestTodoStore_HasTodos(t *testing.T) {
	store := NewTodoStore()
	sessionID := "test-session"

	if store.HasTodos(sessionID) {
		t.Errorf("HasTodos() on empty store = true, want false")
	}

	store.Set(sessionID, []TodoItem{{Content: "Task", ActiveForm: "Task", Status: TodoStatusPending}})

	if !store.HasTodos(sessionID) {
		t.Errorf("HasTodos() after Set() = false, want true")
	}
}

func TestTodoStore_Count(t *testing.T) {
	store := NewTodoStore()
	sessionID := "test-session"

	if got := store.Count(sessionID); got != 0 {
		t.Errorf("Count() on empty store = %d, want 0", got)
	}

	todos := []TodoItem{
		{Content: "Task 1", ActiveForm: "Task 1", Status: TodoStatusPending},
		{Content: "Task 2", ActiveForm: "Task 2", Status: TodoStatusCompleted},
		{Content: "Task 3", ActiveForm: "Task 3", Status: TodoStatusPending},
	}
	store.Set(sessionID, todos)

	if got := store.Count(sessionID); got != 3 {
		t.Errorf("Count() = %d, want 3", got)
	}
}

func TestTodoStore_CountByStatus(t *testing.T) {
	store := NewTodoStore()
	sessionID := "test-session"

	todos := []TodoItem{
		{Content: "Task 1", ActiveForm: "Task 1", Status: TodoStatusPending},
		{Content: "Task 2", ActiveForm: "Task 2", Status: TodoStatusCompleted},
		{Content: "Task 3", ActiveForm: "Task 3", Status: TodoStatusPending},
		{Content: "Task 4", ActiveForm: "Task 4", Status: TodoStatusInProgress},
	}
	store.Set(sessionID, todos)

	tests := []struct {
		status TodoStatus
		want   int
	}{
		{TodoStatusPending, 2},
		{TodoStatusCompleted, 1},
		{TodoStatusInProgress, 1},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := store.CountByStatus(sessionID, tt.status); got != tt.want {
				t.Errorf("CountByStatus(%s) = %d, want %d", tt.status, got, tt.want)
			}
		})
	}
}

func TestTodoStore_GetInProgress(t *testing.T) {
	store := NewTodoStore()
	sessionID := "test-session"

	// No todos
	if got := store.GetInProgress(sessionID); got != nil {
		t.Errorf("GetInProgress() on empty store = %v, want nil", got)
	}

	// No in-progress
	store.Set(sessionID, []TodoItem{
		{Content: "Task 1", ActiveForm: "Task 1", Status: TodoStatusPending},
		{Content: "Task 2", ActiveForm: "Task 2", Status: TodoStatusCompleted},
	})

	if got := store.GetInProgress(sessionID); got != nil {
		t.Errorf("GetInProgress() with no in-progress = %v, want nil", got)
	}

	// With in-progress
	store.Set(sessionID, []TodoItem{
		{Content: "Task 1", ActiveForm: "Doing task 1", Status: TodoStatusInProgress},
		{Content: "Task 2", ActiveForm: "Task 2", Status: TodoStatusPending},
	})

	got := store.GetInProgress(sessionID)
	if got == nil {
		t.Fatal("GetInProgress() = nil, want non-nil")
	}
	if got.Content != "Task 1" {
		t.Errorf("GetInProgress().Content = %q, want %q", got.Content, "Task 1")
	}
}

func TestTodoStore_GetInProgressReturnsCopy(t *testing.T) {
	store := NewTodoStore()
	sessionID := "test-session"

	store.Set(sessionID, []TodoItem{
		{Content: "Task 1", ActiveForm: "Doing task 1", Status: TodoStatusInProgress},
	})

	got := store.GetInProgress(sessionID)
	got.Content = "Modified"

	// Original should be unchanged
	got2 := store.GetInProgress(sessionID)
	if got2.Content != "Task 1" {
		t.Errorf("Store was modified through GetInProgress() return value")
	}
}

func TestTodoStore_ConcurrentAccess(t *testing.T) {
	store := NewTodoStore()
	sessionID := "test-session"

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writes
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.Set(sessionID, []TodoItem{
				{Content: "Task", ActiveForm: "Task", Status: TodoStatusPending},
			})
		}()
	}

	// Concurrent reads
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = store.Get(sessionID)
			_ = store.HasTodos(sessionID)
			_ = store.Count(sessionID)
			_ = store.GetInProgress(sessionID)
		}()
	}

	wg.Wait()

	// Should not panic or have data races
	// (run with -race to verify)
}

func TestTodoStore_MultipleSessions(t *testing.T) {
	store := NewTodoStore()

	store.Set("session1", []TodoItem{
		{Content: "Session 1 Task", ActiveForm: "Task", Status: TodoStatusPending},
	})
	store.Set("session2", []TodoItem{
		{Content: "Session 2 Task", ActiveForm: "Task", Status: TodoStatusInProgress},
	})

	// Each session should have its own data
	got1 := store.Get("session1")
	got2 := store.Get("session2")

	if got1[0].Content != "Session 1 Task" {
		t.Errorf("Session 1 has wrong content")
	}
	if got2[0].Content != "Session 2 Task" {
		t.Errorf("Session 2 has wrong content")
	}

	// Clearing one session shouldn't affect the other
	store.Clear("session1")

	if store.HasTodos("session1") {
		t.Errorf("Session 1 should be empty after Clear()")
	}
	if !store.HasTodos("session2") {
		t.Errorf("Session 2 should still have todos")
	}
}

// Tests for ValidateTodos

func TestValidateTodos_Valid(t *testing.T) {
	tests := []struct {
		name  string
		todos []TodoItem
	}{
		{
			name:  "empty list",
			todos: []TodoItem{},
		},
		{
			name: "single pending",
			todos: []TodoItem{
				{Content: "Task 1", ActiveForm: "Doing task 1", Status: TodoStatusPending},
			},
		},
		{
			name: "single in_progress",
			todos: []TodoItem{
				{Content: "Task 1", ActiveForm: "Doing task 1", Status: TodoStatusInProgress},
			},
		},
		{
			name: "multiple with one in_progress",
			todos: []TodoItem{
				{Content: "Task 1", ActiveForm: "Doing task 1", Status: TodoStatusCompleted},
				{Content: "Task 2", ActiveForm: "Doing task 2", Status: TodoStatusInProgress},
				{Content: "Task 3", ActiveForm: "Doing task 3", Status: TodoStatusPending},
			},
		},
		{
			name: "all completed",
			todos: []TodoItem{
				{Content: "Task 1", ActiveForm: "Doing task 1", Status: TodoStatusCompleted},
				{Content: "Task 2", ActiveForm: "Doing task 2", Status: TodoStatusCompleted},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTodos(tt.todos)
			if err != nil {
				t.Errorf("ValidateTodos() error = %v, want nil", err)
			}
		})
	}
}

func TestValidateTodos_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		todos   []TodoItem
		wantErr string
	}{
		{
			name: "empty content",
			todos: []TodoItem{
				{Content: "", ActiveForm: "Doing task", Status: TodoStatusPending},
			},
			wantErr: "content cannot be empty",
		},
		{
			name: "empty activeForm",
			todos: []TodoItem{
				{Content: "Task", ActiveForm: "", Status: TodoStatusPending},
			},
			wantErr: "activeForm cannot be empty",
		},
		{
			name: "invalid status",
			todos: []TodoItem{
				{Content: "Task", ActiveForm: "Doing task", Status: TodoStatus("invalid")},
			},
			wantErr: "invalid status",
		},
		{
			name: "multiple in_progress",
			todos: []TodoItem{
				{Content: "Task 1", ActiveForm: "Doing task 1", Status: TodoStatusInProgress},
				{Content: "Task 2", ActiveForm: "Doing task 2", Status: TodoStatusInProgress},
			},
			wantErr: "only one todo should be in_progress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTodos(tt.todos)
			if err == nil {
				t.Error("ValidateTodos() error = nil, want error")
				return
			}
			if !contains(err.Error(), tt.wantErr) {
				t.Errorf("ValidateTodos() error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Tests for NewTodoWriteTool

func invokeTodoWriteTool(ctx context.Context, tool fantasy.AgentTool, params TodoWriteParams) (fantasy.ToolResponse, error) {
	inputJSON, err := json.Marshal(params)
	if err != nil {
		return fantasy.ToolResponse{}, err
	}

	call := fantasy.ToolCall{
		ID:    "test-call",
		Name:  TodoWriteToolName,
		Input: string(inputJSON),
	}

	return tool.Run(ctx, call)
}

func TestTodoWriteTool_Success(t *testing.T) {
	store := NewTodoStore()
	hub := pubsub.NewHub()
	defer hub.Shutdown()

	tool := NewTodoWriteTool(store, hub)

	// Create a context with session ID
	ctx := WithSessionID(context.Background(), "test-session")

	params := TodoWriteParams{
		Todos: []TodoItem{
			{Content: "Task 1", ActiveForm: "Doing task 1", Status: TodoStatusPending},
		},
	}

	resp, err := invokeTodoWriteTool(ctx, tool, params)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Check response
	if resp.IsError {
		t.Errorf("Run() returned error response: %v", resp.Content)
	}

	// Check store was updated
	todos := store.Get("test-session")
	if len(todos) != 1 {
		t.Fatalf("Store has %d todos, want 1", len(todos))
	}
	if todos[0].Content != "Task 1" {
		t.Errorf("Todo content = %q, want %q", todos[0].Content, "Task 1")
	}
}

func TestTodoWriteTool_ValidationError(t *testing.T) {
	store := NewTodoStore()
	tool := NewTodoWriteTool(store, nil)

	ctx := WithSessionID(context.Background(), "test-session")

	params := TodoWriteParams{
		Todos: []TodoItem{
			{Content: "", ActiveForm: "Doing task", Status: TodoStatusPending}, // Empty content
		},
	}

	resp, err := invokeTodoWriteTool(ctx, tool, params)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should return error response (not Go error)
	if !resp.IsError {
		t.Error("Run() should return error response for invalid input")
	}

	// Store should not be updated
	if store.HasTodos("test-session") {
		t.Error("Store should not have todos after validation error")
	}
}

func TestTodoWriteTool_MultipleInProgressError(t *testing.T) {
	store := NewTodoStore()
	tool := NewTodoWriteTool(store, nil)

	ctx := WithSessionID(context.Background(), "test-session")

	params := TodoWriteParams{
		Todos: []TodoItem{
			{Content: "Task 1", ActiveForm: "Doing task 1", Status: TodoStatusInProgress},
			{Content: "Task 2", ActiveForm: "Doing task 2", Status: TodoStatusInProgress},
		},
	}

	resp, err := invokeTodoWriteTool(ctx, tool, params)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !resp.IsError {
		t.Error("Run() should return error response for multiple in_progress")
	}
}

func TestTodoWriteTool_UpdateExisting(t *testing.T) {
	store := NewTodoStore()
	tool := NewTodoWriteTool(store, nil)

	ctx := WithSessionID(context.Background(), "test-session")

	// First call - add todos
	params1 := TodoWriteParams{
		Todos: []TodoItem{
			{Content: "Task 1", ActiveForm: "Doing task 1", Status: TodoStatusInProgress},
			{Content: "Task 2", ActiveForm: "Doing task 2", Status: TodoStatusPending},
		},
	}
	_, _ = invokeTodoWriteTool(ctx, tool, params1)

	// Second call - update todos
	params2 := TodoWriteParams{
		Todos: []TodoItem{
			{Content: "Task 1", ActiveForm: "Doing task 1", Status: TodoStatusCompleted},
			{Content: "Task 2", ActiveForm: "Doing task 2", Status: TodoStatusInProgress},
		},
	}
	_, _ = invokeTodoWriteTool(ctx, tool, params2)

	// Check store was updated
	todos := store.Get("test-session")
	if len(todos) != 2 {
		t.Fatalf("Store has %d todos, want 2", len(todos))
	}
	if todos[0].Status != TodoStatusCompleted {
		t.Errorf("Task 1 status = %q, want %q", todos[0].Status, TodoStatusCompleted)
	}
	if todos[1].Status != TodoStatusInProgress {
		t.Errorf("Task 2 status = %q, want %q", todos[1].Status, TodoStatusInProgress)
	}
}

func TestTodoWriteTool_NilHub(t *testing.T) {
	store := NewTodoStore()
	// Pass nil hub - should still work
	tool := NewTodoWriteTool(store, nil)

	ctx := WithSessionID(context.Background(), "test-session")

	params := TodoWriteParams{
		Todos: []TodoItem{
			{Content: "Task 1", ActiveForm: "Doing task 1", Status: TodoStatusPending},
		},
	}

	resp, err := invokeTodoWriteTool(ctx, tool, params)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if resp.IsError {
		t.Errorf("Run() returned error: %v", resp.Content)
	}
}
