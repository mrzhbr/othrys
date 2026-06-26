package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/moritzhuber/othrys/internal/store"
)

// BoardHandler handles the Kanban board REST endpoint.
type BoardHandler struct {
	boardStore *store.BoardStore
}

// NewBoardHandler creates a new BoardHandler.
func NewBoardHandler(boardStore *store.BoardStore) *BoardHandler {
	return &BoardHandler{boardStore: boardStore}
}

// GetBoard handles GET /api/v1/projects/:id/board
// Returns the full board state: tasks grouped by column, agents, and active claims.
func (h *BoardHandler) GetBoard(c *fiber.Ctx) error {
	projectID := c.Params("id")

	state, err := h.boardStore.GetBoardState(c.Context(), projectID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(state)
}
