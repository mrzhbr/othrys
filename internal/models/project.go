package models

import (
	"encoding/json"
	"time"
)

// Project represents a collaboration project in the Othrys system.
type Project struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	RepoURL         string          `json:"repo_url"`
	APIKey          string          `json:"api_key,omitempty"`
	PMDesign        json.RawMessage `json:"pm_design,omitempty"`
	Config          json.RawMessage `json:"config"`
	SharedContracts json.RawMessage `json:"shared_contracts,omitempty"`
	ProjectContext  json.RawMessage `json:"project_context,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}
