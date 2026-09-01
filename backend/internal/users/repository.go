package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrUserNotFound is returned when a user is not found.
var ErrUserNotFound = errors.New("user not found")

// Repository provides access to user storage.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new user repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a new user into the database.
func (r *Repository) Create(ctx context.Context, email, passwordHash string, role Role) (*User, error) {
	query := `
		INSERT INTO users (email, password_hash, role, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, role, status, created_at, updated_at
	`

	var user User
	err := r.pool.QueryRow(ctx, query, email, nullableString(passwordHash), role, StatusActive).Scan(
		&user.ID, &user.Email, &user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return &user, nil
}

// CreateOAuth creates a user from OAuth/Telegram login.
func (r *Repository) CreateOAuth(ctx context.Context, email string, googleID *string, telegramID *int64, role Role) (*User, error) {
	query := `
		INSERT INTO users (email, password_hash, google_id, telegram_id, role, status)
		VALUES ($1, NULL, $2, $3, $4, $5)
		RETURNING id, email, google_id, telegram_id, role, status, created_at, updated_at
	`

	var user User
	err := r.pool.QueryRow(ctx, query, email, googleID, telegramID, role, StatusActive).Scan(
		&user.ID, &user.Email, &user.GoogleID, &user.TelegramID,
		&user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert oauth user: %w", err)
	}

	return &user, nil
}

// GetByEmail returns a user by email address.
func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, role, status, totp_secret,
		       google_id, telegram_id, created_at, updated_at
		FROM users
		WHERE email = $1 AND status != $2
	`

	user, err := r.scanUser(ctx, query, email, StatusDeleted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("select user by email: %w", err)
	}

	return user, nil
}

// GetByGoogleID returns a user by Google ID.
func (r *Repository) GetByGoogleID(ctx context.Context, googleID string) (*User, error) {
	query := `
		SELECT id, email, password_hash, role, status, totp_secret,
		       google_id, telegram_id, created_at, updated_at
		FROM users
		WHERE google_id = $1 AND status != $2
	`

	user, err := r.scanUser(ctx, query, googleID, StatusDeleted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("select user by google_id: %w", err)
	}

	return user, nil
}

// GetByTelegramID returns a user by Telegram ID.
func (r *Repository) GetByTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	query := `
		SELECT id, email, password_hash, role, status, totp_secret,
		       google_id, telegram_id, created_at, updated_at
		FROM users
		WHERE telegram_id = $1 AND status != $2
	`

	user, err := r.scanUser(ctx, query, telegramID, StatusDeleted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("select user by telegram_id: %w", err)
	}

	return user, nil
}

// LinkGoogleID attaches a Google ID to an existing user.
func (r *Repository) LinkGoogleID(ctx context.Context, userID, googleID string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE users SET google_id = $2, updated_at = NOW() WHERE id = $1",
		userID, googleID)
	if err != nil {
		return fmt.Errorf("link google id: %w", err)
	}
	return nil
}

// LinkTelegramID attaches a Telegram ID to an existing user.
func (r *Repository) LinkTelegramID(ctx context.Context, userID string, telegramID int64) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE users SET telegram_id = $2, updated_at = NOW() WHERE id = $1",
		userID, telegramID)
	if err != nil {
		return fmt.Errorf("link telegram id: %w", err)
	}
	return nil
}

// Exists checks if any admin user exists.
func (r *Repository) Exists(ctx context.Context, role Role) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE role = $1 AND status != $2)`

	var exists bool
	if err := r.pool.QueryRow(ctx, query, role, StatusDeleted).Scan(&exists); err != nil {
		return false, fmt.Errorf("check exists: %w", err)
	}

	return exists, nil
}

// List returns all non-deleted users ordered by creation time.
func (r *Repository) List(ctx context.Context) ([]User, error) {
	query := `
		SELECT id, email, password_hash, role, status, totp_secret,
		       google_id, telegram_id, created_at, updated_at
		FROM users
		WHERE status != $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, StatusDeleted)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		user, err := r.scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *user)
	}

	return users, rows.Err()
}

// GetByID returns a user by ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, email, password_hash, role, status, totp_secret,
		       google_id, telegram_id, created_at, updated_at
		FROM users
		WHERE id = $1 AND status != $2
	`

	user, err := r.scanUser(ctx, query, id, StatusDeleted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("select user by id: %w", err)
	}
	return user, nil
}

// Update modifies email and role for an existing user.
func (r *Repository) Update(ctx context.Context, id, email string, role Role) (*User, error) {
	query := `
		UPDATE users
		SET email = $2, role = $3, updated_at = NOW()
		WHERE id = $1 AND status != $4
		RETURNING id, email, password_hash, role, status, totp_secret,
		          google_id, telegram_id, created_at, updated_at
	`

	user, err := r.scanUser(ctx, query, id, email, role, StatusDeleted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("update user: %w", err)
	}
	return user, nil
}

// SetStatus changes a user's status (active/blocked/deleted).
func (r *Repository) SetStatus(ctx context.Context, id string, status Status) error {
	result, err := r.pool.Exec(ctx,
		"UPDATE users SET status = $2, updated_at = NOW() WHERE id = $1 AND status != $3",
		id, status, StatusDeleted)
	if err != nil {
		return fmt.Errorf("set user status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// SetPassword updates the password hash for a user.
func (r *Repository) SetPassword(ctx context.Context, id, passwordHash string) error {
	result, err := r.pool.Exec(ctx,
		"UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1 AND status != $3",
		id, passwordHash, StatusDeleted)
	if err != nil {
		return fmt.Errorf("set password: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *Repository) scanUser(ctx context.Context, query string, args ...any) (*User, error) {
	var user User
	var passwordHash, totpSecret *string

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&user.ID, &user.Email, &passwordHash, &user.Role, &user.Status, &totpSecret,
		&user.GoogleID, &user.TelegramID, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	user.PasswordHash = passwordHash
	user.TOTPSecret = totpSecret
	return &user, nil
}

// userScanner abstracts pgx.Row and pgx.Rows scanning.
type userScanner interface {
	Scan(dest ...any) error
}

func (r *Repository) scanUserRow(row userScanner) (*User, error) {
	var user User
	var passwordHash, totpSecret *string

	err := row.Scan(
		&user.ID, &user.Email, &passwordHash, &user.Role, &user.Status, &totpSecret,
		&user.GoogleID, &user.TelegramID, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	user.PasswordHash = passwordHash
	user.TOTPSecret = totpSecret
	return &user, nil
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
