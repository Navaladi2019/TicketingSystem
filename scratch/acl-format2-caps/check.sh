#!/bin/sh
# SP-2 (TICK-933) items 1-2: format-2 CHECK + tenant-id trigger insert cost at
# the D4 caps. Throwaway postgres from docs/pins.md; loads the functions of
# the TICK-629 migration and the prototype
# docs/data/acl-format2-caps/format2_check.sql, then check_bench.sql.
# Writes check-result.txt next to this script.
#   sh scratch/acl-format2-caps/check.sh     # about 2 minutes
set -eu
export MSYS_NO_PATHCONV=1  # Git Bash on Windows: keep /bench paths as they are
here=$(cd "$(dirname "$0")" && pwd)
root="$here/../.."
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' "$root/docs/pins.md")
[ -n "$tag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
work=$(mktemp -d "$here/.work.XXXX")
# Only the functions (no ALTER TABLE: the bench adds its own CHECK).
sed '/^-- +goose Down/,$d; /^-- +goose/d; /^SET LOCAL/,$d' \
    "$root/db/migrations/20261109100000_acl_versions_shape_check.sql" > "$work/m1.sql"
cp "$root/docs/data/acl-format2-caps/format2_check.sql" "$here/check_bench.sql" "$work/"
name=nava-tick933-check
docker rm -f "$name" >/dev/null 2>&1 || true
docker run -d --name "$name" -e POSTGRES_PASSWORD=bench -v "$work:/bench" "postgres:$tag" >/dev/null
trap 'docker rm -f "$name" >/dev/null; rm -rf "$work"' EXIT
until docker exec "$name" pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
sleep 2
{
  echo "postgres:$tag; host: $(uname -srm); docker: $(docker info --format '{{.NCPU}} vCPUs, {{.MemTotal}} bytes'); other running containers: $(docker ps --format '{{.Names}}' | grep -cv "^$name\$")"
  docker exec "$name" psql -U postgres -v ON_ERROR_STOP=1 -q \
      -f /bench/m1.sql -f /bench/format2_check.sql -f /bench/check_bench.sql
} > "$here/check-result.txt" 2>&1
cat "$here/check-result.txt"
