package sessions

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/guilhermegouw/cdd/internal/session"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
)

// Preview displays detailed information about a selected session.
type Preview struct {
	session *session.SessionWithPreview
	width   int
	height  int
}

// NewPreview creates a new session preview panel.
func NewPreview() *Preview {
	return &Preview{}
}

// SetSession sets the session to preview.
func (p *Preview) SetSession(sess *session.SessionWithPreview) {
	p.session = sess
}

// SetSize sets the preview panel dimensions.
func (p *Preview) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// View renders the preview panel.
func (p *Preview) View() string {
	t := styles.CurrentTheme()

	if p.session == nil {
		emptyStyle := t.S().Muted.
			Width(p.width).
			Height(p.height).
			Align(lipgloss.Center, lipgloss.Center)
		return emptyStyle.Render("Select a session to preview")
	}

	sess := p.session
	var parts []string

	// Title (full, not truncated)
	title := sess.Title
	if title == "" || title == "New Session" {
		title = fmt.Sprintf("Session %s", sess.ID[:12])
	}
	titleStyle := t.S().Primary.Bold(true).Width(p.width)
	parts = append(parts, titleStyle.Render(title))

	// Horizontal divider under title (full width)
	dividerStyle := t.S().Muted
	divider := strings.Repeat("─", p.width)
	parts = append(parts, dividerStyle.Render(divider))
	parts = append(parts, "")

	// Metadata
	metaStyle := t.S().Muted
	parts = append(parts, metaStyle.Render(fmt.Sprintf("ID: %s", sess.ID[:8])))
	parts = append(parts, metaStyle.Render(fmt.Sprintf("Created: %s", formatDateTime(sess.CreatedAt))))
	parts = append(parts, metaStyle.Render(fmt.Sprintf("Updated: %s", formatRelativeTime(sess.UpdatedAt))))
	parts = append(parts, metaStyle.Render(fmt.Sprintf("Messages: %d", sess.MessageCount)))
	parts = append(parts, "")

	// Preview content
	if sess.FirstMessage != "" {
		previewLabel := t.S().Text.Bold(true).Render("First message:")
		parts = append(parts, previewLabel)
		parts = append(parts, "")

		// Wrap the preview text
		preview := sess.FirstMessage
		preview = strings.ReplaceAll(preview, "\n", " ")
		maxLen := p.width * 3 // Allow ~3 lines of text
		if len(preview) > maxLen {
			preview = preview[:maxLen-3] + "..."
		}

		// Word wrap
		wrapped := wordWrap(preview, p.width-2)
		previewStyle := t.S().Text
		parts = append(parts, previewStyle.Render(wrapped))
	} else {
		noMsgStyle := t.S().Muted.Italic(true)
		parts = append(parts, noMsgStyle.Render("No messages yet"))
	}

	content := strings.Join(parts, "\n")

	// Container style
	containerStyle := lipgloss.NewStyle().
		Width(p.width).
		Height(p.height).
		Padding(0, 1)

	return containerStyle.Render(content)
}

// formatDateTime formats a time as a readable date/time string.
func formatDateTime(t time.Time) string {
	now := time.Now()
	if t.Year() == now.Year() {
		return t.Format("Jan 2, 3:04 PM")
	}
	return t.Format("Jan 2, 2006")
}

// wordWrap wraps text to fit within a given width.
func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	words := strings.Fields(text)
	lineLen := 0

	for i, word := range words {
		wordLen := len(word)

		if lineLen+wordLen+1 > width && lineLen > 0 {
			result.WriteString("\n")
			lineLen = 0
		}

		if lineLen > 0 {
			result.WriteString(" ")
			lineLen++
		}

		// Truncate very long words
		if wordLen > width {
			word = word[:width-3] + "..."
			wordLen = width
		}

		result.WriteString(word)
		lineLen += wordLen

		_ = i // avoid unused variable warning
	}

	return result.String()
}
