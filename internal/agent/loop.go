package agent

import (
	"context"
	"sync"
	"time"

	"charm.land/fantasy"
	"github.com/google/uuid"
	"github.com/guilhermegouw/cdd/internal/tools"
)

// DefaultAgent implements the Agent interface using Fantasy.
type DefaultAgent struct {
	model        fantasy.LanguageModel
	systemPrompt string
	tools        []fantasy.AgentTool
	workingDir   string
	sessions     *SessionStore

	activeRequests map[string]context.CancelFunc
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
	}
}

// Send sends a prompt and streams the response.
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
	fantasyOpts := []fantasy.AgentOption{
		fantasy.WithSystemPrompt(a.systemPrompt),
	}
	if len(a.tools) > 0 {
		fantasyOpts = append(fantasyOpts, fantasy.WithTools(a.tools...))
	}

	agent := fantasy.NewAgent(a.model, fantasyOpts...)

	// Prepare history
	history := a.buildHistory(sessionID)

	// Stream call options
	streamOpts := fantasy.AgentStreamCall{
		Prompt:   prompt,
		Messages: history,
	}
	if opts.MaxTokens > 0 {
		streamOpts.MaxOutputTokens = &opts.MaxTokens
	}
	if opts.Temperature != nil {
		streamOpts.Temperature = opts.Temperature
	}

	// Track current assistant message
	var currentAssistant *Message
	var assistantContent string

	streamOpts.OnTextDelta = func(id, text string) error {
		if currentAssistant == nil {
			currentAssistant = &Message{
				ID:        uuid.New().String(),
				Role:      RoleAssistant,
				CreatedAt: time.Now(),
			}
		}
		assistantContent += text
		currentAssistant.Content = assistantContent

		if callbacks.OnTextDelta != nil {
			return callbacks.OnTextDelta(text)
		}
		return nil
	}

	streamOpts.OnToolCall = func(tc fantasy.ToolCallContent) error {
		if currentAssistant == nil {
			currentAssistant = &Message{
				ID:        uuid.New().String(),
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

		if callbacks.OnToolCall != nil {
			return callbacks.OnToolCall(toolCall)
		}
		return nil
	}

	streamOpts.OnToolResult = func(result fantasy.ToolResultContent) error {
		tr := ToolResult{
			ToolCallID: result.ToolCallID,
			Name:       result.ToolName,
		}

		// Extract content from result
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
		}

		// Add tool result as separate message
		toolMsg := Message{
			ID:          uuid.New().String(),
			Role:        RoleTool,
			ToolResults: []ToolResult{tr},
			CreatedAt:   time.Now(),
		}
		a.sessions.AddMessage(sessionID, toolMsg)

		if callbacks.OnToolResult != nil {
			return callbacks.OnToolResult(tr)
		}
		return nil
	}

	// Execute the agent
	_, err := agent.Stream(ctx, streamOpts)

	// Save assistant message if we have content
	if currentAssistant != nil && (currentAssistant.Content != "" || len(currentAssistant.ToolCalls) > 0) {
		a.sessions.AddMessage(sessionID, *currentAssistant)
	}

	if err != nil {
		if callbacks.OnError != nil {
			callbacks.OnError(err)
		}
		return err
	}

	if callbacks.OnComplete != nil {
		return callbacks.OnComplete()
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
						Error: NewAgentError(tr.Content),
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
		}
	}

	return history
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
