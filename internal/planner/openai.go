package planner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultOpenAIModel = "gpt-4o"
const openAIAPIURL = "https://api.openai.com/v1/chat/completions"

// OpenAIProvider implements Provider using the OpenAI chat completions API.
// Uses net/http directly — no SDK dependency.
type OpenAIProvider struct {
	APIKey string
	Model  string
}

// NewOpenAIProvider creates a new OpenAIProvider.
func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
	if model == "" {
		model = defaultOpenAIModel
	}
	return &OpenAIProvider{APIKey: apiKey, Model: model}
}

// SplitDesign calls the OpenAI API to split a PM design into tasks.
func (p *OpenAIProvider) SplitDesign(design string, projectConfig map[string]any) ([]ProposedTask, error) {
	prompt := buildPrompt(design, projectConfig)

	reqBody, err := json.Marshal(map[string]any{
		"model": p.Model,
		"messages": []map[string]any{
			{"role": "system", "content": "You are a software project manager that decomposes PM designs into agent-assignable coding tasks."},
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]string{"type": "json_object"},
		"max_tokens":      4096,
		"temperature":     0.2,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, openAIAPIURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 4 * time.Minute}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse OpenAI response format
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in OpenAI response")
	}

	// With json_object mode, the content may be wrapped in {"tasks": [...]}
	text := apiResp.Choices[0].Message.Content

	// Try direct array parse first
	var tasks []ProposedTask
	if err := json.Unmarshal([]byte(text), &tasks); err == nil {
		return tasks, nil
	}

	// Try wrapped format
	var wrapped struct {
		Tasks []ProposedTask `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(text), &wrapped); err == nil && len(wrapped.Tasks) > 0 {
		return wrapped.Tasks, nil
	}

	return parseTasksFromJSON(text)
}
