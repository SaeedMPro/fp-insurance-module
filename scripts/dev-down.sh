#!/usr/bin/env bash
# Stop everything started by scripts/dev-up.sh. The Postgres data directory is
# a throwaway cluster; pass --purge to delete it (and the demo data with it).
set -uo pipefail

RUN_DIR="${FP_RUN_DIR:-/tmp/fp-dev}"
PG_DIR="$RUN_DIR/pg"

stop_pid() {
  local name="$1" file="$2"
  [ -f "$file" ] || return 0
  local pid
  pid="$(cat "$file")"
  if kill -0 "$pid" 2>/dev/null; then
    # `go run` and `npm run dev` spawn children; kill the whole process group.
    kill -TERM "-$(ps -o pgid= "$pid" | tr -d ' ')" 2>/dev/null || kill -TERM "$pid" 2>/dev/null
    echo "stopped $name (pid $pid)"
  fi
  rm -f "$file"
}

stop_pid "vite dev server" "$RUN_DIR/web.pid"
stop_pid "api" "$RUN_DIR/api.pid"

# Belt and braces: the compiled `go run` child does not always share the pgid.
pkill -f 'exe/api' 2>/dev/null && echo "stopped stray api binary"

if [ -d "$PG_DIR/data" ]; then
  pg_ctl -D "$PG_DIR/data" stop >/dev/null 2>&1 && echo "stopped postgres"
fi

if [ "${1:-}" = "--purge" ]; then
  rm -rf "$RUN_DIR"
  echo "purged $RUN_DIR (demo data gone)"
fi

echo "done"
