package store

import (
	"context"
	"testing"

	"github.com/moritzhuber/othrys/internal/models"
)

// setupProjectForTaskTest creates a test project and agent for task tests.
func setupProjectForTaskTest(t *testing.T, s *Store) (projectID string, agentID string) {
	t.Helper()
	ctx := context.Background()
	ps := NewProjectStore(s)
	as := NewAgentStore(s)

	project, err := ps.Create(ctx, "task-test-proj-"+testRandomSuffix(), "https://github.com/test/repo")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	agent, err := as.Register(ctx, project.ID, "agent-"+testRandomSuffix(), models.AgentToolGeneric)
	if err != nil {
		t.Fatalf("register agent: %v", err)
	}

	return project.ID, agent.ID
}

// TestTaskStoreCreate tests creating a task.
func TestTaskStoreCreate(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	ts := NewTaskStore(s)

	projectID, _ := setupProjectForTaskTest(t, s)

	task, err := ts.Create(ctx, &models.Task{
		ProjectID:   projectID,
		Title:       "Implement auth module",
		Description: "Build the authentication system",
		ModulePath:  "internal/auth/",
		Status:      models.TaskStatusProposed,
		DependsOn:   []string{},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if task.ID == "" {
		t.Error("expected non-empty ID")
	}
	if task.Status != models.TaskStatusProposed {
		t.Errorf("expected proposed status, got %s", task.Status)
	}
}

// TestTaskStoreGetByID tests fetching a task by ID.
func TestTaskStoreGetByID(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	ts := NewTaskStore(s)

	projectID, _ := setupProjectForTaskTest(t, s)
	created, err := ts.Create(ctx, &models.Task{
		ProjectID:  projectID,
		Title:      "Auth task",
		ModulePath: "internal/auth/",
		Status:     models.TaskStatusProposed,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := ts.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected task, got nil")
	}
	if got.Title != "Auth task" {
		t.Errorf("Title mismatch: got %q", got.Title)
	}
}

// TestTaskStoreUpdateStatusTransition tests valid status transitions.
func TestTaskStoreUpdateStatusTransition(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	ts := NewTaskStore(s)

	projectID, agentID := setupProjectForTaskTest(t, s)
	task, err := ts.Create(ctx, &models.Task{
		ProjectID:  projectID,
		Title:      "Status test task",
		ModulePath: "internal/api/",
		Status:     models.TaskStatusProposed,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// proposed → approved
	if err := ts.UpdateStatus(ctx, task.ID, models.TaskStatusApproved); err != nil {
		t.Fatalf("proposed→approved: %v", err)
	}

	// approved → assigned (via Assign)
	if err := ts.Assign(ctx, task.ID, agentID, "othrys/agent/status-test-task"); err != nil {
		t.Fatalf("Assign: %v", err)
	}

	// assigned → in_progress
	if err := ts.UpdateStatus(ctx, task.ID, models.TaskStatusInProgress); err != nil {
		t.Fatalf("assigned→in_progress: %v", err)
	}

	// in_progress → completed
	if err := ts.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted); err != nil {
		t.Fatalf("in_progress→completed: %v", err)
	}
}

// TestTaskStoreInvalidTransition tests that invalid transitions are rejected.
func TestTaskStoreInvalidTransition(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	ts := NewTaskStore(s)

	projectID, _ := setupProjectForTaskTest(t, s)
	task, err := ts.Create(ctx, &models.Task{
		ProjectID:  projectID,
		Title:      "Invalid transition task",
		ModulePath: "internal/bad/",
		Status:     models.TaskStatusProposed,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// proposed → in_progress (invalid — must go through approved and assigned first)
	if err := ts.UpdateStatus(ctx, task.ID, models.TaskStatusInProgress); err == nil {
		t.Error("expected error for invalid transition proposed→in_progress, got nil")
	}
}

// TestTaskStoreListByProject tests listing tasks for a project.
func TestTaskStoreListByProject(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	ts := NewTaskStore(s)

	projectID, _ := setupProjectForTaskTest(t, s)

	for i := 0; i < 3; i++ {
		_, err := ts.Create(ctx, &models.Task{
			ProjectID:  projectID,
			Title:      "Task " + testRandomSuffix(),
			ModulePath: "internal/",
			Status:     models.TaskStatusProposed,
		})
		if err != nil {
			t.Fatalf("Create task %d: %v", i, err)
		}
	}

	tasks, err := ts.ListByProject(ctx, projectID, nil)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(tasks) < 3 {
		t.Errorf("expected at least 3 tasks, got %d", len(tasks))
	}

	// Filter by status
	status := models.TaskStatusProposed
	filtered, err := ts.ListByProject(ctx, projectID, &status)
	if err != nil {
		t.Fatalf("ListByProject with filter: %v", err)
	}
	for _, task := range filtered {
		if task.Status != models.TaskStatusProposed {
			t.Errorf("expected proposed status, got %s", task.Status)
		}
	}
}
