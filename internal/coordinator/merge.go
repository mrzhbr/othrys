package coordinator

import (
	"context"
	"fmt"

	"github.com/moritzhuber/othrys/internal/models"
)

// MergeReadinessReport contains the server-side merge readiness status.
// This is purely database-driven — no git operations are performed here.
type MergeReadinessReport struct {
	Ready        bool           `json:"ready"`
	AllCompleted bool           `json:"all_tasks_completed"`
	PendingTasks []*models.Task `json:"pending_tasks"`
	ActiveClaims []*models.Claim `json:"active_claims"`
	Branches     []string       `json:"branches"`
}

// CheckMergeReadiness returns the server-side merge readiness for a project.
// It checks task and claim status only — no git dependency.
func (c *Coordinator) CheckMergeReadiness(ctx context.Context, projectID string) (*MergeReadinessReport, error) {
	// Get all tasks for the project
	tasks, err := c.Tasks.ListByProject(ctx, projectID, nil)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	var pendingTasks []*models.Task
	var branches []string
	seenBranches := map[string]bool{}

	for _, t := range tasks {
		if t.Status != models.TaskStatusCompleted {
			pendingTasks = append(pendingTasks, t)
		}
		if t.BranchName != nil && !seenBranches[*t.BranchName] {
			branches = append(branches, *t.BranchName)
			seenBranches[*t.BranchName] = true
		}
	}

	// Get active claims
	activeClaims, err := c.Claims.ListActiveByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list active claims: %w", err)
	}

	// Only exclusive claims block merge — shared_read claims are safe
	var blockingClaims []*models.Claim
	for _, cl := range activeClaims {
		if cl.ClaimType == models.ClaimTypeExclusive {
			blockingClaims = append(blockingClaims, cl)
		}
	}

	allCompleted := len(pendingTasks) == 0 && len(tasks) > 0
	ready := allCompleted && len(blockingClaims) == 0

	return &MergeReadinessReport{
		Ready:        ready,
		AllCompleted: allCompleted,
		PendingTasks: pendingTasks,
		ActiveClaims: activeClaims,
		Branches:     branches,
	}, nil
}
