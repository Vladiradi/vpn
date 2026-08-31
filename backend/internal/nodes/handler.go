package nodes

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Handler provides HTTP handlers for node management.
type Handler struct {
	repo *Repository
}

// NewHandler creates a new node handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateRequest represents a request to create a node.
type CreateRequest struct {
	Name     string  `json:"name"`
	Hostname string  `json:"hostname"`
	PublicIP *string `json:"public_ip,omitempty"`
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

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
