package agentstream

import (
	"context"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/vpn-project/backend/internal/nodes"
	"github.com/vpn-project/backend/internal/xray"
)

// Agent represents a connected VPN node agent.
type Agent struct {
	NodeID string
	Conn   *websocket.Conn
	Send   chan Message
}

// Hub manages all connected agents.
type Hub struct {
	mu       sync.RWMutex
	agents   map[string]*Agent // node_id -> agent
	nodeRepo *nodes.Repository
}

// NewHub creates a new agent hub.
func NewHub() *Hub {
	return &Hub{
		agents: make(map[string]*Agent),
	}
}

// WithRepository returns a hub that can load node configuration from storage.
func (h *Hub) WithRepository(repo *nodes.Repository) *Hub {
	h.nodeRepo = repo
	return h
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

// SendConfig pushes a full XRay config message to a specific agent.
func (h *Hub) SendConfig(nodeID string, cfg nodes.Node) bool {
	xrayCfg, err := xray.NewConfigBuilder(toXrayNodeConfig(cfg), nil).Build()
	if err != nil {
		return false
	}
	return h.sendConfigPayload(nodeID, xrayCfg)
}

// PushConfig loads a node from storage, builds its XRay config, and sends it
// to the connected agent. It implements nodes.ConfigPusher.
func (h *Hub) PushConfig(ctx context.Context, nodeID string) error {
	if h.nodeRepo == nil {
		return fmt.Errorf("node repository not configured")
	}

	node, err := h.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("load node: %w", err)
	}

	xrayCfg, err := xray.NewConfigBuilder(toXrayNodeConfig(*node), nil).Build()
	if err != nil {
		return fmt.Errorf("build xray config: %w", err)
	}

	if !h.sendConfigPayload(nodeID, xrayCfg) {
		return fmt.Errorf("agent not connected")
	}
	return nil
}

func toXrayNodeConfig(n nodes.Node) xray.NodeConfig {
	cfg := xray.NodeConfig{
		RealityShortIDs: n.RealityShortIDs,
	}
	if n.RealityDest != nil {
		cfg.RealityDest = *n.RealityDest
	}
	if n.RealityPrivateKey != nil {
		cfg.RealityPrivateKey = *n.RealityPrivateKey
	}
	if n.DecoyHost != nil {
		cfg.DecoyHost = *n.DecoyHost
	}
	return cfg
}

func (h *Hub) sendConfigPayload(nodeID string, xrayCfg map[string]any) bool {
	h.mu.RLock()
	a, ok := h.agents[nodeID]
	h.mu.RUnlock()
	if !ok {
		return false
	}

	payload := map[string]any{
		"node_id":     nodeID,
		"xray_config": xrayCfg,
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
