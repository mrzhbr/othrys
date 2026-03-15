package ws

import "encoding/json"

// Message types for server → client
const (
	MsgTypeTaskAssigned       = "task_assigned"
	MsgTypeClaimGranted       = "claim_granted"
	MsgTypeClaimConflict      = "claim_conflict"
	MsgTypeClaimReleased      = "claim_released"
	MsgTypeTaskStatusChanged  = "task_status_changed"
	MsgTypeMergeReady         = "merge_ready"
	MsgTypeClaimsSnapshot     = "claims_snapshot"
	MsgTypePing               = "ping"
	MsgTypeError              = "error"
)

// Message types for client → server
const (
	MsgTypeClaimRequest  = "claim_request"
	MsgTypeTaskUpdate    = "task_update"
	MsgTypeClaimsSync    = "claims_sync"
	MsgTypeHeartbeat     = "heartbeat"
	MsgTypePong          = "pong"
)

// InboundMessage is a generic inbound WebSocket message from a client.
type InboundMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// OutboundMessage is a generic outbound WebSocket message to a client.
type OutboundMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

// ClaimRequestPayload is the payload for a "claim_request" client message.
type ClaimRequestPayload struct {
	TaskID    string `json:"task_id"`
	Path      string `json:"path"`
	ClaimType string `json:"claim_type"` // "exclusive" or "shared_read"
}

// TaskUpdatePayload is the payload for a "task_update" client message.
type TaskUpdatePayload struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

// ClaimsSyncPayload is the payload for a "claims_sync" client message.
type ClaimsSyncPayload struct {
	SinceTimestamp string `json:"since_timestamp,omitempty"`
}

// ClaimSnapshotItem is a single claim in a "claims_snapshot" server message.
type ClaimSnapshotItem struct {
	ID        string `json:"id"`
	AgentID   string `json:"agent_id"`
	AgentName string `json:"agent_name"`
	Path      string `json:"path"`
	ClaimType string `json:"claim_type"`
	Status    string `json:"status"`
}
