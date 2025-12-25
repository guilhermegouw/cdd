package events

import (
	"time"
)

// TodoStatus represents the status of a todo item.
type TodoStatus string

// Todo status constants.
const (
	TodoStatusPending    TodoStatus = "pending"
	TodoStatusInProgress TodoStatus = "in_progress"
	TodoStatusCompleted  TodoStatus = "completed"
)

// TodoItem represents a single todo item in an event.
type TodoItem struct {
	Content    string     `json:"content"`
	ActiveForm string     `json:"activeForm"`
	Status     TodoStatus `json:"status"`
}

// TodoEvent represents a todo list update event.
type TodoEvent struct {
	SessionID string
	Todos     []TodoItem
	Timestamp time.Time
}

// NewTodoUpdatedEvent creates a todo updated event.
func NewTodoUpdatedEvent(sessionID string, todos []TodoItem) TodoEvent {
	return TodoEvent{
		SessionID: sessionID,
		Todos:     todos,
		Timestamp: time.Now(),
	}
}
