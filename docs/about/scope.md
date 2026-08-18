# Scope and limits

What this system deliberately does not do. Stating the boundary is more useful
than leaving a reader to wonder whether something is missing or merely unfinished.

## Out of scope by design

**A real payment gateway.** `mark-paid` records a payment with a generated `SIM-`
reference. There is no bank integration. The workflow up to and including the
payment decision is complete; the disbursement itself is simulated.

**A live connection to HR and finance systems.** Instead of a live integration
there is an authenticated API seam with two endpoints — see
[HR integration](../reference/hr-integration). Defining and securing the seam was
in scope; connecting it to a specific vendor's system was not.

**Other benefit modules.** Loans, guest housing and organisational housing were
designed to sit alongside this module rather than inside it.

**Laboratory services.** Out of scope from the start.

## Known limitations

Real gaps rather than deliberate exclusions.

| | |
| --- | --- |
| **Monthly reports group by Gregorian month** | The query uses `TO_CHAR(receipt_date,'YYYY-MM')`, so the monthly chart's axis is Gregorian while the rest of the interface is Jalali. Visible on the reports screen. |
| **No notification system** | An employee learns their claim was decided by looking. Email or SMS on status change is the most obviously missing feature for real use. |
| **No load testing** | Behaviour under many concurrent users and a large claim history is unmeasured. |
| **No two-factor authentication** | Worth having given the sensitivity of medical data. |
| **Single approval level** | Some organisations need a manager's approval alongside a reviewer's. |
| **No bulk employee import from file** | Initial data entry and migration from a previous system would want one. The API sync partly covers this. |
| **Audit trail is not tamper-proof** | The application cannot produce an unrecorded change, but anyone with direct database access can edit the table. There is no hash chain. |

## On the security posture

Worth being precise about, since medical claim data is sensitive.

What is in place: bcrypt password hashing, signed tokens with a lifetime,
two-level authorisation, API keys stored only as hashes, content sniffing on
uploads, path-traversal-safe file storage, parameterised queries throughout, and a
production configuration that refuses to start on a default signing key.

What is not: encryption at rest, a documented and tested backup procedure,
operational monitoring, penetration testing, or any formal compliance
certification. A real deployment handling real medical data needs all of those,
and none of them are claimed here.
