package coordinator

import (
	"github.com/moritzhuber/othrys/internal/events"
	"github.com/moritzhuber/othrys/internal/store"
)

// Coordinator bundles all coordination logic and stores.
type Coordinator struct {
	Projects *store.ProjectStore
	Tasks    *store.TaskStore
	Agents   *store.AgentStore
	Claims   *store.ClaimStore
	Events   *store.EventStore
	Bus      events.EventBus
}

// New creates a new Coordinator.
func New(
	projects *store.ProjectStore,
	tasks *store.TaskStore,
	agents *store.AgentStore,
	claims *store.ClaimStore,
	evts *store.EventStore,
	bus events.EventBus,
) *Coordinator {
	return &Coordinator{
		Projects: projects,
		Tasks:    tasks,
		Agents:   agents,
		Claims:   claims,
		Events:   evts,
		Bus:      bus,
	}
}
