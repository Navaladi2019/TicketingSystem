#!/bin/sh
# S21 acceptance (TICK-749, PLAN-007 P7-09): one command from a clean state to
# the verdict. Story S21, TICK-739: flag a top-level field x-nava-filterable,
# watch it index, then filter a 400k type on it.
#
#   1. A fresh stack (volumes deleted) from this checkout, seeded with
#      docs/acceptance/guide/tools/seed.sh (tenants A and B, their users).
#   2. tools/kit.mjs setup: admin@tenant-a adds `due_at` (date-time) to the
#      incident type and masks `description` to the admins group.
#   3. tools/seed.sql: 400,000 incidents in tenant A, 50 in tenant B, as the
#      superuser (about a minute). tools/expect.sql: what a JSON scan gives
#      for the kit's filters, passed to Bruno.
#   4. Bruno folder "S21 filterable fields" (bruno/S21/): criteria 1 to 5.
#      It publishes x-nava-filterable on priority and due_at and waits for
#      the backfill (about 8 minutes a field on the reference machine).
#   5. tools/verify.sql: the side tables against a JSON scan, both fields.
#   6. tools/kit.mjs report: the ADR-006-A2 checks on the evidence (total
#      contract, short results exact, the >= 40% ACL cell's p95 and its share
#      of `unknown`, the no-filter controls) into evidence/p95.txt. While step
#      4 runs, rfv_num's and rfv_ts's relpages / relallvisible are sampled
#      every 10 s into evidence/visibility.txt, so the all-visible share at the
#      moment S21.19's cells run is on record (A2-7).
#
# Usage, from the repository root (Git Bash or any POSIX shell), Docker up,
# Node.js 22:
#   sh docs/acceptance/S21/tools/run-kit.sh
# About 35 minutes: the image build, then mostly the two backfills. Ports and
# the project name come from .env, as compose reads them. It deletes the
# stack's volumes first (docker compose down -v). S21_KEEP_STACK=1 skips step
# 1 and reuses a stack that is up but has had nothing but tools/seed.sh.
# S21_P95_SECONDS (default 15) is the length of each p95 cell.
set -eu
cd "$(dirname "$0")/../../../.."
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV
if [ -f .env ]; then set -a; . ./.env; set +a; fi
COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-nava-s21}
export COMPOSE_PROJECT_NAME
# Its own image tag: another stack on the host building nava/*:dev from
# another commit would otherwise swap the images under this one.
NAVA_IMAGE_TAG=${NAVA_IMAGE_TAG:-s21}
export NAVA_IMAGE_TAG
EV=${S21_EVIDENCE:-docs/acceptance/S21/evidence}
T=docs/acceptance/S21/tools
case "$EV" in
  *..*) echo "s21: S21_EVIDENCE must not contain .. (got: $EV)" >&2; exit 2 ;;
  docs/acceptance/S21/evidence|docs/acceptance/S21/evidence/*) ;;
  *) echo "s21: S21_EVIDENCE must be docs/acceptance/S21/evidence or a folder under it (got: $EV)" >&2; exit 2 ;;
esac
mkdir -p "$EV"
# A fresh run replaces the top-level evidence; subfolders (earlier runs kept
# on purpose, such as stack1-p95-miss/) stay.
find "$EV" -maxdepth 1 -type f -delete
# The raw Bruno reports hold dev-only tokens: never leave them behind (TICK-627).
trap 'rm -f "$EV"/.bruno-*.json' EXIT
FAILED=
stamp() { date -u +%Y-%m-%dT%H:%M:%SZ; }

psqlf() {
  docker compose exec -T postgres sh -c \
    'PGPASSWORD=$(cat /run/pg_superuser_password) psql -X -U pgadmin -d nava -v ON_ERROR_STOP=1 "$@"' sh "$@"
}

if [ "${S21_KEEP_STACK:-}" != 1 ]; then
  echo "s21: 1. fresh stack from $(git rev-parse HEAD) ($(stamp))"
  make dev-ca
  docker compose down -v
  docker compose up -d --build --wait
  env -u MSYS_NO_PATHCONV sh docs/acceptance/guide/tools/seed.sh
fi

echo "s21: 2. due_at and the description mask ($(stamp))"
node "$T/kit.mjs" setup "$EV"

echo "s21: 3. 400k incidents in tenant A, 50 in tenant B ($(stamp))"
start=$(date +%s)
psqlf < "$T/seed.sql" | tee "$EV/seed.txt"
echo "seed took $(( $(date +%s) - start )) s" | tee -a "$EV/seed.txt"
psqlf < "$T/expect.sql" | grep '^s21_' | tee "$EV/expect.txt"
envs=""
while IFS='=' read -r k v; do
  envs="$envs --env-var $k=$v"
done < "$EV/expect.txt"

echo "s21: 4. Bruno folder S21 ($(stamp))"
# The visibility map of the side tables while the folder runs (A2-7: broad ACL
# counts are slow until autovacuum has marked their pages all-visible).
echo "sampled_at|relname|relpages|relallvisible|n_live_tup|last_autovacuum|last_autoanalyze" > "$EV/visibility.txt"
( while :; do
    psqlf -At -c "SELECT to_char(now() AT TIME ZONE 'UTC', 'YYYY-MM-DD\"T\"HH24:MI:SS\"Z\"'), c.relname, c.relpages, c.relallvisible, s.n_live_tup, s.last_autovacuum, s.last_autoanalyze FROM pg_class c JOIN pg_stat_all_tables s ON s.relid = c.oid WHERE c.relname IN ('rfv_num', 'rfv_ts') ORDER BY 2" < /dev/null >> "$EV/visibility.txt" 2>/dev/null || true
    sleep 10
  done ) &
SAMPLER=$!
trap 'kill "$SAMPLER" 2>/dev/null || true; rm -f "$EV"/.bruno-*.json' EXIT
# Bruno (Node on Windows) needs Git Bash to convert its report paths.
# shellcheck disable=SC2086 # envs is a list of words, none with spaces
env -u MSYS_NO_PATHCONV sh docs/acceptance/guide/tools/api-test.sh S21 \
  --env-var "keycloak_issuer_url=${KEYCLOAK_PUBLIC_URL:-http://localhost:18081}" \
  --env-var "s21_p95_seconds=${S21_P95_SECONDS:-15}" $envs || FAILED="$FAILED S21"
cp docs/acceptance/guide/reports/S21.json "$EV/.bruno-S21.json"
kill "$SAMPLER" 2>/dev/null || true
wait "$SAMPLER" 2>/dev/null || true

echo "s21: 5. side tables against a JSON scan ($(stamp))"
psqlf < "$T/verify.sql" | tee "$EV/verify.txt"
# priority and due_at rows: missing, surplus and wrong are the last three columns.
if ! awk -F'|' 'NF == 6 && $2 ~ /^ *[0-9]+ *$/ && $1 ~ /^ *(priority|due_at) *$/ { n++; for (i = NF - 2; i <= NF; i++) { v = $i; gsub(/[ \r]/, "", v); if (v != "0") bad = 1 } }
                $1 ~ /tenant B side rows/ { b = $2; gsub(/[ \r]/, "", b); if (b != "0") bad = 1 }
                END { exit (bad || n != 2) }' "$EV/verify.txt"; then
  echo "s21: verify FAILED: a side table differs from the JSON scan, or tenant B has side rows" >&2
  FAILED="$FAILED verify"
fi
docker compose logs --no-log-prefix nava-worker 2>/dev/null | grep -iE 'fieldindex|field_index|backfill' | tail -n 200 > "$EV/worker-backfill-log.txt" || true

node docs/acceptance/S15/tools/evidence.mjs "$EV" || true
echo "s21: 6. ADR-006-A2 report: total_state, the >= 40% ACL cell, controls ($(stamp))"
node "$T/kit.mjs" report "$EV" || FAILED="$FAILED report"
{ git rev-parse HEAD; git status --porcelain --untracked-files=no | sed 's/^/modified: /'; } > "$EV/build.txt"
docker compose images --format '{{.Repository}}:{{.Tag}} {{.ID}}' 2>/dev/null | grep '^nava/' | sort -u >> "$EV/build.txt" || true
docker info --format 'docker: {{.NCPU}} CPUs, {{.MemTotal}} bytes, {{.ServerVersion}}' >> "$EV/build.txt"
echo "other containers: $(docker ps --format '{{.Names}}' | grep -v "^$COMPOSE_PROJECT_NAME" | tr '\n' ' ')" >> "$EV/build.txt"
if [ -n "$FAILED" ]; then echo "s21: FAILED:$FAILED at $(git rev-parse HEAD); evidence in $EV ($(stamp))" | tee "$EV/result.txt" >&2; exit 1; fi
echo "s21: PASS at $(git rev-parse HEAD); evidence in $EV ($(stamp))" | tee "$EV/result.txt"
