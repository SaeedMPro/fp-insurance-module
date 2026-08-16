COMPOSE := docker compose

.PHONY: up down logs seed seed-attachments create-admin test test-integration lint build-frontend build-backend

up: ## Build images (if needed) and start postgres + backend + frontend
	$(COMPOSE) up -d --build

down: ## Stop and remove the stack
	$(COMPOSE) down

logs: ## Follow logs from all services
	$(COMPOSE) logs -f

# The backend applies db/init.sql automatically on boot when the schema is
# missing (see backend/cmd/api). Reference data (db/seed.sql) is manual.

seed: ## Apply backend/db/seed.sql (reference data; run once on empty DB)
	$(COMPOSE) exec -T postgres \
	  psql -U insurance -d insurance -v ON_ERROR_STOP=1 \
	  < backend/db/seed.sql
	$(MAKE) seed-attachments

# The seeded claim_attachments rows describe documents no upload ever created.
# SQL cannot write files, so put a small valid PDF behind each one to make the
# demo's download button work. Real uploads land in the same directory.
seed-attachments: ## Materialise placeholder files for the seeded attachment rows
	@$(COMPOSE) exec -T postgres \
	    psql -U insurance -d insurance -Atc 'SELECT file_path FROM claim_attachments' \
	  | $(COMPOSE) exec -T backend sh -c 'while read -r key; do \
	      [ -n "$$key" ] || continue; \
	      f="$$ATTACHMENTS_DIR/$$key"; \
	      [ -f "$$f" ] && continue; \
	      mkdir -p "$$(dirname "$$f")"; \
	      printf "%%PDF-1.4\n1 0 obj<</Type/Catalog>>endobj\ntrailer<</Root 1 0 R>>\n%%%%EOF\n" > "$$f"; \
	    done'

create-admin: ## Create the sole admin if missing (defaults: admin / Admin123!)
	$(COMPOSE) run --rm --no-deps \
	  -e DATABASE_URL=postgres://insurance:insurance@postgres:5432/insurance?sslmode=disable \
	  -e ADMIN_USERNAME=$${ADMIN_USERNAME:-admin} \
	  -e ADMIN_PASSWORD=$${ADMIN_PASSWORD:-Admin123!} \
	  -e ADMIN_FULL_NAME=$${ADMIN_FULL_NAME:-مدیر سامانه} \
	  backend /app/createadmin

test: ## Run backend Go tests (integration tests skip if no DB is reachable)
	cd backend && go test ./...

TEST_PGDIR  := /tmp/insurance-test-pg
TEST_PGPORT := 15433
TEST_DSN    := postgres://insurance@127.0.0.1:$(TEST_PGPORT)/insurance?sslmode=disable

test-integration: ## Run backend tests against a disposable, initialized Postgres
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
	psql "$(TEST_DSN)" -v ON_ERROR_STOP=1 -f backend/db/init.sql
	psql "$(TEST_DSN)" -v ON_ERROR_STOP=1 -f backend/db/seed.sql
	cd backend && TEST_DATABASE_URL="$(TEST_DSN)" go test ./... -count=1; \
	  status=$$?; pg_ctl -D $(TEST_PGDIR)/data stop >/dev/null 2>&1; exit $$status

lint: ## Run golangci-lint (backend) and oxlint (frontend)
	cd backend && golangci-lint run
	cd frontend && npm run lint

# NOTE: the browser end-to-end suite that used to live in e2e/ was removed in
# the repository restructure, so there is no `e2e` target here any more. The
# behaviour it covered is exercised by the Go integration tests (which run the
# real services against a real database) — see `make test-integration`.

build-frontend: ## Build only the frontend image
	$(COMPOSE) build frontend

build-backend: ## Build only the backend image
	$(COMPOSE) build backend
