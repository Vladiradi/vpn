package storage

import (
	"embed"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed all:migrations
var migrationsFS embed.FS

// MigrateUp runs all pending database migrations.
func MigrateUp(dsn string) error {
	driver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	// golang-migrate pgx driver expects pgx:// scheme.
	migrationDSN := toPgxDSN(dsn)

	m, err := migrate.NewWithSourceInstance("iofs", driver, migrationDSN)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}

	return nil
}

func toPgxDSN(dsn string) string {
	if strings.HasPrefix(dsn, "postgres://") {
		return "pgx://" + strings.TrimPrefix(dsn, "postgres://")
	}
	return dsn
}
