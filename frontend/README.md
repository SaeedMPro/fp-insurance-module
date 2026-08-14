# Frontend — Supplementary Insurance Module

React 19 + TypeScript + Vite single-page app for the Supplementary Insurance Module.
It talks to the Go REST API documented in [../docs/API-CONTRACT.md](../docs/API-CONTRACT.md).

See the [root README](../README.md) for the full project overview and how to run
everything together with Docker.

## Stack

- React 19 + TypeScript, React Router (role-guarded routes)
- Tailwind CSS (slate/blue palette, light + dark)
- axios (JWT attached via interceptor; global 401 → logout)
- Recharts (reports dashboards)

## Local development

```bash
npm install
npm run dev        # http://localhost:5173
```

Configure the API base URL at **runtime** (not at image build):

- Docker: set `API_BASE_URL` on the frontend service (default `/api/v1`, which
  nginx proxies to the backend). Example: `API_BASE_URL=http://localhost:8080/api/v1`
- Local Vite: edit `public/config.js`, or leave the default `/api/v1` and use the
  Vite proxy (`API_PROXY_TARGET`, default `http://localhost:8080`).

```bash
npm run build      # type-check (tsc -b) + production build to dist/
npm run preview    # serve the production build locally
```

## Structure

```
src/
  api/         axios client + typed wrappers per resource (types.ts = wire types)
  context/     Auth (JWT/session) and Toast providers
  components/  Layout (role-aware nav), route guards, shared UI (Card, StatusBadge, …)
  hooks/       small shared hooks (e.g. cached service types)
  lib/         formatting helpers (money, dates, status labels)
  pages/       one folder per area: claims, employees, employee, admin, auditor
  App.tsx      router: public /login + role-guarded authenticated routes
```

Roles (`admin`, `reviewer`, `employee`, `auditor`) determine both the visible
navigation and the accessible routes; the backend independently enforces the same
rules, so the client-side guards are convenience, not the security boundary.
