package agent

import (
	"context"
	"sync"
	"time"

	"charm.land/fantasy"
	"github.com/google/uuid"

	"github.com/guilhermegouw/cdd/internal/events"
	"github.com/guilhermegouw/cdd/internal/pubsub"
	"github.com/guilhermegouw/cdd/internal/tools"
)

// oauthSystemHeader is required as the first system content block for OAuth authentication.
// This must be sent as a separate block, not concatenated with other prompts.
const oauthSystemHeader = "You are Claude Code, Anthropic's official CLI for Claude."

// DefaultAgent implements the Agent interface using Fantasy.
type DefaultAgent struct { //nolint:govet // fieldalignment: preserving logical field order
	model          fantasy.LanguageModel
	systemPrompt   string
	tools          []fantasy.AgentTool
	workingDir     string
	sessions       *SessionStore
	activeRequests map[string]context.CancelFunc
	hub            *pubsub.Hub
	mu             sync.RWMutex
}

// New creates a new agent with the given configuration.
func New(cfg Config) *DefaultAgent {
	return &DefaultAgent{
		model:          cfg.Model,
		systemPrompt:   cfg.SystemPrompt,
		tools:          cfg.Tools,
		workingDir:     cfg.WorkingDir,
		sessions:       NewSessionStore(),
		activeRequests: make(map[string]context.CancelFunc),
		hub:            cfg.Hub,
	}
}

// Send sends a prompt and streams the response.
//
//nolint:gocyclo // Complex function handling streaming, tools, and history management
func (a *DefaultAgent) Send(ctx context.Context, prompt string, opts SendOptions, callbacks StreamCallbacks) error {
	if prompt == "" {
		return ErrEmptyPrompt
	}

	sessionID := opts.SessionID
	if sessionID == "" {
		session := a.sessions.Current()
		sessionID = session.ID
	}

	// Check if session is busy
	if a.IsBusy(sessionID) {
		return ErrSessionBusy
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	a.setActiveRequest(sessionID, cancel)
	defer func() {
		a.clearActiveRequest(sessionID)
		cancel()
	}()

	// Add context values for tools
	ctx = tools.WithSessionID(ctx, sessionID)
	ctx = tools.WithWorkingDir(ctx, a.workingDir)

	// Add user message to history
	userMsg := Message{
		ID:        uuid.New().String(),
		Role:      RoleUser,
		Content:   prompt,
		CreatedAt: time.Now(),
	}
	a.sessions.AddMessage(sessionID, userMsg)

	// Build Fantasy agent
	// Note: We don't use WithSystemPrompt because OAuth requires the system
	// prompt to be sent as separate content blocks with the OAuth header first.
	fantasyOpts := []fantasy.AgentOption{}
	if len(a.tools) > 0 {
		fantasyOpts = append(fantasyOpts, fantasy.WithTools(a.tools...))
	}

	agent := fantasy.NewAgent(a.model, fantasyOpts...)

	// Prepare history with system messages at the start
	// OAuth requires "You are Claude Code..." as a separate first block
	var messages []fantasy.Message
	messages = append(messages, fantasy.NewSystemMessage(
		oauthSystemHeader, // First block - required for OAuth
		a.systemPrompt,    // Second block - actual system prompt
	))
	messages = append(messages, a.buildHistory(sessionID)...)

	// Stream call options
	streamOpts := fantasy.AgentStreamCall{
		Prompt:   prompt,
		Messages: messages,
	}

	// Set max tokens (Anthropic API requires this)
	maxTokens := opts.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 8192 // Default max tokens
	}
	streamOpts.MaxOutputTokens = &maxTokens
	if opts.Temperature != nil {
		streamOpts.Temperature = opts.Temperature
	}

	// Track current assistant message and tool results
	var currentAssistant *Message
	var assistantContent string
	var pendingToolResults []Message // Collect tool results to save AFTER assistant message

	// Track message ID for events
	var messageID string

	streamOpts.OnTextDelta = func(id, text string) error {
		if currentAssistant == nil {
			messageID = uuid.New().String()
			currentAssistant = &Message{
				ID:        messageID,
				Role:      RoleAssistant,
				CreatedAt: time.Now(),
			}
		}
		assistantContent += text
		currentAssistant.Content = assistantContent

		// Publish text delta event
		if a.hub != nil {
			a.hub.Agent.Publish(pubsub.EventProgress,
				events.NewTextDeltaEvent(sessionID, messageID, text))
		}

		return nil
	}

	streamOpts.OnToolCall = func(tc fantasy.ToolCallContent) error {
		if currentAssistant == nil {
			messageID = uuid.New().String()
			currentAssistant = &Message{
				ID:        messageID,
				Role:      RoleAssistant,
				CreatedAt: time.Now(),
			}
		}

		toolCall := ToolCall{
			ID:    tc.ToolCallID,
			Name:  tc.ToolName,
			Input: tc.Input,
		}
		currentAssistant.ToolCalls = append(currentAssistant.ToolCalls, toolCall)

		// Publish tool call event
		if a.hub != nil {
			a.hub.Agent.Publish(pubsub.EventProgress,
				events.NewToolCallEvent(sessionID, messageID, events.ToolCallInfo{
					ID:    tc.ToolCallID,
					Name:  tc.ToolName,
					Input: tc.Input,
				}))

			// Also publish to Tool broker for tool-specific subscribers
			a.hub.Tool.Publish(pubsub.EventStarted,
				events.NewToolStartedEvent(sessionID, tc.ToolCallID, tc.ToolName, tc.Input))
		}

		return nil
	}

	streamOpts.OnToolResult = func(result fantasy.ToolResultContent) error {
		tr := ToolResult{
			ToolCallID: result.ToolCallID,
			Name:       result.ToolName,
		}

		// Extract content from result
		//nolint:exhaustive // Media type handled by default case
		switch result.Result.GetType() {
		case fantasy.ToolResultContentTypeText:
			if r, ok := fantasy.AsToolResultOutputType[fantasy.ToolResultOutputContentText](result.Result); ok {
				tr.Content = r.Text
			}
		case fantasy.ToolResultContentTypeError:
			if r, ok := fantasy.AsToolResultOutputType[fantasy.ToolResultOutputContentError](result.Result); ok {
				tr.Content = r.Error.Error()
				tr.IsError = true
			}
		default:
			// Handle other types (e.g., Media) - treat as text fallback
			tr.Content = "[Unsupported tool result type]"
		}

		// Collect tool result to save AFTER assistant message (preserves correct order)
		toolMsg := Message{
			ID:          uuid.New().String(),
			Role:        RoleTool,
			ToolResults: []ToolResult{tr},
			CreatedAt:   time.Now(),
		}
		pendingToolResults = append(pendingToolResults, toolMsg)

		// Publish tool result event
		if a.hub != nil {
			a.hub.Agent.Publish(pubsub.EventProgress,
				events.NewToolResultEvent(sessionID, messageID, events.ToolResultInfo{
					ToolCallID: tr.ToolCallID,
					Name:       tr.Name,
					Content:    tr.Content,
					IsError:    tr.IsError,
				}))

			// Publish to Tool broker
			if tr.IsError {
				a.hub.Tool.Publish(pubsub.EventFailed,
					events.NewToolFailedEvent(sessionID, tr.ToolCallID, tr.Name, NewError(tr.Content), 0))
			} else {
				a.hub.Tool.Publish(pubsub.EventCompleted,
					events.NewToolCompletedEvent(sessionID, tr.ToolCallID, tr.Name, tr.Content, 0))
			}
		}

		return nil
	}

	// Execute the agent
	_, err := agent.Stream(ctx, streamOpts)

	// Save assistant message FIRST (before tool results to maintain correct order)
	if currentAssistant != nil && (currentAssistant.Content != "" || len(currentAssistant.ToolCalls) > 0) {
		a.sessions.AddMessage(sessionID, *currentAssistant)
	}

	// Save tool results AFTER assistant message (they reference tool_calls in assistant message)
	for _, toolMsg := range pendingToolResults {
		a.sessions.AddMessage(sessionID, toolMsg)
	}

	if err != nil {
		// Publish error event
		if a.hub != nil {
			a.hub.Agent.Publish(pubsub.EventFailed,
				events.NewErrorEvent(sessionID, messageID, err))
		}
		return err
	}

	// Publish completion event
	if a.hub != nil {
		a.hub.Agent.Publish(pubsub.EventCompleted,
			events.NewCompleteEvent(sessionID, messageID))
	}

	return nil
}

// buildHistory converts session messages to Fantasy messages.
func (a *DefaultAgent) buildHistory(sessionID string) []fantasy.Message {
	messages := a.sessions.GetMessages(sessionID)
	if len(messages) == 0 {
		return nil
	}

	// Don't include the last message (current user input)
	if len(messages) > 0 {
		messages = messages[:len(messages)-1]
	}

	var history []fantasy.Message
	for _, msg := range messages {
		switch msg.Role {
		case RoleUser:
			history = append(history, fantasy.NewUserMessage(msg.Content))

		case RoleAssistant:
			var parts []fantasy.MessagePart
			if msg.Content != "" {
				parts = append(parts, fantasy.TextPart{Text: msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				parts = append(parts, fantasy.ToolCallPart{
					ToolCallID: tc.ID,
					ToolName:   tc.Name,
					Input:      tc.Input,
				})
			}
			if len(parts) > 0 {
				history = append(history, fantasy.Message{
					Role:    fantasy.MessageRoleAssistant,
					Content: parts,
				})
			}

		case RoleTool:
			for _, tr := range msg.ToolResults {
				var output fantasy.ToolResultOutputContent
				if tr.IsError {
					output = fantasy.ToolResultOutputContentError{
						Error: NewError(tr.Content),
					}
				} else {
					output = fantasy.ToolResultOutputContentText{
						Text: tr.Content,
					}
				}
				history = append(history, fantasy.Message{
					Role: fantasy.MessageRoleTool,
					Content: []fantasy.MessagePart{
						fantasy.ToolResultPart{
							ToolCallID: tr.ToolCallID,
							Output:     output,
						},
					},
				})
			}

		case RoleSystem:
			// System messages are handled separately in Send(), skip in history
		}
	}

	return history
}

// SetModel updates the agent's language model.
// This is used to swap in a new model after token refresh without losing session history.
func (a *DefaultAgent) SetModel(model fantasy.LanguageModel) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.model = model
}

// SetSystemPrompt sets the system prompt.
func (a *DefaultAgent) SetSystemPrompt(prompt string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.systemPrompt = prompt
}

// SetTools sets the available tools.
func (a *DefaultAgent) SetTools(toolList []fantasy.AgentTool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tools = toolList
}

// History returns the conversation history for a session.
func (a *DefaultAgent) History(sessionID string) []Message {
	return a.sessions.GetMessages(sessionID)
}

// Clear clears the conversation history for a session.
func (a *DefaultAgent) Clear(sessionID string) {
	a.sessions.ClearMessages(sessionID)
}

// Cancel cancels any ongoing request for a session.
func (a *DefaultAgent) Cancel(sessionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if cancel, ok := a.activeRequests[sessionID]; ok {
		cancel()
		delete(a.activeRequests, sessionID)
	}
}

// IsBusy returns true if the agent is processing a request for the session.
func (a *DefaultAgent) IsBusy(sessionID string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	_, ok := a.activeRequests[sessionID]
	return ok
}

// Sessions returns the session store.
func (a *DefaultAgent) Sessions() *SessionStore {
	return a.sessions
}

func (a *DefaultAgent) setActiveRequest(sessionID string, cancel context.CancelFunc) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.activeRequests[sessionID] = cancel
}

func (a *DefaultAgent) clearActiveRequest(sessionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.activeRequests, sessionID)
}
