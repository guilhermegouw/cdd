package sessions

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/guilhermegouw/cdd/internal/tui/styles"
)

// BorderedPanel renders content inside a bordered box with a centered title.
type BorderedPanel struct {
	title   string
	content string
	width   int
	height  int
	focused bool
}

// NewBorderedPanel creates a new bordered panel.
func NewBorderedPanel() *BorderedPanel {
	return &BorderedPanel{}
}

// SetTitle sets the title to display in the top border.
func (p *BorderedPanel) SetTitle(title string) {
	p.title = title
}

// SetContent sets the content to render inside the panel.
func (p *BorderedPanel) SetContent(content string) {
	p.content = content
}

// SetSize sets the panel dimensions.
func (p *BorderedPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// SetFocused sets whether the panel has focus (affects border color).
func (p *BorderedPanel) SetFocused(focused bool) {
	p.focused = focused
}

// View renders the bordered panel.
func (p *BorderedPanel) View() string {
	t := styles.CurrentTheme()

	// Determine border color based on focus
	borderColor := t.Border
	if p.focused {
		borderColor = t.BorderFocus
	}
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	titleStyle := t.S().Primary.Bold(true)

	// Calculate dimensions
	// Border structure: ╭ + content + ╮ = width
	// So inner content width = width - 2 (for left and right border chars)
	borderWidth := p.width - 2
	if borderWidth < 4 {
		borderWidth = 4
	}

	// Content width accounts for padding spaces inside borders
	contentWidth := borderWidth - 2 // -2 for " " padding on each side

	// Truncate title if too long (leave room for at least 2 dashes on each side)
	title := p.title
	maxTitleLen := borderWidth - 4
	if len(title) > maxTitleLen && maxTitleLen > 3 {
		title = title[:maxTitleLen-3] + "..."
	}

	// Render title with style
	titleRendered := titleStyle.Render(title)
	titleVisualLen := lipgloss.Width(titleRendered)

	// Calculate padding for centered title
	remainingSpace := borderWidth - titleVisualLen
	leftPadding := remainingSpace / 2
	rightPadding := remainingSpace - leftPadding

	// Ensure non-negative padding
	if leftPadding < 0 {
		leftPadding = 0
	}
	if rightPadding < 0 {
		rightPadding = 0
	}

	// Build top border with centered title
	topBorder := borderStyle.Render("╭"+strings.Repeat("─", leftPadding)) +
		titleRendered +
		borderStyle.Render(strings.Repeat("─", rightPadding)+"╮")

	// Build bottom border
	bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", borderWidth) + "╯")

	// Split content into lines and pad to fit
	contentLines := strings.Split(p.content, "\n")
	var borderedLines []string
	borderedLines = append(borderedLines, topBorder)

	// Calculate content height (total height minus top and bottom borders)
	contentHeight := p.height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	for i := 0; i < contentHeight; i++ {
		line := ""
		if i < len(contentLines) {
			line = contentLines[i]
		}

		// Measure visual width and pad to content width
		lineLen := lipgloss.Width(line)
		if lineLen < contentWidth {
			line += strings.Repeat(" ", contentWidth-lineLen)
		} else if lineLen > contentWidth {
			// Truncate if too long
			line = truncateToWidth(line, contentWidth)
		}

		borderedLines = append(borderedLines,
			borderStyle.Render("│ ")+line+borderStyle.Render(" │"))
	}

	borderedLines = append(borderedLines, bottomBorder)

	return strings.Join(borderedLines, "\n")
}

// truncateToWidth truncates a string to fit within a visual width.
func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 3 {
		return strings.Repeat(".", maxWidth)
	}

	// Simple truncation - for styled strings this may not be perfect
	// but works for plain text
	runes := []rune(s)
	if len(runes) <= maxWidth {
		return s
	}
	return string(runes[:maxWidth-3]) + "..."
}
