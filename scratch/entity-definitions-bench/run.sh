#!/bin/sh
# P3-02 plan evidence: a throwaway postgres from docs/pins.md, the real
# migration body.
set -eu
here=$(cd "$(dirname "$0")" && pwd)
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' "$here/../../docs/pins.md")
[ -n "$tag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
sed '/^-- +goose/d' "$here/../../db/migrations/20261031150000_entity_definitions.sql" > "$here/migration.sql"
name=nava-p302-bench
docker rm -f "$name" >/dev/null 2>&1 || true
docker run -d --name "$name" -e POSTGRES_PASSWORD=bench -v "$here:/bench" "postgres:$tag" >/dev/null
trap 'docker rm -f "$name" >/dev/null; rm -f "$here/migration.sql"' EXIT
until docker exec "$name" pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
sleep 2
docker exec "$name" psql -U postgres -v ON_ERROR_STOP=1 -q -f /bench/bench.sql
