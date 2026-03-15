package events

import (
	"context"
	"sync"

	"github.com/moritzhuber/othrys/internal/models"
)

// MockBus is an in-memory EventBus implementation for testing.
// It records all published events and allows test assertions.
type MockBus struct {
	mu       sync.Mutex
	Events   []models.Event
	handlers map[string][]func(models.Event)
}

// NewMockBus creates a new MockBus.
func NewMockBus() *MockBus {
	return &MockBus{
		handlers: make(map[string][]func(models.Event)),
	}
}

// Publish records the event and dispatches to any registered handlers.
func (m *MockBus) Publish(_ context.Context, event models.Event) error {
	m.mu.Lock()
	m.Events = append(m.Events, event)
	channel := "othrys_" + event.ProjectID
	hs := m.handlers[channel]
	m.mu.Unlock()

	for _, h := range hs {
		h(event)
	}
	return nil
}

// Subscribe registers a handler for the given channel.
func (m *MockBus) Subscribe(channel string, handler func(models.Event)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[channel] = append(m.handlers[channel], handler)
	return nil
}

// Close is a no-op for the mock.
func (m *MockBus) Close() error {
	return nil
}

// Reset clears all recorded events (useful between tests).
func (m *MockBus) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = nil
}

// EventsOfType returns all recorded events of a given type.
func (m *MockBus) EventsOfType(eventType string) []models.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []models.Event
	for _, e := range m.Events {
		if e.EventType == eventType {
			result = append(result, e)
		}
	}
	return result
}
