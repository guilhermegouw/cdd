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
	searchBox      *SearchBox
	hintBar        *HintBar
	listPanel      *BorderedPanel
	step           ModalStep
	visible        bool
	width          int
	height         int
	deleteTargetID string
	renameTargetID string
	totalSessions  int // Total count before filtering
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
	m.searchBox = NewSearchBox()
	m.hintBar = NewHintBar()
	m.listPanel = NewBorderedPanel()

	return m
}

// Init initializes the modal.
func (m *Modal) Init() tea.Cmd {
	debug.Log("Modal.Init: initializing modal")
	m.step = StepList
	debug.Log("Modal.Init: calling sessionList.Refresh")
	m.sessionList.Refresh()
	m.totalSessions = m.sessionList.Count()
	// Set initial preview
	m.preview.SetSession(m.sessionList.Selected())
	debug.Log("Modal.Init: done, sessions count=%d", m.sessionList.Count())
	return nil
}

// Show makes the modal visible.
func (m *Modal) Show() {
	debug.Log("Modal.Show: showing modal")
	m.visible = true
	m.step = StepList
	debug.Log("Modal.Show: calling sessionList.Refresh")
	m.sessionList.Refresh()
	m.totalSessions = m.sessionList.Count()
	// Set initial preview
	m.preview.SetSession(m.sessionList.Selected())
	m.hintBar.SetMode(HintModeNormal)
	debug.Log("Modal.Show: done, sessions count=%d", m.sessionList.Count())
}

// Hide hides the modal.
func (m *Modal) Hide() {
	m.visible = false
	m.renameInput.Reset()
	m.searchBox.Hide()
	m.sessionList.ClearSearch()
}

// IsVisible returns whether the modal is visible.
func (m *Modal) IsVisible() bool {
	return m.visible
}

// SetSize sets the modal size.
func (m *Modal) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Calculate total content width (max 110, centered)
	totalWidth := min(width-4, 110)

	// Height calculation:
	// - Hint bar: 1 line
	// - Search box (when visible): 3 lines
	// - Panels: remaining height
	panelHeight := height - 4 // -2 for hint bar padding, -2 for margins
	if m.searchBox.IsVisible() {
		panelHeight -= 4 // Search box height + spacing
	}

	// Split width: 50% for list, 50% for preview (no divider needed)
	listWidth := totalWidth / 2
	previewWidth := totalWidth - listWidth

	m.sessionList.SetSize(listWidth-4, panelHeight-2) // -4 for border/padding
	m.listPanel.SetSize(listWidth, panelHeight)
	m.preview.SetSize(previewWidth, panelHeight)
	m.searchBox.SetWidth(totalWidth)
	m.hintBar.SetWidth(totalWidth)
	m.renameInput.SetWidth(min(totalWidth-4, 56))
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
		// If search is active, clear it first
		if m.searchBox.IsVisible() {
			m.searchBox.Hide()
			m.sessionList.ClearSearch()
			m.totalSessions = m.sessionList.Count()
			m.hintBar.SetMode(HintModeNormal)
			// Recalculate sizes without search box
			m.SetSize(m.width, m.height)
			return m, nil
		}
		// Close modal.
		m.Hide()
		return m, util.CmdHandler(ModalClosedMsg{})
	case StepRename, StepDeleteConfirm, StepExport:
		// Go back to list.
		m.step = StepList
		m.renameInput.Reset()
		m.hintBar.SetMode(HintModeNormal)
		return m, nil
	}
	return m, nil
}

func (m *Modal) updateList(msg tea.Msg) (*Modal, tea.Cmd) {
	// Handle search box input when visible
	if m.searchBox.IsVisible() {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case keyEnter:
				// Exit search mode but keep filtered results
				m.searchBox.Hide()
				m.sessionList.ExitSearchMode()
				m.hintBar.SetMode(HintModeNormal)
				m.SetSize(m.width, m.height)
				return m, nil
			case "up", "down", "j", "k":
				// Allow navigation while in search
				// Pass to list
			default:
				// Update search box
				var cmd tea.Cmd
				m.searchBox, cmd = m.searchBox.Update(msg)

				// Filter sessions based on search
				searchText := m.searchBox.Value()
				m.sessionList.Search(searchText)
				m.searchBox.SetCounts(m.sessionList.Count(), m.totalSessions)
				m.preview.SetSession(m.sessionList.Selected())
				return m, cmd
			}
		}
	}

	switch msg := msg.(type) {
	case SessionSelectedMsg:
		// Switch to selected session.
		m.Hide()
		return m, tea.Batch(
			util.CmdHandler(ModalClosedMsg{}),
			util.CmdHandler(SwitchSessionMsg(msg)),
		)

	case RenameSessionMsg:
		m.renameTargetID = msg.SessionID
		m.renameInput.SetValue(msg.CurrentTitle)
		m.renameInput.Focus()
		m.step = StepRename
		m.hintBar.SetMode(HintModeRename)
		return m, nil

	case DeleteSessionMsg:
		m.deleteTargetID = msg.SessionID
		m.step = StepDeleteConfirm
		m.hintBar.SetMode(HintModeDelete)
		return m, nil

	case ExportSessionMsg:
		m.step = StepExport
		m.hintBar.SetMode(HintModeExport)
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
		return m, util.CmdHandler(RequestTitleGenerationMsg(msg))
	}

	// Handle '/' to enter search mode
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "/" && !m.searchBox.IsVisible() {
		m.searchBox.SetCounts(m.sessionList.Count(), m.totalSessions)
		m.hintBar.SetMode(HintModeSearch)
		m.SetSize(m.width, m.height) // Recalculate with search box
		return m, m.searchBox.Show()
	}

	// Update list component.
	var cmd tea.Cmd
	m.sessionList, cmd = m.sessionList.Update(msg)

	// Update preview with selected session
	m.preview.SetSession(m.sessionList.Selected())

	return m, cmd
}

func (m *Modal) updateRename(msg tea.Msg) (*Modal, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == keyEnter {
		// Submit rename.
		newTitle := m.renameInput.Value()
		if newTitle != "" {
			ctx := context.Background()
			if err := m.sessionSvc.UpdateTitle(ctx, m.renameTargetID, newTitle); err != nil {
				return m, util.ReportError(err)
			}
			m.sessionList.Refresh()
			m.totalSessions = m.sessionList.Count()
		}
		m.step = StepList
		m.renameInput.Reset()
		m.hintBar.SetMode(HintModeNormal)
		return m, util.ReportSuccess("Session renamed")
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
			m.totalSessions = m.sessionList.Count()
			m.hintBar.SetMode(HintModeNormal)
			return m, util.ReportSuccess("Session deleted")
		case "n", "N":
			// Cancel.
			m.step = StepList
			m.hintBar.SetMode(HintModeNormal)
			return m, nil
		}
	}
	return m, nil
}

func (m *Modal) updateExport(msg tea.Msg) (*Modal, tea.Cmd) {
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

	// For rename/delete/export, use dialog style
	if m.step != StepList {
		return m.renderDialog()
	}

	// Telescope-style layout for list view
	return m.renderTelescopeView()
}

// renderTelescopeView renders the Telescope-style layout.
func (m *Modal) renderTelescopeView() string {
	// Calculate total width
	totalWidth := min(m.width-4, 110)

	// Calculate panel heights
	panelHeight := m.height - 4
	if m.searchBox.IsVisible() {
		panelHeight -= 4
	}

	// Split width: 50% each
	listWidth := totalWidth / 2
	previewWidth := totalWidth - listWidth

	// Update sizes
	m.listPanel.SetSize(listWidth, panelHeight)
	m.preview.SetSize(previewWidth, panelHeight)

	// Render list panel with "Sessions" title
	m.listPanel.SetTitle("Sessions")
	m.listPanel.SetContent(m.sessionList.ViewList())
	listView := m.listPanel.View()

	// Preview already uses BorderedPanel internally
	previewView := m.preview.View()

	// Join panels horizontally
	panels := lipgloss.JoinHorizontal(lipgloss.Top, listView, previewView)

	// Build vertical layout
	var parts []string
	parts = append(parts, panels)

	// Add search box if visible
	if m.searchBox.IsVisible() {
		parts = append(parts, m.searchBox.View())
	}

	// Add hint bar
	parts = append(parts, m.hintBar.View())

	content := strings.Join(parts, "\n")

	// Center on screen
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

// renderDialog renders dialogs for rename/delete/export.
func (m *Modal) renderDialog() string {
	t := styles.CurrentTheme()

	var content string
	var title string
	boxWidth := min(m.width-4, 60)

	switch m.step {
	case StepRename:
		title = "Rename Session"
		content = m.renameInput.View()
	case StepDeleteConfirm:
		title = "Delete Session"
		content = m.renderDeleteConfirm()
	case StepExport:
		title = "Export Session"
		content = m.renderExportOptions()
	}

	contentWidth := boxWidth - 6

	titleStyle := t.S().Title.
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginBottom(1)

	contentStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Left)

	innerContent := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		contentStyle.Render(content),
		m.hintBar.View(),
	)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocus).
		Padding(1, 2).
		Width(boxWidth)

	box := boxStyle.Render(innerContent)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		box,
	)
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
	if m.step == StepList && m.searchBox.IsVisible() {
		return m.searchBox.Cursor()
	}
	return nil
}
