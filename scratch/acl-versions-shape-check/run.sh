#!/bin/sh
# TICK-629 evidence: the cost of the rules / masks shape CHECKs on
# acl_versions, on a throwaway postgres from docs/pins.md with the real
# migration bodies. Writes result.txt next to this script.
#
# The table is acl_versions' columns without its foreign keys and RLS: the
# CHECK cost does not depend on them, and the FKs would need the whole
# platform schema. The migration files are applied unchanged (goose markers
# stripped, SET LOCAL made session-wide as psql runs without a transaction).
set -eu
export MSYS_NO_PATHCONV=1  # Git Bash on Windows: keep /bench paths as they are
here=$(cd "$(dirname "$0")" && pwd)
mig="$here/../../db/migrations"
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' "$here/../../docs/pins.md")
[ -n "$tag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
sed '/^-- +goose Down/,$d; /^-- +goose/d; s/SET LOCAL/SET/' "$mig/20261109100000_acl_versions_shape_check.sql" > "$here/m1.sql"
sed '/^-- +goose Down/,$d; /^-- +goose/d; s/SET LOCAL/SET/' "$mig/20261109100100_acl_versions_shape_check_validate.sql" > "$here/m2.sql"
name=nava-tick629-bench
docker rm -f "$name" >/dev/null 2>&1 || true
docker run -d --name "$name" -e POSTGRES_PASSWORD=bench -v "$here:/bench" "postgres:$tag" >/dev/null
trap 'docker rm -f "$name" >/dev/null; rm -f "$here/m1.sql" "$here/m2.sql"' EXIT
until docker exec "$name" pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
sleep 2
{
  echo "postgres:$tag; host: $(uname -srm); docker: $(docker info --format '{{.NCPU}} vCPUs, {{.MemTotal}} bytes')"
  docker exec "$name" psql -U postgres -v ON_ERROR_STOP=1 -q -f /bench/bench.sql
} > "$here/result.txt" 2>&1
cat "$here/result.txt"
