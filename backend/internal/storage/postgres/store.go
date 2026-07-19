package postgres

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"insurance-module/internal/domain"
)

// Store implements every repository interface the service layer declares.
// A Store is either bound to the root connection pool or, inside Atomic, to a
// single transaction — the methods are identical either way.
type Store struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Store { return &Store{db: db} }

// Atomic runs fn inside one database transaction; the *Store handed to fn is
// bound to that transaction. Rolling back on error preserves the system's
// core invariant: an audit row commits atomically with the change it records.
func (s *Store) Atomic(ctx context.Context, fn func(*Store) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&Store{db: tx})
	})
}

func (s *Store) ctx(ctx context.Context) *gorm.DB { return s.db.WithContext(ctx) }

// notFound converts gorm's sentinel into the domain taxonomy so services and
// transport never import gorm to classify an error.
func notFound(err error, what string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.NotFoundf("%s not found", what)
	}
	return err
}
