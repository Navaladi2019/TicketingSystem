#!/bin/sh
# S29 acceptance (TICK-754, formerly TICK-928; PLAN-010 P10-13): one command
# from a clean state to the verdict. Story S29, TICK-915: a published default
# reaches old tickets in the background, and filters count it at once.
#
#   1. A fresh stack (volumes deleted) from this checkout, with the operator
#      profile (Temporal UI, for the workflow-run listing), seeded with
#      docs/acceptance/guide/tools/seed.sh, admin@tenant-b (S18's kit.mjs
#      admin-b and one group sync), and tools/tunnels.sh up.
#   2. prep: kit.mjs setup (priority's default removed, s29_tier added);
#      seed.sql (2,000 tenant A incidents without priority, one poison);
#      kit.mjs held (50 with a priority through the API, 5 in tenant B);
#      expect.sql (what the filter must give).
#   3. check: Bruno folder "S29 schema backfill" (bruno/S29/), criteria 1 to 6;
#      then verify.sql.
#
# Usage, from the repository root (Git Bash or any POSIX shell), Docker up,
# Node.js 22:
#   sh docs/acceptance/S29/tools/run-kit.sh
# About 10 minutes: the image build, then two jobs over about 2,050 records.
# S29_KEEP_STACK=1 skips step 1. S29_PHASE=prep or S29_PHASE=check runs only
# that half (run-s22-s29.sh runs S22 between them on the same stack).
set -eu
cd "$(dirname "$0")/../../../.."
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV
if [ -f .env ]; then set -a; . ./.env; set +a; fi
COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-nava-s29}
export COMPOSE_PROJECT_NAME
NAVA_IMAGE_TAG=${NAVA_IMAGE_TAG:-s29}
export NAVA_IMAGE_TAG
COMPOSE_PROFILES=${COMPOSE_PROFILES:-operator}
export COMPOSE_PROFILES
PHASE=${S29_PHASE:-all}
EV=docs/acceptance/S29/evidence
T=docs/acceptance/S29/tools
mkdir -p "$EV"
trap 'rm -f "$EV"/.bruno-*.json' EXIT
FAILED=
stamp() { date -u +%Y-%m-%dT%H:%M:%SZ; }

psqlf() {
  docker compose exec -T postgres sh -c \
    'PGPASSWORD=$(cat /run/pg_superuser_password) psql -X -U pgadmin -d nava -v ON_ERROR_STOP=1 "$@"' sh "$@"
}

if [ "$PHASE" != check ]; then
  find "$EV" -maxdepth 1 -type f -delete
  if [ "${S29_KEEP_STACK:-}" != 1 ]; then
    echo "s29: 1. fresh stack from $(git rev-parse HEAD) ($(stamp))"
    make dev-ca
    docker compose down -v
    docker compose up -d --build --wait
    env -u MSYS_NO_PATHCONV sh docs/acceptance/guide/tools/seed.sh
    env -u MSYS_NO_PATHCONV sh docs/acceptance/guide/tools/tunnels.sh up
    node docs/acceptance/S18/tools/kit.mjs admin-b
    docker compose run --rm --no-deps nava-groupsync --once
  fi
  echo "s29: 2. prep: setup, 2,000 incidents, 50 held, expectations ($(stamp))"
  node "$T/kit.mjs" setup "$EV"
  psqlf < "$T/seed.sql" | tee "$EV/seed.txt"
  node "$T/kit.mjs" held "$EV"
  { psqlf < "$T/expect.sql" | grep '^s29_'; cat "$EV/held.env"; } | tee "$EV/expect.txt"
  [ "$PHASE" = prep ] && { echo "s29: prep done ($(stamp))"; exit 0; }
fi

echo "s29: 3. Bruno folder S29 ($(stamp))"
envs=""
while IFS='=' read -r k v; do
  envs="$envs --env-var $k=$v"
done < "$EV/expect.txt"
# shellcheck disable=SC2086 # envs is a list of words, none with spaces
env -u MSYS_NO_PATHCONV sh docs/acceptance/guide/tools/api-test.sh S29 \
  --env-var "keycloak_issuer_url=${KEYCLOAK_PUBLIC_URL:-http://localhost:18081}" $envs || FAILED="$FAILED S29"
cp docs/acceptance/guide/reports/S29.json "$EV/.bruno-S29.json" 2>/dev/null || true

echo "s29: 4. the database behind criteria 2, 3 and 6 ($(stamp))"
psqlf < "$T/verify.sql" | tee "$EV/verify.txt"
if grep '^check|' "$EV/verify.txt" | grep -qv '|ok$' || ! grep -q '^check|' "$EV/verify.txt"; then
  echo "s29: verify FAILED" >&2
  FAILED="$FAILED verify"
fi
docker compose logs --no-log-prefix nava-worker 2>/dev/null | grep -iE 'schema.backfill|schema_backfill' | tail -n 200 > "$EV/worker-backfill-log.txt" || true

node docs/acceptance/S15/tools/evidence.mjs "$EV" || true
{ git rev-parse HEAD; git status --porcelain --untracked-files=no | sed 's/^/modified: /'; } > "$EV/build.txt"
docker compose images --format '{{.Repository}}:{{.Tag}} {{.ID}}' 2>/dev/null | grep '^nava/' | sort -u >> "$EV/build.txt" || true
if [ -n "$FAILED" ]; then echo "s29: FAILED:$FAILED at $(git rev-parse HEAD); evidence in $EV ($(stamp))" | tee "$EV/result.txt" >&2; exit 1; fi
echo "s29: PASS at $(git rev-parse HEAD); evidence in $EV ($(stamp))" | tee "$EV/result.txt"
