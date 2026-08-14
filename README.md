# Supplementary Insurance Module

A web-based module for managing employee supplementary (top-up) health insurance:
config-driven coverage rules, a multi-stage claim workflow, a full audit trail,
role-based access control, management reporting, and a REST API for integration
with a parent HR system.

Bachelor's capstone project, Bu-Ali Sina University, Faculty of Engineering
(Software Engineering). Persian title: طراحی و پیاده‌سازی ماژول بیمه تکمیلی.

The design goal, straight from the proposal: every service that an employee receives
is recorded accurately and traceably, the payable amount and remaining ceiling are
computed automatically, and — critically — welfare policy (coverage percentages, caps,
eligibility) can be changed purely through configuration, with **no code change and no
redeployment**.

---

## What it does

- **Config-driven rule engine** — coverage percentage, per-claim cap, annual cap,
  waiting period, and eligible relations are stored per (plan, service type) as
  **versioned** rows in `coverage_rules`. Changing a benefit means inserting a new
  version through the API; the engine never hard-codes a service type, percentage, or
  cap. Approving a claim automatically prices it (payable amount + remaining ceiling)
  against the active rule.
- **Multi-stage claim workflow** — `draft → submitted → under_review →
  (approved | rejected | returned_for_docs) → paid → closed`, with reject/return
  requiring a reason, and an explicit state table that rejects illegal transitions.
- **Audit trail** — every state-changing action (submit, review, approve, reject,
  payment, config change, login) is recorded with actor, timestamp, and a before/after
  snapshot, queryable per entity or across the system.
- **RBAC** — four roles (admin, reviewer, employee, auditor) with JWT auth; a separate
  API-key scheme for the parent-system integration.
- **Reporting** — spend per employee, per service type, per month, plus a dashboard
  summary.
- **Parent-system integration** — API-key-authenticated employee master-data sync and
  claim-status lookup (the seam a real HR system would call).

## Tech stack

| Layer     | Technology |
|-----------|------------|
| Backend   | Go 1.26, chi router, GORM (query only), golang-jwt, bcrypt |
| Database  | PostgreSQL 16, schema in `backend/db/init.sql`, reference data in `backend/db/seed.sql` |
| Frontend  | React 19 + TypeScript, Vite, Tailwind CSS, React Router, Recharts, axios; **Persian (Farsi) RTL UI** with the Vazirmatn font |
| Deploy    | Docker + Docker Compose (postgres + backend + nginx-served frontend) |

## Repository layout

```
backend/
  cmd/api/            HTTP server entrypoint (applies db/init.sql, then serves)
  db/
    init.sql          schema (source of truth)
    seed.sql          contracts, plans, service types, coverage rules
  internal/           layered Go packages (domain, service, storage, transport)
frontend/             React + TypeScript SPA (pages per role)
deploy/               docker-compose.yml + env example
docs/                 API contract, architecture, ERD, use cases
Makefile              up / down / logs / seed / test / build-* targets
```

## Running it

Requires Docker and Docker Compose.

```bash
cp deploy/.env.example deploy/.env    # optional; sensible defaults exist
make up                               # build images, start postgres + backend + frontend
make seed                             # load reference data from db/seed.sql (run once)
```

Then open:

- Frontend: http://localhost:5173
- API:      http://localhost:8080/api/v1  (health: http://localhost:8080/healthz)

The backend applies `db/init.sql` automatically on boot when the schema is missing.
Reference data lives in `db/seed.sql` and is applied manually with `make seed`
(not on boot). No demo users are seeded — create accounts yourself after seed.
`make down` stops the stack; `make logs` follows logs.

> **Port conflicts.** The compose file uses host ports `5173` (frontend), `8080`
> (backend), and `5432` (postgres) by default. If any are already in use, the
> ready-made overlay [deploy/docker-compose.altports.yml](deploy/docker-compose.altports.yml)
> remaps to `15173`/`18080` (and unexposes postgres):
>
> ```bash
> VITE_API_BASE_URL=http://localhost:18080/api/v1 CORS_ORIGIN=http://localhost:15173 \
>   docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.altports.yml up -d --build
> ```

## Development (without Docker)

```bash
# Postgres (any local instance); point the app at it:
export DATABASE_URL="postgres://insurance:insurance@localhost:5432/insurance?sslmode=disable"

# Backend (applies db/init.sql on boot when schema is missing):
cd backend && go run ./cmd/api

# Reference data (once, against the same database):
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f db/seed.sql

# Frontend:
cd frontend && npm install && npm run dev   # http://localhost:5173
```

## Tests

```bash
make test          # backend unit + integration tests (or: cd backend && go test ./...)
make e2e           # browser end-to-end suite against the running stack
```

The `ruleengine` package has pure unit tests for the pricing math across all five
seeded service types plus cap/edge cases, and integration tests (against a real
Postgres) for eligibility, waiting periods, and annual-cap accumulation. The
`workflow` package has integration tests covering the happy path, reject, return-for-
docs/resubmit, illegal-transition rejection, and RBAC. Integration tests skip
automatically if no database is reachable.

The **end-to-end suite** (`e2e/e2e.mjs`, puppeteer-core + headless Chrome) drives the
real Persian UI through the complete lifecycle and asserts outcomes through the API:

1. employee creates and submits a claim → reviewer starts review, approves (auto-priced
   by the rule engine), records payment, closes;
2. the reject path with a mandatory Persian reason;
3. an admin publishes a new coverage-rule version **through the UI** and the next claim
   is priced with the new percentage — the config-not-code acceptance criterion,
   proven end to end;
4. RBAC denials (employee vs admin-only routes, anonymous requests);
5. the audit trail (lifecycle actions + `config_change`) and the reports endpoints.

It needs the stack up and reference data loaded (`make up && make seed`), plus
login accounts you create yourself (demo users are no longer seeded), and a Chrome
binary (`CHROME_PATH`, default `/usr/bin/google-chrome-stable`); `E2E_BASE_URL`/`E2E_API_URL`
override the targets when using the alternate-ports overlay. Failure screenshots
land in `e2e/artifacts/`.

## Documentation

- [docs/API-CONTRACT.md](docs/API-CONTRACT.md) — every endpoint, request/response shape, auth, error codes
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — layered design, RBAC, how the engines compose, deployment topology
- [docs/ERD.md](docs/ERD.md) — entity-relationship diagram and table descriptions
- [docs/USE-CASES.md](docs/USE-CASES.md) — actor-by-actor use cases with flows and error paths

## Mapping to the proposal's acceptance criteria

| Acceptance criterion (from the proposal) | Where |
|---|---|
| Correct payable amount & remaining ceiling for ≥ 5 service types, tested | `ruleengine` + its tests; five seeded service types |
| Change a coverage rule purely via configuration, no code change | `POST /coverage-rules`; verified live and in the browser |
| Full workflow incl. approve / reject / return paths | `workflow` package + claim action endpoints |
| Complete audit log with per-claim history replay | `audit` package; `GET /claims/{id}/history`, `GET /audit-logs` |
| Correct role-based access restrictions | `api/middleware` RBAC; verified per-role in the browser tests |
| Reporting (per employee, per service type, per time range) | `reports` package + `/reports/*` endpoints |
| REST API for connecting to the parent system | `/integration/*` (API-key auth) |
| ERD, use-case analysis, technical & user docs | `docs/` |

## Scope

**In scope:** the standalone supplementary-insurance module — rule engine, workflow,
RBAC, reporting, and the integration API — as specified in the proposal.

**Out of scope (per the proposal):** a real payment gateway (disbursement is
simulated and recorded as a `payments` row), a live connection to real HR/payroll/
finance systems (the integration API is the defined seam, exercised with sample data),
and other welfare modules (loans, guesthouse, etc.) that the proposal positions as
future siblings of this module.

## Assumptions

- The user interface is entirely in **Persian (Farsi) and laid out right-to-left**,
  matching the proposal. Amounts render with Persian digits and dates in the Jalali
  calendar (`Intl` `fa-IR`); date/number entry uses native inputs (Gregorian) and is
  converted for storage. Backend data, code identifiers, API field names, and enum
  values stay in English; only the presentation layer is Persian.
- Monetary amounts are in Iranian Rial, stored as `NUMERIC(14,2)` and shown with
  thousands separators (no currency symbol).
- The annual cap resets on the rule's `effective_from` anniversary (a rolling contract
  year), not the calendar year.
- Annual-cap usage counts claims that reached a payable state
  (`approved`/`payment_calculated`/`paid`/`closed`); drafts and rejections do not
  consume cap.
- Interactive users authenticate with JWT; the parent system uses a static API key
  whose SHA-256 hash is stored (never the raw key).
- Reference seed data in `db/seed.sql` (one contract, Standard + Premium plans,
  five service types with rule versions) is applied manually with `make seed`.
  It does not include demo users or claims; create those yourself.
