package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/moritzhuber/othrys/internal/coordinator"
)

// MergeHandler handles merge coordination REST endpoints.
type MergeHandler struct {
	coord *coordinator.Coordinator
}

// NewMergeHandler creates a new MergeHandler.
func NewMergeHandler(coord *coordinator.Coordinator) *MergeHandler {
	return &MergeHandler{coord: coord}
}

// CheckMerge handles POST /api/v1/projects/:id/merge-check
// Returns server-side readiness: task/claim status and branch list.
// The server does NOT perform git conflict preview — that is the CLI's responsibility.
func (h *MergeHandler) CheckMerge(c *fiber.Ctx) error {
	projectID := c.Params("id")

	report, err := h.coord.CheckMergeReadiness(c.Context(), projectID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(report)
}
