package ws

import (
	"encoding/json"
	"testing"
)

// TestHubRegisterUnregister tests basic registration and unregistration.
func TestHubRegisterUnregister(t *testing.T) {
	hub := NewHub()

	conn1 := &Connection{
		ProjectID: "proj-1",
		AgentID:   "agent-1",
		send:      make(chan []byte, 10),
	}
	conn2 := &Connection{
		ProjectID: "proj-1",
		AgentID:   "agent-2",
		send:      make(chan []byte, 10),
	}

	hub.Register(conn1)
	hub.Register(conn2)

	if count := hub.ConnectionCount("proj-1"); count != 2 {
		t.Errorf("expected 2 connections, got %d", count)
	}

	hub.Unregister(conn1)
	if count := hub.ConnectionCount("proj-1"); count != 1 {
		t.Errorf("expected 1 connection after unregister, got %d", count)
	}

	hub.Unregister(conn2)
	if count := hub.ConnectionCount("proj-1"); count != 0 {
		t.Errorf("expected 0 connections after all unregistered, got %d", count)
	}
}

// TestHubBroadcast tests that Broadcast sends to all project connections.
func TestHubBroadcast(t *testing.T) {
	hub := NewHub()

	conn1 := &Connection{ProjectID: "proj-1", AgentID: "agent-1", send: make(chan []byte, 10)}
	conn2 := &Connection{ProjectID: "proj-1", AgentID: "agent-2", send: make(chan []byte, 10)}
	conn3 := &Connection{ProjectID: "proj-2", AgentID: "agent-3", send: make(chan []byte, 10)}

	hub.Register(conn1)
	hub.Register(conn2)
	hub.Register(conn3)

	msg := OutboundMessage{Type: MsgTypeTaskAssigned, Payload: "test"}
	hub.Broadcast("proj-1", msg)

	// conn1 and conn2 should receive the message
	if len(conn1.send) != 1 {
		t.Errorf("conn1: expected 1 message, got %d", len(conn1.send))
	}
	if len(conn2.send) != 1 {
		t.Errorf("conn2: expected 1 message, got %d", len(conn2.send))
	}
	// conn3 (different project) should not receive it
	if len(conn3.send) != 0 {
		t.Errorf("conn3 (different project): expected 0 messages, got %d", len(conn3.send))
	}

	// Verify message content
	data := <-conn1.send
	var got OutboundMessage
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal message: %v", err)
	}
	if got.Type != MsgTypeTaskAssigned {
		t.Errorf("expected type %q, got %q", MsgTypeTaskAssigned, got.Type)
	}
}

// TestHubSendToAgent tests that SendToAgent delivers to a specific agent.
func TestHubSendToAgent(t *testing.T) {
	hub := NewHub()

	conn1 := &Connection{ProjectID: "proj-1", AgentID: "agent-1", send: make(chan []byte, 10)}
	conn2 := &Connection{ProjectID: "proj-1", AgentID: "agent-2", send: make(chan []byte, 10)}

	hub.Register(conn1)
	hub.Register(conn2)

	msg := OutboundMessage{Type: MsgTypeClaimGranted, Payload: "for-agent-1"}
	hub.SendToAgent("agent-1", msg)

	if len(conn1.send) != 1 {
		t.Errorf("agent-1: expected 1 message, got %d", len(conn1.send))
	}
	if len(conn2.send) != 0 {
		t.Errorf("agent-2: expected 0 messages, got %d", len(conn2.send))
	}
}

// TestHubSendToUnknownAgent tests that SendToAgent is a no-op for unknown agents.
func TestHubSendToUnknownAgent(t *testing.T) {
	hub := NewHub()
	// Should not panic
	hub.SendToAgent("unknown-agent", OutboundMessage{Type: MsgTypePing})
}

// TestHubBroadcastEmptyProject tests Broadcast to a project with no connections.
func TestHubBroadcastEmptyProject(t *testing.T) {
	hub := NewHub()
	// Should not panic
	hub.Broadcast("no-such-project", OutboundMessage{Type: MsgTypePing})
}

// TestHubConnectionWithoutAgent tests connections that have no agent ID.
func TestHubConnectionWithoutAgent(t *testing.T) {
	hub := NewHub()

	conn := &Connection{ProjectID: "proj-1", AgentID: "", send: make(chan []byte, 10)}
	hub.Register(conn)

	if count := hub.ConnectionCount("proj-1"); count != 1 {
		t.Errorf("expected 1 connection, got %d", count)
	}

	hub.Unregister(conn)
	if count := hub.ConnectionCount("proj-1"); count != 0 {
		t.Errorf("expected 0 connections after unregister, got %d", count)
	}
}
