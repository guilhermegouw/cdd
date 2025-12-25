# Pub/Sub Broker Implementation Plan

> Implementation plan for the generic `Broker[T]` event system in CDD CLI.

---

## Table of Contents

1. [Overview](#overview)
2. [Goals & Non-Goals](#goals--non-goals)
3. [Design](#design)
4. [Implementation Steps](#implementation-steps)
5. [File Structure](#file-structure)
6. [Testing Strategy](#testing-strategy)
7. [Integration Points](#integration-points)
8. [Effort Estimate](#effort-estimate)

---

## Overview

The Pub/Sub system enables **decoupled communication** between services in CDD CLI. Instead of services calling each other directly, they publish events that other components can subscribe to.

### Why Pub/Sub?

```
❌ Direct Coupling                    ✅ Event-Driven (Pub/Sub)
─────────────────                    ────────────────────────
SessionService.Create() {            SessionService.Create() {
    session := ...                       session := ...
    messageService.Notify(session)       broker.Publish(Created, session)
    tuiService.Update(session)       }
    historyService.Log(session)
}                                    // Subscribers react independently
                                     // TUI, History, etc. subscribe to events
```

### Benefits

| Benefit | Description |
|---------|-------------|
| **Decoupling** | Services don't know about each other |
| **Extensibility** | Add new subscribers without changing publishers |
| **Testability** | Easy to mock event flow |
| **TUI Sync** | TUI stays updated automatically via `program.Send()` |

---

## Goals & Non-Goals

### Goals

- [x] Generic `Broker[T]` that works with any event payload type
- [x] Thread-safe subscription and publishing
- [x] Context-aware subscriptions (auto-cleanup on context cancellation)
- [x] Simple API: `Subscribe()`, `Publish()`, `Close()`
- [x] Integration pattern for Bubble Tea TUI

### Non-Goals

- Persistence (events are in-memory only)
- Replay/history (no event sourcing)
- Distributed pub/sub (single process only)
- Message acknowledgment (fire-and-forget)
- Filtering by event type at subscription time (subscribers receive all events)

---

## Design

### Core Types

```go
package pubsub

// EventType represents the kind of event
type EventType string

const (
    Created EventType = "created"
    Updated EventType = "updated"
    Deleted EventType = "deleted"
)

// Event wraps a payload with metadata
type Event[T any] struct {
    Type      EventType
    Payload   T
    Timestamp time.Time
}

// Broker manages subscriptions for a specific payload type
type Broker[T any] struct {
    subs map[chan Event[T]]struct{}
    mu   sync.RWMutex
}
```

### API

```go
// NewBroker creates a new broker for type T
func NewBroker[T any]() *Broker[T]

// Subscribe returns a channel that receives events
// The channel is closed when ctx is cancelled
func (b *Broker[T]) Subscribe(ctx context.Context) <-chan Event[T]

// Publish sends an event to all subscribers
func (b *Broker[T]) Publish(eventType EventType, payload T)

// Close shuts down the broker and closes all subscriber channels
func (b *Broker[T]) Close()
```

### Usage Pattern

```go
// Service owns its data and publishes events
type SessionService struct {
    db     *database.DB
    broker *pubsub.Broker[Session]
}

func (s *SessionService) Create(ctx context.Context, title string) (Session, error) {
    session := Session{ID: uuid.New(), Title: title}
    
    // Persist
    if err := s.db.InsertSession(ctx, session); err != nil {
        return Session{}, err
    }
    
    // Publish event
    s.broker.Publish(pubsub.Created, session)
    
    return session, nil
}

// Broker returns the broker for external subscription
func (s *SessionService) Broker() *pubsub.Broker[Session] {
    return s.broker
}
```

### TUI Integration Pattern

```go
// In TUI model initialization
func (m *Model) Init() tea.Cmd {
    return tea.Batch(
        m.subscribeToSessions(),
        m.subscribeToMessages(),
    )
}

// Subscribe and forward events to Bubble Tea
func (m *Model) subscribeToSessions() tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithCancel(context.Background())
        m.cancelSessions = cancel // Store for cleanup
        
        ch := m.app.Sessions().Broker().Subscribe(ctx)
        
        // Goroutine forwards events to TUI
        go func() {
            for event := range ch {
                m.program.Send(SessionEventMsg{Event: event})
            }
        }()
        
        return nil
    }
}

// Handle forwarded events in Update()
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case SessionEventMsg:
        // Update UI based on event
        switch msg.Event.Type {
        case pubsub.Created:
            m.sessions = append(m.sessions, msg.Event.Payload)
        case pubsub.Deleted:
            m.sessions = removeSession(m.sessions, msg.Event.Payload.ID)
        }
        return m, nil
    }
    // ...
}
```

---

## Implementation Steps

### Step 1: Create `internal/pubsub/` Package

Create the base package structure.

```bash
mkdir -p internal/pubsub
```

### Step 2: Implement Event Types (`events.go`)

```go
// internal/pubsub/events.go
package pubsub

import "time"

// EventType categorizes events
type EventType string

const (
    Created EventType = "created"
    Updated EventType = "updated"
    Deleted EventType = "deleted"
)

// Event wraps a payload with metadata
type Event[T any] struct {
    Type      EventType
    Payload   T
    Timestamp time.Time
}

// NewEvent creates a new event with current timestamp
func NewEvent[T any](eventType EventType, payload T) Event[T] {
    return Event[T]{
        Type:      eventType,
        Payload:   payload,
        Timestamp: time.Now(),
    }
}
```

### Step 3: Implement Broker (`broker.go`)

```go
// internal/pubsub/broker.go
package pubsub

import (
    "context"
    "sync"
)

// Broker manages event subscriptions for type T
type Broker[T any] struct {
    subs   map[chan Event[T]]struct{}
    mu     sync.RWMutex
    closed bool
}

// NewBroker creates a new broker
func NewBroker[T any]() *Broker[T] {
    return &Broker[T]{
        subs: make(map[chan Event[T]]struct{}),
    }
}

// Subscribe creates a subscription that receives events
// The returned channel is closed when:
// - The provided context is cancelled
// - The broker is closed
func (b *Broker[T]) Subscribe(ctx context.Context) <-chan Event[T] {
    ch := make(chan Event[T], 16) // Buffered to prevent blocking publishers
    
    b.mu.Lock()
    if b.closed {
        b.mu.Unlock()
        close(ch)
        return ch
    }
    b.subs[ch] = struct{}{}
    b.mu.Unlock()
    
    // Cleanup on context cancellation
    go func() {
        <-ctx.Done()
        b.unsubscribe(ch)
    }()
    
    return ch
}

// Publish sends an event to all subscribers
// Non-blocking: if a subscriber's buffer is full, the event is dropped for that subscriber
func (b *Broker[T]) Publish(eventType EventType, payload T) {
    event := NewEvent(eventType, payload)
    
    b.mu.RLock()
    defer b.mu.RUnlock()
    
    if b.closed {
        return
    }
    
    for ch := range b.subs {
        select {
        case ch <- event:
        default:
            // Channel full, skip (subscriber is slow)
        }
    }
}

// Close shuts down the broker and closes all subscriber channels
func (b *Broker[T]) Close() {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    if b.closed {
        return
    }
    
    b.closed = true
    for ch := range b.subs {
        close(ch)
        delete(b.subs, ch)
    }
}

// unsubscribe removes a channel from subscribers
func (b *Broker[T]) unsubscribe(ch chan Event[T]) {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    if _, ok := b.subs[ch]; ok {
        close(ch)
        delete(b.subs, ch)
    }
}

// SubscriberCount returns the current number of subscribers (for testing)
func (b *Broker[T]) SubscriberCount() int {
    b.mu.RLock()
    defer b.mu.RUnlock()
    return len(b.subs)
}
```

### Step 4: Write Tests (`broker_test.go`)

```go
// internal/pubsub/broker_test.go
package pubsub

import (
    "context"
    "sync"
    "testing"
    "time"
)

type TestPayload struct {
    ID   string
    Name string
}

func TestBroker_SubscribeAndPublish(t *testing.T) {
    broker := NewBroker[TestPayload]()
    defer broker.Close()
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    ch := broker.Subscribe(ctx)
    
    // Publish event
    payload := TestPayload{ID: "1", Name: "test"}
    broker.Publish(Created, payload)
    
    // Receive event
    select {
    case event := <-ch:
        if event.Type != Created {
            t.Errorf("expected Created, got %v", event.Type)
        }
        if event.Payload.ID != "1" {
            t.Errorf("expected ID 1, got %s", event.Payload.ID)
        }
    case <-time.After(time.Second):
        t.Fatal("timeout waiting for event")
    }
}

func TestBroker_MultipleSubscribers(t *testing.T) {
    broker := NewBroker[TestPayload]()
    defer broker.Close()
    
    ctx := context.Background()
    ch1 := broker.Subscribe(ctx)
    ch2 := broker.Subscribe(ctx)
    
    if broker.SubscriberCount() != 2 {
        t.Errorf("expected 2 subscribers, got %d", broker.SubscriberCount())
    }
    
    // Publish
    broker.Publish(Created, TestPayload{ID: "1"})
    
    // Both should receive
    for _, ch := range []<-chan Event[TestPayload]{ch1, ch2} {
        select {
        case event := <-ch:
            if event.Payload.ID != "1" {
                t.Errorf("expected ID 1, got %s", event.Payload.ID)
            }
        case <-time.After(time.Second):
            t.Fatal("timeout")
        }
    }
}

func TestBroker_ContextCancellation(t *testing.T) {
    broker := NewBroker[TestPayload]()
    defer broker.Close()
    
    ctx, cancel := context.WithCancel(context.Background())
    ch := broker.Subscribe(ctx)
    
    if broker.SubscriberCount() != 1 {
        t.Errorf("expected 1 subscriber, got %d", broker.SubscriberCount())
    }
    
    // Cancel context
    cancel()
    
    // Wait for cleanup
    time.Sleep(50 * time.Millisecond)
    
    if broker.SubscriberCount() != 0 {
        t.Errorf("expected 0 subscribers after cancel, got %d", broker.SubscriberCount())
    }
    
    // Channel should be closed
    select {
    case _, ok := <-ch:
        if ok {
            t.Error("channel should be closed")
        }
    default:
        // Channel might be closed but empty
    }
}

func TestBroker_Close(t *testing.T) {
    broker := NewBroker[TestPayload]()
    
    ctx := context.Background()
    ch := broker.Subscribe(ctx)
    
    broker.Close()
    
    // Channel should be closed
    _, ok := <-ch
    if ok {
        t.Error("channel should be closed after broker.Close()")
    }
    
    // Publishing after close should not panic
    broker.Publish(Created, TestPayload{})
}

func TestBroker_ConcurrentAccess(t *testing.T) {
    broker := NewBroker[TestPayload]()
    defer broker.Close()
    
    var wg sync.WaitGroup
    
    // Concurrent subscribers
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            ctx, cancel := context.WithCancel(context.Background())
            ch := broker.Subscribe(ctx)
            
            // Read a few events
            for j := 0; j < 5; j++ {
                select {
                case <-ch:
                case <-time.After(100 * time.Millisecond):
                }
            }
            cancel()
        }()
    }
    
    // Concurrent publishers
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            for j := 0; j < 10; j++ {
                broker.Publish(Created, TestPayload{ID: string(rune(id))})
            }
        }(i)
    }
    
    wg.Wait()
}
```

---

## File Structure

```
internal/pubsub/
├── broker.go       # Broker[T] implementation
├── broker_test.go  # Unit tests
└── events.go       # Event types and EventType constants
```

**Total: ~200 lines of code** (excluding tests)

---

## Testing Strategy

### Unit Tests

| Test Case | Description |
|-----------|-------------|
| `TestBroker_SubscribeAndPublish` | Basic subscribe and receive |
| `TestBroker_MultipleSubscribers` | Multiple subscribers receive same event |
| `TestBroker_ContextCancellation` | Cleanup on context cancel |
| `TestBroker_Close` | Broker shutdown closes all channels |
| `TestBroker_ConcurrentAccess` | Race condition testing |
| `TestBroker_BufferOverflow` | Behavior when subscriber is slow |

### Integration Tests

| Test Case | Description |
|-----------|-------------|
| Service → Broker → TUI | End-to-end event flow |
| Multiple services | Cross-service event handling |

### Running Tests

```bash
# Run pubsub tests
task test -- -run TestBroker ./internal/pubsub/...

# Run with race detector
task test -- -race ./internal/pubsub/...
```

---

## Integration Points

### 1. Session Service Integration

```go
// internal/session/service.go
type Service struct {
    db     *database.DB
    broker *pubsub.Broker[Session]
}

func NewService(db *database.DB) *Service {
    return &Service{
        db:     db,
        broker: pubsub.NewBroker[Session](),
    }
}

func (s *Service) Create(ctx context.Context, title string) (Session, error) {
    session := Session{/* ... */}
    if err := s.db.InsertSession(ctx, session); err != nil {
        return Session{}, err
    }
    s.broker.Publish(pubsub.Created, session)
    return session, nil
}

func (s *Service) Broker() *pubsub.Broker[Session] {
    return s.broker
}
```

### 2. Message Service Integration

```go
// internal/message/service.go
type Service struct {
    db     *database.DB
    broker *pubsub.Broker[Message]
}

func (s *Service) Add(ctx context.Context, msg Message) error {
    if err := s.db.InsertMessage(ctx, msg); err != nil {
        return err
    }
    s.broker.Publish(pubsub.Created, msg)
    return nil
}
```

### 3. Permission Service Integration

```go
// internal/permission/service.go
type Service struct {
    mode   Mode
    broker *pubsub.Broker[Permission]
}

func (s *Service) Request(ctx context.Context, perm Permission) (bool, error) {
    s.broker.Publish(pubsub.Created, perm) // TUI shows dialog
    
    // Wait for response via another channel (approval flow)
    // ...
}
```

### 4. TUI Integration

```go
// internal/tui/tui.go
type Model struct {
    app           *app.App
    program       *tea.Program
    cancelSession context.CancelFunc
    cancelMessage context.CancelFunc
}

func (m *Model) Init() tea.Cmd {
    return tea.Batch(
        m.subscribeToSessions(),
        m.subscribeToMessages(),
    )
}

func (m *Model) subscribeToSessions() tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithCancel(context.Background())
        m.cancelSession = cancel
        
        ch := m.app.Sessions().Broker().Subscribe(ctx)
        go func() {
            for event := range ch {
                m.program.Send(SessionEventMsg{Event: event})
            }
        }()
        return nil
    }
}

// Cleanup on quit
func (m *Model) cleanup() {
    if m.cancelSession != nil {
        m.cancelSession()
    }
    if m.cancelMessage != nil {
        m.cancelMessage()
    }
}
```

---

## Effort Estimate

| Task | Effort | Time |
|------|--------|------|
| `events.go` | Low | 30 min |
| `broker.go` | Medium | 1-2 hours |
| `broker_test.go` | Medium | 1-2 hours |
| Documentation | Low | 30 min |
| **Total** | **~200 LOC** | **3-4 hours** |

---

## Checklist

- [ ] Create `internal/pubsub/` directory
- [ ] Implement `events.go` with `EventType` and `Event[T]`
- [ ] Implement `broker.go` with `Broker[T]`
- [ ] Write comprehensive tests in `broker_test.go`
- [ ] Verify race-free with `go test -race`
- [ ] Document usage patterns in code comments
- [ ] Integrate with first service (Session) as proof of concept

---

## Future Considerations

### Potential Enhancements (Post-MVP)

| Enhancement | Description | Priority |
|-------------|-------------|----------|
| Event filtering | Subscribe to specific event types only | Low |
| Event history | Keep last N events for late subscribers | Low |
| Metrics | Track publish/subscribe counts | Low |
| Persistent events | Store events for replay | Low |

### Design Decisions Log

| Decision | Rationale |
|----------|-----------|
| Buffered channels (16) | Prevent slow subscribers from blocking publishers |
| Non-blocking publish | Dropped events are acceptable for UI updates |
| Context-based cleanup | Automatic unsubscribe on cancellation |
| Generic `Broker[T]` | Type-safe, no casting required |

---

## References

- [CDD Architecture](/docs/architecture/CDD-ARCHITECTURE.md) - Overall system design
- [CDD Features](/docs/architecture/CDD-FEATURES.md) - Feature roadmap
- Go Generics - [Type Parameters Proposal](https://go.dev/blog/intro-generics)
- Bubble Tea - [program.Send() for external messages](https://github.com/charmbracelet/bubbletea#sending-messages-from-outside-the-program)
