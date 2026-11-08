#!/bin/sh
# S15 acceptance (TICK-443): a fresh stack whose seeded tenants A and B get
# their chain head the way an existing production tenant does, from
# migration B (P5-32), then upgraded in place to this checkout.
#
#   1. A fresh stack (volumes deleted) at PRE_REF, default 080dd74: the story
#      branch just before migration B. Seeded with that commit's
#      tools/seed.sh: tenants A and B, with no audit_chain_head row.
#   2. The same stack (same volumes, same COMPOSE_PROJECT_NAME) rebuilt from
#      this checkout. The migrate service applies migration B, which seeds A's
#      and B's head and first audit.seal start row and re-runs the backfill.
#   3. This checkout's tools/seed.sh (idempotent) and tools/tunnels.sh up,
#      then /readyz on nava-core and nava-worker.
#
# Why not a stack created at this checkout: tools/seed.sh writes tenants A
# and B after the migrations ran, and nothing gives them a chain head
# (onboarding does that for new tenants only). Their entries would never be
# sealed. See docs/acceptance/S15/README.md, "Known limits".
#
# Usage, from the repository root (Git Bash or any POSIX shell), Docker up:
#   sh docs/acceptance/S15/tools/upgrade-setup.sh
# Ports and the project name come from .env, as compose reads them. It
# deletes the stack's volumes first (docker compose down -v).
set -eu
cd "$(dirname "$0")/../../../.."
ROOT=$(pwd)
if [ -f .env ]; then set -a; . ./.env; set +a; fi
PRE_REF=${PRE_REF:-080dd74}
COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-nava-s15}
export COMPOSE_PROJECT_NAME
PRE="$ROOT/../s15-pre-b"
ADMIN_PORT=${ACCEPT_ADMIN_PORT:-19090}

echo "s15: pre-migration-B worktree at $PRE_REF"
[ -d "$PRE" ] || git worktree add -q --detach "$PRE" "$PRE_REF"
make dev-ca
mkdir -p "$PRE/deploy/compose/certs" && cp -R deploy/compose/certs/. "$PRE/deploy/compose/certs/"
# The pre-B images get their own tag, so the upgrade below rebuilds and
# recreates every nava/* service from this checkout.
[ -f .env ] && cp .env "$PRE/.env"
( cd "$PRE" && export NAVA_IMAGE_TAG=s15pre \
  && docker compose down -v \
  && docker compose up -d --build --wait \
  && sh docs/acceptance/guide/tools/seed.sh )
docker compose exec -T postgres sh -c \
  'PGPASSWORD=$(cat /run/pg_superuser_password) psql -U pgadmin -d nava -Atc "SELECT count(*) AS heads_before_b FROM audit_chain_head"'

echo "s15: upgrade the same stack to $(git rev-parse --short HEAD)"
docker compose up -d --build --wait
docker compose logs migrate | grep '"migrate done"'
sh docs/acceptance/guide/tools/seed.sh
sh docs/acceptance/guide/tools/tunnels.sh up
curl -fsS "http://127.0.0.1:$ADMIN_PORT/readyz" && echo " s15: nava-core ready"
curl -fsS "http://127.0.0.1:$((ADMIN_PORT + 4))/readyz" && echo " s15: nava-worker ready"
echo "s15: next: sh docs/acceptance/S15/tools/run-kit.sh"
