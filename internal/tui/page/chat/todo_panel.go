package chat

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/guilhermegouw/cdd/internal/tools"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
)

// Todo status icons.
const (
	todoIconCompleted  = "✓"
	todoIconPending    = "○"
	todoIconInProgress = "◐" // Will be animated with spinner
)

// TodoPanel displays a task list with status indicators.
type TodoPanel struct {
	todos   []tools.TodoItem
	spinner int // Current spinner frame index (shared with activity panel)
	width   int
}

// NewTodoPanel creates a new todo panel.
func NewTodoPanel() *TodoPanel {
	return &TodoPanel{}
}

// SetTodos updates the displayed todos.
func (p *TodoPanel) SetTodos(todos []tools.TodoItem) {
	p.todos = todos
}

// Clear removes all todos.
func (p *TodoPanel) Clear() {
	p.todos = nil
	p.spinner = 0
}

// SetWidth sets the panel width.
func (p *TodoPanel) SetWidth(width int) {
	p.width = width
}

// SetSpinner updates the spinner frame (called from activity panel tick).
func (p *TodoPanel) SetSpinner(frame int) {
	p.spinner = frame
}

// Height returns the current height of the panel (0 when empty).
func (p *TodoPanel) Height() int {
	if len(p.todos) == 0 {
		return 0
	}
	// Header + todos + bottom border
	return len(p.todos) + 2
}

// IsActive returns true if the panel has content to show.
func (p *TodoPanel) IsActive() bool {
	return len(p.todos) > 0
}

// HasInProgress returns true if any todo is in progress.
func (p *TodoPanel) HasInProgress() bool {
	for _, todo := range p.todos {
		if todo.Status == tools.TodoStatusInProgress {
			return true
		}
	}
	return false
}

// View renders the todo panel.
func (p *TodoPanel) View() string {
	if !p.IsActive() {
		return ""
	}

	t := styles.CurrentTheme()
	lines := make([]string, 0, len(p.todos)+1) // Pre-allocate for header + todos

	// Header
	headerStyle := t.S().Muted.Bold(true)
	lines = append(lines, headerStyle.Render("─ Tasks "))

	// Todo items
	for _, todo := range p.todos {
		line := p.renderTodoItem(t, todo)
		lines = append(lines, line)
	}

	// Bottom border
	lines = append(lines, t.S().Muted.Render(strings.Repeat("─", 10)))

	content := strings.Join(lines, "\n")

	// Apply padding
	return lipgloss.NewStyle().
		Padding(0, 1).
		Width(p.width).
		Render(content)
}

// renderTodoItem renders a single todo item with status indicator.
func (p *TodoPanel) renderTodoItem(t *styles.Theme, todo tools.TodoItem) string {
	var icon string
	var iconStyle lipgloss.Style
	var text string

	switch todo.Status {
	case tools.TodoStatusCompleted:
		icon = todoIconCompleted
		iconStyle = t.S().Success
		text = todo.Content // Use imperative form for completed
	case tools.TodoStatusInProgress:
		// Use spinner animation
		icon = spinnerFrames[p.spinner]
		iconStyle = t.S().Warning
		text = todo.ActiveForm // Use active form for in-progress
	case tools.TodoStatusPending:
		icon = todoIconPending
		iconStyle = t.S().Muted
		text = todo.Content // Use imperative form for pending
	}

	// Format: "  icon text"
	return "  " + iconStyle.Render(icon) + " " + p.styledText(t, todo.Status, text)
}

// styledText applies appropriate styling to todo text based on status.
func (p *TodoPanel) styledText(t *styles.Theme, status tools.TodoStatus, text string) string {
	text = p.truncateText(text)

	switch status {
	case tools.TodoStatusCompleted:
		// Completed items are dimmed
		return t.S().Muted.Render(text)
	case tools.TodoStatusInProgress:
		// In-progress items are highlighted
		return t.S().Text.Bold(true).Render(text)
	case tools.TodoStatusPending:
		// Pending items are normal
		return t.S().Text.Render(text)
	}
	return t.S().Text.Render(text)
}

// truncateText truncates text to fit the available width.
func (p *TodoPanel) truncateText(text string) string {
	// Reserve space for icon, padding, and some margin
	maxLen := p.width - 10
	if maxLen < 20 {
		maxLen = 20
	}

	if len(text) <= maxLen {
		return text
	}

	return text[:maxLen-3] + "..."
}

// Progress returns the completion progress as a string (e.g., "2/5").
func (p *TodoPanel) Progress() string {
	if len(p.todos) == 0 {
		return ""
	}

	completed := 0
	for _, todo := range p.todos {
		if todo.Status == tools.TodoStatusCompleted {
			completed++
		}
	}

	return string(rune('0'+completed)) + "/" + string(rune('0'+len(p.todos)))
}
