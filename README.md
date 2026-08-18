# Supplementary Insurance Module

Persian RTL web app for employee supplementary (top-up) health insurance.

Coverage rules (percent, caps, waiting period) live in the database as versioned
config — change policy via API, not code. Claims follow
`draft → submitted → under_review → approved/rejected → paid → closed`, with
audit trail, JWT RBAC (`admin` / `reviewer` / `employee` / `auditor`), reports,
and an API-key seam for a parent HR system.

Capstone project, Bu-Ali Sina University — طراحی و پیاده‌سازی ماژول بیمه تکمیلی.

**Stack:** Go + chi + Postgres · React + TypeScript + Vite · Docker Compose

## Run

```bash
make up           # postgres + backend + frontend
make seed         # reference data + demo users (once)
make create-admin # sole admin if missing (defaults: admin / Admin123!)
```

| | URL |
|---|---|
| App | http://localhost:5173 |
| API | http://localhost:8080/api/v1 |
| Swagger | http://localhost:8080/swagger |
| Health | http://localhost:8080/healthz |

`make down` stops the stack · `make logs` · `make test`

Schema (`db/init.sql`) applies on API boot. Seed (`db/seed.sql`) is manual.

### Demo logins

| User | Password | Role |
|---|---|---|
| `admin` | `Admin123!` | admin (seed / `make create-admin` only; API cannot create or promote to admin) |
| `reviewer` | `Reviewer123!` | reviewer |
| `auditor` | `Auditor123!` | auditor |
| `saeed.mazahery` | `Employee123!` | employee (P-1001، طرح استاندارد) |
| `farzin.hamzei` | `Employee123!` | employee (P-1002، طرح ویژه) |

`make seed` loads a full demo dataset: contracts/plans/rules, HR roster + dependents,
claims across every workflow status (with priced payables + payments), audit trail,
and integration key `dev-integration-key` (`X-API-Key`).

## Layout

```
backend/     Go API, OpenAPI, db/init.sql + db/seed.sql
frontend/    Persian RTL SPA
e2e/         browser end-to-end suite
docs/        documentation content (also the source of the docs site)
website/     Docusaurus site that publishes docs/ to GitHub Pages
scripts/     run the stack locally without Docker
docker-compose.yml
Makefile
```

## Docs

Published at **https://SaeedMPro.github.io/fp-insurance-module/** — 23 pages plus
an API reference generated from the OpenAPI document.

The Markdown in [docs/](docs/) *is* the site's source, so there is one copy of
every page and it stays readable here on GitHub.

- [Overview](docs/index.md) · [The one idea](docs/start/the-one-idea.md) · [Run it locally](docs/start/run-locally.md)
- [Architecture](docs/engineering/architecture.md) · [Design decisions](docs/engineering/decisions.md)
- [Database schema](docs/reference/database.md) · Spec: [backend/api/openapi.yaml](backend/api/openapi.yaml)

Building the site locally:

```bash
cd website && npm ci && npm start     # dev server
cd website && npm run build           # production build; fails on broken links
```
