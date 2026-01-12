package sessions

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/guilhermegouw/cdd/internal/tui/styles"
)

// SearchBox is a search input component with count display.
type SearchBox struct {
	input        textinput.Model
	width        int
	filteredCnt  int
	totalCnt     int
	visible      bool
}

// NewSearchBox creates a new search box.
func NewSearchBox() *SearchBox {
	ti := textinput.New()
	ti.Placeholder = "Type to search..."
	ti.CharLimit = 100

	return &SearchBox{
		input:   ti,
		visible: false,
	}
}

// SetWidth sets the search box width.
func (s *SearchBox) SetWidth(width int) {
	s.width = width
	// textinput width is set via CharLimit and rendering
}

// SetCounts sets the filtered and total counts.
func (s *SearchBox) SetCounts(filtered, total int) {
	s.filteredCnt = filtered
	s.totalCnt = total
}

// Show makes the search box visible and focuses the input.
func (s *SearchBox) Show() tea.Cmd {
	s.visible = true
	s.input.SetValue("")
	return s.input.Focus()
}

// Hide hides the search box and clears the input.
func (s *SearchBox) Hide() {
	s.visible = false
	s.input.SetValue("")
	s.input.Blur()
}

// IsVisible returns whether the search box is visible.
func (s *SearchBox) IsVisible() bool {
	return s.visible
}

// Value returns the current search text.
func (s *SearchBox) Value() string {
	return s.input.Value()
}

// Update handles messages for the search input.
func (s *SearchBox) Update(msg tea.Msg) (*SearchBox, tea.Cmd) {
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

// View renders the search box.
func (s *SearchBox) View() string {
	if !s.visible {
		return ""
	}

	t := styles.CurrentTheme()

	// Calculate dimensions
	borderWidth := s.width - 2 // -2 for ╭ and ╮
	if borderWidth < 10 {
		borderWidth = 10
	}
	contentWidth := borderWidth - 2 // -2 for padding spaces

	// Build the input line with count
	inputView := s.input.View()
	countStr := fmt.Sprintf("%d / %d", s.filteredCnt, s.totalCnt)
	countStyle := t.S().Muted

	// Calculate spacing between input and count
	inputLen := lipgloss.Width(inputView)
	countLen := len(countStr)
	spacing := contentWidth - inputLen - countLen
	if spacing < 1 {
		spacing = 1
	}

	content := inputView + strings.Repeat(" ", spacing) + countStyle.Render(countStr)

	// Render with border
	borderStyle := lipgloss.NewStyle().Foreground(t.BorderFocus)
	titleStyle := t.S().Primary.Bold(true)

	title := "Search"
	titleRendered := titleStyle.Render(title)
	titleVisualLen := lipgloss.Width(titleRendered)

	// Calculate padding for centered title
	remainingSpace := borderWidth - titleVisualLen
	leftPadding := remainingSpace / 2
	rightPadding := remainingSpace - leftPadding
	if leftPadding < 0 {
		leftPadding = 0
	}
	if rightPadding < 0 {
		rightPadding = 0
	}

	topBorder := borderStyle.Render("╭"+strings.Repeat("─", leftPadding)) +
		titleRendered +
		borderStyle.Render(strings.Repeat("─", rightPadding)+"╮")

	bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", borderWidth) + "╯")

	// Pad content to width
	contentLen := lipgloss.Width(content)
	if contentLen < contentWidth {
		content = content + strings.Repeat(" ", contentWidth-contentLen)
	}

	contentLine := borderStyle.Render("│ ") + content + borderStyle.Render(" │")

	return strings.Join([]string{topBorder, contentLine, bottomBorder}, "\n")
}

// Cursor returns the cursor for the text input.
func (s *SearchBox) Cursor() *tea.Cursor {
	if s.visible {
		return s.input.Cursor()
	}
	return nil
}

// IsFocused returns whether the input is focused.
func (s *SearchBox) IsFocused() bool {
	return s.input.Focused()
}
