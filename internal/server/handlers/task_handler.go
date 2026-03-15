package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/moritzhuber/othrys/internal/coordinator"
	"github.com/moritzhuber/othrys/internal/models"
)

// TaskHandler handles task REST endpoints.
type TaskHandler struct {
	coord *coordinator.Coordinator
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(coord *coordinator.Coordinator) *TaskHandler {
	return &TaskHandler{coord: coord}
}

// ListTasks handles GET /api/v1/projects/:id/tasks
func (h *TaskHandler) ListTasks(c *fiber.Ctx) error {
	projectID := c.Params("id")
	statusStr := c.Query("status")

	var statusFilter *models.TaskStatus
	if statusStr != "" {
		s := models.TaskStatus(statusStr)
		statusFilter = &s
	}

	tasks, err := h.coord.Tasks.ListByProject(c.Context(), projectID, statusFilter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"tasks": tasks})
}

// CreateTask handles POST /api/v1/projects/:id/tasks
func (h *TaskHandler) CreateTask(c *fiber.Ctx) error {
	projectID := c.Params("id")

	var body struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		ModulePath  string   `json:"module_path"`
		DependsOn   []string `json:"depends_on"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if body.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title is required"})
	}

	task := &models.Task{
		ProjectID:   projectID,
		Title:       body.Title,
		Description: body.Description,
		ModulePath:  body.ModulePath,
		Status:      models.TaskStatusProposed,
		DependsOn:   body.DependsOn,
	}

	created, err := h.coord.Tasks.Create(c.Context(), task)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(created)
}

// UpdateTask handles PATCH /api/v1/tasks/:id
func (h *TaskHandler) UpdateTask(c *fiber.Ctx) error {
	taskID := c.Params("id")

	var body struct {
		Status  *string `json:"status"`
		Approve *bool   `json:"approve"`
		Reject  *bool   `json:"reject"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Approve != nil && *body.Approve {
		task, err := h.coord.ApproveTask(c.Context(), taskID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(task)
	}

	if body.Reject != nil && *body.Reject {
		if err := h.coord.RejectTask(c.Context(), taskID); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusNoContent).Send(nil)
	}

	if body.Status != nil {
		agentID, _ := c.Locals("agent_id").(string)
		if err := h.coord.UpdateTaskStatus(c.Context(), taskID, agentID, models.TaskStatus(*body.Status)); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		task, _ := h.coord.Tasks.GetByID(c.Context(), taskID)
		return c.JSON(task)
	}

	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no update action specified"})
}

// ApproveAll handles POST /api/v1/projects/:id/tasks/approve-all
func (h *TaskHandler) ApproveAll(c *fiber.Ctx) error {
	projectID := c.Params("id")

	proposed := models.TaskStatusProposed
	tasks, err := h.coord.Tasks.ListByProject(c.Context(), projectID, &proposed)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	approved := make([]*models.Task, 0, len(tasks))
	for _, t := range tasks {
		task, err := h.coord.ApproveTask(c.Context(), t.ID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":           err.Error(),
				"failed_task_id":  t.ID,
				"approved_so_far": len(approved),
			})
		}
		approved = append(approved, task)
	}

	return c.JSON(fiber.Map{"approved": len(approved), "tasks": approved})
}

// RejectAllProposed handles POST /api/v1/projects/:id/tasks/reject-all
func (h *TaskHandler) RejectAllProposed(c *fiber.Ctx) error {
	projectID := c.Params("id")

	proposed := models.TaskStatusProposed
	tasks, err := h.coord.Tasks.ListByProject(c.Context(), projectID, &proposed)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	rejected := 0
	for _, t := range tasks {
		if err := h.coord.RejectTask(c.Context(), t.ID); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":            err.Error(),
				"failed_task_id":   t.ID,
				"rejected_so_far":  rejected,
			})
		}
		rejected++
	}

	return c.JSON(fiber.Map{"rejected": rejected})
}

// AssignTask handles POST /api/v1/tasks/:id/assign
func (h *TaskHandler) AssignTask(c *fiber.Ctx) error {
	taskID := c.Params("id")

	var body struct {
		AgentID string `json:"agent_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if body.AgentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "agent_id is required"})
	}

	task, err := h.coord.AssignTask(c.Context(), taskID, body.AgentID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(task)
}
