package billing

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/vpn-project/backend/internal/devices"
	"github.com/vpn-project/backend/internal/middleware"
	"github.com/vpn-project/backend/internal/nodes"
	"github.com/vpn-project/backend/internal/offerings"
	"github.com/vpn-project/backend/internal/payments"
)

// Handler provides HTTP handlers for subscription management.
type Handler struct {
	repo         *Repository
	offeringRepo *offerings.Repository
	deviceRepo   *devices.Repository
	nodeRepo     *nodes.Repository
	pusher       ConfigPusher
	payment      payments.Provider
}

// ConfigPusher pushes an updated XRay config to a node agent.
type ConfigPusher interface {
	PushConfig(ctx context.Context, nodeID string) error
}

// NewHandler creates a new subscription handler.
func NewHandler(repo *Repository, offeringRepo *offerings.Repository, deviceRepo *devices.Repository, nodeRepo *nodes.Repository, pusher ConfigPusher, payment payments.Provider) *Handler {
	return &Handler{
		repo:         repo,
		offeringRepo: offeringRepo,
		deviceRepo:   deviceRepo,
		nodeRepo:     nodeRepo,
		pusher:       pusher,
		payment:      payment,
	}
}

// CreateRequest represents a request to create a subscription.
type CreateRequest struct {
	OfferingID string `json:"offering_id"`
}

// Create handles subscription creation (user only).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, ok := middleware.UserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.OfferingID == "" {
		respondError(w, http.StatusBadRequest, "offering_id required")
		return
	}

	ctx := r.Context()
	offering, err := h.offeringRepo.GetByID(ctx, req.OfferingID)
	if err != nil {
		if errors.Is(err, offerings.ErrOfferingNotFound) {
			respondError(w, http.StatusNotFound, "offering not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to load offering")
		return
	}
	if !offering.IsActive || !offering.IsAvailable {
		respondError(w, http.StatusBadRequest, "offering is not available")
		return
	}

	sub, err := h.repo.Create(ctx, claims.UserID, offering.ID)
	if err != nil {
		log.Printf("create subscription error: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to create subscription")
		return
	}

	respondJSON(w, http.StatusCreated, sub)
}

// Pay handles the payment stub for a subscription.
func (h *Handler) Pay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
		respondError(w, http.StatusBadRequest, "subscription id required")
		return
	}

	ctx := r.Context()
	sub, err := h.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			respondError(w, http.StatusNotFound, "subscription not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to load subscription")
		return
	}
	if sub.UserID != claims.UserID {
		respondError(w, http.StatusForbidden, "access denied")
		return
	}
	if sub.Status != StatusPending {
		respondError(w, http.StatusBadRequest, "subscription is not pending")
		return
	}

	offering, err := h.offeringRepo.GetByID(ctx, sub.OfferingID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load offering")
		return
	}

	paid, err := h.payment.VerifyPayment(ctx, sub.ID)
	if err != nil {
		log.Printf("verify payment error: %v", err)
		respondError(w, http.StatusBadRequest, "payment verification failed")
		return
	}
	if !paid {
		respondError(w, http.StatusBadRequest, "payment not completed")
		return
	}

	activated, err := h.activateAndProvision(ctx, sub, offering)
	if err != nil {
		log.Printf("activate subscription error: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to activate subscription")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":       "paid",
		"subscription": activated,
	})
}

// Get returns a single subscription.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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
		respondError(w, http.StatusBadRequest, "subscription id required")
		return
	}

	sub, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			respondError(w, http.StatusNotFound, "subscription not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to load subscription")
		return
	}
	if sub.UserID != claims.UserID {
		respondError(w, http.StatusForbidden, "access denied")
		return
	}

	respondJSON(w, http.StatusOK, sub)
}

// ListForMe returns the current user's subscriptions.
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

	subs, err := h.repo.List(r.Context(), claims.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list subscriptions")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"subscriptions": subs})
}

// ListAll returns all subscriptions (admin only).
func (h *Handler) ListAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	subs, err := h.repo.List(r.Context(), "")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list subscriptions")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"subscriptions": subs})
}

// AdminActivate allows an admin to manually activate a subscription.
func (h *Handler) AdminActivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "subscription id required")
		return
	}

	ctx := r.Context()
	sub, err := h.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			respondError(w, http.StatusNotFound, "subscription not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to load subscription")
		return
	}

	offering, err := h.offeringRepo.GetByID(ctx, sub.OfferingID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load offering")
		return
	}

	activated, err := h.activateAndProvision(ctx, sub, offering)
	if err != nil {
		log.Printf("admin activate subscription error: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to activate subscription")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":       "activated",
		"subscription": activated,
		"offering":     offering,
	})
}

func (h *Handler) activateAndProvision(ctx context.Context, sub *Subscription, offering *offerings.Offering) (*Subscription, error) {
	activated, err := h.repo.Activate(ctx, sub.ID, offering.DurationMonths)
	if err != nil {
		return nil, err
	}

	// Find an available node for the offering country (MVP — only Germany is available).
	node, err := h.nodeRepo.FindByCountry(ctx, offering.CountryCode)
	if err != nil {
		return nil, err
	}

	// Create a device linked to the subscription.
	device, err := h.deviceRepo.Create(ctx, sub.UserID, offering.CountryName, "vless", &node.ID)
	if err != nil {
		return nil, err
	}

	// Link device to subscription.
	if err := h.deviceRepo.LinkToSubscription(ctx, device.ID, sub.ID); err != nil {
		return nil, err
	}

	// Push updated XRay config to the agent.
	if h.pusher != nil {
		if err := h.pusher.PushConfig(ctx, node.ID); err != nil {
			log.Printf("push config after subscription activation failed: %v", err)
		}
	}

	return activated, nil
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
