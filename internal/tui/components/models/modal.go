package models

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/catwalk/pkg/catwalk"

	"github.com/guilhermegouw/cdd/internal/config"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// ModalStep represents the current step in the modal flow.
type ModalStep int

const (
	// StepList shows the connection list.
	StepList ModalStep = iota
	// StepAddProvider shows provider selection for new connection.
	StepAddProvider
	// StepAddForm shows the add connection form.
	StepAddForm
	// StepEdit shows the edit connection form.
	StepEdit
	// StepDeleteConfirm shows delete confirmation.
	StepDeleteConfirm
	// StepSelectConnection shows connection selection for model.
	StepSelectConnection
	// StepSelectModel shows model selection for a connection.
	StepSelectModel
)

// Modal is the models/connections management modal.
type Modal struct {
	cfg             *config.Config
	connManager     *config.ConnectionManager
	connectionList  *ConnectionList
	providerPicker  *ProviderPicker
	connectionForm  *ConnectionForm
	modelPicker     *ModelPicker
	step            ModalStep
	visible         bool
	width           int
	height          int
	selectedTier    config.SelectedModelType
	deleteTargetID  string
	editTargetID    string
	selectedConn    *config.Connection
}

// New creates a new Modal.
func New(cfg *config.Config, providers []catwalk.Provider) *Modal {
	// Ensure providers are set on config so model picker can access them.
	if len(cfg.KnownProviders()) == 0 && len(providers) > 0 {
		cfg.SetKnownProviders(providers)
	}

	connManager := config.NewConnectionManager(cfg)

	m := &Modal{
		cfg:         cfg,
		connManager: connManager,
		step:        StepList,
		visible:     false,
	}

	// Initialize sub-components.
	m.connectionList = NewConnectionList(connManager, cfg)
	m.providerPicker = NewProviderPicker(providers)
	m.connectionForm = NewConnectionForm()
	m.modelPicker = NewModelPicker(cfg)

	return m
}

// Init initializes the modal.
func (m *Modal) Init() tea.Cmd {
	m.step = StepList
	m.connectionList.Refresh()
	return nil
}

// Show makes the modal visible.
func (m *Modal) Show() {
	m.visible = true
	m.step = StepList
	m.connectionList.Refresh()
}

// Hide hides the modal.
func (m *Modal) Hide() {
	m.visible = false
	// Reset form to prevent any lingering state
	m.connectionForm.Reset()
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
	innerWidth := min(width-10, 60)
	innerHeight := height - 10

	m.connectionList.SetSize(innerWidth, innerHeight)
	m.providerPicker.SetSize(innerWidth, innerHeight)
	m.connectionForm.SetSize(innerWidth, innerHeight)
	m.modelPicker.SetSize(innerWidth, innerHeight)
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
	case StepAddProvider:
		return m.updateAddProvider(msg)
	case StepAddForm:
		return m.updateAddForm(msg)
	case StepEdit:
		return m.updateEdit(msg)
	case StepDeleteConfirm:
		return m.updateDeleteConfirm(msg)
	case StepSelectConnection:
		return m.updateSelectConnection(msg)
	case StepSelectModel:
		return m.updateSelectModel(msg)
	}

	return m, nil
}

func (m *Modal) handleEscape() (*Modal, tea.Cmd) {
	switch m.step {
	case StepList:
		// Close modal.
		m.Hide()
		return m, util.CmdHandler(ModalClosedMsg{})
	case StepAddProvider, StepAddForm, StepEdit, StepDeleteConfirm:
		// Go back to list.
		m.step = StepList
		m.connectionList.Refresh()
		return m, nil
	case StepSelectConnection:
		// Go back to list.
		m.step = StepList
		return m, nil
	case StepSelectModel:
		// Go back to connection selection.
		m.step = StepSelectConnection
		return m, nil
	}
	return m, nil
}

func (m *Modal) updateList(msg tea.Msg) (*Modal, tea.Cmd) {
	// Handle list-specific messages.
	switch msg := msg.(type) {
	case StartAddConnectionMsg:
		m.step = StepAddProvider
		m.providerPicker.Reset()
		return m, nil

	case EditConnectionMsg:
		m.editTargetID = msg.ID
		conn := m.connManager.Get(msg.ID)
		if conn != nil {
			m.connectionForm.SetConnection(conn)
			m.step = StepEdit
		}
		return m, nil

	case DeleteConnectionMsg:
		m.deleteTargetID = msg.ID
		m.step = StepDeleteConfirm
		return m, nil

	case SelectLargeModelMsg:
		m.selectedTier = config.SelectedModelTypeLarge
		m.step = StepSelectConnection
		return m, nil

	case SelectSmallModelMsg:
		m.selectedTier = config.SelectedModelTypeSmall
		m.step = StepSelectConnection
		return m, nil

	case ConnectionSelectedMsg:
		// Quick switch - use as large model.
		m.selectedTier = config.SelectedModelTypeLarge
		m.selectedConn = &msg.Connection
		m.modelPicker.SetConnection(&msg.Connection)
		m.step = StepSelectModel
		return m, nil
	}

	// Update list component.
	var cmd tea.Cmd
	m.connectionList, cmd = m.connectionList.Update(msg)
	return m, cmd
}

func (m *Modal) updateAddProvider(msg tea.Msg) (*Modal, tea.Cmd) {
	if psm, ok := msg.(ProviderSelectedMsg); ok {
		m.connectionForm.Reset()
		m.connectionForm.SetProvider(psm.ProviderID, psm.ProviderName, psm.ProviderType)
		m.step = StepAddForm
		return m, m.connectionForm.Focus()
	}

	var cmd tea.Cmd
	m.providerPicker, cmd = m.providerPicker.Update(msg)
	return m, cmd
}

func (m *Modal) updateAddForm(msg tea.Msg) (*Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case FormSubmitMsg:
		// Create the connection.
		conn := config.Connection{
			Name:       msg.Name,
			ProviderID: m.connectionForm.providerID,
			APIKey:     msg.APIKey,
		}
		if err := m.connManager.Add(conn); err != nil {
			return m, util.ReportError(err)
		}
		m.step = StepList
		m.connectionList.Refresh()
		return m, util.ReportSuccess("Connection added successfully")

	case FormCancelMsg:
		m.step = StepList
		return m, nil
	}

	var cmd tea.Cmd
	m.connectionForm, cmd = m.connectionForm.Update(msg)
	return m, cmd
}

func (m *Modal) updateEdit(msg tea.Msg) (*Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case FormSubmitMsg:
		// Update the connection.
		conn := m.connManager.Get(m.editTargetID)
		if conn == nil {
			return m, util.ReportError(nil)
		}
		conn.Name = msg.Name
		conn.APIKey = msg.APIKey
		if err := m.connManager.Update(*conn); err != nil {
			return m, util.ReportError(err)
		}
		m.step = StepList
		m.connectionList.Refresh()
		return m, util.ReportSuccess("Connection updated successfully")

	case FormCancelMsg:
		m.step = StepList
		return m, nil
	}

	var cmd tea.Cmd
	m.connectionForm, cmd = m.connectionForm.Update(msg)
	return m, cmd
}

func (m *Modal) updateDeleteConfirm(msg tea.Msg) (*Modal, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "y", "Y", "enter":
			// Confirm delete.
			if err := m.connManager.Delete(m.deleteTargetID); err != nil {
				return m, util.ReportError(err)
			}
			m.step = StepList
			m.connectionList.Refresh()
			return m, util.ReportSuccess("Connection deleted")
		case "n", "N":
			// Cancel.
			m.step = StepList
			return m, nil
		}
	}
	return m, nil
}

func (m *Modal) updateSelectConnection(msg tea.Msg) (*Modal, tea.Cmd) {
	if csm, ok := msg.(ConnectionSelectedMsg); ok {
		m.selectedConn = &csm.Connection
		m.modelPicker.SetConnection(&csm.Connection)
		m.step = StepSelectModel
		return m, nil
	}

	var cmd tea.Cmd
	m.connectionList, cmd = m.connectionList.Update(msg)
	return m, cmd
}

func (m *Modal) updateSelectModel(msg tea.Msg) (*Modal, tea.Cmd) {
	if msm, ok := msg.(ModelSelectedMsg); ok {
		// Set the active model in config.
		if err := m.connManager.SetActiveModel(m.selectedTier, msm.ConnectionID, msm.ModelID); err != nil {
			return m, util.ReportError(err)
		}

		// Get model name for display.
		modelName := msm.ModelID
		if selected := m.modelPicker.Selected(); selected != nil && selected.Name != "" {
			modelName = selected.Name
		}

		// Close modal properly (resets form state) and notify parent to switch the model.
		m.Hide()
		return m, tea.Batch(
			util.CmdHandler(ModalClosedMsg{}),
			util.CmdHandler(ModelSwitchedMsg{
				Tier:         m.selectedTier,
				ConnectionID: msm.ConnectionID,
				ModelID:      msm.ModelID,
				ModelName:    modelName,
			}),
		)
	}

	var cmd tea.Cmd
	m.modelPicker, cmd = m.modelPicker.Update(msg)
	return m, cmd
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

	switch m.step {
	case StepList:
		title = "Connections"
		content = m.connectionList.View()
	case StepAddProvider:
		title = "Add Connection - Select Provider"
		content = m.providerPicker.View()
	case StepAddForm:
		title = "Add Connection"
		content = m.connectionForm.View()
	case StepEdit:
		title = "Edit Connection"
		content = m.connectionForm.View()
	case StepDeleteConfirm:
		title = "Delete Connection"
		conn := m.connManager.Get(m.deleteTargetID)
		name := "this connection"
		if conn != nil {
			name = conn.Name
		}
		content = m.renderDeleteConfirm(name)
	case StepSelectConnection:
		tierName := "Large"
		if m.selectedTier == config.SelectedModelTypeSmall {
			tierName = "Small"
		}
		title = "Select Connection for " + tierName + " Model"
		content = m.connectionList.View()
	case StepSelectModel:
		title = "Select Model"
		content = m.modelPicker.View()
	}

	// Build modal box.
	boxWidth := min(m.width-4, 60)
	contentWidth := boxWidth - 6 // Account for border and padding

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

func (m *Modal) renderDeleteConfirm(name string) string {
	t := styles.CurrentTheme()

	var sb strings.Builder
	sb.WriteString(t.S().Text.Render("Are you sure you want to delete "))
	sb.WriteString(t.S().Primary.Bold(true).Render(name))
	sb.WriteString(t.S().Text.Render("?\n\n"))
	sb.WriteString(t.S().Muted.Render("[y] Yes  [n] No  [esc] Cancel"))

	return sb.String()
}

// Cursor returns the cursor position.
func (m *Modal) Cursor() *tea.Cursor {
	if m.step == StepAddForm || m.step == StepEdit {
		return m.connectionForm.Cursor()
	}
	return nil
}
