package tools

import (
	"sync"
)

// TodoStore manages todo lists per session with thread-safe access.
type TodoStore struct {
	mu    sync.RWMutex
	todos map[string][]TodoItem // sessionID -> todos
}

// NewTodoStore creates a new todo store.
func NewTodoStore() *TodoStore {
	return &TodoStore{
		todos: make(map[string][]TodoItem),
	}
}

// Get returns a copy of the todos for a session.
// Returns nil if no todos exist for the session.
func (s *TodoStore) Get(sessionID string) []TodoItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	todos, ok := s.todos[sessionID]
	if !ok || len(todos) == 0 {
		return nil
	}

	// Return a copy to prevent external modification
	result := make([]TodoItem, len(todos))
	copy(result, todos)
	return result
}

// Set updates the todos for a session.
// Passing nil or empty slice clears the todos for that session.
func (s *TodoStore) Set(sessionID string, todos []TodoItem) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(todos) == 0 {
		delete(s.todos, sessionID)
		return
	}

	// Store a copy to prevent external modification
	stored := make([]TodoItem, len(todos))
	copy(stored, todos)
	s.todos[sessionID] = stored
}

// Clear removes todos for a session.
func (s *TodoStore) Clear(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.todos, sessionID)
}

// ClearAll removes all todos from all sessions.
func (s *TodoStore) ClearAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.todos = make(map[string][]TodoItem)
}

// HasTodos returns true if the session has any todos.
func (s *TodoStore) HasTodos(sessionID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	todos, ok := s.todos[sessionID]
	return ok && len(todos) > 0
}

// Count returns the number of todos for a session.
func (s *TodoStore) Count(sessionID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.todos[sessionID])
}

// CountByStatus returns the count of todos with a specific status.
func (s *TodoStore) CountByStatus(sessionID string, status TodoStatus) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, todo := range s.todos[sessionID] {
		if todo.Status == status {
			count++
		}
	}
	return count
}

// GetInProgress returns the currently in-progress todo, if any.
func (s *TodoStore) GetInProgress(sessionID string) *TodoItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, todo := range s.todos[sessionID] {
		if todo.Status == TodoStatusInProgress {
			// Return a copy
			result := todo
			return &result
		}
	}
	return nil
}
