// Package storage provides database and cache connectivity for the application.
package storage

import (
	"context"
	"fmt"

	"github.com/vpn-project/backend/internal/config"
)

// Storage aggregates all persistent storage connections.
type Storage struct {
	Postgres *Postgres
	Redis    *Redis
}

// New initializes all storage backends from configuration.
func New(ctx context.Context, cfg *config.Config) (*Storage, error) {
	pg, err := NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}

	rdb, err := NewRedis(cfg.RedisURL)
	if err != nil {
		pg.Close()
		return nil, fmt.Errorf("redis: %w", err)
	}

	return &Storage{
		Postgres: pg,
		Redis:    rdb,
	}, nil
}

// Close closes all storage connections.
func (s *Storage) Close() {
	if s.Postgres != nil {
		s.Postgres.Close()
	}
	if s.Redis != nil {
		s.Redis.Close()
	}
}

// Ping checks connectivity to all storage backends.
func (s *Storage) Ping(ctx context.Context) error {
	if err := s.Postgres.Ping(ctx); err != nil {
		return fmt.Errorf("postgres ping: %w", err)
	}
	if err := s.Redis.Ping(ctx); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}
