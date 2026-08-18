# Supplementary Insurance Module — REST API Contract

> **The machine-readable contract is [`backend/api/openapi.yaml`](../backend/api/openapi.yaml)** —
> browse it live at `http://localhost:8080/swagger` when the API is running
> (raw YAML: `/openapi.yaml`).
> that document is the single source of truth. The frontend's
> TypeScript types are generated from it (`frontend/src/api/schema.d.ts`, via
> `npm run gen:api`), and two Go tests keep the server honest about it:
> `TestOpenAPIConformance` validates real responses against the schemas, and
> `TestOpenAPISpecCoversEveryRoute` fails if a route is served but undocumented.
>
> This page remains as human-oriented commentary: the same surface with prose
> notes on authorization and behaviour. If the two ever disagree, the spec wins.

Base URL: `/api/v1`. All bodies are JSON. All timestamps are RFC3339. All money
fields are JSON numbers (Rial). Errors are always `{"error": "message"}` with a
non-2xx status.

Auth: `Authorization: Bearer <JWT>` for interactive users. The parent-system
integration endpoints use `X-API-Key: <key>` instead and carry no JWT.

Roles: `admin`, `reviewer`, `employee`, `auditor` (see `internal/domain.Role`).

## Auth

### POST /auth/login
Request: `{"username": string, "password": string}`
Response 200: `{"token": string, "user": User}`
Response 401: error

### GET /auth/me
Auth: any authenticated user.
Response 200: `User`

`User` shape (see `internal/domain.User`, `PasswordHash` never serialized):
```json
{"id":"uuid","username":"string","full_name":"string","role":"admin|reviewer|employee|auditor","employee_id":"uuid|null","is_active":true,"created_at":"...","updated_at":"..."}
```

## Employees

### GET /employees
Auth: admin, reviewer. Query: `q` (search name/personnel_no), `page`, `page_size`.
Response 200: `{"items": Employee[], "total": number}`

### POST /employees
Auth: admin.
Request: `{"personnel_no","full_name","national_id","hire_date","department","plan_id"}`
Response 201: `Employee`

### GET /employees/{id}
Auth: admin, reviewer, or the employee-role user whose `employee_id` matches `{id}`.
Response 200: `Employee`

### PATCH /employees/{id}
Auth: admin. Request: partial `{"employment_status","plan_id","department","full_name"}`.
Response 200: `Employee`

### GET /employees/{id}/dependents
Auth: admin, reviewer, or self. Response 200: `Dependent[]`

### POST /employees/{id}/dependents
Auth: admin. Request: `{"full_name","relation":"spouse|child|parent","national_id","birth_date"}`
Response 201: `Dependent`

### GET /employees/{id}/remaining-caps
Auth: admin, reviewer, or self.
Response 200: array of, one per service type:
```json
{"service_type_code":"dental","service_type_name":"Dental","coverage_percent":50,
 "per_claim_cap":3000000,"annual_cap":15000000,"used_annual":1200000,"remaining_annual":13800000}
```

`Employee` shape: see `internal/domain.Employee`. `Dependent` shape: see `internal/domain.Dependent`.

## Reference / config data

### GET /service-types
Auth: any authenticated user. Response 200: `ServiceType[]` (see `internal/domain.ServiceType`).

### POST /service-types
Auth: admin. Request: `{"code","name"}`. Response 201: `ServiceType`.
`code` must be unique, lowercase letters/digits/underscores (max 30). New types
appear in claim and coverage-rule dropdowns; they still need a coverage rule
before pricing accepts claims for them.

### GET /contracts
Auth: any authenticated user. Response 200: `InsuranceContract[]`.

### POST /contracts
Auth: admin. Request: `{"name","start_date","end_date","is_active"}`. Response 201: `InsuranceContract`.

### GET /plans
Auth: any authenticated user. Query: `contract_id` optional. Response 200: `CoveragePlan[]`.

### POST /plans
Auth: admin. Request: `{"contract_id","name","description"}`. Response 201: `CoveragePlan`.

### GET /coverage-rules
Auth: any authenticated user. Query: `plan_id`, `service_type_id` (both optional filters).
Response 200: `CoverageRule[]`, newest `effective_from` first — this is the full version history.

### POST /coverage-rules
Auth: admin. This is **the** config-driven policy-change endpoint: creating a new
row here changes benefits with no code deploy. Request:
```json
{"plan_id":"uuid","service_type_id":"uuid","coverage_percent":75,
 "per_claim_cap":2000000,"annual_cap":6000000,"waiting_period_days":30,
 "eligible_relations":["self","spouse"],"effective_from":"2026-01-01T00:00:00Z"}
```
Server behaviour: within one transaction, closes the previous open rule for the
same `(plan_id, service_type_id)` by setting its `effective_to = effective_from - 1 day`
— clamped to the old rule's own `effective_from` when the new version starts on the
same day, so the row constraint holds and the newest version wins that day — then
inserts the new row. Writes an `audit_logs` entry (`entity_type="coverage_rule"`,
`action="config_change"`, before/after = the two rule versions).
Response 201: `CoverageRule`.

`CoverageRule` shape: see `internal/domain.CoverageRule`.

## Claims

### POST /claims
Auth: employee (forces `employee_id` = caller's own), admin (may set any `employee_id`).
Request:
```json
{"employee_id":"uuid","beneficiary_type":"self|dependent","dependent_id":"uuid|null",
 "service_type_id":"uuid","requested_amount":400000,"receipt_date":"2026-01-10T00:00:00Z","description":"..."}
```
Server fills `plan_id` from the employee's current `plan_id` and sets `status="draft"`.
Response 201: `Claim`.

### GET /claims
Auth: any authenticated user. Employees see only their own claims (server filters
by their `employee_id`); reviewer/admin/auditor see all. Query: `status`,
`employee_id`, `service_type_id`, `from`, `to`, `page`, `page_size`.
Response 200: `{"items": Claim[], "total": number}`

### GET /claims/{id}
Auth: owner, reviewer, admin, auditor. Response 200: `Claim`.

### POST /claims/{id}/submit
Auth: owner or admin. draft -> submitted. Response 200: `Claim`.

### POST /claims/{id}/resubmit
Auth: owner or admin. returned_for_docs -> submitted. Response 200: `Claim`.

### POST /claims/{id}/start-review
Auth: reviewer, admin. submitted -> under_review. Response 200: `Claim`.

### POST /claims/{id}/approve
Auth: reviewer, admin. under_review -> approved. Runs the rule engine automatically
and sets `coverage_percent_applied` + `payable_amount`. Response 200: `Claim`.

### POST /claims/{id}/reject
Auth: reviewer, admin. Request: `{"reason": string}` (required). under_review -> rejected.
Response 200: `Claim`.

### POST /claims/{id}/return-for-docs
Auth: reviewer, admin. Request: `{"reason": string}` (required). under_review -> returned_for_docs.
Response 200: `Claim`.

### POST /claims/{id}/mark-paid
Auth: reviewer, admin. approved -> paid. Creates a simulated `Payment` row.
Response 200: `Claim`.

### POST /claims/{id}/close
Auth: reviewer, admin. rejected|paid -> closed. Response 200: `Claim`.

### GET /claims/{id}/history
Auth: owner, reviewer, admin, auditor. Response 200: `AuditLog[]` for this claim, newest first.

### GET /claims/{id}/attachments
Auth: owner, reviewer, admin, auditor (same read policy as the claim itself).
Response 200: `Attachment[]`, oldest first. `Attachment` carries `id`, `claim_id`,
`file_name` (the uploader's original name, for display only), `uploaded_at`, and
`download_url`. The storage key is deliberately never exposed.

### POST /claims/{id}/attachments
Auth: the claim's owner, or admin. `multipart/form-data` with a single `file` part.

Accepted only while the claim is in `draft` or `returned_for_docs` — once a claim
is back in the review queue its documents are frozen, so the evidence a reviewer
decided on cannot change underneath them. Any other status is 409
(`ErrAttachmentsFrozen`).

The file type is determined by **sniffing the content**, not by the declared
`Content-Type` or the extension: `application/pdf`, `image/jpeg`, `image/png`,
`image/webp`. Maximum 5 MiB.

Response 201: `Attachment`. Errors: 400 for an empty file
(`ErrAttachmentEmpty`), an oversized file (`ErrAttachmentTooLarge`), or an
unsupported type (`ErrAttachmentTypeNotOK`); 403 for anyone but the owner or an
admin; 409 if the claim's status does not accept documents.

Each successful upload writes an `attachment_upload` audit entry naming the
actor and the file, in the same transaction as the metadata row.

### GET /claims/{id}/attachments/{attachmentID}/download
Auth: same read policy as the claim. Responds with the file bytes,
`Content-Disposition: attachment` (RFC 5987-encoded so Persian filenames survive)
and `X-Content-Type-Options: nosniff`. An attachment id belonging to a different
claim is 404, not a cross-claim read.

Error codes used by all the above transition endpoints: 409 Conflict for an
invalid transition (`ErrInvalidTransition`), 403 Forbidden for a disallowed actor
role (`ErrForbidden`), 400 for a missing required reason (`ErrReasonRequired`), 422
if the rule engine cannot price the claim (no active rule, not eligible, waiting
period, inactive employee — surface the underlying `service/coverage` error message).

`Claim` shape: see `internal/domain.Claim`.

## Audit log

### GET /audit-logs
Auth: admin, auditor. Query: `entity_type`, `entity_id`, `actor_user_id`, `action`,
`from`, `to`, `page`, `page_size`.
Response 200: `{"items": AuditLog[], "total": number}`

`AuditLog` shape: see `internal/domain.AuditLog`.

## Reports

All under `/reports`, auth: admin, auditor. Query `from`, `to` (dates, optional).

### GET /reports/summary
```json
{"total_claims":120,"total_paid_amount":45000000,"pending_review":8,
 "approved_awaiting_payment":3,"rejected":10}
```

### GET /reports/spend-by-employee
```json
[{"employee_id":"uuid","employee_name":"...","personnel_no":"...","total_paid":1200000,"claim_count":4}]
```

### GET /reports/spend-by-service-type
```json
[{"service_type_code":"dental","service_type_name":"Dental","total_paid":8000000,"claim_count":15}]
```

### GET /reports/spend-by-month
```json
[{"month":"2026-01","total_paid":5000000,"claim_count":9}]
```

## Admin: users

### GET /admin/users
Auth: admin. Response 200: `User[]`.

### POST /admin/users
Auth: admin. Request: `{"username","password","full_name","role","employee_id"}`.
`role` may be `reviewer|employee|auditor` only — `admin` is rejected (403). The sole admin is created via seed / `make create-admin`.
Response 201: `User`.

### PATCH /admin/users/{id}
Auth: admin. Request (partial): `{"role","is_active","password"}`.
Cannot set `role` to `admin`, and cannot change the existing admin’s role away from `admin` (403). Password reset and activate/deactivate for admin remain allowed.
Response 200: `User`.

## Parent-system integration (API key, not JWT)

### POST /integration/employees/sync
Auth: `X-API-Key`. Bulk upsert by `personnel_no`.
Request: `{"employees": [{"personnel_no","full_name","national_id","employment_status","hire_date","department","plan_id"}]}`
Response 200: `{"created": number, "updated": number}`

### GET /integration/claims/{id}/status
Auth: `X-API-Key`. Response 200: `{"id":"uuid","status":"...","payable_amount":number|null}`

## Health

### GET /healthz
No auth. Response 200: `{"status":"ok"}`
