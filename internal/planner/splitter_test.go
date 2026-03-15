package planner

import (
	"testing"
)

// mockProvider is a mock LLM provider for testing.
type mockProvider struct {
	tasks []ProposedTask
	err   error
}

func (m *mockProvider) SplitDesign(design string, config map[string]any) ([]ProposedTask, error) {
	return m.tasks, m.err
}

// TestPathsOverlap tests the path prefix overlap detection.
func TestPathsOverlap(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"src/auth/", "src/auth/middleware.go", true},
		{"src/auth/", "src/auth/", true},
		{"src/auth/", "src/api/", false},
		{"src/", "src/auth/", true},
		{"internal/", "internal/store/", true},
		{"cmd/", "internal/", false},
		{"", "anything", true}, // empty prefix matches everything
	}

	for _, tc := range cases {
		got := pathsOverlap(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("pathsOverlap(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

// TestValidateProposedTasks tests the improved validation of LLM output.
func TestValidateProposedTasks(t *testing.T) {
	t.Run("empty tasks returns error", func(t *testing.T) {
		err := validateProposedTasks(nil)
		if err == nil {
			t.Error("expected error for empty tasks, got nil")
		}
	})

	t.Run("task without title returns error", func(t *testing.T) {
		err := validateProposedTasks([]ProposedTask{
			{Title: "", ModulePath: "src/", Description: "A task with enough description text here"},
		})
		if err == nil {
			t.Error("expected error for empty title, got nil")
		}
	})

	t.Run("overlapping paths returns error (not just warning)", func(t *testing.T) {
		err := validateProposedTasks([]ProposedTask{
			{Title: "Root", ModulePath: "src/", Description: "A task with enough description text here"},
			{Title: "Auth", ModulePath: "src/auth/", Description: "A task with enough description text here"},
		})
		if err == nil {
			t.Error("expected error for overlapping paths, got nil — overlapping paths must now be an error, not a warning")
		}
	})

	t.Run("empty description returns error", func(t *testing.T) {
		err := validateProposedTasks([]ProposedTask{
			{Title: "Auth module", ModulePath: "src/auth/", Description: "short"},
		})
		if err == nil {
			t.Error("expected error for description <= 20 chars, got nil")
		}
	})

	t.Run("module path without trailing slash returns error", func(t *testing.T) {
		err := validateProposedTasks([]ProposedTask{
			{Title: "Auth module", ModulePath: "src/auth", Description: "A task with enough description text here"},
		})
		if err == nil {
			t.Error("expected error for module_path without trailing '/', got nil")
		}
	})

	t.Run("dependency cycle detected and returns error", func(t *testing.T) {
		err := validateProposedTasks([]ProposedTask{
			{Title: "Task A", ModulePath: "src/a/", Description: "A task with enough description text here", DependsOn: []string{"Task B"}},
			{Title: "Task B", ModulePath: "src/b/", Description: "A task with enough description text here", DependsOn: []string{"Task A"}},
		})
		if err == nil {
			t.Error("expected error for dependency cycle, got nil")
		}
	})

	t.Run("consumed contract with no matching provided contract returns error", func(t *testing.T) {
		err := validateProposedTasks([]ProposedTask{
			{
				Title:             "Task A",
				ModulePath:        "src/a/",
				Description:       "A task with enough description text here",
				ConsumedContracts: []string{"UserRepository"},
			},
		})
		if err == nil {
			t.Error("expected error for consumed contract with no provider, got nil")
		}
	})

	t.Run("consumed contract matching is case-insensitive", func(t *testing.T) {
		err := validateProposedTasks([]ProposedTask{
			{
				Title:       "Provider task",
				ModulePath:  "src/a/",
				Description: "A task with enough description text here",
				ProvidedContracts: []Contract{
					{Name: "userrepo", Description: "user repo interface", FilePath: "src/a/repo.go"},
				},
			},
			{
				Title:             "Consumer task",
				ModulePath:        "src/b/",
				Description:       "A task with enough description text here",
				ConsumedContracts: []string{"UserRepo"}, // different case, should match
			},
		})
		if err != nil {
			t.Errorf("case-insensitive contract match failed: %v", err)
		}
	})

	t.Run("valid tasks pass validation", func(t *testing.T) {
		err := validateProposedTasks([]ProposedTask{
			{Title: "Auth module", ModulePath: "src/auth/", Description: "A task with enough description text here"},
			{Title: "API module", ModulePath: "src/api/", Description: "A task with enough description text here"},
		})
		if err != nil {
			t.Errorf("unexpected error for valid tasks: %v", err)
		}
	})
}

// TestAutoPopulateBoundaries tests that autoPopulateBoundaries sets forbidden_paths and read_only_paths correctly.
func TestAutoPopulateBoundaries(t *testing.T) {
	tasks := []ProposedTask{
		{
			Title:      "Task A",
			ModulePath: "src/a/",
			ProvidedContracts: []Contract{
				{Name: "AInterface", Description: "interface A", FilePath: "src/a/iface.go"},
			},
		},
		{
			Title:             "Task B",
			ModulePath:        "src/b/",
			ConsumedContracts: []string{"AInterface"},
		},
		{
			Title:      "Task C",
			ModulePath: "src/c/",
		},
	}

	result := autoPopulateBoundaries(tasks)

	// Task A: forbidden = src/b/ and src/c/, read_only = nothing (no consumed contracts)
	taskA := result[0]
	if len(taskA.ForbiddenPaths) != 2 {
		t.Errorf("Task A: expected 2 forbidden paths, got %d: %v", len(taskA.ForbiddenPaths), taskA.ForbiddenPaths)
	}
	if len(taskA.ReadOnlyPaths) != 0 {
		t.Errorf("Task A: expected 0 read_only_paths, got %d", len(taskA.ReadOnlyPaths))
	}

	// Task B: forbidden = src/a/ and src/c/, read_only = src/a/ (consumes AInterface from A)
	taskB := result[1]
	if len(taskB.ForbiddenPaths) != 2 {
		t.Errorf("Task B: expected 2 forbidden paths, got %d: %v", len(taskB.ForbiddenPaths), taskB.ForbiddenPaths)
	}
	foundReadOnly := false
	for _, p := range taskB.ReadOnlyPaths {
		if p == "src/a/" {
			foundReadOnly = true
		}
	}
	if !foundReadOnly {
		t.Errorf("Task B: expected src/a/ in read_only_paths (consumes AInterface), got %v", taskB.ReadOnlyPaths)
	}

	// Task C: forbidden = src/a/ and src/b/, read_only = nothing
	taskC := result[2]
	if len(taskC.ForbiddenPaths) != 2 {
		t.Errorf("Task C: expected 2 forbidden paths, got %d: %v", len(taskC.ForbiddenPaths), taskC.ForbiddenPaths)
	}
}

// TestMockProviderIntegration tests the mock provider returns expected tasks.
func TestMockProviderIntegration(t *testing.T) {
	mock := &mockProvider{
		tasks: []ProposedTask{
			{Title: "Set up auth", Description: "Implement auth module", ModulePath: "internal/auth/"},
			{Title: "Set up API", Description: "Implement REST handlers", ModulePath: "internal/server/"},
		},
	}

	tasks, err := mock.SplitDesign("test design", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].Title != "Set up auth" {
		t.Errorf("expected title 'Set up auth', got %q", tasks[0].Title)
	}
}
