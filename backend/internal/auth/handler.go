package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/vpn-project/backend/internal/users"
)

// Handler provides HTTP handlers for authentication.
type Handler struct {
	userRepo     *users.Repository
	tokenService *TokenService
}

// NewHandler creates a new auth handler.
func NewHandler(userRepo *users.Repository, tokenService *TokenService) *Handler {
	return &Handler{
		userRepo:     userRepo,
		tokenService: tokenService,
	}
}

// RegisterRequest represents a registration request.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register handles admin registration. Only the first admin can register this way.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "email and password required")
		return
	}

	if len(req.Password) < 8 {
		respondError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	ctx := r.Context()

	adminExists, err := h.userRepo.Exists(ctx, users.RoleAdmin)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if adminExists {
		respondError(w, http.StatusForbidden, "admin already exists")
		return
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user, err := h.userRepo.Create(ctx, req.Email, hash, users.RoleAdmin)
	if err != nil {
		respondError(w, http.StatusConflict, "email already registered")
		return
	}

	tokens, err := h.tokenService.GenerateTokenPair(user.ID, user.Email, string(user.Role))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate tokens")
		return
	}

	respondJSON(w, http.StatusCreated, tokens)
}

// Login handles user login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "email and password required")
		return
	}

	ctx := r.Context()

	user, err := h.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			respondError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if user.Status == users.StatusBlocked {
		respondError(w, http.StatusForbidden, "account blocked")
		return
	}

	valid, err := VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !valid {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	tokens, err := h.tokenService.GenerateTokenPair(user.ID, user.Email, string(user.Role))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate tokens")
		return
	}

	respondJSON(w, http.StatusOK, tokens)
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
