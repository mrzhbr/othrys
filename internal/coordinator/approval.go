package coordinator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/moritzhuber/othrys/internal/models"
)

// ApproveTask transitions a single proposed task to approved.
func (c *Coordinator) ApproveTask(ctx context.Context, taskID string) (*models.Task, error) {
	task, err := c.Tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	if task.Status != models.TaskStatusProposed {
		return nil, fmt.Errorf("task %s is not in proposed status (current: %s)", taskID, task.Status)
	}

	if err := c.Tasks.UpdateStatus(ctx, taskID, models.TaskStatusApproved); err != nil {
		return nil, fmt.Errorf("approve task: %w", err)
	}

	payload, _ := json.Marshal(map[string]any{"task_id": taskID})
	_, _ = c.Events.Create(ctx, &models.Event{
		ProjectID: task.ProjectID,
		EventType: models.EventTypeTaskApproved,
		Payload:   payload,
	}, c.Bus)

	task.Status = models.TaskStatusApproved
	return task, nil
}

// ApproveBatch approves multiple tasks transactionally.
func (c *Coordinator) ApproveBatch(ctx context.Context, taskIDs []string) ([]*models.Task, error) {
	var approved []*models.Task
	for _, id := range taskIDs {
		task, err := c.ApproveTask(ctx, id)
		if err != nil {
			return approved, fmt.Errorf("approve task %s: %w", id, err)
		}
		approved = append(approved, task)
	}
	return approved, nil
}

// RejectTask deletes a proposed task.
func (c *Coordinator) RejectTask(ctx context.Context, taskID string) error {
	task, err := c.Tasks.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}
	if task.Status != models.TaskStatusProposed {
		return fmt.Errorf("only proposed tasks can be rejected")
	}
	return c.Tasks.Delete(ctx, taskID)
}
