package server

import (
	"github.com/gofiber/fiber/v2"
	fiberws "github.com/gofiber/websocket/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/moritzhuber/othrys/internal/auth"
	"github.com/moritzhuber/othrys/internal/coordinator"
	"github.com/moritzhuber/othrys/internal/events"
	"github.com/moritzhuber/othrys/internal/planner"
	"github.com/moritzhuber/othrys/internal/server/handlers"
	"github.com/moritzhuber/othrys/internal/store"
	"github.com/moritzhuber/othrys/internal/ws"
)

// RegisterRoutes registers all REST and WebSocket routes on the Fiber app.
func RegisterRoutes(
	app *fiber.App,
	pool *pgxpool.Pool,
	bus events.EventBus,
	hub *ws.Hub,
	st *store.Store,
	coord *coordinator.Coordinator,
) {
	// Create a stub planner provider (will be overridden if configured)
	// In production, the provider is chosen based on config.LLMProvider
	var splitHandler *handlers.SplitHandler

	// Build handlers
	projectH := handlers.NewProjectHandler(coord)
	taskH := handlers.NewTaskHandler(coord)
	agentH := handlers.NewAgentHandler(coord)
	claimH := handlers.NewClaimHandler(coord)
	mergeH := handlers.NewMergeHandler(coord)

	bridge := ws.NewBridge(bus, hub)
	wsH := handlers.NewWSHandler(coord, hub, bridge)

	authMW := auth.RequireAPIKey(pool)
	agentMW := auth.RequireAgentID()

	// Public routes (no auth)
	app.Post("/api/v1/projects", projectH.CreateProject)
	app.Get("/api/v1/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	api := app.Group("/api/v1", authMW)
	api.Get("/projects/:id", projectH.GetProject)
	api.Put("/projects/:id/design", projectH.UpdateDesign)
	api.Get("/projects/:id/events", projectH.GetEvents)
	api.Get("/projects/:id/tasks", taskH.ListTasks)
	api.Post("/projects/:id/tasks", taskH.CreateTask)
	api.Get("/projects/:id/agents", agentH.ListAgents)
	api.Get("/projects/:id/claims", claimH.ListClaims)
	api.Post("/projects/:id/merge-check", mergeH.CheckMerge)
	api.Post("/projects/:id/tasks/approve-all", taskH.ApproveAll)
	api.Post("/projects/:id/tasks/reject-all", taskH.RejectAllProposed)

	// Project context and contracts endpoints
	api.Put("/projects/:id/context", projectH.UpdateContext)
	api.Put("/projects/:id/contracts", projectH.UpdateContracts)

	// Task routes
	api.Patch("/tasks/:id", taskH.UpdateTask)
	api.Post("/tasks/:id/assign", taskH.AssignTask)

	// Agent routes
	api.Post("/agents/register", agentH.RegisterAgent)
	api.Post("/agents/:id/heartbeat", agentH.Heartbeat)

	// Claim routes (require agent ID)
	api.Post("/claims", agentMW, claimH.RequestClaim)
	api.Delete("/claims/:id", agentMW, claimH.ReleaseClaim)

	// Split route (if splitter configured)
	if splitHandler != nil {
		api.Post("/projects/:id/split", splitHandler.SplitTasks)
	}

	// WebSocket endpoint
	// ws://host/ws?token=<api-key>&agent=<agent-id>
	app.Use("/ws", wsH.Upgrade)
	app.Get("/ws", fiberws.New(wsH.Handle))

	// We need the split handler — register it properly after building the provider
	_ = splitHandler
}

// RegisterSplitRoute adds the split and scaffold endpoints with a configured splitter.
// The scaffold endpoint is registered alongside split since it shares the same dependency
// on a configured planner.Provider and uses the same splitH instance.
func RegisterSplitRoute(app *fiber.App, pool *pgxpool.Pool, coord *coordinator.Coordinator, p planner.Provider, taskStore *store.TaskStore) {
	splitter := planner.NewSplitter(p, taskStore)
	splitH := handlers.NewSplitHandler(coord, splitter)
	authMW := auth.RequireAPIKey(pool)
	app.Post("/api/v1/projects/:id/split", authMW, splitH.SplitTasks)
	// Scaffold is only available when a provider is configured (same condition as split)
	app.Post("/api/v1/projects/:id/scaffold", authMW, splitH.Scaffold)
}
