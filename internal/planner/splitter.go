package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/moritzhuber/othrys/internal/models"
	"github.com/moritzhuber/othrys/internal/store"
)

// Splitter orchestrates LLM task splitting for a project.
type Splitter struct {
	provider  Provider
	taskStore *store.TaskStore
}

// NewSplitter creates a new Splitter.
func NewSplitter(provider Provider, taskStore *store.TaskStore) *Splitter {
	return &Splitter{provider: provider, taskStore: taskStore}
}

// Split loads the PM design from the project, calls the provider, validates output,
// and stores proposed tasks in the database. Returns the proposed tasks.
func (s *Splitter) Split(ctx context.Context, project *models.Project) ([]*models.Task, error) {
	if project.PMDesign == nil || string(project.PMDesign) == "null" {
		return nil, fmt.Errorf("project has no PM design document")
	}

	// Extract design text from JSONB
	var designText string
	if err := json.Unmarshal(project.PMDesign, &designText); err != nil {
		// Try as raw object — convert to string representation
		designText = string(project.PMDesign)
	}

	// Parse project config (base layer)
	var projectConfig map[string]any
	if project.Config != nil {
		_ = json.Unmarshal(project.Config, &projectConfig)
	}
	if projectConfig == nil {
		projectConfig = make(map[string]any)
	}

	// Merge ProjectContext on top — ProjectContext keys take precedence over Config keys on collision.
	// This allows the PM-supplied context to override any generic config values.
	if project.ProjectContext != nil && string(project.ProjectContext) != "null" && string(project.ProjectContext) != "{}" {
		var ctx2 map[string]any
		if err := json.Unmarshal(project.ProjectContext, &ctx2); err == nil {
			for k, v := range ctx2 {
				projectConfig[k] = v
			}
		}
	}

	// Call LLM provider
	proposed, err := s.provider.SplitDesign(designText, projectConfig)
	if err != nil {
		return nil, fmt.Errorf("LLM split failed: %w", err)
	}

	// Auto-populate file ownership boundaries before validation.
	// This sets forbidden_paths to other tasks' module_paths and read_only_paths
	// based on consumed contracts.
	proposed = autoPopulateBoundaries(proposed)

	// Validate output
	if err := validateProposedTasks(proposed); err != nil {
		return nil, fmt.Errorf("invalid LLM output: %w", err)
	}

	// Store proposed tasks (first pass: create without dependencies)
	var created []*models.Task
	titleToID := make(map[string]string)
	for _, pt := range proposed {
		// Marshal provided_contracts and consumed_contracts into Contracts JSON
		contractsPayload := struct {
			Provided []Contract `json:"provided_contracts"`
			Consumed []string   `json:"consumed_contracts"`
		}{
			Provided: pt.ProvidedContracts,
			Consumed: pt.ConsumedContracts,
		}
		contractsJSON, err := json.Marshal(contractsPayload)
		if err != nil {
			return nil, fmt.Errorf("marshal contracts for task %q: %w", pt.Title, err)
		}

		task := &models.Task{
			ProjectID:         project.ID,
			Title:             pt.Title,
			Description:       pt.Description,
			ModulePath:        pt.ModulePath,
			Status:            models.TaskStatusProposed,
			ReadOnlyPaths:     pt.ReadOnlyPaths,
			ForbiddenPaths:    pt.ForbiddenPaths,
			IntegrationPoints: pt.IntegrationPoints,
			Contracts:         json.RawMessage(contractsJSON),
		}
		t, err := s.taskStore.Create(ctx, task)
		if err != nil {
			return nil, fmt.Errorf("store task %q: %w", pt.Title, err)
		}
		created = append(created, t)
		titleToID[strings.ToLower(pt.Title)] = t.ID
	}

	// Second pass: resolve dependency titles to UUIDs and update
	for i, pt := range proposed {
		if len(pt.DependsOn) == 0 {
			continue
		}
		var depIDs []string
		for _, dep := range pt.DependsOn {
			if id, ok := titleToID[strings.ToLower(dep)]; ok {
				depIDs = append(depIDs, id)
			} else {
				log.Printf("[splitter] WARNING: task %q depends on unknown task %q — skipping dependency", pt.Title, dep)
			}
		}
		if len(depIDs) > 0 {
			created[i].DependsOn = depIDs
		}
	}

	return created, nil
}

// GenerateContracts implements the scaffold Pass 2 flow.
// It type-asserts the provider to ContractGenerator; returns an error if the
// provider does not support contract generation.
// It loads tasks from the store, reconstructs []ProposedTask (reverse mapping of Split),
// and delegates to the provider's GenerateContracts method.
func (s *Splitter) GenerateContracts(ctx context.Context, project *models.Project) ([]Contract, error) {
	contractGen, ok := s.provider.(ContractGenerator)
	if !ok {
		return nil, fmt.Errorf("provider does not support contract generation")
	}

	// Load tasks from the store
	tasks, err := s.taskStore.ListByProject(ctx, project.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("load tasks for contract generation: %w", err)
	}

	// Reverse-map models.Task → ProposedTask
	proposed := make([]ProposedTask, 0, len(tasks))
	for _, t := range tasks {
		pt := ProposedTask{
			Title:             t.Title,
			Description:       t.Description,
			ModulePath:        t.ModulePath,
			DependsOn:         t.DependsOn,
			ReadOnlyPaths:     t.ReadOnlyPaths,
			ForbiddenPaths:    t.ForbiddenPaths,
			IntegrationPoints: t.IntegrationPoints,
		}

		// Unmarshal Contracts JSON to extract ProvidedContracts and ConsumedContracts.
		// The forward mapping (Split) stored them as {"provided_contracts":...,"consumed_contracts":...}.
		if t.Contracts != nil && string(t.Contracts) != "null" {
			var contractsPayload struct {
				Provided []Contract `json:"provided_contracts"`
				Consumed []string   `json:"consumed_contracts"`
			}
			if err := json.Unmarshal(t.Contracts, &contractsPayload); err == nil {
				pt.ProvidedContracts = contractsPayload.Provided
				pt.ConsumedContracts = contractsPayload.Consumed
			}
		}

		proposed = append(proposed, pt)
	}

	// Build projectConfig using the same merge logic as Split
	var projectConfig map[string]any
	if project.Config != nil {
		_ = json.Unmarshal(project.Config, &projectConfig)
	}
	if projectConfig == nil {
		projectConfig = make(map[string]any)
	}
	if project.ProjectContext != nil && string(project.ProjectContext) != "null" && string(project.ProjectContext) != "{}" {
		var ctx2 map[string]any
		if err := json.Unmarshal(project.ProjectContext, &ctx2); err == nil {
			for k, v := range ctx2 {
				projectConfig[k] = v
			}
		}
	}

	return contractGen.GenerateContracts(proposed, projectConfig)
}

// autoPopulateBoundaries sets file ownership boundaries on each task:
//   - forbidden_paths: all other tasks' module_paths (tasks must not touch each other's modules)
//   - read_only_paths: module_paths of tasks whose contracts this task consumes
//
// Contract name matching uses case-insensitive comparison to prevent silent mismatches.
// The function is unexported — callers use Split which calls it automatically.
func autoPopulateBoundaries(tasks []ProposedTask) []ProposedTask {
	// Build map of lowercase contract name → provider module_path
	contractToModule := make(map[string]string)
	for _, t := range tasks {
		for _, c := range t.ProvidedContracts {
			contractToModule[strings.ToLower(c.Name)] = t.ModulePath
		}
	}

	for i, t := range tasks {
		// forbidden_paths = all other tasks' module_paths
		var forbidden []string
		for _, other := range tasks {
			if other.ModulePath != t.ModulePath && other.ModulePath != "" {
				forbidden = append(forbidden, other.ModulePath)
			}
		}
		tasks[i].ForbiddenPaths = forbidden

		// read_only_paths = module_paths of tasks whose contracts this task consumes
		readOnlySet := make(map[string]bool)
		for _, consumedName := range t.ConsumedContracts {
			if modulePath, ok := contractToModule[strings.ToLower(consumedName)]; ok {
				if modulePath != t.ModulePath {
					readOnlySet[modulePath] = true
				}
			}
		}
		var readOnly []string
		for path := range readOnlySet {
			readOnly = append(readOnly, path)
		}
		tasks[i].ReadOnlyPaths = readOnly
	}

	return tasks
}

// validateProposedTasks checks that the LLM output is valid.
func validateProposedTasks(tasks []ProposedTask) error {
	if len(tasks) == 0 {
		return fmt.Errorf("no tasks returned")
	}

	// Build title → index for dependency cycle detection
	titleIdx := make(map[string]int)
	paths := make([]string, 0, len(tasks))

	// Build map of lowercase contract name → providing task title for cross-reference
	contractProviders := make(map[string]string)

	for i, t := range tasks {
		if t.Title == "" {
			return fmt.Errorf("task %d: title is required", i)
		}
		if len(t.Description) <= 20 {
			return fmt.Errorf("task %q: description must be more than 20 characters", t.Title)
		}
		if t.ModulePath == "" {
			return fmt.Errorf("task %q: module_path is required", t.Title)
		}
		if !strings.HasSuffix(t.ModulePath, "/") {
			return fmt.Errorf("task %q: module_path must end with '/' (got %q)", t.Title, t.ModulePath)
		}
		titleIdx[strings.ToLower(t.Title)] = i
		paths = append(paths, t.ModulePath)

		for _, c := range t.ProvidedContracts {
			contractProviders[strings.ToLower(c.Name)] = t.Title
		}
	}

	// Error on overlapping module paths (changed from warning to error per plan)
	for i := 0; i < len(paths); i++ {
		for j := i + 1; j < len(paths); j++ {
			if pathsOverlap(paths[i], paths[j]) {
				return fmt.Errorf("overlapping module paths: %q and %q — each task must own a distinct path prefix", paths[i], paths[j])
			}
		}
	}

	// Validate consumed_contracts: every consumed contract must be provided by another task.
	// Use case-insensitive comparison to catch mismatches like "UserRepository" vs "userRepository".
	for _, t := range tasks {
		for _, consumed := range t.ConsumedContracts {
			if _, ok := contractProviders[strings.ToLower(consumed)]; !ok {
				return fmt.Errorf("task %q consumes contract %q but no task provides it", t.Title, consumed)
			}
		}
	}

	// Detect dependency cycles using topological sort (Kahn's algorithm)
	if err := detectCycles(tasks, titleIdx); err != nil {
		return err
	}

	// Warn if forbidden_paths doesn't cover other tasks' module_paths
	for _, t := range tasks {
		for _, other := range tasks {
			if other.ModulePath == t.ModulePath {
				continue
			}
			found := false
			for _, fp := range t.ForbiddenPaths {
				if fp == other.ModulePath {
					found = true
					break
				}
			}
			if !found {
				log.Printf("[splitter] WARN: task %q forbidden_paths does not include %q — consider running autoPopulateBoundaries", t.Title, other.ModulePath)
			}
		}
	}

	return nil
}

// detectCycles uses Kahn's algorithm (topological sort) to detect dependency cycles.
// Returns an error if a cycle is found.
func detectCycles(tasks []ProposedTask, titleIdx map[string]int) error {
	n := len(tasks)
	inDegree := make([]int, n)
	adj := make([][]int, n)

	for i, t := range tasks {
		for _, dep := range t.DependsOn {
			if j, ok := titleIdx[strings.ToLower(dep)]; ok {
				adj[j] = append(adj[j], i)
				inDegree[i]++
			}
		}
	}

	queue := []int{}
	for i := 0; i < n; i++ {
		if inDegree[i] == 0 {
			queue = append(queue, i)
		}
	}

	visited := 0
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		visited++
		for _, next := range adj[cur] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if visited != n {
		return fmt.Errorf("dependency cycle detected in tasks — check depends_on fields")
	}
	return nil
}

// pathsOverlap returns true if one path is a prefix of the other.
func pathsOverlap(a, b string) bool {
	if a == b {
		return true
	}
	if strings.HasPrefix(a, b) || strings.HasPrefix(b, a) {
		return true
	}
	return false
}
