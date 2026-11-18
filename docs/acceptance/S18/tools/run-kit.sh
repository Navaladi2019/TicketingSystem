#!/bin/sh
# S18 acceptance (TICK-677): one command from a clean state to the verdict.
# Story S18, TICK-669: release R2 of ADR-005. M3 makes the 3-column counter
# key the primary key, then a tenant admin defines a number format and links
# it to a ticket type.
#
#   1. A fresh stack (volumes deleted) on the S17 build, PRE_REF (default
#      ac253bad: `main` with S17 merged, the commit the story branch was cut
#      from, so the upgrade applies S18's migrations and nothing else), from
#      the sibling worktree ../s18-pre, seeded with that commit's seed.sh.
#      admin@tenant-b.test is added (kit.mjs admin-b, one group sync).
#   2. On that build (kit.mjs pre): the ticket types `visa` (VISA) and
#      `permit` (PMT) in tenant A; VISA0000001..2 and INC0000001..2 in tenant
#      A, INC0000001..2 in tenant B, all on the default numbering.
#   3. The same stack (same volumes and project) is upgraded to this
#      checkout: the migrate service applies S18's migrations, M3 among them.
#   4. Bruno folder S18 (`S18 number formats`, bruno/S18/): the five story
#      criteria as admin@tenant-a, agent@tenant-a (Alice) and admin@tenant-b,
#      and the E9 check (number_space_exhausted and its one audit entry).
#   5. The counters, the primary key and the audit rows from the database.
#
# S18_FORCE_FAIL=1 makes one extra Visa create right after the format is
# linked, before the criterion-1 create, so the first formatted Visa number is
# ...0002: the kit must then exit 1. Give it its own evidence folder:
# S18_EVIDENCE=docs/acceptance/S18/evidence/forced-failure.
#
# Usage, from the repository root (Git Bash or any POSIX shell), Docker up:
#   sh docs/acceptance/S18/tools/run-kit.sh
# About 15 minutes, most of it two image builds. Ports and the project name
# come from .env, as compose reads them. It deletes the stack's volumes first
# (docker compose down -v).
set -eu
cd "$(dirname "$0")/../../../.."
ROOT=$(pwd)
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV
if [ -f .env ]; then set -a; . ./.env; set +a; fi
PRE_REF=${PRE_REF:-ac253bad}
COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-nava-s18}
export COMPOSE_PROJECT_NAME
PRE="$ROOT/../s18-pre"
EV=${S18_EVIDENCE:-docs/acceptance/S18/evidence}
TZ_A=${S18_TENANT_TZ:-Pacific/Kiritimati}
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
  sql -c "SELECT conname, pg_get_constraintdef(oid) AS def FROM pg_constraint
           WHERE conrelid = 'record_counters'::regclass AND contype IN ('p', 'u') ORDER BY conname"
  sql -c "SELECT indexname, indexdef FROM pg_indexes WHERE tablename = 'record_counters' ORDER BY indexname"
  sql -c "SELECT count(*) AS s18_migrations FROM goose_db_version WHERE version_id BETWEEN 20261113210000 AND 20261115999999 AND is_applied"
}
counters() {
  echo "== $1 ($(date -u +%Y-%m-%dT%H:%M:%SZ))"
  sql -c "SELECT c.tenant_id, d.key, d.number_prefix, c.scope_key, c.period_end, c.next,
                 (SELECT count(*) FROM records r WHERE r.tenant_id = c.tenant_id AND r.entity_definition_id = c.entity_definition_id) AS records
            FROM record_counters c JOIN entity_definitions d ON d.tenant_id = c.tenant_id AND d.id = c.entity_definition_id
           ORDER BY c.tenant_id, d.key, c.scope_key"
}

echo "s18: 1. fresh stack on the S17 build ($PRE_REF)"
# A relative path: with MSYS_NO_PATHCONV set, Git for Windows reads /c/... as
# C:/c/... and puts the worktree there.
[ -d "$PRE" ] || git worktree add -q --detach ../s18-pre "$PRE_REF"
( cd "$PRE" && git checkout -q --detach "$PRE_REF" )
make dev-ca
mkdir -p "$PRE/deploy/compose/certs" && cp -R deploy/compose/certs/. "$PRE/deploy/compose/certs/"
rm -f "$PRE/.env"
[ -f .env ] && cp .env "$PRE/.env"
# The S17 images get their own tag, so the upgrade below rebuilds and
# recreates every nava/* service from this checkout.
( cd "$PRE" && export NAVA_IMAGE_TAG=s18pre \
  && docker compose down -v \
  && docker compose up -d --build --wait \
  && env -u MSYS_NO_PATHCONV sh docs/acceptance/guide/tools/seed.sh )
node docs/acceptance/S18/tools/kit.mjs admin-b
( cd "$PRE" && export NAVA_IMAGE_TAG=s18pre && docker compose run --rm --no-deps nava-groupsync --once )
node docs/acceptance/S18/tools/kit.mjs me-b "$EV"
schema "record_counters on the S17 build, before the upgrade" | tee "$EV/schema-before-upgrade.txt"

echo "s18: 2. visa and permit types, default numbers (S17 build)"
node docs/acceptance/S18/tools/kit.mjs pre "$EV"
counters "counters on the S17 build, before the upgrade" | tee "$EV/counters-before-upgrade.txt"

echo "s18: 3. upgrade the same stack to $(git rev-parse --short HEAD)"
docker compose up -d --build --wait
docker compose logs --no-log-prefix migrate | grep -E '2026111[345][0-9]{6}|"migrate done"' | tee "$EV/migrate-log.txt"
grep -q '"migrate done"' "$EV/migrate-log.txt" || { echo "s18: the migrate service did not finish" >&2; exit 1; }
schema "record_counters after the upgrade (M3)" | tee "$EV/schema-after-upgrade.txt"
grep -q 'PRIMARY KEY (tenant_id, entity_definition_id, scope_key)' "$EV/schema-after-upgrade.txt" \
  || { echo "s18: M3 did not make the 3-column key the primary key" >&2; FAILED="$FAILED M3"; }

echo "s18: 4. Bruno folder S18 (tenant A time zone $TZ_A)"
ids=""
while IFS='=' read -r k v; do [ -n "$k" ] && ids="$ids --env-var $k=$v"; done < "$EV/pre-ids.env"
# Bruno (Node on Windows) needs Git Bash to convert its report paths.
# shellcheck disable=SC2086
env -u MSYS_NO_PATHCONV sh docs/acceptance/guide/tools/api-test.sh S18 \
  --env-var "keycloak_issuer_url=${KEYCLOAK_PUBLIC_URL:-http://localhost:18081}" \
  --env-var "s18_tz=$TZ_A" --env-var "s18_force_fail=${S18_FORCE_FAIL:-0}" $ids || FAILED="$FAILED S18"
cp docs/acceptance/guide/reports/S18.json "$EV/.bruno-S18.json"

echo "s18: 5. database after the run"
counters "counters after the Bruno folder" | tee "$EV/counters-after-run.txt"
sql -c "SELECT d.tenant_id, d.key, r.number, r.created_at
          FROM records r JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
         WHERE d.key IN ('visa', 'incident', 'permit') ORDER BY d.tenant_id, d.key, r.created_at" | tee "$EV/numbers-after-run.txt"
sql -c "SELECT tenant_id, action, subject_type, diff::text FROM audit_log
         WHERE action = 'number_space_exhausted' ORDER BY tenant_id, occurred_at" | tee "$EV/audit-exhausted-db.txt"
[ "$(sql -Atc "SELECT count(*) FROM audit_log WHERE action = 'number_space_exhausted' AND tenant_id = '00000000-0000-4000-8000-0000000000a1'")" = 1 ] \
  || { echo "s18: E9 FAILED: tenant A does not have exactly one number_space_exhausted row" >&2; FAILED="$FAILED E9-db"; }
[ "$(sql -Atc "SELECT count(*) FROM audit_log WHERE action = 'number_space_exhausted' AND tenant_id = '00000000-0000-4000-8000-0000000000b1'")" = 0 ] \
  || { echo "s18: E9 FAILED: tenant B has a number_space_exhausted row" >&2; FAILED="$FAILED E9-db-b"; }

node docs/acceptance/S15/tools/evidence.mjs "$EV" || true
rm -f "$EV/pre-ids.env"
git rev-parse HEAD > "$EV/build.txt"
echo "pre: $(cd "$PRE" && git rev-parse HEAD)" >> "$EV/build.txt"
echo "tenant A time zone: $TZ_A" >> "$EV/build.txt"
if [ -n "$FAILED" ]; then echo "s18: FAILED:$FAILED; evidence in $EV" | tee "$EV/result.txt" >&2; exit 1; fi
echo "s18: PASS; evidence in $EV" | tee "$EV/result.txt"
