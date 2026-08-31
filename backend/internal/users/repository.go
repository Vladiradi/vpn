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
	err := r.pool.QueryRow(ctx, query, email, passwordHash, role, StatusActive).Scan(
		&user.ID, &user.Email, &user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return &user, nil
}

// GetByEmail returns a user by email address.
func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, role, status, totp_secret, created_at, updated_at
		FROM users
		WHERE email = $1 AND status != $2
	`

	var user User
	var totpSecret *string
	err := r.pool.QueryRow(ctx, query, email, StatusDeleted).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.Status,
		&totpSecret, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("select user: %w", err)
	}

	user.TOTPSecret = totpSecret
	return &user, nil
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
