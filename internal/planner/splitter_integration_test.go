package planner

import (
	"encoding/json"
	"testing"
)

// TestSplitIntegrationAutoPopulateBoundaries tests that autoPopulateBoundaries is called
// during the split flow and correctly sets forbidden_paths and read_only_paths.
func TestSplitIntegrationAutoPopulateBoundaries(t *testing.T) {
	// autoPopulateBoundaries is called before validateProposedTasks in Split.
	// We test it directly here since testing through Split requires a real DB.
	tasks := []ProposedTask{
		{
			Title:       "Auth module",
			Description: "Implement authentication with sessions and JWTs",
			ModulePath:  "internal/auth/",
			ProvidedContracts: []Contract{
				{Name: "AuthService", Description: "Interface for auth operations", FilePath: "internal/auth/service.go"},
			},
		},
		{
			Title:             "API module",
			Description:       "Implement REST API handlers for user management",
			ModulePath:        "internal/api/",
			ConsumedContracts: []string{"AuthService"},
		},
		{
			Title:       "Store module",
			Description: "Implement database access layer with pgx",
			ModulePath:  "internal/store/",
		},
	}

	result := autoPopulateBoundaries(tasks)

	// Auth: forbidden = api + store, read_only = nothing
	auth := result[0]
	if !containsPath(auth.ForbiddenPaths, "internal/api/") {
		t.Error("auth task should forbid internal/api/")
	}
	if !containsPath(auth.ForbiddenPaths, "internal/store/") {
		t.Error("auth task should forbid internal/store/")
	}
	if containsPath(auth.ForbiddenPaths, "internal/auth/") {
		t.Error("auth task should NOT forbid its own module path")
	}

	// API: forbidden = auth + store, read_only = auth (consumes AuthService)
	api := result[1]
	if !containsPath(api.ForbiddenPaths, "internal/auth/") {
		t.Error("api task should forbid internal/auth/")
	}
	if !containsPath(api.ForbiddenPaths, "internal/store/") {
		t.Error("api task should forbid internal/store/")
	}
	if !containsPath(api.ReadOnlyPaths, "internal/auth/") {
		t.Errorf("api task should have internal/auth/ in read_only_paths (consumes AuthService), got %v", api.ReadOnlyPaths)
	}

	// Store: forbidden = auth + api, read_only = nothing
	store := result[2]
	if !containsPath(store.ForbiddenPaths, "internal/auth/") {
		t.Error("store task should forbid internal/auth/")
	}
	if !containsPath(store.ForbiddenPaths, "internal/api/") {
		t.Error("store task should forbid internal/api/")
	}
	if len(store.ReadOnlyPaths) != 0 {
		t.Errorf("store task should have empty read_only_paths, got %v", store.ReadOnlyPaths)
	}
}

// TestSplitIntegrationValidationCatchesBadData tests that validation catches
// all classes of bad data after autoPopulateBoundaries has run.
func TestSplitIntegrationValidationCatchesBadData(t *testing.T) {
	t.Run("overlapping paths blocked", func(t *testing.T) {
		tasks := []ProposedTask{
			{Title: "Parent", ModulePath: "src/", Description: "A task with enough description text here"},
			{Title: "Child", ModulePath: "src/auth/", Description: "A task with enough description text here"},
		}
		if err := validateProposedTasks(tasks); err == nil {
			t.Error("expected error for overlapping paths")
		}
	})

	t.Run("cycle detection", func(t *testing.T) {
		tasks := []ProposedTask{
			{Title: "A", ModulePath: "src/a/", Description: "Task A with enough description here", DependsOn: []string{"B"}},
			{Title: "B", ModulePath: "src/b/", Description: "Task B with enough description here", DependsOn: []string{"A"}},
		}
		if err := validateProposedTasks(tasks); err == nil {
			t.Error("expected error for dependency cycle")
		}
	})

	t.Run("missing contract", func(t *testing.T) {
		tasks := []ProposedTask{
			{
				Title:             "Consumer",
				ModulePath:        "src/a/",
				Description:       "Consumes a contract that does not exist",
				ConsumedContracts: []string{"NonExistentContract"},
			},
		}
		if err := validateProposedTasks(tasks); err == nil {
			t.Error("expected error for missing provided contract")
		}
	})
}

// TestSplitIntegrationProposedTaskToContractMapping verifies that the JSON
// marshaling used in Split correctly maps ProposedTask contract fields.
func TestSplitIntegrationProposedTaskToContractMapping(t *testing.T) {
	pt := ProposedTask{
		Title:       "Task A",
		Description: "Description of task A with sufficient length",
		ModulePath:  "src/a/",
		ProvidedContracts: []Contract{
			{Name: "AInterface", Description: "The A interface", FilePath: "src/a/iface.go"},
		},
		ConsumedContracts: []string{"BInterface"},
	}

	// Simulate the forward mapping done in Split (Task 3.4)
	contractsPayload := struct {
		Provided []Contract `json:"provided_contracts"`
		Consumed []string   `json:"consumed_contracts"`
	}{
		Provided: pt.ProvidedContracts,
		Consumed: pt.ConsumedContracts,
	}
	contractsJSON, err := json.Marshal(contractsPayload)
	if err != nil {
		t.Fatalf("marshal contracts: %v", err)
	}

	// Simulate the reverse mapping done in GenerateContracts (Task 4.6)
	var reversePayload struct {
		Provided []Contract `json:"provided_contracts"`
		Consumed []string   `json:"consumed_contracts"`
	}
	if err := json.Unmarshal(contractsJSON, &reversePayload); err != nil {
		t.Fatalf("unmarshal contracts: %v", err)
	}

	if len(reversePayload.Provided) != 1 {
		t.Errorf("expected 1 provided contract, got %d", len(reversePayload.Provided))
	}
	if reversePayload.Provided[0].Name != "AInterface" {
		t.Errorf("expected contract name 'AInterface', got %q", reversePayload.Provided[0].Name)
	}
	if len(reversePayload.Consumed) != 1 || reversePayload.Consumed[0] != "BInterface" {
		t.Errorf("expected consumed contract 'BInterface', got %v", reversePayload.Consumed)
	}
}

func containsPath(paths []string, target string) bool {
	for _, p := range paths {
		if p == target {
			return true
		}
	}
	return false
}
