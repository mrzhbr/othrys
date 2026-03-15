package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client is an HTTP client for the Othrys server API.
type Client struct {
	ServerURL string
	APIKey    string
	AgentID   string
	http      *http.Client
}

// NewClient creates a new Client.
func NewClient(serverURL, apiKey, agentID string) *Client {
	return &Client{
		ServerURL: serverURL,
		APIKey:    apiKey,
		AgentID:   agentID,
		http:      http.DefaultClient,
	}
}

// APIError represents an HTTP error response from the server.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

// do executes an HTTP request with auth headers and parses the JSON response.
func (c *Client) do(method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.ServerURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	if c.AgentID != "" {
		req.Header.Set("X-Agent-Id", c.AgentID)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(respBody, &errResp)
		msg := errResp.Error
		if msg == "" {
			msg = string(respBody)
		}
		return &APIError{StatusCode: resp.StatusCode, Message: msg}
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("parse response: %w", err)
		}
	}

	return nil
}

// CreateProject creates a new project.
func (c *Client) CreateProject(name, repoURL string) (map[string]any, error) {
	var result map[string]any
	err := c.do("POST", "/api/v1/projects", map[string]string{"name": name, "repo_url": repoURL}, &result)
	return result, err
}

// GetProject retrieves project details.
func (c *Client) GetProject(id string) (map[string]any, error) {
	var result map[string]any
	err := c.do("GET", "/api/v1/projects/"+id, nil, &result)
	return result, err
}

// UpdateDesign submits the PM design document.
func (c *Client) UpdateDesign(projectID string, design any) (map[string]any, error) {
	var result map[string]any
	err := c.do("PUT", "/api/v1/projects/"+projectID+"/design", design, &result)
	return result, err
}

// SplitTasks triggers LLM task splitting.
func (c *Client) SplitTasks(projectID string) (map[string]any, error) {
	var result map[string]any
	err := c.do("POST", "/api/v1/projects/"+projectID+"/split", nil, &result)
	return result, err
}

// ListTasks returns tasks for a project.
func (c *Client) ListTasks(projectID, status string) (map[string]any, error) {
	path := "/api/v1/projects/" + projectID + "/tasks"
	if status != "" {
		path += "?status=" + status
	}
	var result map[string]any
	err := c.do("GET", path, nil, &result)
	return result, err
}

// CreateTask creates a new task.
func (c *Client) CreateTask(projectID, title, description, modulePath string) (map[string]any, error) {
	var result map[string]any
	err := c.do("POST", "/api/v1/projects/"+projectID+"/tasks", map[string]string{
		"title":       title,
		"description": description,
		"module_path": modulePath,
	}, &result)
	return result, err
}

// ApproveTask approves a proposed task.
func (c *Client) ApproveTask(taskID string) (map[string]any, error) {
	var result map[string]any
	err := c.do("PATCH", "/api/v1/tasks/"+taskID, map[string]bool{"approve": true}, &result)
	return result, err
}

// AssignTask assigns a task to an agent.
func (c *Client) AssignTask(taskID, agentID string) (map[string]any, error) {
	var result map[string]any
	err := c.do("POST", "/api/v1/tasks/"+taskID+"/assign", map[string]string{"agent_id": agentID}, &result)
	return result, err
}

// UpdateTaskStatus updates task status.
func (c *Client) UpdateTaskStatus(taskID, status string) (map[string]any, error) {
	var result map[string]any
	err := c.do("PATCH", "/api/v1/tasks/"+taskID, map[string]string{"status": status}, &result)
	return result, err
}

// RegisterAgent registers an agent.
func (c *Client) RegisterAgent(name, toolType string) (map[string]any, error) {
	var result map[string]any
	err := c.do("POST", "/api/v1/agents/register", map[string]string{
		"name":      name,
		"tool_type": toolType,
	}, &result)
	return result, err
}

// Heartbeat sends an agent heartbeat.
func (c *Client) Heartbeat(agentID string) error {
	return c.do("POST", "/api/v1/agents/"+agentID+"/heartbeat", nil, nil)
}

// RequestClaim requests a claim on a path.
func (c *Client) RequestClaim(agentID, taskID, path, claimType string) (map[string]any, error) {
	var result map[string]any
	err := c.do("POST", "/api/v1/claims", map[string]string{
		"agent_id":   agentID,
		"task_id":    taskID,
		"path":       path,
		"claim_type": claimType,
	}, &result)
	return result, err
}

// ReleaseClaim releases a claim.
func (c *Client) ReleaseClaim(claimID string) error {
	return c.do("DELETE", "/api/v1/claims/"+claimID, nil, nil)
}

// ListClaims returns active claims for a project.
func (c *Client) ListClaims(projectID string) (map[string]any, error) {
	var result map[string]any
	err := c.do("GET", "/api/v1/projects/"+projectID+"/claims", nil, &result)
	return result, err
}

// CheckMerge checks merge readiness.
func (c *Client) CheckMerge(projectID string) (map[string]any, error) {
	var result map[string]any
	err := c.do("POST", "/api/v1/projects/"+projectID+"/merge-check", nil, &result)
	return result, err
}

// ListAgents returns agents for a project.
func (c *Client) ListAgents(projectID string) (map[string]any, error) {
	var result map[string]any
	err := c.do("GET", "/api/v1/projects/"+projectID+"/agents", nil, &result)
	return result, err
}
