// Package apptest gives integration tests the real wiring over a rolled-back
// transaction: a Store + full service graph against live Postgres
// (TEST_DATABASE_URL), with every write undone at test cleanup so the seeded
// reference data stays pristine.
package apptest

import (
	"os"
	"testing"
	"time"

	"insurance-module/internal/app"
	"insurance-module/internal/domain"
	"insurance-module/internal/platform/filestore"
	"insurance-module/internal/storage/postgres"
	transporthttp "insurance-module/internal/transport/http"
)

// Open connects to the test database and returns a tx-bound store plus the
// service graph built over it. Skips the test when no database is reachable.
func Open(t testing.TB) (*postgres.Store, transporthttp.Services) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://insurance:insurance@localhost:5432/insurance?sslmode=disable" // #nosec G101 -- local-dev test database only
	}
	root, err := postgres.Open(dsn)
	if err != nil {
		t.Skipf("skipping: no test database available (%v)", err)
	}
	store, rollback := root.BeginTx()
	t.Cleanup(rollback)

	files, err := filestore.New(t.TempDir())
	if err != nil {
		t.Fatalf("attachment store: %v", err)
	}

	services := app.Build(store, app.Options{
		JWTSecret: "test-secret-not-used-for-http",
		JWTTTL:    time.Hour,
		Clock:     domain.SystemClock{},
		Files:     files,
	})
	return store, services
}
