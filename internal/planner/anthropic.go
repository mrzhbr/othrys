package planner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultAnthropicModel = "claude-3-5-sonnet-20241022"
const anthropicAPIURL = "https://api.anthropic.com/v1/messages"

// AnthropicProvider implements Provider using the Anthropic Claude API.
// It also implements ContractGenerator for the optional two-pass scaffold flow.
// Uses net/http directly — no SDK dependency.
type AnthropicProvider struct {
	APIKey string
	Model  string
}

// NewAnthropicProvider creates a new AnthropicProvider.
func NewAnthropicProvider(apiKey, model string) *AnthropicProvider {
	if model == "" {
		model = defaultAnthropicModel
	}
	return &AnthropicProvider{APIKey: apiKey, Model: model}
}

// SplitDesign calls the Anthropic Claude API to split a PM design into tasks.
func (p *AnthropicProvider) SplitDesign(design string, projectConfig map[string]any) ([]ProposedTask, error) {
	prompt := buildPrompt(design, projectConfig)

	reqBody, err := json.Marshal(map[string]any{
		"model":      p.Model,
		"max_tokens": 16384,
		"messages": []map[string]any{
			{"role": "user", "content": prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, anthropicAPIURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := (&http.Client{Timeout: 4 * time.Minute}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("call Anthropic API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Anthropic API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse Anthropic response format
	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	var text string
	for _, c := range apiResp.Content {
		if c.Type == "text" {
			text = c.Text
			break
		}
	}

	return parseTasksFromJSON(text)
}

// GenerateContracts implements ContractGenerator. It sends a second LLM prompt
// (Pass 2 of the two-pass split flow) that synthesizes shared interface/type
// contracts from the already-generated tasks. This is optional — the PM can skip
// Pass 2 for simple projects.
func (p *AnthropicProvider) GenerateContracts(tasks []ProposedTask, projectConfig map[string]any) ([]Contract, error) {
	prompt := buildContractsPrompt(tasks, projectConfig)

	reqBody, err := json.Marshal(map[string]any{
		"model":      p.Model,
		"max_tokens": 8192,
		"messages": []map[string]any{
			{"role": "user", "content": prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal contracts request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, anthropicAPIURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create contracts request: %w", err)
	}
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := (&http.Client{Timeout: 4 * time.Minute}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("call Anthropic API for contracts: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read contracts response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Anthropic API error %d: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("parse contracts response: %w", err)
	}

	var text string
	for _, c := range apiResp.Content {
		if c.Type == "text" {
			text = c.Text
			break
		}
	}

	return parseContractsFromJSON(text)
}

// buildPrompt creates the task-splitting prompt with optional project context injection.
// projectCtx may contain: tech_stack, module_path, directory_tree, conventions, additional_context.
func buildPrompt(design string, projectCtx map[string]any) string {
	var sb strings.Builder

	sb.WriteString("ROLE: You are a project coordinator splitting work for concurrent AI coding agents.\n\n")

	// Inject project context if available
	if len(projectCtx) > 0 {
		sb.WriteString("PROJECT CONTEXT:\n")
		if v, ok := projectCtx["tech_stack"]; ok {
			sb.WriteString(fmt.Sprintf("Tech Stack: %v\n", v))
		}
		if v, ok := projectCtx["module_path"]; ok {
			sb.WriteString(fmt.Sprintf("Module/Package Path: %v\n", v))
		}
		if v, ok := projectCtx["directory_tree"]; ok {
			sb.WriteString(fmt.Sprintf("Directory Structure: %v\n", v))
		}
		if v, ok := projectCtx["conventions"]; ok {
			sb.WriteString(fmt.Sprintf("Conventions: %v\n", v))
		}
		if v, ok := projectCtx["additional_context"]; ok {
			sb.WriteString(fmt.Sprintf("Additional Context: %v\n", v))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("PM DESIGN DOCUMENT:\n")
	sb.WriteString(design)
	sb.WriteString("\n\n")

	sb.WriteString(`OUTPUT FORMAT:
Return ONLY a valid JSON array of tasks with this exact structure:
[
  {
    "title": "Short task title",
    "description": "Detailed description of what the agent should implement (>20 chars)",
    "module_path": "path/to/module/",
    "depends_on": [],
    "read_only_paths": [],
    "forbidden_paths": [],
    "integration_points": [],
    "provided_contracts": [
      {
        "name": "InterfaceName",
        "description": "What it is and its signature/shape",
        "file_path": "path/to/file.go"
      }
    ],
    "consumed_contracts": ["ContractNameFromAnotherTask"]
  }
]

RULES:
- All file paths must be relative to the project root
- module_path must be a directory path ending with "/" (e.g., "internal/auth/", "cmd/server/")
- module_paths must NOT overlap (no task should claim a path that is a prefix of another's path)
- Each task's forbidden_paths must include all other tasks' module_paths
- If multiple tasks need the same type or interface, define it as a provided_contract on exactly one task and a consumed_contract (plain string name, NOT an object) on the others
- Tasks with empty depends_on will run concurrently -- do NOT assume sequential execution
- Do not invent project metadata (module paths, package names, frameworks) -- use only what is provided in the project context
- read_only_paths lists paths this task needs to read but must not modify (e.g., shared types defined by another task)
- description must be more than 20 characters
- depends_on is an array of task titles that must complete first (empty array if no deps)
- Output ONLY the JSON array, no other text
`)

	return sb.String()
}

// buildContractsPrompt creates the Pass 2 prompt for generating shared contracts
// from already-generated tasks.
func buildContractsPrompt(tasks []ProposedTask, projectCtx map[string]any) string {
	var sb strings.Builder

	sb.WriteString("ROLE: You are a software architect defining shared interfaces and types for concurrent AI coding agents.\n\n")

	if len(projectCtx) > 0 {
		sb.WriteString("PROJECT CONTEXT:\n")
		if v, ok := projectCtx["tech_stack"]; ok {
			sb.WriteString(fmt.Sprintf("Tech Stack: %v\n", v))
		}
		if v, ok := projectCtx["module_path"]; ok {
			sb.WriteString(fmt.Sprintf("Module/Package Path: %v\n", v))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("TASKS:\n")
	for _, t := range tasks {
		sb.WriteString(fmt.Sprintf("- Title: %s\n", t.Title))
		sb.WriteString(fmt.Sprintf("  Module: %s\n", t.ModulePath))
		if len(t.ProvidedContracts) > 0 {
			sb.WriteString("  Provided Contracts:\n")
			for _, c := range t.ProvidedContracts {
				sb.WriteString(fmt.Sprintf("    - %s: %s (at %s)\n", c.Name, c.Description, c.FilePath))
			}
		}
		if len(t.ConsumedContracts) > 0 {
			sb.WriteString(fmt.Sprintf("  Consumed Contracts: %v\n", t.ConsumedContracts))
		}
	}

	sb.WriteString(`
OUTPUT FORMAT:
Return ONLY a valid JSON array of shared contracts:
[
  {
    "name": "ContractName",
    "description": "Full interface/type definition including all method signatures and field types",
    "file_path": "path/to/file/where/it/should/live.go"
  }
]

RULES:
- Only include contracts that are referenced by multiple tasks
- Provide complete interface/type signatures in the description field
- file_path must be relative to project root
- Output ONLY the JSON array, no other text
`)

	return sb.String()
}

// parseTasksFromJSON extracts a JSON task array from the model's response text.
func parseTasksFromJSON(text string) ([]ProposedTask, error) {
	// Find the JSON array in the response
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || end < start {
		return nil, fmt.Errorf("no JSON array found in response: %q", text[:min(100, len(text))])
	}

	jsonStr := text[start : end+1]
	var tasks []ProposedTask
	if err := json.Unmarshal([]byte(jsonStr), &tasks); err != nil {
		return nil, fmt.Errorf("parse tasks JSON: %w", err)
	}

	return tasks, nil
}

// parseContractsFromJSON extracts a JSON contract array from the model's response text.
func parseContractsFromJSON(text string) ([]Contract, error) {
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || end < start {
		return nil, fmt.Errorf("no JSON array found in contracts response: %q", text[:min(100, len(text))])
	}

	jsonStr := text[start : end+1]
	var contracts []Contract
	if err := json.Unmarshal([]byte(jsonStr), &contracts); err != nil {
		return nil, fmt.Errorf("parse contracts JSON: %w", err)
	}

	return contracts, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
