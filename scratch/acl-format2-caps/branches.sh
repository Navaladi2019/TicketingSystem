#!/bin/sh
# SP-2 (TICK-933) item 3: the ReadBranches total-branch threshold. Seeds
# bench/records-list's 800k records (P3-01: 00_roles, 01_schema, 02_seed) on
# a throwaway postgres from docs/pins.md with P3-01's memory settings, then
# runs branches_bench.sql. Writes branches-result.txt next to this script.
#   sh scratch/acl-format2-caps/branches.sh     # about 15 minutes
# BENCH_KEEP=1 keeps the container.
set -eu
export MSYS_NO_PATHCONV=1
here=$(cd "$(dirname "$0")" && pwd)
# docker cp on Windows needs C:/... rather than Git Bash's /c/...
host=$(cd "$here" && { pwd -W 2>/dev/null || pwd; })
root="$here/../.."
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' "$root/docs/pins.md")
[ -n "$tag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
name=nava-tick933-branches
docker rm -f "$name" >/dev/null 2>&1 || true
docker run -d --name "$name" --shm-size=3g -e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=bench "postgres:$tag" \
    -c shared_buffers=2GB -c effective_cache_size=6GB -c max_wal_size=8GB -c track_io_timing=on >/dev/null
cleanup() { [ "${BENCH_KEEP:-0}" = 1 ] && { echo "kept: $name"; return; }; docker rm -f "$name" >/dev/null 2>&1 || true; }
trap cleanup EXIT
until docker exec "$name" pg_isready -U pgadmin -d postgres >/dev/null 2>&1; do sleep 1; done
sleep 2
until docker exec "$name" pg_isready -U pgadmin -d postgres >/dev/null 2>&1; do sleep 1; done
docker exec "$name" mkdir -p /bench
for f in 00_roles.sql 01_schema.sql 02_seed.sql; do docker cp "$host/../../bench/records-list/$f" "$name:/bench/$f"; done
docker cp "$host/branches_bench.sql" "$name:/bench/branches_bench.sql"
su() { docker exec -i "$name" psql -X -q -v ON_ERROR_STOP=1 -U pgadmin -d "${2:-nava}" -f "/bench/$1"; }
{
  echo "postgres:$tag; host: $(uname -srm); docker: $(docker info --format '{{.NCPU}} vCPUs, {{.MemTotal}} bytes'); other database containers: $(docker ps --format '{{.Names}}' | grep -E 'postgres|-pg' | grep -v "^$name\$" | tr '\n' ' ')"
  su 00_roles.sql postgres
  su 01_schema.sql
  su 02_seed.sql
  su branches_bench.sql
} > "$here/branches-result.txt" 2>&1
cat "$here/branches-result.txt"
