package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/moritzhuber/othrys/internal/models"
)

// AgentStore handles database operations for agents.
type AgentStore struct {
	*Store
}

// NewAgentStore creates a new AgentStore.
func NewAgentStore(s *Store) *AgentStore {
	return &AgentStore{s}
}

// Register upserts an agent by name+project. Returns the existing or newly created agent.
// This is idempotent: same name returns the same agent.
func (as *AgentStore) Register(ctx context.Context, projectID, name string, toolType models.AgentToolType) (*models.Agent, error) {
	now := time.Now().UTC()
	var a models.Agent
	err := as.Pool.QueryRow(ctx, `
		INSERT INTO agents (project_id, name, tool_type, status, last_heartbeat, connected_at, created_at)
		VALUES ($1, $2, $3, 'idle', $4, $4, $4)
		ON CONFLICT (project_id, name) DO UPDATE
		  SET status = 'idle', last_heartbeat = $4, connected_at = $4
		RETURNING id, project_id, name, tool_type, status, branch_name,
		          last_heartbeat, connected_at, created_at
	`, projectID, name, toolType, now).Scan(
		&a.ID, &a.ProjectID, &a.Name, &a.ToolType, &a.Status, &a.BranchName,
		&a.LastHeartbeat, &a.ConnectedAt, &a.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("register agent: %w", err)
	}
	return &a, nil
}

// GetByID retrieves an agent by UUID.
func (as *AgentStore) GetByID(ctx context.Context, id string) (*models.Agent, error) {
	var a models.Agent
	err := as.Pool.QueryRow(ctx, `
		SELECT id, project_id, name, tool_type, status, branch_name,
		       last_heartbeat, connected_at, created_at
		FROM agents WHERE id = $1
	`, id).Scan(
		&a.ID, &a.ProjectID, &a.Name, &a.ToolType, &a.Status, &a.BranchName,
		&a.LastHeartbeat, &a.ConnectedAt, &a.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}
	return &a, nil
}

// UpdateHeartbeat refreshes the agent's last_heartbeat timestamp and sets status to idle/working.
func (as *AgentStore) UpdateHeartbeat(ctx context.Context, id string) error {
	now := time.Now().UTC()
	ct, err := as.Pool.Exec(ctx, `
		UPDATE agents SET last_heartbeat = $1 WHERE id = $2
	`, now, id)
	if err != nil {
		return fmt.Errorf("update heartbeat: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("agent not found: %s", id)
	}
	return nil
}

// UpdateStatus changes an agent's status.
func (as *AgentStore) UpdateStatus(ctx context.Context, id string, status models.AgentStatus) error {
	_, err := as.Pool.Exec(ctx, `
		UPDATE agents SET status = $1 WHERE id = $2
	`, status, id)
	return err
}

// ListByProject returns all agents for a project.
func (as *AgentStore) ListByProject(ctx context.Context, projectID string) ([]*models.Agent, error) {
	rows, err := as.Pool.Query(ctx, `
		SELECT id, project_id, name, tool_type, status, branch_name,
		       last_heartbeat, connected_at, created_at
		FROM agents WHERE project_id = $1 ORDER BY created_at ASC
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	var agents []*models.Agent
	for rows.Next() {
		var a models.Agent
		if err := rows.Scan(
			&a.ID, &a.ProjectID, &a.Name, &a.ToolType, &a.Status, &a.BranchName,
			&a.LastHeartbeat, &a.ConnectedAt, &a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		agents = append(agents, &a)
	}
	return agents, rows.Err()
}

// ListStale returns agents whose last_heartbeat is older than the given threshold.
func (as *AgentStore) ListStale(ctx context.Context, projectID string, before time.Time) ([]*models.Agent, error) {
	rows, err := as.Pool.Query(ctx, `
		SELECT id, project_id, name, tool_type, status, branch_name,
		       last_heartbeat, connected_at, created_at
		FROM agents
		WHERE project_id = $1 AND status != 'disconnected' AND last_heartbeat < $2
	`, projectID, before)
	if err != nil {
		return nil, fmt.Errorf("list stale agents: %w", err)
	}
	defer rows.Close()

	var agents []*models.Agent
	for rows.Next() {
		var a models.Agent
		if err := rows.Scan(
			&a.ID, &a.ProjectID, &a.Name, &a.ToolType, &a.Status, &a.BranchName,
			&a.LastHeartbeat, &a.ConnectedAt, &a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan stale agent: %w", err)
		}
		agents = append(agents, &a)
	}
	return agents, rows.Err()
}
