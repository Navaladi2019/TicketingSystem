#!/bin/sh
# S22 + S29 acceptance (TICK-754): the ONE fresh-stack run that covers both
# stories (they share the story branch TICK-740/search-finish).
#
#   1. A fresh stack (operator profile), tools/seed.sh, tools/tunnels.sh up,
#      admin@tenant-b (S18's kit.mjs admin-b, one group sync).
#   2. S29 prep (S29_PHASE=prep): its incidents exist before S22 runs, so
#      S22's AC3 priority filters on the seed type compare real rows.
#   3. S22 (S22_KEEP_STACK=1): criteria 1 to 4.
#   4. S29 check (S29_PHASE=check): criteria 1 to 6.
#
# Usage, from the repository root (Git Bash or any POSIX shell), Docker up,
# Node.js 22:
#   sh docs/acceptance/S22/tools/run-s22-s29.sh
set -eu
cd "$(dirname "$0")/../../../.."
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV
if [ -f .env ]; then set -a; . ./.env; set +a; fi
COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-nava-s22s29}
NAVA_IMAGE_TAG=${NAVA_IMAGE_TAG:-s22s29}
COMPOSE_PROFILES=${COMPOSE_PROFILES:-operator}
export COMPOSE_PROJECT_NAME NAVA_IMAGE_TAG COMPOSE_PROFILES
stamp() { date -u +%Y-%m-%dT%H:%M:%SZ; }

echo "s22+s29: 1. fresh stack from $(git rev-parse HEAD) ($(stamp))"
make dev-ca
docker compose down -v
docker compose up -d --build --wait
env -u MSYS_NO_PATHCONV sh docs/acceptance/guide/tools/seed.sh
env -u MSYS_NO_PATHCONV sh docs/acceptance/guide/tools/tunnels.sh up
node docs/acceptance/S18/tools/kit.mjs admin-b
docker compose run --rm --no-deps nava-groupsync --once

FAILED=
S29_KEEP_STACK=1 S29_PHASE=prep sh docs/acceptance/S29/tools/run-kit.sh || FAILED="$FAILED s29-prep"
S22_KEEP_STACK=1 sh docs/acceptance/S22/tools/run-kit.sh || FAILED="$FAILED s22"
S29_KEEP_STACK=1 S29_PHASE=check sh docs/acceptance/S29/tools/run-kit.sh || FAILED="$FAILED s29"
if [ -n "$FAILED" ]; then echo "s22+s29: FAILED:$FAILED at $(git rev-parse HEAD) ($(stamp))" >&2; exit 1; fi
echo "s22+s29: PASS at $(git rev-parse HEAD) ($(stamp))"
