package pubsub

import (
	"sync"

	"github.com/guilhermegouw/cdd/internal/events"
)

// Hub is the central container for all domain brokers.
// It provides lifecycle management and debugging capabilities.
type Hub struct { //nolint:govet // fieldalignment: preserving logical field order
	Agent   *Broker[events.AgentEvent]
	Tool    *Broker[events.ToolEvent]
	Session *Broker[events.SessionEvent]
	Auth    *Broker[events.AuthEvent]
	Todo    *Broker[events.TodoEvent]

	registry *Registry
	done     chan struct{}
}

// NewHub creates a new Hub with all domain brokers initialized.
func NewHub() *Hub {
	h := &Hub{
		Agent:    NewBroker[events.AgentEvent]("agent"),
		Tool:     NewBroker[events.ToolEvent]("tool"),
		Session:  NewBroker[events.SessionEvent]("session"),
		Auth:     NewBroker[events.AuthEvent]("auth"),
		Todo:     NewBroker[events.TodoEvent]("todo"),
		registry: NewRegistry(),
		done:     make(chan struct{}),
	}

	// Register all brokers in the registry for debugging
	h.registry.Register("agent", h.Agent)
	h.registry.Register("tool", h.Tool)
	h.registry.Register("session", h.Session)
	h.registry.Register("auth", h.Auth)
	h.registry.Register("todo", h.Todo)

	return h
}

// Shutdown gracefully shuts down all brokers.
func (h *Hub) Shutdown() {
	select {
	case <-h.done:
		return // Already shut down
	default:
		close(h.done)
	}

	// Shutdown all brokers concurrently
	var wg sync.WaitGroup
	wg.Add(5)

	go func() { defer wg.Done(); h.Agent.Shutdown() }()
	go func() { defer wg.Done(); h.Tool.Shutdown() }()
	go func() { defer wg.Done(); h.Session.Shutdown() }()
	go func() { defer wg.Done(); h.Auth.Shutdown() }()
	go func() { defer wg.Done(); h.Todo.Shutdown() }()

	wg.Wait()
}

// IsShutdown returns true if the hub has been shut down.
func (h *Hub) IsShutdown() bool {
	select {
	case <-h.done:
		return true
	default:
		return false
	}
}

// Done returns a channel that's closed when the hub is shut down.
func (h *Hub) Done() <-chan struct{} {
	return h.done
}

// Registry returns the debug registry for introspection.
func (h *Hub) Registry() *Registry {
	return h.registry
}

// AllMetrics returns metrics for all brokers.
func (h *Hub) AllMetrics() []BrokerMetrics {
	return []BrokerMetrics{
		h.Agent.Metrics(),
		h.Tool.Metrics(),
		h.Session.Metrics(),
		h.Auth.Metrics(),
		h.Todo.Metrics(),
	}
}

// DebugString returns a formatted debug string for all brokers.
func (h *Hub) DebugString() string {
	return h.registry.DebugString()
}
