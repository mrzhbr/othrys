package models

import "time"

// ClaimType represents the type of access being claimed.
type ClaimType string

const (
	ClaimTypeExclusive  ClaimType = "exclusive"
	ClaimTypeSharedRead ClaimType = "shared_read"
)

// ClaimStatus represents the lifecycle state of a claim.
type ClaimStatus string

const (
	ClaimStatusActive   ClaimStatus = "active"
	ClaimStatusReleased ClaimStatus = "released"
	ClaimStatusRevoked  ClaimStatus = "revoked"
)

// Claim represents an agent's lock on a module/file path.
type Claim struct {
	ID         string      `json:"id"`
	ProjectID  string      `json:"project_id"`
	AgentID    string      `json:"agent_id"`
	TaskID     string      `json:"task_id"`
	Path       string      `json:"path"`
	ClaimType  ClaimType   `json:"claim_type"`
	Status     ClaimStatus `json:"status"`
	GrantedAt  time.Time   `json:"granted_at"`
	ReleasedAt *time.Time  `json:"released_at,omitempty"`
}
