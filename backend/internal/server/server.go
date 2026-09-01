// Package server provides the HTTP server and route definitions.
package server

import (
	"encoding/json"
	"net/http"

	"github.com/vpn-project/backend/internal/agentstream"
	"github.com/vpn-project/backend/internal/auth"
	"github.com/vpn-project/backend/internal/billing"
	"github.com/vpn-project/backend/internal/clientapps"
	"github.com/vpn-project/backend/internal/config"
	"github.com/vpn-project/backend/internal/devices"
	"github.com/vpn-project/backend/internal/middleware"
	"github.com/vpn-project/backend/internal/nodes"
	"github.com/vpn-project/backend/internal/offerings"
	"github.com/vpn-project/backend/internal/payments"
	"github.com/vpn-project/backend/internal/storage"
	"github.com/vpn-project/backend/internal/subscription"
	"github.com/vpn-project/backend/internal/users"
)

// Server is the HTTP application server.
type Server struct {
	cfg                 *config.Config
	storage             *storage.Storage
	authHandler         *auth.Handler
	userHandler         *users.Handler
	offeringHandler     *offerings.Handler
	billingHandler      *billing.Handler
	clientAppsHandler   *clientapps.Handler
	nodeHandler         *nodes.Handler
	deviceHandler       *devices.Handler
	subscriptionHandler *subscription.Handler
	publicSubHandler    *subscription.PublicHandler
	agentWSHandler      *agentstream.Handler
	authMW              func(http.Handler) http.Handler
	mux                 *http.ServeMux
}

// New creates a new Server instance with configured routes.
func New(cfg *config.Config, st *storage.Storage, userRepo *users.Repository, offeringRepo *offerings.Repository, billingRepo *billing.Repository, nodeRepo *nodes.Repository, deviceRepo *devices.Repository, tokenService *auth.TokenService, agentWS *agentstream.Handler, hub *agentstream.Hub) *Server {
	s := &Server{
		cfg:                 cfg,
		storage:             st,
		authHandler:         auth.NewHandler(cfg, userRepo, tokenService),
		userHandler:         users.NewHandler(userRepo, passwordHasher{}),
		offeringHandler:     offerings.NewHandler(offeringRepo),
		billingHandler:      billing.NewHandler(billingRepo, offeringRepo, deviceRepo, nodeRepo, hub, paymentsProvider(cfg)),
		clientAppsHandler:   clientapps.NewHandler(),
		nodeHandler:         nodes.NewHandler(nodeRepo, hub),
		deviceHandler:       devices.NewHandler(deviceRepo, nodeRepo, hub),
		subscriptionHandler: subscription.NewHandler(deviceRepo, nodeRepo),
		publicSubHandler:    subscription.NewPublicHandler(billingRepo, deviceRepo, nodeRepo),
		agentWSHandler:      agentWS,
		authMW:              middleware.Auth(tokenService),
		mux:                 http.NewServeMux(),
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

	// Admin-only routes
	adminDashboard := s.authMW(middleware.AdminOnly(http.HandlerFunc(s.handleAdminDashboard)))
	s.mux.Handle("/api/v1/admin/dashboard", adminDashboard)

	// Node management (admin only)
	nodesHandler := s.authMW(middleware.AdminOnly(http.HandlerFunc(s.handleNodes)))
	s.mux.Handle("/api/v1/nodes", nodesHandler)

	nodeByIDHandler := s.authMW(middleware.AdminOnly(http.HandlerFunc(s.handleNodeByID)))
	s.mux.Handle("/api/v1/nodes/{id}", nodeByIDHandler)

	nodeConfigHandler := s.authMW(middleware.AdminOnly(http.HandlerFunc(s.nodeHandler.UpdateConfig)))
	s.mux.Handle("/api/v1/nodes/{id}/config", nodeConfigHandler)

	// User management (admin only)
	usersHandler := s.authMW(middleware.AdminOnly(http.HandlerFunc(s.handleUsers)))
	s.mux.Handle("/api/v1/users", usersHandler)

	userByIDHandler := s.authMW(middleware.AdminOnly(http.HandlerFunc(s.handleUserByID)))
	s.mux.Handle("/api/v1/users/{id}", userByIDHandler)

	userStatusHandler := s.authMW(middleware.AdminOnly(http.HandlerFunc(s.userHandler.SetStatus)))
	s.mux.Handle("/api/v1/users/{id}/status", userStatusHandler)

	adminDeviceHandler := s.authMW(middleware.AdminOnly(http.HandlerFunc(s.deviceHandler.CreateForUser)))
	s.mux.Handle("/api/v1/users/{id}/devices", adminDeviceHandler)

	// Plans (public list + admin CRUD)
	plansHandler := s.authMW(middleware.AdminOnly(http.HandlerFunc(s.handleOfferings)))
	s.mux.Handle("/api/v1/offerings", plansHandler)

	planByIDHandler := s.authMW(middleware.AdminOnly(http.HandlerFunc(s.handleOfferingByID)))
	s.mux.Handle("/api/v1/offerings/{id}", planByIDHandler)

	// User routes (authenticated)
	meHandler := s.authMW(http.HandlerFunc(s.handleMe))
	s.mux.Handle("/api/v1/me", meHandler)

	myDevicesHandler := s.authMW(http.HandlerFunc(s.handleMyDevices))
	s.mux.Handle("/api/v1/me/devices", myDevicesHandler)

	myDeviceConfigHandler := s.authMW(http.HandlerFunc(s.subscriptionHandler.GetConfig))
	s.mux.Handle("/api/v1/me/devices/{id}/config", myDeviceConfigHandler)

	myDeviceQRHandler := s.authMW(http.HandlerFunc(s.subscriptionHandler.GetQR))
	s.mux.Handle("/api/v1/me/devices/{id}/qr", myDeviceQRHandler)

	myDeviceFileHandler := s.authMW(http.HandlerFunc(s.subscriptionHandler.GetFile))
	s.mux.Handle("/api/v1/me/devices/{id}/file", myDeviceFileHandler)

	// Public active plans list (no auth required)
	s.mux.HandleFunc("/api/v1/public/offerings", s.offeringHandler.List)

	// Subscriptions (user)
	mySubscriptionsHandler := s.authMW(http.HandlerFunc(s.handleMySubscriptions))
	s.mux.Handle("/api/v1/subscriptions", mySubscriptionsHandler)

	mySubscriptionByIDHandler := s.authMW(http.HandlerFunc(s.handleSubscriptionByID))
	s.mux.Handle("/api/v1/subscriptions/{id}", mySubscriptionByIDHandler)

	paySubscriptionHandler := s.authMW(http.HandlerFunc(s.billingHandler.Pay))
	s.mux.Handle("/api/v1/subscriptions/{id}/pay", paySubscriptionHandler)

	// Subscriptions (admin)
	allSubscriptionsHandler := s.authMW(middleware.AdminOnly(http.HandlerFunc(s.billingHandler.ListAll)))
	s.mux.Handle("/api/v1/admin/subscriptions", allSubscriptionsHandler)

	adminActivateSubscriptionHandler := s.authMW(middleware.AdminOnly(http.HandlerFunc(s.billingHandler.AdminActivate)))
	s.mux.Handle("/api/v1/admin/subscriptions/{id}/activate", adminActivateSubscriptionHandler)

	// Client apps (public)
	s.mux.HandleFunc("/api/v1/client-apps", s.clientAppsHandler.List)

	// Public subscription link
	s.mux.Handle("/sub/{token}", s.publicSubHandler)
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

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.userHandler.List(w, r)
	case http.MethodPost:
		s.userHandler.Create(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleUserByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.userHandler.Get(w, r)
	case http.MethodPut, http.MethodPatch:
		s.userHandler.Update(w, r)
	case http.MethodDelete:
		s.userHandler.Delete(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, ok := middleware.UserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"user_id": claims.UserID,
		"email":   claims.Email,
		"role":    claims.Role,
	})
}

func (s *Server) handleOfferings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.offeringHandler.List(w, r)
	case http.MethodPost:
		s.offeringHandler.Create(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleOfferingByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.offeringHandler.Get(w, r)
	case http.MethodPut, http.MethodPatch:
		s.offeringHandler.Update(w, r)
	case http.MethodDelete:
		s.offeringHandler.Delete(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleMySubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.billingHandler.ListForMe(w, r)
	case http.MethodPost:
		s.billingHandler.Create(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleSubscriptionByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.billingHandler.Get(w, r)
}

func (s *Server) handleMyDevices(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.deviceHandler.ListForMe(w, r)
	case http.MethodPost:
		s.deviceHandler.CreateForMe(w, r)
	case http.MethodDelete:
		s.deviceHandler.RevokeForMe(w, r)
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

func paymentsProvider(cfg *config.Config) payments.Provider {
	switch cfg.PaymentProvider {
	case "platega":
		return payments.NewPlategaProvider(cfg.PlategaAPIKey, cfg.PlategaEndpoint)
	default:
		return payments.NewStubProvider()
	}
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

// passwordHasher adapts auth.HashPassword to users.PasswordHasher.
type passwordHasher struct{}

func (passwordHasher) Hash(password string) (string, error) {
	return auth.HashPassword(password)
}
