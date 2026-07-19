// Package database owns schema migrations. Connections are opened by
// storage/postgres.Open — migrations run first, against the same URL, via
// golang-migrate: the SQL files in backend/migrations are the single source
// of truth for schema, and GORM never auto-migrates anything.
package database

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Migrate applies all pending up-migrations found under migrationsPath
// (e.g. "file://migrations") to databaseURL.
func Migrate(databaseURL, migrationsPath string) error {
	m, err := migrate.New(migrationsPath, databaseURL)
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
