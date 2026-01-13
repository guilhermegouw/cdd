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
	panel   *BorderedPanel
	width   int
	height  int
}

// NewPreview creates a new session preview panel.
func NewPreview() *Preview {
	return &Preview{
		panel: NewBorderedPanel(),
	}
}

// SetSession sets the session to preview.
func (p *Preview) SetSession(sess *session.SessionWithPreview) {
	p.session = sess
}

// SetSize sets the preview panel dimensions.
func (p *Preview) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.panel.SetSize(width, height)
}

// Title returns the title for the preview panel.
func (p *Preview) Title() string {
	if p.session == nil {
		return "Preview"
	}
	title := p.session.Title
	if title == "" || title == "New Session" {
		return fmt.Sprintf("Session %s", p.session.ID[:12])
	}
	return title
}

// View renders the preview panel.
func (p *Preview) View() string {
	t := styles.CurrentTheme()

	if p.session == nil {
		p.panel.SetTitle("Preview")
		p.panel.SetContent(t.S().Muted.Render("Select a session to preview"))
		return p.panel.View()
	}

	sess := p.session

	// Build content
	content := p.buildContent(sess)

	// Set panel properties and render
	p.panel.SetTitle(p.Title())
	p.panel.SetContent(content)
	p.panel.SetFocused(false)

	return p.panel.View()
}

// buildContent generates the content for the preview panel.
func (p *Preview) buildContent(sess *session.SessionWithPreview) string {
	t := styles.CurrentTheme()
	var parts []string

	// Session metadata
	metaStyle := t.S().Muted
	parts = append(parts,
		metaStyle.Render(fmt.Sprintf("ID: %s", sess.ID[:8])),
		metaStyle.Render(fmt.Sprintf("Created: %s", formatDateTime(sess.CreatedAt))),
		metaStyle.Render(fmt.Sprintf("Updated: %s", formatRelativeTime(sess.UpdatedAt))),
		metaStyle.Render(fmt.Sprintf("Messages: %d", sess.MessageCount)),
		"",
	)

	// Preview content
	// Content width is panel width - 4 (borders and padding)
	contentWidth := p.width - 4
	if contentWidth < 10 {
		contentWidth = 10
	}

	if sess.FirstMessage != "" {
		previewLabel := t.S().Text.Bold(true).Render("First message:")
		parts = append(parts, previewLabel, "")

		// Wrap the preview text
		preview := sess.FirstMessage
		preview = strings.ReplaceAll(preview, "\n", " ")
		maxLen := contentWidth * 3 // Allow ~3 lines of text
		if len(preview) > maxLen {
			preview = preview[:maxLen-3] + "..."
		}

		// Word wrap
		wrapped := wordWrap(preview, contentWidth-2)
		previewStyle := t.S().Text
		parts = append(parts, previewStyle.Render(wrapped))
	} else {
		noMsgStyle := t.S().Muted.Italic(true)
		parts = append(parts, noMsgStyle.Render("No messages yet"))
	}

	return strings.Join(parts, "\n")
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

// unused import guard.
var _ = lipgloss.Width
