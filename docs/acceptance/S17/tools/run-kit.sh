#!/bin/sh
# S17 acceptance (TICK-668): one command from a clean state to the verdict.
# Story S17, TICK-664: release R1 of ADR-005 (migrations M1 and M2, the
# 3-column counter upsert). The story's claim is that nothing changes for a
# user: every type keeps counting where it was, on both numbering modes.
#
#   1. A fresh stack (volumes deleted) on the S10 build, PRE_REF (default
#      0e79d828, `main` after S10 merged, the story branch's base), from the
#      sibling worktree ../s17-pre, seeded with that commit's tools/seed.sh.
#   2. On that build (kit.mjs pre): admin@tenant-a sets tenant A's incident
#      type to gapless; tenant B's stays gaps. Each tenant's incidents are
#      created until its last number is INC0000042.
#   3. The same stack (same volumes, same COMPOSE_PROJECT_NAME) is upgraded
#      to this checkout: the migrate service applies M1 and M2.
#   4. Bruno folder S17 (bruno/S17/): INC0000043 on both types (AC1), 8
#      clients at once (AC2), tenant B against tenant A's numbering (AC3).
#   5. The database before and after, and the runbook check (AC4).
#
# S17_FORCE_FAIL=1 adds one tenant A create between steps 3 and 4, so tenant
# A's first number in Bruno is INC0000044: the kit must then exit 1 (the
# proof that it fails when the counter does not carry on). Give it its own
# evidence folder: S17_EVIDENCE=docs/acceptance/S17/evidence/forced-failure.
#
# Usage, from the repository root (Git Bash or any POSIX shell), Docker up:
#   sh docs/acceptance/S17/tools/run-kit.sh
# About 15 minutes, most of it the two image builds. Ports and the project
# name come from .env, as compose reads them. It deletes the stack's volumes
# first (docker compose down -v).
set -eu
cd "$(dirname "$0")/../../../.."
ROOT=$(pwd)
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV
if [ -f .env ]; then set -a; . ./.env; set +a; fi
PRE_REF=${PRE_REF:-0e79d828}
COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-nava-s17}
export COMPOSE_PROJECT_NAME
PRE="$ROOT/../s17-pre"
EV=${S17_EVIDENCE:-docs/acceptance/S17/evidence}
rm -rf "$EV"
mkdir -p "$EV"
# The raw Bruno reports hold dev-only tokens: never leave them behind (TICK-627).
trap 'rm -f "$EV"/.bruno-*.json' EXIT
FAILED=

sql() {
  docker compose exec -T postgres sh -c \
    'PGPASSWORD=$(cat /run/pg_superuser_password) psql -U pgadmin -d nava -v ON_ERROR_STOP=1 -P footer=off "$@"' sh "$@"
}
schema() {
  echo "== $1 ($(date -u +%Y-%m-%dT%H:%M:%SZ))"
  sql -c "SELECT column_name, data_type, is_nullable, column_default FROM information_schema.columns
           WHERE table_name = 'record_counters' ORDER BY ordinal_position"
  sql -c "SELECT indexname, indexdef FROM pg_indexes WHERE tablename = 'record_counters' ORDER BY indexname"
  sql -c "SELECT version_id FROM goose_db_version WHERE version_id >= 20261111000000 ORDER BY version_id"
}
counters() {
  echo "== $1 ($(date -u +%Y-%m-%dT%H:%M:%SZ))"
  if [ "$(sql -Atc "SELECT count(*) FROM information_schema.columns WHERE table_name = 'record_counters' AND column_name = 'scope_key'")" = 1 ]; then
    scope="c.scope_key, c.period_end,"
  else
    scope=""
  fi
  sql -c "SELECT c.tenant_id, d.key, d.number_prefix, d.numbering_mode, $scope c.next,
                 (SELECT max(r.number) FROM records r WHERE r.tenant_id = c.tenant_id AND r.entity_definition_id = c.entity_definition_id) AS max_number,
                 (SELECT count(*) FROM records r WHERE r.tenant_id = c.tenant_id AND r.entity_definition_id = c.entity_definition_id) AS records
            FROM record_counters c JOIN entity_definitions d ON d.tenant_id = c.tenant_id AND d.id = c.entity_definition_id
           ORDER BY c.tenant_id, d.key"
}

echo "s17: 1. fresh stack on the S10 build ($PRE_REF)"
# A relative path: with MSYS_NO_PATHCONV set, Git for Windows reads /c/... as
# C:/c/... and puts the worktree there.
[ -d "$PRE" ] || git worktree add -q --detach ../s17-pre "$PRE_REF"
( cd "$PRE" && git checkout -q --detach "$PRE_REF" )
make dev-ca
mkdir -p "$PRE/deploy/compose/certs" && cp -R deploy/compose/certs/. "$PRE/deploy/compose/certs/"
rm -f "$PRE/.env"
[ -f .env ] && cp .env "$PRE/.env"
# The S10 images get their own tag, so the upgrade below rebuilds and
# recreates every nava/* service from this checkout.
( cd "$PRE" && export NAVA_IMAGE_TAG=s17pre \
  && docker compose down -v \
  && docker compose up -d --build --wait \
  && env -u MSYS_NO_PATHCONV sh docs/acceptance/guide/tools/seed.sh )
schema "schema on the S10 build, before the upgrade" | tee "$EV/schema-before-upgrade.txt"

echo "s17: 2. tenant A gapless, tenant B gaps, both to INC0000042 (S10 build)"
node docs/acceptance/S17/tools/kit.mjs pre "$EV"
counters "counters on the S10 build, before the upgrade" | tee "$EV/counters-before-upgrade.txt"

echo "s17: 3. upgrade the same stack to $(git rev-parse --short HEAD)"
docker compose up -d --build --wait
docker compose logs --no-log-prefix migrate | grep -E '20261111123[0-3]00|"migrate done"' | tee "$EV/migrate-log.txt"
grep -q '"migrate done"' "$EV/migrate-log.txt" || { echo "s17: the migrate service did not finish" >&2; exit 1; }
schema "schema after the upgrade (M1, M2)" | tee "$EV/schema-after-upgrade.txt"
counters "counters after the upgrade, before any create" | tee "$EV/counters-after-upgrade.txt"

if [ "${S17_FORCE_FAIL:-}" = 1 ]; then
  node docs/acceptance/S17/tools/kit.mjs force "$EV"
fi

echo "s17: 4. Bruno folder S17"
# Bruno (Node on Windows) needs Git Bash to convert its report paths.
env -u MSYS_NO_PATHCONV sh docs/acceptance/guide/tools/api-test.sh S17 \
  --env-var "keycloak_issuer_url=${KEYCLOAK_PUBLIC_URL:-http://localhost:18081}" || FAILED="$FAILED S17"
cp docs/acceptance/guide/reports/S17.json "$EV/.bruno-S17.json"

echo "s17: 5. database after the run, and the runbook (AC4)"
counters "counters after the Bruno folder" | tee "$EV/counters-after-run.txt"
sql -c "SELECT r.tenant_id, d.numbering_mode, count(*) AS records, count(DISTINCT r.number) AS distinct_numbers
          FROM records r JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
         WHERE d.key = 'incident' GROUP BY 1, 2 ORDER BY 1" | tee "$EV/unique-numbers.txt"
if ! awk -F'|' 'NR > 2 && NF >= 4 { gsub(/ /, ""); if ($3 != $4) bad = 1 } END { exit bad }' "$EV/unique-numbers.txt"; then
  echo "s17: AC2 FAILED: a tenant has a duplicate incident number" >&2
  FAILED="$FAILED unique-numbers"
fi
RB=docs/data/record-counters-scope-key.md
{
  echo "== $RB: M3 is a later release, and the pre-S17 build cannot be deployed after it"
  grep -nE 'M3 \(S18|then and only then run M3 with R2|R0 can never be deployed again after M3|\*\*M3 is the point of no return\.\*\*|\| R2 / M3 \| anything earlier \| \*\*Not possible\.\*\* \|' "$RB"
} | tee "$EV/ac4-runbook.txt"
[ "$(grep -c "^[0-9]" "$EV/ac4-runbook.txt")" -ge 5 ] || { echo "s17: AC4 FAILED: the runbook lines are missing" >&2; FAILED="$FAILED runbook"; }

node docs/acceptance/S15/tools/evidence.mjs "$EV" || true
git rev-parse HEAD > "$EV/build.txt"
echo "pre: $(cd "$PRE" && git rev-parse HEAD)" >> "$EV/build.txt"
if [ -n "$FAILED" ]; then echo "s17: FAILED:$FAILED; evidence in $EV" | tee "$EV/result.txt" >&2; exit 1; fi
echo "s17: PASS; evidence in $EV" | tee "$EV/result.txt"
