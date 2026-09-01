package offerings

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrOfferingNotFound is returned when an offering is not found.
var ErrOfferingNotFound = errors.New("offering not found")

// Repository provides access to offering storage.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new offering repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a new offering.
func (r *Repository) Create(ctx context.Context, name string, durationMonths, priceRubles, speedMbps int, countryCode, countryName, region string, isAvailable bool) (*Offering, error) {
	query := `
		INSERT INTO offerings (name, duration_months, price_rubles, speed_mbps, country_code, country_name, region, is_available)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, name, duration_months, price_rubles, speed_mbps, country_code, country_name, region, is_active, is_available, created_at, updated_at
	`

	var offering Offering
	var regionPtr *string
	if region != "" {
		regionPtr = &region
	}

	err := r.pool.QueryRow(ctx, query, name, durationMonths, priceRubles, speedMbps, countryCode, countryName, regionPtr, isAvailable).Scan(
		&offering.ID, &offering.Name, &offering.DurationMonths, &offering.PriceRubles, &offering.SpeedMbps,
		&offering.CountryCode, &offering.CountryName, &regionPtr,
		&offering.IsActive, &offering.IsAvailable, &offering.CreatedAt, &offering.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert offering: %w", err)
	}
	offering.Region = regionPtr
	return &offering, nil
}

// GetByID returns an offering by ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*Offering, error) {
	query := `
		SELECT id, name, duration_months, price_rubles, speed_mbps, country_code, country_name, region, is_active, is_available, created_at, updated_at
		FROM offerings
		WHERE id = $1
	`

	offering, err := r.scanOffering(ctx, query, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOfferingNotFound
		}
		return nil, fmt.Errorf("select offering by id: %w", err)
	}
	return offering, nil
}

// GetDefault returns the first active available offering (MVP — Германия).
func (r *Repository) GetDefault(ctx context.Context) (*Offering, error) {
	query := `
		SELECT id, name, duration_months, price_rubles, speed_mbps, country_code, country_name, region, is_active, is_available, created_at, updated_at
		FROM offerings
		WHERE is_active = true AND is_available = true
		ORDER BY created_at
		LIMIT 1
	`

	offering, err := r.scanOffering(ctx, query)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOfferingNotFound
		}
		return nil, fmt.Errorf("select default offering: %w", err)
	}
	return offering, nil
}

// List returns all offerings ordered by country and duration.
func (r *Repository) List(ctx context.Context, onlyActive bool) ([]Offering, error) {
	query := `
		SELECT id, name, duration_months, price_rubles, speed_mbps, country_code, country_name, region, is_active, is_available, created_at, updated_at
		FROM offerings
	`
	var args []any
	if onlyActive {
		query += " WHERE is_active = true"
	}
	query += " ORDER BY country_name, duration_months"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list offerings: %w", err)
	}
	defer rows.Close()

	var offerings []Offering
	for rows.Next() {
		offering, err := r.scanOfferingRow(rows)
		if err != nil {
			return nil, err
		}
		offerings = append(offerings, *offering)
	}

	return offerings, rows.Err()
}

// Update modifies an offering.
func (r *Repository) Update(ctx context.Context, id, name string, durationMonths, priceRubles, speedMbps int, countryCode, countryName, region string, isActive, isAvailable bool) (*Offering, error) {
	query := `
		UPDATE offerings
		SET name = $2, duration_months = $3, price_rubles = $4, speed_mbps = $5,
		    country_code = $6, country_name = $7, region = $8, is_active = $9, is_available = $10, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, duration_months, price_rubles, speed_mbps, country_code, country_name, region, is_active, is_available, created_at, updated_at
	`

	var regionPtr *string
	if region != "" {
		regionPtr = &region
	}

	offering, err := r.scanOffering(ctx, query, id, name, durationMonths, priceRubles, speedMbps, countryCode, countryName, regionPtr, isActive, isAvailable)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOfferingNotFound
		}
		return nil, fmt.Errorf("update offering: %w", err)
	}
	return offering, nil
}

// Delete removes an offering.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result, err := r.pool.Exec(ctx, "DELETE FROM offerings WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete offering: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrOfferingNotFound
	}
	return nil
}

func (r *Repository) scanOffering(ctx context.Context, query string, args ...any) (*Offering, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	return r.scanOfferingRow(row)
}

type offeringScanner interface {
	Scan(dest ...any) error
}

func (r *Repository) scanOfferingRow(row offeringScanner) (*Offering, error) {
	var offering Offering
	var region *string
	err := row.Scan(
		&offering.ID, &offering.Name, &offering.DurationMonths, &offering.PriceRubles, &offering.SpeedMbps,
		&offering.CountryCode, &offering.CountryName, &region,
		&offering.IsActive, &offering.IsAvailable, &offering.CreatedAt, &offering.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	offering.Region = region
	return &offering, nil
}
