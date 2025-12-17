package chat

import (
	"charm.land/lipgloss/v2"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
)

// Status represents the current chat status.
type Status int

const (
	StatusReady Status = iota
	StatusThinking
	StatusToolRunning
	StatusError
)

// StatusBar displays the current chat status.
type StatusBar struct {
	status    Status
	toolName  string
	errorMsg  string
	width     int
}

// NewStatusBar creates a new status bar.
func NewStatusBar() *StatusBar {
	return &StatusBar{
		status: StatusReady,
	}
}

// SetStatus sets the current status.
func (s *StatusBar) SetStatus(status Status) {
	s.status = status
	if status == StatusReady {
		s.toolName = ""
		s.errorMsg = ""
	}
}

// SetToolName sets the name of the currently running tool.
func (s *StatusBar) SetToolName(name string) {
	s.status = StatusToolRunning
	s.toolName = name
}

// SetError sets an error message.
func (s *StatusBar) SetError(msg string) {
	s.status = StatusError
	s.errorMsg = msg
}

// SetWidth sets the status bar width.
func (s *StatusBar) SetWidth(width int) {
	s.width = width
}

// View renders the status bar.
func (s *StatusBar) View() string {
	t := styles.CurrentTheme()

	var statusText string
	var statusStyle lipgloss.Style

	switch s.status {
	case StatusReady:
		statusText = "Ready"
		statusStyle = t.S().Success
	case StatusThinking:
		statusText = "Thinking..."
		statusStyle = t.S().Info
	case StatusToolRunning:
		statusText = "Running: " + s.toolName
		statusStyle = t.S().Warning
	case StatusError:
		statusText = "Error: " + s.errorMsg
		statusStyle = t.S().Error
	}

	barStyle := lipgloss.NewStyle().
		Width(s.width).
		Padding(0, 1).
		Background(t.BgSubtle)

	help := t.S().Muted.Render("Enter to send • Ctrl+C to quit")

	left := statusStyle.Render(statusText)
	right := help

	gap := s.width - lipgloss.Width(left) - lipgloss.Width(right) - 4
	if gap < 1 {
		gap = 1
	}

	content := left + lipgloss.NewStyle().Width(gap).Render("") + right

	return barStyle.Render(content)
}
