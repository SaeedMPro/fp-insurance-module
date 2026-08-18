# Where else this could be used

The domain is supplementary health insurance, but the mechanism underneath it is
not about insurance at all.

## The reusable core

Three things, composed:

1. **A configurable entitlement** — a percentage, a per-transaction ceiling, a
   periodic ceiling, an eligibility window, and who qualifies.
2. **A multi-stage approval workflow** — with rejection and
   return-for-information branches, and a reason recorded on each.
3. **A complete audit trail** — written inseparably from the change it describes.

Nothing in that description mentions medicine. Any benefit where an organisation
pays a share of an employee's expense up to limits, with someone checking the
claim, has the same shape.

## Plausible neighbours

| Use | What changes | What does not |
| --- | --- | --- |
| **Staff loans** | Entitlement becomes an amount and a repayment period; the workflow gains a disbursement step | Versioned rules, approval workflow, audit, caps |
| **Training and tuition** | A yearly allowance per employee, possibly per grade | Everything else |
| **Travel and subsistence** | Per-diem rates by destination; more claim types | Everything else |
| **Wellness and gym subsidy** | A simple percentage with a yearly cap | Everything else |
| **Equipment allowance** | A cap per item category and a renewal period | Everything else |

The versioning story matters more, not less, in these: "what was the tuition
policy when I enrolled?" is exactly the question the effective-date model exists
to answer.

## What would need work first

Being honest about the distance:

- **Service types are a flat catalogue.** A benefit needing categories and
  sub-categories would need a hierarchy.
- **Caps are per (plan, service type).** A shared pool across several categories
  — one wellness budget spanning gym, therapy and equipment — is not expressible.
- **The entitlement is a percentage with ceilings.** A tiered structure (100% of
  the first tranche, 50% thereafter) would need the pricing function extended,
  though it is a pure function, which is the easiest place to extend.
- **One approval level.** Loans in particular usually need more.

## What would carry over unchanged

The layering and its dependency rule, the money and time decisions, the
contract-first API discipline, the audit mechanism, the two-level authorisation,
and the test structure. That is most of the engineering — the insurance-specific
part is smaller than it looks, which is the point.
