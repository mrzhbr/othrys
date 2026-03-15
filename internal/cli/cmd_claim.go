package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewClaimCmd creates the "claim" subcommand.
func NewClaimCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim",
		Short: "Manage path claims",
	}

	cmd.AddCommand(newClaimRequestCmd())
	cmd.AddCommand(newClaimReleaseCmd())
	cmd.AddCommand(newClaimListCmd())

	return cmd
}

func newClaimRequestCmd() *cobra.Command {
	var taskID, path, claimType string
	cmd := &cobra.Command{
		Use:   "request",
		Short: "Request a claim on a path",
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := globalCfg.AgentID
			if agentID == "" {
				return fmt.Errorf("--agent-id is required")
			}
			if taskID == "" {
				return fmt.Errorf("--task is required")
			}
			if path == "" {
				return fmt.Errorf("--path is required")
			}
			if claimType == "" {
				claimType = "exclusive"
			}

			client := newClient()
			result, err := client.RequestClaim(agentID, taskID, path, claimType)
			if err != nil {
				// Check for conflict (409)
				if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == 409 {
					fmt.Printf("DENIED: path %q is claimed by another agent\n", path)
					fmt.Printf("Details: %s\n", apiErr.Message)
					return nil
				}
				return err
			}

			granted, _ := result["granted"].(bool)
			if granted {
				claim, _ := result["claim"].(map[string]any)
				claimID, _ := claim["id"].(string)
				fmt.Printf("GRANTED: claim %s on path %q (%s)\n", claimID, path, claimType)
			} else {
				fmt.Printf("DENIED: path %q conflicts with existing claims\n", path)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&taskID, "task", "", "Task ID this claim is for (required)")
	cmd.Flags().StringVar(&path, "path", "", "Path to claim (required)")
	cmd.Flags().StringVar(&claimType, "type", "exclusive", "Claim type: exclusive or shared_read")
	return cmd
}

func newClaimReleaseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "release <claim-id>",
		Short: "Release a claim",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()
			if err := client.ReleaseClaim(args[0]); err != nil {
				return err
			}
			fmt.Printf("Claim %s released.\n", args[0])
			return nil
		},
	}
}

func newClaimListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List active claims",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := globalCfg.ProjectID
			if projectID == "" {
				return fmt.Errorf("--project-id is required")
			}
			client := newClient()
			result, err := client.ListClaims(projectID)
			if err != nil {
				return err
			}
			claims, _ := result["claims"].([]any)
			if len(claims) == 0 {
				fmt.Println("No active claims.")
				return nil
			}
			fmt.Printf("%-36s  %-40s  %-12s  %-36s\n", "CLAIM ID", "PATH", "TYPE", "AGENT ID")
			for _, c := range claims {
				claim, _ := c.(map[string]any)
				id, _ := claim["id"].(string)
				path, _ := claim["path"].(string)
				ct, _ := claim["claim_type"].(string)
				agent, _ := claim["agent_id"].(string)
				fmt.Printf("%-36s  %-40s  %-12s  %-36s\n", id, path, ct, agent)
			}
			return nil
		},
	}
}
