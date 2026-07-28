COMPOSE := docker compose -f deploy/docker-compose.yml

.PHONY: up down logs seed test test-integration lint e2e build-frontend build-backend

up: ## Build images (if needed) and start postgres + backend + frontend
	$(COMPOSE) up -d --build

down: ## Stop and remove the stack
	$(COMPOSE) down

logs: ## Follow logs from all services
	$(COMPOSE) logs -f

# The backend applies its own DB migrations automatically on boot
# (see backend/cmd/api/main.go -> db.Migrate), so there is no separate
# migrate target here.

seed: ## Run the demo-data seeder against the compose postgres
	# cmd/seed is baked into the backend image alongside cmd/api (see
	# backend/Dockerfile), so we just exec it in the already-running
	# backend container -- no separate build or extra container needed.
	$(COMPOSE) exec backend /app/seed

test: ## Run backend Go tests (integration tests skip if no DB is reachable)
	cd backend && go test ./...

TEST_PGDIR  := /tmp/insurance-test-pg
TEST_PGPORT := 15433
TEST_DSN    := postgres://insurance@127.0.0.1:$(TEST_PGPORT)/insurance?sslmode=disable

test-integration: ## Run backend tests against a disposable, migrated Postgres
	# Uses a throwaway host-native cluster (initdb/pg_ctl, no root, no Docker):
	# works even where Docker port publishing is unavailable. CI uses its own
	# postgres service container and just sets TEST_DATABASE_URL instead.
	@pg_ctl -D $(TEST_PGDIR)/data stop >/dev/null 2>&1 || true
	rm -rf $(TEST_PGDIR) && mkdir -p $(TEST_PGDIR)/data $(TEST_PGDIR)/sock
	initdb -D $(TEST_PGDIR)/data -U insurance --auth=trust >/dev/null
	# -k keeps the unix socket inside the throwaway dir: the system default
	# (/run/postgresql) is not writable for a non-root cluster.
	pg_ctl -D $(TEST_PGDIR)/data -l $(TEST_PGDIR)/log \
	  -o "-p $(TEST_PGPORT) -k $(TEST_PGDIR)/sock -c listen_addresses=127.0.0.1" start >/dev/null
	@until pg_isready -h 127.0.0.1 -p $(TEST_PGPORT) -U insurance >/dev/null 2>&1; do sleep 1; done
	createdb -h 127.0.0.1 -p $(TEST_PGPORT) -U insurance insurance
	cd backend && go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1 \
	  -path migrations -database "$(TEST_DSN)" up
	cd backend && TEST_DATABASE_URL="$(TEST_DSN)" go test ./... -count=1; \
	  status=$$?; pg_ctl -D $(TEST_PGDIR)/data stop >/dev/null 2>&1; exit $$status

lint: ## Run golangci-lint (backend) and oxlint (frontend)
	cd backend && golangci-lint run
	cd frontend && npm run lint

e2e: ## Run the browser end-to-end suite against a running stack (make up + make seed first)
	# Drives the real Persian UI headlessly through the full claim lifecycle,
	# a config-driven rule change, RBAC checks, and the audit trail.
	# Override E2E_BASE_URL / E2E_API_URL / CHROME_PATH if not on the defaults.
	cd e2e && npm install --silent && node e2e.mjs

build-frontend: ## Build only the frontend image
	$(COMPOSE) build frontend

build-backend: ## Build only the backend image
	$(COMPOSE) build backend
