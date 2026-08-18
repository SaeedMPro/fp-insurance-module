# Testing strategy

The suite is layered the way the code is, so a failure points at a layer rather
than at "something broke". 54 Go tests and a 14-step browser suite.

## Unit — no database

Fast, and where the arithmetic lives.

| Area | What it pins |
| --- | --- |
| Money | That percentage conversion **rounds rather than truncates** — the defect that would otherwise underpay every claim under a 33.33% rule — plus half-up rounding and no overflow at the schema's ceiling |
| Pricing | The pure `Compute` against all five service types: per-claim cap, annual cap, both at once, an exhausted cap, and an uncapped rule |
| Time | That the contract-year window is identical whatever the host timezone, that it anchors on the rule's anniversary, and that business days normalise to Tehran midnight |
| Configuration | That production refuses a default secret, a short secret, and a missing database URL |
| File store | That hostile keys — `../secret`, absolute paths, empty — cannot escape the storage root |

## Integration — real PostgreSQL

Wired by the same composition root production uses, so these exercise the real
object graph rather than a hand-assembled approximation. Each test rolls back, so
seeded data is never polluted.

They cover the workflow happy path (asserting the computed amount and the payment
row), the reject and return-for-documents paths and their reason requirements,
illegal transitions, forbidden actors, an employee seeing only their own claims,
every refusal in the pricing engine, same-day rule republishing and its tiebreak,
the admin-account invariants, and the whole document story — upload, list,
download, the audit entry, the freeze across the returned-for-documents loop, a
disguised file type, and the access boundaries.

## Contract — the spec and the code held together

Two tests, described in [Design decisions](decisions#the-openapi-document-is-the-contract).
One fails if the router and the spec disagree in either direction; the other
validates real responses against the schemas.

To check the second one actually works, a divergence was introduced on purpose
and confirmed to fail the test. A guard nobody has seen fail is not yet a guard.

## Golden — pricing frozen

An exhaustive table of pricing boundaries locked to expected rial amounts,
captured from the implementation that preceded the integer-money migration.
Regenerating takes a deliberate flag, so a change in pricing behaviour is a
reviewable diff.

## Browser — the real Persian interface

14 steps driving headless Chrome through the actual UI: an employee creating and
submitting a claim, a reviewer approving, paying and closing it, the reject path
with its mandatory Persian reason, an administrator publishing a rule version
through the form and the next claim being repriced by it, the
return-for-documents loop including a file upload and the freeze on resubmission,
RBAC denials, the audit trail, and the reports.

Numeric assertions go through the API; the UI assertions confirm the Persian
screens actually drive those transitions.

## One thing that was silently broken

Integration tests **skip** when they cannot reach the test database — and a
package whose tests all skip still prints `ok`. A wrong database URL therefore
produced a green build with zero integration coverage:

```
TEST_DATABASE_URL=postgres://nobody@127.0.0.1:9/none  go test ./internal/service/claims/
ok  	insurance-module/internal/service/claims	0.008s
```

CI now runs the suite verbosely and fails the build if the skip message appears,
reporting how many cases actually ran. Green means covered, which it did not
before. See [CI pipeline](ci).
