# Design decisions

Four decisions that shaped the codebase, each with the situation that prompted it
and what it cost. The code shows *what* was decided; this page is the *why*,
which is what you need before changing something deliberate.

## Layers, with a domain that depends on nothing

**The situation.** The first working version was organised the way small Go
services usually start: one package of structs whose JSON tags were also the wire
format, and handlers reaching into the ORM directly. It worked, and for a while
it was the fastest thing to change.

The costs appeared as soon as the system had more than one reason to change. A
struct tag change was an API change, so renaming a column could break the
frontend. Business rules could not be read in one place — "may this user submit
this claim?" was spread across a route group, a handler and a closure. Nothing
could be tested without a database. And HTTP status codes were being chosen inside
business logic.

**The decision.** `transport → service → storage`, with `domain` at the centre
and a one-directional dependency rule. Entities carry no struct tags at all;
serialisation belongs to transport, column mapping to storage.

**What it cost.** Three representations of a claim and the mappers between them.
For a purely CRUD endpoint that is pure overhead, and we accept it — the
endpoints that matter here are not CRUD, and a uniform structure is worth more
than a locally shorter one.

**What keeps it honest.** The dependency direction is the whole decision. One
`gorm.io/gorm` import in a service package, or `net/http` in `domain`, undoes it
quietly. Review for that specifically.

## The OpenAPI document is the contract

**The situation.** Three things must agree about this API: the router, the spec,
and the frontend's types. Discipline alone does not survive a semester. The usual
failure is silent — someone adds a route and the document keeps describing the
old API — and it becomes worthless exactly when it matters, when another system
integrates against it.

**Alternatives rejected.** Generating the server from the spec constrains handler
structure and adds a build step to every change. Generating the spec from
annotations drifts the other way: it documents what the code does, bugs included,
and cannot be reviewed before the code exists.

**The decision.** The document is hand-written and reviewed, and agreement is
enforced by three mechanisms:

- A test enumerates every route the router serves and fails if the spec and the
  router disagree **in either direction**.
- A second test boots the real router against a real database and validates
  actual requests and responses against the schemas — so the contract is checked
  as behaviour, not as prose.
- The frontend's TypeScript types are generated from the same document, and CI
  fails if the committed output is stale.

**What it cost.** Adding an endpoint is a two-file change and the spec goes first.
On small additions that feels slower. Two guards also means being told the same
thing twice when a route is genuinely new.

## Money is an integer number of rials

**The situation.** Amounts and percentages were originally `float64`, matching
the database columns. This is the standard way to get money wrong, and it
produced two concrete defects rather than hypothetical ones.

`33.33 × 100` is `3332.9999999999995` in float64. Truncating gives 3332 basis
points, so every claim under a 33.33% rule was quietly underpaid. And rounding
happened repeatedly: applying a percentage, then comparing against a per-claim
cap, then against a remaining annual cap gave three chances for a fraction to
appear — and fractions reaching the annual total accumulated.

**The decision.** `Rial` is an `int64` of whole rials; `Percent` is basis points,
so 70.00% is 7000. Conversion from the database's decimals **rounds, never
truncates**. And there is exactly **one rounding decision in the whole pricing
path** — half-up to the whole rial, inside the percentage application — so caps
are always compared against an already-whole number.

The rial has no fractional unit in everyday use, so carrying decimals was solving
a problem that does not exist while creating one that does.

**What it cost.** Reading `7000` and knowing it means 70% takes a moment's
translation. Transport and storage must convert at their edges, since JSON and
`NUMERIC` still speak decimals.

**Guarding the migration.** Changing the numeric representation of a working
pricing engine is exactly the change that silently reprices claims. A golden file
freezes an exhaustive table of boundaries against values captured from the old
implementation; regenerating it takes a deliberate flag. Any behavioural change
appears as a reviewable diff rather than as a difference in someone's
reimbursement.

## An injected clock, and business days in Asia/Tehran

**The situation.** Several rules are about *when*: a waiting period, a contract
year that resets on a rule's anniversary, which rule version was in force on a
date.

Written with `time.Now()` inside the services, none of them can be tested
properly — a test for "the day the waiting period ends" would have to manipulate
the system clock or seed dates relative to today, making it either flaky or
unreadable.

There is a subtler problem too. Comparing `time.Time` values compares *instants*,
not days. A receipt timestamped on the first eligible day, compared against an
instant computed in the server's local zone, can land on either side of the
boundary depending on where the server runs — so a container in UTC and a laptop
in Tehran would disagree about whether a claim qualifies, and the version that
runs in production is the one nobody tested.

**The decision.** Services depend on a `Clock` interface rather than the `time`
package, so tests state the date the scenario is about. And every date-boundary
rule compares **civil days normalised to Asia/Tehran**, not instants. For a
system whose users, contracts and calendar are all Iranian, that is
unambiguously the correct day.

Timestamps that merely record *when something happened* — submitted, created,
audited — stay as instants. Only rules about days use business days.

**What it cost.** Every service that reasons about time carries a clock in its
constructor, and the container image needs timezone data. Developers have to know
which treatment a field gets: anything a *policy* is expressed in terms of is a
business day; anything recording an *event* is an instant.

**What it bought.** Deployment can no longer change business outcomes. "What
timezone is the container in?" stopped being a business question.
