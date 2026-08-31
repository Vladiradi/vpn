package agentstream

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vpn-project/backend/internal/nodes"
)

const (
	writeTimeout = 10 * time.Second
	readTimeout  = 60 * time.Second
	pingInterval = 30 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, restrict this to known origins.
		return true
	},
}

// Handler handles agent websocket connections.
type Handler struct {
	hub  *Hub
	repo *nodes.Repository
}

// NewHandler creates a new websocket handler.
func NewHandler(hub *Hub, repo *nodes.Repository) *Handler {
	return &Handler{hub: hub, repo: repo}
}

// ServeHTTP upgrades the connection to websocket and handles the agent session.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	// First message must be hello with token.
	conn.SetReadDeadline(time.Now().Add(readTimeout))
	var msg Message
	if err := conn.ReadJSON(&msg); err != nil {
		log.Printf("agent failed to send hello: %v", err)
		conn.Close()
		return
	}

	if msg.Type != MessageTypeHello {
		_ = conn.WriteJSON(Message{Type: MessageTypeError, Payload: map[string]any{"message": "expected hello message"}})
		conn.Close()
		return
	}

	var hello HelloPayload
	payloadBytes, _ := json.Marshal(msg.Payload)
	if err := json.Unmarshal(payloadBytes, &hello); err != nil {
		_ = conn.WriteJSON(Message{Type: MessageTypeError, Payload: map[string]any{"message": "invalid hello payload"}})
		conn.Close()
		return
	}

	node, err := h.repo.GetByAPITokenHash(r.Context(), nodes.HashToken(hello.Token))
	if err != nil {
		_ = conn.WriteJSON(Message{Type: MessageTypeError, Payload: map[string]any{"message": "invalid token"}})
		conn.Close()
		return
	}

	// Update node status to online.
	if err := h.repo.UpdateStatus(r.Context(), node.ID, nodes.StatusOnline); err != nil {
		log.Printf("failed to update node status: %v", err)
	}

	agent := &Agent{
		NodeID: node.ID,
		Conn:   conn,
		Send:   make(chan Message, 16),
	}
	h.hub.Register(agent)

	// Start goroutines for reading and writing.
	go h.writePump(agent)
	go h.readPump(agent, node.ID)
}

func (h *Handler) readPump(agent *Agent, nodeID string) {
	defer func() {
		h.hub.Unregister(nodeID)
		if err := h.repo.UpdateStatus(context.Background(), nodeID, nodes.StatusOffline); err != nil {
			log.Printf("failed to set node offline: %v", err)
		}
	}()

	agent.Conn.SetReadDeadline(time.Now().Add(readTimeout))
	agent.Conn.SetPongHandler(func(string) error {
		agent.Conn.SetReadDeadline(time.Now().Add(readTimeout))
		return nil
	})

	for {
		var msg Message
		if err := agent.Conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("agent read error: %v", err)
			}
			return
		}

		agent.Conn.SetReadDeadline(time.Now().Add(readTimeout))

		switch msg.Type {
		case MessageTypeHeartbeat:
			if err := h.repo.UpdateStatus(context.Background(), nodeID, nodes.StatusOnline); err != nil {
				log.Printf("heartbeat status update failed: %v", err)
			}
		case MessageTypeStats:
			// TODO: aggregate traffic stats
		default:
			log.Printf("unknown message type from agent: %s", msg.Type)
		}
	}
}

func (h *Handler) writePump(agent *Agent) {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		agent.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-agent.Send:
			agent.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if !ok {
				agent.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := agent.Conn.WriteJSON(msg); err != nil {
				return
			}

		case <-ticker.C:
			agent.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := agent.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
