package traffic

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/vpn-project/backend/internal/middleware"
)

// Handler provides HTTP handlers for traffic statistics (admin only).
type Handler struct {
	repo RepositoryReader
}

// RepositoryReader is the read interface required by the traffic handler.
type RepositoryReader interface {
	ListAll(ctx context.Context, limit int) ([]Stat, error)
	ListByNode(ctx context.Context, nodeID string, limit int) ([]Stat, error)
	TotalsByNode(ctx context.Context, nodeID string) (uplink, downlink int64, err error)
}

// NewHandler creates a new traffic handler.
func NewHandler(repo RepositoryReader) *Handler {
	return &Handler{repo: repo}
}

// ListAll handles GET /api/v1/admin/traffic.
func (h *Handler) ListAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	stats, err := h.repo.ListAll(r.Context(), limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list traffic")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"traffic": stats})
}

// ListByNode handles GET /api/v1/admin/nodes/{id}/traffic.
func (h *Handler) ListByNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	nodeID := r.PathValue("id")
	if nodeID == "" {
		respondError(w, http.StatusBadRequest, "node id required")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	stats, err := h.repo.ListByNode(r.Context(), nodeID, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list node traffic")
		return
	}

	uplink, downlink, err := h.repo.TotalsByNode(r.Context(), nodeID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to compute node totals")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"node_id":  nodeID,
		"traffic":  stats,
		"totals":   map[string]int64{"uplink": uplink, "downlink": downlink},
		"total_mb": float64(uplink+downlink) / 1024 / 1024,
	})
}

// RegisterRoutes wires traffic routes into the mux. Called by server.Server.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMW func(http.Handler) http.Handler) {
	allTraffic := authMW(middleware.AdminOnly(http.HandlerFunc(h.ListAll)))
	mux.Handle("/api/v1/admin/traffic", allTraffic)

	nodeTraffic := authMW(middleware.AdminOnly(http.HandlerFunc(h.ListByNode)))
	mux.Handle("/api/v1/admin/nodes/{id}/traffic", nodeTraffic)
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
