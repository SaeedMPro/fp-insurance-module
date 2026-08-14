package postgres

import (
	"fmt"

	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open connects to PostgreSQL and returns a Store bound to the connection
// pool. Schema init is NOT run here — that is platform/database.InitSchema,
// invoked by the entrypoints before opening the store.
func Open(databaseURL string) (*Store, error) {
	gdb, err := gorm.Open(gormpg.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	return New(gdb), nil
}

// BeginTx starts a transaction and returns a Store bound to it plus a
// rollback func. Integration tests use this to run against real Postgres and
// leave no trace; production code uses Atomic instead.
func (s *Store) BeginTx() (*Store, func()) {
	tx := s.db.Begin()
	return &Store{db: tx}, func() { tx.Rollback() }
}
