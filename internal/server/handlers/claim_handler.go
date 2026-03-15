package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/moritzhuber/othrys/internal/coordinator"
	"github.com/moritzhuber/othrys/internal/models"
)

// ClaimHandler handles claim REST endpoints.
type ClaimHandler struct {
	coord *coordinator.Coordinator
}

// NewClaimHandler creates a new ClaimHandler.
func NewClaimHandler(coord *coordinator.Coordinator) *ClaimHandler {
	return &ClaimHandler{coord: coord}
}

// RequestClaim handles POST /api/v1/claims
func (h *ClaimHandler) RequestClaim(c *fiber.Ctx) error {
	agentID, _ := c.Locals("agent_id").(string)
	projectID, _ := c.Locals("project_id").(string)

	var body struct {
		AgentID   string `json:"agent_id"`
		TaskID    string `json:"task_id"`
		Path      string `json:"path"`
		ClaimType string `json:"claim_type"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.AgentID != "" {
		agentID = body.AgentID
	}
	if agentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "agent_id is required"})
	}
	if body.TaskID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "task_id is required"})
	}
	if body.Path == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "path is required"})
	}

	claimType := models.ClaimTypeExclusive
	if body.ClaimType == string(models.ClaimTypeSharedRead) {
		claimType = models.ClaimTypeSharedRead
	}

	result, err := h.coord.RequestClaim(c.Context(), projectID, agentID, body.TaskID, body.Path, claimType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if !result.Granted {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"granted":   false,
			"conflicts": result.Conflicts,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"granted": true,
		"claim":   result.Claim,
	})
}

// ReleaseClaim handles DELETE /api/v1/claims/:id
func (h *ClaimHandler) ReleaseClaim(c *fiber.Ctx) error {
	claimID := c.Params("id")
	agentID, _ := c.Locals("agent_id").(string)
	if agentID == "" {
		agentID = c.Get("X-Agent-Id")
	}

	if err := h.coord.ReleaseClaim(c.Context(), claimID, agentID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ListClaims handles GET /api/v1/projects/:id/claims
func (h *ClaimHandler) ListClaims(c *fiber.Ctx) error {
	projectID := c.Params("id")
	claims, err := h.coord.Claims.ListActiveByProject(c.Context(), projectID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"claims": claims})
}
