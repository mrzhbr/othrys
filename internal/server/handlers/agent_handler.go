package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/moritzhuber/othrys/internal/coordinator"
	"github.com/moritzhuber/othrys/internal/models"
)

// AgentHandler handles agent REST endpoints.
type AgentHandler struct {
	coord *coordinator.Coordinator
}

// NewAgentHandler creates a new AgentHandler.
func NewAgentHandler(coord *coordinator.Coordinator) *AgentHandler {
	return &AgentHandler{coord: coord}
}

// RegisterAgent handles POST /api/v1/agents/register
func (h *AgentHandler) RegisterAgent(c *fiber.Ctx) error {
	var body struct {
		ProjectID string `json:"project_id"`
		Name      string `json:"name"`
		ToolType  string `json:"tool_type"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	projectID, _ := c.Locals("project_id").(string)
	if body.ProjectID != "" {
		projectID = body.ProjectID
	}

	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	toolType := models.AgentToolType(body.ToolType)
	if toolType == "" {
		toolType = models.AgentToolGeneric
	}

	agent, err := h.coord.Agents.Register(c.Context(), projectID, body.Name, toolType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(agent)
}

// ListAgents handles GET /api/v1/projects/:id/agents
func (h *AgentHandler) ListAgents(c *fiber.Ctx) error {
	projectID := c.Params("id")
	agents, err := h.coord.Agents.ListByProject(c.Context(), projectID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"agents": agents})
}

// Heartbeat handles POST /api/v1/agents/:id/heartbeat
func (h *AgentHandler) Heartbeat(c *fiber.Ctx) error {
	agentID := c.Params("id")
	if err := h.coord.Agents.UpdateHeartbeat(c.Context(), agentID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusNoContent).Send(nil)
}
