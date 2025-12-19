package chat

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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
}

// NewMessageList creates a new message list component.
func NewMessageList() *MessageList {
	return &MessageList{
		messages:   []agent.Message{},
		mdRenderer: NewMarkdownRenderer(),
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
	return m.viewport.View()
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
