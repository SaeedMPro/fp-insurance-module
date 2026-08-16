# ADR-0002 — The OpenAPI document is the contract, enforced by tests

**Status**: accepted

## Context

Three things have to agree about this API: the Go router, the OpenAPI
document, and the frontend's TypeScript types. Keeping them in agreement by
discipline alone does not survive a semester of changes. The usual failure is
silent and one-directional — someone adds a route, or changes a field from
required to optional, and the document keeps describing the old API. The
document then becomes something nobody trusts, which makes it worthless
exactly when it is needed: when the parent HR system integrates against it.

Generating the server from the spec was considered and rejected — it
constrains handler structure and adds a build step to every change. Generating
the spec from the code was also rejected: annotation-derived documents drift
in the other direction (they describe what the code does, including its bugs)
and cannot be reviewed before the code exists.

## Decision

`backend/api/openapi.yaml` is the single source of truth, hand-written and
reviewed. Nothing generates it. Agreement is enforced by three mechanisms:

1. **Route coverage.** `transport/http.Routes()` enumerates every
   `METHOD /path` the router serves. `TestOpenAPISpecCoversEveryRoute` fails
   if the router and the document disagree **in either direction** — an
   undocumented route and a documented-but-missing route are both failures.
2. **Behavioural conformance.** `TestOpenAPIConformance` boots the real router
   against a real database and validates actual requests and responses against
   the spec with `kin-openapi`. This catches what a path list cannot: a
   response field that changed type, a status code that is not in the
   document, a required property the handler stopped sending.
3. **Generated client types.** `frontend/src/api/schema.d.ts` is produced from
   the same document by `npm run gen:api`. `src/api/types.ts` only re-exports
   friendly aliases from it. CI fails if the committed output is stale.

The document is served at `/openapi.yaml`, with Swagger UI at `/swagger`.

## Consequences

The spec cannot silently fall behind, and a breaking API change is visible as
a failing test plus a diff in the generated types — usually before the
frontend is touched at all. The parent system can integrate against a document
that is checked rather than promised.

The cost is that adding an endpoint is a two-file change (spec and router) and
the spec must be written first, which feels slower on small additions. Two
guards also means two ways to be told the same thing when a route is genuinely
new. Both are accepted: the alternative is a document that is wrong in ways
nobody notices.

One nuance worth recording: because the frontend's types are generated,
"widening" a field in the spec (making a required property optional) surfaces
as TypeScript errors at every use site. That is the intended behaviour — it is
the compiler pointing at exactly the code that assumed otherwise.
