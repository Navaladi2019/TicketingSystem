#!/bin/sh
# S15 acceptance (TICK-443): walks the four S15 acceptance steps on a stack
# set up by upgrade-setup.sh, and writes the evidence to
# docs/acceptance/S15/evidence/.
#
#   AC1  Bruno folder S15 (S15 seal): a new tenant is onboarded and its
#        onboarding entries are sealed; Alice's incident is sealed within
#        about 10 s; the metrics are on /metrics (AC4); isolation checks.
#        Then the per-tenant seal latency from the database.
#   AC2  The worker is killed (docker compose kill nava-worker, SIGKILL)
#        while both tenants write incidents (load.mjs), kept down while the
#        writes go on, then started again. Once nothing is unsealed,
#        seq-check.sql must show no gap and no duplicate seq for any tenant,
#        and Bruno folder S15K shows the same through the API.
#   AC3  The script waits until tenant A's sealer has backed off to 300 s,
#        then Bruno folder S15I writes one incident and checks it is sealed
#        within 300 s.
#
# Usage, from the repository root (Git Bash or any POSIX shell):
#   sh docs/acceptance/S15/tools/run-kit.sh
# Takes about 20 minutes, most of it the idle back-off of AC3.
set -eu
cd "$(dirname "$0")/../../../.."
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV
if [ -f .env ]; then set -a; . ./.env; set +a; fi
EV=docs/acceptance/S15/evidence
mkdir -p "$EV"
A=00000000-0000-4000-8000-0000000000a1
B=00000000-0000-4000-8000-0000000000b1
CORE_PORT=${NAVA_CORE_PORT:-18080}
KC_PORT=${KEYCLOAK_PORT:-18081}
ADMIN_PORT=${ACCEPT_ADMIN_PORT:-19090}
FAILED=
# bruno <folder> [args]: runs a folder, keeps its report, and records a failure
# so the remaining steps still run.
bruno() {
  f=$1; shift
  # shellcheck disable=SC2086
  sh docs/acceptance/guide/tools/api-test.sh "$f" $BRU "$@" || FAILED="$FAILED $f"
  cp "docs/acceptance/guide/reports/$f.json" "$EV/.bruno-$f.json"
}
BRU="--env-var core_url=http://127.0.0.1:$CORE_PORT \
 --env-var keycloak_url=http://127.0.0.1:$KC_PORT \
 --env-var keycloak_issuer_url=${KEYCLOAK_PUBLIC_URL:-http://localhost:18081} \
 --env-var onboard_url=http://127.0.0.1:${ACCEPT_ONBOARD_PORT:-18082} \
 --env-var worker_admin_url=http://127.0.0.1:$((ADMIN_PORT + 4))"

sql() {
  docker compose exec -T postgres sh -c \
    'PGPASSWORD=$(cat /run/pg_superuser_password) psql -U pgadmin -d nava -v ON_ERROR_STOP=1 -P footer=off "$@"' sh "$@"
}
# One line per tenant: the head, the unsealed backlog, the seal start rows.
state() {
  echo "== $1 ($(date -u +%Y-%m-%dT%H:%M:%SZ))"
  sql -c "SELECT h.tenant_id, h.version, h.seq AS head_seq, h.seal_interval_s,
       to_char(h.next_seal_at AT TIME ZONE 'UTC', 'HH24:MI:SS') AS next_seal_at,
       (SELECT count(*) FROM audit_log a WHERE a.tenant_id = h.tenant_id AND a.seq IS NULL) AS unsealed,
       (SELECT round(extract(epoch FROM now() - min(occurred_at))) FROM audit_log a
         WHERE a.tenant_id = h.tenant_id AND a.seq IS NULL) AS oldest_unsealed_s,
       (SELECT count(*) FROM outbox o WHERE o.tenant_id = h.tenant_id AND o.kind = 'audit.seal' AND o.dispatched_at IS NULL AND o.dead_at IS NULL) AS seal_rows_pending,
       (SELECT count(*) FROM outbox o WHERE o.tenant_id = h.tenant_id AND o.kind = 'audit.seal' AND o.dead_at IS NOT NULL) AS seal_rows_dead
  FROM audit_chain_head h ORDER BY h.tenant_id"
}
unsealed() { sql -Atc "SELECT count(*) FROM audit_log WHERE seq IS NULL"; }
latency() {
  sql -c "SELECT tenant_id, count(*) AS entries, count(seq) AS sealed,
       round(extract(epoch FROM percentile_cont(0.5) WITHIN GROUP (ORDER BY sealed_at - occurred_at))::numeric, 1) AS p50_s,
       round(extract(epoch FROM max(sealed_at - occurred_at))::numeric, 1) AS max_s
  FROM audit_log WHERE occurred_at >= '$1' GROUP BY tenant_id ORDER BY tenant_id"
}

echo "s15: migration B gave the seeded tenants their head (AC2 setup)"
state "heads after the upgrade" | tee "$EV/heads-after-upgrade.txt"

T0=$(date -u +%Y-%m-%dT%H:%M:%SZ)
echo "s15: AC1, AC4: Bruno folder S15"
bruno S15
{ echo "Seal latency of every entry written since $T0 (sealed_at - occurred_at)"; latency "$T0"; } | tee "$EV/ac1-seal-latency.txt"

echo "s15: AC2: kill the worker while both tenants write"
{
  state "before the load"
  node docs/acceptance/S15/tools/load.mjs 60 15 "$EV/load.json" > "$EV/.load.log" 2>&1 &
  LOAD=$!
  sleep 15
  state "just before the kill (worker running, writes arriving)"
  KILL_AT=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  echo "== $KILL_AT docker compose kill nava-worker"
  docker compose kill nava-worker
  docker compose ps -a nava-worker --format '{{.Name}} {{.State}} {{.Status}}'
  sleep 25
  state "worker down 25 s, writes still arriving: unsealed grows, head_seq stands"
  echo "== $(date -u +%Y-%m-%dT%H:%M:%SZ) docker compose up -d --no-deps --wait nava-worker"
  docker compose up -d --no-deps --wait --wait-timeout 180 nava-worker
  wait "$LOAD"
  cat "$EV/.load.log"
  state "writes done"
  i=0
  while [ "$(unsealed)" != 0 ]; do
    i=$((i + 1)); [ "$i" -le 66 ] || { echo "s15: still unsealed after 660 s" >&2; exit 1; }
    sleep 10
  done
  state "backlog sealed"
  echo "== kill at $KILL_AT"
} 2>&1 | tee "$EV/ac2-kill-restart.txt"
KILL_AT=$(sed -n 's/^== kill at //p' "$EV/ac2-kill-restart.txt")
[ -n "$KILL_AT" ] || { echo "s15: the kill step did not finish" >&2; exit 1; }
sql -f - < docs/acceptance/S15/tools/seq-check.sql | tee "$EV/ac2-seq-check.txt"
if grep -Eq '\| f$' "$EV/ac2-seq-check.txt" || ! grep -Eq '^ +0$' "$EV/ac2-seq-check.txt"; then
  echo "s15: AC2 FAILED: a tenant's chain has a gap, a duplicate or an unsealed entry" >&2
  FAILED="$FAILED seq-check"
fi
{ echo "Seal latency of the entries written from the kill on"; latency "$KILL_AT"; } | tee "$EV/ac2-seal-latency.txt"
bruno S15K --env-var "s15_kill_at=$KILL_AT"

echo "s15: AC3: wait until tenant A's sealer has backed off to 300 s"
i=0
while [ "$(sql -Atc "SELECT seal_interval_s FROM audit_chain_head WHERE tenant_id = '$A'")" != 300 ]; do
  i=$((i + 1)); [ "$i" -le 90 ] || { echo "s15: tenant A not at 300 s after 900 s" >&2; exit 1; }
  sleep 10
done
state "tenant A idle: interval 300 s" | tee "$EV/ac3-idle.txt"
T3=$(date -u +%Y-%m-%dT%H:%M:%SZ)
bruno S15I
{ state "after the idle write"; echo "Seal latency since $T3"; latency "$T3"; } | tee -a "$EV/ac3-idle.txt"

node docs/acceptance/S15/tools/evidence.mjs "$EV" || true
if [ -n "$FAILED" ]; then echo "s15: FAILED:$FAILED; evidence in $EV" >&2; exit 1; fi
echo "s15: PASS; evidence in $EV"
