# Screenshot tour

The interface is Persian and right-to-left throughout, with Jalali dates and
Persian digits. Captions here are in English.

## Signing in

![The Persian login screen](../img/login.png)

Four roles share one login screen. What a user can reach afterwards is decided
by their role at the route and by ownership inside each service — never by
hiding a button.

## An employee's claims

![An employee's claim list, showing status badges in Persian](../img/employee-claims.png)

Employees see only claims they created. That restriction lives in the claims
service, so it holds no matter which entry point asks.

## A claim that was returned for documents

![Claim detail: the reviewer's reason, the documents section with an upload control, and the full history](../img/claim-documents.png)

The richest screen in the application, and the one worth understanding:

- The reviewer's reason for returning it is shown to the employee.
- The **مدارک** (documents) section lists what is attached, with a download for
  each, and offers an upload control — but only because this claim is in a state
  that accepts one.
- The history at the bottom is the audit trail for this claim, each entry
  expandable to its before and after state.

Submit the claim again and the upload control disappears, because the documents
freeze. See [Claim documents](../how-it-works/documents) for why that matters.

## Remaining allowance

![The employee's own coverage: cover, per-claim cap and remaining annual cap per service type](../img/my-coverage.png)

Computed live from the rules that apply to this employee's plan and what they
have already committed this contract year — not stored and not cached.

## The reviewer's queue

![The reviewer's queue of submitted claims](../img/review-queue.png)

A reviewer sees every claim, not just their own, and has the transition actions
an employee does not.

## Publishing a policy change

![The coverage rules screen with the publish form and version history](../img/coverage-rules.png)

The screen the whole project is arranged around. See
[The one idea](the-one-idea).

## Reports

![The reports dashboard: totals, spend by service type, spend by month, and spend per employee](../img/reports.png)

Read-only aggregations over committed spend, filterable by date range.

:::note
The monthly chart's axis is labelled with Gregorian months. Monthly grouping is
done in the database with `TO_CHAR(receipt_date, 'YYYY-MM')`, so it does not
follow the Jalali calendar the rest of the interface uses. It is a known
limitation rather than an oversight — see [Scope and limits](../about/scope).
:::

## The audit trail

![The audit log with Persian action labels and expandable before/after state](../img/audit-log.png)

Filterable by entity, actor, action and date range. The realistic use is
narrow and specific: an employee disputes a decision, you enter their claim's id
and read the whole history back.
