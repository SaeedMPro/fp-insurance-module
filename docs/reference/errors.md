# Errors

Every error response has the same shape:

```json
{ "error": "a human-readable message" }
```

## How the status code is chosen

Services never pick HTTP status codes. They return a domain error carrying a
*kind*, and the transport layer maps kind to status in exactly one place.

| Kind | Status | When |
| --- | --- | --- |
| `Validation` | 400 | Malformed or missing input |
| `Unauthorized` | 401 | No credentials, or invalid ones |
| `Forbidden` | 403 | Authenticated, but not allowed |
| `NotFound` | 404 | The entity does not exist |
| `Conflict` | 409 | Well-formed, but wrong state — an illegal transition |
| `Unprocessable` | 422 | Well-formed, but the business rules refuse it |
| `Internal` | 500 | Unclassified; no detail is returned |

The distinction between 409 and 422 is worth stating: **409** means *not now*
(approving a draft), **422** means *not ever, as asked* (a claim that cannot be
priced because no rule covers its date).

## Business errors by area

### Claim workflow

| Message | Status |
| --- | --- |
| `transition not allowed from the claim's current status` | 409 |
| `a reason is required for this action` | 400 |
| `actor is not permitted to perform this action` | 403 |
| `claim has no payable amount to disburse` | 409 |

### Pricing and eligibility

All 422 — the request is valid, the rules refuse it.

| Message | Cause |
| --- | --- |
| `no active coverage rule for this plan/service type on the receipt date` | Often a receipt dated before the contract began |
| `beneficiary is not eligible for this service under the current rule` | The relation is not in the rule's eligible list |
| `employee has not completed the required waiting period` | Counted in Tehran civil days from the hire date |
| `employee is not active` | Employment terminated |
| `dependent does not belong to this employee` | Valid dependent, wrong employee |
| `employee has no coverage plan assigned` | Refused at claim creation |

### Documents

| Message | Status |
| --- | --- |
| `documents can only be added while the claim is a draft or has been returned for documents` | 409 |
| `unsupported file type` | 400 — sniffed content is not PDF/JPEG/PNG/WebP |
| `file exceeds the maximum allowed size` | 400 |
| `file is empty` | 400 |
| `attachment file not found` | 404 — the row exists, the file does not |

### Accounts

| Message | Status |
| --- | --- |
| `admin accounts cannot be created via the API; use seed or make create-admin` | 400 |
| `cannot assign the admin role via the API` | 400 |
| `the admin account cannot be deactivated via the API` | 400 |
| `employee role requires a linked employee record` | 400 |
| `password must be at least 8 characters` | 400 |

## A note on language

The API answers in **English**, deliberately: it is a machine-facing contract
that the parent HR system also consumes, and the messages double as log text.

The Persian interface translates them at the presentation boundary, with a
pass-through fallback — an untranslated message degrades to English rather than
to a useless generic string. Every message listed above has a Persian
translation.
