// Package pubsub provides a type-safe pub/sub broker implementation.
package pubsub

import (
	"context"
	"time"
)

// EventType represents the type of event.
type EventType string

// Standard event types.
const (
	EventCreated   EventType = "created"
	EventUpdated   EventType = "updated"
	EventDeleted   EventType = "deleted"
	EventStarted   EventType = "started"
	EventCompleted EventType = "completed"
	EventFailed    EventType = "failed"
	EventProgress  EventType = "progress"
)

// Event represents a typed event with metadata.
type Event[T any] struct { //nolint:govet // fieldalignment: preserving logical field order
	Type      EventType
	Payload   T
	Timestamp time.Time
}

// Publisher is the interface for publishing events.
type Publisher[T any] interface {
	Publish(EventType, T)
	PublishAsync(EventType, T)
}

// Subscriber is the interface for subscribing to events.
type Subscriber[T any] interface {
	Subscribe(context.Context) <-chan Event[T]
}

// PubSub combines Publisher and Subscriber interfaces.
type PubSub[T any] interface {
	Publisher[T]
	Subscriber[T]
}
