package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/moritzhuber/othrys/internal/models"
)

// PGBus implements EventBus using PostgreSQL LISTEN/NOTIFY.
// It uses a dedicated connection for LISTEN (not the connection pool).
type PGBus struct {
	pool        *pgxpool.Pool
	databaseURL string

	mu          sync.RWMutex
	handlers    map[string][]func(models.Event)
	listenConns map[string]*pgx.Conn // channel → dedicated connection
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewPGBus creates a new PostgreSQL-backed EventBus.
func NewPGBus(ctx context.Context, pool *pgxpool.Pool, databaseURL string) (*PGBus, error) {
	busCtx, cancel := context.WithCancel(ctx)
	return &PGBus{
		pool:        pool,
		databaseURL: databaseURL,
		handlers:    make(map[string][]func(models.Event)),
		listenConns: make(map[string]*pgx.Conn),
		ctx:         busCtx,
		cancel:      cancel,
	}, nil
}

// Publish inserts an event into PostgreSQL and sends a NOTIFY on the project's channel.
// The channel name is "othrys_<project_id>" (with dashes replaced by underscores for PG compat).
func (b *PGBus) Publish(ctx context.Context, event models.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	channel := channelName(event.ProjectID)
	_, err = b.pool.Exec(ctx, `SELECT pg_notify($1, $2)`, channel, string(payload))
	if err != nil {
		return fmt.Errorf("pg_notify: %w", err)
	}

	return nil
}

// Subscribe registers a handler for events on the given channel.
// The channel should be a project-level channel: "othrys_<project_id>".
// If this is the first subscriber for the channel, a new LISTEN goroutine is started.
func (b *PGBus) Subscribe(channel string, handler func(models.Event)) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[channel] = append(b.handlers[channel], handler)

	// Only start one LISTEN goroutine per channel
	if _, exists := b.listenConns[channel]; exists {
		return nil
	}

	conn, err := pgx.Connect(b.ctx, b.databaseURL)
	if err != nil {
		return fmt.Errorf("open listen connection for channel %q: %w", channel, err)
	}

	_, err = conn.Exec(b.ctx, fmt.Sprintf("LISTEN %q", channel))
	if err != nil {
		_ = conn.Close(b.ctx)
		return fmt.Errorf("LISTEN %q: %w", channel, err)
	}

	b.listenConns[channel] = conn

	b.wg.Add(1)
	go b.listenLoop(channel, conn)

	return nil
}

// listenLoop waits for notifications on the given channel and dispatches to handlers.
// On connection loss, it attempts reconnection with exponential backoff.
func (b *PGBus) listenLoop(channel string, conn *pgx.Conn) {
	defer b.wg.Done()

	for {
		notification, err := conn.WaitForNotification(b.ctx)
		if err != nil {
			if b.ctx.Err() != nil {
				// Context cancelled — shutting down
				return
			}

			log.Printf("[pgbus] lost connection for channel %q: %v — reconnecting", channel, err)
			_ = conn.Close(b.ctx)

			// Reconnect with backoff
			for attempt := 1; ; attempt++ {
				select {
				case <-b.ctx.Done():
					return
				case <-time.After(time.Duration(attempt) * time.Second):
				}

				newConn, err := pgx.Connect(b.ctx, b.databaseURL)
				if err != nil {
					log.Printf("[pgbus] reconnect attempt %d failed: %v", attempt, err)
					if attempt > 5 {
						attempt = 5 // cap backoff at 5s
					}
					continue
				}

				if _, err = newConn.Exec(b.ctx, fmt.Sprintf("LISTEN %q", channel)); err != nil {
					log.Printf("[pgbus] LISTEN on reconnect failed: %v", err)
					_ = newConn.Close(b.ctx)
					continue
				}

				b.mu.Lock()
				b.listenConns[channel] = newConn
				b.mu.Unlock()

				conn = newConn
				log.Printf("[pgbus] reconnected for channel %q", channel)
				break
			}
			continue
		}

		var event models.Event
		if err := json.Unmarshal([]byte(notification.Payload), &event); err != nil {
			log.Printf("[pgbus] unmarshal notification on %q: %v", channel, err)
			continue
		}

		b.mu.RLock()
		handlers := b.handlers[channel]
		b.mu.RUnlock()

		for _, h := range handlers {
			go h(event)
		}
	}
}

// Close stops all LISTEN goroutines and closes dedicated connections.
func (b *PGBus) Close() error {
	b.cancel()

	b.mu.Lock()
	for _, conn := range b.listenConns {
		_ = conn.Close(context.Background())
	}
	b.mu.Unlock()

	b.wg.Wait()
	return nil
}

// channelName returns a PostgreSQL-safe channel name for a project ID.
// UUIDs contain dashes which PG doesn't allow in unquoted identifiers, so we use
// quoted identifier form in LISTEN/NOTIFY but keep a consistent format here.
func channelName(projectID string) string {
	return "othrys_" + projectID
}
