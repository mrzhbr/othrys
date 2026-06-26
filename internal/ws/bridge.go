package ws

import (
	"encoding/json"
	"log"

	"github.com/moritzhuber/othrys/internal/events"
	"github.com/moritzhuber/othrys/internal/models"
)

// Bridge subscribes to the EventBus and routes events to the WebSocket Hub for broadcast.
// It does NOT use PG LISTEN/NOTIFY directly — all event transport is through the EventBus interface.
type Bridge struct {
	bus events.EventBus
	hub *Hub
}

// NewBridge creates a new Bridge.
func NewBridge(bus events.EventBus, hub *Hub) *Bridge {
	return &Bridge{bus: bus, hub: hub}
}

// Start subscribes to all relevant event channels and begins routing events to the hub.
// It must be called with a project ID to subscribe to that project's event channel.
func (b *Bridge) SubscribeProject(projectID string) error {
	channel := "othrys_" + projectID
	return b.bus.Subscribe(channel, func(event models.Event) {
		b.routeEvent(event)
	})
}

// routeEvent converts a models.Event to a WebSocket OutboundMessage and broadcasts it.
func (b *Bridge) routeEvent(event models.Event) {
	var msgType string

	switch event.EventType {
	case models.EventTypeTaskAssigned:
		msgType = MsgTypeTaskAssigned
	case models.EventTypeTaskApproved:
		msgType = MsgTypeTaskApproved
	case models.EventTypeTaskStatusChange:
		msgType = MsgTypeTaskStatusChanged
	case models.EventTypeClaimGranted:
		msgType = MsgTypeClaimGranted
	case models.EventTypeClaimDenied:
		msgType = MsgTypeClaimConflict
	case models.EventTypeClaimReleased:
		msgType = MsgTypeClaimReleased
	case models.EventTypeMergeReady:
		msgType = MsgTypeMergeReady
	default:
		log.Printf("[bridge] unhandled event type %q for project %s", event.EventType, event.ProjectID)
		return
	}

	msg := OutboundMessage{
		Type:    msgType,
		Payload: event.Payload,
	}

	// For task_assigned, send directly to the assigned agent instead of broadcasting
	if event.EventType == models.EventTypeTaskAssigned {
		var payload struct {
			AgentID string `json:"agent_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err == nil && payload.AgentID != "" {
			b.hub.SendToAgent(payload.AgentID, msg)
			return
		}
	}

	b.hub.Broadcast(event.ProjectID, msg)
}
