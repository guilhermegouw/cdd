package pubsub

import (
	"fmt"
	"strings"
	"sync"
)

// BrokerInfo provides debug information about a registered broker.
type BrokerInfo interface {
	Name() string
	SubscriberCount() int
	IsShutdown() bool
	Metrics() BrokerMetrics
}

// Registry tracks all brokers for debugging and introspection.
type Registry struct {
	brokers map[string]BrokerInfo
	mu      sync.RWMutex
}

// NewRegistry creates a new broker registry.
func NewRegistry() *Registry {
	return &Registry{
		brokers: make(map[string]BrokerInfo),
	}
}

// Register adds a broker to the registry.
func (r *Registry) Register(name string, broker BrokerInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.brokers[name] = broker
}

// Unregister removes a broker from the registry.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.brokers, name)
}

// Get retrieves a broker by name.
func (r *Registry) Get(name string) (BrokerInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.brokers[name]
	return b, ok
}

// List returns all registered broker names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.brokers))
	for name := range r.brokers {
		names = append(names, name)
	}
	return names
}

// AllMetrics returns metrics for all registered brokers.
func (r *Registry) AllMetrics() map[string]BrokerMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics := make(map[string]BrokerMetrics, len(r.brokers))
	for name, broker := range r.brokers {
		metrics[name] = broker.Metrics()
	}
	return metrics
}

// DebugString returns a formatted debug string for all brokers.
func (r *Registry) DebugString() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== Broker Registry (%d brokers) ===\n", len(r.brokers)))

	for name, broker := range r.brokers {
		m := broker.Metrics()
		sb.WriteString(fmt.Sprintf(
			"  %s: subs=%d (peak=%d), published=%d, dropped=%d, shutdown=%v\n",
			name, m.SubscriberCount, m.SubscriberPeak,
			m.PublishCount, m.DropCount, broker.IsShutdown(),
		))
	}

	return sb.String()
}
