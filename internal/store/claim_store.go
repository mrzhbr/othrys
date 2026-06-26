package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/moritzhuber/othrys/internal/models"
)

// ClaimStore handles database operations for claims.
type ClaimStore struct {
	*Store
}

// NewClaimStore creates a new ClaimStore.
func NewClaimStore(s *Store) *ClaimStore {
	return &ClaimStore{s}
}

// Create inserts a new active claim.
func (cs *ClaimStore) Create(ctx context.Context, c *models.Claim) (*models.Claim, error) {
	now := time.Now().UTC()
	var out models.Claim
	err := cs.Pool.QueryRow(ctx, `
		INSERT INTO claims (project_id, agent_id, task_id, path, claim_type, status, granted_at)
		VALUES ($1, $2, $3, $4, $5, 'active', $6)
		RETURNING id, project_id, agent_id, task_id, path, claim_type, status, granted_at, released_at
	`, c.ProjectID, c.AgentID, c.TaskID, c.Path, c.ClaimType, now).Scan(
		&out.ID, &out.ProjectID, &out.AgentID, &out.TaskID, &out.Path,
		&out.ClaimType, &out.Status, &out.GrantedAt, &out.ReleasedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create claim: %w", err)
	}
	return &out, nil
}

// GetByID retrieves a claim by UUID.
func (cs *ClaimStore) GetByID(ctx context.Context, id string) (*models.Claim, error) {
	var c models.Claim
	err := cs.Pool.QueryRow(ctx, `
		SELECT id, project_id, agent_id, task_id, path, claim_type, status, granted_at, released_at
		FROM claims WHERE id = $1
	`, id).Scan(
		&c.ID, &c.ProjectID, &c.AgentID, &c.TaskID, &c.Path,
		&c.ClaimType, &c.Status, &c.GrantedAt, &c.ReleasedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get claim: %w", err)
	}
	return &c, nil
}

// Release sets a claim's status to "released".
func (cs *ClaimStore) Release(ctx context.Context, id string) error {
	now := time.Now().UTC()
	ct, err := cs.Pool.Exec(ctx, `
		UPDATE claims SET status = 'released', released_at = $1 WHERE id = $2 AND status = 'active'
	`, now, id)
	if err != nil {
		return fmt.Errorf("release claim: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("claim not found or not active: %s", id)
	}
	return nil
}

// Revoke sets a claim's status to "revoked" (used by cleanup for disconnected agents).
func (cs *ClaimStore) Revoke(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := cs.Pool.Exec(ctx, `
		UPDATE claims SET status = 'revoked', released_at = $1 WHERE id = $2 AND status = 'active'
	`, now, id)
	return err
}

// RevokeByAgent revokes all active claims held by a specific agent.
func (cs *ClaimStore) RevokeByAgent(ctx context.Context, agentID string) ([]string, error) {
	now := time.Now().UTC()
	rows, err := cs.Pool.Query(ctx, `
		UPDATE claims SET status = 'revoked', released_at = $1
		WHERE agent_id = $2 AND status = 'active'
		RETURNING id
	`, now, agentID)
	if err != nil {
		return nil, fmt.Errorf("revoke agent claims: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ListActiveByProject returns all active claims for a project.
func (cs *ClaimStore) ListActiveByProject(ctx context.Context, projectID string) ([]*models.Claim, error) {
	rows, err := cs.Pool.Query(ctx, `
		SELECT id, project_id, agent_id, task_id, path, claim_type, status, granted_at, released_at
		FROM claims WHERE project_id = $1 AND status = 'active'
		ORDER BY granted_at ASC
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list active claims: %w", err)
	}
	defer rows.Close()

	var claims []*models.Claim
	for rows.Next() {
		var c models.Claim
		if err := rows.Scan(
			&c.ID, &c.ProjectID, &c.AgentID, &c.TaskID, &c.Path,
			&c.ClaimType, &c.Status, &c.GrantedAt, &c.ReleasedAt,
		); err != nil {
			return nil, fmt.Errorf("scan claim: %w", err)
		}
		claims = append(claims, &c)
	}
	return claims, rows.Err()
}

// CheckConflict returns any active exclusive claims that overlap with the given path.
// Path overlap is defined as: one path is a directory prefix of the other
// (i.e., "src/auth/" overlaps with "src/auth/main.go" but NOT with "src/auth-middleware/").
// For shared_read requests, only exclusive conflicts are returned.
func (cs *ClaimStore) CheckConflict(ctx context.Context, projectID, path string, requestedType models.ClaimType) ([]*models.Claim, error) {
	// Normalize path to end with "/" for consistent prefix matching
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	// Shared_read vs shared_read never conflicts
	// Only exclusive claims can block other claims
	var rows pgx.Rows
	var err error

	if requestedType == models.ClaimTypeExclusive {
		// Block on any overlapping active claim (both exclusive and shared_read)
		rows, err = cs.Pool.Query(ctx, `
			SELECT id, project_id, agent_id, task_id, path, claim_type, status, granted_at, released_at
			FROM claims
			WHERE project_id = $1
			  AND status = 'active'
			  AND (
			    path = $2
			    OR (path || '/') LIKE ($2 || '/%')
			    OR ($2 || '/') LIKE (path || '/%')
			  )
		`, projectID, path)
	} else {
		// shared_read: only blocked by exclusive claims
		rows, err = cs.Pool.Query(ctx, `
			SELECT id, project_id, agent_id, task_id, path, claim_type, status, granted_at, released_at
			FROM claims
			WHERE project_id = $1
			  AND status = 'active'
			  AND claim_type = 'exclusive'
			  AND (
			    path = $2
			    OR (path || '/') LIKE ($2 || '/%')
			    OR ($2 || '/') LIKE (path || '/%')
			  )
		`, projectID, path)
	}

	if err != nil {
		return nil, fmt.Errorf("check conflict: %w", err)
	}
	defer rows.Close()

	var conflicts []*models.Claim
	for rows.Next() {
		var c models.Claim
		if err := rows.Scan(
			&c.ID, &c.ProjectID, &c.AgentID, &c.TaskID, &c.Path,
			&c.ClaimType, &c.Status, &c.GrantedAt, &c.ReleasedAt,
		); err != nil {
			return nil, fmt.Errorf("scan conflict: %w", err)
		}
		conflicts = append(conflicts, &c)
	}
	return conflicts, rows.Err()
}
