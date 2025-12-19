package chat

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/guilhermegouw/cdd/internal/tui/styles"
)

// Input is the chat input component.
type Input struct {
	textArea textarea.Model
	width    int
	enabled  bool
}

// NewInput creates a new input component.
func NewInput() *Input {
	ta := textarea.New()
	ta.Placeholder = "Type a message... (ctrl+j for newline)"
	ta.CharLimit = 4096
	ta.MaxHeight = 5           // Allow up to 5 lines
	ta.SetHeight(1)            // Start with single line
	ta.ShowLineNumbers = false
	ta.Focus()

	// Remove cursor line highlight
	styles := ta.Styles()
	styles.Focused.CursorLine = lipgloss.NewStyle()
	styles.Blurred.CursorLine = lipgloss.NewStyle()
	ta.SetStyles(styles)

	// Customize key bindings: Enter should NOT insert newline (we handle submit externally)
	// ctrl+j will insert newline
	ta.KeyMap.InsertNewline = key.NewBinding(
		key.WithKeys("ctrl+j"),
		key.WithHelp("ctrl+j", "new line"),
	)

	return &Input{
		textArea: ta,
		enabled:  true,
	}
}

// Init initializes the input.
func (i *Input) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles input events.
func (i *Input) Update(msg tea.Msg) (*Input, tea.Cmd) {
	if !i.enabled {
		return i, nil
	}

	var cmd tea.Cmd
	i.textArea, cmd = i.textArea.Update(msg)

	// Auto-resize height based on content (max 5 lines)
	lines := i.textArea.LineCount()
	if lines < 1 {
		lines = 1
	}
	if lines > 5 {
		lines = 5
	}
	i.textArea.SetHeight(lines)

	return i, cmd
}

// View renders the input.
func (i *Input) View() string {
	t := styles.CurrentTheme()

	// Ensure width is never negative
	width := i.width - 4
	if width < 1 {
		width = 1
	}

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocus).
		Padding(0, 1).
		Width(width)

	if !i.enabled {
		inputStyle = inputStyle.BorderForeground(t.Border)
	}

	return inputStyle.Render(i.textArea.View())
}

// SetWidth sets the input width.
func (i *Input) SetWidth(width int) {
	i.width = width
	// Ensure textarea width is never negative
	inputWidth := width - 8 // Account for border and padding
	if inputWidth < 1 {
		inputWidth = 1
	}
	i.textArea.SetWidth(inputWidth)
}

// Value returns the current input value.
func (i *Input) Value() string {
	return i.textArea.Value()
}

// SetValue sets the input value.
func (i *Input) SetValue(value string) {
	i.textArea.SetValue(value)
}

// Clear clears the input.
func (i *Input) Clear() {
	i.textArea.SetValue("")
	i.textArea.SetHeight(1)
}

// Enable enables the input.
func (i *Input) Enable() {
	i.enabled = true
	i.textArea.Focus()
}

// Disable disables the input.
func (i *Input) Disable() {
	i.enabled = false
	i.textArea.Blur()
}

// IsEnabled returns whether the input is enabled.
func (i *Input) IsEnabled() bool {
	return i.enabled
}

// Focus focuses the input.
func (i *Input) Focus() tea.Cmd {
	return i.textArea.Focus()
}

// Blur removes focus from the input.
func (i *Input) Blur() {
	i.textArea.Blur()
}

// Cursor returns the cursor for the input.
func (i *Input) Cursor() *tea.Cursor {
	return i.textArea.Cursor()
}
