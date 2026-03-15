package handlers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/gofiber/fiber/v2"
	fiberws "github.com/gofiber/websocket/v2"
	"github.com/moritzhuber/othrys/internal/coordinator"
	"github.com/moritzhuber/othrys/internal/models"
	"github.com/moritzhuber/othrys/internal/ws"
)

// WSHandler handles WebSocket connections.
type WSHandler struct {
	coord  *coordinator.Coordinator
	hub    *ws.Hub
	bridge *ws.Bridge
}

// NewWSHandler creates a new WSHandler.
func NewWSHandler(coord *coordinator.Coordinator, hub *ws.Hub, bridge *ws.Bridge) *WSHandler {
	return &WSHandler{coord: coord, hub: hub, bridge: bridge}
}

// Upgrade handles the WebSocket upgrade and connection lifecycle.
// Query params: token=<api-key>, agent=<agent-id>
func (h *WSHandler) Upgrade(c *fiber.Ctx) error {
	if !fiberws.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}

	projectID, _ := c.Locals("project_id").(string)
	agentID := c.Query("agent")

	// Store for the WebSocket handler
	c.Locals("ws_project_id", projectID)
	c.Locals("ws_agent_id", agentID)

	return c.Next()
}

// Handle is the actual WebSocket handler function.
func (h *WSHandler) Handle(c *fiberws.Conn) {
	projectID, _ := c.Locals("ws_project_id").(string)
	agentID, _ := c.Locals("ws_agent_id").(string)

	if projectID == "" {
		projectID = c.Query("project_id")
	}
	if agentID == "" {
		agentID = c.Query("agent")
	}

	conn := ws.NewConnection(c, projectID, agentID, h.hub)

	// Subscribe to project events on the bridge
	if h.bridge != nil {
		_ = h.bridge.SubscribeProject(projectID)
	}

	// Send initial claims snapshot for cache seeding
	go h.sendClaimsSnapshot(conn, projectID, agentID)

	// Run connection (blocks until closed)
	go h.handleInbound(conn)
	conn.Run()
}

// sendClaimsSnapshot sends all active claims to a newly connected agent.
func (h *WSHandler) sendClaimsSnapshot(conn *ws.Connection, projectID, agentID string) {
	ctx := context.Background()
	claims, err := h.coord.Claims.ListActiveByProject(ctx, projectID)
	if err != nil {
		log.Printf("[ws] get claims snapshot: %v", err)
		return
	}

	// Get agent names for the snapshot
	agents, _ := h.coord.Agents.ListByProject(ctx, projectID)
	agentNames := map[string]string{}
	for _, a := range agents {
		agentNames[a.ID] = a.Name
	}

	var items []ws.ClaimSnapshotItem
	for _, c := range claims {
		items = append(items, ws.ClaimSnapshotItem{
			ID:        c.ID,
			AgentID:   c.AgentID,
			AgentName: agentNames[c.AgentID],
			Path:      c.Path,
			ClaimType: string(c.ClaimType),
			Status:    string(c.Status),
		})
	}

	conn.Send(ws.OutboundMessage{
		Type:    ws.MsgTypeClaimsSnapshot,
		Payload: items,
	})
}

// handleInbound processes inbound messages from the client.
func (h *WSHandler) handleInbound(conn *ws.Connection) {
	for msg := range conn.InboundMessages {
		ctx := context.Background()
		switch msg.Type {
		case ws.MsgTypeHeartbeat, ws.MsgTypePong:
			// Update agent heartbeat
			if conn.AgentID != "" {
				_ = h.coord.Agents.UpdateHeartbeat(ctx, conn.AgentID)
			}

		case ws.MsgTypeClaimRequest:
			var payload ws.ClaimRequestPayload
			if err := json.Unmarshal(msg.Data, &payload); err != nil {
				conn.Send(ws.OutboundMessage{Type: ws.MsgTypeError, Payload: "invalid claim_request payload"})
				continue
			}
			h.handleClaimRequest(conn, ctx, payload)

		case ws.MsgTypeTaskUpdate:
			var payload ws.TaskUpdatePayload
			if err := json.Unmarshal(msg.Data, &payload); err != nil {
				conn.Send(ws.OutboundMessage{Type: ws.MsgTypeError, Payload: "invalid task_update payload"})
				continue
			}
			if err := h.coord.UpdateTaskStatus(ctx, payload.TaskID, conn.AgentID, models.TaskStatus(payload.Status)); err != nil {
				conn.Send(ws.OutboundMessage{Type: ws.MsgTypeError, Payload: err.Error()})
			}

		case ws.MsgTypeClaimsSync:
			// Return full claims snapshot for cache seeding
			h.sendClaimsSnapshot(conn, conn.ProjectID, conn.AgentID)

		default:
			log.Printf("[ws] unknown message type %q from agent %s", msg.Type, conn.AgentID)
		}
	}
}

// handleClaimRequest processes a WebSocket claim request.
func (h *WSHandler) handleClaimRequest(conn *ws.Connection, ctx context.Context, payload ws.ClaimRequestPayload) {
	claimType := models.ClaimTypeExclusive
	if payload.ClaimType == string(models.ClaimTypeSharedRead) {
		claimType = models.ClaimTypeSharedRead
	}

	result, err := h.coord.RequestClaim(ctx, conn.ProjectID, conn.AgentID, payload.TaskID, payload.Path, claimType)
	if err != nil {
		conn.Send(ws.OutboundMessage{Type: ws.MsgTypeError, Payload: err.Error()})
		return
	}

	if result.Granted {
		conn.Send(ws.OutboundMessage{
			Type: ws.MsgTypeClaimGranted,
			Payload: map[string]any{
				"claim_id": result.Claim.ID,
				"path":     payload.Path,
			},
		})
	} else {
		conn.Send(ws.OutboundMessage{
			Type: ws.MsgTypeClaimConflict,
			Payload: map[string]any{
				"path":      payload.Path,
				"conflicts": result.Conflicts,
			},
		})
	}
}
