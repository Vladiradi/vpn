package agentstream

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vpn-project/backend/internal/devices"
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
	hub         *Hub
	repo        *nodes.Repository
	deviceRepo  *devices.Repository
	trafficRepo TrafficRepository
	billingRepo BillingRepository
}

// TrafficRepository stores traffic statistics.
type TrafficRepository interface {
	Save(ctx context.Context, deviceID, nodeID string, uplink, downlink int64) error
}

// BillingRepository updates subscription traffic totals.
type BillingRepository interface {
	AddTraffic(ctx context.Context, id string, bytes int64) error
}

// NewHandler creates a new websocket handler.
func NewHandler(hub *Hub, repo *nodes.Repository, deviceRepo *devices.Repository, trafficRepo TrafficRepository, billingRepo BillingRepository) *Handler {
	return &Handler{
		hub:         hub,
		repo:        repo,
		deviceRepo:  deviceRepo,
		trafficRepo: trafficRepo,
		billingRepo: billingRepo,
	}
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
			if err := h.handleStats(nodeID, msg.Payload); err != nil {
				log.Printf("handle stats failed: %v", err)
			}
		default:
			log.Printf("unknown message type from agent: %s", msg.Type)
		}
	}
}

func (h *Handler) handleStats(nodeID string, payload map[string]any) error {
	devicesRaw, ok := payload["devices"].([]any)
	if !ok {
		return fmt.Errorf("invalid devices payload")
	}

	ctx := context.Background()
	for _, d := range devicesRaw {
		raw, err := json.Marshal(d)
		if err != nil {
			continue
		}
		var ds DeviceStat
		if err := json.Unmarshal(raw, &ds); err != nil {
			continue
		}
		if ds.UUID == "" {
			continue
		}

		device, err := h.deviceRepo.GetByUUID(ctx, ds.UUID)
		if err != nil {
			continue
		}

		if err := h.trafficRepo.Save(ctx, device.ID, nodeID, ds.Uplink, ds.Downlink); err != nil {
			log.Printf("save traffic stat failed: %v", err)
			continue
		}

		if device.SubscriptionID != nil && h.billingRepo != nil {
			if err := h.billingRepo.AddTraffic(ctx, *device.SubscriptionID, ds.Uplink+ds.Downlink); err != nil {
				log.Printf("update subscription traffic failed: %v", err)
			}
		}
	}
	return nil
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
