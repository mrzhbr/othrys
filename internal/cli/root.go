package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	serverFlag    string
	apiKeyFlag    string
	agentNameFlag string
	agentIDFlag   string
	projectIDFlag string
)

// globalCfg holds the resolved CLI configuration.
var globalCfg *CLIConfig

// NewRootCmd creates the root cobra command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "othrys",
		Short: "Othrys CLI — multi-agent collaboration coordinator",
		Long: `Othrys is a coordination server for multi-agent software development.
Use this CLI to manage projects, tasks, claims, and agents.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cfg := LoadConfig()

			// Flags override config file/env
			if serverFlag != "" {
				cfg.ServerURL = serverFlag
			}
			if apiKeyFlag != "" {
				cfg.APIKey = apiKeyFlag
			}
			if agentNameFlag != "" {
				cfg.AgentName = agentNameFlag
			}
			if agentIDFlag != "" {
				cfg.AgentID = agentIDFlag
			}
			if projectIDFlag != "" {
				cfg.ProjectID = projectIDFlag
			}

			globalCfg = cfg
		},
	}

	root.PersistentFlags().StringVar(&serverFlag, "server", "", "Othrys server URL (default: http://localhost:8080)")
	root.PersistentFlags().StringVar(&apiKeyFlag, "api-key", "", "Project API key")
	root.PersistentFlags().StringVar(&agentNameFlag, "agent-name", "", "Agent name")
	root.PersistentFlags().StringVar(&agentIDFlag, "agent-id", "", "Agent ID")
	root.PersistentFlags().StringVar(&projectIDFlag, "project-id", "", "Project ID")

	// Add subcommands
	root.AddCommand(NewProjectCmd())
	root.AddCommand(NewTaskCmd())
	root.AddCommand(NewClaimCmd())
	root.AddCommand(NewAgentCmd())
	root.AddCommand(NewMergeCmd())

	return root
}

// Execute runs the CLI.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// newClient creates an HTTP client from the global config.
func newClient() *Client {
	if globalCfg == nil {
		globalCfg = LoadConfig()
	}
	return NewClient(globalCfg.ServerURL, globalCfg.APIKey, globalCfg.AgentID)
}
