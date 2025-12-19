// Package chat provides the chat page for CDD CLI.
package chat

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	tea "charm.land/bubbletea/v2"
	"charm.land/fantasy"
	"charm.land/lipgloss/v2"

	"github.com/guilhermegouw/cdd/internal/agent"
	"github.com/guilhermegouw/cdd/internal/debug"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// Stream message types for TUI updates.
type (
	// StreamTextMsg is sent when text is streamed.
	StreamTextMsg struct {
		Text string
	}

	// StreamToolCallMsg is sent when a tool is called.
	StreamToolCallMsg struct {
		ToolCall agent.ToolCall
	}

	// StreamToolResultMsg is sent when a tool completes.
	StreamToolResultMsg struct {
		ToolResult agent.ToolResult
	}

	// StreamCompleteMsg is sent when streaming completes.
	StreamCompleteMsg struct{}

	// StreamErrorMsg is sent when an error occurs.
	StreamErrorMsg struct {
		Error error
	}
)

// AgentFactory creates a new agent (used for rebuilding after token refresh).
type AgentFactory func() (*agent.DefaultAgent, error)

// Model is the chat page model.
type Model struct {
	agent        *agent.DefaultAgent
	agentFactory AgentFactory
	messages     *MessageList
	input        *Input
	status       *StatusBar
	program      *tea.Program
	sessionID    string
	isStreaming  bool
	width        int
	height       int
}

// New creates a new chat page model.
func New(ag *agent.DefaultAgent) *Model {
	return &Model{
		agent:    ag,
		messages: NewMessageList(),
		input:    NewInput(),
		status:   NewStatusBar(),
	}
}

// SetProgram sets the tea.Program for sending messages.
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

// SetAgentFactory sets the factory for creating new agents (used after token refresh).
func (m *Model) SetAgentFactory(factory AgentFactory) {
	m.agentFactory = factory
}

// isUnauthorizedError checks if the error is a 401 Unauthorized response.
func isUnauthorizedError(err error) bool {
	var providerErr *fantasy.ProviderError
	return errors.As(err, &providerErr) && providerErr.StatusCode == http.StatusUnauthorized
}

// Init initializes the chat page.
func (m *Model) Init() tea.Cmd {
	// Get or create a session
	session := m.agent.Sessions().Current()
	m.sessionID = session.ID
	m.messages.SetMessages(session.Messages)

	return m.input.Init()
}

// Update handles messages.
func (m *Model) Update(msg tea.Msg) (util.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		debug.Event("chat", "KeyMsg", fmt.Sprintf("key=%q", msg.String()))
		return m.handleKey(msg)

	case tea.MouseWheelMsg:
		debug.Event("chat", "MouseWheel", fmt.Sprintf("button=%v x=%d y=%d", msg.Button, msg.X, msg.Y))
		// Route mouse wheel events to viewport
		var cmd tea.Cmd
		m.messages, cmd = m.messages.Update(msg)
		debug.Event("chat", "MouseWheel", "routed to viewport")
		return m, cmd

	case StreamTextMsg:
		if len(m.messages.messages) > 0 {
			m.messages.UpdateLast(m.messages.messages[len(m.messages.messages)-1].Content + msg.Text)
		}
		return m, nil

	case StreamToolCallMsg:
		m.status.SetToolName(msg.ToolCall.Name)
		return m, nil

	case StreamToolResultMsg:
		m.status.SetStatus(StatusThinking)
		return m, nil

	case StreamCompleteMsg:
		m.isStreaming = false
		m.status.SetStatus(StatusReady)
		m.input.Enable()
		// Refresh messages from session
		m.messages.SetMessages(m.agent.Sessions().GetMessages(m.sessionID))
		return m, m.input.Focus()

	case StreamErrorMsg:
		m.isStreaming = false
		m.status.SetError(msg.Error.Error())
		m.input.Enable()
		return m, m.input.Focus()
	}

	// Update messages (for viewport scrolling)
	var msgCmd tea.Cmd
	m.messages, msgCmd = m.messages.Update(msg)
	if msgCmd != nil {
		cmds = append(cmds, msgCmd)
	}

	// Update input
	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)
	if inputCmd != nil {
		cmds = append(cmds, inputCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleKey(msg tea.KeyMsg) (util.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.isStreaming {
			return m, nil
		}

		value := m.input.Value()
		if value == "" {
			return m, nil
		}

		// Clear input and start streaming
		m.input.Clear()
		m.input.Disable()
		m.isStreaming = true
		m.status.SetStatus(StatusThinking)

		// Add placeholder for assistant response
		m.messages.AppendMessage(agent.Message{
			Role:    agent.RoleUser,
			Content: value,
		})
		m.messages.AppendMessage(agent.Message{
			Role:    agent.RoleAssistant,
			Content: "",
		})

		// Send to agent
		cmd := m.sendMessage(value)
		return m, cmd

	case "ctrl+c":
		if m.isStreaming {
			m.agent.Cancel(m.sessionID)
			return m, nil
		}
		return m, tea.Quit

	case "esc":
		if m.isStreaming {
			m.agent.Cancel(m.sessionID)
			return m, nil
		}
	}

	// Pass key events to both viewport and input
	var cmds []tea.Cmd

	// Viewport handles scroll keys (up, down, pgup, pgdown, j, k, etc.)
	var msgCmd tea.Cmd
	m.messages, msgCmd = m.messages.Update(msg)
	if msgCmd != nil {
		cmds = append(cmds, msgCmd)
	}

	// Input handles typing
	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)
	if inputCmd != nil {
		cmds = append(cmds, inputCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) sendMessage(prompt string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		var streamedContent string

		callbacks := agent.StreamCallbacks{
			OnTextDelta: func(text string) error {
				streamedContent += text
				if m.program != nil {
					m.program.Send(StreamTextMsg{Text: text})
				}
				return nil
			},
			OnToolCall: func(tc agent.ToolCall) error {
				if m.program != nil {
					m.program.Send(StreamToolCallMsg{ToolCall: tc})
				}
				return nil
			},
			OnToolResult: func(tr agent.ToolResult) error {
				if m.program != nil {
					m.program.Send(StreamToolResultMsg{ToolResult: tr})
				}
				return nil
			},
			OnComplete: func() error {
				if m.program != nil {
					m.program.Send(StreamCompleteMsg{})
				}
				return nil
			},
			OnError: func(err error) {
				if m.program != nil {
					m.program.Send(StreamErrorMsg{Error: err})
				}
			},
		}

		opts := agent.SendOptions{
			SessionID: m.sessionID,
		}

		err := m.agent.Send(ctx, prompt, opts, callbacks)

		// Handle 401 errors by rebuilding agent and retrying once.
		if err != nil && isUnauthorizedError(err) && m.agentFactory != nil {
			debug.Event("chat", "TokenRefresh", "401 error, attempting to rebuild agent")

			// Try to rebuild the agent with fresh tokens.
			newAgent, factoryErr := m.agentFactory()
			if factoryErr != nil {
				debug.Error("chat", factoryErr, "failed to rebuild agent after 401")
				return StreamErrorMsg{Error: fmt.Errorf("session expired, please restart: %w", err)}
			}

			// Update agent reference and retry.
			m.agent = newAgent
			streamedContent = "" // Reset streamed content for retry

			err = m.agent.Send(ctx, prompt, opts, callbacks)
		}

		if err != nil {
			return StreamErrorMsg{Error: err}
		}

		return StreamCompleteMsg{}
	}
}

// View renders the chat page.
func (m *Model) View() string {
	t := styles.CurrentTheme()

	// Calculate heights
	statusHeight := 1
	inputHeight := 3
	messagesHeight := m.height - statusHeight - inputHeight - 2

	// Set component sizes
	m.messages.SetSize(m.width, messagesHeight)
	m.input.SetWidth(m.width)
	m.status.SetWidth(m.width)

	// Render components
	messagesView := m.messages.View()
	inputView := m.input.View()
	statusView := m.status.View()

	// Separator
	separator := lipgloss.NewStyle().
		Width(m.width).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(t.Border).
		Render("")

	return lipgloss.JoinVertical(lipgloss.Left,
		messagesView,
		separator,
		inputView,
		statusView,
	)
}

// SetSize sets the chat page size.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Cursor returns the cursor position.
func (m *Model) Cursor() *tea.Cursor {
	if !m.isStreaming {
		return m.input.Cursor()
	}
	return nil
}
