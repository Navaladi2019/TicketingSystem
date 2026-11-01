#!/bin/sh
# P3-09 (TICK-353) evidence: a throwaway postgres from docs/pins.md, the real
# migration body applied to a records table of PLAN-001 bare rows.
set -eu
here=$(cd "$(dirname "$0")" && pwd)
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' "$here/../../docs/pins.md")
[ -n "$tag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
sed '/^-- +goose/d' "$here/../../db/migrations/20261101190000_records_pipeline.sql" > "$here/migration.sql"
name=nava-p309-bench
docker rm -f "$name" >/dev/null 2>&1 || true
MSYS_NO_PATHCONV=1 docker run -d --name "$name" --shm-size=1g -e POSTGRES_PASSWORD=bench -v "$here:/bench" "postgres:$tag" >/dev/null
trap 'docker rm -f "$name" >/dev/null; rm -f "$here/migration.sql"' EXIT
until docker exec "$name" pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
sleep 2
MSYS_NO_PATHCONV=1 docker exec "$name" psql -U postgres -v ON_ERROR_STOP=1 -q -f /bench/bench.sql
