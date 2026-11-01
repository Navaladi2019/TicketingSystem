#!/bin/sh
# TICK-378 evidence: the cost of the fixed_columns / states CHECKs on
# entity_definition_versions, on a throwaway postgres from docs/pins.md with
# the real migration bodies. Writes result.txt next to this script.
set -eu
export MSYS_NO_PATHCONV=1  # Git Bash on Windows: keep /bench paths as they are
here=$(cd "$(dirname "$0")" && pwd)
mig="$here/../../db/migrations"
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' "$here/../../docs/pins.md")
[ -n "$tag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
sed '/^-- +goose/d' "$mig/20261031150000_entity_definitions.sql" > "$here/m1.sql"
sed '/^-- +goose/d; s/SET LOCAL/SET/' "$mig/20261101180000_entity_definition_versions_jsonb_check.sql" > "$here/m2.sql"
sed '/^-- +goose/d; s/SET LOCAL/SET/' "$mig/20261101180100_entity_definition_versions_jsonb_check_validate.sql" > "$here/m3.sql"
name=nava-tick378-bench
docker rm -f "$name" >/dev/null 2>&1 || true
docker run -d --name "$name" -e POSTGRES_PASSWORD=bench -v "$here:/bench" "postgres:$tag" >/dev/null
trap 'docker rm -f "$name" >/dev/null; rm -f "$here/m1.sql" "$here/m2.sql" "$here/m3.sql"' EXIT
until docker exec "$name" pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
sleep 2
{
  echo "postgres:$tag; host: $(uname -srm)"
  docker exec "$name" psql -U postgres -v ON_ERROR_STOP=1 -q -f /bench/bench.sql
} > "$here/result.txt" 2>&1
cat "$here/result.txt"
