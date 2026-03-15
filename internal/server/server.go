package server

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/moritzhuber/othrys/internal/coordinator"
	"github.com/moritzhuber/othrys/internal/events"
	"github.com/moritzhuber/othrys/internal/store"
	"github.com/moritzhuber/othrys/internal/ws"
)

// App holds the Fiber application and its dependencies.
type App struct {
	fiber *fiber.App
	pool  *pgxpool.Pool
}

// New creates a new Fiber app with all middleware and routes registered.
func New(
	pool *pgxpool.Pool,
	bus events.EventBus,
	hub *ws.Hub,
	st *store.Store,
	coord *coordinator.Coordinator,
) *App {
	app := fiber.New(fiber.Config{
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 5 * time.Minute,
		IdleTimeout:  2 * time.Minute,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(recover.New())
	app.Use(logger.New())

	RegisterRoutes(app, pool, bus, hub, st, coord)

	return &App{
		fiber: app,
		pool:  pool,
	}
}

// Listen starts the HTTP server on the given port.
func (a *App) Listen(port int) error {
	return a.fiber.Listen(fmt.Sprintf(":%d", port))
}

// Shutdown gracefully shuts down the server.
func (a *App) Shutdown() error {
	return a.fiber.Shutdown()
}

// Fiber returns the underlying Fiber app (for testing).
func (a *App) Fiber() *fiber.App {
	return a.fiber
}
