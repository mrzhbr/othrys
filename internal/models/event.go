package models

import (
	"encoding/json"
	"time"
)

// Event represents a system event stored in the events table and published via EventBus.
type Event struct {
	ID        int64           `json:"id"`
	ProjectID string          `json:"project_id"`
	EventType string          `json:"event_type"`
	AgentID   *string         `json:"agent_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

// Common event types
const (
	EventTypeTaskCreated      = "task_created"
	EventTypeTaskApproved     = "task_approved"
	EventTypeTaskAssigned     = "task_assigned"
	EventTypeTaskStatusChange = "task_status_changed"
	EventTypeClaimGranted     = "claim_granted"
	EventTypeClaimDenied      = "claim_denied"
	EventTypeClaimReleased    = "claim_released"
	EventTypeClaimRevoked     = "claim_revoked"
	EventTypeAgentDisconnect  = "agent_disconnected"
	EventTypeMergeReady       = "merge_ready"
)
