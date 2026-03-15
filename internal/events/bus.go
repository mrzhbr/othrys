package events

import (
	"context"

	"github.com/moritzhuber/othrys/internal/models"
)

// EventBus is the interface for publishing and subscribing to system events.
// The MVP implementation uses PostgreSQL LISTEN/NOTIFY (pgbus.go).
// Future implementations could use Redis pub/sub, NATS, or Kafka.
type EventBus interface {
	// Publish broadcasts an event to all subscribers on the event's project channel.
	Publish(ctx context.Context, event models.Event) error

	// Subscribe registers a handler for events on the given channel.
	// The handler is called in a goroutine for each received event.
	Subscribe(channel string, handler func(models.Event)) error

	// Close stops all listeners and releases resources.
	Close() error
}
