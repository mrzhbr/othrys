package coordinator

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/moritzhuber/othrys/internal/models"
)

func makeTask(id, title, module string) *models.Task {
	return &models.Task{
		ID:         id,
		Title:      title,
		ModulePath: module,
	}
}

func makeProject(contextJSON, sharedContractsJSON string) *models.Project {
	p := &models.Project{ID: "proj-1"}
	if contextJSON != "" {
		p.ProjectContext = json.RawMessage(contextJSON)
	}
	if sharedContractsJSON != "" {
		p.SharedContracts = json.RawMessage(sharedContractsJSON)
	}
	return p
}

// TestBriefingIncludesProjectContext verifies project context appears in the briefing.
func TestBriefingIncludesProjectContext(t *testing.T) {
	task := makeTask("t1", "Build Auth Module", "internal/auth/")
	project := makeProject(`{"tech_stack":"Go 1.22, Fiber","module_path":"github.com/example/app"}`, "")

	briefing := AssembleBriefing(task, project, nil)

	if !strings.Contains(briefing, "Go 1.22, Fiber") {
		t.Error("expected briefing to include tech_stack from project context")
	}
	if !strings.Contains(briefing, "github.com/example/app") {
		t.Error("expected briefing to include module_path from project context")
	}
	if !strings.Contains(briefing, "## Project Context") {
		t.Error("expected 'Project Context' section header")
	}
}

// TestBriefingIncludesConcurrencyNotice verifies sibling tasks appear in the concurrency section.
func TestBriefingIncludesConcurrencyNotice(t *testing.T) {
	task := makeTask("t1", "Build Auth Module", "internal/auth/")
	project := makeProject("", "")
	siblings := []*models.Task{
		makeTask("t2", "Build API Module", "internal/api/"),
		makeTask("t3", "Build Store Module", "internal/store/"),
	}

	briefing := AssembleBriefing(task, project, siblings)

	if !strings.Contains(briefing, "## Concurrency Notice") {
		t.Error("expected 'Concurrency Notice' section")
	}
	// 2 siblings + 1 self = 3 agents total
	if !strings.Contains(briefing, "3 agents") {
		t.Errorf("expected '3 agents' in concurrency notice, got:\n%s", briefing)
	}
	if !strings.Contains(briefing, "Build API Module") {
		t.Error("expected sibling task 'Build API Module' in concurrency notice")
	}
	if !strings.Contains(briefing, "Build Store Module") {
		t.Error("expected sibling task 'Build Store Module' in concurrency notice")
	}
}

// TestBriefingIncludesFileOwnership verifies file ownership section is present.
func TestBriefingIncludesFileOwnership(t *testing.T) {
	task := &models.Task{
		ID:             "t1",
		Title:          "Build Auth",
		ModulePath:     "internal/auth/",
		ReadOnlyPaths:  []string{"internal/models/"},
		ForbiddenPaths: []string{"internal/store/", "internal/api/"},
	}

	briefing := AssembleBriefing(task, makeProject("", ""), nil)

	if !strings.Contains(briefing, "## File Ownership") {
		t.Error("expected 'File Ownership' section")
	}
	if !strings.Contains(briefing, "internal/auth/") {
		t.Error("expected owned module path in file ownership section")
	}
	if !strings.Contains(briefing, "internal/models/") {
		t.Error("expected read_only_paths in file ownership section")
	}
	if !strings.Contains(briefing, "internal/store/") {
		t.Error("expected forbidden_paths in file ownership section")
	}
}

// TestBriefingIncludesConsumedContracts verifies shared contracts appear for consuming tasks.
func TestBriefingIncludesConsumedContracts(t *testing.T) {
	consumedPayload, _ := json.Marshal(map[string]any{
		"provided_contracts": []any{},
		"consumed_contracts": []string{"UserRepository"},
	})
	task := &models.Task{
		ID:         "t1",
		Title:      "Build API",
		ModulePath: "internal/api/",
		Contracts:  json.RawMessage(consumedPayload),
	}

	providedPayload, _ := json.Marshal(map[string]any{
		"provided_contracts": []map[string]any{
			{"name": "UserRepository", "description": "Interface for user persistence", "file_path": "internal/store/user_repo.go"},
		},
		"consumed_contracts": []string{},
	})
	sibling := &models.Task{
		ID:         "t2",
		Title:      "Build Store",
		ModulePath: "internal/store/",
		Contracts:  json.RawMessage(providedPayload),
	}

	briefing := AssembleBriefing(task, makeProject("", ""), []*models.Task{sibling})

	if !strings.Contains(briefing, "## Shared Contracts") {
		t.Error("expected 'Shared Contracts' section when task has consumed contracts")
	}
	if !strings.Contains(briefing, "UserRepository") {
		t.Error("expected contract 'UserRepository' in shared contracts section")
	}
}

// TestBriefingGracefulWithEmptyProjectContext verifies no panic and no empty header for nil/empty context.
func TestBriefingGracefulWithEmptyProjectContext(t *testing.T) {
	task := makeTask("t1", "Build Auth", "internal/auth/")

	// Test nil project context
	projectNilCtx := &models.Project{ID: "p1"}
	briefing := AssembleBriefing(task, projectNilCtx, nil)
	if strings.Contains(briefing, "## Project Context") {
		t.Error("briefing should NOT include 'Project Context' header when context is nil")
	}

	// Test empty JSON object project context
	projectEmptyCtx := makeProject("{}", "")
	briefing2 := AssembleBriefing(task, projectEmptyCtx, nil)
	if strings.Contains(briefing2, "## Project Context") {
		t.Error("briefing should NOT include 'Project Context' header when context is empty object")
	}

	// Should not panic with nil project
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("AssembleBriefing panicked with nil project: %v", r)
			}
		}()
		AssembleBriefing(task, nil, nil)
	}()
}

// TestBriefingSiblingListedByTitleAndModule verifies sibling tasks include module paths.
func TestBriefingSiblingListedByTitleAndModule(t *testing.T) {
	task := makeTask("t1", "Build Auth", "internal/auth/")
	siblings := []*models.Task{
		makeTask("t2", "Build API", "internal/api/"),
	}

	briefing := AssembleBriefing(task, makeProject("", ""), siblings)

	if !strings.Contains(briefing, "internal/api/") {
		t.Error("expected sibling module path 'internal/api/' listed in concurrency notice")
	}
	if !strings.Contains(briefing, "Build API") {
		t.Error("expected sibling title 'Build API' listed in concurrency notice")
	}
}

// TestBriefingRulesSection verifies the rules section is present.
func TestBriefingRulesSection(t *testing.T) {
	task := makeTask("t1", "Build Auth", "internal/auth/")
	briefing := AssembleBriefing(task, makeProject("", ""), nil)

	if !strings.Contains(briefing, "## Rules") {
		t.Error("expected '## Rules' section in briefing")
	}
	if !strings.Contains(briefing, "IMPORTANT") {
		t.Error("expected IMPORTANT note in briefing")
	}
}
