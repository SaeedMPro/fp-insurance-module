# ADR-0001 — Layered architecture with a dependency-free domain

**Status**: accepted

## Context

The first working version of the backend was organised the way most small Go
services start out: `internal/models` held GORM structs whose JSON tags were
also the REST wire format, and `internal/api` handlers reached into GORM
directly alongside two engine packages. It worked, and for a while it was
faster to change than anything else would have been.

The costs showed up as soon as the system had more than one reason to change:

- A GORM struct tag change was an API change. Renaming a column, adding an
  index hint, or changing a `NUMERIC` mapping altered the JSON contract, so a
  persistence detail could break the frontend.
- Business rules could not be read in one place. "May this user submit this
  claim?" was partly in a route group, partly in a handler, and partly in an
  engine closure.
- Nothing could be tested without a database, because the rules were written
  against `*gorm.DB` rather than against an interface.
- HTTP status codes were chosen inside business logic, which meant the
  business logic had an opinion about HTTP.

## Decision

Organise the backend into layers with a one-directional dependency rule:

```
transport/http  →  service/…  →  storage/postgres
                       ↓
                    domain
```

- **`domain`** imports nothing of ours. It holds entities, enums, the money
  types, the `Clock` interface, and the error kinds. Entities carry **no**
  struct tags of any kind.
- **`service/…`** is the business logic, one package per capability. Each
  service declares the storage interface *it* needs — the interface is owned
  by the consumer, not the provider — and receives an implementation by
  injection. Multi-write operations take an `atomic` function so a service can
  say "these commit together" without knowing what a transaction handle is.
- **`storage/postgres`** is the only package that imports GORM. It has its own
  row structs with the GORM tags, plus mappers to and from domain entities.
- **`transport/http`** owns the wire format (its own DTOs), authentication,
  role gating, and the single mapping from `domain.Kind` to HTTP status.
- **`platform/…`** holds infrastructure with no business meaning: config,
  database, file store, logging.
- **`app`** is the composition root — the one place the object graph is built,
  used by both `cmd/api` and the integration tests.

## Consequences

**What this buys.** The wire format, the schema, and the business model can
now change independently, because each has its own representation and an
explicit mapping between them. Business rules are testable without a database
(the pricing function is a pure function; services take interfaces). A handler
cannot invent a status code, and a service cannot import `net/http`. Because
tests build their object graph with `app.Build`, they exercise the same wiring
production uses instead of a hand-assembled approximation.

**What it costs.** There is more code: three representations of a claim
(domain entity, storage row, transport DTO) and the mappers between them. For
a purely CRUD endpoint this is pure overhead, and we accept that — the
endpoints that matter here are not CRUD, and a uniform structure is worth more
than a locally shorter one.

**The rule that keeps it honest.** The dependency direction is the whole
decision. A single `import` of `gorm.io/gorm` in a service package, or of
`net/http` in `domain`, undoes it quietly. Review for that specifically.
