#!/bin/sh
# ADR-010-A1 (TICK-971): CHECK + tenant-id trigger cost of a 32 KiB format-2
# document at the SP-2 caps. Same harness as SP-2's
# scratch/acl-format2-caps/check.sh (TICK-933, PR #727): a throwaway postgres
# from docs/pins.md, TICK-629's functions, the SP-2 prototype
# docs/data/acl-format2-caps/format2_check.sql, then check32k_bench.sql.
# Writes check32k-result.txt next to this script.
#   sh docs/adr/ADR-010/A1/check32k.sh     # about 1 minute
set -eu
export MSYS_NO_PATHCONV=1  # Git Bash on Windows: keep /bench paths as they are
here=$(cd "$(dirname "$0")" && pwd)
root="$here/../../../.."
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' "$root/docs/pins.md")
[ -n "$tag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
work=$(mktemp -d "$here/.work.XXXX")
sed '/^-- +goose Down/,$d; /^-- +goose/d; /^SET LOCAL/,$d' \
    "$root/db/migrations/20261109100000_acl_versions_shape_check.sql" > "$work/m1.sql"
cp "$root/docs/data/acl-format2-caps/format2_check.sql" "$here/check32k_bench.sql" "$work/"
name=nava-tick971-check32k
docker rm -f "$name" >/dev/null 2>&1 || true
docker run -d --name "$name" -e POSTGRES_PASSWORD=bench -v "$work:/bench" "postgres:$tag" >/dev/null
trap 'docker rm -f "$name" >/dev/null; rm -rf "$work"' EXIT
until docker exec "$name" pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
sleep 2
{
  echo "postgres:$tag; host: $(uname -srm); docker: $(docker info --format '{{.NCPU}} vCPUs, {{.MemTotal}} bytes'); other running containers: $(docker ps --format '{{.Names}}' | grep -cv "^$name\$")"
  docker exec "$name" psql -U postgres -v ON_ERROR_STOP=1 -q \
      -f /bench/m1.sql -f /bench/format2_check.sql -f /bench/check32k_bench.sql
} > "$here/check32k-result.txt" 2>&1
cat "$here/check32k-result.txt"
