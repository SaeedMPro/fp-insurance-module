# HR integration

The module is designed to sit beside an organisation's existing HR or payroll
system rather than replace it. Two endpoints exist for that system to call.

They are **API-only and deliberately have no interface** — this is a
system-to-system seam, not a screen someone operates.

## Authentication

`X-API-Key`, not a user token. The key is compared as a SHA-256 hash against
active rows; the plaintext is never stored. These endpoints carry no role and sit
outside the user model. See [Authentication](authentication).

## Syncing the roster

```
POST /api/v1/integration/employees/sync
X-API-Key: <key>
```

```json
{
  "employees": [
    {
      "personnel_no": "P-2001",
      "full_name": "کارمند جدید",
      "national_id": "0012345678",
      "employment_status": "active",
      "hire_date": "2026-01-01T00:00:00Z",
      "department": "عملیات"
    }
  ]
}
```

```json
{ "created": 1, "updated": 0 }
```

Matching is on **`personnel_no`** — an existing employee is updated, an unknown
one is created. That makes the call idempotent: sending the same roster twice
leaves the same state, so the parent system can push on a schedule without
tracking what it sent last time.

Note what sync does *not* set: a coverage plan. Assigning a plan is a benefits
decision, not an HR record, so it stays with an administrator. A synced employee
with no plan cannot have claims created for them until one is assigned — which is
a deliberate, visible gap rather than a silent default.

## Checking a claim's status

```
GET /api/v1/integration/claims/{id}/status
X-API-Key: <key>
```

Returns the status and, once decided, the payable amount — enough for a payroll
system to reconcile without being given the whole claim record.

## What is not built

There is no live connection: no scheduled pull, no webhook out, no message queue.
The seam is defined and authenticated; the integration itself is
[out of scope](../about/scope) and would be the first thing to build for real
deployment.
