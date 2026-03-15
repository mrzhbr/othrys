package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/moritzhuber/othrys/internal/models"
)

// ProjectStore handles database operations for projects.
type ProjectStore struct {
	*Store
}

// NewProjectStore creates a new ProjectStore.
func NewProjectStore(s *Store) *ProjectStore {
	return &ProjectStore{s}
}

// generateAPIKey generates a 32-byte (64 hex char) API key.
func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate API key: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Create inserts a new project and returns it with a generated API key.
func (ps *ProjectStore) Create(ctx context.Context, name, repoURL string) (*models.Project, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	var p models.Project
	err = ps.Pool.QueryRow(ctx, `
		INSERT INTO projects (name, repo_url, api_key, config, created_at, updated_at)
		VALUES ($1, $2, $3, '{}', $4, $4)
		RETURNING id, name, repo_url, api_key, pm_design, config, shared_contracts, project_context, created_at, updated_at
	`, name, repoURL, apiKey, now).Scan(
		&p.ID, &p.Name, &p.RepoURL, &p.APIKey,
		&p.PMDesign, &p.Config, &p.SharedContracts, &p.ProjectContext,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	return &p, nil
}

// GetByID retrieves a project by its UUID.
func (ps *ProjectStore) GetByID(ctx context.Context, id string) (*models.Project, error) {
	var p models.Project
	err := ps.Pool.QueryRow(ctx, `
		SELECT id, name, repo_url, api_key, pm_design, config, shared_contracts, project_context, created_at, updated_at
		FROM projects WHERE id = $1
	`, id).Scan(
		&p.ID, &p.Name, &p.RepoURL, &p.APIKey,
		&p.PMDesign, &p.Config, &p.SharedContracts, &p.ProjectContext,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project by ID: %w", err)
	}
	return &p, nil
}

// GetByAPIKey retrieves a project by its API key.
func (ps *ProjectStore) GetByAPIKey(ctx context.Context, apiKey string) (*models.Project, error) {
	var p models.Project
	err := ps.Pool.QueryRow(ctx, `
		SELECT id, name, repo_url, api_key, pm_design, config, shared_contracts, project_context, created_at, updated_at
		FROM projects WHERE api_key = $1
	`, apiKey).Scan(
		&p.ID, &p.Name, &p.RepoURL, &p.APIKey,
		&p.PMDesign, &p.Config, &p.SharedContracts, &p.ProjectContext,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project by API key: %w", err)
	}
	return &p, nil
}

// UpdateDesign stores the PM design document (as JSON) for a project.
func (ps *ProjectStore) UpdateDesign(ctx context.Context, id string, design []byte) error {
	now := time.Now().UTC()
	ct, err := ps.Pool.Exec(ctx, `
		UPDATE projects SET pm_design = $1, updated_at = $2 WHERE id = $3
	`, design, now, id)
	if err != nil {
		return fmt.Errorf("update design: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("project not found: %s", id)
	}
	return nil
}

// UpdateProjectContext stores the project context (tech stack, conventions, etc.) as JSONB.
func (ps *ProjectStore) UpdateProjectContext(ctx context.Context, projectID string, context json.RawMessage) error {
	now := time.Now().UTC()
	ct, err := ps.Pool.Exec(ctx, `
		UPDATE projects SET project_context = $1, updated_at = $2 WHERE id = $3
	`, context, now, projectID)
	if err != nil {
		return fmt.Errorf("update project context: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("project not found: %s", projectID)
	}
	return nil
}

// UpdateSharedContracts stores the shared contracts (interfaces/types) as JSONB.
func (ps *ProjectStore) UpdateSharedContracts(ctx context.Context, projectID string, contracts json.RawMessage) error {
	now := time.Now().UTC()
	ct, err := ps.Pool.Exec(ctx, `
		UPDATE projects SET shared_contracts = $1, updated_at = $2 WHERE id = $3
	`, contracts, now, projectID)
	if err != nil {
		return fmt.Errorf("update shared contracts: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("project not found: %s", projectID)
	}
	return nil
}

// ListAll returns all projects.
func (ps *ProjectStore) ListAll(ctx context.Context) ([]*models.Project, error) {
	rows, err := ps.Pool.Query(ctx, `
		SELECT id, name, repo_url, api_key, pm_design, config, shared_contracts, project_context, created_at, updated_at
		FROM projects ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []*models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(
			&p.ID, &p.Name, &p.RepoURL, &p.APIKey,
			&p.PMDesign, &p.Config, &p.SharedContracts, &p.ProjectContext,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project row: %w", err)
		}
		projects = append(projects, &p)
	}
	return projects, rows.Err()
}
