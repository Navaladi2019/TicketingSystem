#!/bin/sh
# S22 acceptance (TICK-754, PLAN-007 P7-14): one command from a clean state to
# the verdict. Story S22, TICK-740: unflag a filterable field, the 409 while
# its index is removed, the per-type quota and the operator raise, and the
# seed types filtering through the index only.
#
#   1. A fresh stack (volumes deleted) from this checkout, seeded with
#      docs/acceptance/guide/tools/seed.sh, and the tunnels to the operator
#      API (tools/tunnels.sh up).
#   2. tools/kit.mjs setup: admin@tenant-a creates the type s22_asset.
#   3. tools/seed.sql: 20,000 s22_asset records (a few seconds).
#   4. AC3, static half: the JSONB filter path and store.DataFilterTimeout are
#      gone from the non-test Go code (git grep).
#   5. Bruno folder "S22 unflag and quota" (bruno/S22/): criteria 1 to 4.
#   6. tools/verify.sql: no orphan side rows, the re-built index equals the
#      JSON, the tenant_limits row and its one audit entry.
#
# Usage, from the repository root (Git Bash or any POSIX shell), Docker up,
# Node.js 22:
#   sh docs/acceptance/S22/tools/run-kit.sh
# It deletes the stack's volumes first (docker compose down -v).
# S22_KEEP_STACK=1 skips step 1 on a stack that is up and has had nothing but
# tools/seed.sh (and, from run-s22-s29.sh, nothing touching s22_asset).
set -eu
cd "$(dirname "$0")/../../../.."
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV
if [ -f .env ]; then set -a; . ./.env; set +a; fi
COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-nava-s22}
export COMPOSE_PROJECT_NAME
NAVA_IMAGE_TAG=${NAVA_IMAGE_TAG:-s22}
export NAVA_IMAGE_TAG
COMPOSE_PROFILES=${COMPOSE_PROFILES:-operator}
export COMPOSE_PROFILES
EV=docs/acceptance/S22/evidence
T=docs/acceptance/S22/tools
mkdir -p "$EV"
find "$EV" -maxdepth 1 -type f -delete
# The raw Bruno reports hold dev-only tokens: never leave them behind (TICK-627).
trap 'rm -f "$EV"/.bruno-*.json' EXIT
FAILED=
stamp() { date -u +%Y-%m-%dT%H:%M:%SZ; }

psqlf() {
  docker compose exec -T postgres sh -c \
    'PGPASSWORD=$(cat /run/pg_superuser_password) psql -X -U pgadmin -d nava -v ON_ERROR_STOP=1 "$@"' sh "$@"
}

if [ "${S22_KEEP_STACK:-}" != 1 ]; then
  echo "s22: 1. fresh stack from $(git rev-parse HEAD) ($(stamp))"
  make dev-ca
  docker compose down -v
  docker compose up -d --build --wait
  env -u MSYS_NO_PATHCONV sh docs/acceptance/guide/tools/seed.sh
  env -u MSYS_NO_PATHCONV sh docs/acceptance/guide/tools/tunnels.sh up
fi

echo "s22: 2. the s22_asset type ($(stamp))"
node "$T/kit.mjs" setup "$EV"

echo "s22: 3. 20,000 s22_asset records ($(stamp))"
psqlf < "$T/seed.sql" | tee "$EV/seed.txt"

echo "s22: 4. AC3: the JSONB filter path and DataFilterTimeout are gone ($(stamp))"
# Non-test Go code only. Each pattern is one the retired path used
# (TICK-752's acceptance criteria).
{
  for p in 'DataFilterTimeout' 'LegacyScanFields' 'boundDataFilter'; do
    hits=$(git grep -n "$p" -- '*.go' ':!*_test.go' || true)
    echo "pattern $p: $(printf '%s' "$hits" | grep -c . || true) hits"
    [ -z "$hits" ] || printf '%s\n' "$hits"
  done
} | tee "$EV/ac3-static.txt"
if grep -qE '^pattern .*: [1-9][0-9]* hits' "$EV/ac3-static.txt"; then FAILED="$FAILED ac3-static"; fi

echo "s22: 5. Bruno folder S22 ($(stamp))"
env -u MSYS_NO_PATHCONV sh docs/acceptance/guide/tools/api-test.sh S22 \
  --env-var "keycloak_issuer_url=${KEYCLOAK_PUBLIC_URL:-http://localhost:18081}" || FAILED="$FAILED S22"
cp docs/acceptance/guide/reports/S22.json "$EV/.bruno-S22.json" 2>/dev/null || true

echo "s22: 6. the database behind criteria 1, 2 and 4 ($(stamp))"
psqlf < "$T/verify.sql" | tee "$EV/verify.txt"
if grep '^check|' "$EV/verify.txt" | grep -qv '|ok$' || ! grep -q '^check|' "$EV/verify.txt"; then
  echo "s22: verify FAILED" >&2
  FAILED="$FAILED verify"
fi
docker compose logs --no-log-prefix nava-worker 2>/dev/null | grep -iE 'field_index|fieldindex|remove' | tail -n 200 > "$EV/worker-field-index-log.txt" || true

node docs/acceptance/S15/tools/evidence.mjs "$EV" || true
{ git rev-parse HEAD; git status --porcelain --untracked-files=no | sed 's/^/modified: /'; } > "$EV/build.txt"
docker compose images --format '{{.Repository}}:{{.Tag}} {{.ID}}' 2>/dev/null | grep '^nava/' | sort -u >> "$EV/build.txt" || true
if [ -n "$FAILED" ]; then echo "s22: FAILED:$FAILED at $(git rev-parse HEAD); evidence in $EV ($(stamp))" | tee "$EV/result.txt" >&2; exit 1; fi
echo "s22: PASS at $(git rev-parse HEAD); evidence in $EV ($(stamp))" | tee "$EV/result.txt"
