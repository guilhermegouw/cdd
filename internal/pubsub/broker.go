package pubsub

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultBufferSize is the default channel buffer for subscribers.
const DefaultBufferSize = 64

// BrokerOption configures a Broker.
type BrokerOption[T any] func(*Broker[T])

// WithBufferSize sets the subscriber channel buffer size.
func WithBufferSize[T any](size int) BrokerOption[T] {
	return func(b *Broker[T]) {
		b.bufferSize = size
	}
}

// WithDropPolicy sets whether to drop events when subscriber is full.
func WithDropPolicy[T any](drop bool) BrokerOption[T] {
	return func(b *Broker[T]) {
		b.dropOnFull = drop
	}
}

// Broker is a type-safe pub/sub broker using Go generics.
// It is thread-safe and supports context-based subscription lifecycle.
type Broker[T any] struct { //nolint:govet // fieldalignment: preserving logical field order
	name       string
	subs       map[chan Event[T]]struct{}
	mu         sync.RWMutex
	done       chan struct{}
	bufferSize int
	dropOnFull bool

	// Metrics (atomic for lock-free reads)
	publishCount   atomic.Int64
	dropCount      atomic.Int64
	subscriberPeak atomic.Int32
	subscriberCurr atomic.Int32
}

// NewBroker creates a new typed broker with optional configuration.
func NewBroker[T any](name string, opts ...BrokerOption[T]) *Broker[T] {
	b := &Broker[T]{
		name:       name,
		subs:       make(map[chan Event[T]]struct{}),
		done:       make(chan struct{}),
		bufferSize: DefaultBufferSize,
		dropOnFull: true, // Default: non-blocking
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// Name returns the broker's name for debugging.
func (b *Broker[T]) Name() string {
	return b.name
}

// Subscribe creates a new subscription that receives events until context is cancelled.
// The returned channel is closed when the context is done or the broker shuts down.
func (b *Broker[T]) Subscribe(ctx context.Context) <-chan Event[T] {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check if broker is already shut down
	select {
	case <-b.done:
		ch := make(chan Event[T])
		close(ch)
		return ch
	default:
	}

	sub := make(chan Event[T], b.bufferSize)
	b.subs[sub] = struct{}{}

	// Update subscriber metrics
	curr := b.subscriberCurr.Add(1)
	for {
		peak := b.subscriberPeak.Load()
		if curr <= peak || b.subscriberPeak.CompareAndSwap(peak, curr) {
			break
		}
	}

	// Cleanup goroutine
	go func() {
		select {
		case <-ctx.Done():
		case <-b.done:
		}

		b.mu.Lock()
		defer b.mu.Unlock()

		// Check if already removed (during shutdown)
		if _, ok := b.subs[sub]; !ok {
			return
		}

		delete(b.subs, sub)
		close(sub)
		b.subscriberCurr.Add(-1)
	}()

	return sub
}

// Publish sends an event to all subscribers.
// If dropOnFull is true (default), events are dropped for slow subscribers.
// If dropOnFull is false, this blocks until all subscribers receive the event.
func (b *Broker[T]) Publish(eventType EventType, payload T) {
	b.mu.RLock()

	// Check if broker is shut down
	select {
	case <-b.done:
		b.mu.RUnlock()
		return
	default:
	}

	// Snapshot subscribers for lock-free publishing
	subscribers := make([]chan Event[T], 0, len(b.subs))
	for sub := range b.subs {
		subscribers = append(subscribers, sub)
	}
	b.mu.RUnlock()

	if len(subscribers) == 0 {
		return
	}

	event := Event[T]{
		Type:      eventType,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	b.publishCount.Add(1)

	for _, sub := range subscribers {
		if b.dropOnFull {
			select {
			case sub <- event:
			default:
				b.dropCount.Add(1)
			}
		} else {
			sub <- event // Blocking
		}
	}
}

// PublishAsync publishes an event asynchronously and returns immediately.
func (b *Broker[T]) PublishAsync(eventType EventType, payload T) {
	go b.Publish(eventType, payload)
}

// Shutdown gracefully shuts down the broker.
// All subscriber channels are closed and pending events are dropped.
func (b *Broker[T]) Shutdown() {
	select {
	case <-b.done:
		return // Already shut down
	default:
		close(b.done)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for ch := range b.subs {
		delete(b.subs, ch)
		close(ch)
	}
	b.subscriberCurr.Store(0)
}

// IsShutdown returns true if the broker has been shut down.
func (b *Broker[T]) IsShutdown() bool {
	select {
	case <-b.done:
		return true
	default:
		return false
	}
}

// SubscriberCount returns the current number of subscribers.
func (b *Broker[T]) SubscriberCount() int {
	return int(b.subscriberCurr.Load())
}

// Metrics returns the broker's metrics for debugging.
func (b *Broker[T]) Metrics() BrokerMetrics {
	return BrokerMetrics{
		Name:            b.name,
		PublishCount:    b.publishCount.Load(),
		DropCount:       b.dropCount.Load(),
		SubscriberCount: int(b.subscriberCurr.Load()),
		SubscriberPeak:  int(b.subscriberPeak.Load()),
	}
}

// BrokerMetrics contains broker statistics for debugging.
type BrokerMetrics struct {
	Name            string
	PublishCount    int64
	DropCount       int64
	SubscriberCount int
	SubscriberPeak  int
}
