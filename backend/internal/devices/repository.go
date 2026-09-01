package devices

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrDeviceNotFound is returned when a device is not found.
var ErrDeviceNotFound = errors.New("device not found")

// Repository provides access to device storage.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new device repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a new device for a user.
func (r *Repository) Create(ctx context.Context, userID, name, protocol string, nodeID *string) (*Device, error) {
	query := `
		INSERT INTO devices (user_id, name, protocol, node_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, name, uuid, protocol, node_id, created_at, revoked_at
	`

	var device Device
	var rawNodeID, rawSubID *string
	err := r.pool.QueryRow(ctx, query, userID, name, protocol, nodeID).Scan(
		&device.ID, &device.UserID, &device.Name, &device.UUID,
		&device.Protocol, &rawNodeID, &device.CreatedAt, &device.RevokedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert device: %w", err)
	}
	device.NodeID = rawNodeID
	device.SubscriptionID = rawSubID
	return &device, nil
}

// GetByID returns a device by ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*Device, error) {
	query := `
		SELECT id, user_id, subscription_id, name, uuid, protocol, node_id, created_at, revoked_at
		FROM devices
		WHERE id = $1 AND revoked_at IS NULL
	`

	device, err := r.scanDevice(ctx, query, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("select device by id: %w", err)
	}
	return device, nil
}

// GetBySubscription returns the active device linked to a subscription.
func (r *Repository) GetBySubscription(ctx context.Context, subscriptionID string) (*Device, error) {
	query := `
		SELECT id, user_id, subscription_id, name, uuid, protocol, node_id, created_at, revoked_at
		FROM devices
		WHERE subscription_id = $1 AND revoked_at IS NULL
		LIMIT 1
	`

	device, err := r.scanDevice(ctx, query, subscriptionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("select device by subscription: %w", err)
	}
	return device, nil
}

// ListByUser returns all active devices for a user.
func (r *Repository) ListByUser(ctx context.Context, userID string) ([]Device, error) {
	query := `
		SELECT id, user_id, subscription_id, name, uuid, protocol, node_id, created_at, revoked_at
		FROM devices
		WHERE user_id = $1 AND revoked_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list devices by user: %w", err)
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		device, err := r.scanDeviceRow(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, *device)
	}

	return devices, rows.Err()
}

// Revoke marks a device as revoked.
func (r *Repository) Revoke(ctx context.Context, id, userID string) error {
	result, err := r.pool.Exec(ctx,
		"UPDATE devices SET revoked_at = NOW() WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL",
		id, userID)
	if err != nil {
		return fmt.Errorf("revoke device: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

// LinkToSubscription attaches a device to a subscription.
func (r *Repository) LinkToSubscription(ctx context.Context, deviceID, subscriptionID string) error {
	result, err := r.pool.Exec(ctx,
		"UPDATE devices SET subscription_id = $2, updated_at = NOW() WHERE id = $1",
		deviceID, subscriptionID)
	if err != nil {
		return fmt.Errorf("link device to subscription: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

// ListByNode returns all active devices assigned to a node.
func (r *Repository) ListByNode(ctx context.Context, nodeID string) ([]Device, error) {
	query := `
		SELECT id, user_id, subscription_id, name, uuid, protocol, node_id, created_at, revoked_at
		FROM devices
		WHERE node_id = $1 AND revoked_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list devices by node: %w", err)
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		device, err := r.scanDeviceRow(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, *device)
	}

	return devices, rows.Err()
}

func (r *Repository) scanDevice(ctx context.Context, query string, args ...any) (*Device, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	return r.scanDeviceRow(row)
}

type deviceScanner interface {
	Scan(dest ...any) error
}

func (r *Repository) scanDeviceRow(row deviceScanner) (*Device, error) {
	var device Device
	var nodeID, subID *string
	err := row.Scan(
		&device.ID, &device.UserID, &subID, &device.Name, &device.UUID,
		&device.Protocol, &nodeID, &device.CreatedAt, &device.RevokedAt,
	)
	if err != nil {
		return nil, err
	}
	device.NodeID = nodeID
	device.SubscriptionID = subID
	return &device, nil
}
