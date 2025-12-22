# Agent Module

The agent module provides the core AI agent implementation for CDD CLI. It handles communication with language models, manages conversation sessions, executes tools, and streams responses.

## Overview

| Aspect | Details |
|--------|---------|
| Location | `internal/agent/` |
| Files | 4 source files + 2 test files (~1370 lines total) |
| Purpose | AI agent orchestration and conversation management |
| Dependencies | Fantasy (LLM), Tools registry |

## Package Structure

```
internal/agent/
├── agent.go       - Core types: Message, ToolCall, ToolResult, Agent interface
├── loop.go        - DefaultAgent implementation, Send() logic, streaming
├── session.go     - Session and SessionStore for conversation state
├── prompt.go      - Default system prompt definition
├── loop_test.go   - Tests for agent execution
└── session_test.go - Tests for session management
```

## Architecture

```mermaid
graph TB
    subgraph "Agent Module"
        Agent[Agent Interface]
        DefaultAgent[DefaultAgent]
        SessionStore[SessionStore]
        Sessions[(Sessions)]
    end

    subgraph "External Dependencies"
        Fantasy[Fantasy LLM]
        Tools[Tools Registry]
        Model[Language Model]
    end

    subgraph "Consumers"
        TUI[TUI Chat Page]
        CMD[cmd/root.go]
    end

    CMD -->|creates| DefaultAgent
    TUI -->|calls Send| DefaultAgent
    DefaultAgent -->|implements| Agent
    DefaultAgent -->|uses| SessionStore
    SessionStore -->|stores| Sessions
    DefaultAgent -->|streams via| Fantasy
    Fantasy -->|calls| Model
    Fantasy -->|executes| Tools
```

## Data Types

```mermaid
classDiagram
    class Agent {
        <<interface>>
        +Send(ctx, prompt, opts, callbacks) error
        +SetSystemPrompt(prompt)
        +SetTools(tools)
        +History(sessionID) []Message
        +Clear(sessionID)
        +Cancel(sessionID)
        +IsBusy(sessionID) bool
    }

    class DefaultAgent {
        -model LanguageModel
        -systemPrompt string
        -tools []AgentTool
        -workingDir string
        -sessions *SessionStore
        -activeRequests map~string~CancelFunc
        -mu RWMutex
        +New(cfg Config) *DefaultAgent
        +Send(ctx, prompt, opts, callbacks) error
        +Sessions() *SessionStore
    }

    class Message {
        +ID string
        +Role Role
        +Content string
        +ToolCalls []ToolCall
        +ToolResults []ToolResult
        +CreatedAt time.Time
    }

    class ToolCall {
        +ID string
        +Name string
        +Input string
    }

    class ToolResult {
        +ToolCallID string
        +Name string
        +Content string
        +IsError bool
    }

    class StreamCallbacks {
        +OnTextDelta func(string) error
        +OnToolCall func(ToolCall) error
        +OnToolResult func(ToolResult) error
        +OnComplete func() error
        +OnError func(error)
    }

    class Config {
        +Model LanguageModel
        +SystemPrompt string
        +Tools []AgentTool
        +WorkingDir string
    }

    Agent <|.. DefaultAgent
    DefaultAgent --> Message
    Message --> ToolCall
    Message --> ToolResult
    DefaultAgent --> StreamCallbacks
    DefaultAgent --> Config
```

### Role Enum

| Role | Value | Description |
|------|-------|-------------|
| `RoleUser` | `"user"` | User input messages |
| `RoleAssistant` | `"assistant"` | AI responses (text + tool calls) |
| `RoleSystem` | `"system"` | System prompt (handled separately) |
| `RoleTool` | `"tool"` | Tool execution results |

## Agent Execution Flow

The `Send()` method orchestrates the complete request-response cycle:

```mermaid
sequenceDiagram
    participant TUI as TUI/Chat
    participant Agent as DefaultAgent
    participant Session as SessionStore
    participant Fantasy as Fantasy Agent
    participant Model as LLM (Claude/GPT)
    participant Tool as Tool Registry

    TUI->>Agent: Send(ctx, prompt, opts, callbacks)

    Note over Agent: Validate prompt not empty
    Note over Agent: Check session not busy

    Agent->>Agent: Create cancellable context
    Agent->>Session: AddMessage(userMsg)

    Agent->>Fantasy: NewAgent(model, tools)
    Agent->>Fantasy: Stream(ctx, streamOpts)

    loop Streaming Response
        Model-->>Fantasy: Text delta
        Fantasy-->>Agent: OnTextDelta(text)
        Agent-->>TUI: callbacks.OnTextDelta(text)

        alt Tool Call Requested
            Model-->>Fantasy: Tool call
            Fantasy-->>Agent: OnToolCall(tc)
            Agent-->>TUI: callbacks.OnToolCall(tc)
            Fantasy->>Tool: Execute tool
            Tool-->>Fantasy: Result
            Fantasy-->>Agent: OnToolResult(tr)
            Agent->>Session: AddMessage(toolResultMsg)
            Agent-->>TUI: callbacks.OnToolResult(tr)
        end
    end

    Agent->>Session: AddMessage(assistantMsg)
    Agent-->>TUI: callbacks.OnComplete()
```

## Session Management

Sessions maintain conversation state across multiple exchanges.

```mermaid
classDiagram
    class Session {
        +ID string
        +Title string
        +Messages []Message
        +CreatedAt time.Time
        +UpdatedAt time.Time
    }

    class SessionStore {
        -sessions map~string~*Session
        -current string
        -mu RWMutex
        +NewSessionStore() *SessionStore
        +Create(title) *Session
        +Get(id) (*Session, bool)
        +Current() *Session
        +SetCurrent(id) bool
        +List() []*Session
        +Delete(id) bool
        +AddMessage(sessionID, msg) bool
        +GetMessages(sessionID) []Message
        +ClearMessages(sessionID) bool
        +UpdateTitle(sessionID, title) bool
    }

    SessionStore "1" --> "*" Session
    Session "1" --> "*" Message
```

### Session Lifecycle

```mermaid
stateDiagram-v2
    [*] --> NoSession: App starts

    NoSession --> Active: Current() called
    Note right of Active: Auto-creates "New Session"

    Active --> Active: Send() adds messages
    Active --> Active: Tool results added

    Active --> Cleared: Clear() called
    Cleared --> Active: New message sent

    Active --> Deleted: Delete() called
    Deleted --> NoSession: Session removed

    Active --> Switched: SetCurrent(otherId)
    Switched --> Active: Different session active
```

### Thread Safety

All SessionStore operations are protected by `sync.RWMutex`:

| Operation | Lock Type | Notes |
|-----------|-----------|-------|
| `Create()` | Write | Creates session and sets as current |
| `Get()` | Read | Returns session reference |
| `Current()` | Read → Write | Read first, create if needed |
| `AddMessage()` | Write | Appends to session |
| `GetMessages()` | Read | Returns copy of messages |
| `ClearMessages()` | Write | Empties message slice |
| `Delete()` | Write | Removes from map |

## Streaming and Callbacks

The agent uses callbacks for real-time streaming:

```mermaid
flowchart LR
    subgraph "StreamCallbacks"
        OnTextDelta[OnTextDelta]
        OnToolCall[OnToolCall]
        OnToolResult[OnToolResult]
        OnComplete[OnComplete]
        OnError[OnError]
    end

    subgraph "TUI Handlers"
        RenderText[Render text incrementally]
        ShowTool[Show tool execution]
        ShowResult[Display result]
        Finish[Mark complete]
        HandleErr[Show error]
    end

    OnTextDelta --> RenderText
    OnToolCall --> ShowTool
    OnToolResult --> ShowResult
    OnComplete --> Finish
    OnError --> HandleErr
```

**Callback Flow:**

1. **OnTextDelta**: Called for each text chunk from the model
2. **OnToolCall**: Called when the model requests a tool execution
3. **OnToolResult**: Called after a tool completes (success or error)
4. **OnComplete**: Called when the full response is complete
5. **OnError**: Called if an error occurs during streaming

## System Prompt

The agent uses a two-part system prompt for OAuth compatibility:

```mermaid
flowchart TD
    subgraph "System Messages"
        OAuth["Block 1: OAuth Header<br/>'You are Claude Code...'"]
        Main["Block 2: Main Prompt<br/>CDD instructions"]
    end

    subgraph "Why Two Blocks?"
        Reason["Anthropic OAuth API requires<br/>specific header as first block"]
    end

    OAuth --> Main
    Reason -.-> OAuth
```

**Default System Prompt** (`prompt.go`):

```
You are CDD (Context-Driven Development), an AI coding assistant.

You help developers write, understand, and improve code through structured workflows.

When working with code:
1. Read files before modifying them
2. Use appropriate tools for the task
3. Explain your reasoning clearly
4. Ask clarifying questions when requirements are unclear

Available tools allow you to read files, search code, write files, edit code,
and execute shell commands.
```

## Request Cancellation

The agent supports cancelling in-flight requests:

```mermaid
sequenceDiagram
    participant TUI
    participant Agent
    participant Context
    participant Fantasy

    TUI->>Agent: Send(ctx, prompt, ...)
    Agent->>Agent: Create child context with cancel
    Agent->>Agent: Store cancel func in activeRequests
    Agent->>Fantasy: Stream(childCtx, ...)

    Note over Fantasy: Streaming in progress...

    TUI->>Agent: Cancel(sessionID)
    Agent->>Agent: Lookup cancel func
    Agent->>Context: cancel()
    Context-->>Fantasy: Context cancelled
    Fantasy-->>Agent: Return with context error
    Agent->>Agent: Clear from activeRequests
```

**Key Methods:**

| Method | Purpose |
|--------|---------|
| `Cancel(sessionID)` | Cancels ongoing request for session |
| `IsBusy(sessionID)` | Returns true if request in progress |
| `setActiveRequest()` | Internal: stores cancel func |
| `clearActiveRequest()` | Internal: removes cancel func |

## Tool Execution

Tools are executed by Fantasy during the agent loop:

```mermaid
flowchart TD
    A[Model generates tool call] --> B[Fantasy receives ToolCallContent]
    B --> C[Agent.OnToolCall callback]
    C --> D[Fantasy executes tool]
    D --> E{Tool result}

    E -->|Success| F[ToolResultOutputContentText]
    E -->|Error| G[ToolResultOutputContentError]

    F --> H[Agent.OnToolResult callback]
    G --> H
    H --> I[Add tool message to session]
    I --> J[Continue streaming]
```

**Tool Result Types:**

| Type | Description |
|------|-------------|
| `Text` | Successful result with text content |
| `Error` | Failed execution with error message |
| `Media` | Binary content (treated as unsupported) |

## History Building

The agent converts session messages to Fantasy format for context:

```mermaid
flowchart LR
    subgraph "Session Messages"
        UM[User Message]
        AM[Assistant Message]
        TM[Tool Message]
    end

    subgraph "Fantasy Messages"
        FU[fantasy.UserMessage]
        FA[fantasy.AssistantMessage<br/>TextPart + ToolCallPart]
        FT[fantasy.ToolMessage<br/>ToolResultPart]
    end

    UM --> FU
    AM --> FA
    TM --> FT
```

**Important:** The last message (current user input) is excluded from history since it's passed separately as the prompt.

## API Reference

### Agent Interface

| Method | Signature | Description |
|--------|-----------|-------------|
| `Send` | `(ctx, prompt, opts, callbacks) error` | Send message and stream response |
| `SetSystemPrompt` | `(prompt string)` | Update system prompt |
| `SetTools` | `(tools []AgentTool)` | Update available tools |
| `History` | `(sessionID) []Message` | Get conversation history |
| `Clear` | `(sessionID)` | Clear session history |
| `Cancel` | `(sessionID)` | Cancel ongoing request |
| `IsBusy` | `(sessionID) bool` | Check if request in progress |

### DefaultAgent Additional Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `New` | `(cfg Config) *DefaultAgent` | Create new agent |
| `Sessions` | `() *SessionStore` | Get session store |

### SessionStore Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `NewSessionStore` | `() *SessionStore` | Create empty store |
| `Create` | `(title) *Session` | Create and set as current |
| `Get` | `(id) (*Session, bool)` | Get session by ID |
| `Current` | `() *Session` | Get or create current session |
| `SetCurrent` | `(id) bool` | Switch current session |
| `List` | `() []*Session` | Get all sessions |
| `Delete` | `(id) bool` | Remove session |
| `AddMessage` | `(sessionID, msg) bool` | Add message to session |
| `GetMessages` | `(sessionID) []Message` | Get session messages (copy) |
| `ClearMessages` | `(sessionID) bool` | Clear session messages |
| `UpdateTitle` | `(sessionID, title) bool` | Update session title |

### Error Types

| Error | Description |
|-------|-------------|
| `ErrSessionBusy` | Session already processing a request |
| `ErrEmptyPrompt` | Empty prompt provided |

## Design Decisions

1. **Interface-based design**: `Agent` interface allows for alternative implementations (testing, mocking)

2. **Session isolation**: Each session maintains its own message history, enabling multiple conversations

3. **In-memory storage**: Sessions are stored in memory for simplicity; persistence can be added later

4. **Callback-based streaming**: Allows TUI to render responses incrementally without buffering

5. **Two-block system prompt**: Required for Anthropic OAuth API compatibility

6. **Copy on read**: `GetMessages()` returns a copy to prevent external mutation

7. **Context cancellation**: Proper cleanup of in-flight requests using context

8. **Tool context injection**: Working directory and session ID passed via context values

## Integration Points

### With TUI (Chat Page)

```go
// Create agent in cmd/root.go
agent := agent.New(agent.Config{
    Model:        largeModel,
    Tools:        registry.All(),
    SystemPrompt: agent.DefaultSystemPrompt,
})

// Use in chat page
agent.Send(ctx, userInput, agent.SendOptions{}, agent.StreamCallbacks{
    OnTextDelta: func(text string) error {
        // Update UI with streaming text
        return nil
    },
    OnToolCall: func(tc agent.ToolCall) error {
        // Show tool being executed
        return nil
    },
})
```

### With Tools Registry

```go
// Tools are passed at agent creation
registry := tools.DefaultRegistry(workingDir)
agent := agent.New(agent.Config{
    Tools: registry.All(),
})
```

### With Provider/Fantasy

```go
// Model comes from provider builder
builder := provider.NewBuilder(cfg)
largeModel, _, _ := builder.BuildModels(ctx)

agent := agent.New(agent.Config{
    Model: largeModel.Model, // fantasy.LanguageModel
})
```

---

## Related Documentation

- [Provider System](./provider-system.md) - How models are built and configured
- [TUI Wizard](./tui-wizard.md) - Chat page that uses the agent
- [Config Module](./config-module.md) - Configuration that drives agent setup
