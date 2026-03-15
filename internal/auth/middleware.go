package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RequireAPIKey is a Fiber middleware that validates the Authorization: Bearer <api-key> header.
// On success, sets locals: "project_id" (string).
func RequireAPIKey(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing Authorization header",
			})
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authorization header must be: Bearer <api-key>",
			})
		}

		apiKey := strings.TrimSpace(parts[1])
		projectID, err := ValidateAPIKey(c.Context(), pool, apiKey)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to validate API key",
			})
		}

		if projectID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid API key",
			})
		}

		c.Locals("project_id", projectID)
		c.Locals("api_key", apiKey)
		return c.Next()
	}
}

// RequireAgentID is a Fiber middleware that validates the X-Agent-Id header.
// Must be used after RequireAPIKey. Sets local: "agent_id" (string).
func RequireAgentID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		agentID := strings.TrimSpace(c.Get("X-Agent-Id"))
		if agentID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing X-Agent-Id header",
			})
		}
		c.Locals("agent_id", agentID)
		return c.Next()
	}
}
