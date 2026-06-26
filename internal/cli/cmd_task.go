package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewTaskCmd creates the "task" subcommand.
func NewTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage tasks",
	}

	cmd.AddCommand(newTaskListCmd())
	cmd.AddCommand(newTaskMineCmd())
	cmd.AddCommand(newTaskCreateCmd())
	cmd.AddCommand(newTaskApproveCmd())
	cmd.AddCommand(newTaskAssignCmd())
	cmd.AddCommand(newTaskUpdateCmd())

	return cmd
}

func newTaskListCmd() *cobra.Command {
	var status string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := globalCfg.ProjectID
			if projectID == "" {
				return fmt.Errorf("--project-id is required")
			}
			client := newClient()
			result, err := client.ListTasks(projectID, status)
			if err != nil {
				return err
			}
			tasks, _ := result["tasks"].([]any)
			if len(tasks) == 0 {
				fmt.Println("No tasks found.")
				return nil
			}
			fmt.Printf("%-36s  %-40s  %-12s  %-30s\n", "ID", "TITLE", "STATUS", "MODULE")
			fmt.Printf("%-36s  %-40s  %-12s  %-30s\n",
				"------------------------------------",
				"----------------------------------------",
				"------------",
				"------------------------------")
			for _, t := range tasks {
				task, _ := t.(map[string]any)
				id, _ := task["id"].(string)
				title, _ := task["title"].(string)
				st, _ := task["status"].(string)
				module, _ := task["module_path"].(string)
				if len(id) > 8 {
					id = id[:8]
				}
				if len(title) > 40 {
					title = title[:37] + "..."
				}
				fmt.Printf("%-36s  %-40s  %-12s  %-30s\n", id, title, st, module)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (proposed, approved, assigned, in_progress, completed, failed)")
	return cmd
}

func newTaskMineCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mine",
		Short: "Show tasks assigned to your agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := globalCfg.ProjectID
			agentID := globalCfg.AgentID
			if projectID == "" {
				return fmt.Errorf("--project-id is required")
			}
			if agentID == "" {
				return fmt.Errorf("--agent-id is required")
			}
			client := newClient()

			// Get all non-completed tasks and filter by agent
			found := false
			for _, status := range []string{"assigned", "in_progress"} {
				result, err := client.ListTasks(projectID, status)
				if err != nil {
					return err
				}
				tasks, _ := result["tasks"].([]any)
				for _, t := range tasks {
					task, _ := t.(map[string]any)
					assignedTo, _ := task["assigned_agent_id"].(string)
					if assignedTo != agentID {
						continue
					}
					if !found {
						fmt.Println("Your assigned tasks:")
						found = true
					}
					id, _ := task["id"].(string)
					title, _ := task["title"].(string)
					st, _ := task["status"].(string)
					module, _ := task["module_path"].(string)
					desc, _ := task["description"].(string)
					branch, _ := task["branch_name"].(string)
					fmt.Printf("━━━ %s ━━━\n", title)
					fmt.Printf("  ID:          %s\n", id)
					fmt.Printf("  Status:      %s\n", st)
					fmt.Printf("  Module:      %s\n", module)
					fmt.Printf("  Branch:      %s\n", branch)
					fmt.Printf("  Description: %s\n\n", desc)
				}
			}
			if !found {
				fmt.Println("No tasks assigned to you.")
			}
			return nil
		},
	}
}

func newTaskCreateCmd() *cobra.Command {
	var title, description, module string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new task",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := globalCfg.ProjectID
			if projectID == "" {
				return fmt.Errorf("--project-id is required")
			}
			client := newClient()
			result, err := client.CreateTask(projectID, title, description, module)
			if err != nil {
				return err
			}
			id, _ := result["id"].(string)
			fmt.Printf("Task created: %s\n", id)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Task title (required)")
	cmd.Flags().StringVar(&description, "description", "", "Task description")
	cmd.Flags().StringVar(&module, "module", "", "Module path this task covers")
	_ = cmd.MarkFlagRequired("title")
	return cmd
}

func newTaskApproveCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "approve [task-id]",
		Short: "Approve a proposed task (or --all to approve all proposed tasks)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()

			if all {
				projectID := globalCfg.ProjectID
				if projectID == "" {
					return fmt.Errorf("--project-id is required with --all")
				}
				result, err := client.ListTasks(projectID, "proposed")
				if err != nil {
					return err
				}
				tasks, _ := result["tasks"].([]any)
				if len(tasks) == 0 {
					fmt.Println("No proposed tasks to approve.")
					return nil
				}
				count := 0
				for _, t := range tasks {
					task, _ := t.(map[string]any)
					id, _ := task["id"].(string)
					title, _ := task["title"].(string)
					if _, err := client.ApproveTask(id); err != nil {
						fmt.Printf("  ✗ Failed to approve %q: %v\n", title, err)
					} else {
						fmt.Printf("  ✓ Approved: %s\n", title)
						count++
					}
				}
				fmt.Printf("\n%d tasks approved.\n", count)
				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("provide a task ID or use --all")
			}
			result, err := client.ApproveTask(args[0])
			if err != nil {
				return err
			}
			st, _ := result["status"].(string)
			fmt.Printf("Task %s approved (status: %s)\n", args[0], st)
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Approve all proposed tasks")
	return cmd
}

func newTaskAssignCmd() *cobra.Command {
	var agentID string
	cmd := &cobra.Command{
		Use:   "assign <task-id>",
		Short: "Assign a task to an agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if agentID == "" {
				agentID = globalCfg.AgentID
			}
			if agentID == "" {
				return fmt.Errorf("--agent-id is required")
			}
			client := newClient()
			result, err := client.AssignTask(args[0], agentID)
			if err != nil {
				return err
			}
			branch, _ := result["branch_name"].(string)
			fmt.Printf("Task %s assigned. Branch: %s\n", args[0], branch)
			return nil
		},
	}
	cmd.Flags().StringVar(&agentID, "agent-id", "", "Agent ID to assign to")
	return cmd
}

func newTaskUpdateCmd() *cobra.Command {
	var status string
	cmd := &cobra.Command{
		Use:   "update <task-id>",
		Short: "Update task status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if status == "" {
				return fmt.Errorf("--status is required")
			}
			client := newClient()
			_, err := client.UpdateTaskStatus(args[0], status)
			if err != nil {
				return err
			}
			fmt.Printf("Task %s updated to status: %s\n", args[0], status)
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "New status (in_progress, completed, failed)")
	return cmd
}
