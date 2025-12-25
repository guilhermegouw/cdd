package tools

import (
	"context"
	"errors"
	"fmt"

	"charm.land/fantasy"

	"github.com/guilhermegouw/cdd/internal/events"
	"github.com/guilhermegouw/cdd/internal/pubsub"
)

// TodoStatus represents the status of a todo item.
type TodoStatus string

// Todo status constants.
const (
	TodoStatusPending    TodoStatus = "pending"
	TodoStatusInProgress TodoStatus = "in_progress"
	TodoStatusCompleted  TodoStatus = "completed"
)

// Tool constants for todo operations.
const (
	TodoWriteToolName = "todo_write"
)

// TodoItem represents a single todo item.
type TodoItem struct {
	// Content is the imperative form of the task (e.g., "Run the build").
	Content string `json:"content"`

	// ActiveForm is the present continuous form shown during execution
	// (e.g., "Running the build").
	ActiveForm string `json:"activeForm"`

	// Status is the current status of the todo item.
	Status TodoStatus `json:"status"`
}

// TodoWriteParams are the parameters for the TodoWrite tool.
type TodoWriteParams struct {
	Todos []TodoItem `json:"todos" description:"The updated todo list"`
}

// IsValid checks if a TodoStatus is valid.
func (s TodoStatus) IsValid() bool {
	switch s {
	case TodoStatusPending, TodoStatusInProgress, TodoStatusCompleted:
		return true
	default:
		return false
	}
}

const todoWriteDescription = `Manages a task list for tracking progress on complex tasks.

Usage:
- Create a todo list when starting a multi-step task
- Update item status as you work (pending → in_progress → completed)
- Only one item should be "in_progress" at a time
- Mark items complete immediately when finished

Each todo item requires:
- content: Imperative form ("Run the build", "Fix type errors")
- activeForm: Present continuous shown during execution ("Running the build")
- status: pending, in_progress, or completed`

// ValidateTodos validates a list of todo items.
func ValidateTodos(todos []TodoItem) error {
	inProgressCount := 0

	for i, t := range todos {
		if t.Content == "" {
			return fmt.Errorf("todo[%d]: content cannot be empty", i)
		}
		if t.ActiveForm == "" {
			return fmt.Errorf("todo[%d]: activeForm cannot be empty", i)
		}
		if !t.Status.IsValid() {
			return fmt.Errorf("todo[%d]: invalid status %q", i, t.Status)
		}
		if t.Status == TodoStatusInProgress {
			inProgressCount++
		}
	}

	if inProgressCount > 1 {
		return errors.New("only one todo should be in_progress at a time")
	}

	return nil
}

// NewTodoWriteTool creates a new TodoWrite tool.
func NewTodoWriteTool(store *TodoStore, hub *pubsub.Hub) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		TodoWriteToolName,
		todoWriteDescription,
		func(ctx context.Context, params TodoWriteParams, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			sessionID := SessionIDFromContext(ctx)

			// Validate todos
			if err := ValidateTodos(params.Todos); err != nil {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}

			// Update store
			store.Set(sessionID, params.Todos)

			// Publish event
			if hub != nil && hub.Todo != nil {
				hub.Todo.Publish(pubsub.EventUpdated, events.NewTodoUpdatedEvent(
					sessionID,
					toEventTodos(params.Todos),
				))
			}

			return fantasy.NewTextResponse("Todos updated successfully. Continue with the current task."), nil
		})
}

// toEventTodos converts tools.TodoItem slice to events.TodoItem slice.
func toEventTodos(todos []TodoItem) []events.TodoItem {
	result := make([]events.TodoItem, len(todos))
	for i, t := range todos {
		result[i] = events.TodoItem{
			Content:    t.Content,
			ActiveForm: t.ActiveForm,
			Status:     events.TodoStatus(t.Status),
		}
	}
	return result
}
