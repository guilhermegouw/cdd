package chat

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/guilhermegouw/cdd/internal/agent"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
)

// MessageList displays the conversation messages.
type MessageList struct {
	messages []agent.Message
	width    int
	height   int
	offset   int
}

// NewMessageList creates a new message list component.
func NewMessageList() *MessageList {
	return &MessageList{
		messages: []agent.Message{},
	}
}

// SetMessages sets the messages to display.
func (m *MessageList) SetMessages(messages []agent.Message) {
	m.messages = messages
}

// AppendMessage adds a message to the list.
func (m *MessageList) AppendMessage(msg agent.Message) {
	m.messages = append(m.messages, msg)
}

// UpdateLast updates the last message (for streaming).
func (m *MessageList) UpdateLast(content string) {
	if len(m.messages) == 0 {
		return
	}
	m.messages[len(m.messages)-1].Content = content
}

// SetSize sets the component size.
func (m *MessageList) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// ScrollUp scrolls the message list up.
func (m *MessageList) ScrollUp() {
	if m.offset > 0 {
		m.offset--
	}
}

// ScrollDown scrolls the message list down.
func (m *MessageList) ScrollDown() {
	// We'll handle bounds in View
	m.offset++
}

// ScrollToBottom scrolls to the bottom of the list.
func (m *MessageList) ScrollToBottom() {
	m.offset = 0 // We render bottom-up, so 0 is the bottom
}

// View renders the message list.
func (m *MessageList) View() string {
	t := styles.CurrentTheme()

	if len(m.messages) == 0 {
		empty := t.S().Muted.Render("No messages yet. Type something to start chatting.")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, empty)
	}

	// Render messages
	var rendered []string
	for _, msg := range m.messages {
		rendered = append(rendered, m.renderMessage(msg))
	}

	// Join with spacing
	content := strings.Join(rendered, "\n\n")

	// Create scrollable container
	containerStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(0, 1)

	return containerStyle.Render(content)
}

func (m *MessageList) renderMessage(msg agent.Message) string {
	t := styles.CurrentTheme()

	contentWidth := m.width - 4 // Account for padding

	switch msg.Role {
	case agent.RoleUser:
		return m.renderUserMessage(msg, contentWidth)
	case agent.RoleAssistant:
		return m.renderAssistantMessage(msg, contentWidth)
	case agent.RoleTool:
		return m.renderToolMessage(msg, contentWidth)
	default:
		return t.S().Muted.Render(msg.Content)
	}
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

	var parts []string
	parts = append(parts, header)

	if msg.Content != "" {
		content := t.S().Text.Width(width).Render(msg.Content)
		parts = append(parts, content)
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

	var parts []string
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
