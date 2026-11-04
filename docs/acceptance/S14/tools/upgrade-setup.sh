#!/bin/sh
# S14 acceptance (TICK-437): builds the "before this release" state that AC2
# needs, then upgrades the same stack to the checkout this script runs from.
#
#   1. A fresh stack at PRE_REF (default origin/main, which has no audit_log),
#      seeded with that commit's tools/seed.sh, and incident version 2
#      published by admin@tenant-a.test.
#   2. The same stack (same volumes, same COMPOSE_PROJECT_NAME) rebuilt from
#      this checkout: the migrate service runs the S14 migrations, the
#      backfill among them.
#   3. The auditors-group reconcile (nava-onboard reconcile-auditors, TICK-567)
#      for the tenants that existed before S14, then the S14 realm delta
#      (auditor@tenant-a.test in A's auditors group), which
#      Keycloak does not import into a realm that already exists, then this
#      checkout's tools/seed.sh and tools/tunnels.sh up.
#
# Usage, from the repository root (Git Bash or any POSIX shell), Docker up:
#   sh docs/acceptance/S14/tools/upgrade-setup.sh
# It deletes the stack's volumes first (docker compose down -v).
set -eu
cd "$(dirname "$0")/../../../.."
ROOT=$(pwd)
PRE_REF=${PRE_REF:-origin/main}
COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-nava-s14}
export COMPOSE_PROJECT_NAME
PRE="$ROOT/../s14-pre-release"

echo "s14: pre-release worktree at $PRE_REF"
git fetch -q origin
[ -d "$PRE" ] || git worktree add -q --detach "$PRE" "$PRE_REF"
( cd "$PRE" && make dev-ca && docker compose --profile operator down -v \
  && docker compose --profile operator up -d --build --wait \
  && sh docs/acceptance/guide/tools/seed.sh )
node docs/acceptance/S14/tools/prepublish.mjs

echo "s14: upgrade the same stack to $(git rev-parse --short HEAD)"
mkdir -p deploy/compose/certs && cp -R "$PRE/deploy/compose/certs/." deploy/compose/certs/
docker compose --profile operator up -d --build --wait
docker compose logs migrate | grep '"migrate done"'
echo "s14: reconcile the auditors group of the tenants that existed before S14"
docker compose exec -T nava-onboard nava-onboard reconcile-auditors \
  00000000-0000-4000-8000-0000000000a1 00000000-0000-4000-8000-0000000000b1
node docs/acceptance/S14/tools/realm-delta.mjs
sh docs/acceptance/guide/tools/seed.sh
sh docs/acceptance/guide/tools/tunnels.sh up
curl -fsS http://127.0.0.1:19090/readyz && echo " s14: ready; run: sh docs/acceptance/guide/tools/api-test.sh S14A"
