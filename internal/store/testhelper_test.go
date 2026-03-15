package store

import (
	"context"
	"os"
	"testing"
)

// testDB sets up a database connection for integration tests.
// Skips the test if DATABASE_URL is not set.
func testDB(t *testing.T) *Store {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set — skipping store integration test")
	}

	ctx := context.Background()
	pool, err := NewDB(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect to test DB: %v", err)
	}

	// Run migrations
	if err := RunMigrations(ctx, pool, "../../migrations"); err != nil {
		pool.Close()
		t.Fatalf("run migrations: %v", err)
	}

	s := &Store{Pool: pool}
	t.Cleanup(func() {
		pool.Close()
	})

	return s
}
