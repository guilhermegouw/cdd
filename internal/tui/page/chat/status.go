package chat

import (
	"charm.land/lipgloss/v2"

	"github.com/guilhermegouw/cdd/internal/tui/styles"
)

// Status represents the current chat status.
type Status int

// Status constants for chat state.
const (
	StatusReady Status = iota
	StatusThinking
	StatusError
)

// StatusBar displays the current chat status.
type StatusBar struct {
	modelName string
	errorMsg  string
	width     int
	status    Status
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
		s.errorMsg = ""
	}
}

// SetModelName sets the model name to display.
func (s *StatusBar) SetModelName(name string) {
	s.modelName = name
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

	// Don't set background color - let terminal use its native background
	// to avoid polluting terminal state on exit
	barStyle := lipgloss.NewStyle().
		Width(s.width).
		Padding(0, 1)

	// Left side: model name or error
	var left string
	if s.status == StatusError && s.errorMsg != "" {
		// Truncate long error messages
		errMsg := s.errorMsg
		maxLen := s.width / 2
		if len(errMsg) > maxLen {
			errMsg = errMsg[:maxLen-3] + "..."
		}
		left = t.S().Error.Render("Error: " + errMsg)
	} else if s.modelName != "" {
		left = t.S().Muted.Render(s.modelName)
	}

	// Right side: context-aware shortcuts
	var shortcuts string
	//nolint:exhaustive // StatusReady and StatusError use default case
	switch s.status {
	case StatusThinking:
		shortcuts = "Esc cancel · Ctrl+C quit"
	default:
		shortcuts = "Enter send · Esc cancel · Ctrl+C quit"
	}
	right := t.S().Muted.Render(shortcuts)

	gap := s.width - lipgloss.Width(left) - lipgloss.Width(right) - 4
	if gap < 1 {
		gap = 1
	}

	content := left + lipgloss.NewStyle().Width(gap).Render("") + right

	return barStyle.Render(content)
}
