package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewAgentCmd creates the "agent" subcommand.
func NewAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage agents",
	}

	cmd.AddCommand(newAgentRegisterCmd())
	cmd.AddCommand(newAgentHeartbeatCmd())
	cmd.AddCommand(newAgentListCmd())

	return cmd
}

func newAgentRegisterCmd() *cobra.Command {
	var name, toolType string
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register an agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				name = globalCfg.AgentName
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if toolType == "" {
				toolType = "generic"
			}

			client := newClient()
			result, err := client.RegisterAgent(name, toolType)
			if err != nil {
				return err
			}
			id, _ := result["id"].(string)
			fmt.Printf("Agent registered!\n")
			fmt.Printf("  ID:   %s\n", id)
			fmt.Printf("  Name: %s\n", name)
			fmt.Printf("  Tool: %s\n", toolType)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Agent name")
	cmd.Flags().StringVar(&toolType, "tool", "generic", "Tool type: omo, cursor, copilot, generic")
	return cmd
}

func newAgentHeartbeatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "heartbeat",
		Short: "Send agent heartbeat",
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := globalCfg.AgentID
			if agentID == "" {
				return fmt.Errorf("--agent-id is required")
			}
			client := newClient()
			if err := client.Heartbeat(agentID); err != nil {
				return err
			}
			fmt.Printf("Heartbeat sent for agent %s\n", agentID)
			return nil
		},
	}
}

func newAgentListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List agents for a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := globalCfg.ProjectID
			if projectID == "" {
				return fmt.Errorf("--project-id is required")
			}
			client := newClient()
			result, err := client.ListAgents(projectID)
			if err != nil {
				return err
			}
			agents, _ := result["agents"].([]any)
			if len(agents) == 0 {
				fmt.Println("No agents registered.")
				return nil
			}
			fmt.Printf("%-36s  %-20s  %-10s  %-12s\n", "ID", "NAME", "TOOL", "STATUS")
			for _, a := range agents {
				agent, _ := a.(map[string]any)
				id, _ := agent["id"].(string)
				name, _ := agent["name"].(string)
				tool, _ := agent["tool_type"].(string)
				status, _ := agent["status"].(string)
				fmt.Printf("%-36s  %-20s  %-10s  %-12s\n", id, name, tool, status)
			}
			return nil
		},
	}
}
