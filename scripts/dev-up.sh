#!/usr/bin/env bash
# Bring the whole project up WITHOUT Docker: a throwaway host-native Postgres
# cluster, the Go API, and the Vite dev server.
#
# Why this exists alongside `make up` (docker compose): on hosts where Docker's
# host->container port publishing is unavailable (e.g. after a suspend that
# loses the daemon's firewall rules, with no root access to restart it), the
# compose stack builds and runs but is unreachable from the browser. This path
# has no such dependency and doubles as the fastest inner-loop for development
# (Vite HMR, no image rebuilds).
#
# Usage:  ./scripts/dev-up.sh            # start everything, apply db/seed.sql
#         ./scripts/dev-down.sh          # stop everything
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_DIR="${FP_RUN_DIR:-/tmp/fp-dev}"
PG_DIR="$RUN_DIR/pg"
PG_PORT="${FP_PG_PORT:-15432}"
API_PORT="${FP_API_PORT:-18080}"
WEB_PORT="${FP_WEB_PORT:-5173}"
DSN="postgres://insurance@127.0.0.1:$PG_PORT/insurance?sslmode=disable"

mkdir -p "$RUN_DIR"

say() { printf '\n\033[1m==> %s\033[0m\n' "$1"; }

# ---------------------------------------------------------------- postgres ---
if pg_isready -h 127.0.0.1 -p "$PG_PORT" -U insurance >/dev/null 2>&1; then
  say "postgres already running on :$PG_PORT"
else
  say "starting postgres on :$PG_PORT (throwaway cluster in $PG_DIR)"
  rm -rf "$PG_DIR"
  mkdir -p "$PG_DIR/data" "$PG_DIR/sock"
  initdb -D "$PG_DIR/data" -U insurance --auth=trust >/dev/null
  # -k puts the unix socket inside the throwaway dir; the system default
  # (/run/postgresql) is not writable for a non-root cluster.
  pg_ctl -D "$PG_DIR/data" -l "$PG_DIR/log" \
    -o "-p $PG_PORT -k $PG_DIR/sock -c listen_addresses=127.0.0.1" start >/dev/null
  until pg_isready -h 127.0.0.1 -p "$PG_PORT" -U insurance >/dev/null 2>&1; do sleep 1; done
  createdb -h 127.0.0.1 -p "$PG_PORT" -U insurance insurance
fi

# ----------------------------------------------------------------- backend ---
say "starting API on :$API_PORT (applies db/init.sql on boot)"
cd "$ROOT/backend"
DATABASE_URL="$DSN" \
JWT_SECRET="${JWT_SECRET:-dev-secret-not-for-production}" \
DB_INIT_PATH="db/init.sql" \
ATTACHMENTS_DIR="$RUN_DIR/attachments" \
HTTP_PORT="$API_PORT" \
CORS_ORIGIN="http://localhost:$WEB_PORT" \
  nohup go run ./cmd/api >"$RUN_DIR/api.log" 2>&1 &
echo $! >"$RUN_DIR/api.pid"

for _ in $(seq 1 60); do
  if curl -sf -m 2 "http://localhost:$API_PORT/healthz" >/dev/null 2>&1; then break; fi
  sleep 1
done
if ! curl -sf -m 2 "http://localhost:$API_PORT/healthz" >/dev/null 2>&1; then
  echo "API failed to start; last log lines:" >&2
  tail -20 "$RUN_DIR/api.log" >&2
  exit 1
fi

say "applying reference seed (db/seed.sql)"
# Skip if service types already present (re-running the script).
if [[ "$(psql "$DSN" -Atc 'SELECT COUNT(*) FROM service_types')" == "0" ]]; then
  psql "$DSN" -v ON_ERROR_STOP=1 -f "$ROOT/backend/db/seed.sql"
else
  say "reference seed already present; skipping"
fi

# The seeded attachment rows describe files that no upload ever created. Write a
# small valid PDF for each missing one so the demo's download button works;
# real uploads land in the same directory alongside them.
say "materialising demo attachment files"
ATT_DIR="$RUN_DIR/attachments"
psql "$DSN" -Atc 'SELECT file_path FROM claim_attachments' | while read -r key; do
  [ -n "$key" ] || continue
  target="$ATT_DIR/$key"
  [ -f "$target" ] && continue
  mkdir -p "$(dirname "$target")"
  printf '%%PDF-1.4\n1 0 obj<</Type/Catalog>>endobj\ntrailer<</Root 1 0 R>>\n%%%%EOF\n' >"$target"
done

# ---------------------------------------------------------------- frontend ---
say "starting Vite dev server on :$WEB_PORT"
cd "$ROOT/frontend"
[ -d node_modules ] || npm install --silent
API_PROXY_TARGET="http://localhost:$API_PORT" \
  nohup npm run dev -- --port "$WEB_PORT" --strictPort >"$RUN_DIR/web.log" 2>&1 &
echo $! >"$RUN_DIR/web.pid"

for _ in $(seq 1 45); do
  if curl -sf -m 2 "http://localhost:$WEB_PORT/" >/dev/null 2>&1; then break; fi
  sleep 1
done

cat <<EOF

  ✅ up and running

     App   http://localhost:$WEB_PORT
     API   http://localhost:$API_PORT/api/v1   (health: /healthz)

     Reference data loaded from backend/db/seed.sql
     (demo users, claims, reports sample — see README).

     logs   $RUN_DIR/api.log   $RUN_DIR/web.log
     stop   ./scripts/dev-down.sh
EOF
