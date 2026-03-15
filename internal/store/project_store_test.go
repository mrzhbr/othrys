package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"testing"
)

// testRandomSuffix generates a short random hex suffix for unique test names.
func testRandomSuffix() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// TestProjectStoreCreate tests creating a project and verifying the API key is generated.
func TestProjectStoreCreate(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	ps := NewProjectStore(s)

	project, err := ps.Create(ctx, "test-project-"+testRandomSuffix(), "https://github.com/test/repo")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if project.ID == "" {
		t.Error("expected non-empty ID")
	}
	if len(project.APIKey) != 64 {
		t.Errorf("expected 64-char API key, got %d chars", len(project.APIKey))
	}
}

// TestProjectStoreGetByID tests fetching a project by ID.
func TestProjectStoreGetByID(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	ps := NewProjectStore(s)

	created, err := ps.Create(ctx, "get-by-id-"+testRandomSuffix(), "https://github.com/test/repo")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := ps.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected project, got nil")
	}
	if got.ID != created.ID {
		t.Errorf("ID mismatch: got %q, want %q", got.ID, created.ID)
	}
}

// TestProjectStoreGetByAPIKey tests fetching a project by API key.
func TestProjectStoreGetByAPIKey(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	ps := NewProjectStore(s)

	created, err := ps.Create(ctx, "api-key-"+testRandomSuffix(), "https://github.com/test/repo")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := ps.GetByAPIKey(ctx, created.APIKey)
	if err != nil {
		t.Fatalf("GetByAPIKey: %v", err)
	}
	if got == nil {
		t.Fatal("expected project, got nil")
	}
	if got.ID != created.ID {
		t.Errorf("ID mismatch: got %q, want %q", got.ID, created.ID)
	}
}

// TestProjectStoreGetByIDNotFound tests that missing ID returns nil.
func TestProjectStoreGetByIDNotFound(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	ps := NewProjectStore(s)

	got, err := ps.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("GetByID for nonexistent: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for nonexistent ID, got %+v", got)
	}
}

// TestProjectStoreUpdateDesign tests updating the PM design.
func TestProjectStoreUpdateDesign(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	ps := NewProjectStore(s)

	project, err := ps.Create(ctx, "design-"+testRandomSuffix(), "https://github.com/test/repo")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	design := []byte(`"This is the PM design document"`)
	if err := ps.UpdateDesign(ctx, project.ID, design); err != nil {
		t.Fatalf("UpdateDesign: %v", err)
	}

	updated, err := ps.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if updated.PMDesign == nil {
		t.Error("expected non-nil PMDesign after update")
	}
}
