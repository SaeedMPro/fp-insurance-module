# ADR-0001: Layered architecture with services and repositories

Date: 2026-07-19 · Status: accepted

## Context

The backend grew feature-first: HTTP handlers issued GORM queries directly (29 call
sites), two multi-step transactions lived inside handlers, and `internal/models`
carried both `gorm` and `json` tags so the persistence schema *was* the wire format.
Authorization policy existed in three places (route middleware, handler helpers, and
the workflow engine).

## Decision

Adopt a three-layer architecture with dependencies pointing inward:

```
transport/http  →  service/*  →  storage/postgres
        └──────────── domain ←──────────────┘
```

- `internal/domain` — pure entities, enums, money/clock primitives, and the error
  taxonomy. No gorm/json tags, no HTTP or database imports.
- `internal/service/*` — one package per use-case area (claims, coverage, employees,
  users, audit, reports, integration). Services own transactions and authorization
  decisions. Repository **interfaces are declared here**, next to their consumers.
- `internal/storage/postgres` — GORM row models (gorm tags only), repository
  implementations, and the `TxManager` that gives a service a transaction-scoped
  repository set (preserving the "audit row commits atomically with its change"
  invariant).
- `internal/transport/http` — chi router, authn/RBAC middleware, request/response
  DTOs (json tags only), and a single domain-error → HTTP-status mapper.

Two entity representations, not three: domain structs are the service currency;
storage and transport each map to/from domain at their boundary.

## Consequences

- A schema change no longer silently changes the public API (and vice versa).
- Services are unit-testable with fake repositories; GORM appears nowhere outside
  `storage/`.
- The cost is mapping code at two boundaries — accepted as mechanical and explicit.
- GORM and chi stay; swapping them is a non-goal (repositories/handlers isolate them).
