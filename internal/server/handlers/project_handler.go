package handlers

import (
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"github.com/moritzhuber/othrys/internal/coordinator"
	"github.com/moritzhuber/othrys/internal/models"
)

// ProjectHandler handles project REST endpoints.
type ProjectHandler struct {
	coord *coordinator.Coordinator
}

// NewProjectHandler creates a new ProjectHandler.
func NewProjectHandler(coord *coordinator.Coordinator) *ProjectHandler {
	return &ProjectHandler{coord: coord}
}

// CreateProject handles POST /api/v1/projects
// Auth is NOT required for project creation (returns the API key).
func (h *ProjectHandler) CreateProject(c *fiber.Ctx) error {
	var body struct {
		Name    string `json:"name"`
		RepoURL string `json:"repo_url"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	project, err := h.coord.Projects.Create(c.Context(), body.Name, body.RepoURL)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(project)
}

// GetProject handles GET /api/v1/projects/:id
func (h *ProjectHandler) GetProject(c *fiber.Ctx) error {
	id := c.Params("id")
	project, err := h.coord.Projects.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if project == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	// Get task and agent counts
	tasks, _ := h.coord.Tasks.ListByProject(c.Context(), id, nil)
	agents, _ := h.coord.Agents.ListByProject(c.Context(), id)
	claims, _ := h.coord.Claims.ListActiveByProject(c.Context(), id)

	taskCounts := map[string]int{}
	for _, t := range tasks {
		taskCounts[string(t.Status)]++
	}

	return c.JSON(fiber.Map{
		"project":      project,
		"task_counts":  taskCounts,
		"agent_count":  len(agents),
		"claim_count":  len(claims),
	})
}

// UpdateDesign handles PUT /api/v1/projects/:id/design
func (h *ProjectHandler) UpdateDesign(c *fiber.Ctx) error {
	id := c.Params("id")

	// Accept any JSON body as the design document
	body := c.Body()
	if len(body) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "request body is required"})
	}

	if err := h.coord.Projects.UpdateDesign(c.Context(), id, body); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	project, err := h.coord.Projects.GetByID(c.Context(), id)
	if err != nil || project == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	return c.JSON(project)
}

// GetEvents handles GET /api/v1/projects/:id/events
func (h *ProjectHandler) GetEvents(c *fiber.Ctx) error {
	id := c.Params("id")
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	_ = models.EventTypeTaskCreated // ensure models import is used
	evts, err := h.coord.Events.ListByProject(c.Context(), id, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"events": evts, "limit": limit, "offset": offset})
}

// UpdateContext handles PUT /api/v1/projects/:id/context
// Stores project context (tech stack, module path, conventions, etc.) as JSONB.
// This context is passed to the LLM task splitter to ground generated tasks in reality.
func (h *ProjectHandler) UpdateContext(c *fiber.Ctx) error {
	id := c.Params("id")

	body := c.Body()
	if len(body) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "request body is required"})
	}

	// Validate body is valid JSON with at least one field
	var parsed map[string]any
	if err := c.BodyParser(&parsed); err != nil || len(parsed) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "request body must be a non-empty JSON object"})
	}

	if err := h.coord.Projects.UpdateProjectContext(c.Context(), id, body); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	project, err := h.coord.Projects.GetByID(c.Context(), id)
	if err != nil || project == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	return c.JSON(fiber.Map{"project_context": project.ProjectContext})
}

// UpdateContracts handles PUT /api/v1/projects/:id/contracts
// Allows the PM to review and set shared contracts (interface/type definitions)
// before agents start. These contracts are injected into agent briefings.
func (h *ProjectHandler) UpdateContracts(c *fiber.Ctx) error {
	id := c.Params("id")

	var body struct {
		Contracts []map[string]any `json:"contracts"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if len(body.Contracts) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "contracts array must be non-empty"})
	}

	contractsJSON, err := json.Marshal(body.Contracts)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "failed to serialize contracts"})
	}

	if err := h.coord.Projects.UpdateSharedContracts(c.Context(), id, contractsJSON); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"contracts": body.Contracts})
}
