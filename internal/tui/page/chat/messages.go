package chat

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/x/ansi"

	"github.com/guilhermegouw/cdd/internal/agent"
	"github.com/guilhermegouw/cdd/internal/debug"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
)

// MessageList displays the conversation messages with scrolling support.
type MessageList struct { //nolint:govet // fieldalignment: preserving logical field order
	messages   []agent.Message
	viewport   viewport.Model
	mdRenderer *MarkdownRenderer
	width      int
	height     int
	ready      bool

	// Selection state
	selectionStartCol  int
	selectionStartLine int
	selectionEndCol    int
	selectionEndLine   int
	selectionActive    bool
	renderedContent    string // cached for selection extraction
}

// NewMessageList creates a new message list component.
func NewMessageList() *MessageList {
	return &MessageList{
		messages:           []agent.Message{},
		mdRenderer:         NewMarkdownRenderer(),
		selectionStartCol:  -1,
		selectionStartLine: -1,
		selectionEndCol:    -1,
		selectionEndLine:   -1,
	}
}

// SetMessages sets the messages to display.
func (m *MessageList) SetMessages(messages []agent.Message) {
	m.messages = messages
	m.updateContent()
}

// AppendMessage adds a message to the list.
func (m *MessageList) AppendMessage(msg agent.Message) {
	m.messages = append(m.messages, msg)
	m.updateContent()
}

// UpdateLast updates the last message (for streaming).
func (m *MessageList) UpdateLast(content string) {
	if len(m.messages) == 0 {
		return
	}
	m.messages[len(m.messages)-1].Content = content
	m.updateContent()
}

// SetSize sets the component size.
func (m *MessageList) SetSize(width, height int) {
	// Skip if size hasn't changed
	if m.ready && m.width == width && m.height == height {
		return
	}

	m.width = width
	m.height = height

	if !m.ready {
		debug.Event("messages", "SetSize", fmt.Sprintf("initializing viewport width=%d height=%d", width, height))
		m.viewport = viewport.New(
			viewport.WithWidth(width),
			viewport.WithHeight(height),
		)
		m.viewport.MouseWheelEnabled = true
		m.ready = true
		debug.Event("messages", "SetSize", fmt.Sprintf("viewport initialized, mouseEnabled=%v", m.viewport.MouseWheelEnabled))
		m.updateContent()
	} else {
		debug.Event("messages", "SetSize", fmt.Sprintf("resizing viewport width=%d height=%d", width, height))
		m.viewport.SetWidth(width)
		m.viewport.SetHeight(height)
		// Don't call updateContent on resize - preserves scroll position
	}
}

// Update handles viewport events.
func (m *MessageList) Update(msg tea.Msg) (*MessageList, tea.Cmd) {
	debug.Event("messages", "Update", fmt.Sprintf("msgType=%T ready=%v", msg, m.ready))

	if !m.ready {
		debug.Event("messages", "Update", "viewport not ready, skipping")
		return m, nil
	}

	// Log viewport state before update
	debug.Event("messages", "ViewportBefore", fmt.Sprintf(
		"yOffset=%d totalLines=%d height=%d atBottom=%v mouseEnabled=%v",
		m.viewport.YOffset(), m.viewport.TotalLineCount(), m.viewport.Height(),
		m.viewport.AtBottom(), m.viewport.MouseWheelEnabled,
	))

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)

	// Log viewport state after update
	debug.Event("messages", "ViewportAfter", fmt.Sprintf(
		"yOffset=%d atBottom=%v",
		m.viewport.YOffset(), m.viewport.AtBottom(),
	))

	return m, cmd
}

// View renders the message list.
func (m *MessageList) View() string {
	if !m.ready {
		return "Loading..."
	}

	view := m.viewport.View()

	// Apply selection highlighting if there's a selection
	if m.HasSelection() {
		view = m.applySelectionHighlight(view)
	}

	return view
}

// ScrollToBottom scrolls to the bottom of the list.
func (m *MessageList) ScrollToBottom() {
	m.viewport.GotoBottom()
}

// updateContent re-renders all messages and updates the viewport.
func (m *MessageList) updateContent() {
	if !m.ready {
		return
	}

	t := styles.CurrentTheme()

	if len(m.messages) == 0 {
		empty := t.S().Muted.Render("No messages yet. Type something to start chatting.")
		content := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, empty)
		m.viewport.SetContent(content)
		m.renderedContent = ""
		return
	}

	// Check if we were at the bottom before updating
	wasAtBottom := m.viewport.AtBottom()

	// Render messages
	rendered := make([]string, 0, len(m.messages))
	for _, msg := range m.messages {
		rendered = append(rendered, m.renderMessage(msg))
	}

	// Join with spacing
	content := strings.Join(rendered, "\n\n")

	// Add padding
	paddedContent := lipgloss.NewStyle().
		Width(m.width-2).
		Padding(0, 1).
		Render(content)

	// Cache for selection text extraction
	m.renderedContent = paddedContent

	m.viewport.SetContent(paddedContent)

	// Only auto-scroll to bottom if we were already at the bottom
	// This preserves scroll position when user is reading earlier content
	if wasAtBottom {
		m.viewport.GotoBottom()
	}
}

func (m *MessageList) renderMessage(msg agent.Message) string {
	contentWidth := m.width - 4 // Account for padding
	if contentWidth < 1 {
		contentWidth = 1
	}

	switch msg.Role {
	case agent.RoleUser:
		return m.renderUserMessage(msg, contentWidth)
	case agent.RoleAssistant:
		return m.renderAssistantMessage(msg, contentWidth)
	case agent.RoleTool:
		return m.renderToolMessage(msg, contentWidth)
	case agent.RoleSystem:
		// System messages are typically not displayed in chat
		return ""
	}
	return "" // Unreachable, all cases handled
}

func (m *MessageList) renderUserMessage(msg agent.Message, width int) string {
	t := styles.CurrentTheme()

	header := t.S().Text.Bold(true).Render("You")
	content := t.S().Text.Width(width).Render(msg.Content)

	return lipgloss.JoinVertical(lipgloss.Left, header, content)
}

func (m *MessageList) renderAssistantMessage(msg agent.Message, width int) string {
	t := styles.CurrentTheme()

	header := t.S().Primary.Bold(true).Render("Assistant")

	parts := make([]string, 0, 3)
	parts = append(parts, header)

	if msg.Content != "" {
		// Try to render markdown
		rendered, err := m.mdRenderer.Render(msg.Content, width)
		if err != nil {
			// Fallback to plain text on error
			rendered = t.S().Text.Width(width).Render(msg.Content)
		}
		// Trim trailing newlines that glamour adds
		rendered = strings.TrimRight(rendered, "\n")
		parts = append(parts, rendered)
	}

	// Render tool calls if any
	for _, tc := range msg.ToolCalls {
		toolCall := t.S().Muted.Render(fmt.Sprintf("  [Tool: %s]", tc.Name))
		parts = append(parts, toolCall)
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m *MessageList) renderToolMessage(msg agent.Message, width int) string {
	t := styles.CurrentTheme()

	parts := make([]string, 0, len(msg.ToolResults)*2)
	for _, tr := range msg.ToolResults {
		header := t.S().Muted.Render(fmt.Sprintf("  [Result: %s]", tr.Name))

		var contentStyle lipgloss.Style
		if tr.IsError {
			contentStyle = t.S().Error
		} else {
			contentStyle = t.S().Subtle
		}

		// Truncate long results
		content := tr.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}

		result := contentStyle.Width(width - 4).Render(content)
		parts = append(parts, header, result)
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// Selection methods

// StartSelection begins a text selection at the given coordinates.
func (m *MessageList) StartSelection(col, line int) {
	// Adjust for viewport scroll offset
	line += m.viewport.YOffset()

	m.selectionStartCol = col
	m.selectionStartLine = line
	m.selectionEndCol = col
	m.selectionEndLine = line
	m.selectionActive = true

	debug.Event("messages", "StartSelection", fmt.Sprintf("col=%d line=%d", col, line))
}

// EndSelection updates the end point of the current selection.
func (m *MessageList) EndSelection(col, line int) {
	if !m.selectionActive {
		return
	}

	// Adjust for viewport scroll offset
	line += m.viewport.YOffset()

	m.selectionEndCol = col
	m.selectionEndLine = line

	debug.Event("messages", "EndSelection", fmt.Sprintf("col=%d line=%d", col, line))
}

// SelectionStop stops the active selection (mouse released).
func (m *MessageList) SelectionStop() {
	m.selectionActive = false
}

// SelectionClear clears the current selection.
func (m *MessageList) SelectionClear() {
	m.selectionStartCol = -1
	m.selectionStartLine = -1
	m.selectionEndCol = -1
	m.selectionEndLine = -1
	m.selectionActive = false
}

// HasSelection returns whether there is a non-empty selection.
func (m *MessageList) HasSelection() bool {
	return m.selectionStartCol >= 0 &&
		(m.selectionEndCol != m.selectionStartCol || m.selectionEndLine != m.selectionStartLine)
}

// IsSelecting returns whether selection is currently active (mouse down).
func (m *MessageList) IsSelecting() bool {
	return m.selectionActive
}

// CopySelection copies the selected text to clipboard and returns a command.
func (m *MessageList) CopySelection() tea.Cmd {
	if !m.HasSelection() {
		return nil
	}

	selectedText := m.getSelectedText()
	if selectedText == "" {
		return nil
	}

	// Clear selection after copy
	m.SelectionClear()

	// Use both OSC 52 and native clipboard for compatibility
	return tea.Batch(
		tea.SetClipboard(selectedText),
		func() tea.Msg {
			//nolint:errcheck // Best effort clipboard write; OSC 52 is primary
			clipboard.WriteAll(selectedText)
			return SelectionCopiedMsg{Text: selectedText}
		},
	)
}

// SelectionCopiedMsg is sent when text has been copied to clipboard.
type SelectionCopiedMsg struct {
	Text string
}

// getSelectedText extracts the text within the current selection.
//
//nolint:gocyclo // Complex due to multi-line selection handling
func (m *MessageList) getSelectedText() string {
	if !m.HasSelection() || m.renderedContent == "" {
		return ""
	}

	// Normalize selection coordinates (start should be before end)
	startLine, endLine := m.selectionStartLine, m.selectionEndLine
	startCol, endCol := m.selectionStartCol, m.selectionEndCol

	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}

	lines := strings.Split(m.renderedContent, "\n")
	var result strings.Builder

	for lineIdx := startLine; lineIdx <= endLine && lineIdx < len(lines); lineIdx++ {
		if lineIdx < 0 {
			continue
		}

		line := ansi.Strip(lines[lineIdx])

		var lineStart, lineEnd int
		switch {
		case startLine == endLine:
			// Single line selection
			lineStart = startCol
			lineEnd = endCol
		case lineIdx == startLine:
			// First line of multi-line selection
			lineStart = startCol
			lineEnd = len(line)
		case lineIdx == endLine:
			// Last line of multi-line selection
			lineStart = 0
			lineEnd = endCol
		default:
			// Middle lines
			lineStart = 0
			lineEnd = len(line)
		}

		// Clamp to line bounds
		if lineStart < 0 {
			lineStart = 0
		}
		if lineEnd > len(line) {
			lineEnd = len(line)
		}
		if lineStart > lineEnd {
			lineStart = lineEnd
		}

		if lineStart < len(line) {
			result.WriteString(line[lineStart:lineEnd])
		}

		if lineIdx < endLine {
			result.WriteByte('\n')
		}
	}

	return strings.TrimSpace(result.String())
}

// applySelectionHighlight renders the view with selection highlighting.
//
//nolint:gocyclo // Complex due to multi-line selection highlighting
func (m *MessageList) applySelectionHighlight(content string) string {
	if !m.HasSelection() {
		return content
	}

	t := styles.CurrentTheme()
	selStyle := t.S().TextSelection

	// Normalize selection coordinates
	startLine, endLine := m.selectionStartLine, m.selectionEndLine
	startCol, endCol := m.selectionStartCol, m.selectionEndCol

	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}

	// Adjust for viewport offset (convert absolute line to viewport-relative)
	viewportOffset := m.viewport.YOffset()
	startLine -= viewportOffset
	endLine -= viewportOffset

	lines := strings.Split(content, "\n")
	var result strings.Builder

	for lineIdx, line := range lines {
		if lineIdx >= startLine && lineIdx <= endLine {
			// This line has selection
			plainLine := ansi.Strip(line)

			var lineStart, lineEnd int
			switch {
			case startLine == endLine:
				lineStart = startCol
				lineEnd = endCol
			case lineIdx == startLine:
				lineStart = startCol
				lineEnd = len(plainLine)
			case lineIdx == endLine:
				lineStart = 0
				lineEnd = endCol
			default:
				lineStart = 0
				lineEnd = len(plainLine)
			}

			// Clamp to line bounds
			if lineStart < 0 {
				lineStart = 0
			}
			if lineEnd > len(plainLine) {
				lineEnd = len(plainLine)
			}
			if lineStart > len(plainLine) {
				lineStart = len(plainLine)
			}

			// Build the highlighted line
			if lineStart >= lineEnd {
				result.WriteString(line)
			} else {
				// Before selection
				if lineStart > 0 && lineStart <= len(plainLine) {
					result.WriteString(plainLine[:lineStart])
				}
				// Selected text
				if lineEnd > lineStart {
					selectedPart := plainLine[lineStart:lineEnd]
					result.WriteString(selStyle.Render(selectedPart))
				}
				// After selection
				if lineEnd < len(plainLine) {
					result.WriteString(plainLine[lineEnd:])
				}
			}
		} else {
			result.WriteString(line)
		}

		if lineIdx < len(lines)-1 {
			result.WriteByte('\n')
		}
	}

	return result.String()
}
