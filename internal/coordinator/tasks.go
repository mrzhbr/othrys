package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/moritzhuber/othrys/internal/models"
)

var nonAlphanumRE = regexp.MustCompile(`[^a-z0-9]+`)

// slugify converts a string to a URL-safe slug.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlphanumRE.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 50 {
		s = s[:50]
	}
	return s
}

// AssignTask validates the task and agent, generates a branch name, assigns the task,
// assembles an agent briefing (including project context, sibling tasks, and contracts),
// and publishes an assignment event through the EventBus.
func (c *Coordinator) AssignTask(ctx context.Context, taskID, agentID string) (*models.Task, error) {
	task, err := c.Tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	if task.Status != models.TaskStatusApproved {
		return nil, fmt.Errorf("task %s is not approved (status: %s)", taskID, task.Status)
	}

	agent, err := c.Agents.GetByID(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}
	if agent == nil {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	branchName := fmt.Sprintf("othrys/%s/%s", slugify(agent.Name), slugify(task.Title))

	if err := c.Tasks.Assign(ctx, taskID, agentID, branchName); err != nil {
		return nil, fmt.Errorf("assign task: %w", err)
	}

	// Reload task after assignment update
	task, err = c.Tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Assemble agent briefing:
	// 1. Load sibling tasks (assigned or in_progress) to build the concurrency notice.
	//    Filter out the current task (it may already appear with status "assigned").
	siblingTasks, err := c.Tasks.ListByProjectMultiStatus(ctx, task.ProjectID, []models.TaskStatus{
		models.TaskStatusAssigned,
		models.TaskStatusInProgress,
	})
	if err != nil {
		// Non-fatal: proceed without sibling context
		siblingTasks = nil
	}
	// Filter out the task being assigned so it doesn't appear in its own concurrency notice
	filtered := siblingTasks[:0]
	for _, s := range siblingTasks {
		if s.ID != taskID {
			filtered = append(filtered, s)
		}
	}
	siblingTasks = filtered

	// 2. Load project for context and shared contracts
	project, err := c.Projects.GetByID(ctx, task.ProjectID)
	if err != nil {
		project = nil // Non-fatal
	}

	// 3. Assemble and store the briefing
	briefing := AssembleBriefing(task, project, siblingTasks)
	if err := c.Tasks.UpdateAgentBriefing(ctx, taskID, briefing); err != nil {
		// Non-fatal: log but don't fail the assignment
		_ = err
	}

	// 4. Reload task one final time so the returned task includes the briefing
	task, err = c.Tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Publish assignment event
	payload, _ := json.Marshal(map[string]any{
		"task_id":    taskID,
		"agent_id":   agentID,
		"agent_name": agent.Name,
		"branch":     branchName,
	})
	_, _ = c.Events.Create(ctx, &models.Event{
		ProjectID: task.ProjectID,
		EventType: models.EventTypeTaskAssigned,
		AgentID:   &agentID,
		Payload:   payload,
	}, c.Bus)

	return task, nil
}

// UpdateTaskStatus updates a task's status and publishes a status change event.
func (c *Coordinator) UpdateTaskStatus(ctx context.Context, taskID, agentID string, status models.TaskStatus) error {
	task, err := c.Tasks.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if err := c.Tasks.UpdateStatus(ctx, taskID, status); err != nil {
		return err
	}

	payload, _ := json.Marshal(map[string]any{
		"task_id":    taskID,
		"new_status": status,
	})
	var agentIDPtr *string
	if agentID != "" {
		agentIDPtr = &agentID
	}
	_, _ = c.Events.Create(ctx, &models.Event{
		ProjectID: task.ProjectID,
		EventType: models.EventTypeTaskStatusChange,
		AgentID:   agentIDPtr,
		Payload:   payload,
	}, c.Bus)

	return nil
}
