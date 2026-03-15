package coordinator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/moritzhuber/othrys/internal/models"
)

// ClaimResult holds the outcome of a claim request.
type ClaimResult struct {
	Granted   bool
	Claim     *models.Claim
	Conflicts []*models.Claim
}

// RequestClaim implements the full conflict resolution algorithm:
//  1. Check for overlapping active claims.
//  2. If no conflicts (or shared_read vs shared_read): grant and create claim.
//  3. If conflict: deny and create a conflict event.
//
// Events are published through the EventBus (not pg_notify directly).
func (c *Coordinator) RequestClaim(ctx context.Context, projectID, agentID, taskID, path string, claimType models.ClaimType) (*ClaimResult, error) {
	// Check for conflicts
	conflicts, err := c.Claims.CheckConflict(ctx, projectID, path, claimType)
	if err != nil {
		return nil, fmt.Errorf("check conflict: %w", err)
	}

	// Filter out conflicts from the same agent (agents can re-claim their own paths)
	var realConflicts []*models.Claim
	for _, conflict := range conflicts {
		if conflict.AgentID != agentID {
			realConflicts = append(realConflicts, conflict)
		}
	}

	if len(realConflicts) > 0 {
		// Deny: publish conflict event
		conflictPayload, _ := json.Marshal(map[string]any{
			"path":       path,
			"claim_type": claimType,
			"conflicts":  realConflicts,
		})
		_, _ = c.Events.Create(ctx, &models.Event{
			ProjectID: projectID,
			EventType: models.EventTypeClaimDenied,
			AgentID:   &agentID,
			Payload:   conflictPayload,
		}, c.Bus)

		return &ClaimResult{
			Granted:   false,
			Conflicts: realConflicts,
		}, nil
	}

	// Grant: create the claim
	claim, err := c.Claims.Create(ctx, &models.Claim{
		ProjectID: projectID,
		AgentID:   agentID,
		TaskID:    taskID,
		Path:      path,
		ClaimType: claimType,
	})
	if err != nil {
		return nil, fmt.Errorf("create claim: %w", err)
	}

	// Publish grant event
	grantPayload, _ := json.Marshal(map[string]any{
		"claim_id":   claim.ID,
		"path":       path,
		"claim_type": claimType,
	})
	_, _ = c.Events.Create(ctx, &models.Event{
		ProjectID: projectID,
		EventType: models.EventTypeClaimGranted,
		AgentID:   &agentID,
		Payload:   grantPayload,
	}, c.Bus)

	return &ClaimResult{
		Granted: true,
		Claim:   claim,
	}, nil
}

// ReleaseClaim releases a claim and publishes a release event.
func (c *Coordinator) ReleaseClaim(ctx context.Context, claimID, agentID string) error {
	claim, err := c.Claims.GetByID(ctx, claimID)
	if err != nil {
		return err
	}
	if claim == nil {
		return fmt.Errorf("claim not found: %s", claimID)
	}
	if claim.AgentID != agentID {
		return fmt.Errorf("claim %s is not owned by agent %s", claimID, agentID)
	}

	if err := c.Claims.Release(ctx, claimID); err != nil {
		return err
	}

	releasePayload, _ := json.Marshal(map[string]any{
		"claim_id": claimID,
		"path":     claim.Path,
	})
	_, _ = c.Events.Create(ctx, &models.Event{
		ProjectID: claim.ProjectID,
		EventType: models.EventTypeClaimReleased,
		AgentID:   &agentID,
		Payload:   releasePayload,
	}, c.Bus)

	return nil
}
