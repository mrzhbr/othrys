package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewProjectCmd creates the "project" subcommand.
func NewProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage Othrys projects",
	}

	cmd.AddCommand(newProjectCreateCmd())
	cmd.AddCommand(newProjectStatusCmd())
	cmd.AddCommand(newProjectDesignCmd())
	cmd.AddCommand(newProjectSplitCmd())

	return cmd
}

func newProjectCreateCmd() *cobra.Command {
	var name, repo string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new project",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()
			result, err := client.CreateProject(name, repo)
			if err != nil {
				return err
			}

			apiKey, _ := result["api_key"].(string)
			id, _ := result["id"].(string)

			fmt.Printf("Project created successfully!\n")
			fmt.Printf("  ID:      %s\n", id)
			fmt.Printf("  Name:    %s\n", name)
			fmt.Printf("\n*** SAVE THIS API KEY — it won't be shown again ***\n")
			fmt.Printf("  API Key: %s\n\n", apiKey)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Project name (required)")
	cmd.Flags().StringVar(&repo, "repo", "", "Git repository URL")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newProjectStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show project status",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()
			projectID := globalCfg.ProjectID
			if projectID == "" {
				return fmt.Errorf("--project-id is required")
			}

			result, err := client.GetProject(projectID)
			if err != nil {
				return err
			}

			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
			return nil
		},
	}
}

func newProjectDesignCmd() *cobra.Command {
	var filePath string
	cmd := &cobra.Command{
		Use:   "design",
		Short: "Upload PM design document",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := globalCfg.ProjectID
			if projectID == "" {
				return fmt.Errorf("--project-id is required")
			}

			var design any
			if filePath != "" {
				data, err := os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("read file: %w", err)
				}
				// Try JSON first, fall back to string
				if err := json.Unmarshal(data, &design); err != nil {
					design = string(data)
				}
			} else {
				return fmt.Errorf("--file is required")
			}

			client := newClient()
			_, err := client.UpdateDesign(projectID, design)
			if err != nil {
				return err
			}
			fmt.Println("PM design uploaded successfully.")
			return nil
		},
	}
	cmd.Flags().StringVar(&filePath, "file", "", "Path to design JSON file (required)")
	return cmd
}

func newProjectSplitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "split",
		Short: "Trigger LLM task splitting from the PM design",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := globalCfg.ProjectID
			if projectID == "" {
				return fmt.Errorf("--project-id is required")
			}

			fmt.Println("Splitting design into tasks (calling LLM)...")
			client := newClient()
			result, err := client.SplitTasks(projectID)
			if err != nil {
				return err
			}

			tasks, ok := result["tasks"].([]any)
			if !ok {
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("\n%d tasks proposed:\n\n", len(tasks))
			for i, t := range tasks {
				task, _ := t.(map[string]any)
				title, _ := task["title"].(string)
				module, _ := task["module_path"].(string)
				fmt.Printf("  %d. %s\n     Module: %s\n\n", i+1, title, module)
			}
			fmt.Println("Tasks are in 'proposed' status. Use 'othrys task approve <id>' to approve them.")
			return nil
		},
	}
}
