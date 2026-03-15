package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestRepo creates a temporary git repository with an initial commit.
// Returns the repo path and a cleanup function.
func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	// Verify git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH — skipping git integration test")
	}

	dir, err := os.MkdirTemp("", "othrys-git-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init", "-b", "main")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")

	// Create initial commit on main
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test Repo\n"), 0644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "initial commit")

	return dir, func() { os.RemoveAll(dir) }
}

// TestListBranches tests listing branches in a repo.
func TestListBranches(t *testing.T) {
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	svc := NewLocalGitService()
	branches, err := svc.ListBranches(dir, "*")
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}

	found := false
	for _, b := range branches {
		if b == "main" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'main' in branches, got: %v", branches)
	}
}

// TestCreateBranch tests creating a new branch.
func TestCreateBranch(t *testing.T) {
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	svc := NewLocalGitService()
	err := svc.CreateBranch(dir, "othrys/agent-alice/implement-auth", "main")
	if err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}

	branches, err := svc.ListBranches(dir, "othrys/*")
	if err != nil {
		t.Fatalf("ListBranches after create: %v", err)
	}

	found := false
	for _, b := range branches {
		if strings.Contains(b, "implement-auth") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected new branch in list, got: %v", branches)
	}
}

// TestPreviewMergeNoConflict tests merge preview for non-conflicting branches.
func TestPreviewMergeNoConflict(t *testing.T) {
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	// Create a feature branch with a new file (no conflict with main)
	run("checkout", "-b", "othrys/agent-alice/auth")
	if err := os.WriteFile(filepath.Join(dir, "auth.go"), []byte("package auth\n"), 0644); err != nil {
		t.Fatalf("write auth.go: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "add auth")
	run("checkout", "main")

	svc := NewLocalGitService()
	preview, err := svc.PreviewMerge(dir, []string{"othrys/agent-alice/auth"}, "main")
	if err != nil {
		t.Fatalf("PreviewMerge: %v", err)
	}

	if preview.HasConflicts {
		t.Errorf("expected no conflicts, got: %v", preview.ConflictingFiles)
	}
}

// TestPreviewMergeEmpty tests merge preview with no branches.
func TestPreviewMergeEmpty(t *testing.T) {
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	svc := NewLocalGitService()
	preview, err := svc.PreviewMerge(dir, []string{}, "main")
	if err != nil {
		t.Fatalf("PreviewMerge with empty branches: %v", err)
	}
	if preview == nil {
		t.Error("expected non-nil preview, got nil")
	}
	if preview.HasConflicts {
		t.Error("expected no conflicts for empty branches")
	}
}
