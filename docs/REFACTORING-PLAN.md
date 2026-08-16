# Refactoring Plan — Target Architecture for Long-Term Maintenance

Status: **proposed** (nothing in this document is implemented yet).
Scope: whole project — Go backend, React frontend, contract, tooling, ops.

---

## 1. Context — why refactor, and why now

The project is feature-complete and behaviourally solid: the rule engine and claim
workflow have unit + integration tests, and a 13-step browser E2E suite exercises the
full lifecycle, RBAC, the audit trail, and the config-driven rule-change flow. **That
safety net is exactly what makes a structural refactor cheap and safe now** — every
phase below ends with `make test && make e2e` green.

The code, however, grew feature-first, and the seams that don't matter at 3.5k+3.9k
LOC will hurt at 2–3× that size. Measured, current-state facts (verified 2026-07-19):

| Problem | Evidence |
|---|---|
| Business logic lives in HTTP handlers | 29 direct `h.d.DB` GORM call sites across 8 files in `backend/internal/api/`; two full transactions written inline in handlers (`reference_handlers.go:143` rule-version publish, `integration_handlers.go:43` employee sync) |
| Wire format is welded to the persistence model | every struct in `backend/internal/models/models.go` (231 lines) carries `gorm:` **and** `json:` tags; nearly every handler serialises `models.*` directly, so renaming a DB column silently changes the public API |
| Authorization policy in three places | route-role gates in `api/middleware`, ownership checks in `api/handlers.go` (`authorizeEmployeeAccess`, `authorizeClaimAccess`), actor-role checks again inside `workflow.go` (`requireReviewer`, owner checks in `Submit`) |
| Money as `float64` | 25 sites in `models.go` + `ruleengine.go` alone; ad-hoc `round2()`; financial software should not do binary floating-point arithmetic |
| Wall-clock coupling | `time.Now()` at 5 sites in `workflow.go` and in `ruleengine.contractYearWindow` — untestable time logic, and the annual-cap window silently uses server-local time instead of Iran's calendar day |
| Hand-maintained duplicate API types | `frontend/src/api/types.ts` (182 lines) mirrors `models.go` by hand; drift is only caught by the E2E |
| Frontend duplication | identical `<table>` scaffolding in 7 pages; identical loading/error/`useState` boilerplate in 14 files; identical input class strings |
| One 728 KB JS chunk | no `React.lazy`/code-splitting anywhere; recharts ships to users who never open Reports |
| Role knowledge duplicated in UI | `App.tsx` route guards + `Layout.tsx` `NAV_ITEMS` + backend RBAC each hold their own copy of "which role sees what" |
| Zero CI, weak config hygiene | no `.github/`, no `golangci-lint`; `config.go` happily boots production with `JWT_SECRET=dev-only-insecure-secret-change-me` |
| Go test gap in the middle | `ruleengine` and `workflow` are well tested; `api` handlers, `middleware`, `audit`, `reports`, `auth` have **no** Go tests (only the E2E covers them) |

Goal: a layered, contract-first architecture where each concern has exactly one home,
enforced by CI — without astronaut abstractions that a two-person project can't feed.

---

## 2. Guiding principles

1. **Dependencies point inward**: `transport → service → storage`; domain types know
   nothing about HTTP or GORM.
2. **One home per concern**: pricing math, state machine, authorization, error
   mapping, pagination — each defined once.
3. **Contract-first**: an OpenAPI document is the single source of truth; both the Go
   server types and the TypeScript client types are generated from it.
4. **Interfaces only at real seams** (repositories, clock). No speculative genericity.
5. **Every phase ships green**: `make test` + `make e2e` after each phase; a phase is
   also a natural PR boundary.
6. **Behaviour freeze**: REST paths, payloads, RBAC semantics, and the Persian UI do
   not change (except where a phase explicitly says so, e.g. money precision).

---

## 3. Target structure

### 3.1 Backend

```
backend/
  cmd/
    api/                     wiring only (config → platform → services → transport)
    seed/                    thin main; fixtures move to internal/fixtures
  api/
    openapi.yaml             THE contract (phase 3)
  internal/
    domain/                  pure types: entities, enums, Rial, domain errors
      claim.go coverage.go employee.go user.go audit.go
      money.go               type Rial int64 + pricing-rounding policy
      errors.go              NotFound / Forbidden / Conflict / Validation / Unprocessable
      clock.go               Clock interface (real impl in platform)
    service/                 use-cases; owns transactions & authorization decisions
      claims/                state machine (from internal/workflow) + create/list/get
                             + ownership policy (from api/handlers.go authorize*)
      coverage/              pricing engine (from internal/ruleengine) +
                             PublishRuleVersion (tx from reference_handlers.go:143)
      employees/             CRUD + dependents + remaining-caps orchestration
      users/                 login/token issuance (from api/auth_handlers.go) + admin CRUD
      audit/                 (from internal/audit) — Log/Query/Trail
      reports/               (from internal/reports)
      integration/           employee sync upsert (tx from integration_handlers.go:43)
    storage/
      postgres/              GORM models (gorm tags ONLY) + repository impls + TxManager
                             repositories satisfy interfaces declared BY the services
    transport/
      http/                  chi router, authn/RBAC middleware, DTOs (json tags ONLY),
                             one error mapper (domain error → status + {"error": ...}),
                             params.go (pagination / date-range parsing, from handlers.go)
    platform/
      config/                env parsing + hard validation (see phase 0)
      database/              connect + migrate (from internal/db)
      logging/               slog setup; request-ID-aware logger
  migrations/                unchanged (golang-migrate stays the schema owner)
```

Entity shape policy (deliberately **two** representations, not three):
- `domain.*` structs are the service-layer currency — no tags.
- `storage/postgres` maps rows ↔ domain; `transport/http` maps DTOs ↔ domain.
- Request DTOs mostly exist already (`createClaimRequest`, …); response DTOs are new
  but small (mostly field-for-field, written once).

The audit invariant ("audit row commits atomically with the change it describes",
currently via passing `*gorm.DB` into `audit.Log`) is preserved by `TxManager`:
`txm.Do(ctx, func(r Repos) error { ... })` hands services a transaction-scoped
repository set.

### 3.2 Frontend

```
frontend/src/
  api/
    schema.d.ts              GENERATED from backend/api/openapi.yaml (phase 3)
    client.ts                axios + interceptors (unchanged)
    <resource>.ts            thin typed wrappers (types imported from schema)
  app/
    router.tsx               route table w/ React.lazy per page (phase 5)
    routes.ts                ONE role→routes/nav map consumed by router AND Layout
  components/                DataTable, FilterBar, Field, StatusBadge, … (extracted)
  features/                  pages grouped by area (claims/ employees/ admin/ auditor/)
  hooks/                     useFetch (loading/error/refetch) or @tanstack/react-query
  i18n/messages.ts           ALL user-facing strings (joins the enum maps from lib/format.ts)
  lib/format.ts              number/date formatting only
```

### 3.3 Repo level

```
.github/workflows/ci.yml     backend + frontend + docker + (nightly) e2e jobs
backend/.golangci.yml        linter config
.editorconfig
docs/adr/                    0001 layering, 0002 contract-first, 0003 money, 0004 timezone
```

---

## 4. Phases

Ordered so risk lands where the safety net is strongest; each phase is independently
valuable and shippable.

### Phase 0 — Tooling & guardrails *(no behaviour change; small)*
- `golangci-lint` (errcheck, govet, staticcheck, revive, gosec) + fix findings.
- GitHub Actions: ① backend — lint, build, `go test ./...` with a Postgres service
  container (reuse the `TEST_DATABASE_URL` + migrate pattern from
  `ruleengine_integration_test.go`); ② frontend — `tsc -b`, `vite build`, oxlint;
  ③ docker images build; ④ E2E via compose (nightly / on-demand — it needs Chrome).
- `platform/config` validation: **refuse to boot** when `APP_ENV=production` and
  `JWT_SECRET` is the known default or `DATABASE_URL` is missing.
- Replace stdlib `log` with `slog` (JSON in production), request ID from chi
  propagated into every request-scoped log line.
- `.editorconfig`; `docs/adr/` seeded with the decisions in this plan.

### Phase 1 — Domain & error foundation *(small–medium)*
- Create `internal/domain`: move enums + entities out of `internal/models`, strip all
  tags; add the error taxonomy and wrap existing sentinels
  (`ruleengine.Err*`, `workflow.Err*`) so they classify into it.
- `transport/http/errors.go`: single mapper (grows out of `handlers.go
  mapDomainError`); handlers stop hand-picking status codes.
- Mechanical, compile-driven; no test semantics change.

### Phase 2 — Services & repositories *(the core move; large, ~half the total effort)*
- Introduce `storage/postgres` (GORM models + repo impls + TxManager) and the seven
  service packages, absorbing in this order (each its own commit):
  1. `service/coverage` ← `internal/ruleengine` + rule publishing (pure `Compute`
     stays a pure function; its table tests move untouched).
  2. `service/claims` ← `internal/workflow` + claim handlers' logic + ownership policy.
  3. `service/employees`, `service/users`, `service/integration`, `service/reports`,
     `service/audit` ← remaining handler bodies.
- Handlers shrink to: decode DTO → call service → write DTO/error. Shared param
  parsing goes to `transport/http/params.go`.
- `cmd/seed` calls services instead of driving GORM + workflow directly; fixture data
  moves to `internal/fixtures`.
- **Tests**: existing integration tests move with their logic (same rollback-tx
  pattern); add service unit tests with in-memory fake repos for authorization edges;
  add `httptest` tests for the error mapper and one representative handler per verb.
- Exit criterion: `internal/api`, `internal/models`, `internal/ruleengine`,
  `internal/workflow`, `internal/audit`, `internal/reports`, `internal/db` are gone;
  grep for `gorm.` outside `storage/` returns nothing.

### Phase 3 — Contract-first API *(medium)*
- Author `backend/api/openapi.yaml` to match today's behaviour exactly
  (`docs/API-CONTRACT.md` is the draft; it becomes prose commentary afterwards).
- `oapi-codegen`: generate request/response types (and optionally the chi server
  interface) into `transport/http/openapi/`; hand-written DTOs from phase 2 are
  replaced by generated ones where they match.
- CI job: handlers validated against the spec (kin-openapi request/response
  validation middleware enabled in the httptest suite).
- Frontend: `openapi-typescript` generates `src/api/schema.d.ts`; delete the
  hand-maintained `types.ts`; `api/*.ts` wrappers import generated types.
- Exit criterion: a field added to the spec fails CI until both sides regenerate.

### Phase 4 — Money & time correctness *(medium; the only behaviour-adjacent phase)*
- `domain.Rial int64` (whole rials — the currency has no fractional everyday use):
  migration changes money columns `NUMERIC(14,2) → NUMERIC(14,0)`; JSON stays plain
  integers (no client change — current values are all whole).
- All pricing in integer math; **one** rounding function (half-up to whole rial at
  the final payable only) with golden table tests reproducing today's outputs for the
  seed scenarios byte-for-byte.
- `coverage_percent` stays `NUMERIC(5,2)`; computed as basis points
  (`amount × bp / 10_000`).
- Inject `domain.Clock` into claims/coverage services (kills the 6 raw `time.Now()`
  sites); pin day-boundary logic (`contractYearWindow`, waiting-period comparison) to
  a configured `TZ=Asia/Tehran` location. ADR-0003/0004 record both decisions.

### Phase 5 — Frontend restructure *(medium)*
- Route-level `React.lazy` + `Suspense` (in `app/router.tsx`); recharts and the
  auditor pages split out of the main bundle — target < 250 KB initial chunk.
- Extract `DataTable`, `FilterBar`; introduce **@tanstack/react-query** for
  fetch/cache/invalidate (replaces the 14 copies of loading/error/refetch state) —
  mutations invalidate list queries, killing the hand-rolled `reload()` pattern.
- `app/routes.ts`: one `{path, roles, nav}` table drives both the router guards and
  the sidebar (single copy of role knowledge in the UI).
- `i18n/messages.ts`: hoist inline Persian strings (pages keep JSX structure; strings
  become `t.claims.submitAction`-style constants — no i18n framework, just one file).
- E2E updates only where labels move; suite must stay 13/13.

### Phase 6 — Ops polish *(small, optional)*
- `/livez` (process) vs `/readyz` (DB ping) split; Prometheus `/metrics`
  (chi middleware) exposed on an internal port.
- Connection-pool sizing via env; nginx gzip already covers the frontend.
- README "operations" section: backup/restore (`pg_dump`), log shipping note,
  secret rotation.

---

## 5. Explicit non-goals

- No ORM swap (GORM stays, hidden behind repositories), no framework swap (chi stays).
- No microservices, no message queues, no CQRS — wrong weight class for this system.
- No API versioning beyond the existing `/api/v1` freeze.
- No visual/UX redesign; the Persian UI and its terminology stay as shipped.
- `migrations/` stays the single schema owner (no GORM AutoMigrate, ever).

## 6. Risks & mitigations

| Risk | Mitigation |
|---|---|
| Phase 2 touches everything at once | absorb one service per commit, in the listed order; E2E after each commit, not just each phase |
| Money migration changes stored values | `NUMERIC(14,2)→(14,0)` only after a `SELECT` proves every existing value is integral (seed + E2E data are); golden pricing tests before the switch |
| Same-behaviour regressions the E2E can't see (timezones, rounding tails) | golden table tests in phase 4 written **before** the change, against current outputs |
| Generated-code churn in reviews | generation output committed, regenerated in CI with a diff check |
| Plan fatigue | phases 0–2 alone already deliver most of the maintenance value; 3–6 can follow opportunistically |

## 7. Suggested order of execution & rough effort

`0 → 1 → 2` (the core, ~70% of value) then `3 → 5` (contract + frontend, most of the
rest) then `4 → 6`. Relative effort: 0:▮ 1:▮▮ 2:▮▮▮▮▮▮ 3:▮▮▮ 4:▮▮▮ 5:▮▮▮ 6:▮.

## 8. Verification (every phase)

```bash
cd backend && go build ./... && golangci-lint run   # from phase 0
make test                                           # unit + integration (Postgres)
make e2e                                            # 13-step browser suite, twice for idempotency-sensitive phases
```
plus phase-specific gates named above (grep-for-gorm in phase 2, spec-validation in
phase 3, golden pricing tables in phase 4, bundle-size budget in phase 5).
