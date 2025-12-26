package pubsub

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/guilhermegouw/cdd/internal/events"
)

func TestNewHub(t *testing.T) {
	t.Run("creates hub with all brokers initialized", func(t *testing.T) {
		hub := NewHub()
		defer hub.Shutdown()

		if hub.Agent == nil {
			t.Error("Agent broker should be initialized")
		}
		if hub.Tool == nil {
			t.Error("Tool broker should be initialized")
		}
		if hub.Session == nil {
			t.Error("Session broker should be initialized")
		}
		if hub.Auth == nil {
			t.Error("Auth broker should be initialized")
		}
	})

	t.Run("creates hub with registry", func(t *testing.T) {
		hub := NewHub()
		defer hub.Shutdown()

		if hub.Registry() == nil {
			t.Error("Registry should be initialized")
		}
	})
}

func TestHubShutdown(t *testing.T) {
	t.Run("shutdown closes all brokers", func(t *testing.T) {
		hub := NewHub()

		hub.Shutdown()

		if !hub.IsShutdown() {
			t.Error("hub should be shutdown")
		}
		if !hub.Agent.IsShutdown() {
			t.Error("Agent broker should be shutdown")
		}
		if !hub.Tool.IsShutdown() {
			t.Error("Tool broker should be shutdown")
		}
		if !hub.Session.IsShutdown() {
			t.Error("Session broker should be shutdown")
		}
		if !hub.Auth.IsShutdown() {
			t.Error("Auth broker should be shutdown")
		}
	})

	t.Run("double shutdown is safe", func(t *testing.T) {
		hub := NewHub()

		hub.Shutdown()
		hub.Shutdown() // Should not panic

		if !hub.IsShutdown() {
			t.Error("hub should still be shutdown")
		}
	})
}

func TestHubDone(t *testing.T) {
	t.Run("Done channel is open before shutdown", func(t *testing.T) {
		hub := NewHub()
		defer hub.Shutdown()

		select {
		case <-hub.Done():
			t.Error("Done channel should not be closed before shutdown")
		default:
			// Expected
		}
	})

	t.Run("Done channel is closed after shutdown", func(t *testing.T) {
		hub := NewHub()
		hub.Shutdown()

		select {
		case <-hub.Done():
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Error("Done channel should be closed after shutdown")
		}
	})
}

func TestHubAllMetrics(t *testing.T) {
	t.Run("returns metrics for all brokers", func(t *testing.T) {
		hub := NewHub()
		defer hub.Shutdown()

		metrics := hub.AllMetrics()

		if len(metrics) != 5 {
			t.Errorf("expected 5 broker metrics, got %d", len(metrics))
		}

		// Verify broker names
		names := make(map[string]bool)
		for _, m := range metrics {
			names[m.Name] = true
		}

		expectedNames := []string{"agent", "tool", "session", "auth", "todo"}
		for _, name := range expectedNames {
			if !names[name] {
				t.Errorf("expected broker %q in metrics", name)
			}
		}
	})

	t.Run("metrics reflect publish activity", func(t *testing.T) {
		hub := NewHub()
		defer hub.Shutdown()

		ctx := context.Background()
		_ = hub.Agent.Subscribe(ctx)

		hub.Agent.Publish(EventProgress, events.AgentEvent{})
		hub.Agent.Publish(EventProgress, events.AgentEvent{})

		metrics := hub.AllMetrics()
		var agentMetrics BrokerMetrics
		for _, m := range metrics {
			if m.Name == "agent" {
				agentMetrics = m
				break
			}
		}

		if agentMetrics.PublishCount != 2 {
			t.Errorf("expected 2 publishes, got %d", agentMetrics.PublishCount)
		}
	})
}

func TestHubDebugString(t *testing.T) {
	t.Run("returns formatted debug string", func(t *testing.T) {
		hub := NewHub()
		defer hub.Shutdown()

		debugStr := hub.DebugString()

		if debugStr == "" {
			t.Error("debug string should not be empty")
		}
	})
}

func TestHubConcurrentOperations(t *testing.T) {
	t.Run("handles concurrent subscribe and publish", func(t *testing.T) {
		hub := NewHub()
		defer hub.Shutdown()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var wg sync.WaitGroup
		numGoroutines := 10

		// Concurrent subscribers
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				sub := hub.Agent.Subscribe(ctx)
				// Drain a few events
				for j := 0; j < 3; j++ {
					select {
					case <-sub:
					case <-time.After(50 * time.Millisecond):
					}
				}
			}()
		}

		// Concurrent publishers
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 5; j++ {
					hub.Agent.Publish(EventProgress, events.AgentEvent{})
				}
			}()
		}

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Error("concurrent operations timed out")
		}
	})
}

//nolint:gocyclo // Integration test with multiple scenarios
func TestHubBrokerIntegration(t *testing.T) {
	t.Run("events flow correctly between publishers and subscribers", func(t *testing.T) {
		hub := NewHub()
		defer hub.Shutdown()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Subscribe to agent broker
		agentEvents := hub.Agent.Subscribe(ctx)

		// Publish an agent event
		expectedEvent := events.NewTextDeltaEvent("session-1", "msg-1", "Hello")
		hub.Agent.Publish(EventProgress, expectedEvent)

		// Verify event is received
		select {
		case event := <-agentEvents:
			if event.Payload.TextDelta != "Hello" {
				t.Errorf("expected TextDelta 'Hello', got %q", event.Payload.TextDelta)
			}
			if event.Payload.SessionID != "session-1" {
				t.Errorf("expected SessionID 'session-1', got %q", event.Payload.SessionID)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("timeout waiting for event")
		}
	})

	t.Run("multiple brokers work independently", func(t *testing.T) {
		hub := NewHub()
		defer hub.Shutdown()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Subscribe to all brokers
		agentEvents := hub.Agent.Subscribe(ctx)
		toolEvents := hub.Tool.Subscribe(ctx)
		sessionEvents := hub.Session.Subscribe(ctx)
		authEvents := hub.Auth.Subscribe(ctx)

		// Publish to each broker
		hub.Agent.Publish(EventProgress, events.NewTextDeltaEvent("s", "m", "agent"))
		hub.Tool.Publish(EventStarted, events.NewToolStartedEvent("s", "tc", "tool", "{}"))
		hub.Session.Publish(EventCreated, events.NewSessionCreatedEvent("s", "session"))
		hub.Auth.Publish(EventCompleted, events.NewTokenRefreshedEvent("provider", time.Now()))

		// Verify each broker received its event
		timeout := 100 * time.Millisecond

		select {
		case e := <-agentEvents:
			if e.Payload.TextDelta != "agent" {
				t.Error("agent event mismatch")
			}
		case <-time.After(timeout):
			t.Error("timeout waiting for agent event")
		}

		select {
		case e := <-toolEvents:
			if e.Payload.ToolName != "tool" {
				t.Error("tool event mismatch")
			}
		case <-time.After(timeout):
			t.Error("timeout waiting for tool event")
		}

		select {
		case e := <-sessionEvents:
			if e.Payload.Title != "session" {
				t.Error("session event mismatch")
			}
		case <-time.After(timeout):
			t.Error("timeout waiting for session event")
		}

		select {
		case e := <-authEvents:
			if e.Payload.ProviderID != "provider" {
				t.Error("auth event mismatch")
			}
		case <-time.After(timeout):
			t.Error("timeout waiting for auth event")
		}
	})
}
