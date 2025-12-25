# Pub/Sub Broker Implementation Plan (Hybrid)

> A hybrid approach combining Go generics type safety with flexible integration patterns.

---

## Table of Contents

1. [Overview](#overview)
2. [Design Principles](#design-principles)
3. [Core Implementation](#core-implementation)
4. [Event Definitions](#event-definitions)
5. [Integration Patterns](#integration-patterns)
6. [Implementation Steps](#implementation-steps)
7. [Testing Strategy](#testing-strategy)
8. [Migration Guide](#migration-guide)

---

## Overview

This plan combines:
- **Type-safe generics** from the Crush approach
- **Context-based lifecycle** for automatic cleanup
- **Channel-based delivery** for idiomatic Go
- **Comprehensive integration patterns** from the Claude approach
- **Centralized bridge** for TUI connectivity

### Architecture Summary

```
┌─────────────────────────────────────────────────────────────────┐
│                         Services Layer                          │
├──────────────┬──────────────┬──────────────┬───────────────────┤
│ AgentService │ ToolService  │ AuthService  │  SessionService   │
│   Broker     │   Broker     │   Broker     │     Broker        │
│ [AgentEvent] │ [ToolEvent]  │ [AuthEvent]  │  [SessionEvent]   │
└──────┬───────┴──────┬───────┴──────┬───────┴────────┬──────────┘
       │              │              │                │
       └──────────────┴──────────────┴────────────────┘
                              │
                    ┌─────────▼─────────┐
                    │  BrokerRegistry   │
                    │  (Debug/Metrics)  │
                    └─────────┬─────────┘
                              │
                    ┌─────────▼─────────┐
                    │    TUI Bridge     │
                    │ Channels → tea.Msg│
                    └─────────┬─────────┘
                              │
                    ┌─────────▼─────────┐
                    │   Bubble Tea UI   │
                    └───────────────────┘
```

---

## Design Principles

| Principle | Implementation |
|-----------|----------------|
| **Type Safety** | Generic `Broker[T]` with typed event payloads |
| **Idiomatic Go** | Channels, context cancellation, standard patterns |
| **Decoupling** | Services publish without knowing subscribers |
| **Observability** | Registry enables debugging and metrics |
| **Simplicity** | ~250 lines core, clear mental model |

---

## Core Implementation

### File Structure

```
internal/pubsub/
├── broker.go         # Generic Broker[T] implementation
├── broker_test.go    # Unit tests
├── event.go          # Base event types
├── registry.go       # Broker registry for debugging
└── registry_test.go  # Registry tests

internal/pubsub/events/
├── agent.go          # Agent event types
├── tool.go           # Tool event types
├── auth.go           # Auth event types
└── session.go        # Session event types
```

### Phase 1: Generic Broker

```go
// internal/pubsub/broker.go
package pubsub

import (
	"context"
	"sync"
	"sync/atomic"
)

// Broker manages typed event subscriptions
type Broker[T any] struct {
	name       string
	subs       map[uint64]chan T
	nextID     uint64
	mu         sync.RWMutex
	closed     atomic.Bool
	bufferSize int
}

// BrokerOption configures a broker
type BrokerOption func(*brokerConfig)

type brokerConfig struct {
	bufferSize int
}

// WithBufferSize sets the channel buffer size (default: 16)
func WithBufferSize(size int) BrokerOption {
	return func(c *brokerConfig) {
		c.bufferSize = size
	}
}

// NewBroker creates a new typed broker
func NewBroker[T any](name string, opts ...BrokerOption) *Broker[T] {
	cfg := &brokerConfig{bufferSize: 16}
	for _, opt := range opts {
		opt(cfg)
	}

	return &Broker[T]{
		name:       name,
		subs:       make(map[uint64]chan T),
		bufferSize: cfg.bufferSize,
	}
}

// Name returns the broker name (for debugging)
func (b *Broker[T]) Name() string {
	return b.name
}

// Subscribe creates a subscription that receives events
// The channel closes when ctx is cancelled or broker is closed
func (b *Broker[T]) Subscribe(ctx context.Context) <-chan T {
	ch := make(chan T, b.bufferSize)

	b.mu.Lock()
	if b.closed.Load() {
		b.mu.Unlock()
		close(ch)
		return ch
	}

	id := b.nextID
	b.nextID++
	b.subs[id] = ch
	b.mu.Unlock()

	// Cleanup on context cancellation
	go func() {
		<-ctx.Done()
		b.unsubscribe(id)
	}()

	return ch
}

// Publish sends an event to all subscribers
// Non-blocking: drops event for slow subscribers
func (b *Broker[T]) Publish(event T) {
	if b.closed.Load() {
		return
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subs {
		select {
		case ch <- event:
		default:
			// Subscriber buffer full, drop event
		}
	}
}

// Close shuts down the broker
func (b *Broker[T]) Close() {
	if !b.closed.CompareAndSwap(false, true) {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for id, ch := range b.subs {
		close(ch)
		delete(b.subs, id)
	}
}

// SubscriberCount returns current subscriber count
func (b *Broker[T]) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs)
}

func (b *Broker[T]) unsubscribe(id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.subs[id]; ok {
		close(ch)
		delete(b.subs, id)
	}
}
```

### Phase 2: Base Event Types

```go
// internal/pubsub/event.go
package pubsub

import "time"

// EventType represents the kind of event
type EventType string

const (
	Created   EventType = "created"
	Updated   EventType = "updated"
	Deleted   EventType = "deleted"
	Started   EventType = "started"
	Completed EventType = "completed"
	Failed    EventType = "failed"
)

// Event wraps a payload with metadata
type Event[T any] struct {
	Type      EventType
	Payload   T
	Timestamp time.Time
}

// NewEvent creates a timestamped event
func NewEvent[T any](eventType EventType, payload T) Event[T] {
	return Event[T]{
		Type:      eventType,
		Payload:   payload,
		Timestamp: time.Now(),
	}
}
```

### Phase 3: Broker Registry

```go
// internal/pubsub/registry.go
package pubsub

import (
	"fmt"
	"sync"
)

// BrokerInfo contains broker metadata
type BrokerInfo struct {
	Name            string
	SubscriberCount int
}

// Registry tracks all brokers for debugging
type Registry struct {
	mu      sync.RWMutex
	brokers map[string]interface{ SubscriberCount() int; Name() string }
}

// NewRegistry creates a broker registry
func NewRegistry() *Registry {
	return &Registry{
		brokers: make(map[string]interface{ SubscriberCount() int; Name() string }),
	}
}

// Register adds a broker to the registry
func (r *Registry) Register(b interface{ SubscriberCount() int; Name() string }) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.brokers[b.Name()] = b
}

// Unregister removes a broker from the registry
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.brokers, name)
}

// Info returns information about all registered brokers
func (r *Registry) Info() []BrokerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info := make([]BrokerInfo, 0, len(r.brokers))
	for _, b := range r.brokers {
		info = append(info, BrokerInfo{
			Name:            b.Name(),
			SubscriberCount: b.SubscriberCount(),
		})
	}
	return info
}

// String returns a debug summary
func (r *Registry) String() string {
	info := r.Info()
	result := fmt.Sprintf("Brokers (%d):\n", len(info))
	for _, b := range info {
		result += fmt.Sprintf("  - %s: %d subscribers\n", b.Name, b.SubscriberCount)
	}
	return result
}
```

---

## Event Definitions

### Agent Events

```go
// internal/pubsub/events/agent.go
package events

import (
	"time"

	"github.com/guilhermegouw/cdd/internal/pubsub"
)

// AgentEventType categorizes agent events
type AgentEventType string

const (
	AgentStreamText       AgentEventType = "stream.text"
	AgentStreamToolCall   AgentEventType = "stream.tool_call"
	AgentStreamToolResult AgentEventType = "stream.tool_result"
	AgentStreamComplete   AgentEventType = "stream.complete"
	AgentStreamError      AgentEventType = "stream.error"
)

// AgentEvent represents an agent lifecycle event
type AgentEvent struct {
	Type      AgentEventType
	SessionID string
	Timestamp time.Time
	Payload   AgentPayload
}

// AgentPayload contains event-specific data
type AgentPayload struct {
	// For StreamText
	Text string

	// For StreamToolCall
	ToolID   string
	ToolName string
	Args     string

	// For StreamToolResult
	Result  string
	IsError bool

	// For StreamComplete
	InputTokens  int
	OutputTokens int

	// For StreamError
	Error error
}

// NewAgentEvent creates a new agent event
func NewAgentEvent(eventType AgentEventType, sessionID string, payload AgentPayload) AgentEvent {
	return AgentEvent{
		Type:      eventType,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Payload:   payload,
	}
}

// AgentBroker is a typed broker for agent events
type AgentBroker = pubsub.Broker[AgentEvent]

// NewAgentBroker creates an agent event broker
func NewAgentBroker() *AgentBroker {
	return pubsub.NewBroker[AgentEvent]("agent")
}
```

### Tool Events

```go
// internal/pubsub/events/tool.go
package events

import (
	"time"

	"github.com/guilhermegouw/cdd/internal/pubsub"
)

// ToolEventType categorizes tool events
type ToolEventType string

const (
	ToolStarted   ToolEventType = "started"
	ToolCompleted ToolEventType = "completed"
	ToolFailed    ToolEventType = "failed"
)

// ToolEvent represents a tool lifecycle event
type ToolEvent struct {
	Type      ToolEventType
	SessionID string
	ToolID    string
	ToolName  string
	Timestamp time.Time
	Payload   ToolPayload
}

// ToolPayload contains tool-specific data
type ToolPayload struct {
	Arguments string
	Result    string
	Error     error
	Duration  time.Duration
}

// NewToolEvent creates a new tool event
func NewToolEvent(eventType ToolEventType, sessionID, toolID, toolName string, payload ToolPayload) ToolEvent {
	return ToolEvent{
		Type:      eventType,
		SessionID: sessionID,
		ToolID:    toolID,
		ToolName:  toolName,
		Timestamp: time.Now(),
		Payload:   payload,
	}
}

// ToolBroker is a typed broker for tool events
type ToolBroker = pubsub.Broker[ToolEvent]

// NewToolBroker creates a tool event broker
func NewToolBroker() *ToolBroker {
	return pubsub.NewBroker[ToolEvent]("tool")
}
```

### Auth Events

```go
// internal/pubsub/events/auth.go
package events

import (
	"time"

	"github.com/guilhermegouw/cdd/internal/pubsub"
)

// AuthEventType categorizes auth events
type AuthEventType string

const (
	AuthTokenRefreshed AuthEventType = "token.refreshed"
	AuthTokenExpired   AuthEventType = "token.expired"
	AuthError          AuthEventType = "error"
)

// AuthEvent represents an auth lifecycle event
type AuthEvent struct {
	Type       AuthEventType
	ProviderID string
	Timestamp  time.Time
	Error      error
}

// NewAuthEvent creates a new auth event
func NewAuthEvent(eventType AuthEventType, providerID string, err error) AuthEvent {
	return AuthEvent{
		Type:       eventType,
		ProviderID: providerID,
		Timestamp:  time.Now(),
		Error:      err,
	}
}

// AuthBroker is a typed broker for auth events
type AuthBroker = pubsub.Broker[AuthEvent]

// NewAuthBroker creates an auth event broker
func NewAuthBroker() *AuthBroker {
	return pubsub.NewBroker[AuthEvent]("auth")
}
```

### Session Events

```go
// internal/pubsub/events/session.go
package events

import (
	"time"

	"github.com/guilhermegouw/cdd/internal/pubsub"
)

// SessionEventType categorizes session events
type SessionEventType string

const (
	SessionCreated      SessionEventType = "created"
	SessionSwitched     SessionEventType = "switched"
	SessionDeleted      SessionEventType = "deleted"
	SessionMessageAdded SessionEventType = "message.added"
	SessionCleared      SessionEventType = "cleared"
)

// SessionEvent represents a session lifecycle event
type SessionEvent struct {
	Type      SessionEventType
	SessionID string
	Timestamp time.Time
	Payload   SessionPayload
}

// SessionPayload contains session-specific data
type SessionPayload struct {
	Title       string
	MessageRole string
	MessageText string
}

// NewSessionEvent creates a new session event
func NewSessionEvent(eventType SessionEventType, sessionID string, payload SessionPayload) SessionEvent {
	return SessionEvent{
		Type:      eventType,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Payload:   payload,
	}
}

// SessionBroker is a typed broker for session events
type SessionBroker = pubsub.Broker[SessionEvent]

// NewSessionBroker creates a session event broker
func NewSessionBroker() *SessionBroker {
	return pubsub.NewBroker[SessionEvent]("session")
}
```

---

## Integration Patterns

### Pattern 1: Event Hub

Centralized container for all brokers:

```go
// internal/pubsub/hub.go
package pubsub

import "github.com/guilhermegouw/cdd/internal/pubsub/events"

// Hub contains all event brokers
type Hub struct {
	Agent    *events.AgentBroker
	Tool     *events.ToolBroker
	Auth     *events.AuthBroker
	Session  *events.SessionBroker
	Registry *Registry
}

// NewHub creates a hub with all brokers
func NewHub() *Hub {
	h := &Hub{
		Agent:    events.NewAgentBroker(),
		Tool:     events.NewToolBroker(),
		Auth:     events.NewAuthBroker(),
		Session:  events.NewSessionBroker(),
		Registry: NewRegistry(),
	}

	// Register all brokers
	h.Registry.Register(h.Agent)
	h.Registry.Register(h.Tool)
	h.Registry.Register(h.Auth)
	h.Registry.Register(h.Session)

	return h
}

// Close shuts down all brokers
func (h *Hub) Close() {
	h.Agent.Close()
	h.Tool.Close()
	h.Auth.Close()
	h.Session.Close()
}
```

### Pattern 2: TUI Bridge

Connects brokers to Bubble Tea:

```go
// internal/tui/bridge/bridge.go
package bridge

import (
	"context"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/guilhermegouw/cdd/internal/pubsub"
	"github.com/guilhermegouw/cdd/internal/pubsub/events"
)

// Bridge connects event brokers to Bubble Tea
type Bridge struct {
	hub     *pubsub.Hub
	program *tea.Program
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewBridge creates a new TUI bridge
func NewBridge(hub *pubsub.Hub) *Bridge {
	ctx, cancel := context.WithCancel(context.Background())
	return &Bridge{
		hub:    hub,
		ctx:    ctx,
		cancel: cancel,
	}
}

// SetProgram sets the Bubble Tea program for message sending
func (b *Bridge) SetProgram(p *tea.Program) {
	b.program = p
}

// Start begins forwarding events to the TUI
func (b *Bridge) Start() {
	if b.program == nil {
		return
	}

	// Subscribe to agent events
	go b.forwardAgent()

	// Subscribe to tool events
	go b.forwardTool()

	// Subscribe to session events
	go b.forwardSession()

	// Subscribe to auth events
	go b.forwardAuth()
}

// Stop halts event forwarding
func (b *Bridge) Stop() {
	b.cancel()
}

func (b *Bridge) forwardAgent() {
	ch := b.hub.Agent.Subscribe(b.ctx)
	for event := range ch {
		b.program.Send(AgentEventMsg{Event: event})
	}
}

func (b *Bridge) forwardTool() {
	ch := b.hub.Tool.Subscribe(b.ctx)
	for event := range ch {
		b.program.Send(ToolEventMsg{Event: event})
	}
}

func (b *Bridge) forwardSession() {
	ch := b.hub.Session.Subscribe(b.ctx)
	for event := range ch {
		b.program.Send(SessionEventMsg{Event: event})
	}
}

func (b *Bridge) forwardAuth() {
	ch := b.hub.Auth.Subscribe(b.ctx)
	for event := range ch {
		b.program.Send(AuthEventMsg{Event: event})
	}
}

// TUI Messages
type AgentEventMsg struct{ Event events.AgentEvent }
type ToolEventMsg struct{ Event events.ToolEvent }
type SessionEventMsg struct{ Event events.SessionEvent }
type AuthEventMsg struct{ Event events.AuthEvent }
```

### Pattern 3: Publishing Tool Wrapper

Wraps tools to publish lifecycle events:

```go
// internal/tools/publisher.go
package tools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/guilhermegouw/cdd/internal/pubsub/events"
)

// PublishingTool wraps a tool to publish events
type PublishingTool struct {
	tool   Tool
	broker *events.ToolBroker
}

// NewPublishingTool creates a publishing wrapper
func NewPublishingTool(t Tool, broker *events.ToolBroker) *PublishingTool {
	return &PublishingTool{tool: t, broker: broker}
}

// Name returns the tool name
func (pt *PublishingTool) Name() string {
	return pt.tool.Name()
}

// Description returns the tool description
func (pt *PublishingTool) Description() string {
	return pt.tool.Description()
}

// Schema returns the tool schema
func (pt *PublishingTool) Schema() any {
	return pt.tool.Schema()
}

// Execute runs the tool and publishes events
func (pt *PublishingTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	sessionID := GetSessionID(ctx)
	toolID := GetToolID(ctx)

	// Publish started
	pt.broker.Publish(events.NewToolEvent(
		events.ToolStarted,
		sessionID,
		toolID,
		pt.tool.Name(),
		events.ToolPayload{Arguments: string(args)},
	))

	start := time.Now()
	result, err := pt.tool.Execute(ctx, args)
	duration := time.Since(start)

	// Publish result
	if err != nil {
		pt.broker.Publish(events.NewToolEvent(
			events.ToolFailed,
			sessionID,
			toolID,
			pt.tool.Name(),
			events.ToolPayload{Error: err, Duration: duration},
		))
	} else {
		pt.broker.Publish(events.NewToolEvent(
			events.ToolCompleted,
			sessionID,
			toolID,
			pt.tool.Name(),
			events.ToolPayload{Result: result, Duration: duration},
		))
	}

	return result, err
}

// WrapTools wraps multiple tools with publishing
func WrapTools(tools []Tool, broker *events.ToolBroker) []Tool {
	wrapped := make([]Tool, len(tools))
	for i, t := range tools {
		wrapped[i] = NewPublishingTool(t, broker)
	}
	return wrapped
}
```

### Pattern 4: Agent Integration

Publishing from the agent streaming loop:

```go
// internal/agent/agent.go (modifications)

type DefaultAgent struct {
	model      fantasy.LanguageModel
	tools      []fantasy.AgentTool
	prompt     string
	sessions   *SessionStore
	mu         sync.Mutex
	activeReqs map[string]context.CancelFunc
	broker     *events.AgentBroker // Add broker
}

func NewDefaultAgent(model fantasy.LanguageModel, broker *events.AgentBroker) *DefaultAgent {
	return &DefaultAgent{
		model:      model,
		sessions:   NewSessionStore(),
		activeReqs: make(map[string]context.CancelFunc),
		broker:     broker,
	}
}

// In loop.go Send method:
func (a *DefaultAgent) Send(ctx context.Context, prompt string, opts SendOptions, callbacks StreamCallbacks) error {
	sessionID := opts.SessionID

	// Create wrapped callbacks that publish events
	wrappedCallbacks := fantasy.AgentCallbackHandler{
		OnTextDelta: func(text string) {
			// Publish to broker
			if a.broker != nil {
				a.broker.Publish(events.NewAgentEvent(
					events.AgentStreamText,
					sessionID,
					events.AgentPayload{Text: text},
				))
			}
			// Call original
			if callbacks.OnTextDelta != nil {
				callbacks.OnTextDelta(text)
			}
		},
		OnToolCall: func(id, name string, args json.RawMessage) {
			if a.broker != nil {
				a.broker.Publish(events.NewAgentEvent(
					events.AgentStreamToolCall,
					sessionID,
					events.AgentPayload{
						ToolID:   id,
						ToolName: name,
						Args:     string(args),
					},
				))
			}
			if callbacks.OnToolCall != nil {
				callbacks.OnToolCall(id, name, args)
			}
		},
		OnToolResult: func(id, result string, isError bool) {
			if a.broker != nil {
				a.broker.Publish(events.NewAgentEvent(
					events.AgentStreamToolResult,
					sessionID,
					events.AgentPayload{
						ToolID:  id,
						Result:  result,
						IsError: isError,
					},
				))
			}
			if callbacks.OnToolResult != nil {
				callbacks.OnToolResult(id, result, isError)
			}
		},
		OnComplete: func(inputTokens, outputTokens int) {
			if a.broker != nil {
				a.broker.Publish(events.NewAgentEvent(
					events.AgentStreamComplete,
					sessionID,
					events.AgentPayload{
						InputTokens:  inputTokens,
						OutputTokens: outputTokens,
					},
				))
			}
			if callbacks.OnComplete != nil {
				callbacks.OnComplete(inputTokens, outputTokens)
			}
		},
		OnError: func(err error) {
			if a.broker != nil {
				a.broker.Publish(events.NewAgentEvent(
					events.AgentStreamError,
					sessionID,
					events.AgentPayload{Error: err},
				))
			}
			if callbacks.OnError != nil {
				callbacks.OnError(err)
			}
		},
	}

	// ... rest of Send implementation using wrappedCallbacks
}
```

---

## Implementation Steps

| Step | Task | Files | Effort |
|------|------|-------|--------|
| 1 | Create pubsub package structure | `internal/pubsub/` | Small |
| 2 | Implement generic Broker[T] | `broker.go` | Medium |
| 3 | Implement base event types | `event.go` | Small |
| 4 | Implement registry | `registry.go` | Small |
| 5 | Write broker tests | `broker_test.go` | Medium |
| 6 | Create event definitions | `events/*.go` | Medium |
| 7 | Implement Hub | `hub.go` | Small |
| 8 | Create TUI Bridge | `internal/tui/bridge/` | Medium |
| 9 | Add tool wrapper | `internal/tools/publisher.go` | Small |
| 10 | Integrate with agent | `internal/agent/agent.go`, `loop.go` | Medium |
| 11 | Update TUI to use bridge | `internal/tui/tui.go` | Medium |
| 12 | Remove direct program.Send | `internal/tui/page/chat/chat.go` | Medium |
| 13 | Add auth event publishing | `internal/config/config.go` | Small |
| 14 | Integration testing | Various | Medium |

### Dependency Graph

```
Step 1-5 (Core Broker)
    │
    ├─► Step 6 (Event Definitions)
    │       │
    │       └─► Step 7 (Hub)
    │               │
    │               ├─► Step 8 (TUI Bridge)
    │               │       │
    │               │       └─► Step 11-12 (TUI Integration)
    │               │
    │               ├─► Step 9 (Tool Wrapper)
    │               │
    │               ├─► Step 10 (Agent Integration)
    │               │
    │               └─► Step 13 (Auth Integration)
    │
    └─► Step 14 (Integration Testing)
```

---

## Testing Strategy

### Unit Tests

```go
// internal/pubsub/broker_test.go
package pubsub

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestBroker_SubscribeAndPublish(t *testing.T) {
	broker := NewBroker[string]("test")
	defer broker.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := broker.Subscribe(ctx)

	broker.Publish("hello")

	select {
	case msg := <-ch:
		if msg != "hello" {
			t.Errorf("expected 'hello', got %q", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestBroker_MultipleSubscribers(t *testing.T) {
	broker := NewBroker[int]("test")
	defer broker.Close()

	ctx := context.Background()
	ch1 := broker.Subscribe(ctx)
	ch2 := broker.Subscribe(ctx)

	if broker.SubscriberCount() != 2 {
		t.Errorf("expected 2 subscribers, got %d", broker.SubscriberCount())
	}

	broker.Publish(42)

	for i, ch := range []<-chan int{ch1, ch2} {
		select {
		case val := <-ch:
			if val != 42 {
				t.Errorf("subscriber %d: expected 42, got %d", i, val)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timeout", i)
		}
	}
}

func TestBroker_ContextCancellation(t *testing.T) {
	broker := NewBroker[string]("test")
	defer broker.Close()

	ctx, cancel := context.WithCancel(context.Background())
	ch := broker.Subscribe(ctx)

	if broker.SubscriberCount() != 1 {
		t.Fatalf("expected 1 subscriber, got %d", broker.SubscriberCount())
	}

	cancel()
	time.Sleep(50 * time.Millisecond)

	if broker.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers after cancel, got %d", broker.SubscriberCount())
	}

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("channel should be closed")
	}
}

func TestBroker_Close(t *testing.T) {
	broker := NewBroker[string]("test")

	ctx := context.Background()
	ch := broker.Subscribe(ctx)

	broker.Close()

	_, ok := <-ch
	if ok {
		t.Error("channel should be closed after broker.Close()")
	}

	// Publishing after close should not panic
	broker.Publish("test")
}

func TestBroker_ConcurrentAccess(t *testing.T) {
	broker := NewBroker[int]("test")
	defer broker.Close()

	var wg sync.WaitGroup

	// Concurrent subscribers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			ch := broker.Subscribe(ctx)
			for range ch {
				// Drain channel
			}
		}()
	}

	// Concurrent publishers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				broker.Publish(id*100 + j)
			}
		}(i)
	}

	wg.Wait()
}

func TestBroker_BufferOverflow(t *testing.T) {
	broker := NewBroker[int]("test", WithBufferSize(2))
	defer broker.Close()

	ctx := context.Background()
	ch := broker.Subscribe(ctx)

	// Fill buffer
	broker.Publish(1)
	broker.Publish(2)

	// This should be dropped (buffer full, no receiver)
	broker.Publish(3)

	// Receive buffered messages
	if v := <-ch; v != 1 {
		t.Errorf("expected 1, got %d", v)
	}
	if v := <-ch; v != 2 {
		t.Errorf("expected 2, got %d", v)
	}

	// No more messages (3 was dropped)
	select {
	case v := <-ch:
		t.Errorf("unexpected message: %d", v)
	case <-time.After(50 * time.Millisecond):
		// Expected
	}
}
```

### Integration Test

```go
// internal/pubsub/integration_test.go
package pubsub_test

import (
	"context"
	"testing"
	"time"

	"github.com/guilhermegouw/cdd/internal/pubsub"
	"github.com/guilhermegouw/cdd/internal/pubsub/events"
)

func TestHub_EndToEnd(t *testing.T) {
	hub := pubsub.NewHub()
	defer hub.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Subscribe to agent events
	agentCh := hub.Agent.Subscribe(ctx)

	// Simulate agent publishing
	hub.Agent.Publish(events.NewAgentEvent(
		events.AgentStreamText,
		"session-1",
		events.AgentPayload{Text: "Hello"},
	))

	select {
	case event := <-agentCh:
		if event.Type != events.AgentStreamText {
			t.Errorf("expected AgentStreamText, got %s", event.Type)
		}
		if event.Payload.Text != "Hello" {
			t.Errorf("expected 'Hello', got %q", event.Payload.Text)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}

	// Check registry
	info := hub.Registry.Info()
	if len(info) != 4 {
		t.Errorf("expected 4 brokers registered, got %d", len(info))
	}
}
```

---

## Migration Guide

### Step 1: Add Hub to Application Bootstrap

```go
// cmd/root.go
func Execute() error {
	// Create event hub
	hub := pubsub.NewHub()
	defer hub.Close()

	// Create agent with broker
	agentFactory := func(model fantasy.LanguageModel) *agent.DefaultAgent {
		return agent.NewDefaultAgent(model, hub.Agent)
	}

	// Pass hub to TUI
	if err := tui.Run(ag, agentFactory, modelFactory, hub); err != nil {
		return err
	}
}
```

### Step 2: Update TUI Initialization

```go
// internal/tui/tui.go
func Run(ag *agent.DefaultAgent, agentFactory AgentFactory, modelFactory ModelFactory, hub *pubsub.Hub) error {
	bridge := bridge.NewBridge(hub)

	m := Model{
		// ... existing fields ...
		hub:    hub,
		bridge: bridge,
	}

	p := tea.NewProgram(m, tea.WithMouseCellMotion())
	bridge.SetProgram(p)
	bridge.Start()
	defer bridge.Stop()

	_, err := p.Run()
	return err
}
```

### Step 3: Update Chat Page Message Handling

```go
// internal/tui/page/chat/chat.go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case bridge.AgentEventMsg:
		return m.handleAgentEvent(msg.Event)
	// ... other cases
	}
}

func (m *Model) handleAgentEvent(event events.AgentEvent) (tea.Model, tea.Cmd) {
	switch event.Type {
	case events.AgentStreamText:
		m.messages.AppendToLast(event.Payload.Text)
	case events.AgentStreamToolCall:
		m.statusBar.SetStatus(StatusToolRunning)
		m.statusBar.SetToolName(event.Payload.ToolName)
	case events.AgentStreamComplete:
		m.statusBar.SetStatus(StatusReady)
		m.input.Enable()
	case events.AgentStreamError:
		m.statusBar.SetStatus(StatusError)
		m.statusBar.SetError(event.Payload.Error)
	}
	return m, nil
}
```

### Step 4: Remove Direct program.Send()

Remove the `program` field from chat.Model and all direct `program.Send()` calls. The bridge now handles this.

---

## Rollback Strategy

Each component can be reverted independently:

| Component | Rollback Action |
|-----------|-----------------|
| Hub | Remove from cmd/root.go, pass nil to agent |
| Bridge | Remove from tui.go, restore direct callbacks |
| Tool Wrapper | Use unwrapped tools |
| Agent Integration | Remove broker field, remove publish calls |

---

## Success Criteria

1. All existing tests pass
2. New pubsub tests achieve >90% coverage
3. Race detector passes: `go test -race ./internal/pubsub/...`
4. Agent has no imports from `internal/tui`
5. Chat page has no direct `program.Send()` calls
6. Registry shows correct subscriber counts
7. No performance regression in streaming latency

---

## Future Enhancements

| Enhancement | Description | Priority |
|-------------|-------------|----------|
| Event filtering | Subscribe to specific event types only | Low |
| Metrics | Track publish/subscribe counts, latency | Low |
| Debug subscriber | Log all events when debug mode enabled | Medium |
| Event replay | Buffer last N events for late subscribers | Low |
| Typed Hub | Generic hub without casting | Low |

---

## Summary

This hybrid approach provides:

- **Type safety** via Go generics (no runtime type assertions)
- **Idiomatic Go** with channels and context cancellation
- **Clear ownership** with service-owned brokers
- **Centralized debugging** via the registry
- **Clean TUI integration** via the bridge pattern
- **Incremental adoption** with clear migration steps

Total implementation: ~400 lines of core code + ~300 lines of event definitions + tests.
