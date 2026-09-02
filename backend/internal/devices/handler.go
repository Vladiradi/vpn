package devices

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/vpn-project/backend/internal/middleware"
	"github.com/vpn-project/backend/internal/nodes"
)

// ConfigPusher pushes an updated XRay config to a node agent.
type ConfigPusher interface {
	PushConfig(ctx context.Context, nodeID string) error
}

// Handler provides HTTP handlers for device management.
type Handler struct {
	repo     *Repository
	nodeRepo *nodes.Repository
	pusher   ConfigPusher
}

// NewHandler creates a new device handler.
func NewHandler(repo *Repository, nodeRepo *nodes.Repository, pusher ConfigPusher) *Handler {
	return &Handler{repo: repo, nodeRepo: nodeRepo, pusher: pusher}
}

// CreateRequest represents a request to create a device.
type CreateRequest struct {
	Name   string  `json:"name"`
	NodeID *string `json:"node_id,omitempty"`
}

// CreateForMe handles a user creating their own device.
func (h *Handler) CreateForMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, ok := middleware.UserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	device, err := h.create(w, r.Context(), claims.UserID, r)
	if err != nil {
		return
	}

	respondJSON(w, http.StatusCreated, device)
}

// CreateForUser handles an admin creating a device for a specific user.
func (h *Handler) CreateForUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := r.PathValue("id")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "user id required")
		return
	}

	device, err := h.create(w, r.Context(), userID, r)
	if err != nil {
		return
	}

	respondJSON(w, http.StatusCreated, device)
}

func (h *Handler) create(w http.ResponseWriter, ctx context.Context, userID string, r *http.Request) (*Device, error) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return nil, err
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name required")
		return nil, errors.New("name required")
	}
	if req.NodeID != nil && *req.NodeID != "" {
		if _, err := h.nodeRepo.GetByID(ctx, *req.NodeID); err != nil {
			if errors.Is(err, nodes.ErrNodeNotFound) {
				respondError(w, http.StatusNotFound, "node not found")
				return nil, err
			}
			respondError(w, http.StatusInternalServerError, "failed to validate node")
			return nil, err
		}
	}

	if err := h.repo.CheckLimits(ctx, userID, req.NodeID); err != nil {
		if errors.Is(err, ErrLimitExceeded) {
			respondError(w, http.StatusConflict, err.Error())
			return nil, err
		}
		log.Printf("check device limits error: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to check device limits")
		return nil, err
	}

	device, err := h.repo.Create(ctx, userID, req.Name, "vless", req.NodeID)
	if err != nil {
		log.Printf("create device error: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to create device")
		return nil, err
	}

	if h.pusher != nil && device.NodeID != nil {
		if err := h.pusher.PushConfig(ctx, *device.NodeID); err != nil {
			log.Printf("push config after device create failed: %v", err)
		}
	}

	return device, nil
}

// ListForMe handles listing the current user's devices.
func (h *Handler) ListForMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, ok := middleware.UserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	devices, err := h.repo.ListByUser(r.Context(), claims.UserID)
	if err != nil {
		log.Printf("list devices error: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to list devices")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"devices": devices})
}

// RevokeForMe handles a user revoking their own device.
func (h *Handler) RevokeForMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, ok := middleware.UserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "device id required")
		return
	}

	ctx := r.Context()
	device, err := h.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			respondError(w, http.StatusNotFound, "device not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to load device")
		return
	}

	if device.UserID != claims.UserID {
		respondError(w, http.StatusForbidden, "access denied")
		return
	}

	if err := h.repo.Revoke(ctx, id, claims.UserID); err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			respondError(w, http.StatusNotFound, "device not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to revoke device")
		return
	}

	if h.pusher != nil && device.NodeID != nil {
		if err := h.pusher.PushConfig(ctx, *device.NodeID); err != nil {
			log.Printf("push config after device revoke failed: %v", err)
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
