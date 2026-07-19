# ADR-0004: Business time pinned to Asia/Tehran via an injected Clock

Date: 2026-07-19 · Status: accepted

## Context

`time.Now()` was called directly at six sites inside the workflow and rule engines.
Day-boundary logic — the annual-cap contract-year window and waiting-period
eligibility — therefore depended on the *server's* local timezone: the same claim
could price differently on a UTC host and a local host near midnight. Time-dependent
behaviour was also untestable without sleeping or monkey-patching.

## Decision

- `domain.Clock` (interface: `Now() time.Time`) is injected into every service that
  reasons about time. Production wiring uses a real clock; tests use a fixed one.
- All *business-day* decisions (contract-year windows, waiting periods, effective
  dates) are evaluated in a single configured location, `Asia/Tehran` — the
  organisation's civil calendar — regardless of host timezone.
- Timestamps remain stored as `TIMESTAMPTZ`/UTC on the wire; only day-boundary
  interpretation is Tehran-pinned.

## Consequences

- Deterministic pricing regardless of deployment region; reproducible tests with a
  frozen clock.
- Iran abolished DST in 2022, so the offset is a stable +03:30 — but using the IANA
  zone (not a fixed offset) keeps historical dates correct either way.
