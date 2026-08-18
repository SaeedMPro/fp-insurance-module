# The one idea

Every part of this system is arranged around one property: **a benefit rule is a
row in a table, not a line of code.**

That sounds like an implementation detail. It is the difference between a policy
change taking a form submission and taking a software release.

## What a rule contains

A coverage rule answers, for one plan and one service type: what share do we
pay, up to what per-claim ceiling, up to what annual ceiling, after how long an
employee must have worked here, and for which family members.

Here are the rules the demo data ships with for the *Standard* plan:

| Service | Cover | Per claim | Per year | Waiting period | Eligible |
| --- | --- | --- | --- | --- | --- |
| Outpatient visit | 70% | 500,000 | 5,000,000 | none | self, spouse, child, parent |
| Pharmacy | 80% | 1,000,000 | 10,000,000 | none | self, spouse, child, parent |
| Dental | 50% | 3,000,000 | 15,000,000 | 90 days | self, spouse, child |
| Hospitalisation | 90% | 50,000,000 | 100,000,000 | 30 days | self, spouse, child, parent |
| Optometry | 60% | 2,000,000 | 4,000,000 | 180 days | self, spouse, child |

Amounts are Iranian rials. Not one of those numbers appears in the Go source.
The pricing package's own documentation says so outright: *"Changing a benefit
means publishing a new rule version through this service — never a code
change."*

## Changing a benefit

An administrator opens **Coverage rules**, fills in the new version, and
publishes it.

![The coverage rules screen: a form for publishing a new rule version, above the history of existing versions](../img/coverage-rules.png)

The lower table is the part worth looking at. Dental on the Standard plan has
two rows: one that ran from ۱۴۰۴/۰۱/۰۱ to ۱۴۰۴/۰۶/۳۱ at 45%, and one that opened
on ۱۴۰۴/۰۷/۰۱ at 50%. The first was not edited or deleted when policy changed —
it was **closed**, and it is still there.

## Why closing beats overwriting

Because a claim is priced against the rule that was in force **on its receipt
date**, not the rule that is in force today.

A claim with a receipt dated ۱۴۰۴/۰۵/۱۰ is priced at 45%. A claim dated
۱۴۰۴/۰۸/۱۰ is priced at 50%. Both remain correct, and both remain explainable a
year later — which is what an employee querying their reimbursement, or an
auditor sampling last year's payments, actually needs.

If rules were overwritten in place, the honest answer to "why was I paid this?"
would be *we no longer know*.

## What follows from it

Three consequences run through the rest of the system:

- **Pricing happens exactly once, at approval**, and the amount, the percentage
  applied and the rule version are stored on the claim. See
  [Pricing engine](../how-it-works/pricing).
- **"Active" is a date range, not a flag** — which is why publishing has to
  handle a rule republished on the day it took effect. See
  [Rule versioning](../how-it-works/rule-versioning).
- **A configuration change is audited like any other change**, with the previous
  and new rule recorded. See [Audit trail](../how-it-works/audit-trail).
