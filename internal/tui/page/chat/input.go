package chat

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/guilhermegouw/cdd/internal/debug"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
)

// Input is the chat input component.
type Input struct {
	textArea textarea.Model
	width    int
	enabled  bool
	viewID   int // Debug: track view renders
}

// NewInput creates a new input component.
func NewInput() *Input {
	ta := textarea.New()
	ta.Placeholder = "Type a message... (ctrl+j for newline)"
	ta.CharLimit = 4096
	ta.MaxHeight = 5 // Allow up to 5 lines
	ta.SetHeight(1)  // Start with single line
	ta.ShowLineNumbers = false
	ta.Focus()

	// Remove cursor line highlight
	taStyles := ta.Styles()
	taStyles.Focused.CursorLine = lipgloss.NewStyle()
	taStyles.Blurred.CursorLine = lipgloss.NewStyle()
	ta.SetStyles(taStyles)

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

	// Track lines before update to detect large content changes (like paste)
	linesBefore := i.textArea.LineCount()

	// Pre-expand height before processing newline to prevent viewport scrolling.
	// This ensures the textarea has room for the new line before it's added,
	// so the viewport doesn't scroll and hide the first line.
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if key.Matches(keyMsg, i.textArea.KeyMap.InsertNewline) {
			currentLines := i.textArea.LineCount()
			newHeight := currentLines + 1
			if newHeight > 5 {
				newHeight = 5
			}
			i.textArea.SetHeight(newHeight)
		}
	}

	var cmd tea.Cmd
	i.textArea, cmd = i.textArea.Update(msg)

	// Adjust height based on actual content (handles deletions and other changes)
	actualLines := i.textArea.LineCount()
	displayLines := actualLines
	if displayLines < 1 {
		displayLines = 1
	}
	if displayLines > 5 {
		displayLines = 5
	}
	i.textArea.SetHeight(displayLines)

	// If content grew significantly (paste) and exceeds visible area, scroll to show cursor
	if actualLines > 5 && actualLines-linesBefore > 1 {
		i.textArea.MoveToEnd()
	}

	return i, cmd
}

// View renders the input.
func (i *Input) View() string {
	t := styles.CurrentTheme()

	// Track view renders for debugging
	i.viewID++
	debug.Event("input", "View", fmt.Sprintf("viewID=%d width=%d taWidth=%d taHeight=%d enabled=%v focused=%v", i.viewID, i.width, i.textArea.Width(), i.textArea.Height(), i.enabled, i.textArea.Focused()))

	// Ensure width is never negative
	width := i.width - 4
	if width < 1 {
		width = 1
	}

	textAreaView := i.textArea.View()

	// Debug: log exact content of textarea view
	taLines := strings.Count(textAreaView, "\n") + 1
	debug.Event("input", "TextAreaView", fmt.Sprintf("lines=%d len=%d expectedHeight=%d", taLines, len(textAreaView), i.textArea.Height()))
	if taLines > i.textArea.Height() {
		debug.Event("input", "TextAreaOVERFLOW", fmt.Sprintf("textArea returned %d lines but height is %d!", taLines, i.textArea.Height()))
	}

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocus).
		Padding(0, 1).
		Width(width)

	if !i.enabled {
		inputStyle = inputStyle.BorderForeground(t.Border)
	}

	// Debug: add view ID to help identify duplicates
	debug.Event("input", "Render", fmt.Sprintf("viewID=%d rendering input, textAreaView len=%d lines=%d", i.viewID, len(textAreaView), strings.Count(textAreaView, "\n")+1))
	result := inputStyle.Render(textAreaView)
	debug.Event("input", "Render", fmt.Sprintf("viewID=%d result len=%d lines=%d", i.viewID, len(result), strings.Count(result, "\n")+1))
	// Log first 100 chars of textarea view to see what it contains
	if textAreaView != "" {
		preview := textAreaView
		if len(preview) > 100 {
			preview = preview[:100]
		}
		debug.Event("input", "TextAreaPreview", fmt.Sprintf("%q", preview))
	}
	return result
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

// Height returns the current height of the input including borders.
func (i *Input) Height() int {
	// textarea height + 2 for border (top + bottom)
	h := i.textArea.Height() + 2
	debug.Event("input", "Height", fmt.Sprintf("textAreaHeight=%d totalHeight=%d", i.textArea.Height(), h))
	return h
}
