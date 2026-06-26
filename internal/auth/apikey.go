package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ValidateAPIKey checks whether the provided API key exists in the projects table.
// Returns the project ID if valid, empty string if not found.
func ValidateAPIKey(ctx context.Context, pool *pgxpool.Pool, apiKey string) (string, error) {
	if strings.TrimSpace(apiKey) == "" {
		return "", nil
	}

	var projectID string
	err := pool.QueryRow(ctx,
		`SELECT id FROM projects WHERE api_key = $1`,
		apiKey,
	).Scan(&projectID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}

	return projectID, nil
}
