package bridge

import (
	"context"
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/guilhermegouw/cdd/internal/debug"
	"github.com/guilhermegouw/cdd/internal/pubsub"
)

// TUIBridge subscribes to all Hub brokers and forwards events to tea.Program.
// It handles the conversion from domain events to Bubble Tea messages.
type TUIBridge struct { //nolint:govet // fieldalignment: preserving logical field order
	hub     *pubsub.Hub
	program *tea.Program

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Optional filters
	sessionFilter string // Only forward events for this session
}

// TUIBridgeOption configures the TUIBridge.
type TUIBridgeOption func(*TUIBridge)

// WithSessionFilter only forwards events for the specified session.
func WithSessionFilter(sessionID string) TUIBridgeOption {
	return func(b *TUIBridge) {
		b.sessionFilter = sessionID
	}
}

// NewTUIBridge creates a new TUI bridge.
func NewTUIBridge(hub *pubsub.Hub, program *tea.Program, opts ...TUIBridgeOption) *TUIBridge {
	b := &TUIBridge{
		hub:     hub,
		program: program,
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// Start begins forwarding events to the TUI.
// Call Stop() to gracefully shut down.
func (b *TUIBridge) Start(ctx context.Context) {
	b.ctx, b.cancel = context.WithCancel(ctx)

	// Start subscriber goroutines for each broker
	b.wg.Add(5)
	go b.subscribeAgent()
	go b.subscribeTool()
	go b.subscribeSession()
	go b.subscribeAuth()
	go b.subscribeTodo()

	debug.Event("bridge", "start", "TUI bridge started")
}

// Stop gracefully shuts down the bridge.
func (b *TUIBridge) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
	b.wg.Wait()
	debug.Event("bridge", "stop", "TUI bridge stopped")
}

func (b *TUIBridge) subscribeAgent() {
	defer b.wg.Done()

	events := b.hub.Agent.Subscribe(b.ctx)
	for {
		select {
		case <-b.ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}

			// Apply session filter if set
			if b.sessionFilter != "" && event.Payload.SessionID != b.sessionFilter {
				continue
			}

			b.program.Send(AgentEventMsg{Event: event})
		}
	}
}

func (b *TUIBridge) subscribeTool() {
	defer b.wg.Done()

	events := b.hub.Tool.Subscribe(b.ctx)
	for {
		select {
		case <-b.ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}

			// Apply session filter if set
			if b.sessionFilter != "" && event.Payload.SessionID != b.sessionFilter {
				continue
			}

			b.program.Send(ToolEventMsg{Event: event})
		}
	}
}

func (b *TUIBridge) subscribeSession() {
	defer b.wg.Done()

	events := b.hub.Session.Subscribe(b.ctx)
	for {
		select {
		case <-b.ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}

			b.program.Send(SessionEventMsg{Event: event})
		}
	}
}

func (b *TUIBridge) subscribeAuth() {
	defer b.wg.Done()

	events := b.hub.Auth.Subscribe(b.ctx)
	for {
		select {
		case <-b.ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}

			b.program.Send(AuthEventMsg{Event: event})
		}
	}
}

func (b *TUIBridge) subscribeTodo() {
	defer b.wg.Done()

	events := b.hub.Todo.Subscribe(b.ctx)
	for {
		select {
		case <-b.ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}

			// Apply session filter if set
			if b.sessionFilter != "" && event.Payload.SessionID != b.sessionFilter {
				continue
			}

			b.program.Send(TodoEventMsg{Event: event})
		}
	}
}

// SetSessionFilter updates the session filter at runtime.
func (b *TUIBridge) SetSessionFilter(sessionID string) {
	b.sessionFilter = sessionID
}

// ClearSessionFilter removes the session filter.
func (b *TUIBridge) ClearSessionFilter() {
	b.sessionFilter = ""
}
