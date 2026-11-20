#!/bin/sh
# ADR-010-A1 rev 1 (TICK-971): CHECK + masks CHECK + tenant-id trigger cost
# at the SP-2 caps with the A1-2 size caps at 32 KiB (the default) and
# 128 KiB (the ceiling). Same harness as SP-2's
# scratch/acl-format2-caps/check.sh (TICK-933, PR #727): a throwaway postgres
# from docs/pins.md, TICK-629's functions, the SP-2 prototype
# docs/data/acl-format2-caps/format2_check.sql, then cap_bench.sql once per cap.
# Writes cap-32k-result.txt and cap-128k-result.txt next to this script (the
# committed runs are renamed -runA and -runB; run A used rev 0's constraint
# names, so only its refusal rows differ).
#   sh docs/adr/ADR-010/A1/cap_bench.sh     # about 15 minutes
set -eu
export MSYS_NO_PATHCONV=1  # Git Bash on Windows: keep /bench paths as they are
here=$(cd "$(dirname "$0")" && pwd)
root="$here/../../../.."
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' "$root/docs/pins.md")
[ -n "$tag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
work=$(mktemp -d "$here/.work.XXXX")
sed '/^-- +goose Down/,$d; /^-- +goose/d; /^SET LOCAL/,$d' \
    "$root/db/migrations/20261109100000_acl_versions_shape_check.sql" > "$work/m1.sql"
cp "$root/docs/data/acl-format2-caps/format2_check.sql" "$here/cap_bench.sql" "$work/"
name=nava-tick971-capbench
docker rm -f "$name" >/dev/null 2>&1 || true
docker run -d --name "$name" -e POSTGRES_PASSWORD=bench -v "$work:/bench" "postgres:$tag" >/dev/null
trap 'docker rm -f "$name" >/dev/null; rm -rf "$work"' EXIT
until docker exec "$name" pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
sleep 2
for cap in 32768 131072; do
  out="$here/cap-$((cap / 1024))k-result.txt"
  docker exec "$name" psql -U postgres -q -c 'DROP DATABASE IF EXISTS b' -c 'CREATE DATABASE b' >/dev/null
  {
    echo "cap=$cap; postgres:$tag; host: $(uname -srm); docker: $(docker info --format '{{.NCPU}} vCPUs, {{.MemTotal}} bytes'); other running containers: $(docker ps --format '{{.Names}}' | grep -cv "^$name\$")"
    docker exec "$name" psql -U postgres -d b -v ON_ERROR_STOP=1 -v cap=$cap -q \n        -f /bench/m1.sql -f /bench/format2_check.sql -f /bench/cap_bench.sql
  } > "$out" 2>&1
  cat "$out"
done
