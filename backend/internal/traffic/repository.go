package traffic

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository provides access to traffic statistics storage.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new traffic repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Save inserts a traffic stat record.
func (r *Repository) Save(ctx context.Context, deviceID, nodeID string, uplink, downlink int64) error {
	query := `
		INSERT INTO traffic_stats (device_id, node_id, uplink, downlink)
		VALUES ($1, $2, $3, $4)
	`
	if _, err := r.pool.Exec(ctx, query, deviceID, nodeID, uplink, downlink); err != nil {
		return fmt.Errorf("insert traffic stat: %w", err)
	}
	return nil
}

// ListByNode returns traffic stats for a node, newest first.
func (r *Repository) ListByNode(ctx context.Context, nodeID string, limit int) ([]Stat, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
		SELECT id, device_id, node_id, ts, uplink, downlink
		FROM traffic_stats
		WHERE node_id = $1
		ORDER BY ts DESC
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, query, nodeID, limit)
	if err != nil {
		return nil, fmt.Errorf("list traffic by node: %w", err)
	}
	defer rows.Close()

	var stats []Stat
	for rows.Next() {
		var stat Stat
		if err := rows.Scan(&stat.ID, &stat.DeviceID, &stat.NodeID, &stat.Ts, &stat.Uplink, &stat.Downlink); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}
	return stats, rows.Err()
}

// ListByDevice returns traffic stats for a device, newest first.
func (r *Repository) ListByDevice(ctx context.Context, deviceID string, limit int) ([]Stat, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
		SELECT id, device_id, node_id, ts, uplink, downlink
		FROM traffic_stats
		WHERE device_id = $1
		ORDER BY ts DESC
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, query, deviceID, limit)
	if err != nil {
		return nil, fmt.Errorf("list traffic by device: %w", err)
	}
	defer rows.Close()

	var stats []Stat
	for rows.Next() {
		var stat Stat
		if err := rows.Scan(&stat.ID, &stat.DeviceID, &stat.NodeID, &stat.Ts, &stat.Uplink, &stat.Downlink); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}
	return stats, rows.Err()
}

// TotalsByDevice returns the sum of uplink/downlink for a device.
func (r *Repository) TotalsByDevice(ctx context.Context, deviceID string) (uplink, downlink int64, err error) {
	query := `
		SELECT COALESCE(SUM(uplink), 0), COALESCE(SUM(downlink), 0)
		FROM traffic_stats
		WHERE device_id = $1
	`
	if err := r.pool.QueryRow(ctx, query, deviceID).Scan(&uplink, &downlink); err != nil {
		return 0, 0, fmt.Errorf("totals by device: %w", err)
	}
	return uplink, downlink, nil
}

// ListAll returns the most recent traffic stats across all nodes.
func (r *Repository) ListAll(ctx context.Context, limit int) ([]Stat, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
		SELECT id, device_id, node_id, ts, uplink, downlink
		FROM traffic_stats
		ORDER BY ts DESC
		LIMIT $1
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list all traffic stats: %w", err)
	}
	defer rows.Close()

	var stats []Stat
	for rows.Next() {
		var stat Stat
		if err := rows.Scan(&stat.ID, &stat.DeviceID, &stat.NodeID, &stat.Ts, &stat.Uplink, &stat.Downlink); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}
	return stats, rows.Err()
}

// TotalsByNode returns the sum of uplink/downlink for a node.
func (r *Repository) TotalsByNode(ctx context.Context, nodeID string) (uplink, downlink int64, err error) {
	query := `
		SELECT COALESCE(SUM(uplink), 0), COALESCE(SUM(downlink), 0)
		FROM traffic_stats
		WHERE node_id = $1
	`
	if err := r.pool.QueryRow(ctx, query, nodeID).Scan(&uplink, &downlink); err != nil {
		return 0, 0, fmt.Errorf("totals by node: %w", err)
	}
	return uplink, downlink, nil
}
