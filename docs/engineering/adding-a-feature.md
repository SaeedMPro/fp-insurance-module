# Adding a feature

The order the layers are built in, traced through a feature that was genuinely
added this way: **claim document upload**.

The database table already existed and was seeded, but nothing read or wrote it —
so the return-for-documents flow had no way to actually supply documents.

## 1. Decide where the rule lives

Before any code: *whose rule is this?*

Uploading looked like a file concern, but the rule that governs it is a **claim**
rule — documents may only be added while a claim is a draft or has been returned
for documents. So it belongs in the claims service, next to the state machine,
not in a new "attachments" service.

Getting this wrong is the expensive mistake. Everything else is mechanical.

## 2. Write the spec first

Add the paths, the schema and the responses to `backend/api/openapi.yaml`. The
contract test will fail until the routes exist, which is the correct order — the
document leads.

Deliberately excluded from the response schema: the storage key. Clients get the
original filename and a download URL; where the bytes live is not their business.

## 3. Domain

The entity, plus the *policy as a predicate* — a method on the claim answering
whether it accepts documents, and the list of statuses that do. Putting the
predicate on the entity means both the service and the transport layer can ask
the same question without either owning the answer.

## 4. Platform, if new infrastructure is needed

Documents are files, so this feature needed a blob store — a small package with
`Save`, `Open`, `Remove`, server-generated UUID filenames, and key resolution that
refuses anything absolute or escaping the root. It has no business meaning, so it
lives under `platform/` and has its own hostile-input tests.

## 5. Storage

A row type with its column tags, the mappers to and from the domain entity, and
the repository methods. This is the only layer where the ORM appears.

## 6. Service — the policy

The interesting layer:

- the permission check (owner or admin),
- the freeze check via the entity's predicate,
- content sniffing on the actual bytes rather than the declared type,
- the size limit,
- and the write ordering: blob first, then row and audit entry in one
  transaction, with the blob removed if that transaction fails.

The service also declares the interface it needs from the file store, so it can
be read and tested without the platform package in view.

## 7. Transport

Thin: parse the multipart form, call the service, encode the result. Plus the
delivery details that only matter at the HTTP boundary — `Content-Disposition`
RFC 5987-encoded so Persian filenames survive, and `nosniff`.

## 8. Wire it up

Add the routes to the existing role group, and the dependency to the composition
root — which both `cmd/api` and the integration tests use, so tests exercise
production wiring.

## 9. Tests, at the layer that owns the behaviour

| Layer | What it tests |
| --- | --- |
| Platform unit | Traversal and absolute keys cannot escape the store |
| Service integration | The round trip, the audit entry, the freeze across the whole returned-for-documents loop, a disguised file type, and the access boundaries |
| Contract | Automatically — the spec now covers routes that exist |
| Browser | The employee actually uploading through the Persian UI, and the control disappearing on resubmission |

## 10. Frontend

Generate the types (`npm run gen:api`), then the API module and the component.
Show the control only when the API would accept the action — the server enforces
it either way, but offering a button that will be refused is a poor interface.

## 11. Everything the change makes untrue

The step most easily skipped. This feature falsified claims in six documents,
including a report that listed document upload under "not implemented". Grep for
what the change contradicts: counts of routes and screens, "not yet built"
statements, and configuration tables.

## Checklist

```
[ ] Decide which service owns the rule
[ ] OpenAPI document
[ ] Domain entity + policy predicate
[ ] platform/ package, if new infrastructure
[ ] storage/postgres row, mappers, repository methods
[ ] Service: permissions, policy, transaction boundaries
[ ] transport/http: DTO, handler, route in the right role group
[ ] app: wire the dependency
[ ] Tests at each owning layer
[ ] npm run gen:api, then the UI
[ ] Update whatever the change made untrue
```
