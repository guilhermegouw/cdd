package chat

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
)

// Input is the chat input component.
type Input struct {
	textInput textinput.Model
	width     int
	enabled   bool
}

// NewInput creates a new input component.
func NewInput() *Input {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.CharLimit = 4096
	ti.Focus()

	return &Input{
		textInput: ti,
		enabled:   true,
	}
}

// Init initializes the input.
func (i *Input) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles input events.
func (i *Input) Update(msg tea.Msg) (*Input, tea.Cmd) {
	if !i.enabled {
		return i, nil
	}

	var cmd tea.Cmd
	i.textInput, cmd = i.textInput.Update(msg)
	return i, cmd
}

// View renders the input.
func (i *Input) View() string {
	t := styles.CurrentTheme()

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocus).
		Padding(0, 1).
		Width(i.width - 4)

	if !i.enabled {
		inputStyle = inputStyle.BorderForeground(t.Border)
	}

	return inputStyle.Render(i.textInput.View())
}

// SetWidth sets the input width.
func (i *Input) SetWidth(width int) {
	i.width = width
	i.textInput.SetWidth(width - 8) // Account for border and padding
}

// Value returns the current input value.
func (i *Input) Value() string {
	return i.textInput.Value()
}

// SetValue sets the input value.
func (i *Input) SetValue(value string) {
	i.textInput.SetValue(value)
}

// Clear clears the input.
func (i *Input) Clear() {
	i.textInput.SetValue("")
}

// Enable enables the input.
func (i *Input) Enable() {
	i.enabled = true
	i.textInput.Focus()
}

// Disable disables the input.
func (i *Input) Disable() {
	i.enabled = false
	i.textInput.Blur()
}

// IsEnabled returns whether the input is enabled.
func (i *Input) IsEnabled() bool {
	return i.enabled
}

// Focus focuses the input.
func (i *Input) Focus() tea.Cmd {
	return i.textInput.Focus()
}

// Blur removes focus from the input.
func (i *Input) Blur() {
	i.textInput.Blur()
}

// Cursor returns the cursor for the input.
func (i *Input) Cursor() *tea.Cursor {
	return i.textInput.Cursor()
}
