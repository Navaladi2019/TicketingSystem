#!/bin/sh
# Finding 5 margin spike (TICK-450). Needs Docker. Usage: [ND='-0.0001 1'] run.sh [rows...]
# Starts postgres (tag from docs/pins.md), applies the real migration on stub parents, then for
# each row count loads 10,000 tenants and records the probe's generic plans.
set -eu
cd "$(dirname "$0")"
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' ../../docs/pins.md)
[ -n "$tag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
MSYS_NO_PATHCONV=1; export MSYS_NO_PATHCONV
name=nava-f1-margin-$$
trap 'docker rm -f "$name" >/dev/null 2>&1 || true' EXIT
docker run -d --name "$name" -e POSTGRES_PASSWORD=spike "postgres:$tag" >/dev/null
until docker exec "$name" pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
sleep 2
docker exec "$name" mkdir -p /work
docker cp ../../db/migrations/20261102233600_audit_log.sql "$name":/work/migration.sql
for f in setup.sql load.sql probe.sql; do docker cp "$f" "$name":/work/"$f"; done
psql() { docker exec -i "$name" psql -U postgres -X -q "$@"; }
mkdir -p out
{
  echo "# $(date -u +%FT%TZ) postgres:$tag, host: $(uname -srm)"
  docker exec "$name" nproc | sed 's/^/# container cpus: /'
} > out/env.txt
psql -f /work/setup.sql
# ND: the n_distinct overrides to compare; arguments: row counts.
for rows in "${@:-1000000}"; do
  for nd in ${ND:--0.0001 1}; do
    out=out/margin-10k-tenants-${rows}-rows-nd${nd}.txt
    psql -v tenants=10000 -v rows="$rows" -v nd="$nd" -f /work/load.sql > "$out" 2>&1
    psql -f /work/probe.sql >> "$out" 2>&1
    echo "== $out"; grep -E '^--- |Scan using|Seq Scan|ERROR' "$out" | sed 's/^ *//'
  done
done
