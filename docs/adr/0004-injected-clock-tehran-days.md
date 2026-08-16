# ADR-0004 — Injected clock, and business days in Asia/Tehran

**Status**: accepted

## Context

Several rules in this system are about *when*, not *how much*:

- A waiting period: eligibility starts `hire_date + waiting_period_days`.
- The contract year: the annual cap resets on the rule's `effective_from`
  anniversary, not on 1 January.
- Rule versioning: which version was in force on a claim's receipt date, and
  what happens when a rule is republished the same day it took effect.

Written with `time.Now()` inside the services, none of these can be tested
properly — a test for "the day the waiting period ends" would have to
manipulate the system clock or seed dates relative to today, which makes it
either flaky or unreadable.

There is a second, subtler problem. `time.Time` comparison is instant
comparison, not day comparison. A receipt timestamped on the first eligible
day, compared against an instant computed in the server's local zone, can land
on either side of the boundary depending on where the server runs. A container
in UTC and a developer's laptop in `Asia/Tehran` would disagree about whether
a claim qualifies — and the version that runs in production would be the one
nobody tested.

For a system whose users, contracts and calendar are all Iranian, the
correct civil day is unambiguously the Tehran one.

## Decision

**Inject the clock.** Services depend on a `domain.Clock` interface
(`Now() time.Time`), not on the `time` package. Production wires
`domain.SystemClock`; tests wire `domain.FixedClock` and state the date the
scenario is about.

**Compare civil days, in Tehran.** `domain.BusinessDay` normalises an instant
to midnight of its civil day in `Asia/Tehran`. Every date-boundary rule
compares business days, not instants:

- The waiting period compares `BusinessDay(receiptDate)` against
  `BusinessDay(hireDate) + waitingPeriodDays`.
- The contract-year window is computed in the business location.
- Rule effectivity (`effective_from` / `effective_to`) is a day comparison.

Timestamps that are records of *when something happened* — `submitted_at`,
`created_at`, audit entries — stay as instants. Only rules about days use
business days.

## Consequences

Time-dependent rules became testable as statements about dates:
`timezone_test.go` asserts the contract-year window is identical whatever the
host timezone, that it anchors on the anniversary, and that business days
normalise to Tehran midnight. A test can now say "the receipt is dated the
first eligible day" and mean it.

Deployment stops being able to change business outcomes. A server in UTC and
one in Tehran price the same claim identically, so the question "what timezone
is the container in?" is no longer a business question.

The costs: every service that reasons about time carries a `Clock` in its
constructor, which is a small amount of plumbing at every call site; and
`Asia/Tehran` must be resolvable at runtime, so the container image needs
timezone data. Developers also have to know which of the two treatments a
given field gets — the rule of thumb is that anything a *policy* is written in
terms of is a business day, and anything that is a record of an *event* is an
instant.
