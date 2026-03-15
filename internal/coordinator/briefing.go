package coordinator

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/moritzhuber/othrys/internal/models"
)

// AssembleBriefing builds a markdown briefing for an agent about to start a task.
// It includes project context, concurrency notice, file ownership boundaries,
// shared contracts, integration points, and rules for collaborative coding.
//
// If ProjectContext is empty/nil, the Project Context section is omitted gracefully.
// If there are no sibling tasks, the concurrency notice reflects solo execution.
func AssembleBriefing(task *models.Task, project *models.Project, siblingTasks []*models.Task) string {
	var sb strings.Builder

	// Section 1: Project Context (omit if empty)
	if project != nil && project.ProjectContext != nil {
		var ctx map[string]any
		if err := json.Unmarshal(project.ProjectContext, &ctx); err == nil && len(ctx) > 0 {
			sb.WriteString("## Project Context\n\n")
			if v, ok := ctx["tech_stack"]; ok {
				sb.WriteString(fmt.Sprintf("**Tech Stack:** %v\n\n", v))
			}
			if v, ok := ctx["module_path"]; ok {
				sb.WriteString(fmt.Sprintf("**Module/Package Path:** %v\n\n", v))
			}
			if v, ok := ctx["directory_tree"]; ok {
				sb.WriteString(fmt.Sprintf("**Directory Structure:** %v\n\n", v))
			}
			if v, ok := ctx["conventions"]; ok {
				sb.WriteString(fmt.Sprintf("**Conventions:** %v\n\n", v))
			}
			if v, ok := ctx["additional_context"]; ok {
				sb.WriteString(fmt.Sprintf("**Additional Context:** %v\n\n", v))
			}
		}
	}

	// Section 2: Your Task
	sb.WriteString("## Your Task\n\n")
	sb.WriteString(fmt.Sprintf("**Title:** %s\n\n", task.Title))
	sb.WriteString(fmt.Sprintf("**Module Path:** %s\n\n", task.ModulePath))
	sb.WriteString(fmt.Sprintf("**Description:**\n\n%s\n\n", task.Description))

	// Section 3: Concurrency Notice
	sb.WriteString("## Concurrency Notice\n\n")
	if len(siblingTasks) == 0 {
		sb.WriteString("You are the only agent currently working on this project.\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("You are one of %d agents working simultaneously. The following tasks are being worked on in parallel by other agents:\n\n", len(siblingTasks)+1))
		for _, sibling := range siblingTasks {
			sb.WriteString(fmt.Sprintf("- **%s** (module: `%s`)\n", sibling.Title, sibling.ModulePath))
		}
		sb.WriteString("\nDo NOT create files in their module paths. Do NOT assume their work is available to you yet.\n\n")
	}

	// Section 4: File Ownership
	sb.WriteString("## File Ownership\n\n")
	sb.WriteString(fmt.Sprintf("- **You OWN (read/write):** `%s`\n", task.ModulePath))
	if len(task.ReadOnlyPaths) > 0 {
		sb.WriteString("- **You may READ (but not write):**\n")
		for _, p := range task.ReadOnlyPaths {
			sb.WriteString(fmt.Sprintf("  - `%s`\n", p))
		}
	}
	if len(task.ForbiddenPaths) > 0 {
		sb.WriteString("- **You must NOT touch:**\n")
		for _, p := range task.ForbiddenPaths {
			sb.WriteString(fmt.Sprintf("  - `%s`\n", p))
		}
	}
	sb.WriteString("\n")

	// Section 5: Shared Contracts
	// Include contracts from project.SharedContracts and from sibling tasks' provided_contracts
	// that match this task's consumed_contracts
	contracts := gatherRelevantContracts(task, project, siblingTasks)
	if len(contracts) > 0 {
		sb.WriteString("## Shared Contracts\n\n")
		sb.WriteString("The following shared interfaces/types are available for your use:\n\n")
		for _, c := range contracts {
			sb.WriteString(fmt.Sprintf("### %s\n\n", c.name))
			sb.WriteString(fmt.Sprintf("**File:** `%s`\n\n", c.filePath))
			sb.WriteString(fmt.Sprintf("%s\n\n", c.description))
		}
	}

	// Section 6: Integration Points
	if len(task.IntegrationPoints) > 0 {
		sb.WriteString("## Integration Points\n\n")
		for _, ip := range task.IntegrationPoints {
			sb.WriteString(fmt.Sprintf("- %s\n", ip))
		}
		sb.WriteString("\n")
	}

	// Section 7: Rules
	sb.WriteString("## Rules\n\n")
	sb.WriteString("- Use the project's actual module/package path as specified in Project Context\n")
	sb.WriteString("- Do not create stub/mock implementations of other agents' modules\n")
	sb.WriteString("- If you need a type defined by another task's contract, import it -- do not redefine it\n")
	sb.WriteString("- Do not produce handoff documents -- all tasks execute concurrently\n")
	sb.WriteString("- IMPORTANT: Only create or modify files under your owned module path. Read the Shared Contracts section above before writing any code.\n")

	return sb.String()
}

// briefingContract is an internal helper for contract display.
type briefingContract struct {
	name        string
	description string
	filePath    string
}

// gatherRelevantContracts collects contracts relevant to this task from:
// 1. project.SharedContracts (global shared contracts set by PM)
// 2. Sibling tasks' provided_contracts that match this task's consumed_contracts
func gatherRelevantContracts(task *models.Task, project *models.Project, siblingTasks []*models.Task) []briefingContract {
	var result []briefingContract

	// Parse this task's consumed_contracts for case-insensitive lookup
	var consumedNames map[string]bool
	if task.Contracts != nil {
		var payload struct {
			Consumed []string `json:"consumed_contracts"`
		}
		if err := json.Unmarshal(task.Contracts, &payload); err == nil {
			consumedNames = make(map[string]bool, len(payload.Consumed))
			for _, n := range payload.Consumed {
				consumedNames[strings.ToLower(n)] = true
			}
		}
	}

	// From project shared contracts
	if project != nil && project.SharedContracts != nil {
		var shared []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			FilePath    string `json:"file_path"`
		}
		if err := json.Unmarshal(project.SharedContracts, &shared); err == nil {
			for _, c := range shared {
				if len(consumedNames) == 0 || consumedNames[strings.ToLower(c.Name)] {
					result = append(result, briefingContract{
						name:        c.Name,
						description: c.Description,
						filePath:    c.FilePath,
					})
				}
			}
		}
	}

	// From sibling tasks' provided_contracts
	if len(consumedNames) > 0 {
		seen := make(map[string]bool)
		for _, sibling := range siblingTasks {
			if sibling.Contracts == nil {
				continue
			}
			var payload struct {
				Provided []struct {
					Name        string `json:"name"`
					Description string `json:"description"`
					FilePath    string `json:"file_path"`
				} `json:"provided_contracts"`
			}
			if err := json.Unmarshal(sibling.Contracts, &payload); err != nil {
				continue
			}
			for _, c := range payload.Provided {
				key := strings.ToLower(c.Name)
				if consumedNames[key] && !seen[key] {
					seen[key] = true
					result = append(result, briefingContract{
						name:        c.Name,
						description: c.Description,
						filePath:    c.FilePath,
					})
				}
			}
		}
	}

	return result
}
