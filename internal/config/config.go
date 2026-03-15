package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Server settings
	Port int

	// Database settings
	DatabaseURL string

	// Auth settings
	APISecret string

	// LLM settings
	LLMProvider string
	LLMAPIKey   string
	LLMModel    string
}

// Load reads configuration from environment variables.
// Required: DATABASE_URL, API_SECRET, LLM_API_KEY
// Optional (with defaults): PORT=8080, LLM_PROVIDER=anthropic, LLM_MODEL=claude-3-5-sonnet-20241022
func Load() (*Config, error) {
	cfg := &Config{
		Port:        8080,
		LLMProvider: "anthropic",
		LLMModel:    "claude-3-5-sonnet-20241022",
	}

	if portStr := os.Getenv("PORT"); portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid PORT value %q: %w", portStr, err)
		}
		cfg.Port = p
	}

	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	cfg.APISecret = os.Getenv("API_SECRET")
	if cfg.APISecret == "" {
		return nil, fmt.Errorf("API_SECRET is required")
	}

	cfg.LLMAPIKey = os.Getenv("LLM_API_KEY")

	if provider := os.Getenv("LLM_PROVIDER"); provider != "" {
		cfg.LLMProvider = provider
	}

	if model := os.Getenv("LLM_MODEL"); model != "" {
		cfg.LLMModel = model
	}

	return cfg, nil
}
