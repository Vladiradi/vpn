package nodes

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNodeNotFound = errors.New("node not found")

// Repository provides access to node storage.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new node repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a new node. Returns the node and the plaintext enroll token.
func (r *Repository) Create(ctx context.Context, name, hostname string, publicIP *string) (*Node, string, error) {
	enrollToken, enrollHash, err := generateTokenAndHash()
	if err != nil {
		return nil, "", fmt.Errorf("generate enroll token: %w", err)
	}

	query := `
		INSERT INTO nodes (name, hostname, ip, status, token_hash, api_token)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, hostname, ip, status, token_hash, api_token, created_at, updated_at
	`

	var node Node
	var rawIP *string
	err = r.pool.QueryRow(ctx, query, name, hostname, publicIP, StatusOffline, enrollHash, enrollHash).Scan(
		&node.ID, &node.Name, &node.Hostname, &rawIP, &node.Status,
		&node.TokenHash, &node.APIToken, &node.CreatedAt, &node.UpdatedAt,
	)
	if err != nil {
		return nil, "", fmt.Errorf("insert node: %w", err)
	}
	node.PublicIP = rawIP

	return &node, enrollToken, nil
}

// GetByID returns a node by ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*Node, error) {
	query := `
		SELECT id, name, hostname, ip, status, token_hash, api_token,
		       reality_dest, reality_private_key, reality_public_key, reality_short_ids, decoy_host,
		       last_seen_at, created_at, updated_at
		FROM nodes
		WHERE id = $1
	`

	node, err := r.scanNode(ctx, query, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNodeNotFound
		}
		return nil, err
	}
	return node, nil
}

// GetByAPITokenHash returns a node by API token hash.
func (r *Repository) GetByAPITokenHash(ctx context.Context, hash string) (*Node, error) {
	query := `
		SELECT id, name, hostname, ip, status, token_hash, api_token,
		       reality_dest, reality_private_key, reality_public_key, reality_short_ids, decoy_host,
		       last_seen_at, created_at, updated_at
		FROM nodes
		WHERE api_token = $1
	`

	node, err := r.scanNode(ctx, query, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNodeNotFound
		}
		return nil, err
	}
	return node, nil
}

// List returns all nodes.
func (r *Repository) List(ctx context.Context) ([]Node, error) {
	query := `
		SELECT id, name, hostname, ip, status, token_hash, api_token,
		       reality_dest, reality_private_key, reality_public_key, reality_short_ids, decoy_host,
		       last_seen_at, created_at, updated_at
		FROM nodes
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		node, err := r.scanNodeRow(rows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, *node)
	}

	return nodes, rows.Err()
}

// Delete removes a node.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result, err := r.pool.Exec(ctx, "DELETE FROM nodes WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete node: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNodeNotFound
	}
	return nil
}

// UpdateConfig updates node XRay/REALITY configuration.
func (r *Repository) UpdateConfig(ctx context.Context, id string, cfg NodeConfigUpdate) error {
	query := `
		UPDATE nodes
		SET reality_dest = $2,
		    reality_private_key = $3,
		    reality_public_key = $4,
		    reality_short_ids = $5,
		    decoy_host = $6,
		    updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id, cfg.RealityDest, cfg.RealityPrivateKey,
		cfg.RealityPublicKey, cfg.RealityShortIDs, cfg.DecoyHost)
	if err != nil {
		return fmt.Errorf("update node config: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNodeNotFound
	}
	return nil
}

// UpdateStatus updates node status and last seen time.
func (r *Repository) UpdateStatus(ctx context.Context, id string, status Status) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE nodes SET status = $2, last_seen_at = NOW(), updated_at = NOW() WHERE id = $1",
		id, status)
	if err != nil {
		return fmt.Errorf("update node status: %w", err)
	}
	return nil
}

// NodeConfigUpdate holds editable node configuration fields.
type NodeConfigUpdate struct {
	RealityDest       *string
	RealityPrivateKey *string
	RealityPublicKey  *string
	RealityShortIDs   []string
	DecoyHost         *string
}

func (r *Repository) scanNode(ctx context.Context, query string, args ...any) (*Node, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	return r.scanNodeRow(row)
}

// nodeScanner abstracts pgx.Row and pgx.Rows scanning.
type nodeScanner interface {
	Scan(dest ...any) error
}

func (r *Repository) scanNodeRow(row nodeScanner) (*Node, error) {
	var node Node
	var rawIP *string

	err := row.Scan(
		&node.ID, &node.Name, &node.Hostname, &rawIP, &node.Status,
		&node.TokenHash, &node.APIToken,
		&node.RealityDest, &node.RealityPrivateKey, &node.RealityPublicKey,
		&node.RealityShortIDs, &node.DecoyHost,
		&node.LastSeenAt, &node.CreatedAt, &node.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	node.PublicIP = rawIP
	return &node, nil
}

func generateTokenAndHash() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(b)
	hash := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(hash[:]), nil
}

// HashToken returns SHA-256 hash of a token.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
