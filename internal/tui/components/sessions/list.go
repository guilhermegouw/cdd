package sessions

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/guilhermegouw/cdd/internal/debug"
	"github.com/guilhermegouw/cdd/internal/session"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// SessionList displays a list of sessions with navigation.
type SessionList struct {
	sessionSvc *session.Service
	sessions   []*session.SessionWithPreview
	cursor     int
	width      int
	height     int
	offset     int // Scroll offset
	searchMode bool
	searchText string
}

// NewSessionList creates a new session list.
func NewSessionList(svc *session.Service) *SessionList {
	return &SessionList{
		sessionSvc: svc,
		cursor:     0,
		offset:     0,
	}
}

// Refresh reloads the session list from the database.
func (l *SessionList) Refresh() {
	ctx := context.Background()
	debug.Log("SessionList.Refresh: starting refresh")
	if l.sessionSvc == nil {
		debug.Log("SessionList.Refresh: sessionSvc is nil!")
		l.sessions = nil
		return
	}
	sessions, err := l.sessionSvc.ListWithPreview(ctx)
	if err != nil {
		debug.Log("SessionList.Refresh: error loading sessions: %v", err)
		l.sessions = nil
		return
	}
	debug.Log("SessionList.Refresh: loaded %d sessions", len(sessions))
	l.sessions = sessions

	// Reset cursor if out of bounds.
	if l.cursor >= len(l.sessions) {
		l.cursor = max(0, len(l.sessions)-1)
	}
}

// Search filters sessions by keyword.
func (l *SessionList) Search(keyword string) {
	ctx := context.Background()
	l.searchText = keyword

	if keyword == "" {
		l.Refresh()
		return
	}

	sessions, err := l.sessionSvc.SearchWithPreview(ctx, keyword)
	if err != nil {
		l.sessions = nil
		return
	}
	l.sessions = sessions
	l.cursor = 0
	l.offset = 0
}

// SetSize sets the list dimensions.
func (l *SessionList) SetSize(width, height int) {
	l.width = width
	l.height = height
}

// Selected returns the currently selected session.
func (l *SessionList) Selected() *session.SessionWithPreview {
	if l.cursor >= 0 && l.cursor < len(l.sessions) {
		return l.sessions[l.cursor]
	}
	return nil
}

// Update handles messages.
func (l *SessionList) Update(msg tea.Msg) (*SessionList, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if l.cursor > 0 {
				l.cursor--
				l.ensureVisible()
			}
		case "down", "j":
			if l.cursor < len(l.sessions)-1 {
				l.cursor++
				l.ensureVisible()
			}
		case "home", "g":
			l.cursor = 0
			l.offset = 0
		case "end", "G":
			l.cursor = max(0, len(l.sessions)-1)
			l.ensureVisible()
		case "enter":
			if selected := l.Selected(); selected != nil {
				return l, util.CmdHandler(SessionSelectedMsg{SessionID: selected.ID})
			}
		case "n":
			return l, util.CmdHandler(NewSessionMsg{})
		case "r":
			if selected := l.Selected(); selected != nil {
				return l, util.CmdHandler(RenameSessionMsg{
					SessionID:    selected.ID,
					CurrentTitle: selected.Title,
				})
			}
		case "d":
			if selected := l.Selected(); selected != nil {
				return l, util.CmdHandler(DeleteSessionMsg{SessionID: selected.ID})
			}
		case "e":
			if selected := l.Selected(); selected != nil {
				return l, util.CmdHandler(ExportSessionMsg{SessionID: selected.ID})
			}
		case "/":
			l.searchMode = true
		}
	}

	return l, nil
}

func (l *SessionList) ensureVisible() {
	visibleRows := l.visibleRows()
	if l.cursor < l.offset {
		l.offset = l.cursor
	} else if l.cursor >= l.offset+visibleRows {
		l.offset = l.cursor - visibleRows + 1
	}
}

func (l *SessionList) visibleRows() int {
	// Each session takes 3 lines (title + preview + spacing)
	return max(1, (l.height-2)/3)
}

// View renders the session list.
func (l *SessionList) View() string {
	t := styles.CurrentTheme()

	if len(l.sessions) == 0 {
		emptyStyle := t.S().Muted.
			Width(l.width).
			Align(lipgloss.Center).
			Padding(2, 0)
		if l.searchText != "" {
			return emptyStyle.Render("No sessions match your search.")
		}
		return emptyStyle.Render("No sessions yet. Press [n] to create one.")
	}

	var rows []string
	visibleRows := l.visibleRows()
	endIdx := min(l.offset+visibleRows, len(l.sessions))

	for i := l.offset; i < endIdx; i++ {
		sess := l.sessions[i]
		isSelected := i == l.cursor
		rows = append(rows, l.renderSession(sess, isSelected))
	}

	// Add scroll indicators.
	var header string
	if l.offset > 0 {
		header = t.S().Muted.Render(fmt.Sprintf("  ↑ %d more above", l.offset))
	}

	var footer string
	remaining := len(l.sessions) - endIdx
	if remaining > 0 {
		footer = t.S().Muted.Render(fmt.Sprintf("  ↓ %d more below", remaining))
	}

	content := strings.Join(rows, "\n")
	if header != "" {
		content = header + "\n" + content
	}
	if footer != "" {
		content = content + "\n" + footer
	}

	return content
}

func (l *SessionList) renderSession(sess *session.SessionWithPreview, selected bool) string {
	t := styles.CurrentTheme()

	// Title line.
	title := sess.Title
	if title == "" || title == "New Session" {
		title = fmt.Sprintf("Session %s...", sess.ID[:8])
	}

	// Truncate title if too long.
	maxTitleLen := l.width - 20
	if len(title) > maxTitleLen {
		title = title[:maxTitleLen-3] + "..."
	}

	// Message count and time.
	timeStr := formatRelativeTime(sess.UpdatedAt)
	meta := fmt.Sprintf("%d msgs · %s", sess.MessageCount, timeStr)

	// Preview line.
	preview := sess.FirstMessage
	if preview == "" {
		preview = "(no messages)"
	}
	maxPreviewLen := l.width - 6
	if len(preview) > maxPreviewLen {
		preview = preview[:maxPreviewLen-3] + "..."
	}
	// Remove newlines from preview.
	preview = strings.ReplaceAll(preview, "\n", " ")

	// Build the row.
	var sb strings.Builder

	if selected {
		// Selected style.
		sb.WriteString(t.S().Primary.Bold(true).Render("> "))
		sb.WriteString(t.S().Primary.Bold(true).Render(title))
		sb.WriteString("  ")
		sb.WriteString(t.S().Muted.Render(meta))
		sb.WriteString("\n")
		sb.WriteString(t.S().Text.Render("  " + preview))
	} else {
		// Normal style.
		sb.WriteString(t.S().Text.Render("  "))
		sb.WriteString(t.S().Text.Render(title))
		sb.WriteString("  ")
		sb.WriteString(t.S().Muted.Render(meta))
		sb.WriteString("\n")
		sb.WriteString(t.S().Muted.Render("  " + preview))
	}

	return sb.String()
}

// formatRelativeTime formats a time as a relative string.
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 48*time.Hour:
		return "yesterday"
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2")
	}
}
