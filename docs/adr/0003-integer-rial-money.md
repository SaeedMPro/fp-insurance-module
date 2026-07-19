# ADR-0003: Money as integer Rial with a single rounding policy

Date: 2026-07-19 · Status: accepted

## Context

Monetary amounts flowed through the system as Go `float64` (25 sites in the models
and rule engine alone), with an ad-hoc `round2()` at the pricing boundary. Binary
floating point cannot represent decimal amounts exactly, and rounding behaviour was
implicit. The Iranian rial has no fractional unit in everyday use; every value ever
stored by the system is a whole number of rials.

## Decision

- `domain.Rial` is an `int64` number of whole rials, used end-to-end in Go.
- Database money columns become `NUMERIC(14,0)` (migration verifies all existing
  values are integral first).
- JSON representation stays a plain number — no client-visible change.
- Percentages remain `NUMERIC(5,2)`; arithmetic uses basis points:
  `covered = amount × bp / 10_000`, computed in integer math.
- Exactly one rounding function exists (half-up to whole rial), applied only at the
  final payable amount; golden table tests pin today's outputs before the switch.

## Consequences

- No floating-point drift in sums (reports) or comparisons (caps).
- Amounts above ~92 quadrillion rial overflow int64 — five orders of magnitude above
  the schema's own NUMERIC(14) ceiling, so not a practical constraint.
- Any future fractional-currency requirement would need a minor-unit redesign; out of
  scope by explicit decision.
