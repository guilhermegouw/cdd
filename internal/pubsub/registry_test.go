package pubsub

import (
	"strings"
	"testing"
)

// mockBrokerInfo is a mock implementation of BrokerInfo for testing.
type mockBrokerInfo struct {
	name            string
	subscriberCount int
	isShutdown      bool
	metrics         BrokerMetrics
}

func (m *mockBrokerInfo) Name() string {
	return m.name
}

func (m *mockBrokerInfo) SubscriberCount() int {
	return m.subscriberCount
}

func (m *mockBrokerInfo) IsShutdown() bool {
	return m.isShutdown
}

func (m *mockBrokerInfo) Metrics() BrokerMetrics {
	return m.metrics
}

func TestNewRegistry(t *testing.T) {
	t.Run("creates empty registry", func(t *testing.T) {
		reg := NewRegistry()

		if reg == nil {
			t.Fatal("registry should not be nil")
		}

		brokers := reg.List()
		if len(brokers) != 0 {
			t.Errorf("expected 0 brokers, got %d", len(brokers))
		}
	})
}

func TestRegistryRegister(t *testing.T) {
	t.Run("registers broker by name", func(t *testing.T) {
		reg := NewRegistry()

		broker := &mockBrokerInfo{name: "test", metrics: BrokerMetrics{Name: "test"}}
		reg.Register("test", broker)

		brokers := reg.List()
		if len(brokers) != 1 {
			t.Errorf("expected 1 broker, got %d", len(brokers))
		}
	})

	t.Run("registers multiple brokers", func(t *testing.T) {
		reg := NewRegistry()

		reg.Register("broker1", &mockBrokerInfo{name: "broker1"})
		reg.Register("broker2", &mockBrokerInfo{name: "broker2"})
		reg.Register("broker3", &mockBrokerInfo{name: "broker3"})

		brokers := reg.List()
		if len(brokers) != 3 {
			t.Errorf("expected 3 brokers, got %d", len(brokers))
		}
	})

	t.Run("overwrites broker with same name", func(t *testing.T) {
		reg := NewRegistry()

		broker1 := &mockBrokerInfo{name: "test", metrics: BrokerMetrics{PublishCount: 1}}
		broker2 := &mockBrokerInfo{name: "test", metrics: BrokerMetrics{PublishCount: 2}}

		reg.Register("test", broker1)
		reg.Register("test", broker2)

		brokers := reg.List()
		if len(brokers) != 1 {
			t.Errorf("expected 1 broker, got %d", len(brokers))
		}
	})
}

func TestRegistryUnregister(t *testing.T) {
	t.Run("removes registered broker", func(t *testing.T) {
		reg := NewRegistry()

		broker := &mockBrokerInfo{name: "test"}
		reg.Register("test", broker)
		reg.Unregister("test")

		brokers := reg.List()
		if len(brokers) != 0 {
			t.Errorf("expected 0 brokers after unregister, got %d", len(brokers))
		}
	})

	t.Run("unregistering nonexistent broker is safe", func(t *testing.T) {
		reg := NewRegistry()

		// Should not panic
		reg.Unregister("nonexistent")
	})
}

func TestRegistryGet(t *testing.T) {
	t.Run("gets registered broker", func(t *testing.T) {
		reg := NewRegistry()

		broker := &mockBrokerInfo{name: "test"}
		reg.Register("test", broker)

		retrieved, ok := reg.Get("test")
		if !ok {
			t.Error("expected to find broker")
		}
		if retrieved != broker {
			t.Error("retrieved broker should match registered broker")
		}
	})

	t.Run("returns false for nonexistent broker", func(t *testing.T) {
		reg := NewRegistry()

		_, ok := reg.Get("nonexistent")
		if ok {
			t.Error("expected not to find nonexistent broker")
		}
	})
}

func TestRegistryList(t *testing.T) {
	t.Run("returns all registered broker names", func(t *testing.T) {
		reg := NewRegistry()

		names := []string{"agent", "tool", "session"}
		for _, name := range names {
			reg.Register(name, &mockBrokerInfo{name: name})
		}

		list := reg.List()

		if len(list) != 3 {
			t.Errorf("expected 3 brokers, got %d", len(list))
		}

		// Check all names are present
		nameMap := make(map[string]bool)
		for _, name := range list {
			nameMap[name] = true
		}
		for _, name := range names {
			if !nameMap[name] {
				t.Errorf("expected broker name %q in list", name)
			}
		}
	})

	t.Run("returns empty slice for empty registry", func(t *testing.T) {
		reg := NewRegistry()

		list := reg.List()

		if list == nil {
			t.Error("List should return non-nil slice")
		}
		if len(list) != 0 {
			t.Errorf("expected 0 brokers, got %d", len(list))
		}
	})
}

func TestRegistryAllMetrics(t *testing.T) {
	t.Run("returns metrics for all brokers", func(t *testing.T) {
		reg := NewRegistry()

		reg.Register("agent", &mockBrokerInfo{
			name:    "agent",
			metrics: BrokerMetrics{Name: "agent", PublishCount: 10},
		})
		reg.Register("tool", &mockBrokerInfo{
			name:    "tool",
			metrics: BrokerMetrics{Name: "tool", PublishCount: 5},
		})

		metrics := reg.AllMetrics()

		if len(metrics) != 2 {
			t.Errorf("expected 2 metrics, got %d", len(metrics))
		}

		if metrics["agent"].PublishCount != 10 {
			t.Errorf("expected agent publish count 10, got %d", metrics["agent"].PublishCount)
		}
		if metrics["tool"].PublishCount != 5 {
			t.Errorf("expected tool publish count 5, got %d", metrics["tool"].PublishCount)
		}
	})

	t.Run("returns empty map for empty registry", func(t *testing.T) {
		reg := NewRegistry()

		metrics := reg.AllMetrics()

		if len(metrics) != 0 {
			t.Errorf("expected 0 metrics, got %d", len(metrics))
		}
	})
}

func TestRegistryDebugString(t *testing.T) {
	t.Run("returns formatted debug string", func(t *testing.T) {
		reg := NewRegistry()

		reg.Register("agent", &mockBrokerInfo{
			name: "agent",
			metrics: BrokerMetrics{
				Name:            "agent",
				PublishCount:    100,
				DropCount:       5,
				SubscriberCount: 2,
				SubscriberPeak:  3,
			},
		})

		debugStr := reg.DebugString()

		// Verify it contains expected information
		if !strings.Contains(debugStr, "agent") {
			t.Error("debug string should contain broker name")
		}
		if !strings.Contains(debugStr, "100") {
			t.Error("debug string should contain publish count")
		}
		if !strings.Contains(debugStr, "Broker Registry") {
			t.Error("debug string should contain header")
		}
	})

	t.Run("handles empty registry", func(t *testing.T) {
		reg := NewRegistry()

		debugStr := reg.DebugString()

		if !strings.Contains(debugStr, "0 brokers") {
			t.Error("debug string should indicate 0 brokers")
		}
	})

	t.Run("includes all broker names", func(t *testing.T) {
		reg := NewRegistry()

		names := []string{"agent", "tool", "session", "auth"}
		for _, name := range names {
			reg.Register(name, &mockBrokerInfo{
				name:    name,
				metrics: BrokerMetrics{Name: name},
			})
		}

		debugStr := reg.DebugString()

		for _, name := range names {
			if !strings.Contains(debugStr, name) {
				t.Errorf("debug string should contain broker name %q", name)
			}
		}
	})

	t.Run("shows shutdown status", func(t *testing.T) {
		reg := NewRegistry()

		reg.Register("shutdown_broker", &mockBrokerInfo{
			name:       "shutdown_broker",
			isShutdown: true,
			metrics:    BrokerMetrics{Name: "shutdown_broker"},
		})

		debugStr := reg.DebugString()

		if !strings.Contains(debugStr, "true") {
			t.Error("debug string should show shutdown=true")
		}
	})
}

func TestRegistryConcurrency(t *testing.T) {
	t.Run("handles concurrent register and get", func(t *testing.T) {
		reg := NewRegistry()

		done := make(chan bool)

		// Concurrent registrations
		go func() {
			for i := 0; i < 100; i++ {
				reg.Register("broker", &mockBrokerInfo{name: "broker"})
			}
			done <- true
		}()

		// Concurrent reads
		go func() {
			for i := 0; i < 100; i++ {
				_ = reg.List()
				_ = reg.AllMetrics()
				_ = reg.DebugString()
				_, _ = reg.Get("broker")
			}
			done <- true
		}()

		<-done
		<-done
	})

	t.Run("handles concurrent register and unregister", func(t *testing.T) {
		reg := NewRegistry()

		done := make(chan bool)

		go func() {
			for i := 0; i < 100; i++ {
				reg.Register("broker", &mockBrokerInfo{name: "broker"})
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 100; i++ {
				reg.Unregister("broker")
			}
			done <- true
		}()

		<-done
		<-done
	})
}
