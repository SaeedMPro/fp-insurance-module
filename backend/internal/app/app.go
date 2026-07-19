// Package app is the composition root: it wires the postgres Store into the
// service layer (including each service's transaction adapter) so that
// entrypoints (cmd/api, cmd/seed) and integration tests build the exact same
// object graph from a single place.
package app

import (
	"context"
	"time"

	"insurance-module/internal/domain"
	"insurance-module/internal/service/audit"
	"insurance-module/internal/service/claims"
	"insurance-module/internal/service/coverage"
	"insurance-module/internal/service/employees"
	"insurance-module/internal/service/integration"
	"insurance-module/internal/service/reports"
	"insurance-module/internal/service/users"
	"insurance-module/internal/storage/postgres"
	transporthttp "insurance-module/internal/transport/http"
)

type Options struct {
	JWTSecret string
	JWTTTL    time.Duration
	Clock     domain.Clock // nil → system clock
}

// Build wires every service over the given store.
func Build(store *postgres.Store, opts Options) transporthttp.Services {
	clock := opts.Clock
	if clock == nil {
		clock = domain.SystemClock{}
	}

	coverageSvc := coverage.NewService(store,
		func(ctx context.Context, fn func(coverage.Repo) error) error {
			return store.Atomic(ctx, func(tx *postgres.Store) error { return fn(tx) })
		}, clock)

	claimsSvc := claims.NewService(store,
		func(ctx context.Context, fn func(claims.Repo) error) error {
			return store.Atomic(ctx, func(tx *postgres.Store) error { return fn(tx) })
		}, coverageSvc, clock)

	integrationSvc := integration.NewService(store,
		func(ctx context.Context, fn func(integration.Repo) error) error {
			return store.Atomic(ctx, func(tx *postgres.Store) error { return fn(tx) })
		})

	return transporthttp.Services{
		Users:       users.NewService(store, opts.JWTSecret, opts.JWTTTL, clock),
		Claims:      claimsSvc,
		Coverage:    coverageSvc,
		Employees:   employees.NewService(store, coverageSvc, clock),
		Audit:       audit.NewService(store),
		Reports:     reports.NewService(store),
		Integration: integrationSvc,
	}
}
