// Package database applies the schema SQL file. Connections are opened by
// storage/postgres.Open — schema init runs first, against the same URL.
// Reference data is applied manually via Makefile (psql + db/seed.sql).
// GORM never auto-migrates anything.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/lib/pq"
)

// InitSchema runs init.sql once when the claims table is missing.
func InitSchema(ctx context.Context, databaseURL, initPath string) error {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "close db: %v\n", err)
		}
	}()

	exists, err := tableExists(ctx, db, "claims")
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return execFile(ctx, db, initPath)
}

func tableExists(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)`, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check table %s: %w", name, err)
	}
	return exists, nil
}

func execFile(ctx context.Context, db *sql.DB, path string) error {
	clean, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("resolve path %s: %w", path, err)
	}
	if !strings.HasSuffix(strings.ToLower(clean), ".sql") {
		return fmt.Errorf("schema file must be a .sql path, got %s", path)
	}

	// Path comes from operator config (DB_INIT_PATH), not request input.
	body, err := os.ReadFile(clean) // #nosec G304 -- trusted config path, validated .sql suffix
	if err != nil {
		return fmt.Errorf("read %s: %w", clean, err)
	}
	if _, err := db.ExecContext(ctx, string(body)); err != nil {
		return fmt.Errorf("exec %s: %w", clean, err)
	}
	return nil
}
