// Package server provides the HTTP server and route definitions.
package server

import (
	"encoding/json"
	"net/http"

	"github.com/vpn-project/backend/internal/agentstream"
	"github.com/vpn-project/backend/internal/auth"
	"github.com/vpn-project/backend/internal/config"
	"github.com/vpn-project/backend/internal/middleware"
	"github.com/vpn-project/backend/internal/nodes"
	"github.com/vpn-project/backend/internal/storage"
	"github.com/vpn-project/backend/internal/users"
)

// Server is the HTTP application server.
type Server struct {
	cfg            *config.Config
	storage        *storage.Storage
	authHandler    *auth.Handler
	nodeHandler    *nodes.Handler
	agentWSHandler *agentstream.Handler
	authMW         func(http.Handler) http.Handler
	mux            *http.ServeMux
}

// New creates a new Server instance with configured routes.
func New(cfg *config.Config, st *storage.Storage, userRepo *users.Repository, nodeRepo *nodes.Repository, tokenService *auth.TokenService, agentWS *agentstream.Handler, hub *agentstream.Hub) *Server {
	s := &Server{
		cfg:            cfg,
		storage:        st,
		authHandler:    auth.NewHandler(cfg, userRepo, tokenService),
		nodeHandler:    nodes.NewHandler(nodeRepo, hub),
		agentWSHandler: agentWS,
		authMW:         middleware.Auth(tokenService),
		mux:            http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/api/v1/info", s.handleInfo)

	// Auth endpoints
	s.mux.HandleFunc("/api/v1/auth/register", s.authHandler.Register)
	s.mux.HandleFunc("/api/v1/auth/login", s.authHandler.Login)
	s.mux.HandleFunc("/api/v1/auth/google", s.authHandler.GoogleLogin)
	s.mux.HandleFunc("/api/v1/auth/telegram", s.authHandler.TelegramLogin)

	// Agent websocket endpoint (authenticated by node token, not JWT)
	s.mux.Handle("/api/v1/nodes/stream", s.agentWSHandler)

	// Admin-only routes: authMW runs first (sets user context), then AdminOnly checks role.
	adminDashboard := s.authMW(middleware.AdminOnly(http.HandlerFunc(s.handleAdminDashboard)))
	s.mux.Handle("/api/v1/admin/dashboard", adminDashboard)

	// Node management (admin only)
	nodesHandler := s.authMW(middleware.AdminOnly(http.HandlerFunc(s.handleNodes)))
	s.mux.Handle("/api/v1/nodes", nodesHandler)

	nodeByIDHandler := s.authMW(middleware.AdminOnly(http.HandlerFunc(s.handleNodeByID)))
	s.mux.Handle("/api/v1/nodes/{id}", nodeByIDHandler)

	nodeConfigHandler := s.authMW(middleware.AdminOnly(http.HandlerFunc(s.nodeHandler.UpdateConfig)))
	s.mux.Handle("/api/v1/nodes/{id}/config", nodeConfigHandler)
}

func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.nodeHandler.List(w, r)
	case http.MethodPost:
		s.nodeHandler.Create(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleNodeByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.nodeHandler.Get(w, r)
	case http.MethodDelete:
		s.nodeHandler.Delete(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	code := http.StatusOK

	if s.storage != nil {
		if err := s.storage.Ping(r.Context()); err != nil {
			status = "degraded: " + err.Error()
			code = http.StatusServiceUnavailable
		}
	}

	respondJSON(w, code, map[string]string{"status": status})
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"service": "vpn-panel",
		"version": "0.1.0",
	})
}

func (s *Server) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "admin dashboard",
	})
}

// Handler returns the http.Handler for the server.
func (s *Server) Handler() http.Handler {
	return s.mux
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
