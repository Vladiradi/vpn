package nodes

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/vpn-project/backend/internal/xray"
)

// ConfigPusher pushes a generated XRay config to a node agent.
type ConfigPusher interface {
	PushConfig(ctx context.Context, nodeID string) error
}

// Handler provides HTTP handlers for node management.
type Handler struct {
	repo   *Repository
	pusher ConfigPusher
}

// NewHandler creates a new node handler.
func NewHandler(repo *Repository, pusher ConfigPusher) *Handler {
	return &Handler{repo: repo, pusher: pusher}
}

// CreateRequest represents a request to create a node.
type CreateRequest struct {
	Name     string  `json:"name"`
	Hostname string  `json:"hostname"`
	PublicIP *string `json:"public_ip,omitempty"`
}

// UpdateConfigRequest represents a request to update node XRay/REALITY settings.
type UpdateConfigRequest struct {
	RealityDest      *string  `json:"reality_dest,omitempty"`
	RealityPublicKey *string  `json:"reality_public_key,omitempty"`
	RealityShortIDs  []string `json:"reality_short_ids,omitempty"`
	DecoyHost        *string  `json:"decoy_host,omitempty"`
}

// Create handles node creation (admin only).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Hostname == "" {
		respondError(w, http.StatusBadRequest, "name and hostname required")
		return
	}

	node, enrollToken, err := h.repo.Create(r.Context(), req.Name, req.Hostname, req.PublicIP)
	if err != nil {
		log.Printf("create node error: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to create node")
		return
	}

	// Return node with plaintext enroll token. This is the only time it's shown.
	respondJSON(w, http.StatusCreated, map[string]any{
		"node":         node,
		"enroll_token": enrollToken,
		"docker_run_command": "docker compose -f docker-compose.vpn.yml up -d",
	})
}

// List handles listing all nodes (admin only).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	nodes, err := h.repo.List(r.Context())
	if err != nil {
		log.Printf("list nodes error: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to list nodes")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"nodes": nodes})
}

// Get handles getting a single node (admin only).
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "node id required")
		return
	}

	node, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNodeNotFound) {
			respondError(w, http.StatusNotFound, "node not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get node")
		return
	}

	respondJSON(w, http.StatusOK, node)
}

// Delete handles node deletion (admin only).
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "node id required")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNodeNotFound) {
			respondError(w, http.StatusNotFound, "node not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to delete node")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// UpdateConfig handles updating XRay/REALITY settings for a node (admin only).
func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "node id required")
		return
	}

	var req UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx := r.Context()
	node, err := h.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNodeNotFound) {
			respondError(w, http.StatusNotFound, "node not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get node")
		return
	}

	// Merge incoming values with existing node, generating keys/ids when missing.
	cfg := mergeNodeConfig(node, req)
	if err := h.repo.UpdateConfig(ctx, id, cfg); err != nil {
		if errors.Is(err, ErrNodeNotFound) {
			respondError(w, http.StatusNotFound, "node not found")
			return
		}
		log.Printf("update node config error: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to update node config")
		return
	}

	// Push updated config to the agent if it is online.
	if h.pusher != nil {
		if err := h.pusher.PushConfig(ctx, id); err != nil {
			log.Printf("push config to node %s failed: %v", id, err)
			// Return 202; the DB is updated and the agent will get the config on reconnect.
			respondJSON(w, http.StatusAccepted, map[string]any{
				"status":  "stored",
				"warning": "agent offline or config push failed",
			})
			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func mergeNodeConfig(node *Node, req UpdateConfigRequest) NodeConfigUpdate {
	dest := req.RealityDest
	if dest == nil || *dest == "" {
		dest = node.RealityDest
	}

	shortIDs := req.RealityShortIDs
	if len(shortIDs) == 0 {
		shortIDs = node.RealityShortIDs
	}

	decoy := req.DecoyHost
	if decoy == nil || *decoy == "" {
		decoy = node.DecoyHost
	}

	var privateKey, publicKey *string
	if req.RealityPublicKey != nil && *req.RealityPublicKey != "" {
		// If public key is explicitly supplied, generate a matching private key.
		// In normal flow the backend generates both and only public key is exposed.
		pk, pub, err := xray.GenerateRealityKeypair()
		if err == nil {
			privateKey = &pk
			publicKey = &pub
		}
	} else if node.RealityPrivateKey == nil || *node.RealityPrivateKey == "" {
		pk, pub, err := xray.GenerateRealityKeypair()
		if err == nil {
			privateKey = &pk
			publicKey = &pub
		}
	} else {
		privateKey = node.RealityPrivateKey
		publicKey = node.RealityPublicKey
	}

	if len(shortIDs) == 0 {
		if ids, err := xray.GenerateShortIDs(3); err == nil {
			shortIDs = ids
		}
	}

	return NodeConfigUpdate{
		RealityDest:       dest,
		RealityPrivateKey: privateKey,
		RealityPublicKey:  publicKey,
		RealityShortIDs:   shortIDs,
		DecoyHost:         decoy,
	}
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
