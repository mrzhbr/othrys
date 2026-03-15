package events

import (
	"context"
	"testing"

	"github.com/moritzhuber/othrys/internal/models"
)

// TestMockBusImplementsInterface verifies MockBus satisfies the EventBus interface.
func TestMockBusImplementsInterface(t *testing.T) {
	var _ EventBus = NewMockBus()
}

// TestMockBusPublishRecords tests that published events are recorded.
func TestMockBusPublishRecords(t *testing.T) {
	bus := NewMockBus()
	ctx := context.Background()

	event1 := models.Event{
		ProjectID: "proj-1",
		EventType: models.EventTypeTaskCreated,
		Payload:   []byte(`{"task_id":"t1"}`),
	}
	event2 := models.Event{
		ProjectID: "proj-1",
		EventType: models.EventTypeClaimGranted,
		Payload:   []byte(`{"claim_id":"c1"}`),
	}

	if err := bus.Publish(ctx, event1); err != nil {
		t.Fatalf("Publish event1: %v", err)
	}
	if err := bus.Publish(ctx, event2); err != nil {
		t.Fatalf("Publish event2: %v", err)
	}

	if len(bus.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(bus.Events))
	}
}

// TestMockBusEventsOfType tests filtering events by type.
func TestMockBusEventsOfType(t *testing.T) {
	bus := NewMockBus()
	ctx := context.Background()

	_ = bus.Publish(ctx, models.Event{ProjectID: "p1", EventType: models.EventTypeTaskCreated})
	_ = bus.Publish(ctx, models.Event{ProjectID: "p1", EventType: models.EventTypeClaimGranted})
	_ = bus.Publish(ctx, models.Event{ProjectID: "p1", EventType: models.EventTypeTaskCreated})

	taskEvents := bus.EventsOfType(models.EventTypeTaskCreated)
	if len(taskEvents) != 2 {
		t.Errorf("expected 2 task_created events, got %d", len(taskEvents))
	}

	claimEvents := bus.EventsOfType(models.EventTypeClaimGranted)
	if len(claimEvents) != 1 {
		t.Errorf("expected 1 claim_granted event, got %d", len(claimEvents))
	}
}

// TestMockBusReset tests that Reset clears all events.
func TestMockBusReset(t *testing.T) {
	bus := NewMockBus()
	ctx := context.Background()

	_ = bus.Publish(ctx, models.Event{ProjectID: "p1", EventType: models.EventTypeTaskCreated})
	bus.Reset()

	if len(bus.Events) != 0 {
		t.Errorf("expected 0 events after reset, got %d", len(bus.Events))
	}
}

// TestMockBusSubscribe tests that handlers receive published events.
func TestMockBusSubscribe(t *testing.T) {
	bus := NewMockBus()
	ctx := context.Background()

	received := make(chan models.Event, 5)
	err := bus.Subscribe("othrys_proj-1", func(e models.Event) {
		received <- e
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	event := models.Event{
		ProjectID: "proj-1",
		EventType: models.EventTypeTaskCreated,
	}
	_ = bus.Publish(ctx, event)

	select {
	case got := <-received:
		if got.EventType != models.EventTypeTaskCreated {
			t.Errorf("expected task_created, got %s", got.EventType)
		}
	default:
		t.Error("handler was not called after Publish")
	}
}

// TestMockBusClose tests that Close is a no-op.
func TestMockBusClose(t *testing.T) {
	bus := NewMockBus()
	if err := bus.Close(); err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}
