package ws

import (
	"encoding/json"
	"log"
	"sync"
)

// Hub manages all active WebSocket connections, organized by project.
type Hub struct {
	mu          sync.RWMutex
	// projectConns maps project_id → set of connections
	projectConns map[string]map[*Connection]struct{}
	// agentConns maps agent_id → connection (one connection per agent)
	agentConns   map[string]*Connection
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		projectConns: make(map[string]map[*Connection]struct{}),
		agentConns:   make(map[string]*Connection),
	}
}

// Register adds a connection to the hub.
func (h *Hub) Register(conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.projectConns[conn.ProjectID] == nil {
		h.projectConns[conn.ProjectID] = make(map[*Connection]struct{})
	}
	h.projectConns[conn.ProjectID][conn] = struct{}{}

	if conn.AgentID != "" {
		h.agentConns[conn.AgentID] = conn
	}
}

// Unregister removes a connection from the hub.
func (h *Hub) Unregister(conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conns, ok := h.projectConns[conn.ProjectID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.projectConns, conn.ProjectID)
		}
	}

	if conn.AgentID != "" {
		delete(h.agentConns, conn.AgentID)
	}
}

// Broadcast sends a message to all connections in a project.
func (h *Hub) Broadcast(projectID string, msg OutboundMessage) {
	h.mu.RLock()
	conns := h.projectConns[projectID]
	h.mu.RUnlock()

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[hub] marshal broadcast message: %v", err)
		return
	}

	for conn := range conns {
		select {
		case conn.send <- data:
		default:
			log.Printf("[hub] send buffer full for agent %s — dropping message", conn.AgentID)
		}
	}
}

// SendToAgent sends a message to a specific agent's connection.
func (h *Hub) SendToAgent(agentID string, msg OutboundMessage) {
	h.mu.RLock()
	conn := h.agentConns[agentID]
	h.mu.RUnlock()

	if conn == nil {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[hub] marshal agent message: %v", err)
		return
	}

	select {
	case conn.send <- data:
	default:
		log.Printf("[hub] send buffer full for agent %s — dropping message", agentID)
	}
}

// ConnectionCount returns the number of active connections for a project.
func (h *Hub) ConnectionCount(projectID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.projectConns[projectID])
}
