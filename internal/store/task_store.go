package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/moritzhuber/othrys/internal/models"
)

// TaskStore handles database operations for tasks.
type TaskStore struct {
	*Store
}

// NewTaskStore creates a new TaskStore.
func NewTaskStore(s *Store) *TaskStore {
	return &TaskStore{s}
}

// Create inserts a new task.
func (ts *TaskStore) Create(ctx context.Context, t *models.Task) (*models.Task, error) {
	now := time.Now().UTC()
	if t.DependsOn == nil {
		t.DependsOn = []string{}
	}
	if t.ReadOnlyPaths == nil {
		t.ReadOnlyPaths = []string{}
	}
	if t.ForbiddenPaths == nil {
		t.ForbiddenPaths = []string{}
	}
	if t.IntegrationPoints == nil {
		t.IntegrationPoints = []string{}
	}
	contracts := t.Contracts
	if contracts == nil {
		contracts = json.RawMessage("[]")
	}

	var out models.Task
	err := ts.Pool.QueryRow(ctx, `
		INSERT INTO tasks (
			project_id, title, description, module_path, status, depends_on,
			read_only_paths, forbidden_paths, integration_points, contracts, agent_briefing,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12)
		RETURNING id, project_id, title, description, module_path, status,
		          assigned_agent_id, branch_name, depends_on,
		          read_only_paths, forbidden_paths, integration_points, contracts, agent_briefing,
		          created_at, updated_at
	`, t.ProjectID, t.Title, t.Description, t.ModulePath, t.Status, t.DependsOn,
		t.ReadOnlyPaths, t.ForbiddenPaths, t.IntegrationPoints, contracts, t.AgentBriefing, now).Scan(
		&out.ID, &out.ProjectID, &out.Title, &out.Description, &out.ModulePath, &out.Status,
		&out.AssignedAgentID, &out.BranchName, &out.DependsOn,
		&out.ReadOnlyPaths, &out.ForbiddenPaths, &out.IntegrationPoints, &out.Contracts, &out.AgentBriefing,
		&out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return &out, nil
}

// GetByID retrieves a task by UUID.
func (ts *TaskStore) GetByID(ctx context.Context, id string) (*models.Task, error) {
	var t models.Task
	err := ts.Pool.QueryRow(ctx, `
		SELECT id, project_id, title, description, module_path, status,
		       assigned_agent_id, branch_name, depends_on,
		       read_only_paths, forbidden_paths, integration_points, contracts, agent_briefing,
		       created_at, updated_at
		FROM tasks WHERE id = $1
	`, id).Scan(
		&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.ModulePath, &t.Status,
		&t.AssignedAgentID, &t.BranchName, &t.DependsOn,
		&t.ReadOnlyPaths, &t.ForbiddenPaths, &t.IntegrationPoints, &t.Contracts, &t.AgentBriefing,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	return &t, nil
}

// ListByProject returns tasks for a project, optionally filtered by status.
func (ts *TaskStore) ListByProject(ctx context.Context, projectID string, status *models.TaskStatus) ([]*models.Task, error) {
	query := `
		SELECT id, project_id, title, description, module_path, status,
		       assigned_agent_id, branch_name, depends_on,
		       read_only_paths, forbidden_paths, integration_points, contracts, agent_briefing,
		       created_at, updated_at
		FROM tasks WHERE project_id = $1`
	args := []any{projectID}

	if status != nil {
		query += ` AND status = $2`
		args = append(args, *status)
	}
	query += ` ORDER BY created_at ASC`

	rows, err := ts.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(
			&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.ModulePath, &t.Status,
			&t.AssignedAgentID, &t.BranchName, &t.DependsOn,
			&t.ReadOnlyPaths, &t.ForbiddenPaths, &t.IntegrationPoints, &t.Contracts, &t.AgentBriefing,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, &t)
	}
	return tasks, rows.Err()
}

// ListByProjectMultiStatus returns tasks for a project filtered by a slice of statuses.
// Uses ANY($2) for efficient multi-status filtering.
// This is used by the briefing assembler to load sibling tasks in assigned or in_progress state.
func (ts *TaskStore) ListByProjectMultiStatus(ctx context.Context, projectID string, statuses []models.TaskStatus) ([]*models.Task, error) {
	rows, err := ts.Pool.Query(ctx, `
		SELECT id, project_id, title, description, module_path, status,
		       assigned_agent_id, branch_name, depends_on,
		       read_only_paths, forbidden_paths, integration_points, contracts, agent_briefing,
		       created_at, updated_at
		FROM tasks
		WHERE project_id = $1 AND status = ANY($2)
		ORDER BY created_at ASC
	`, projectID, statuses)
	if err != nil {
		return nil, fmt.Errorf("list tasks by multi-status: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(
			&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.ModulePath, &t.Status,
			&t.AssignedAgentID, &t.BranchName, &t.DependsOn,
			&t.ReadOnlyPaths, &t.ForbiddenPaths, &t.IntegrationPoints, &t.Contracts, &t.AgentBriefing,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan task (multi-status): %w", err)
		}
		tasks = append(tasks, &t)
	}
	return tasks, rows.Err()
}

// UpdateAgentBriefing updates only the agent_briefing column for a task.
// Called after AssignTask to store the assembled briefing.
func (ts *TaskStore) UpdateAgentBriefing(ctx context.Context, taskID string, briefing string) error {
	now := time.Now().UTC()
	ct, err := ts.Pool.Exec(ctx, `
		UPDATE tasks SET agent_briefing = $1, updated_at = $2 WHERE id = $3
	`, briefing, now, taskID)
	if err != nil {
		return fmt.Errorf("update agent briefing: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return nil
}

// UpdateStatus changes a task's status. Validates the transition is allowed.
func (ts *TaskStore) UpdateStatus(ctx context.Context, id string, newStatus models.TaskStatus) error {
	// Fetch current status to validate transition
	current, err := ts.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if current == nil {
		return fmt.Errorf("task not found: %s", id)
	}

	allowed := models.ValidTaskTransitions[current.Status]
	valid := false
	for _, s := range allowed {
		if s == newStatus {
			valid = true
			break
		}
	}
	// Allow same-status for idempotency on some transitions
	if !valid && current.Status != newStatus {
		return fmt.Errorf("invalid status transition %s → %s", current.Status, newStatus)
	}

	now := time.Now().UTC()
	_, err = ts.Pool.Exec(ctx, `
		UPDATE tasks SET status = $1, updated_at = $2 WHERE id = $3
	`, newStatus, now, id)
	return err
}

// Assign sets the agent and branch for a task, transitioning it to "assigned".
func (ts *TaskStore) Assign(ctx context.Context, taskID, agentID, branchName string) error {
	now := time.Now().UTC()
	ct, err := ts.Pool.Exec(ctx, `
		UPDATE tasks
		SET assigned_agent_id = $1, branch_name = $2, status = 'assigned', updated_at = $3
		WHERE id = $4 AND status = 'approved'
	`, agentID, branchName, now, taskID)
	if err != nil {
		return fmt.Errorf("assign task: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("task %s not found or not in approved status", taskID)
	}
	return nil
}

// Delete removes a task (only proposed tasks can be deleted/rejected).
func (ts *TaskStore) Delete(ctx context.Context, id string) error {
	_, err := ts.Pool.Exec(ctx, `DELETE FROM tasks WHERE id = $1 AND status = 'proposed'`, id)
	return err
}
