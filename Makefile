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

test-integration: ## Run backend tests against a disposable, migrated Postgres (port 15432)
	docker rm -f fp-test-pg >/dev/null 2>&1 || true
	docker run --rm -d --name fp-test-pg -e POSTGRES_USER=insurance -e POSTGRES_PASSWORD=insurance \
	  -e POSTGRES_DB=insurance -p 15432:5432 postgres:16-alpine >/dev/null
	until docker exec fp-test-pg pg_isready -U insurance -d insurance >/dev/null 2>&1; do sleep 1; done
	cd backend && go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1 \
	  -path migrations -database "postgres://insurance:insurance@localhost:15432/insurance?sslmode=disable" up
	cd backend && TEST_DATABASE_URL="postgres://insurance:insurance@localhost:15432/insurance?sslmode=disable" \
	  go test ./... -count=1; status=$$?; docker rm -f fp-test-pg >/dev/null; exit $$status

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
