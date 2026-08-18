# Audit trail

Every action that changes state writes an audit entry. The design goal is narrow:
make "a change happened but was not recorded" **unreachable**, rather than
merely unlikely.

## How that is guaranteed

The audit entry and the change it describes are written **in the same database
transaction**. Not afterwards, not on a queue, not best-effort. If the audit
write fails, the change rolls back with it.

This is why every claim transition goes through one shared code path — the audit
write sits in the shared part, so a new transition cannot forget it.

The same applies to configuration: publishing a coverage rule closes the old
version, inserts the new one, and writes the audit entry in one transaction.

## What an entry contains

| Field | Purpose |
| --- | --- |
| `entity_type`, `entity_id` | What changed — a claim, a coverage rule, a user |
| `action` | What was done |
| `actor_user_id`, `actor_username` | Who did it — the username is stored alongside the id **on purpose**, so the actor survives the user record being renamed or deactivated |
| `before_data`, `after_data` | JSON snapshots either side of the change |
| `metadata` | Anything action-specific |
| `occurred_at` | When |

## Recorded actions

Eleven kinds:

`login` · `submit` · `resubmit` · `start_review` · `approve` · `reject` ·
`return_for_docs` · `attachment_upload` · `mark_paid` · `close` · `config_change`

## Reading it

Two views over the same table.

**One claim's history** — `GET /claims/{id}/history`, shown at the bottom of the
claim detail screen with each entry expandable to its before and after state.
This is the view an employee's dispute actually needs.

**The whole trail** — `GET /audit-logs`, filterable by entity type, entity id,
actor, action and date range, with pagination. Available to administrators and
auditors.

![The audit log with Persian action labels and expandable before/after state](../img/audit-log.png)

The table is indexed on `(entity_type, entity_id)`, on `occurred_at`, and on
`actor_user_id` — the three ways it is actually queried.

## What it is not

It is not tamper-proof. Anyone with direct database access can edit the table;
there is no hash chain or append-only enforcement at the storage level. The
guarantee is that the *application* cannot produce an unrecorded change, which is
the realistic threat for an internal benefits system. Claiming more would be
overstating it.
