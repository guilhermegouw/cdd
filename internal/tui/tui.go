// Package tui provides the terminal user interface for CDD CLI.
package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/fantasy"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/catwalk/pkg/catwalk"

	"github.com/guilhermegouw/cdd/internal/agent"
	"github.com/guilhermegouw/cdd/internal/debug"
	"github.com/guilhermegouw/cdd/internal/tui/components/welcome"
	"github.com/guilhermegouw/cdd/internal/tui/components/wizard"
	"github.com/guilhermegouw/cdd/internal/tui/page"
	"github.com/guilhermegouw/cdd/internal/tui/page/chat"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// AgentFactory is a function that creates an agent from the current config.
// It's called after the wizard completes to create the agent without restarting.
type AgentFactory func() (*agent.DefaultAgent, error)

// ModelFactory rebuilds the model with fresh tokens from config.
// This allows swapping the model without creating a new agent, preserving session history.
type ModelFactory func() (fantasy.LanguageModel, error)

// Model is the main TUI model.
type Model struct {
	welcome      *welcome.Welcome
	wizard       *wizard.Wizard
	chatPage     *chat.Model
	agent        *agent.DefaultAgent
	agentFactory AgentFactory
	modelFactory ModelFactory
	program      *tea.Program
	currentPage  page.ID
	statusMsg    string
	keyMap       KeyMap
	providers    []catwalk.Provider
	width        int
	height       int
	isFirstRun   bool
	ready        bool
}

// New creates a new TUI model.
func New(providers []catwalk.Provider, isFirstRun bool, ag *agent.DefaultAgent, agentFactory AgentFactory, modelFactory ModelFactory) *Model {
	m := &Model{
		keyMap:       DefaultKeyMap(),
		providers:    providers,
		isFirstRun:   isFirstRun,
		currentPage:  page.Welcome,
		welcome:      welcome.New(),
		agent:        ag,
		agentFactory: agentFactory,
		modelFactory: modelFactory,
	}

	// If we have an agent and it's not first run, go directly to chat.
	if ag != nil && !isFirstRun {
		m.chatPage = chat.New(ag)
		m.chatPage.SetAgentFactory(chat.AgentFactory(agentFactory))
		m.chatPage.SetModelFactory(chat.ModelFactory(modelFactory))
		m.currentPage = page.Chat
	}

	return m
}

// Init initializes the TUI.
func (m *Model) Init() tea.Cmd {
	// If we have an agent and chat page is active, initialize it.
	if m.currentPage == page.Chat && m.chatPage != nil {
		return m.chatPage.Init()
	}

	// For first run or if no agent, show welcome.
	return m.welcome.Init()
}

// Update handles messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Log all incoming messages for debugging
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		debug.Event("tui", "WindowSize", fmt.Sprintf("width=%d height=%d", msg.Width, msg.Height))
		m.handleWindowSize(msg)
		return m, nil
	case tea.KeyMsg:
		debug.Event("tui", "KeyMsg", fmt.Sprintf("key=%q", msg.String()))
		if cmd := m.handleGlobalKeys(msg); cmd != nil {
			return m, cmd
		}
	case tea.MouseWheelMsg:
		debug.Event("tui", "MouseWheel", fmt.Sprintf("button=%v x=%d y=%d", msg.Button, msg.X, msg.Y))
	case tea.MouseClickMsg:
		debug.Event("tui", "MouseClick", fmt.Sprintf("button=%v x=%d y=%d", msg.Button, msg.X, msg.Y))
	case tea.MouseMotionMsg:
		// Don't log motion events - too noisy
	case welcome.StartWizardMsg:
		debug.Event("tui", "StartWizard", "wizard starting")
		return m.handleStartWizard()
	case wizard.CompleteMsg:
		debug.Event("tui", "WizardComplete", "wizard finished")
		// Wizard complete - create agent and transition to chat.
		if m.agent == nil && m.agentFactory != nil {
			ag, err := m.agentFactory()
			if err != nil {
				debug.Error("tui", err, "creating agent after wizard")
				m.statusMsg = fmt.Sprintf("Failed to create agent: %v", err)
				return m, nil
			}
			m.agent = ag
		}

		if m.agent != nil {
			m.chatPage = chat.New(m.agent)
			m.chatPage.SetAgentFactory(chat.AgentFactory(m.agentFactory))
			m.chatPage.SetModelFactory(chat.ModelFactory(m.modelFactory))
			m.chatPage.SetSize(m.width, m.height)
			m.chatPage.SetProgram(m.program)
			m.currentPage = page.Chat
			return m, m.chatPage.Init()
		}
		m.statusMsg = "Configuration saved successfully!"
		return m, nil
	case util.InfoMsg:
		m.statusMsg = msg.Msg
		return m, nil
	case page.ChangeMsg:
		debug.Event("tui", "PageChange", fmt.Sprintf("page=%s", msg.Page))
		m.currentPage = msg.Page
		return m, nil
	default:
		debug.Event("tui", "UnhandledMsg", fmt.Sprintf("type=%T", msg))
	}

	cmd := m.routeToPage(msg)
	return m, cmd
}

func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
	m.ready = true
	m.updateComponentSizes()
}

func (m *Model) handleGlobalKeys(msg tea.KeyMsg) tea.Cmd {
	if msg.String() == "ctrl+c" {
		return tea.Quit
	}
	if msg.String() == "q" && m.canQuit() {
		return tea.Quit
	}
	return nil
}

func (m *Model) canQuit() bool {
	if m.currentPage == page.Welcome {
		return true
	}
	return m.currentPage == page.Wizard && m.wizard != nil && m.wizard.IsComplete()
}

func (m *Model) handleStartWizard() (*Model, tea.Cmd) {
	m.wizard = wizard.NewWizard(m.providers)
	m.currentPage = page.Wizard
	m.updateComponentSizes()
	return m, m.wizard.Init()
}

func (m *Model) routeToPage(msg tea.Msg) tea.Cmd {
	switch m.currentPage {
	case page.Welcome:
		_, cmd := m.welcome.Update(msg)
		return cmd
	case page.Wizard:
		return m.updateWizard(msg)
	case page.Chat:
		return m.updateChat(msg)
	case page.Main:
		return nil
	}
	return nil
}

func (m *Model) updateChat(msg tea.Msg) tea.Cmd {
	if m.chatPage == nil {
		return nil
	}
	_, cmd := m.chatPage.Update(msg)
	return cmd
}

func (m *Model) updateWizard(msg tea.Msg) tea.Cmd {
	if m.wizard == nil {
		return nil
	}
	if m.wizard.IsComplete() {
		if _, ok := msg.(tea.KeyMsg); ok {
			return tea.Quit
		}
	}
	_, cmd := m.wizard.Update(msg)
	return cmd
}

// View renders the TUI.
func (m *Model) View() tea.View {
	t := styles.CurrentTheme()

	var view tea.View
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	// Don't force background color - let terminal use its native background
	// to avoid polluting the terminal state on exit

	if !m.ready {
		view.Content = "Loading..."
		return view
	}

	var content string
	switch m.currentPage {
	case page.Welcome:
		content = m.welcome.View()
	case page.Wizard:
		if m.wizard != nil {
			content = m.wizard.View()
		}
	case page.Chat:
		if m.chatPage != nil {
			content = m.chatPage.View()
		}
	case page.Main:
		content = m.renderMain()
	default:
		content = "Unknown page"
	}

	// Add status message if present.
	if m.statusMsg != "" {
		status := t.S().Info.Render(m.statusMsg)
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", status)
	}

	view.Content = content

	// Set cursor if available.
	switch m.currentPage {
	case page.Wizard:
		if m.wizard != nil {
			view.Cursor = m.wizard.Cursor()
		}
	case page.Chat:
		if m.chatPage != nil {
			view.Cursor = m.chatPage.Cursor()
		}
	case page.Welcome, page.Main:
		// No cursor for these pages
	}

	return view
}

func (m *Model) renderMain() string {
	t := styles.CurrentTheme()
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		t.S().Title.Render("CDD - Ready"),
	)
}

func (m *Model) updateComponentSizes() {
	if m.welcome != nil {
		m.welcome.SetSize(m.width, m.height)
	}
	if m.wizard != nil {
		m.wizard.SetSize(m.width, m.height)
	}
	if m.chatPage != nil {
		m.chatPage.SetSize(m.width, m.height)
	}
}

// Run starts the TUI program.
func Run(providers []catwalk.Provider, isFirstRun bool, ag *agent.DefaultAgent, agentFactory AgentFactory, modelFactory ModelFactory) error {
	// Initialize theme.
	styles.NewManager()

	model := New(providers, isFirstRun, ag, agentFactory, modelFactory)
	// In Bubble Tea v2, AltScreen and MouseMode are set in View()
	p := tea.NewProgram(model)

	// Set the program reference so chat can send stream messages.
	model.program = p
	if model.chatPage != nil {
		model.chatPage.SetProgram(p)
	}

	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}
