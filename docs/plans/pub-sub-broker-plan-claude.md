# Pub/Sub Broker Implementation Plan

## Overview

This document outlines the implementation plan for introducing a pub/sub broker pattern into the CDD codebase. The goal is to decouple components—particularly the agent streaming layer from the TUI—while maintaining compatibility with Bubble Tea's existing message system.

## Goals

1. **Decouple agent from TUI** - Agent should publish events without knowledge of subscribers
2. **Enable observability** - Tool execution, streaming, and auth events can be monitored
3. **Future-proof architecture** - Support for plugins, multi-session, and testing
4. **Minimal disruption** - Incremental adoption without rewriting existing code

## Non-Goals

- Replacing Bubble Tea's internal message system for TUI components
- Distributed pub/sub (this is in-process only)
- Persistent event storage or replay

---

## Phase 1: Core Broker Implementation

### 1.1 Create the Broker Package

**Location:** `internal/broker/`

**Files to create:**

```
internal/broker/
├── broker.go      # Core broker implementation
├── event.go       # Event types and interfaces
├── topics.go      # Topic constants
└── broker_test.go # Unit tests
```

### 1.2 Core Broker Interface

```go
// internal/broker/broker.go
package broker

import (
    "sync"
    "strings"
)

// Event represents a published event
type Event struct {
    Topic     string
    Payload   any
    Timestamp time.Time
}

// Subscriber is a function that handles events
type Subscriber func(Event)

// Broker manages pub/sub communication
type Broker struct {
    mu          sync.RWMutex
    subscribers map[string][]subscriberEntry
    closed      bool
}

type subscriberEntry struct {
    id       string
    callback Subscriber
}

// New creates a new broker instance
func New() *Broker {
    return &Broker{
        subscribers: make(map[string][]subscriberEntry),
    }
}

// Publish sends an event to all matching subscribers
func (b *Broker) Publish(topic string, payload any) {
    b.mu.RLock()
    defer b.mu.RUnlock()

    if b.closed {
        return
    }

    event := Event{
        Topic:     topic,
        Payload:   payload,
        Timestamp: time.Now(),
    }

    for pattern, subs := range b.subscribers {
        if matchTopic(pattern, topic) {
            for _, sub := range subs {
                go sub.callback(event)
            }
        }
    }
}

// Subscribe registers a subscriber for a topic pattern
// Supports wildcards: "agent.*" matches "agent.stream.text"
func (b *Broker) Subscribe(pattern string, callback Subscriber) string {
    b.mu.Lock()
    defer b.mu.Unlock()

    id := generateID()
    b.subscribers[pattern] = append(b.subscribers[pattern], subscriberEntry{
        id:       id,
        callback: callback,
    })
    return id
}

// Unsubscribe removes a subscriber by ID
func (b *Broker) Unsubscribe(id string) {
    b.mu.Lock()
    defer b.mu.Unlock()

    for pattern, subs := range b.subscribers {
        for i, sub := range subs {
            if sub.id == id {
                b.subscribers[pattern] = append(subs[:i], subs[i+1:]...)
                return
            }
        }
    }
}

// Close shuts down the broker
func (b *Broker) Close() {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.closed = true
    b.subscribers = make(map[string][]subscriberEntry)
}

// matchTopic checks if a topic matches a pattern with wildcard support
func matchTopic(pattern, topic string) bool {
    if pattern == topic {
        return true
    }

    patternParts := strings.Split(pattern, ".")
    topicParts := strings.Split(topic, ".")

    for i, part := range patternParts {
        if part == "*" {
            if i == len(patternParts)-1 {
                return true // Trailing wildcard matches rest
            }
            continue
        }
        if i >= len(topicParts) || part != topicParts[i] {
            return false
        }
    }

    return len(patternParts) == len(topicParts)
}
```

### 1.3 Define Topic Constants

```go
// internal/broker/topics.go
package broker

const (
    // Agent streaming events
    TopicAgentStreamText     = "agent.stream.text"
    TopicAgentStreamToolCall = "agent.stream.toolcall"
    TopicAgentStreamToolResult = "agent.stream.toolresult"
    TopicAgentStreamComplete = "agent.stream.complete"
    TopicAgentStreamError    = "agent.stream.error"

    // Session events
    TopicSessionCreated      = "session.created"
    TopicSessionSwitched     = "session.switched"
    TopicSessionMessageAdded = "session.message.added"
    TopicSessionCleared      = "session.cleared"

    // Tool events
    TopicToolStarted         = "tool.started"
    TopicToolCompleted       = "tool.completed"
    TopicToolFailed          = "tool.failed"

    // Auth events
    TopicAuthTokenRefreshed  = "auth.token.refreshed"
    TopicAuthTokenExpired    = "auth.token.expired"
    TopicAuthError           = "auth.error"

    // Config events
    TopicConfigSaved         = "config.saved"
    TopicConfigLoaded        = "config.loaded"
)
```

### 1.4 Define Event Payloads

```go
// internal/broker/event.go
package broker

// StreamTextPayload for text delta events
type StreamTextPayload struct {
    SessionID string
    Text      string
}

// StreamToolCallPayload for tool invocation events
type StreamToolCallPayload struct {
    SessionID string
    ToolName  string
    ToolID    string
    Arguments string
}

// StreamToolResultPayload for tool result events
type StreamToolResultPayload struct {
    SessionID string
    ToolID    string
    Result    string
    IsError   bool
}

// StreamCompletePayload for completion events
type StreamCompletePayload struct {
    SessionID    string
    InputTokens  int
    OutputTokens int
}

// StreamErrorPayload for error events
type StreamErrorPayload struct {
    SessionID string
    Error     error
}

// SessionMessagePayload for session message events
type SessionMessagePayload struct {
    SessionID string
    Role      string
    Content   string
}

// ToolEventPayload for tool lifecycle events
type ToolEventPayload struct {
    SessionID string
    ToolName  string
    ToolID    string
    Arguments string
    Result    string
    Error     error
    Duration  time.Duration
}

// AuthTokenPayload for auth events
type AuthTokenPayload struct {
    ProviderID string
}
```

### 1.5 Write Tests

```go
// internal/broker/broker_test.go
package broker

import (
    "sync"
    "testing"
    "time"
)

func TestBroker_PublishSubscribe(t *testing.T) {
    b := New()
    defer b.Close()

    var received Event
    var wg sync.WaitGroup
    wg.Add(1)

    b.Subscribe("test.topic", func(e Event) {
        received = e
        wg.Done()
    })

    b.Publish("test.topic", "hello")

    wg.Wait()

    if received.Payload != "hello" {
        t.Errorf("expected 'hello', got %v", received.Payload)
    }
}

func TestBroker_WildcardSubscribe(t *testing.T) {
    b := New()
    defer b.Close()

    var count int
    var mu sync.Mutex
    var wg sync.WaitGroup
    wg.Add(3)

    b.Subscribe("agent.stream.*", func(e Event) {
        mu.Lock()
        count++
        mu.Unlock()
        wg.Done()
    })

    b.Publish("agent.stream.text", "text")
    b.Publish("agent.stream.complete", "done")
    b.Publish("agent.stream.error", "err")

    wg.Wait()

    if count != 3 {
        t.Errorf("expected 3 events, got %d", count)
    }
}

func TestBroker_Unsubscribe(t *testing.T) {
    b := New()
    defer b.Close()

    called := false
    id := b.Subscribe("test", func(e Event) {
        called = true
    })

    b.Unsubscribe(id)
    b.Publish("test", "data")

    time.Sleep(10 * time.Millisecond)

    if called {
        t.Error("subscriber should not have been called")
    }
}
```

---

## Phase 2: Integrate with Agent Layer

### 2.1 Inject Broker into Agent

**File:** `internal/agent/agent.go`

```go
type DefaultAgent struct {
    model       fantasy.LanguageModel
    tools       []fantasy.AgentTool
    prompt      string
    sessions    *SessionStore
    mu          sync.Mutex
    activeReqs  map[string]context.CancelFunc
    broker      *broker.Broker  // Add broker reference
}

func NewDefaultAgent(model fantasy.LanguageModel, b *broker.Broker) *DefaultAgent {
    return &DefaultAgent{
        model:      model,
        sessions:   NewSessionStore(),
        activeReqs: make(map[string]context.CancelFunc),
        broker:     b,
    }
}
```

### 2.2 Publish Events in Streaming Loop

**File:** `internal/agent/loop.go`

Modify the `Send` method to publish events:

```go
func (a *DefaultAgent) Send(ctx context.Context, prompt string, opts SendOptions, callbacks StreamCallbacks) error {
    // ... existing setup code ...

    // Wrap callbacks to also publish to broker
    wrappedCallbacks := fantasy.AgentCallbackHandler{
        OnTextDelta: func(text string) {
            // Publish to broker
            if a.broker != nil {
                a.broker.Publish(broker.TopicAgentStreamText, broker.StreamTextPayload{
                    SessionID: sessionID,
                    Text:      text,
                })
            }
            // Call original callback
            if callbacks.OnTextDelta != nil {
                callbacks.OnTextDelta(text)
            }
        },
        OnToolCall: func(id, name string, args json.RawMessage) {
            if a.broker != nil {
                a.broker.Publish(broker.TopicAgentStreamToolCall, broker.StreamToolCallPayload{
                    SessionID: sessionID,
                    ToolName:  name,
                    ToolID:    id,
                    Arguments: string(args),
                })
            }
            if callbacks.OnToolCall != nil {
                callbacks.OnToolCall(id, name, args)
            }
        },
        // ... similar for other callbacks
    }

    // ... rest of method ...
}
```

### 2.3 Update Agent Factory in cmd/root.go

```go
// Create a shared broker instance
eventBroker := broker.New()

agentFactory := func(model fantasy.LanguageModel) *agent.DefaultAgent {
    return agent.NewDefaultAgent(model, eventBroker)
}

// Pass broker to TUI for subscribing
if err := tui.Run(ag, agentFactory, modelFactory, eventBroker); err != nil {
    // ...
}
```

---

## Phase 3: Update TUI Subscription

### 3.1 Create Bridge Between Broker and Bubble Tea

**File:** `internal/tui/bridge.go`

```go
package tui

import (
    "github.com/charmbracelet/bubbletea/v2"
    "github.com/yourorg/cdd/internal/broker"
)

// BrokerBridge connects the broker to Bubble Tea's message system
type BrokerBridge struct {
    broker  *broker.Broker
    program *tea.Program
    subIDs  []string
}

func NewBrokerBridge(b *broker.Broker) *BrokerBridge {
    return &BrokerBridge{
        broker: b,
    }
}

func (bb *BrokerBridge) SetProgram(p *tea.Program) {
    bb.program = p
}

func (bb *BrokerBridge) Start() {
    if bb.broker == nil || bb.program == nil {
        return
    }

    // Subscribe to agent stream events and convert to Bubble Tea messages
    id := bb.broker.Subscribe("agent.stream.*", func(e broker.Event) {
        var msg tea.Msg

        switch e.Topic {
        case broker.TopicAgentStreamText:
            payload := e.Payload.(broker.StreamTextPayload)
            msg = chat.StreamTextMsg{Text: payload.Text}
        case broker.TopicAgentStreamToolCall:
            payload := e.Payload.(broker.StreamToolCallPayload)
            msg = chat.StreamToolCallMsg{
                ID:   payload.ToolID,
                Name: payload.ToolName,
                Args: payload.Arguments,
            }
        case broker.TopicAgentStreamToolResult:
            payload := e.Payload.(broker.StreamToolResultPayload)
            msg = chat.StreamToolResultMsg{
                ID:      payload.ToolID,
                Result:  payload.Result,
                IsError: payload.IsError,
            }
        case broker.TopicAgentStreamComplete:
            payload := e.Payload.(broker.StreamCompletePayload)
            msg = chat.StreamCompleteMsg{
                InputTokens:  payload.InputTokens,
                OutputTokens: payload.OutputTokens,
            }
        case broker.TopicAgentStreamError:
            payload := e.Payload.(broker.StreamErrorPayload)
            msg = chat.StreamErrorMsg{Err: payload.Error}
        }

        if msg != nil {
            bb.program.Send(msg)
        }
    })
    bb.subIDs = append(bb.subIDs, id)
}

func (bb *BrokerBridge) Stop() {
    for _, id := range bb.subIDs {
        bb.broker.Unsubscribe(id)
    }
    bb.subIDs = nil
}
```

### 3.2 Update TUI Model

**File:** `internal/tui/tui.go`

```go
type Model struct {
    // ... existing fields ...
    broker *broker.Broker
    bridge *BrokerBridge
}

func Run(ag *agent.DefaultAgent, agentFactory AgentFactory, modelFactory ModelFactory, b *broker.Broker) error {
    bridge := NewBrokerBridge(b)

    m := Model{
        // ... existing initialization ...
        broker: b,
        bridge: bridge,
    }

    p := tea.NewProgram(m, tea.WithMouseCellMotion())

    // Connect bridge to program
    bridge.SetProgram(p)
    bridge.Start()
    defer bridge.Stop()

    _, err := p.Run()
    return err
}
```

### 3.3 Remove Direct program.Send() from Chat

**File:** `internal/tui/page/chat/chat.go`

Remove the `program` field and direct `program.Send()` calls. The bridge now handles this.

```go
// Before (remove this pattern):
func (m *Model) sendMessage() tea.Cmd {
    return func() tea.Msg {
        callbacks := agent.StreamCallbacks{
            OnTextDelta: func(text string) {
                m.program.Send(StreamTextMsg{Text: text})  // Remove
            },
        }
        // ...
    }
}

// After (agent publishes, bridge subscribes):
func (m *Model) sendMessage() tea.Cmd {
    return func() tea.Msg {
        // No callbacks needed - agent publishes to broker
        err := m.agent.Send(ctx, prompt, opts, agent.StreamCallbacks{})
        if err != nil {
            return StreamErrorMsg{Err: err}
        }
        return nil
    }
}
```

---

## Phase 4: Add Tool Event Publishing

### 4.1 Create Tool Event Wrapper

**File:** `internal/tools/publisher.go`

```go
package tools

import (
    "context"
    "time"

    "github.com/yourorg/cdd/internal/broker"
    "github.com/anthropic-ai/go-anthropic/tools"
)

// PublishingTool wraps a tool to publish lifecycle events
type PublishingTool struct {
    tool   tools.Tool
    broker *broker.Broker
}

func NewPublishingTool(t tools.Tool, b *broker.Broker) *PublishingTool {
    return &PublishingTool{tool: t, broker: b}
}

func (pt *PublishingTool) Name() string {
    return pt.tool.Name()
}

func (pt *PublishingTool) Description() string {
    return pt.tool.Description()
}

func (pt *PublishingTool) Schema() any {
    return pt.tool.Schema()
}

func (pt *PublishingTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
    sessionID := GetSessionID(ctx)
    toolID := GetToolID(ctx)

    // Publish started event
    pt.broker.Publish(broker.TopicToolStarted, broker.ToolEventPayload{
        SessionID: sessionID,
        ToolName:  pt.tool.Name(),
        ToolID:    toolID,
        Arguments: string(args),
    })

    start := time.Now()
    result, err := pt.tool.Execute(ctx, args)
    duration := time.Since(start)

    // Publish completion or failure
    if err != nil {
        pt.broker.Publish(broker.TopicToolFailed, broker.ToolEventPayload{
            SessionID: sessionID,
            ToolName:  pt.tool.Name(),
            ToolID:    toolID,
            Error:     err,
            Duration:  duration,
        })
    } else {
        pt.broker.Publish(broker.TopicToolCompleted, broker.ToolEventPayload{
            SessionID: sessionID,
            ToolName:  pt.tool.Name(),
            ToolID:    toolID,
            Result:    result,
            Duration:  duration,
        })
    }

    return result, err
}
```

### 4.2 Update Tool Registry

**File:** `internal/tools/registry.go`

```go
func (r *Registry) WrapWithPublisher(b *broker.Broker) []tools.Tool {
    wrapped := make([]tools.Tool, len(r.tools))
    for i, t := range r.tools {
        wrapped[i] = NewPublishingTool(t, b)
    }
    return wrapped
}
```

---

## Phase 5: Add Auth Event Publishing

### 5.1 Update Config Token Refresh

**File:** `internal/config/config.go`

```go
func (c *Config) RefreshOAuthToken(provider *ProviderConfig, b *broker.Broker) error {
    // ... existing refresh logic ...

    if err != nil {
        if b != nil {
            b.Publish(broker.TopicAuthError, broker.AuthTokenPayload{
                ProviderID: provider.ID,
            })
        }
        return err
    }

    if b != nil {
        b.Publish(broker.TopicAuthTokenRefreshed, broker.AuthTokenPayload{
            ProviderID: provider.ID,
        })
    }

    return nil
}
```

---

## Phase 6: Testing & Observability

### 6.1 Add Debug Subscriber

**File:** `internal/debug/subscriber.go`

```go
package debug

import (
    "github.com/yourorg/cdd/internal/broker"
)

func SubscribeAll(b *broker.Broker, logger *Logger) {
    b.Subscribe("*", func(e broker.Event) {
        logger.Log("event", "topic=%s payload=%+v", e.Topic, e.Payload)
    })
}
```

### 6.2 Add Test Helpers

**File:** `internal/broker/testing.go`

```go
package broker

import "sync"

// TestCollector collects events for testing
type TestCollector struct {
    mu     sync.Mutex
    events []Event
}

func NewTestCollector() *TestCollector {
    return &TestCollector{}
}

func (tc *TestCollector) Collect(e Event) {
    tc.mu.Lock()
    defer tc.mu.Unlock()
    tc.events = append(tc.events, e)
}

func (tc *TestCollector) Events() []Event {
    tc.mu.Lock()
    defer tc.mu.Unlock()
    return append([]Event{}, tc.events...)
}

func (tc *TestCollector) Clear() {
    tc.mu.Lock()
    defer tc.mu.Unlock()
    tc.events = nil
}

func (tc *TestCollector) WaitFor(topic string, timeout time.Duration) (Event, bool) {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        tc.mu.Lock()
        for _, e := range tc.events {
            if e.Topic == topic {
                tc.mu.Unlock()
                return e, true
            }
        }
        tc.mu.Unlock()
        time.Sleep(10 * time.Millisecond)
    }
    return Event{}, false
}
```

---

## Implementation Order

| Step | Description | Files | Effort |
|------|-------------|-------|--------|
| 1 | Create broker package | `internal/broker/*` | Small |
| 2 | Add broker tests | `internal/broker/broker_test.go` | Small |
| 3 | Inject broker into agent | `internal/agent/agent.go` | Small |
| 4 | Publish stream events | `internal/agent/loop.go` | Medium |
| 5 | Create TUI bridge | `internal/tui/bridge.go` | Medium |
| 6 | Update TUI to use bridge | `internal/tui/tui.go` | Medium |
| 7 | Remove direct program.Send | `internal/tui/page/chat/chat.go` | Medium |
| 8 | Add tool publishing wrapper | `internal/tools/publisher.go` | Small |
| 9 | Add auth event publishing | `internal/config/config.go` | Small |
| 10 | Add debug subscriber | `internal/debug/subscriber.go` | Small |
| 11 | Integration testing | Various test files | Medium |

---

## Rollback Strategy

Each phase is designed to be independently reversible:

1. **Phase 1**: Delete `internal/broker/` package
2. **Phase 2**: Remove broker field from agent, restore original callbacks
3. **Phase 3**: Remove bridge, restore direct `program.Send()` calls
4. **Phase 4-5**: Remove wrapper/publishing code, use original tools/config

---

## Success Criteria

1. All existing tests pass
2. Agent has no imports from `internal/tui`
3. Chat page has no direct `program.Send()` calls for streaming
4. New broker tests achieve >90% coverage
5. Debug logging shows all expected events during a chat session
6. No performance regression in streaming latency

---

## Future Enhancements

Once the broker is in place, these become straightforward:

- **Metrics collection**: Subscribe to events for latency/usage tracking
- **Plugin system**: External plugins subscribe to events
- **Multi-session sync**: Broadcast session changes across windows
- **Event replay**: Store events for debugging/replay
- **Rate limiting**: Middleware between publisher and subscribers
