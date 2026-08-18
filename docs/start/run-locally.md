# Run it locally

Two ways. Docker Compose is the normal one; a host-native script exists for
machines where Docker port publishing does not work.

## With Docker Compose

```bash
make up      # postgres + backend + frontend
make seed    # reference data and the demo dataset (once, on an empty database)
```

| | |
| --- | --- |
| Application | http://localhost:5173 |
| API | http://localhost:8080/api/v1 |
| Swagger UI | http://localhost:8080/swagger |
| Health | http://localhost:8080/healthz |

`make down` stops the stack, `make logs` follows it.

The schema (`backend/db/init.sql`) is applied by the backend on boot when the
database is empty. The demo dataset (`backend/db/seed.sql`) is deliberately
manual — you would not want it running against production.

## Without Docker

```bash
./scripts/dev-up.sh     # API on :18080, app on :5173, throwaway Postgres on :15432
./scripts/dev-down.sh
```

This provisions its own PostgreSQL cluster with `initdb` — no root, no Docker —
and is what to reach for if `make up` starts but nothing answers on the
published ports.

:::note
The script keeps its cluster and uploaded files under `/tmp`, which on most
Linux systems is a RAM disk. The stack therefore does not survive a reboot; just
run `dev-up.sh` again, which reseeds from scratch in a few seconds.
:::

## Demo accounts

| Username | Password | Role |
| --- | --- | --- |
| `admin` | `Admin123!` | Administrator |
| `reviewer` | `Reviewer123!` | Reviewer |
| `auditor` | `Auditor123!` | Auditor |
| `saeed.mazahery` | `Employee123!` | Employee — Standard plan |
| `farzin.hamzei` | `Employee123!` | Employee — Premium plan |

The administrator account cannot be created through the API, only by the seed or
`make create-admin`. That is enforced in the users service, not by a screen —
see [Roles and permissions](../how-it-works/permissions).

## What the demo dataset contains

Enough to exercise every path without setting anything up: two plans, eleven
coverage rules including one superseded version, seven employees, and twenty
claims spread across every workflow status — with priced payables, payments, an
audit trail, and two attached documents.

It also includes employees who exist to make the refusals demonstrable:

| Employee | Why they are there |
| --- | --- |
| P-1004 (hired recently) | Waiting period not yet served |
| P-1005 | Employment terminated — claims are refused |
| P-1006 | Active but assigned no plan — claims cannot be created |

## Running the tests

```bash
make test-integration   # unit + integration + contract, against a throwaway database
make lint               # golangci-lint and oxlint
make e2e                # 14-step browser suite against a running stack
```

:::warning
`make e2e` writes to whatever database it points at. It leaves claims, an
uploaded document and a new coverage-rule version behind, because publishing a
rule version is one of the things it proves. Do not run it against a database
you are about to demonstrate — reseed afterwards.
:::
