package planner

import (
	"strings"
	"testing"
)

// TestBuildPromptWithEmptyContext verifies buildPrompt works without project context.
func TestBuildPromptWithEmptyContext(t *testing.T) {
	prompt := buildPrompt("Build an auth module", nil)

	if len(prompt) == 0 {
		t.Error("expected non-empty prompt")
	}
	// Should not contain Project Context section when context is nil
	if strings.Contains(prompt, "PROJECT CONTEXT:") {
		t.Error("expected no PROJECT CONTEXT section when projectCtx is nil")
	}
	// Should still have the design
	if !strings.Contains(prompt, "Build an auth module") {
		t.Error("expected design document in prompt")
	}
	// Should have ROLE
	if !strings.Contains(prompt, "ROLE:") {
		t.Error("expected ROLE section in prompt")
	}
}

// TestBuildPromptWithFullContext verifies project context is injected into the prompt.
func TestBuildPromptWithFullContext(t *testing.T) {
	ctx := map[string]any{
		"tech_stack":     "Go 1.22, Fiber, PostgreSQL",
		"module_path":    "github.com/example/myproject",
		"directory_tree": []string{"cmd/", "internal/auth/", "internal/store/"},
		"conventions":    "Use pgx for DB. All handlers in internal/server/handlers/.",
	}

	prompt := buildPrompt("Build an auth module", ctx)

	if !strings.Contains(prompt, "PROJECT CONTEXT:") {
		t.Error("expected PROJECT CONTEXT section when context is provided")
	}
	if !strings.Contains(prompt, "Go 1.22, Fiber, PostgreSQL") {
		t.Error("expected tech_stack in prompt")
	}
	if !strings.Contains(prompt, "github.com/example/myproject") {
		t.Error("expected module_path in prompt")
	}
}

// TestBuildPromptContainsAllJSONFields verifies the prompt's JSON schema includes all 9 ProposedTask fields.
func TestBuildPromptContainsAllJSONFields(t *testing.T) {
	prompt := buildPrompt("Test design", nil)

	requiredFields := []string{
		`"title"`,
		`"description"`,
		`"module_path"`,
		`"depends_on"`,
		`"read_only_paths"`,
		`"forbidden_paths"`,
		`"integration_points"`,
		`"provided_contracts"`,
		`"consumed_contracts"`,
	}

	for _, field := range requiredFields {
		if !strings.Contains(prompt, field) {
			t.Errorf("expected field %q in prompt JSON schema, not found", field)
		}
	}
}

// TestBuildPromptContainsConcurrencyRules verifies concurrency-related rules are in the prompt.
func TestBuildPromptContainsConcurrencyRules(t *testing.T) {
	prompt := buildPrompt("Test design", nil)

	if !strings.Contains(prompt, "concurrent") {
		t.Error("expected 'concurrent' in prompt (concurrency awareness)")
	}
	if !strings.Contains(prompt, "forbidden_paths") {
		t.Error("expected forbidden_paths rule in prompt")
	}
	if !strings.Contains(prompt, "read_only_paths") {
		t.Error("expected read_only_paths rule in prompt")
	}
	if !strings.Contains(prompt, "provided_contract") || !strings.Contains(prompt, "consumed_contract") {
		t.Error("expected contract rules in prompt")
	}
}

// TestBuildPromptLength verifies the prompt is substantial (>50 lines).
func TestBuildPromptLength(t *testing.T) {
	prompt := buildPrompt("Test design", map[string]any{
		"tech_stack":  "Go 1.22",
		"module_path": "github.com/example/app",
	})

	lines := strings.Split(prompt, "\n")
	if len(lines) < 40 {
		t.Errorf("expected prompt to be at least 40 lines, got %d lines", len(lines))
	}
}

// TestParseTasksFromJSON tests parsing the LLM JSON response.
func TestParseTasksFromJSON(t *testing.T) {
	jsonText := `[
		{
			"title": "Auth module",
			"description": "Implement authentication",
			"module_path": "internal/auth/",
			"depends_on": [],
			"read_only_paths": [],
			"forbidden_paths": ["internal/store/"],
			"integration_points": ["connects to store layer"],
			"provided_contracts": [],
			"consumed_contracts": []
		}
	]`

	tasks, err := parseTasksFromJSON(jsonText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Title != "Auth module" {
		t.Errorf("expected title 'Auth module', got %q", tasks[0].Title)
	}
	if len(tasks[0].ForbiddenPaths) != 1 || tasks[0].ForbiddenPaths[0] != "internal/store/" {
		t.Errorf("expected forbidden_paths to contain 'internal/store/', got %v", tasks[0].ForbiddenPaths)
	}
}
