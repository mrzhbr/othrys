package store

import (
	"context"
	"testing"

	"github.com/moritzhuber/othrys/internal/models"
)

// setupForClaimTest creates a project, agent, and task for claim tests.
func setupForClaimTest(t *testing.T, s *Store) (projectID, agentID, taskID string) {
	t.Helper()
	ctx := context.Background()

	ps := NewProjectStore(s)
	as := NewAgentStore(s)
	ts := NewTaskStore(s)

	project, err := ps.Create(ctx, "claim-proj-"+testRandomSuffix(), "https://github.com/test/repo")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	agent, err := as.Register(ctx, project.ID, "claim-agent-"+testRandomSuffix(), models.AgentToolGeneric)
	if err != nil {
		t.Fatalf("register agent: %v", err)
	}

	task, err := ts.Create(ctx, &models.Task{
		ProjectID:  project.ID,
		Title:      "Claim test task",
		ModulePath: "internal/auth/",
		Status:     models.TaskStatusProposed,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	return project.ID, agent.ID, task.ID
}

// TestClaimStoreCreate tests creating an active claim.
func TestClaimStoreCreate(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	cs := NewClaimStore(s)

	projectID, agentID, taskID := setupForClaimTest(t, s)

	claim, err := cs.Create(ctx, &models.Claim{
		ProjectID: projectID,
		AgentID:   agentID,
		TaskID:    taskID,
		Path:      "internal/auth/",
		ClaimType: models.ClaimTypeExclusive,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if claim.ID == "" {
		t.Error("expected non-empty claim ID")
	}
	if claim.Status != models.ClaimStatusActive {
		t.Errorf("expected active status, got %s", claim.Status)
	}
}

// TestClaimStoreRelease tests releasing an active claim.
func TestClaimStoreRelease(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	cs := NewClaimStore(s)

	projectID, agentID, taskID := setupForClaimTest(t, s)

	claim, err := cs.Create(ctx, &models.Claim{
		ProjectID: projectID,
		AgentID:   agentID,
		TaskID:    taskID,
		Path:      "internal/api/",
		ClaimType: models.ClaimTypeExclusive,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := cs.Release(ctx, claim.ID); err != nil {
		t.Fatalf("Release: %v", err)
	}

	got, err := cs.GetByID(ctx, claim.ID)
	if err != nil {
		t.Fatalf("GetByID after release: %v", err)
	}
	if got.Status != models.ClaimStatusReleased {
		t.Errorf("expected released status, got %s", got.Status)
	}
}

// TestClaimStoreCheckConflictExclusiveVsExclusive tests that two exclusive claims on overlapping paths conflict.
func TestClaimStoreCheckConflictExclusiveVsExclusive(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	cs := NewClaimStore(s)

	// Create a second agent for conflicts
	as := NewAgentStore(s)
	ps := NewProjectStore(s)
	ts := NewTaskStore(s)

	project, _ := ps.Create(ctx, "conflict-proj-"+testRandomSuffix(), "https://github.com/test/repo")
	agent1, _ := as.Register(ctx, project.ID, "agent1-"+testRandomSuffix(), models.AgentToolGeneric)
	agent2, _ := as.Register(ctx, project.ID, "agent2-"+testRandomSuffix(), models.AgentToolGeneric)
	task1, _ := ts.Create(ctx, &models.Task{ProjectID: project.ID, Title: "t1", ModulePath: "internal/auth/", Status: models.TaskStatusProposed})
	task2, _ := ts.Create(ctx, &models.Task{ProjectID: project.ID, Title: "t2", ModulePath: "internal/auth/", Status: models.TaskStatusProposed})

	// Agent 1 claims internal/auth/
	_, err := cs.Create(ctx, &models.Claim{
		ProjectID: project.ID,
		AgentID:   agent1.ID,
		TaskID:    task1.ID,
		Path:      "internal/auth/",
		ClaimType: models.ClaimTypeExclusive,
	})
	if err != nil {
		t.Fatalf("create claim for agent1: %v", err)
	}

	// Agent 2 tries to claim overlapping path — should conflict
	conflicts, err := cs.CheckConflict(ctx, project.ID, "internal/auth/middleware.go", models.ClaimTypeExclusive)
	if err != nil {
		t.Fatalf("CheckConflict: %v", err)
	}

	// Filter out agent2's own claims (there are none yet)
	_ = task2
	_ = agent2

	if len(conflicts) == 0 {
		t.Error("expected conflict for overlapping exclusive claim, got none")
	}
}

// TestClaimStoreCheckConflictNoOverlap tests non-overlapping paths have no conflict.
func TestClaimStoreCheckConflictNoOverlap(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	cs := NewClaimStore(s)
	ps := NewProjectStore(s)
	as := NewAgentStore(s)
	ts := NewTaskStore(s)

	project, _ := ps.Create(ctx, "no-overlap-proj-"+testRandomSuffix(), "https://github.com/test/repo")
	agent, _ := as.Register(ctx, project.ID, "agent-"+testRandomSuffix(), models.AgentToolGeneric)
	task, _ := ts.Create(ctx, &models.Task{ProjectID: project.ID, Title: "t", ModulePath: "internal/auth/", Status: models.TaskStatusProposed})

	// Agent claims internal/auth/
	_, err := cs.Create(ctx, &models.Claim{
		ProjectID: project.ID,
		AgentID:   agent.ID,
		TaskID:    task.ID,
		Path:      "internal/auth/",
		ClaimType: models.ClaimTypeExclusive,
	})
	if err != nil {
		t.Fatalf("create claim: %v", err)
	}

	// Check a completely different path — should not conflict
	conflicts, err := cs.CheckConflict(ctx, project.ID, "internal/api/", models.ClaimTypeExclusive)
	if err != nil {
		t.Fatalf("CheckConflict: %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts for non-overlapping path, got %d", len(conflicts))
	}
}

// TestClaimStoreListActiveByProject tests listing active claims.
func TestClaimStoreListActiveByProject(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	cs := NewClaimStore(s)

	projectID, agentID, taskID := setupForClaimTest(t, s)

	_, err := cs.Create(ctx, &models.Claim{
		ProjectID: projectID,
		AgentID:   agentID,
		TaskID:    taskID,
		Path:      "internal/store/",
		ClaimType: models.ClaimTypeExclusive,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	claims, err := cs.ListActiveByProject(ctx, projectID)
	if err != nil {
		t.Fatalf("ListActiveByProject: %v", err)
	}
	if len(claims) == 0 {
		t.Error("expected at least 1 active claim")
	}
	for _, c := range claims {
		if c.Status != models.ClaimStatusActive {
			t.Errorf("expected active claim, got %s", c.Status)
		}
	}
}
