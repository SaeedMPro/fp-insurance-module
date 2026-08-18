# Pricing engine

Answers one question: *given this employee, this service, this amount and this
receipt date, how much is payable?*

## What it checks, in order

1. **Is the employee active?** A terminated employee is refused.
2. **Who is the beneficiary?** For a dependent, verify the dependent actually
   belongs to the claiming employee, and take their relation.
3. **Which rule applies?** The version for this (plan, service type) whose date
   range contains the receipt date. None → refused.
4. **Is that relation eligible** under this rule?
5. **Has the waiting period elapsed**, comparing civil days in Asia/Tehran from
   the employee's hire date?
6. **How much annual allowance is already committed** this contract year for the
   same employee, service type and plan?
7. **Compute.**

Steps 1–6 read the database. Step 7 does not.

## The computation

`Compute` is a pure function — rule, requested amount, and annual spend so far
in; payable amount and which cap bound it out. No database handle, which is why
it can be table-tested exhaustively.

```
payable = requested × coverage_percent      (rounded half-up to the whole rial)
        → capped at per_claim_cap           (if set)
        → capped at (annual_cap − already_used)
        → floored at zero
```

Order matters. The percentage is applied first and rounded immediately, so both
caps are compared against a whole number and no fraction can reach a payment or
an annual total.

## Worked examples

Outpatient visit on the Standard plan: **70%**, per-claim cap **500,000**,
annual cap **5,000,000**.

| Requested | × 70% | Per-claim cap | Annual left | **Payable** | What bound it |
| --- | --- | --- | --- | --- | --- |
| 350,000 | 245,000 | 500,000 | 5,000,000 | **245,000** | nothing — under both caps |
| 1,000,000 | 700,000 | 500,000 | 4,755,000 | **500,000** | per-claim cap |
| 1,000,000 | 700,000 | 500,000 | 300,000 | **300,000** | remaining annual cap |
| 1,000,000 | 700,000 | 500,000 | 0 | **0** | annual cap exhausted |

The last row is not an error. The claim is approvable and payable at zero, and
the audit trail records why — which is more useful to an employee than a refusal
with no number attached.

## When it refuses

Each refusal is a distinct error, surfaced as HTTP 422 with its own message:

| Situation | Meaning |
| --- | --- |
| No rule version covers the receipt date | Often a claim dated before the contract started |
| Beneficiary relation not eligible | e.g. a parent under a rule covering self, spouse, child |
| Waiting period not served | Counted in Tehran civil days from the hire date |
| Employee not active | Employment terminated |
| Dependent does not belong to this employee | Fails even if the dependent id is valid |

## The contract year

Annual caps reset on the **rule's own `effective_from` anniversary**, not on
1 January. A rule effective 2025-03-21 resets every 21 March, matching the
Iranian contract period the data is built around.

## Why pricing runs exactly once

`Approve` is the only transition that prices, and the transition table has no
`approved → approved` edge, so a claim cannot be silently repriced. The amount,
the applied percentage and the rule version are written in the same transaction
as the status change and the audit entry — so an approved claim carries the
evidence of how it was priced.

If the engine refuses, the approval fails atomically: the claim stays under
review and nothing is persisted.

## How the arithmetic is protected

Money is an integer number of rials and percentages are basis points, so no
float arithmetic happens between reading a rule and writing an amount. A golden
file freezes an exhaustive table of boundaries — fractional percentages,
half-way rounding, cap collisions, exhausted caps — against expected values, and
regenerating it takes a deliberate flag. Any change in pricing behaviour shows up
as a reviewable diff instead of as a difference in somebody's reimbursement.

See [Design decisions](../engineering/decisions#money-is-an-integer-number-of-rials).
