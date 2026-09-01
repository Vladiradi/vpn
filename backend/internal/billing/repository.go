package billing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrSubscriptionNotFound is returned when a subscription is not found.
var ErrSubscriptionNotFound = errors.New("subscription not found")

// Repository provides access to subscription storage.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new subscription repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// generateSubscriptionToken creates a random URL-safe token.
func generateSubscriptionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Create inserts a new pending subscription.
func (r *Repository) Create(ctx context.Context, userID, offeringID string) (*Subscription, error) {
	query := `
		INSERT INTO subscriptions (user_id, offering_id, status)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, offering_id, subscription_token, status, started_at, expires_at, total_traffic_bytes, created_at, updated_at
	`

	var sub Subscription
	err := r.pool.QueryRow(ctx, query, userID, offeringID, StatusPending).Scan(
		&sub.ID, &sub.UserID, &sub.OfferingID, &sub.SubscriptionToken, &sub.Status,
		&sub.StartedAt, &sub.ExpiresAt, &sub.TotalTrafficBytes,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert subscription: %w", err)
	}
	return &sub, nil
}

// GetByID returns a subscription by ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*Subscription, error) {
	query := `
		SELECT id, user_id, offering_id, subscription_token, status, started_at, expires_at, total_traffic_bytes, created_at, updated_at
		FROM subscriptions
		WHERE id = $1
	`

	sub, err := r.scanSubscription(ctx, query, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("select subscription by id: %w", err)
	}
	return sub, nil
}

// GetByToken returns a subscription by public subscription token.
func (r *Repository) GetByToken(ctx context.Context, token string) (*Subscription, error) {
	query := `
		SELECT id, user_id, offering_id, subscription_token, status, started_at, expires_at, total_traffic_bytes, created_at, updated_at
		FROM subscriptions
		WHERE subscription_token = $1
	`

	sub, err := r.scanSubscription(ctx, query, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("select subscription by token: %w", err)
	}
	return sub, nil
}

// GetActiveByUser returns the active subscription for a user, if any.
func (r *Repository) GetActiveByUser(ctx context.Context, userID string) (*Subscription, error) {
	query := `
		SELECT id, user_id, offering_id, subscription_token, status, started_at, expires_at, total_traffic_bytes, created_at, updated_at
		FROM subscriptions
		WHERE user_id = $1 AND status = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	sub, err := r.scanSubscription(ctx, query, userID, StatusActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("select active subscription: %w", err)
	}
	return sub, nil
}

// List returns all subscriptions for a user, or all subscriptions if userID is empty.
func (r *Repository) List(ctx context.Context, userID string) ([]Subscription, error) {
	query := `
		SELECT id, user_id, offering_id, subscription_token, status, started_at, expires_at, total_traffic_bytes, created_at, updated_at
		FROM subscriptions
	`
	var args []any
	if userID != "" {
		query += " WHERE user_id = $1"
		args = append(args, userID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		sub, err := r.scanSubscriptionRow(rows)
		if err != nil {
			return nil, err
		}
		subs = append(subs, *sub)
	}

	return subs, rows.Err()
}

// Activate marks a subscription as active, generates a public subscription token, and sets started_at/expires_at.
func (r *Repository) Activate(ctx context.Context, id string, durationMonths int) (*Subscription, error) {
	startedAt := time.Now().UTC()
	expiresAt := startedAt.AddDate(0, durationMonths, 0)
	token, err := generateSubscriptionToken()
	if err != nil {
		return nil, fmt.Errorf("generate subscription token: %w", err)
	}

	query := `
		UPDATE subscriptions
		SET status = $2, subscription_token = $3, started_at = $4, expires_at = $5, updated_at = NOW()
		WHERE id = $1 AND status = $6
		RETURNING id, user_id, offering_id, subscription_token, status, started_at, expires_at, total_traffic_bytes, created_at, updated_at
	`

	sub, err := r.scanSubscription(ctx, query, id, StatusActive, token, startedAt, expiresAt, StatusPending)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("activate subscription: %w", err)
	}
	return sub, nil
}

// SetStatus changes a subscription status.
func (r *Repository) SetStatus(ctx context.Context, id, status string) error {
	result, err := r.pool.Exec(ctx,
		"UPDATE subscriptions SET status = $2, updated_at = NOW() WHERE id = $1",
		id, status)
	if err != nil {
		return fmt.Errorf("set subscription status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}

// AddTraffic increments the total traffic counter.
func (r *Repository) AddTraffic(ctx context.Context, id string, bytes int64) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE subscriptions SET total_traffic_bytes = total_traffic_bytes + $2, updated_at = NOW() WHERE id = $1",
		id, bytes)
	return err
}

func (r *Repository) scanSubscription(ctx context.Context, query string, args ...any) (*Subscription, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	return r.scanSubscriptionRow(row)
}

type subscriptionScanner interface {
	Scan(dest ...any) error
}

func (r *Repository) scanSubscriptionRow(row subscriptionScanner) (*Subscription, error) {
	var sub Subscription
	err := row.Scan(
		&sub.ID, &sub.UserID, &sub.OfferingID, &sub.SubscriptionToken, &sub.Status,
		&sub.StartedAt, &sub.ExpiresAt, &sub.TotalTrafficBytes,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}
