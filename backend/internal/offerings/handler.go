package offerings

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
)

// Handler provides HTTP handlers for offering management.
type Handler struct {
	repo *Repository
}

// NewHandler creates a new offering handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateRequest represents a request to create an offering.
type CreateRequest struct {
	Name           string `json:"name"`
	DurationMonths int    `json:"duration_months"`
	PriceRubles    int    `json:"price_rubles"`
	SpeedMbps      int    `json:"speed_mbps"`
	CountryCode    string `json:"country_code"`
	CountryName    string `json:"country_name"`
	Region         string `json:"region,omitempty"`
	IsAvailable    bool   `json:"is_available"`
}

// Create handles offering creation (admin only).
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
	if req.Name == "" || req.CountryCode == "" || req.CountryName == "" {
		respondError(w, http.StatusBadRequest, "name, country_code and country_name required")
		return
	}
	if req.DurationMonths <= 0 || req.PriceRubles <= 0 {
		respondError(w, http.StatusBadRequest, "duration and price must be positive")
		return
	}
	if req.SpeedMbps <= 0 {
		req.SpeedMbps = 100
	}

	offering, err := h.repo.Create(r.Context(), req.Name, req.DurationMonths, req.PriceRubles, req.SpeedMbps, req.CountryCode, req.CountryName, req.Region, req.IsAvailable)
	if err != nil {
		log.Printf("create offering error: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to create offering")
		return
	}

	respondJSON(w, http.StatusCreated, offering)
}

// List handles listing offerings.
// For public requests only active offerings are returned.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	onlyActive := r.URL.Query().Get("active") != "false"

	offerings, err := h.repo.List(r.Context(), onlyActive)
	if err != nil {
		log.Printf("list offerings error: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to list offerings")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"offerings": offerings})
}

// Get handles getting a single offering.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "offering id required")
		return
	}

	offering, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrOfferingNotFound) {
			respondError(w, http.StatusNotFound, "offering not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get offering")
		return
	}

	respondJSON(w, http.StatusOK, offering)
}

// Update handles updating an offering (admin only).
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "offering id required")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.CountryCode == "" || req.CountryName == "" {
		respondError(w, http.StatusBadRequest, "name, country_code and country_name required")
		return
	}
	if req.DurationMonths <= 0 || req.PriceRubles <= 0 {
		respondError(w, http.StatusBadRequest, "duration and price must be positive")
		return
	}

	isActive, _ := strconv.ParseBool(r.URL.Query().Get("is_active"))
	if r.URL.Query().Get("is_active") == "" {
		isActive = true
	}

	offering, err := h.repo.Update(r.Context(), id, req.Name, req.DurationMonths, req.PriceRubles, req.SpeedMbps, req.CountryCode, req.CountryName, req.Region, isActive, req.IsAvailable)
	if err != nil {
		if errors.Is(err, ErrOfferingNotFound) {
			respondError(w, http.StatusNotFound, "offering not found")
			return
		}
		log.Printf("update offering error: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to update offering")
		return
	}

	respondJSON(w, http.StatusOK, offering)
}

// Delete handles offering deletion (admin only).
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "offering id required")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrOfferingNotFound) {
			respondError(w, http.StatusNotFound, "offering not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to delete offering")
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
