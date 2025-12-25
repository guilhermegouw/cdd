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
	"github.com/guilhermegouw/cdd/internal/bridge"
	"github.com/guilhermegouw/cdd/internal/debug"
	"github.com/guilhermegouw/cdd/internal/events"
	"github.com/guilhermegouw/cdd/internal/pubsub"
	"github.com/guilhermegouw/cdd/internal/tools"
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

// ModelFactory rebuilds the model with fresh tokens from config.
// This allows swapping the model without creating a new agent, preserving session history.
type ModelFactory func() (fantasy.LanguageModel, error)

// Model is the chat page model.
type Model struct {
	agent        *agent.DefaultAgent
	agentFactory AgentFactory
	modelFactory ModelFactory
	messages     *MessageList
	activity     *ActivityPanel
	todoPanel    *TodoPanel
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
		agent:     ag,
		messages:  NewMessageList(),
		activity:  NewActivityPanel(),
		todoPanel: NewTodoPanel(),
		input:     NewInput(),
		status:    NewStatusBar(),
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

// SetModelFactory sets the factory for rebuilding models with fresh tokens.
func (m *Model) SetModelFactory(factory ModelFactory) {
	m.modelFactory = factory
}

// SetModelName sets the model name to display in the status bar.
func (m *Model) SetModelName(name string) {
	m.status.SetModelName(name)
}

// isAuthError checks if the error is an authentication-related HTTP error.
// Only 401 and 403 indicate token issues. 400 is NOT included because it
// can indicate many things (invalid request format, message history issues, etc).
func isAuthError(err error) bool {
	var providerErr *fantasy.ProviderError
	if !errors.As(err, &providerErr) {
		debug.Auth("error_check", fmt.Sprintf("not a ProviderError: %T - %v", err, err))
		return false
	}
	debug.Auth("error_check", fmt.Sprintf("ProviderError status=%d message=%s", providerErr.StatusCode, providerErr.Message))
	switch providerErr.StatusCode {
	case http.StatusUnauthorized, // 401 - token expired/invalid
		http.StatusForbidden: // 403 - token revoked/no permissions
		debug.Auth("auth_error_detected", fmt.Sprintf("status=%d is auth error, will retry", providerErr.StatusCode))
		return true
	}
	return false
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
//
//nolint:gocyclo // Complex due to handling many message types including mouse events
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

	case tea.MouseClickMsg:
		// Only handle clicks in the messages area
		messagesHeight := m.messagesAreaHeight()
		if msg.Y < messagesHeight && msg.Button == tea.MouseLeft {
			debug.Event("chat", "MouseClick", fmt.Sprintf("x=%d y=%d in messages area", msg.X, msg.Y))
			m.messages.StartSelection(msg.X, msg.Y)
		}
		return m, nil

	case tea.MouseMotionMsg:
		// Handle selection drag with auto-scroll at edges
		messagesHeight := m.messagesAreaHeight()
		if msg.Button == tea.MouseLeft && m.messages.IsSelecting() {
			x, y := msg.X, msg.Y

			// Auto-scroll when dragging near edges
			if y < 0 {
				m.messages.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
				y = 0
			} else if y >= messagesHeight {
				m.messages.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
				y = messagesHeight - 1
			}

			// Clamp x to valid range
			if x < 0 {
				x = 0
			} else if x >= m.width {
				x = m.width - 1
			}

			m.messages.EndSelection(x, y)
			debug.Event("chat", "MouseMotion", fmt.Sprintf("selection updated x=%d y=%d", x, y))
		}
		return m, nil

	case tea.MouseReleaseMsg:
		if msg.Button == tea.MouseLeft && m.messages.IsSelecting() {
			debug.Event("chat", "MouseRelease", fmt.Sprintf("x=%d y=%d", msg.X, msg.Y))
			m.messages.SelectionStop()

			// Copy selection to clipboard if there's a selection
			if m.messages.HasSelection() {
				cmd := m.messages.CopySelection()
				if cmd != nil {
					return m, cmd
				}
			}
		}
		return m, nil

	case SelectionCopiedMsg:
		// Show feedback in status bar briefly
		m.status.SetStatus(StatusReady)
		debug.Event("chat", "SelectionCopied", fmt.Sprintf("copied %d chars", len(msg.Text)))
		return m, nil

	case StreamTextMsg:
		if len(m.messages.messages) > 0 {
			m.messages.UpdateLast(m.messages.messages[len(m.messages.messages)-1].Content + msg.Text)
		}
		return m, nil

	case StreamToolCallMsg:
		m.activity.AddTool(msg.ToolCall.Name, msg.ToolCall.Input)
		return m, nil

	case StreamToolResultMsg:
		m.activity.MarkToolDone(msg.ToolResult.Name)
		return m, nil

	case StreamCompleteMsg:
		m.isStreaming = false
		m.activity.Clear()
		m.status.SetStatus(StatusReady)
		m.input.Enable()
		// Refresh messages from session
		m.messages.SetMessages(m.agent.Sessions().GetMessages(m.sessionID))
		return m, m.input.Focus()

	case StreamErrorMsg:
		m.isStreaming = false
		m.activity.Clear()
		m.status.SetError(msg.Error.Error())
		m.input.Enable()
		return m, m.input.Focus()

	case SpinnerTickMsg:
		var cmd tea.Cmd
		m.activity, cmd = m.activity.Update(msg)
		// Sync spinner frame with todo panel
		m.todoPanel.SetSpinner(m.activity.spinner)
		return m, cmd

	// Bridge messages from pub/sub system
	case bridge.AgentEventMsg:
		return m.handleAgentEvent(msg.Event)

	case bridge.ToolEventMsg:
		return m.handleToolEvent(msg.Event)

	case bridge.AuthEventMsg:
		return m.handleAuthEvent(msg.Event)

	case bridge.TodoEventMsg:
		return m.handleTodoEvent(msg.Event)
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

		// Start activity panel with spinner
		spinnerCmd := m.activity.SetThinking(true)

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
		sendCmd := m.sendMessage(value)
		return m, tea.Batch(spinnerCmd, sendCmd)

	case "ctrl+c":
		if m.isStreaming {
			m.agent.Cancel(m.sessionID)
			m.activity.Clear()
			return m, nil
		}
		return m, tea.Quit

	case "esc":
		if m.isStreaming {
			m.agent.Cancel(m.sessionID)
			m.activity.Clear()
			return m, nil
		}
	}

	var cmds []tea.Cmd

	// Only pass key events to viewport when input is disabled (streaming mode).
	// This prevents vim-style scroll keys (j/k) from interfering with typing.
	if !m.input.IsEnabled() {
		var msgCmd tea.Cmd
		m.messages, msgCmd = m.messages.Update(msg)
		if msgCmd != nil {
			cmds = append(cmds, msgCmd)
		}
	}

	// Input handles typing when enabled
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

		debug.Auth("send_start", fmt.Sprintf("sending prompt length=%d", len(prompt)))
		err := m.agent.Send(ctx, prompt, opts, callbacks)

		// Handle auth errors (400/401/403) by rebuilding model and retrying once.
		if err != nil && isAuthError(err) && m.modelFactory != nil {
			debug.Auth("retry_start", "auth error detected, rebuilding model with fresh tokens")

			// Rebuild model with fresh tokens from config.
			newModel, factoryErr := m.modelFactory()
			if factoryErr != nil {
				debug.Auth("retry_failed", fmt.Sprintf("failed to rebuild model: %v", factoryErr))
				return StreamErrorMsg{Error: fmt.Errorf("session expired, please restart: %w", err)}
			}

			// Swap the model - agent keeps its session history intact.
			m.agent.SetModel(newModel)
			streamedContent = "" // Reset streamed content for retry

			debug.Auth("retry_attempt", "model rebuilt, retrying request")
			err = m.agent.Send(ctx, prompt, opts, callbacks)
			if err != nil {
				debug.Auth("retry_result", fmt.Sprintf("retry failed: %v", err))
			} else {
				debug.Auth("retry_result", "retry succeeded")
			}
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

	// Set component sizes (messages height adjusts dynamically based on input, activity, and todos)
	m.messages.SetSize(m.width, m.messagesAreaHeight())
	m.todoPanel.SetWidth(m.width)
	m.activity.SetWidth(m.width)
	m.input.SetWidth(m.width)
	m.status.SetWidth(m.width)

	// Render components
	messagesView := m.messages.View()
	todoView := m.todoPanel.View()
	activityView := m.activity.View()
	inputView := m.input.View()
	statusView := m.status.View()

	// Separator
	separator := lipgloss.NewStyle().
		Width(m.width).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(t.Border).
		Render("")

	// Build layout - include panels only if they have content
	var parts []string
	parts = append(parts, messagesView)

	// Todo panel appears above activity panel
	if m.todoPanel.IsActive() {
		parts = append(parts, separator, todoView)
	}

	if m.activity.IsActive() {
		parts = append(parts, separator, activityView)
	}

	parts = append(parts, separator, inputView, statusView)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// SetSize sets the chat page size.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// messagesAreaHeight calculates the current height of the messages area.
func (m *Model) messagesAreaHeight() int {
	statusHeight := 1
	inputHeight := m.input.Height()
	separatorHeight := 1

	// Account for todo panel if active (height + separator)
	todoHeight := m.todoPanel.Height()
	if todoHeight > 0 {
		todoHeight++ // Add separator height
	}

	// Account for activity panel if active (height + separator)
	activityHeight := m.activity.Height()
	if activityHeight > 0 {
		activityHeight++ // Add separator height
	}

	h := m.height - statusHeight - inputHeight - separatorHeight - todoHeight - activityHeight
	if h < 1 {
		h = 1
	}
	return h
}

// Cursor returns the cursor position.
func (m *Model) Cursor() *tea.Cursor {
	if !m.isStreaming {
		return m.input.Cursor()
	}
	return nil
}

// handleAgentEvent processes agent events from the pub/sub bridge.
func (m *Model) handleAgentEvent(event pubsub.Event[events.AgentEvent]) (util.Model, tea.Cmd) {
	// Only handle events for our session
	if event.Payload.SessionID != m.sessionID {
		return m, nil
	}

	//nolint:exhaustive // AgentEventCancelled handled same as Complete
	switch event.Payload.Type {
	case events.AgentEventTextDelta:
		// Update the last message (assistant response) with new text
		if len(m.messages.messages) > 0 {
			lastMsg := m.messages.messages[len(m.messages.messages)-1]
			m.messages.UpdateLast(lastMsg.Content + event.Payload.TextDelta)
		}

	case events.AgentEventToolCall:
		if event.Payload.ToolCall != nil {
			m.activity.AddTool(event.Payload.ToolCall.Name, event.Payload.ToolCall.Input)
		}

	case events.AgentEventToolResult:
		if event.Payload.ToolResult != nil {
			if event.Payload.ToolResult.IsError {
				m.activity.MarkToolError(event.Payload.ToolResult.Name)
			} else {
				m.activity.MarkToolDone(event.Payload.ToolResult.Name)
			}
		}

	case events.AgentEventComplete, events.AgentEventCancelled:
		m.isStreaming = false
		m.activity.Clear()
		m.status.SetStatus(StatusReady)
		m.input.Enable()
		// Refresh messages from session to get final state
		m.messages.SetMessages(m.agent.Sessions().GetMessages(m.sessionID))
		return m, m.input.Focus()

	case events.AgentEventError:
		m.isStreaming = false
		m.activity.Clear()
		if event.Payload.Error != nil {
			m.status.SetError(event.Payload.Error.Error())
		} else {
			m.status.SetError("unknown error")
		}
		m.input.Enable()
		return m, m.input.Focus()
	}

	return m, nil
}

// handleToolEvent processes tool events from the pub/sub bridge.
func (m *Model) handleToolEvent(event pubsub.Event[events.ToolEvent]) (util.Model, tea.Cmd) {
	// Only handle events for our session
	if event.Payload.SessionID != m.sessionID {
		return m, nil
	}

	//nolint:exhaustive // ToolEventProgress only used for logging
	switch event.Payload.Type {
	case events.ToolEventStarted:
		debug.Event("chat", "ToolStarted", fmt.Sprintf("tool=%s", event.Payload.ToolName))
		m.activity.AddTool(event.Payload.ToolName, event.Payload.Input)

	case events.ToolEventCompleted:
		debug.Event("chat", "ToolCompleted", fmt.Sprintf("tool=%s duration=%v", event.Payload.ToolName, event.Payload.Duration))
		m.activity.MarkToolDone(event.Payload.ToolName)

	case events.ToolEventFailed:
		debug.Event("chat", "ToolFailed", fmt.Sprintf("tool=%s error=%v", event.Payload.ToolName, event.Payload.Error))
		m.activity.MarkToolError(event.Payload.ToolName)

	case events.ToolEventProgress:
		debug.Event("chat", "ToolProgress", fmt.Sprintf("tool=%s", event.Payload.ToolName))
	}

	return m, nil
}

// handleAuthEvent processes authentication events from the pub/sub bridge.
func (m *Model) handleAuthEvent(event pubsub.Event[events.AuthEvent]) (util.Model, tea.Cmd) {
	//nolint:exhaustive // AuthEventTokenExpired handled same as Expiring
	switch event.Payload.Type {
	case events.AuthEventTokenExpiring, events.AuthEventTokenExpired:
		debug.Auth("token_expiring", fmt.Sprintf("provider=%s expires_at=%v", event.Payload.ProviderID, event.Payload.ExpiresAt))

	case events.AuthEventTokenRefreshed:
		debug.Auth("token_refreshed", fmt.Sprintf("provider=%s new_expires_at=%v", event.Payload.ProviderID, event.Payload.ExpiresAt))

	case events.AuthEventRefreshFailed:
		debug.Auth("refresh_failed", fmt.Sprintf("provider=%s error=%v", event.Payload.ProviderID, event.Payload.Error))
		// Show error in status bar
		if event.Payload.Error != nil {
			m.status.SetError(fmt.Sprintf("Token refresh failed: %v", event.Payload.Error))
		}
	}

	return m, nil
}

// handleTodoEvent processes todo events from the pub/sub bridge.
func (m *Model) handleTodoEvent(event pubsub.Event[events.TodoEvent]) (util.Model, tea.Cmd) {
	// Only handle events for our session
	if event.Payload.SessionID != m.sessionID {
		return m, nil
	}

	// Convert event todos to tools.TodoItem
	todos := make([]tools.TodoItem, len(event.Payload.Todos))
	for i, t := range event.Payload.Todos {
		todos[i] = tools.TodoItem{
			Content:    t.Content,
			ActiveForm: t.ActiveForm,
			Status:     tools.TodoStatus(t.Status),
		}
	}

	m.todoPanel.SetTodos(todos)

	// If there's an in-progress todo and no spinner running, start one
	if m.todoPanel.HasInProgress() && !m.activity.IsActive() {
		return m, m.activity.SetThinking(true)
	}

	return m, nil
}
