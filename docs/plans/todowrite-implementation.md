# TodoWrite Implementation Plan

## Overview

TodoWrite is a tool that allows the AI agent to create, update, and track a visible task list in the terminal. This provides users with real-time visibility into the agent's plan and progress.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                           Agent                                  │
│                              │                                   │
│                    calls TodoWrite tool                          │
│                              ▼                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    TodoWrite Tool                         │   │
│  │  - Validates input                                        │   │
│  │  - Updates TodoStore                                      │   │
│  │  - Publishes event via Hub                                │   │
│  └──────────────────────────────────────────────────────────┘   │
│                              │                                   │
│               publishes TodoUpdatedEvent                         │
│                              ▼                                   │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                           TUI                                    │
│                              │                                   │
│              subscribes to TodoUpdatedEvent                      │
│                              ▼                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                     TodoPanel                             │   │
│  │  ✓ Update tests                                           │   │
│  │  ◐ Fixing type errors                                     │   │
│  │  ○ Run the build                                          │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Components

### 1. Data Types (`internal/tools/todo.go`)

```go
// TodoStatus represents the status of a todo item.
type TodoStatus string

const (
    TodoStatusPending    TodoStatus = "pending"
    TodoStatusInProgress TodoStatus = "in_progress"
    TodoStatusCompleted  TodoStatus = "completed"
)

// TodoItem represents a single todo item.
type TodoItem struct {
    Content    string     `json:"content"`    // Imperative form: "Run the build"
    ActiveForm string     `json:"activeForm"` // Present continuous: "Running the build"
    Status     TodoStatus `json:"status"`
}

// TodoWriteParams are the parameters for the TodoWrite tool.
type TodoWriteParams struct {
    Todos []TodoItem `json:"todos" description:"The updated todo list"`
}
```

### 2. Todo Store (`internal/tools/todo_store.go`)

Thread-safe storage for the current todo list, scoped by session.

```go
// TodoStore manages todo lists per session.
type TodoStore struct {
    mu    sync.RWMutex
    todos map[string][]TodoItem // sessionID -> todos
}

// NewTodoStore creates a new todo store.
func NewTodoStore() *TodoStore

// Get returns the todos for a session.
func (s *TodoStore) Get(sessionID string) []TodoItem

// Set updates the todos for a session.
func (s *TodoStore) Set(sessionID string, todos []TodoItem)

// Clear removes todos for a session.
func (s *TodoStore) Clear(sessionID string)
```

### 3. TodoWrite Tool (`internal/tools/todo.go`)

```go
const TodoWriteToolName = "todo_write"

const todoWriteDescription = `Manages a task list for tracking progress on complex tasks.

Usage:
- Create a todo list when starting a multi-step task
- Update item status as you work (pending → in_progress → completed)
- Only one item should be "in_progress" at a time
- Mark items complete immediately when finished

Each todo item has:
- content: Imperative form ("Run the build", "Fix type errors")
- activeForm: Present continuous shown during execution ("Running the build")
- status: pending, in_progress, or completed`

func NewTodoWriteTool(store *TodoStore, hub *pubsub.Hub) fantasy.AgentTool {
    return fantasy.NewAgentTool(
        TodoWriteToolName,
        todoWriteDescription,
        func(ctx context.Context, params TodoWriteParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
            sessionID := SessionIDFromContext(ctx)

            // Validate todos
            if err := validateTodos(params.Todos); err != nil {
                return fantasy.NewTextErrorResponse(err.Error()), nil
            }

            // Update store
            store.Set(sessionID, params.Todos)

            // Publish event
            if hub != nil {
                hub.PublishTodo(pubsub.EventUpdated, TodoEvent{
                    SessionID: sessionID,
                    Todos:     params.Todos,
                })
            }

            return fantasy.NewTextResponse("Todos updated successfully"), nil
        })
}

func validateTodos(todos []TodoItem) error {
    inProgressCount := 0
    for _, t := range todos {
        if t.Content == "" {
            return errors.New("todo content cannot be empty")
        }
        if t.ActiveForm == "" {
            return errors.New("todo activeForm cannot be empty")
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
```

### 4. PubSub Event (`internal/pubsub/hub.go`)

Add todo event support to the Hub:

```go
// TodoEvent represents a todo list update.
type TodoEvent struct {
    SessionID string
    Todos     []tools.TodoItem
}

// Add to Hub struct:
type Hub struct {
    // ... existing fields
    todoBroker *Broker[TodoEvent]
}

// PublishTodo publishes a todo event.
func (h *Hub) PublishTodo(eventType EventType, event TodoEvent)

// SubscribeTodo subscribes to todo events.
func (h *Hub) SubscribeTodo(ctx context.Context) <-chan Event[TodoEvent]
```

### 5. TUI Todo Panel (`internal/tui/page/chat/todo_panel.go`)

```go
// TodoPanel displays the current todo list with visual status indicators.
type TodoPanel struct {
    todos    []tools.TodoItem
    spinner  int
    width    int
}

func NewTodoPanel() *TodoPanel

// SetTodos updates the displayed todos.
func (p *TodoPanel) SetTodos(todos []tools.TodoItem)

// Height returns the panel height (0 when empty).
func (p *TodoPanel) Height() int

// View renders the todo list.
func (p *TodoPanel) View() string
```

Visual rendering:
```
┌─ Tasks ──────────────────────────────────────────┐
│  ✓ Update tests                                  │  (completed - green)
│  ◐ Fixing type errors                            │  (in_progress - spinner)
│  ○ Run the build                                 │  (pending - dim)
└──────────────────────────────────────────────────┘
```

Status indicators:
- `✓` - completed (green checkmark)
- `◐◑◒◓` - in_progress (animated spinner, uses activeForm text)
- `○` - pending (dim circle, uses content text)

### 6. Chat Integration (`internal/tui/page/chat/chat.go`)

Add TodoPanel to the chat model and wire up events:

```go
type Model struct {
    // ... existing fields
    todoPanel *TodoPanel
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case TodoUpdatedMsg:
        m.todoPanel.SetTodos(msg.Todos)
        return m, nil
    // ... existing cases
    }
}

func (m *Model) View() string {
    // Include todo panel in layout
    // Position: above the activity panel, below messages
}
```

---

## File Changes Summary

| File | Action | Description |
|------|--------|-------------|
| `internal/tools/todo.go` | Create | TodoItem types, TodoWrite tool |
| `internal/tools/todo_store.go` | Create | Thread-safe todo storage |
| `internal/tools/todo_test.go` | Create | Unit tests |
| `internal/tools/registry.go` | Modify | Register TodoWrite tool |
| `internal/pubsub/hub.go` | Modify | Add todo event broker |
| `internal/tui/page/chat/todo_panel.go` | Create | Todo list UI component |
| `internal/tui/page/chat/todo_panel_test.go` | Create | UI component tests |
| `internal/tui/page/chat/chat.go` | Modify | Integrate TodoPanel |

---

## Implementation Phases

### Phase 1: Core Types and Store
1. Create `internal/tools/todo.go` with types
2. Create `internal/tools/todo_store.go` with thread-safe storage
3. Write tests for store operations

### Phase 2: Tool Implementation
1. Implement `NewTodoWriteTool` in `todo.go`
2. Add validation logic
3. Register in `registry.go`
4. Write tool tests

### Phase 3: Event System
1. Add `TodoEvent` to pubsub
2. Add broker to Hub
3. Wire up tool to publish events

### Phase 4: TUI Component
1. Create `todo_panel.go` with rendering logic
2. Implement spinner animation (reuse from ActivityPanel)
3. Handle different status styles
4. Write component tests

### Phase 5: Integration
1. Add TodoPanel to chat Model
2. Subscribe to todo events
3. Update layout calculations
4. Handle panel visibility (hide when empty)

### Phase 6: System Prompt Update
1. Add TodoWrite guidance to system prompt
2. Include examples of when to use it
3. Add reminder mechanism (optional)

---

## Visual Behavior

### Positioning
```
┌──────────────────────────────────────────────────┐
│  Messages area                                   │
│  ...                                             │
│  ...                                             │
├──────────────────────────────────────────────────┤
│  ┌─ Tasks ────────────────────────────────────┐  │  ← TodoPanel
│  │  ✓ Phase 1 complete                        │  │
│  │  ◐ Implementing Phase 2                    │  │
│  │  ○ Phase 3 pending                         │  │
│  └────────────────────────────────────────────┘  │
├──────────────────────────────────────────────────┤
│  ⠋ Thinking...                                   │  ← ActivityPanel
│     └─ read: chat.go                             │
├──────────────────────────────────────────────────┤
│  > user input                                    │  ← Input
├──────────────────────────────────────────────────┤
│  model: claude-3.5  [Enter] send  [Esc] cancel   │  ← StatusBar
└──────────────────────────────────────────────────┘
```

### Animations
- Spinner cycles through: `◐ ◓ ◑ ◒` (or braille: `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`)
- Same tick rate as ActivityPanel (100ms)
- Share ticker to avoid multiple timers

### Visibility
- Panel is hidden when todo list is empty
- Panel appears when first todo is added
- Panel remains visible until explicitly cleared or session ends

---

## Example Tool Call

Agent sends:
```json
{
  "name": "todo_write",
  "input": {
    "todos": [
      {"content": "Read existing code", "activeForm": "Reading existing code", "status": "completed"},
      {"content": "Implement new feature", "activeForm": "Implementing new feature", "status": "in_progress"},
      {"content": "Write tests", "activeForm": "Writing tests", "status": "pending"},
      {"content": "Update documentation", "activeForm": "Updating documentation", "status": "pending"}
    ]
  }
}
```

TUI renders:
```
┌─ Tasks ──────────────────────────────────────────┐
│  ✓ Read existing code                            │
│  ◐ Implementing new feature                      │
│  ○ Write tests                                   │
│  ○ Update documentation                          │
└──────────────────────────────────────────────────┘
```

---

## Testing Strategy

1. **Unit tests**: Store operations, validation, rendering
2. **Integration tests**: Tool → Store → Event → Panel flow
3. **Visual tests**: Manually verify spinner, colors, layout

---

## Future Enhancements

- Collapsible panel (toggle with keybind)
- Nested/hierarchical todos
- Time tracking per task
- Persist todos across sessions
- Progress percentage indicator
