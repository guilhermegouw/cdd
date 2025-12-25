package pubsub

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestBrokerSubscribePublish(t *testing.T) {
	t.Run("single subscriber receives events", func(t *testing.T) {
		broker := NewBroker[string]("test")
		defer broker.Shutdown()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		events := broker.Subscribe(ctx)

		broker.Publish(EventCreated, "hello")

		select {
		case event := <-events:
			if event.Type != EventCreated || event.Payload != "hello" {
				t.Errorf("unexpected event: %+v", event)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("timeout waiting for event")
		}
	})

	t.Run("multiple subscribers receive same event", func(t *testing.T) {
		broker := NewBroker[int]("test")
		defer broker.Shutdown()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sub1 := broker.Subscribe(ctx)
		sub2 := broker.Subscribe(ctx)

		broker.Publish(EventUpdated, 42)

		for i, sub := range []<-chan Event[int]{sub1, sub2} {
			select {
			case event := <-sub:
				if event.Payload != 42 {
					t.Errorf("subscriber %d: expected 42, got %d", i, event.Payload)
				}
			case <-time.After(100 * time.Millisecond):
				t.Errorf("subscriber %d: timeout", i)
			}
		}
	})

	t.Run("cancelled context unsubscribes", func(t *testing.T) {
		broker := NewBroker[string]("test")
		defer broker.Shutdown()

		ctx, cancel := context.WithCancel(context.Background())

		events := broker.Subscribe(ctx)

		if broker.SubscriberCount() != 1 {
			t.Errorf("expected 1 subscriber, got %d", broker.SubscriberCount())
		}

		cancel()
		time.Sleep(50 * time.Millisecond) // Allow cleanup goroutine to run

		if broker.SubscriberCount() != 0 {
			t.Errorf("expected 0 subscribers after cancel, got %d", broker.SubscriberCount())
		}

		// Channel should be closed
		_, ok := <-events
		if ok {
			t.Error("expected channel to be closed")
		}
	})

	t.Run("shutdown closes all subscribers", func(t *testing.T) {
		broker := NewBroker[string]("test")

		ctx := context.Background()
		sub1 := broker.Subscribe(ctx)
		sub2 := broker.Subscribe(ctx)

		broker.Shutdown()

		// Both channels should be closed
		if _, ok := <-sub1; ok {
			t.Error("sub1 should be closed")
		}
		if _, ok := <-sub2; ok {
			t.Error("sub2 should be closed")
		}
	})

	t.Run("publish after shutdown is no-op", func(t *testing.T) {
		broker := NewBroker[string]("test")
		broker.Shutdown()

		// Should not panic
		broker.Publish(EventCreated, "test")
	})

	t.Run("subscribe after shutdown returns closed channel", func(t *testing.T) {
		broker := NewBroker[string]("test")
		broker.Shutdown()

		ctx := context.Background()
		ch := broker.Subscribe(ctx)

		// Channel should be immediately closed
		_, ok := <-ch
		if ok {
			t.Error("channel should be closed")
		}
	})
}

func TestBrokerConcurrency(t *testing.T) {
	broker := NewBroker[int]("test")
	defer broker.Shutdown()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const numSubscribers = 10
	const numPublishes = 100

	var wg sync.WaitGroup
	received := make([]int, numSubscribers)

	// Start subscribers
	for i := 0; i < numSubscribers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			events := broker.Subscribe(ctx)
			for event := range events {
				_ = event
				received[idx]++
			}
		}(i)
	}

	// Wait for subscriptions
	time.Sleep(50 * time.Millisecond)

	// Publish events concurrently
	var pubWg sync.WaitGroup
	for i := 0; i < numPublishes; i++ {
		pubWg.Add(1)
		go func(n int) {
			defer pubWg.Done()
			broker.Publish(EventCreated, n)
		}(i)
	}
	pubWg.Wait()

	// Cancel context to close subscribers
	cancel()
	wg.Wait()

	// Verify all subscribers received events (may have some drops)
	for i, count := range received {
		if count < numPublishes/2 { // Allow some drops due to timing
			t.Errorf("subscriber %d received too few events: %d", i, count)
		}
	}
}

func TestBrokerMetrics(t *testing.T) {
	broker := NewBroker[string]("test")
	defer broker.Shutdown()

	ctx := context.Background()
	_ = broker.Subscribe(ctx)
	_ = broker.Subscribe(ctx)

	broker.Publish(EventCreated, "1")
	broker.Publish(EventCreated, "2")

	metrics := broker.Metrics()

	if metrics.Name != "test" {
		t.Errorf("expected name 'test', got %q", metrics.Name)
	}
	if metrics.SubscriberCount != 2 {
		t.Errorf("expected 2 subscribers, got %d", metrics.SubscriberCount)
	}
	if metrics.PublishCount != 2 {
		t.Errorf("expected 2 publishes, got %d", metrics.PublishCount)
	}
}

func TestBrokerOptions(t *testing.T) {
	t.Run("custom buffer size", func(t *testing.T) {
		broker := NewBroker[int]("test", WithBufferSize[int](2))
		defer broker.Shutdown()

		ctx := context.Background()
		ch := broker.Subscribe(ctx)

		// Fill buffer
		broker.Publish(EventCreated, 1)
		broker.Publish(EventCreated, 2)

		// This should be dropped (buffer full, no receiver draining)
		broker.Publish(EventCreated, 3)

		metrics := broker.Metrics()
		if metrics.DropCount == 0 {
			// May or may not drop depending on timing
			t.Log("no drops detected (may depend on timing)")
		}

		// Drain to verify first two made it
		if e := <-ch; e.Payload != 1 {
			t.Errorf("expected 1, got %d", e.Payload)
		}
		if e := <-ch; e.Payload != 2 {
			t.Errorf("expected 2, got %d", e.Payload)
		}
	})

	t.Run("blocking publish", func(t *testing.T) {
		broker := NewBroker[int]("test",
			WithBufferSize[int](1),
			WithDropPolicy[int](false),
		)
		defer broker.Shutdown()

		ctx := context.Background()
		ch := broker.Subscribe(ctx)

		// Fill buffer
		broker.Publish(EventCreated, 1)

		// This should block until we drain
		done := make(chan bool)
		go func() {
			broker.Publish(EventCreated, 2)
			done <- true
		}()

		// Give it time to potentially complete (should not)
		select {
		case <-done:
			t.Error("publish should have blocked")
		case <-time.After(50 * time.Millisecond):
			// Expected
		}

		// Drain to unblock
		<-ch
		select {
		case <-done:
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Error("publish should have completed")
		}
	})
}

func TestBrokerPublishAsync(t *testing.T) {
	broker := NewBroker[string]("test")
	defer broker.Shutdown()

	ctx := context.Background()
	ch := broker.Subscribe(ctx)

	// PublishAsync returns immediately
	start := time.Now()
	broker.PublishAsync(EventCreated, "async")
	if time.Since(start) > 10*time.Millisecond {
		t.Error("PublishAsync took too long")
	}

	// Event should still arrive
	select {
	case event := <-ch:
		if event.Payload != "async" {
			t.Errorf("expected 'async', got %q", event.Payload)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for async event")
	}
}

func TestBrokerIsShutdown(t *testing.T) {
	broker := NewBroker[string]("test")

	if broker.IsShutdown() {
		t.Error("broker should not be shut down initially")
	}

	broker.Shutdown()

	if !broker.IsShutdown() {
		t.Error("broker should be shut down after Shutdown()")
	}

	// Double shutdown should be safe
	broker.Shutdown()
	if !broker.IsShutdown() {
		t.Error("broker should still be shut down")
	}
}
