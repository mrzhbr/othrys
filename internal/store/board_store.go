package store

import (
	"context"
	"fmt"
	"time"

	"github.com/moritzhuber/othrys/internal/models"
)

// BoardTaskCard is the per-task data needed to render a Kanban card.
type BoardTaskCard struct {
	ID              string            `json:"id"`
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	ModulePath      string            `json:"module_path"`
	Status          models.TaskStatus `json:"status"`
	AssignedAgentID *string           `json:"assigned_agent_id,omitempty"`
	AssignedAgent   *AgentSummary     `json:"assigned_agent,omitempty"`
	BranchName      *string           `json:"branch_name,omitempty"`
	DependsOn       []string          `json:"depends_on"`
	Claim           *ClaimSummary     `json:"claim,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// AgentSummary is a lightweight agent view for the board.
type AgentSummary struct {
	ID       string              `json:"id"`
	Name     string              `json:"name"`
	ToolType models.AgentToolType `json:"tool_type"`
	Status   models.AgentStatus  `json:"status"`
}

// ClaimSummary is a lightweight claim view for the board.
type ClaimSummary struct {
	ID        string            `json:"id"`
	AgentID   string            `json:"agent_id"`
	AgentName string            `json:"agent_name,omitempty"`
	Path      string            `json:"path"`
	ClaimType models.ClaimType  `json:"claim_type"`
	TaskID    string            `json:"task_id"`
}

// BoardState is the full Kanban board read-model for a project.
type BoardState struct {
	Columns      BoardColumns   `json:"columns"`
	Agents       []AgentSummary `json:"agents"`
	ActiveClaims []ClaimSummary `json:"active_claims"`
}

// BoardColumns groups tasks by their pipeline status.
type BoardColumns struct {
	Proposed   []BoardTaskCard `json:"proposed"`
	Approved   []BoardTaskCard `json:"approved"`
	Assigned   []BoardTaskCard `json:"assigned"`
	InProgress []BoardTaskCard `json:"in_progress"`
	Completed  []BoardTaskCard `json:"completed"`
	Failed     []BoardTaskCard `json:"failed"`
}

// BoardStore handles the Kanban board read-model queries.
type BoardStore struct {
	*Store
}

// NewBoardStore creates a new BoardStore.
func NewBoardStore(s *Store) *BoardStore {
	return &BoardStore{s}
}

// GetBoardState returns the full Kanban board state for a project.
// It runs 3 queries (tasks with agents, all agents, active claims) and assembles the result.
func (bs *BoardStore) GetBoardState(ctx context.Context, projectID string) (*BoardState, error) {
	tasks, err := bs.loadBoardTasks(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("load board tasks: %w", err)
	}

	agents, err := bs.loadBoardAgents(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("load board agents: %w", err)
	}

	claims, err := bs.loadBoardClaims(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("load board claims: %w", err)
	}

	// Build agent lookup
	agentByID := make(map[string]AgentSummary, len(agents))
	for _, a := range agents {
		agentByID[a.ID] = a
	}

	// Build claim lookup by task_id
	claimByTaskID := make(map[string]*ClaimSummary)
	for _, c := range claims {
		// Only attach the first claim per task (simplest card decoration)
		if _, exists := claimByTaskID[c.TaskID]; !exists {
			cc := c
			if ag, ok := agentByID[c.AgentID]; ok {
				cc.AgentName = ag.Name
			}
			claimByTaskID[c.TaskID] = &cc
		}
	}

	// Enrich task cards with agent and claim info
	for _, taskList := range [6][]BoardTaskCard{
		tasks.Proposed, tasks.Approved, tasks.Assigned,
		tasks.InProgress, tasks.Completed, tasks.Failed,
	} {
		for i := range taskList {
			t := &taskList[i]
			if t.AssignedAgentID != nil {
				if ag, ok := agentByID[*t.AssignedAgentID]; ok {
					t.AssignedAgent = &ag
				}
			}
			if c, ok := claimByTaskID[t.ID]; ok {
				t.Claim = c
			}
		}
	}

	// Build agent name into claims for the separate claims list
	enrichedClaims := make([]ClaimSummary, len(claims))
	for i, c := range claims {
		enrichedClaims[i] = c
		if c.AgentName == "" {
			if ag, ok := agentByID[c.AgentID]; ok {
				enrichedClaims[i].AgentName = ag.Name
			}
		}
	}

	return &BoardState{
		Columns:      tasks,
		Agents:       agents,
		ActiveClaims: enrichedClaims,
	}, nil
}

// loadBoardTasks loads all tasks for the project and groups them by status.
func (bs *BoardStore) loadBoardTasks(ctx context.Context, projectID string) (BoardColumns, error) {
	rows, err := bs.Pool.Query(ctx, `
		SELECT id, project_id, title, description, module_path, status,
		       assigned_agent_id, branch_name, depends_on,
		       read_only_paths, forbidden_paths, integration_points, contracts, agent_briefing,
		       created_at, updated_at
		FROM tasks
		WHERE project_id = $1
		ORDER BY created_at ASC
	`, projectID)
	if err != nil {
		return BoardColumns{}, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	cols := BoardColumns{
		Proposed:   make([]BoardTaskCard, 0),
		Approved:   make([]BoardTaskCard, 0),
		Assigned:   make([]BoardTaskCard, 0),
		InProgress: make([]BoardTaskCard, 0),
		Completed:  make([]BoardTaskCard, 0),
		Failed:     make([]BoardTaskCard, 0),
	}

	for rows.Next() {
		var t models.Task
		if err := rows.Scan(
			&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.ModulePath, &t.Status,
			&t.AssignedAgentID, &t.BranchName, &t.DependsOn,
			&t.ReadOnlyPaths, &t.ForbiddenPaths, &t.IntegrationPoints, &t.Contracts, &t.AgentBriefing,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return BoardColumns{}, fmt.Errorf("scan task: %w", err)
		}

		card := BoardTaskCard{
			ID:              t.ID,
			Title:           t.Title,
			Description:     t.Description,
			ModulePath:      t.ModulePath,
			Status:          t.Status,
			AssignedAgentID: t.AssignedAgentID,
			BranchName:      t.BranchName,
			DependsOn:       t.DependsOn,
			CreatedAt:       t.CreatedAt,
			UpdatedAt:       t.UpdatedAt,
		}

		switch t.Status {
		case models.TaskStatusProposed:
			cols.Proposed = append(cols.Proposed, card)
		case models.TaskStatusApproved:
			cols.Approved = append(cols.Approved, card)
		case models.TaskStatusAssigned:
			cols.Assigned = append(cols.Assigned, card)
		case models.TaskStatusInProgress:
			cols.InProgress = append(cols.InProgress, card)
		case models.TaskStatusCompleted:
			cols.Completed = append(cols.Completed, card)
		case models.TaskStatusFailed:
			cols.Failed = append(cols.Failed, card)
		}
	}

	return cols, rows.Err()
}

// loadBoardAgents loads all agents for the project.
func (bs *BoardStore) loadBoardAgents(ctx context.Context, projectID string) ([]AgentSummary, error) {
	rows, err := bs.Pool.Query(ctx, `
		SELECT id, name, tool_type, status
		FROM agents
		WHERE project_id = $1
		ORDER BY created_at ASC
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("query agents: %w", err)
	}
	defer rows.Close()

	var agents []AgentSummary
	for rows.Next() {
		var a AgentSummary
		if err := rows.Scan(&a.ID, &a.Name, &a.ToolType, &a.Status); err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

// loadBoardClaims loads all active claims for the project.
func (bs *BoardStore) loadBoardClaims(ctx context.Context, projectID string) ([]ClaimSummary, error) {
	rows, err := bs.Pool.Query(ctx, `
		SELECT c.id, c.agent_id, c.path, c.claim_type, c.task_id
		FROM claims c
		WHERE c.project_id = $1 AND c.status = 'active'
		ORDER BY c.granted_at ASC
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("query claims: %w", err)
	}
	defer rows.Close()

	// Use pgx.ErrNoRows sentinel check if needed
	var claims []ClaimSummary
	for rows.Next() {
		var c ClaimSummary
		if err := rows.Scan(&c.ID, &c.AgentID, &c.Path, &c.ClaimType, &c.TaskID); err != nil {
			return nil, fmt.Errorf("scan claim: %w", err)
		}
		claims = append(claims, c)
	}
	return claims, rows.Err()
}
