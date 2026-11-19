#!/bin/sh
# TICK-940 (S30 P11-08): insert cost of the shipped format-2 CHECK, masks
# CHECK, byte ceilings and tenant-id trigger (ADR-010 R6, ADR-010-A1 A1-R1).
# A throwaway postgres from docs/pins.md; TICK-629's functions and
# 20261119100000's functions, both read from db/migrations; then bench.sql
# once with documents sized to 32 KiB and once to 128 KiB. Writes
# result-32k.txt and result-128k.txt next to this script.
#   sh scratch/acl-format2-checks/bench.sh [cap ...]   # about 15 minutes for both
set -eu
export MSYS_NO_PATHCONV=1  # Git Bash on Windows: keep /bench paths as they are
here=$(cd "$(dirname "$0")" && pwd)
root="$here/../.."
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' "$root/docs/pins.md")
[ -n "$tag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
work=$(mktemp -d "$here/.work.XXXX")
for m in 20261109100000_acl_versions_shape_check 20261119100000_acl_rules_valid_format2; do
  sed '/^-- +goose Down/,$d; /^-- +goose/d; /^SET LOCAL/,$d' "$root/db/migrations/$m.sql" > "$work/$m.sql"
done
cp "$here/bench.sql" "$work/"
name=nava-tick940-bench
docker rm -f "$name" >/dev/null 2>&1 || true
docker run -d --name "$name" -e POSTGRES_PASSWORD=bench -v "$work:/bench" "postgres:$tag" >/dev/null
trap 'docker rm -f "$name" >/dev/null; rm -rf "$work"' EXIT
until docker exec "$name" pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
sleep 2
for cap in ${*:-32768 131072}; do
  out="$here/result-$((cap / 1024))k.txt"
  docker exec "$name" psql -U postgres -q -c 'DROP DATABASE IF EXISTS b' -c 'CREATE DATABASE b' >/dev/null
  {
    echo "cap=$cap; postgres:$tag; host: $(uname -srm); docker: $(docker info --format '{{.NCPU}} vCPUs, {{.MemTotal}} bytes'); other running containers: $(docker ps --format '{{.Names}}' | grep -cv "^$name\$")"
    docker exec "$name" psql -U postgres -d b -v ON_ERROR_STOP=1 -v cap=$cap -q \
        -f /bench/20261109100000_acl_versions_shape_check.sql \
        -f /bench/20261119100000_acl_rules_valid_format2.sql -f /bench/bench.sql
  } > "$out" 2>&1
  cat "$out"
done
