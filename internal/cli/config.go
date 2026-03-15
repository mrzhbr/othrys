package cli

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// CLIConfig holds CLI configuration.
type CLIConfig struct {
	ServerURL string
	APIKey    string
	AgentName string
	AgentID   string
	ProjectID string
}

// LoadConfig loads CLI configuration from flags, env vars, and config file.
// Priority: flags > env vars > config file.
func LoadConfig() *CLIConfig {
	viper.SetConfigName(".othrys")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath(".")

	viper.AutomaticEnv()
	viper.SetEnvPrefix("OTHRYS")

	// Set defaults
	viper.SetDefault("server", "http://localhost:8080")

	_ = viper.ReadInConfig()

	return &CLIConfig{
		ServerURL: viper.GetString("server"),
		APIKey:    viper.GetString("api_key"),
		AgentName: viper.GetString("agent_name"),
		AgentID:   viper.GetString("agent_id"),
		ProjectID: viper.GetString("project_id"),
	}
}

// SaveConfig writes the config to ~/.othrys.yaml
func SaveConfig(cfg *CLIConfig) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	viper.Set("server", cfg.ServerURL)
	viper.Set("api_key", cfg.APIKey)
	viper.Set("agent_name", cfg.AgentName)
	viper.Set("agent_id", cfg.AgentID)
	viper.Set("project_id", cfg.ProjectID)

	return viper.WriteConfigAs(home + "/.othrys.yaml")
}
