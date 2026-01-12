// Package sessions provides session management UI components.
package sessions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/guilhermegouw/cdd/internal/debug"
	"github.com/guilhermegouw/cdd/internal/session"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// Common key constants.
const (
	keyEnter = "enter"
)

// defaultSessionTitle is the placeholder title for new sessions.
const defaultSessionTitle = "New Session"

// SessionList displays a list of sessions with navigation.
type SessionList struct {
	sessionSvc  *session.Service
	sessions    []*session.SessionWithPreview
	searchInput textinput.Model
	cursor      int
	width       int
	height      int
	offset      int // Scroll offset
	searchMode  bool
	searchText  string
}

// NewSessionList creates a new session list.
func NewSessionList(svc *session.Service) *SessionList {
	ti := textinput.New()
	ti.Placeholder = "Search sessions..."
	ti.CharLimit = 100

	return &SessionList{
		sessionSvc:  svc,
		searchInput: ti,
		cursor:      0,
		offset:      0,
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
//
//nolint:gocyclo // Complex due to handling many key bindings
func (l *SessionList) Update(msg tea.Msg) (*SessionList, tea.Cmd) {
	// Handle search mode separately
	if l.searchMode {
		return l.updateSearchMode(msg)
	}

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
		case keyEnter:
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
			l.searchInput.SetValue("")
			return l, l.searchInput.Focus()
		}
	}

	return l, nil
}

// updateSearchMode handles input when in search mode.
func (l *SessionList) updateSearchMode(msg tea.Msg) (*SessionList, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			// Exit search mode and show all sessions
			l.searchMode = false
			l.searchText = ""
			l.searchInput.SetValue("")
			l.searchInput.Blur()
			l.Refresh()
			return l, nil
		case keyEnter:
			// Exit search mode but keep filtered results
			l.searchMode = false
			l.searchInput.Blur()
			return l, nil
		case "up", "down":
			// Allow navigation while searching
			l.searchMode = false
			l.searchInput.Blur()
			return l.Update(msg)
		}
	}

	// Update the text input
	var cmd tea.Cmd
	l.searchInput, cmd = l.searchInput.Update(msg)

	// Filter sessions as user types
	newText := l.searchInput.Value()
	if newText != l.searchText {
		l.searchText = newText
		l.Search(newText)
	}

	return l, cmd
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

// View renders the session list (includes search box for standalone use).
func (l *SessionList) View() string {
	return l.ViewList()
}

// ViewList renders just the session list without the search box.
func (l *SessionList) ViewList() string {
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
	if title == "" || title == defaultSessionTitle {
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

// Cursor returns the cursor for the search input when in search mode.
func (l *SessionList) Cursor() *tea.Cursor {
	if l.searchMode {
		return l.searchInput.Cursor()
	}
	return nil
}

// IsSearchMode returns whether the list is in search mode.
func (l *SessionList) IsSearchMode() bool {
	return l.searchMode
}

// HasSearchText returns whether there is active search text.
func (l *SessionList) HasSearchText() bool {
	return l.searchText != ""
}

// SearchInputView returns just the search input view.
func (l *SessionList) SearchInputView() string {
	return l.searchInput.View()
}

// Count returns the number of currently visible sessions.
func (l *SessionList) Count() int {
	return len(l.sessions)
}

// TotalCount returns the total number of sessions (used for search display).
func (l *SessionList) TotalCount() int {
	if l.searchText == "" {
		return len(l.sessions)
	}
	// When searching, we need to get total from a fresh query
	ctx := context.Background()
	all, err := l.sessionSvc.ListWithPreview(ctx)
	if err != nil {
		return len(l.sessions)
	}
	return len(all)
}

// SearchText returns the current search text.
func (l *SessionList) SearchText() string {
	return l.searchText
}

// ExitSearchMode exits search mode while keeping filtered results.
func (l *SessionList) ExitSearchMode() {
	l.searchMode = false
	l.searchInput.Blur()
}

// ClearSearch clears search and returns to showing all sessions.
func (l *SessionList) ClearSearch() {
	l.searchMode = false
	l.searchText = ""
	l.searchInput.SetValue("")
	l.searchInput.Blur()
	l.Refresh()
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
