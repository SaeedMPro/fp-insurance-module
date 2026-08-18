# Configuration

Everything environment-dependent is read in one place, each value with a
development-safe default.

| Variable | Default | Purpose |
| --- | --- | --- |
| `APP_ENV` | `development` | Switches on the production safety checks below |
| `HTTP_PORT` | `8080` | Port the API listens on |
| `DATABASE_URL` | local dev DSN | PostgreSQL connection string |
| `JWT_SECRET` | insecure placeholder | HS256 signing key |
| `JWT_TTL` | 8h | Token lifetime |
| `DB_INIT_PATH` | `db/init.sql` | Schema applied on boot when the database is empty |
| `ATTACHMENTS_DIR` | `data/attachments` | Root of the document store |
| `CORS_ORIGIN` | `http://localhost:5173` | Allowed browser origin |

## Production refuses to start when misconfigured

With `APP_ENV=production`, startup fails rather than continuing, if:

- `JWT_SECRET` is the default placeholder, or is too short
- `DATABASE_URL` was not set explicitly
- `JWT_TTL` is not positive

The reasoning is that a service running on a default signing key does not
announce itself — it is discovered by being exploited. Failing loudly at boot is
the cheaper outcome. This behaviour has its own tests.

## The frontend

The API base URL is a **runtime** value, not baked in at build time. The
container writes `/config.js` from `API_BASE_URL` (default `/api/v1`), so one
built image can be pointed at any backend.

| Variable | Where | Purpose |
| --- | --- | --- |
| `API_BASE_URL` | container start | Written into `/config.js` |
| `API_PROXY_TARGET` | Vite dev server | Where `/api/v1` is proxied in development |

## Storage that must persist

Claim documents are files, not database rows. In Compose the attachments
directory is backed by its own named volume — without it, uploads would be lost
whenever the container is replaced. See [Claim documents](../how-it-works/documents).

## The demo integration key

The seed installs an API key whose plaintext is `dev-integration-key`, stored as
a SHA-256 hash. It is for development only.
