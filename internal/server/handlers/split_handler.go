package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/moritzhuber/othrys/internal/coordinator"
	"github.com/moritzhuber/othrys/internal/planner"
)

// SplitHandler handles the LLM task splitting endpoint.
type SplitHandler struct {
	coord    *coordinator.Coordinator
	splitter *planner.Splitter
}

// NewSplitHandler creates a new SplitHandler.
func NewSplitHandler(coord *coordinator.Coordinator, splitter *planner.Splitter) *SplitHandler {
	return &SplitHandler{coord: coord, splitter: splitter}
}

// SplitTasks handles POST /api/v1/projects/:id/split
// Checks PM design exists, calls the splitter, returns proposed tasks.
func (h *SplitHandler) SplitTasks(c *fiber.Ctx) error {
	projectID := c.Params("id")

	project, err := h.coord.Projects.GetByID(c.Context(), projectID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if project == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}
	if project.PMDesign == nil || string(project.PMDesign) == "null" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project has no PM design document — use PUT /api/v1/projects/:id/design first",
		})
	}

	tasks, err := h.splitter.Split(c.Context(), project)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"tasks": tasks,
		"count": len(tasks),
		"note":  "Tasks are in 'proposed' status. Approve with PATCH /api/v1/tasks/:id {\"approve\": true}",
	})
}

// Scaffold handles POST /api/v1/projects/:id/scaffold
// Triggers LLM Pass 2 (contract generation) from previously split tasks.
// Stores the generated contracts in projects.shared_contracts and returns them for PM review.
// This endpoint is only registered when the provider supports ContractGenerator (via RegisterSplitRoute).
func (h *SplitHandler) Scaffold(c *fiber.Ctx) error {
	projectID := c.Params("id")

	project, err := h.coord.Projects.GetByID(c.Context(), projectID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if project == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	contracts, err := h.splitter.GenerateContracts(c.Context(), project)
	if err != nil {
		if err.Error() == "provider does not support contract generation" {
			return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
				"error": "the configured LLM provider does not support contract generation",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Marshal contracts and store in project.shared_contracts via coordinator
	contractsJSON, err := json.Marshal(contracts)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to serialize contracts"})
	}

	if err := h.coord.Projects.UpdateSharedContracts(c.Context(), projectID, contractsJSON); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"contracts": contracts,
		"count":     len(contracts),
		"note":      "Review contracts above. Set final version with PUT /api/v1/projects/:id/contracts",
	})
}
