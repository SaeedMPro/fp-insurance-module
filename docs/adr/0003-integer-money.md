# ADR-0003 — Money is integer rials, with exactly one rounding point

**Status**: accepted

## Context

The pricing engine originally carried amounts and percentages as `float64`,
matching the `NUMERIC(14,2)` and `NUMERIC(5,2)` columns through GORM's default
mapping. This is the standard way to get money wrong.

Two concrete defects, not hypotheticals:

- **Percentages do not survive the trip.** `33.33 * 100` is
  `3332.9999999999995` in float64. Truncating that to an integer gives 3332
  basis points — every claim priced under a 33.33% rule is quietly underpaid.
- **Rounding happens repeatedly and invisibly.** Applying a percentage, then
  comparing against a per-claim cap, then against a remaining annual cap, gave
  three chances for a fractional value to appear. Amounts that differed only
  in the last binary place produced different results at cap boundaries, and
  fractions could reach the annual-cap total, where they accumulated.

The rial has no fractional unit in everyday use, so carrying decimal places at
all was solving a problem that does not exist while creating one that does.

## Decision

Money is an integer type in the domain:

- `domain.Rial` is an `int64` of **whole rials**. int64 covers ±9.2×10^18
  rial, five orders of magnitude past the schema's `NUMERIC(14,0)` ceiling.
- `domain.Percent` is an `int32` of **basis points**: 1% = 100, so 70.00% is
  7000 and 33.33% is 3333. This matches `NUMERIC(5,2)` exactly.
- Conversion from the database's float representation **rounds, never
  truncates** (`PercentFromFloat`, `RialFromFloat`).
- There is **exactly one rounding decision in the whole pricing path**:
  half-up to the whole rial, inside `Percent.ApplyTo`. Caps are therefore
  always compared against an already-whole amount, and no fractional value can
  reach a payment row or an annual-cap total.

No float arithmetic occurs between reading a rule and writing a payable
amount.

## Consequences

Pricing is exact and reproducible: the same inputs give the same rial, on any
machine, in any order. Cap comparisons are integer comparisons, so boundary
behaviour is decidable rather than approximate. Overflow is not a practical
concern at the schema's scale.

The costs are real but small. Reading `7000` and knowing it means 70% takes a
moment's translation, so the boundaries convert explicitly and the type
carries a doc comment. The transport and storage layers must convert at their
edges, since JSON and `NUMERIC` still speak decimals — that conversion is
confined to the DTO and row mappers.

**Guarding the migration.** Changing the numeric representation of an existing
pricing engine is exactly the kind of change that silently re-prices claims.
`service/coverage/golden_pricing_test.go` freezes an exhaustive table of
boundaries — fractional percentages, .5 rounding, cap collisions, exhausted
caps — against values captured from the pre-migration implementation.
Regenerating it requires a deliberate `UPDATE_GOLDEN=1`, so any behavioural
change appears as a reviewable diff rather than as a difference in someone's
reimbursement.
