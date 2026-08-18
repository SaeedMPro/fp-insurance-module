# Roles and permissions

Authorisation happens at two levels, and the split is deliberate.

**Role, at the route.** *May this kind of user call this endpoint at all?*
Declared once per route group, where the whole surface can be read at a glance.

**Ownership, in the service.** *May this user touch this record?* That depends on
the record, so it lives with the business logic — an employee may submit only
claims they created, sees only their own claims in a list, and has the employee id
forced to their own record when creating one.

Putting ownership in the service rather than the handler means it holds no matter
what calls it: another endpoint, a background job, a future entry point.

## What each role can do

| | Employee | Reviewer | Admin | Auditor |
| --- | :--: | :--: | :--: | :--: |
| Create a claim | own only | — | any employee | — |
| Submit / resubmit | own only | — | any | — |
| List / view claims | own only | all | all | all |
| Upload documents | own, when open | — | any, when open | — |
| Read documents | own | all | all | all |
| Start review, approve, reject, return | — | ✓ | ✓ | — |
| Mark paid, close | — | ✓ | ✓ | — |
| List and view employees | own record | ✓ | ✓ | — |
| Create / edit employees, dependents | — | — | ✓ | — |
| Read reference data (plans, rules, service types) | ✓ | ✓ | ✓ | ✓ |
| Publish coverage rules | — | — | ✓ | — |
| Manage user accounts | — | — | ✓ | — |
| Audit log and reports | — | — | ✓ | ✓ |

An auditor is deliberately unable to change anything at all — not a claim, not a
rule, not an account. That is the point of the role.

## Two rules about the admin account

Enforced in the users service, not at the route, because they are invariants
rather than route permissions:

- **An admin cannot be created, demoted or deactivated through the API.** The
  only ways in are the seed and `make create-admin`. This prevents the API from
  being used to lock everyone out of administration.
- **An account with the `employee` role must be linked to an employee record.**
  An employee login with nothing to own would be able to create claims against
  nobody.

## The parent HR system

The two integration endpoints sit outside the role model entirely. They are gated
on an `X-API-Key` header matched against a stored SHA-256 hash, with no user
token and no role — they represent a trusted system, not a person.

See [HR integration](../reference/hr-integration).

## How this is verified

Route-level roles and service-level ownership are both covered by integration
tests, and the browser suite additionally checks the denials from the outside: an
employee is refused the user-administration endpoint, an anonymous request is
refused, and a reviewer is never offered an upload control on a claim they can
otherwise read in full.
