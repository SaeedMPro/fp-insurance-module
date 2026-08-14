// Package database applies the schema SQL file. Connections are opened by
// storage/postgres.Open — schema init runs first, against the same URL.
// Reference data is applied manually via Makefile (psql + db/seed.sql).
// GORM never auto-migrates anything.
package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

// InitSchema runs init.sql once when the claims table is missing.
func InitSchema(databaseURL, initPath string) error {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	exists, err := tableExists(db, "claims")
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return execFile(db, initPath)
}

func tableExists(db *sql.DB, name string) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)`, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check table %s: %w", name, err)
	}
	return exists, nil
}

func execFile(db *sql.DB, path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if _, err := db.Exec(string(body)); err != nil {
		return fmt.Errorf("exec %s: %w", path, err)
	}
	return nil
}
