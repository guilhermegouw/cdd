package sessions

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/guilhermegouw/cdd/internal/debug"
	"github.com/guilhermegouw/cdd/internal/session"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// ModalStep represents the current step in the modal flow.
type ModalStep int

const (
	// StepList shows the session list.
	StepList ModalStep = iota
	// StepRename shows the rename input.
	StepRename
	// StepDeleteConfirm shows delete confirmation.
	StepDeleteConfirm
	// StepExport shows export options.
	StepExport
)

// Modal is the sessions management modal.
type Modal struct {
	sessionSvc     *session.Service
	sessionList    *SessionList
	preview        *Preview
	renameInput    *RenameInput
	step           ModalStep
	visible        bool
	width          int
	height         int
	deleteTargetID string
	renameTargetID string
	searchQuery    string
}

// New creates a new sessions Modal.
func New(sessionSvc *session.Service) *Modal {
	m := &Modal{
		sessionSvc: sessionSvc,
		step:       StepList,
		visible:    false,
	}

	m.sessionList = NewSessionList(sessionSvc)
	m.preview = NewPreview()
	m.renameInput = NewRenameInput()

	return m
}

// Init initializes the modal.
func (m *Modal) Init() tea.Cmd {
	debug.Log("Modal.Init: initializing modal")
	m.step = StepList
	debug.Log("Modal.Init: calling sessionList.Refresh")
	m.sessionList.Refresh()
	// Set initial preview
	m.preview.SetSession(m.sessionList.Selected())
	debug.Log("Modal.Init: done, sessions count=%d", len(m.sessionList.sessions))
	return nil
}

// Show makes the modal visible.
func (m *Modal) Show() {
	debug.Log("Modal.Show: showing modal")
	m.visible = true
	m.step = StepList
	debug.Log("Modal.Show: calling sessionList.Refresh")
	m.sessionList.Refresh()
	// Set initial preview
	m.preview.SetSession(m.sessionList.Selected())
	debug.Log("Modal.Show: done, sessions count=%d", len(m.sessionList.sessions))
}

// Hide hides the modal.
func (m *Modal) Hide() {
	m.visible = false
	m.renameInput.Reset()
	m.searchQuery = ""
}

// IsVisible returns whether the modal is visible.
func (m *Modal) IsVisible() bool {
	return m.visible
}

// SetSize sets the modal size.
func (m *Modal) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Update sub-component sizes.
	// Modal is now wider to accommodate split view
	innerWidth := min(width-10, 110)
	innerHeight := height - 12

	// Split width: 55% for list, 45% for preview
	listWidth := (innerWidth * 55) / 100
	previewWidth := innerWidth - listWidth - 3 // -3 for divider

	m.sessionList.SetSize(listWidth, innerHeight)
	m.preview.SetSize(previewWidth, innerHeight)
	m.renameInput.SetWidth(innerWidth - 4)
}

// Update handles messages.
func (m *Modal) Update(msg tea.Msg) (*Modal, tea.Cmd) {
	// Handle key events first for Escape.
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "esc" {
			return m.handleEscape()
		}
	}

	// Route to current step handler.
	switch m.step {
	case StepList:
		return m.updateList(msg)
	case StepRename:
		return m.updateRename(msg)
	case StepDeleteConfirm:
		return m.updateDeleteConfirm(msg)
	case StepExport:
		return m.updateExport(msg)
	}

	return m, nil
}

func (m *Modal) handleEscape() (*Modal, tea.Cmd) {
	switch m.step {
	case StepList:
		// Close modal.
		m.Hide()
		return m, util.CmdHandler(ModalClosedMsg{})
	case StepRename, StepDeleteConfirm, StepExport:
		// Go back to list.
		m.step = StepList
		m.renameInput.Reset()
		return m, nil
	}
	return m, nil
}

func (m *Modal) updateList(msg tea.Msg) (*Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case SessionSelectedMsg:
		// Switch to selected session.
		m.Hide()
		return m, tea.Batch(
			util.CmdHandler(ModalClosedMsg{}),
			util.CmdHandler(SwitchSessionMsg{SessionID: msg.SessionID}),
		)

	case RenameSessionMsg:
		m.renameTargetID = msg.SessionID
		m.renameInput.SetValue(msg.CurrentTitle)
		m.renameInput.Focus()
		m.step = StepRename
		return m, nil

	case DeleteSessionMsg:
		m.deleteTargetID = msg.SessionID
		m.step = StepDeleteConfirm
		return m, nil

	case ExportSessionMsg:
		// TODO: Implement export flow
		m.step = StepExport
		return m, nil

	case NewSessionMsg:
		// Create new session and switch to it.
		ctx := context.Background()
		sess, err := m.sessionSvc.Create(ctx, "New Session")
		if err != nil {
			return m, util.ReportError(err)
		}
		m.Hide()
		return m, tea.Batch(
			util.CmdHandler(ModalClosedMsg{}),
			util.CmdHandler(SwitchSessionMsg{SessionID: sess.ID}),
		)

	case GenerateTitleMsg:
		// Request LLM to generate title.
		return m, util.CmdHandler(RequestTitleGenerationMsg{SessionID: msg.SessionID})
	}

	// Update list component.
	var cmd tea.Cmd
	m.sessionList, cmd = m.sessionList.Update(msg)

	// Update preview with selected session
	m.preview.SetSession(m.sessionList.Selected())

	return m, cmd
}

func (m *Modal) updateRename(msg tea.Msg) (*Modal, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			// Submit rename.
			newTitle := m.renameInput.Value()
			if newTitle != "" {
				ctx := context.Background()
				if err := m.sessionSvc.UpdateTitle(ctx, m.renameTargetID, newTitle); err != nil {
					return m, util.ReportError(err)
				}
				m.sessionList.Refresh()
			}
			m.step = StepList
			m.renameInput.Reset()
			return m, util.ReportSuccess("Session renamed")
		}
	}

	// Update rename input.
	var cmd tea.Cmd
	m.renameInput, cmd = m.renameInput.Update(msg)
	return m, cmd
}

func (m *Modal) updateDeleteConfirm(msg tea.Msg) (*Modal, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "y", "Y", "enter":
			// Confirm delete.
			ctx := context.Background()
			if err := m.sessionSvc.Delete(ctx, m.deleteTargetID); err != nil {
				return m, util.ReportError(err)
			}
			m.step = StepList
			m.sessionList.Refresh()
			return m, util.ReportSuccess("Session deleted")
		case "n", "N":
			// Cancel.
			m.step = StepList
			return m, nil
		}
	}
	return m, nil
}

func (m *Modal) updateExport(msg tea.Msg) (*Modal, tea.Cmd) {
	// TODO: Implement export flow
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "enter" || keyMsg.String() == "m" {
			// Export to markdown
			selected := m.sessionList.Selected()
			if selected != nil {
				m.Hide()
				return m, tea.Batch(
					util.CmdHandler(ModalClosedMsg{}),
					util.CmdHandler(ExportMarkdownMsg{SessionID: selected.ID}),
				)
			}
		}
	}
	return m, nil
}

// View renders the modal.
func (m *Modal) View() string {
	if !m.visible {
		return ""
	}

	t := styles.CurrentTheme()

	// Render current step content.
	var content string
	var title string
	var footer string
	var boxWidth int

	switch m.step {
	case StepList:
		title = "Sessions"
		content = m.renderSplitView()
		footer = m.renderListFooter()
		boxWidth = min(m.width-4, 110) // Wider for split view
	case StepRename:
		title = "Rename Session"
		content = m.renameInput.View()
		footer = "[enter] Save  [esc] Cancel"
		boxWidth = min(m.width-4, 60) // Narrower for simple dialogs
	case StepDeleteConfirm:
		title = "Delete Session"
		content = m.renderDeleteConfirm()
		footer = "[y] Yes  [n] No  [esc] Cancel"
		boxWidth = min(m.width-4, 60)
	case StepExport:
		title = "Export Session"
		content = m.renderExportOptions()
		footer = "[m] Markdown  [esc] Cancel"
		boxWidth = min(m.width-4, 60)
	}

	contentWidth := boxWidth - 6

	titleStyle := t.S().Title.
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginBottom(1)

	contentStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Left)

	footerStyle := t.S().Muted.
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginTop(1)

	innerContent := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		contentStyle.Render(content),
		footerStyle.Render(footer),
	)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocus).
		Padding(1, 2).
		Width(boxWidth)

	box := boxStyle.Render(innerContent)

	// Center on screen.
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		box,
	)
}

// renderSplitView renders the session list and preview side by side.
func (m *Modal) renderSplitView() string {
	t := styles.CurrentTheme()

	// Calculate widths
	totalWidth := min(m.width-10, 104) // inner width
	listWidth := (totalWidth * 55) / 100
	previewWidth := totalWidth - listWidth - 3 // -3 for divider spacing

	var parts []string

	// Search box at top spanning full width
	if m.sessionList.IsSearchMode() || m.sessionList.HasSearchText() {
		searchBoxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.BorderFocus).
			Padding(0, 1).
			Width(totalWidth)

		parts = append(parts, searchBoxStyle.Render(m.sessionList.SearchInputView()))
		parts = append(parts, "") // Empty line after search
	}

	// Update preview with current selection
	m.preview.SetSession(m.sessionList.Selected())

	// Get list view (without search box)
	listView := m.sessionList.ViewList()
	previewView := m.preview.View()

	// Count lines to make divider full height
	listLines := strings.Count(listView, "\n") + 1
	previewLines := strings.Count(previewView, "\n") + 1
	maxLines := max(listLines, previewLines)

	// Create full-height vertical divider
	dividerStyle := t.S().Muted
	var dividerLines []string
	for i := 0; i < maxLines; i++ {
		dividerLines = append(dividerLines, "│")
	}
	divider := dividerStyle.Render(strings.Join(dividerLines, "\n"))

	// Ensure consistent widths
	listStyle := lipgloss.NewStyle().Width(listWidth)
	previewStyle := lipgloss.NewStyle().Width(previewWidth)

	// Join list and preview horizontally with divider
	splitView := lipgloss.JoinHorizontal(lipgloss.Top,
		listStyle.Render(listView),
		" "+divider+" ",
		previewStyle.Render(previewView),
	)

	parts = append(parts, splitView)
	return strings.Join(parts, "\n")
}

func (m *Modal) renderListFooter() string {
	if m.sessionList.IsSearchMode() {
		return "[enter] Done  [esc] Clear  [↑↓] Navigate"
	}
	return "[/] Search  [enter] Open  [n] New  [r] Rename  [d] Delete  [e] Export  [esc] Close"
}

func (m *Modal) renderDeleteConfirm() string {
	t := styles.CurrentTheme()
	selected := m.sessionList.Selected()
	name := "this session"
	if selected != nil {
		name = selected.Title
		if name == "" || name == "New Session" {
			name = fmt.Sprintf("Session %s...", selected.ID[:8])
		}
	}

	var sb strings.Builder
	sb.WriteString(t.S().Text.Render("Are you sure you want to delete "))
	sb.WriteString(t.S().Primary.Bold(true).Render(name))
	sb.WriteString(t.S().Text.Render("?\n\n"))
	sb.WriteString(t.S().Warning.Render("This will permanently delete all messages in this session."))

	return sb.String()
}

func (m *Modal) renderExportOptions() string {
	t := styles.CurrentTheme()

	var sb strings.Builder
	sb.WriteString(t.S().Text.Render("Export session to:\n\n"))
	sb.WriteString(t.S().Primary.Render("  [m] Markdown (.md)\n"))
	sb.WriteString(t.S().Muted.Render("\nFile will be saved to current directory."))

	return sb.String()
}

// Cursor returns the cursor position.
func (m *Modal) Cursor() *tea.Cursor {
	if m.step == StepRename {
		return m.renameInput.Cursor()
	}
	if m.step == StepList && m.sessionList.IsSearchMode() {
		return m.sessionList.Cursor()
	}
	return nil
}
