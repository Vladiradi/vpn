package agentstream

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/vpn-project/backend/internal/nodes"
)

// Agent represents a connected VPN node agent.
type Agent struct {
	NodeID string
	Conn   *websocket.Conn
	Send   chan Message
}

// Hub manages all connected agents.
type Hub struct {
	mu     sync.RWMutex
	agents map[string]*Agent // node_id -> agent
}

// NewHub creates a new agent hub.
func NewHub() *Hub {
	return &Hub{
		agents: make(map[string]*Agent),
	}
}

// Register adds an agent to the hub.
func (h *Hub) Register(a *Agent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if old, ok := h.agents[a.NodeID]; ok {
		close(old.Send)
		old.Conn.Close()
	}
	h.agents[a.NodeID] = a
}

// Unregister removes an agent from the hub.
func (h *Hub) Unregister(nodeID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if a, ok := h.agents[nodeID]; ok {
		delete(h.agents, nodeID)
		close(a.Send)
		a.Conn.Close()
	}
}

// Get returns an agent by node ID.
func (h *Hub) Get(nodeID string) (*Agent, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	a, ok := h.agents[nodeID]
	return a, ok
}

// SendConfig pushes a config message to a specific agent.
func (h *Hub) SendConfig(nodeID string, cfg nodes.Node) bool {
	h.mu.RLock()
	a, ok := h.agents[nodeID]
	h.mu.RUnlock()
	if !ok {
		return false
	}

	payload := map[string]any{
		"node_id":          cfg.ID,
		"reality_dest":     cfg.RealityDest,
		"reality_public_key": cfg.RealityPublicKey,
		"reality_short_ids": cfg.RealityShortIDs,
		"decoy_host":       cfg.DecoyHost,
	}

	select {
	case a.Send <- Message{Type: MessageTypeConfig, Payload: payload}:
		return true
	default:
		return false
	}
}

// Broadcast sends a message to all connected agents.
func (h *Hub) Broadcast(msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, a := range h.agents {
		select {
		case a.Send <- msg:
		default:
		}
	}
}
