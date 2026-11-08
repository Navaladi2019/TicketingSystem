#!/bin/sh
# P5-15 (TICK-433) backfill timing. Needs Docker. Usage: run.sh ["tenants types versions" ...]
# Starts postgres (tag from docs/pins.md), applies the real audit_log and
# backfill migrations on stub parents, then per size loads synthetic S8
# history and times the backfill, its re-run and the SHA-1 alone.
set -eu
cd "$(dirname "$0")"
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' ../../docs/pins.md)
[ -n "$tag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
MSYS_NO_PATHCONV=1; export MSYS_NO_PATHCONV
name=nava-backfill-s8-$$
trap 'docker rm -f "$name" >/dev/null 2>&1 || true' EXIT
docker run -d --name "$name" -e POSTGRES_PASSWORD=spike "postgres:$tag" >/dev/null
until docker exec "$name" pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
sleep 2
docker exec "$name" mkdir -p /work
docker cp ../../db/migrations/20261102233600_audit_log.sql "$name":/work/audit_log.sql
docker cp ../../db/migrations/20261103120000_audit_backfill_s8.sql "$name":/work/backfill.sql
for f in setup.sql load.sql run_backfill.sql plan.sql; do docker cp "$f" "$name":/work/"$f"; done
psql() { docker exec -i "$name" psql -U postgres -X -q "$@"; }
mkdir -p out
{
  echo "# $(date -u +%FT%TZ) postgres:$tag, host: $(uname -srm)"
  docker exec "$name" nproc | sed 's/^/# container cpus: /'
} > out/env.txt
psql -f /work/setup.sql > out/setup.txt 2>&1
for size in "${@:-1000 10 3}"; do
  set -- $size
  out=out/backfill-$1-tenants-$2-types-$3-versions.txt
  psql -v tenants="$1" -v types="$2" -v versions="$3" -f /work/load.sql > "$out" 2>&1
  psql -f /work/run_backfill.sql >> "$out" 2>&1
  psql -f /work/plan.sql >> "$out" 2>&1
  echo "== $out"; cat "$out"
done
