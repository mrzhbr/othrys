package coordinator

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/moritzhuber/othrys/internal/models"
)

// StartCleanup starts a background goroutine that:
//   - Checks agent heartbeats every 60s
//   - Marks agents as disconnected if heartbeat > 5min old
//   - Revokes their active claims
//   - Publishes disconnect events through the EventBus
//
// The goroutine runs until ctx is cancelled.
func (c *Coordinator) StartCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.runCleanup(ctx)
			}
		}
	}()
}

func (c *Coordinator) runCleanup(ctx context.Context) {
	// Get all projects
	projects, err := c.Projects.ListAll(ctx)
	if err != nil {
		log.Printf("[cleanup] list projects error: %v", err)
		return
	}

	staleThreshold := time.Now().UTC().Add(-5 * time.Minute)

	for _, project := range projects {
		agents, err := c.Agents.ListStale(ctx, project.ID, staleThreshold)
		if err != nil {
			log.Printf("[cleanup] list stale agents for %s: %v", project.ID, err)
			continue
		}

		for _, agent := range agents {
			log.Printf("[cleanup] agent %s (%s) is stale — disconnecting", agent.Name, agent.ID)

			// Mark disconnected
			if err := c.Agents.UpdateStatus(ctx, agent.ID, models.AgentStatusDisconnected); err != nil {
				log.Printf("[cleanup] update agent status: %v", err)
			}

			// Revoke claims
			revokedIDs, err := c.Claims.RevokeByAgent(ctx, agent.ID)
			if err != nil {
				log.Printf("[cleanup] revoke claims for agent %s: %v", agent.ID, err)
			}

			// Publish disconnect event
			payload, _ := json.Marshal(map[string]any{
				"agent_id":       agent.ID,
				"agent_name":     agent.Name,
				"revoked_claims": revokedIDs,
			})
			_, _ = c.Events.Create(ctx, &models.Event{
				ProjectID: project.ID,
				EventType: models.EventTypeAgentDisconnect,
				AgentID:   &agent.ID,
				Payload:   payload,
			}, c.Bus)

			log.Printf("[cleanup] agent %s disconnected, revoked %d claims", agent.Name, len(revokedIDs))
		}
	}
}
