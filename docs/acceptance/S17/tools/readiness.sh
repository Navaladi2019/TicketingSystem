#!/bin/sh
# S17 acceptance, the two checks carried from the TICK-778 review. Run after
# run-kit.sh, on the same stack (it adds tenant B incidents, so run-kit.sh's
# exact numbers no longer hold afterwards).
#
#   1. Cold restart: restart PgBouncer and nava-core together, wait for
#      nava-core's /readyz 200, then one tenant B (gaps) create. It must be
#      201, with no 503 retry_later (the 200 ms allocation budget).
#   2. Readiness under saturation: /readyz polled every 250 ms inside the
#      nava-core container while every allocator connection is busy:
#      a. load: 40 clients creating tenant B incidents for 20 s;
#      b. stalled: the same load while a superuser session holds tenant B's
#         counter row locked, so every allocation waits its full 200 ms.
#      Reports how many polls were not 200.
#
# Usage, from the repository root, Git Bash or any POSIX shell:
#   sh docs/acceptance/S17/tools/readiness.sh
set -eu
cd "$(dirname "$0")/../../../.."
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV
if [ -f .env ]; then set -a; . ./.env; set +a; fi
COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-nava-s17}
export COMPOSE_PROJECT_NAME
EV=${S17_EVIDENCE:-docs/acceptance/S17/evidence}/readiness
rm -rf "$EV"
mkdir -p "$EV"
T=docs/acceptance/S17/tools
FAILED=

ready() { docker compose exec -T nava-core wget -q -O /dev/null http://127.0.0.1:9090/readyz >/dev/null 2>&1; }
# Polls /readyz every 250 ms for $1 seconds inside the container, one line each.
poll() {
  docker compose exec -T nava-core sh -c '
    end=$(( $(date +%s) + $1 ))
    while [ "$(date +%s)" -lt "$end" ]; do
      if wget -q -T 3 -O /dev/null http://127.0.0.1:9090/readyz 2>/dev/null; then echo 200; else echo NOT200; fi
      sleep 0.25
    done' sh "$1"
}
summary() { echo "$1: $(grep -c . "$2") polls, $(grep -c NOT200 "$2" || true) not 200"; }

echo "readiness: 1. cold restart of pgbouncer and nava-core"
{
  echo "restart at $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  docker compose restart pgbouncer nava-core
  i=0
  until ready; do i=$((i + 1)); [ "$i" -lt 120 ] || { echo "not ready after 60 s"; exit 1; }; sleep 0.5; done
  echo "readyz 200 at $(date -u +%Y-%m-%dT%H:%M:%SZ)"
} 2>&1 | tee "$EV/cold-restart.txt"
node "$T/load.mjs" first "$EV/cold-restart-first-create.json" | tee -a "$EV/cold-restart.txt" || FAILED="$FAILED cold-restart"
docker compose logs --no-log-prefix --since 2m nava-core 2>&1 | grep -iE 'retry_later|allocation took|deadline exceeded' > "$EV/cold-restart-alloc-errors.txt" || true
echo "allocation errors in the nava-core log since the restart: $(grep -c . "$EV/cold-restart-alloc-errors.txt" || true)" | tee -a "$EV/cold-restart.txt"
[ -s "$EV/cold-restart-alloc-errors.txt" ] && FAILED="$FAILED cold-restart-log"

echo "readiness: 2a. /readyz under create load (40 clients, 20 s)"
poll 24 > "$EV/saturation-load-polls.txt" &
P=$!
sleep 2
node "$T/load.mjs" saturate "$EV/saturation-load.json" 20 40
wait "$P"
summary "load" "$EV/saturation-load-polls.txt" | tee "$EV/saturation.txt"

echo "readiness: 2b. /readyz with tenant B's counter row locked (40 clients, 20 s)"
docker compose exec -T postgres sh -c \
  'PGPASSWORD=$(cat /run/pg_superuser_password) psql -U pgadmin -d nava -v ON_ERROR_STOP=1 -c "BEGIN; SELECT tenant_id, next FROM record_counters WHERE tenant_id = '\''00000000-0000-4000-8000-0000000000b1'\'' FOR UPDATE; SELECT pg_sleep(26); COMMIT;"' \
  > "$EV/saturation-lock.txt" 2>&1 &
L=$!
sleep 1
poll 24 > "$EV/saturation-stalled-polls.txt" &
P=$!
sleep 2
node "$T/load.mjs" saturate "$EV/saturation-stalled.json" 20 40
wait "$P"
wait "$L" || true
summary "stalled" "$EV/saturation-stalled-polls.txt" | tee -a "$EV/saturation.txt"
if grep -v ', 0 not 200$' "$EV/saturation.txt" | grep -q .; then FAILED="$FAILED saturation"; fi

if [ -n "$FAILED" ]; then echo "readiness: FAILED:$FAILED" | tee "$EV/result.txt"; exit 1; fi
echo "readiness: PASS (cold restart and saturation)" | tee "$EV/result.txt"
