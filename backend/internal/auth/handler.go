package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/vpn-project/backend/internal/config"
	"github.com/vpn-project/backend/internal/users"
)

// Handler provides HTTP handlers for authentication.
type Handler struct {
	cfg          *config.Config
	userRepo     *users.Repository
	tokenService *TokenService
}

// NewHandler creates a new auth handler.
func NewHandler(cfg *config.Config, userRepo *users.Repository, tokenService *TokenService) *Handler {
	return &Handler{
		cfg:          cfg,
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

// GoogleLoginRequest represents a Google login request.
type GoogleLoginRequest struct {
	IDToken string `json:"id_token"`
}

// TelegramLoginRequest represents a Telegram login request.
type TelegramLoginRequest struct {
	ID        int64             `json:"id"`
	FirstName string            `json:"first_name"`
	Username  string            `json:"username"`
	Email     string            `json:"email"`
	AuthDate  string            `json:"auth_date"`
	Hash      string            `json:"hash"`
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

	if user.PasswordHash == nil {
		respondError(w, http.StatusUnauthorized, "use oauth or telegram login")
		return
	}

	valid, err := VerifyPassword(req.Password, *user.PasswordHash)
	if err != nil || !valid {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	h.respondTokens(w, user)
}

// GoogleLogin handles Google OAuth login.
func (h *Handler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.cfg.GoogleClientID == "" {
		respondError(w, http.StatusNotImplemented, "google auth not configured")
		return
	}

	var req GoogleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.IDToken == "" {
		respondError(w, http.StatusBadRequest, "id_token required")
		return
	}

	gUser, err := VerifyGoogleIDToken(r.Context(), h.cfg.GoogleClientID, req.IDToken)
	if err != nil {
		log.Printf("google verification failed: %v", err)
		respondError(w, http.StatusUnauthorized, "invalid google token")
		return
	}

	ctx := r.Context()
	user, err := h.findOrCreateOAuthUser(ctx, gUser.Email, &gUser.ID, nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to process login")
		return
	}

	h.respondTokens(w, user)
}

// TelegramLogin handles Telegram Login Widget login.
func (h *Handler) TelegramLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.cfg.TelegramBotToken == "" {
		respondError(w, http.StatusNotImplemented, "telegram auth not configured")
		return
	}

	var req TelegramLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		respondError(w, http.StatusBadRequest, "email required for telegram login")
		return
	}

	data := map[string]string{
		"id":         jsonNumber(req.ID),
		"first_name": req.FirstName,
		"username":   req.Username,
		"auth_date":  req.AuthDate,
	}

	tUser, err := VerifyTelegramAuth(h.cfg.TelegramBotToken, data, req.Hash)
	if err != nil {
		log.Printf("telegram verification failed: %v", err)
		respondError(w, http.StatusUnauthorized, "invalid telegram data")
		return
	}

	ctx := r.Context()
	user, err := h.findOrCreateOAuthUser(ctx, req.Email, nil, &tUser.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to process login")
		return
	}

	h.respondTokens(w, user)
}

func (h *Handler) findOrCreateOAuthUser(ctx context.Context, email string, googleID *string, telegramID *int64) (*users.User, error) {
	// 1. Try to find by google_id or telegram_id.
	if googleID != nil {
		if user, err := h.userRepo.GetByGoogleID(ctx, *googleID); err == nil {
			return user, nil
		}
	}
	if telegramID != nil {
		if user, err := h.userRepo.GetByTelegramID(ctx, *telegramID); err == nil {
			return user, nil
		}
	}

	// 2. Try to find by email and link OAuth IDs.
	user, err := h.userRepo.GetByEmail(ctx, email)
	if err == nil {
		if googleID != nil && user.GoogleID == nil {
			if err := h.userRepo.LinkGoogleID(ctx, user.ID, *googleID); err != nil {
				return nil, err
			}
		}
		if telegramID != nil && user.TelegramID == nil {
			if err := h.userRepo.LinkTelegramID(ctx, user.ID, *telegramID); err != nil {
				return nil, err
			}
		}
		return user, nil
	}
	if !errors.Is(err, users.ErrUserNotFound) {
		return nil, err
	}

	// 3. Create new user.
	return h.userRepo.CreateOAuth(ctx, email, googleID, telegramID, users.RoleUser)
}

func (h *Handler) respondTokens(w http.ResponseWriter, user *users.User) {
	if user.Status == users.StatusBlocked {
		respondError(w, http.StatusForbidden, "account blocked")
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

func jsonNumber(n int64) string {
	return strconv.FormatInt(n, 10)
}
