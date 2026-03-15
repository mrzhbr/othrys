package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/moritzhuber/othrys/internal/events"
	"github.com/moritzhuber/othrys/internal/models"
)

// EventStore handles database operations for events.
// Events are inserted into the DB and then published through the EventBus.
// The store does NOT call pg_notify directly — that is the EventBus implementation's responsibility.
type EventStore struct {
	*Store
}

// NewEventStore creates a new EventStore.
func NewEventStore(s *Store) *EventStore {
	return &EventStore{s}
}

// Create inserts an event into the database and publishes it through the provided EventBus.
func (es *EventStore) Create(ctx context.Context, event *models.Event, bus events.EventBus) (*models.Event, error) {
	if event.Payload == nil {
		event.Payload = json.RawMessage("{}")
	}

	now := time.Now().UTC()
	err := es.Pool.QueryRow(ctx, `
		INSERT INTO events (project_id, event_type, agent_id, payload, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, project_id, event_type, agent_id, payload, created_at
	`, event.ProjectID, event.EventType, event.AgentID, event.Payload, now).Scan(
		&event.ID, &event.ProjectID, &event.EventType, &event.AgentID,
		&event.Payload, &event.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create event: %w", err)
	}

	// Publish through EventBus (not direct pg_notify)
	if bus != nil {
		if err := bus.Publish(ctx, *event); err != nil {
			// Log but don't fail — event is already persisted
			fmt.Printf("[event_store] publish event %d failed: %v\n", event.ID, err)
		}
	}

	return event, nil
}

// ListByProject returns paginated events for a project, newest first.
func (es *EventStore) ListByProject(ctx context.Context, projectID string, limit, offset int) ([]*models.Event, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := es.Pool.Query(ctx, `
		SELECT id, project_id, event_type, agent_id, payload, created_at
		FROM events
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, projectID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var evts []*models.Event
	for rows.Next() {
		var e models.Event
		if err := rows.Scan(
			&e.ID, &e.ProjectID, &e.EventType, &e.AgentID,
			&e.Payload, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		evts = append(evts, &e)
	}
	return evts, rows.Err()
}
