package models

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/guilhermegouw/cdd/internal/config"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// ConnectionList displays the list of connections.
type ConnectionList struct {
	connManager    *config.ConnectionManager
	cfg            *config.Config
	connections    []config.Connection
	cursor         int
	width          int
	height         int
	activeConnLg   string // Connection ID for large model
	activeConnSm   string // Connection ID for small model
}

// NewConnectionList creates a new ConnectionList.
func NewConnectionList(connManager *config.ConnectionManager, cfg *config.Config) *ConnectionList {
	return &ConnectionList{
		connManager: connManager,
		cfg:         cfg,
		cursor:      0,
	}
}

// Refresh reloads the connections from the manager.
func (l *ConnectionList) Refresh() {
	l.connections = l.connManager.List()
	if l.cursor >= len(l.connections) {
		l.cursor = max(0, len(l.connections)-1)
	}

	// Get active connection IDs.
	if lg := l.connManager.GetActiveConnection(config.SelectedModelTypeLarge); lg != nil {
		l.activeConnLg = lg.ID
	}
	if sm := l.connManager.GetActiveConnection(config.SelectedModelTypeSmall); sm != nil {
		l.activeConnSm = sm.ID
	}
}

// SetSize sets the component size.
func (l *ConnectionList) SetSize(width, height int) {
	l.width = width
	l.height = height
}

// Update handles messages.
func (l *ConnectionList) Update(msg tea.Msg) (*ConnectionList, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if l.cursor > 0 {
				l.cursor--
			}
			return l, nil

		case "down", "j":
			if l.cursor < len(l.connections)-1 {
				l.cursor++
			}
			return l, nil

		case "a":
			return l, util.CmdHandler(StartAddConnectionMsg{})

		case "e":
			if len(l.connections) > 0 {
				return l, util.CmdHandler(EditConnectionMsg{ID: l.connections[l.cursor].ID})
			}
			return l, nil

		case "d":
			if len(l.connections) > 0 {
				return l, util.CmdHandler(DeleteConnectionMsg{ID: l.connections[l.cursor].ID})
			}
			return l, nil

		case "l":
			return l, util.CmdHandler(SelectLargeModelMsg{})

		case "s":
			return l, util.CmdHandler(SelectSmallModelMsg{})

		case "enter":
			if len(l.connections) > 0 {
				return l, util.CmdHandler(ConnectionSelectedMsg{Connection: l.connections[l.cursor]})
			}
			return l, nil
		}
	}

	return l, nil
}

// View renders the connection list.
func (l *ConnectionList) View() string {
	t := styles.CurrentTheme()

	if len(l.connections) == 0 {
		emptyMsg := t.S().Muted.Render("No connections configured.\n\n")
		hint := t.S().Muted.Render("Press [a] to add a new connection.")
		return emptyMsg + hint
	}

	var sb strings.Builder

	for i := range l.connections {
		// Cursor indicator.
		cursor := "  "
		style := t.S().Text
		if i == l.cursor {
			cursor = t.S().Success.Render("> ")
			style = t.S().Text.Bold(true)
		}

		// Build connection line.
		name := style.Render(l.connections[i].Name)

		// Provider info in muted.
		providerInfo := t.S().Muted.Render(fmt.Sprintf(" (%s)", l.connections[i].ProviderID))

		// Active model indicators.
		var indicators []string
		if l.connections[i].ID == l.activeConnLg {
			indicators = append(indicators, t.S().Primary.Render("[L]"))
		}
		if l.connections[i].ID == l.activeConnSm {
			indicators = append(indicators, t.S().Subtitle.Render("[S]"))
		}
		indicatorStr := ""
		if len(indicators) > 0 {
			indicatorStr = " " + strings.Join(indicators, " ")
		}

		sb.WriteString(cursor)
		sb.WriteString(name)
		sb.WriteString(providerInfo)
		sb.WriteString(indicatorStr)
		sb.WriteString("\n")
	}

	// Actions section.
	sb.WriteString("\n")
	sb.WriteString(t.S().Subtitle.Render("Actions"))
	sb.WriteString("\n")
	sb.WriteString(t.S().Muted.Render("  [a] add connection"))
	sb.WriteString("\n")
	sb.WriteString(t.S().Muted.Render("  [e] edit selected"))
	sb.WriteString("\n")
	sb.WriteString(t.S().Muted.Render("  [d] delete selected"))
	sb.WriteString("\n")
	sb.WriteString(t.S().Muted.Render("  [l] set as large model"))
	sb.WriteString("\n")
	sb.WriteString(t.S().Muted.Render("  [s] set as small model"))
	sb.WriteString("\n")
	sb.WriteString(t.S().Muted.Render("  [enter] quick select"))
	sb.WriteString("\n")
	sb.WriteString(t.S().Muted.Render("  [esc] close"))

	return sb.String()
}

// Selected returns the currently selected connection.
func (l *ConnectionList) Selected() *config.Connection {
	if l.cursor >= 0 && l.cursor < len(l.connections) {
		return &l.connections[l.cursor]
	}
	return nil
}
