package models

import "time"

// AgentToolType represents the type of AI coding tool the agent uses.
type AgentToolType string

const (
	AgentToolOmo     AgentToolType = "omo"
	AgentToolCursor  AgentToolType = "cursor"
	AgentToolCopilot AgentToolType = "copilot"
	AgentToolGeneric AgentToolType = "generic"
)

// AgentStatus represents the connectivity state of an agent.
type AgentStatus string

const (
	AgentStatusIdle         AgentStatus = "idle"
	AgentStatusWorking      AgentStatus = "working"
	AgentStatusDisconnected AgentStatus = "disconnected"
)

// Agent represents a connected coding assistant agent.
type Agent struct {
	ID            string        `json:"id"`
	ProjectID     string        `json:"project_id"`
	Name          string        `json:"name"`
	ToolType      AgentToolType `json:"tool_type"`
	Status        AgentStatus   `json:"status"`
	BranchName    *string       `json:"branch_name,omitempty"`
	LastHeartbeat time.Time     `json:"last_heartbeat"`
	ConnectedAt   time.Time     `json:"connected_at"`
	CreatedAt     time.Time     `json:"created_at"`
}
