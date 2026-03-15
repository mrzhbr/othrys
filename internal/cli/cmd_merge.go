package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/moritzhuber/othrys/internal/git"
	"github.com/spf13/cobra"
)

// NewMergeCmd creates the "merge" subcommand.
func NewMergeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "merge",
		Short: "Check merge readiness",
	}

	cmd.AddCommand(newMergeCheckCmd())
	cmd.AddCommand(newMergeStatusCmd())

	return cmd
}

func newMergeCheckCmd() *cobra.Command {
	var forcePreview bool
	var target string
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check if all tasks are complete and preview merge conflicts",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := globalCfg.ProjectID
			if projectID == "" {
				return fmt.Errorf("--project-id is required")
			}

			client := newClient()

			// Step 1: Server-side readiness check (no git dependency)
			fmt.Println("=== Server Readiness Check ===")
			result, err := client.CheckMerge(projectID)
			if err != nil {
				return fmt.Errorf("server check failed: %w", err)
			}

			ready, _ := result["ready"].(bool)
			allCompleted, _ := result["all_tasks_completed"].(bool)

			fmt.Printf("All tasks completed: %v\n", allCompleted)
			fmt.Printf("Ready for merge:     %v\n", ready)

			// Show pending tasks
			if pendingRaw, ok := result["pending_tasks"].([]any); ok && len(pendingRaw) > 0 {
				fmt.Printf("\nPending tasks (%d):\n", len(pendingRaw))
				for _, t := range pendingRaw {
					task, _ := t.(map[string]any)
					title, _ := task["title"].(string)
					status, _ := task["status"].(string)
					fmt.Printf("  - %s (%s)\n", title, status)
				}
			}

			// Show active claims
			if claimsRaw, ok := result["active_claims"].([]any); ok && len(claimsRaw) > 0 {
				fmt.Printf("\nActive claims (%d):\n", len(claimsRaw))
				for _, c := range claimsRaw {
					claim, _ := c.(map[string]any)
					path, _ := claim["path"].(string)
					agentID, _ := claim["agent_id"].(string)
					fmt.Printf("  - %s (held by %s)\n", path, agentID)
				}
			}

			// Step 2: Local git conflict preview (optional)
			if ready || forcePreview {
				fmt.Println("\n=== Local Conflict Preview ===")
				branches := extractBranches(result)
				if len(branches) == 0 {
					fmt.Println("No agent branches to preview.")
					return nil
				}

				if target == "" {
					target = "main"
				}

				repoPath, err := os.Getwd()
				if err != nil {
					repoPath = "."
				}

				gitSvc := git.NewLocalGitService()
				preview, err := gitSvc.PreviewMerge(repoPath, branches, target)
				if err != nil {
					fmt.Printf("WARNING: git preview failed (git may not be available): %v\n", err)
					return nil
				}

				if preview.HasConflicts {
					fmt.Printf("CONFLICTS detected in %d file(s):\n", len(preview.ConflictingFiles))
					for _, f := range preview.ConflictingFiles {
						fmt.Printf("  CONFLICT: %s\n", f)
					}
				} else {
					fmt.Printf("No conflicts detected. %d file(s) can be cleanly merged.\n", len(preview.CleanFiles))
				}
			}

			return nil
		},
	}
	cmd.Flags().BoolVar(&forcePreview, "force-preview", false, "Run local git conflict preview even if server says not ready")
	cmd.Flags().StringVar(&target, "target", "main", "Target branch for merge preview")
	return cmd
}

func newMergeStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show merge readiness status",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := globalCfg.ProjectID
			if projectID == "" {
				return fmt.Errorf("--project-id is required")
			}
			client := newClient()
			result, err := client.CheckMerge(projectID)
			if err != nil {
				return err
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
			return nil
		},
	}
}

func extractBranches(result map[string]any) []string {
	raw, ok := result["branches"].([]any)
	if !ok {
		return nil
	}
	var branches []string
	for _, b := range raw {
		if s, ok := b.(string); ok {
			branches = append(branches, s)
		}
	}
	return branches
}
