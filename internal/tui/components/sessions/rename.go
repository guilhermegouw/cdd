package sessions

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/guilhermegouw/cdd/internal/tui/styles"
)

// RenameInput is a text input for renaming sessions.
type RenameInput struct {
	input textinput.Model
	width int
}

// NewRenameInput creates a new rename input.
func NewRenameInput() *RenameInput {
	ti := textinput.New()
	ti.Placeholder = "Enter session name..."
	ti.CharLimit = 100

	return &RenameInput{
		input: ti,
	}
}

// SetWidth sets the input width.
func (r *RenameInput) SetWidth(width int) {
	r.width = width
	// Note: textinput width is handled by the container style
}

// SetValue sets the input value.
func (r *RenameInput) SetValue(value string) {
	r.input.SetValue(value)
	r.input.CursorEnd()
}

// Value returns the current input value.
func (r *RenameInput) Value() string {
	return r.input.Value()
}

// Focus focuses the input.
func (r *RenameInput) Focus() tea.Cmd {
	return r.input.Focus()
}

// Reset clears the input.
func (r *RenameInput) Reset() {
	r.input.SetValue("")
	r.input.Blur()
}

// Update handles messages.
func (r *RenameInput) Update(msg tea.Msg) (*RenameInput, tea.Cmd) {
	var cmd tea.Cmd
	r.input, cmd = r.input.Update(msg)
	return r, cmd
}

// View renders the input.
func (r *RenameInput) View() string {
	t := styles.CurrentTheme()

	label := t.S().Text.Render("New name: ")
	input := r.input.View()

	return label + "\n\n" + input
}

// Cursor returns the cursor position.
func (r *RenameInput) Cursor() *tea.Cursor {
	return r.input.Cursor()
}
