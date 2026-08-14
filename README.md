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
make up      # postgres + backend + frontend
make seed    # reference data + demo users (once)
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
| `admin` | `Admin123!` | admin |
| `reviewer` | `Reviewer123!` | reviewer |
| `auditor` | `Auditor123!` | auditor |
| `saeed.mazahery` | `Employee123!` | employee |
| `farzin.hamzei` | `Employee123!` | employee |

## Layout

```
backend/     Go API, OpenAPI, db/init.sql + db/seed.sql
frontend/    Persian RTL SPA
docs/        architecture, ERD, API contract, use cases
docker-compose.yml
Makefile
```

## Docs

- [docs/API-CONTRACT.md](docs/API-CONTRACT.md)
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- [docs/ERD.md](docs/ERD.md)
- [docs/USE-CASES.md](docs/USE-CASES.md)
- Spec: [backend/api/openapi.yaml](backend/api/openapi.yaml)
