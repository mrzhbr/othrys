package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/moritzhuber/othrys/internal/config"
	"github.com/moritzhuber/othrys/internal/coordinator"
	"github.com/moritzhuber/othrys/internal/events"
	"github.com/moritzhuber/othrys/internal/planner"
	"github.com/moritzhuber/othrys/internal/server"
	"github.com/moritzhuber/othrys/internal/store"
	"github.com/moritzhuber/othrys/internal/ws"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Connect to database
	pool, err := store.NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to database.")

	// Run migrations
	migrationsDir := filepath.Join("migrations")
	if _, err := os.Stat(migrationsDir); err == nil {
		if err := store.RunMigrations(ctx, pool, migrationsDir); err != nil {
			log.Fatalf("migrations: %v", err)
		}
		log.Println("Migrations applied.")
	}

	// Create store
	st := store.New(pool)

	// Create EventBus (PG LISTEN/NOTIFY)
	bus, err := events.NewPGBus(ctx, pool, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("event bus: %v", err)
	}
	defer bus.Close()
	log.Println("EventBus ready.")

	// Create WebSocket hub
	hub := ws.NewHub()

	// Create stores
	projectStore := store.NewProjectStore(st)
	taskStore := store.NewTaskStore(st)
	agentStore := store.NewAgentStore(st)
	claimStore := store.NewClaimStore(st)
	eventStore := store.NewEventStore(st)

	// Create coordinator
	coord := coordinator.New(projectStore, taskStore, agentStore, claimStore, eventStore, bus)

	// Start cleanup goroutine
	coord.StartCleanup(ctx)
	log.Println("Cleanup goroutine started.")

	// Create Fiber app
	app := server.New(pool, bus, hub, st, coord)

	// Register split endpoint if LLM is configured
	if cfg.LLMAPIKey != "" {
		var llmProvider planner.Provider
		switch cfg.LLMProvider {
		case "openai":
			llmProvider = planner.NewOpenAIProvider(cfg.LLMAPIKey, cfg.LLMModel)
		default:
			llmProvider = planner.NewAnthropicProvider(cfg.LLMAPIKey, cfg.LLMModel)
		}
		server.RegisterSplitRoute(app.Fiber(), pool, coord, llmProvider, taskStore)
		log.Printf("LLM provider: %s (%s)", cfg.LLMProvider, cfg.LLMModel)
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down...")
		cancel()
		if err := app.Shutdown(); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Server starting on %s", addr)
	if err := app.Listen(cfg.Port); err != nil {
		log.Printf("server stopped: %v", err)
	}
}
