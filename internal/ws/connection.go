package ws

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gofiber/websocket/v2"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 30 * time.Second
	maxMessageSize = 4096
)

// Connection wraps a Fiber WebSocket with read/write pumps.
type Connection struct {
	ProjectID string
	AgentID   string
	conn      *websocket.Conn
	hub       *Hub
	send      chan []byte

	// InboundMessages receives parsed inbound messages for external processing.
	InboundMessages chan InboundMessage
}

// NewConnection creates a new Connection and registers it with the Hub.
func NewConnection(conn *websocket.Conn, projectID, agentID string, hub *Hub) *Connection {
	c := &Connection{
		ProjectID:       projectID,
		AgentID:         agentID,
		conn:            conn,
		hub:             hub,
		send:            make(chan []byte, 256),
		InboundMessages: make(chan InboundMessage, 64),
	}
	hub.Register(c)
	return c
}

// Run starts the read and write pumps and blocks until the connection closes.
func (c *Connection) Run() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
		close(c.InboundMessages)
	}()

	go c.writePump()
	c.readPump()
}

// readPump reads messages from the WebSocket and dispatches them.
func (c *Connection) readPump() {
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[ws] read error for agent %s: %v", c.AgentID, err)
			}
			return
		}

		var msg InboundMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("[ws] invalid message from agent %s: %v", c.AgentID, err)
			continue
		}

		// Non-blocking send to inbound channel
		select {
		case c.InboundMessages <- msg:
		default:
			log.Printf("[ws] inbound buffer full for agent %s — dropping", c.AgentID)
		}
	}
}

// writePump writes queued messages to the WebSocket with periodic pings.
func (c *Connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case data, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Send queues a message for delivery.
func (c *Connection) Send(msg OutboundMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
	}
}
