package models

import (
	"encoding/json"
	"time"
)

// TaskStatus represents the lifecycle state of a task.
type TaskStatus string

const (
	TaskStatusProposed   TaskStatus = "proposed"
	TaskStatusApproved   TaskStatus = "approved"
	TaskStatusAssigned   TaskStatus = "assigned"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
)

// ValidTaskTransitions defines allowed status transitions.
var ValidTaskTransitions = map[TaskStatus][]TaskStatus{
	TaskStatusProposed:   {TaskStatusApproved},
	TaskStatusApproved:   {TaskStatusAssigned},
	TaskStatusAssigned:   {TaskStatusInProgress},
	TaskStatusInProgress: {TaskStatusCompleted, TaskStatusFailed},
}

// Task represents a unit of work assigned to an agent.
type Task struct {
	ID                string          `json:"id"`
	ProjectID         string          `json:"project_id"`
	Title             string          `json:"title"`
	Description       string          `json:"description"`
	ModulePath        string          `json:"module_path"`
	Status            TaskStatus      `json:"status"`
	AssignedAgentID   *string         `json:"assigned_agent_id,omitempty"`
	BranchName        *string         `json:"branch_name,omitempty"`
	DependsOn         []string        `json:"depends_on"`
	ReadOnlyPaths     []string        `json:"read_only_paths"`
	ForbiddenPaths    []string        `json:"forbidden_paths"`
	IntegrationPoints []string        `json:"integration_points"`
	Contracts         json.RawMessage `json:"contracts,omitempty"`
	AgentBriefing     string          `json:"agent_briefing,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}
