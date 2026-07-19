# ADR-0002: Contract-first API with OpenAPI as source of truth

Date: 2026-07-19 · Status: accepted

## Context

The REST contract lived in prose (`docs/API-CONTRACT.md`) and was mirrored by hand in
`frontend/src/api/types.ts` (182 lines). Nothing but the browser E2E suite caught
drift between backend serialization, the prose, and the TypeScript types.

## Decision

`backend/api/openapi.yaml` becomes the single source of truth for the HTTP contract.

- Go: `oapi-codegen` generates request/response DTO types consumed by
  `transport/http`.
- TypeScript: `openapi-typescript` generates `frontend/src/api/schema.d.ts`; the
  hand-written `types.ts` is deleted.
- CI regenerates both and fails on diff, so the spec cannot lag the code (or vice
  versa).
- `docs/API-CONTRACT.md` remains as human-oriented commentary and examples.

## Consequences

- Adding/changing a field is a spec edit + regenerate on both sides — one workflow,
  no drift.
- Generated code is committed (reviewable, no build-time tool dependency for users).
- The spec also becomes usable for request validation middleware and future client
  generation.
