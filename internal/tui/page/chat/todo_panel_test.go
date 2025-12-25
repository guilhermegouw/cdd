package chat

import (
	"strings"
	"testing"

	"github.com/guilhermegouw/cdd/internal/tools"
)

func TestTodoPanel_NewTodoPanel(t *testing.T) {
	p := NewTodoPanel()

	if p == nil {
		t.Fatal("expected non-nil panel")
	}
	if len(p.todos) != 0 {
		t.Error("expected empty todos initially")
	}
	if p.spinner != 0 {
		t.Error("expected spinner=0 initially")
	}
}

func TestTodoPanel_Height(t *testing.T) {
	p := NewTodoPanel()
	p.SetWidth(80)

	// Empty panel has zero height
	if h := p.Height(); h != 0 {
		t.Errorf("expected height=0 when empty, got %d", h)
	}

	// One todo: header + 1 todo + footer = 3
	p.SetTodos([]tools.TodoItem{
		{Content: "Task 1", ActiveForm: "Doing task 1", Status: tools.TodoStatusPending},
	})
	if h := p.Height(); h != 3 {
		t.Errorf("expected height=3 with 1 todo, got %d", h)
	}

	// Three todos: header + 3 todos + footer = 5
	p.SetTodos([]tools.TodoItem{
		{Content: "Task 1", ActiveForm: "Doing task 1", Status: tools.TodoStatusCompleted},
		{Content: "Task 2", ActiveForm: "Doing task 2", Status: tools.TodoStatusInProgress},
		{Content: "Task 3", ActiveForm: "Doing task 3", Status: tools.TodoStatusPending},
	})
	if h := p.Height(); h != 5 {
		t.Errorf("expected height=5 with 3 todos, got %d", h)
	}

	// Clear resets height
	p.Clear()
	if h := p.Height(); h != 0 {
		t.Errorf("expected height=0 after clear, got %d", h)
	}
}

func TestTodoPanel_IsActive(t *testing.T) {
	p := NewTodoPanel()

	if p.IsActive() {
		t.Error("expected IsActive=false when empty")
	}

	p.SetTodos([]tools.TodoItem{
		{Content: "Task 1", ActiveForm: "Doing task 1", Status: tools.TodoStatusPending},
	})
	if !p.IsActive() {
		t.Error("expected IsActive=true when has todos")
	}

	p.SetTodos(nil)
	if p.IsActive() {
		t.Error("expected IsActive=false after setting nil")
	}
}

func TestTodoPanel_HasInProgress(t *testing.T) {
	p := NewTodoPanel()

	// No todos
	if p.HasInProgress() {
		t.Error("expected HasInProgress=false when empty")
	}

	// Only pending
	p.SetTodos([]tools.TodoItem{
		{Content: "Task 1", ActiveForm: "Doing task 1", Status: tools.TodoStatusPending},
	})
	if p.HasInProgress() {
		t.Error("expected HasInProgress=false with only pending")
	}

	// With in_progress
	p.SetTodos([]tools.TodoItem{
		{Content: "Task 1", ActiveForm: "Doing task 1", Status: tools.TodoStatusCompleted},
		{Content: "Task 2", ActiveForm: "Doing task 2", Status: tools.TodoStatusInProgress},
	})
	if !p.HasInProgress() {
		t.Error("expected HasInProgress=true with in_progress todo")
	}
}

func TestTodoPanel_SetSpinner(t *testing.T) {
	p := NewTodoPanel()

	p.SetSpinner(5)
	if p.spinner != 5 {
		t.Errorf("expected spinner=5, got %d", p.spinner)
	}
}

func TestTodoPanel_View_Empty(t *testing.T) {
	p := NewTodoPanel()
	p.SetWidth(80)

	if v := p.View(); v != "" {
		t.Errorf("expected empty view when inactive, got %q", v)
	}
}

func TestTodoPanel_View_WithTodos(t *testing.T) {
	p := NewTodoPanel()
	p.SetWidth(80)

	p.SetTodos([]tools.TodoItem{
		{Content: "Read files", ActiveForm: "Reading files", Status: tools.TodoStatusCompleted},
		{Content: "Write code", ActiveForm: "Writing code", Status: tools.TodoStatusInProgress},
		{Content: "Run tests", ActiveForm: "Running tests", Status: tools.TodoStatusPending},
	})

	view := p.View()

	// Should contain header
	if !strings.Contains(view, "Tasks") {
		t.Error("expected 'Tasks' header in view")
	}

	// Completed task should show content (imperative form)
	if !strings.Contains(view, "Read files") {
		t.Error("expected completed task content 'Read files' in view")
	}

	// In-progress task should show activeForm
	if !strings.Contains(view, "Writing code") {
		t.Error("expected in-progress task activeForm 'Writing code' in view")
	}

	// Pending task should show content
	if !strings.Contains(view, "Run tests") {
		t.Error("expected pending task content 'Run tests' in view")
	}

	// Should contain status icons
	if !strings.Contains(view, todoIconCompleted) {
		t.Error("expected completed icon in view")
	}
	if !strings.Contains(view, todoIconPending) {
		t.Error("expected pending icon in view")
	}
}

func TestTodoPanel_View_SpinnerAnimation(t *testing.T) {
	p := NewTodoPanel()
	p.SetWidth(80)

	p.SetTodos([]tools.TodoItem{
		{Content: "Task", ActiveForm: "Doing task", Status: tools.TodoStatusInProgress},
	})

	// Get first frame
	p.SetSpinner(0)
	frame1 := p.View()

	// Get second frame
	p.SetSpinner(1)
	frame2 := p.View()

	// Frames should be different (spinner advanced)
	if frame1 == frame2 {
		t.Error("expected spinner to change between frames")
	}

	// Both should contain spinner frames
	if !containsSpinnerFrame(frame1) {
		t.Error("expected spinner character in first frame")
	}
	if !containsSpinnerFrame(frame2) {
		t.Error("expected spinner character in second frame")
	}
}

func TestTodoPanel_Clear(t *testing.T) {
	p := NewTodoPanel()

	p.SetTodos([]tools.TodoItem{
		{Content: "Task 1", ActiveForm: "Doing task 1", Status: tools.TodoStatusPending},
	})
	p.SetSpinner(5)

	p.Clear()

	if len(p.todos) != 0 {
		t.Error("expected empty todos after Clear")
	}
	if p.spinner != 0 {
		t.Error("expected spinner=0 after Clear")
	}
}

func TestTodoPanel_Progress(t *testing.T) {
	p := NewTodoPanel()

	// Empty
	if prog := p.Progress(); prog != "" {
		t.Errorf("expected empty progress when no todos, got %q", prog)
	}

	// None completed
	p.SetTodos([]tools.TodoItem{
		{Content: "Task 1", ActiveForm: "Task 1", Status: tools.TodoStatusPending},
		{Content: "Task 2", ActiveForm: "Task 2", Status: tools.TodoStatusInProgress},
	})
	if prog := p.Progress(); prog != "0/2" {
		t.Errorf("expected progress '0/2', got %q", prog)
	}

	// Some completed
	p.SetTodos([]tools.TodoItem{
		{Content: "Task 1", ActiveForm: "Task 1", Status: tools.TodoStatusCompleted},
		{Content: "Task 2", ActiveForm: "Task 2", Status: tools.TodoStatusCompleted},
		{Content: "Task 3", ActiveForm: "Task 3", Status: tools.TodoStatusInProgress},
		{Content: "Task 4", ActiveForm: "Task 4", Status: tools.TodoStatusPending},
	})
	if prog := p.Progress(); prog != "2/4" {
		t.Errorf("expected progress '2/4', got %q", prog)
	}

	// All completed
	p.SetTodos([]tools.TodoItem{
		{Content: "Task 1", ActiveForm: "Task 1", Status: tools.TodoStatusCompleted},
		{Content: "Task 2", ActiveForm: "Task 2", Status: tools.TodoStatusCompleted},
	})
	if prog := p.Progress(); prog != "2/2" {
		t.Errorf("expected progress '2/2', got %q", prog)
	}
}

func TestTodoPanel_TruncateText(t *testing.T) {
	p := NewTodoPanel()
	p.SetWidth(50) // Will give maxLen of 40

	// Short text unchanged
	short := "Short task"
	if result := p.truncateText(short); result != short {
		t.Errorf("expected short text unchanged, got %q", result)
	}

	// Long text truncated
	long := "This is a very long task description that should be truncated with ellipsis"
	result := p.truncateText(long)
	if !strings.HasSuffix(result, "...") {
		t.Error("expected truncated text to end with '...'")
	}
	if len(result) > 43 { // 40 + "..."
		t.Errorf("expected truncated text to be <= 43 chars, got %d", len(result))
	}
}

func TestTodoPanel_SetTodos_Nil(t *testing.T) {
	p := NewTodoPanel()

	p.SetTodos([]tools.TodoItem{
		{Content: "Task", ActiveForm: "Task", Status: tools.TodoStatusPending},
	})

	p.SetTodos(nil)

	if len(p.todos) != 0 {
		t.Error("expected empty todos after SetTodos(nil)")
	}
	if p.IsActive() {
		t.Error("expected IsActive=false after SetTodos(nil)")
	}
}
