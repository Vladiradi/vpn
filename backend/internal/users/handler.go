package users

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

// PasswordHasher hashes plaintext passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
}

// Handler provides admin HTTP handlers for user management.
type Handler struct {
	repo   *Repository
	hasher PasswordHasher
}

// NewHandler creates a new user handler.
func NewHandler(repo *Repository, hasher PasswordHasher) *Handler {
	return &Handler{repo: repo, hasher: hasher}
}

// CreateRequest represents a request to create a user.
type CreateRequest struct {
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
	Role     Role   `json:"role"`
}

// UpdateRequest represents a request to update a user.
type UpdateRequest struct {
	Email string `json:"email"`
	Role  Role   `json:"role"`
}

// Create handles user creation (admin only).
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

	if req.Email == "" {
		respondError(w, http.StatusBadRequest, "email required")
		return
	}
	if req.Role != RoleAdmin && req.Role != RoleUser {
		respondError(w, http.StatusBadRequest, "invalid role")
		return
	}

	ctx := r.Context()

	var user *User
	var err error
	if req.Password != "" {
		if len(req.Password) < 8 {
			respondError(w, http.StatusBadRequest, "password must be at least 8 characters")
			return
		}
		if h.hasher == nil {
			respondError(w, http.StatusInternalServerError, "password hasher not configured")
			return
		}
		hash, perr := h.hasher.Hash(req.Password)
		if perr != nil {
			respondError(w, http.StatusInternalServerError, "failed to hash password")
			return
		}
		user, err = h.repo.Create(ctx, req.Email, hash, req.Role)
	} else {
		user, err = h.repo.CreateOAuth(ctx, req.Email, nil, nil, req.Role)
	}
	if err != nil {
		log.Printf("create user error: %v", err)
		respondError(w, http.StatusConflict, "email already registered")
		return
	}

	respondJSON(w, http.StatusCreated, user)
}

// List handles listing all users (admin only).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	users, err := h.repo.List(r.Context())
	if err != nil {
		log.Printf("list users error: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"users": users})
}

// Get handles getting a single user (admin only).
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "user id required")
		return
	}

	user, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// Update handles updating a user (admin only).
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "user id required")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" {
		respondError(w, http.StatusBadRequest, "email required")
		return
	}
	if req.Role != RoleAdmin && req.Role != RoleUser {
		respondError(w, http.StatusBadRequest, "invalid role")
		return
	}

	user, err := h.repo.Update(r.Context(), id, req.Email, req.Role)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		log.Printf("update user error: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// SetStatus handles blocking or activating a user (admin only).
func (h *Handler) SetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "user id required")
		return
	}

	var req struct {
		Status Status `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Status != StatusActive && req.Status != StatusBlocked {
		respondError(w, http.StatusBadRequest, "invalid status")
		return
	}

	if err := h.repo.SetStatus(r.Context(), id, req.Status); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to set status")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": string(req.Status)})
}

// Delete soft-deletes a user (admin only).
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "user id required")
		return
	}

	if err := h.repo.SetStatus(r.Context(), id, StatusDeleted); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to delete user")
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
