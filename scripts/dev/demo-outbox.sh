#!/bin/sh
# The S7 demo (make demo-outbox, TICK-275): a committed outbox row runs exactly
# one tenant-scoped workflow. Writes a start row and two signal rows for a
# record of the seeded tenant tenant-a: the start through cascade.ClaimStart
# and outbox.EnqueueStart, the signals through outbox.Enqueue, in
# InTenantTx(tenant-a) (cmd/demo-outbox), as nava_app, from a throwaway
# container on the stack's internal network: the database port is never
# published. The relay hands the rows to Temporal. cascade_start has a foreign
# key to tenants (ADR-001 A5), so the script first seeds tenant-a's tenants
# row as the superuser; cmd/demo-outbox writes no tenants row (TICK-416).
# Usage: DEDUPE_KEY=<key> sh scripts/dev/demo-outbox.sh   (the stack is up)
set -eu
cd "$(dirname "$0")/../.."

# Git Bash would rewrite the container paths (--entrypoint /demo-outbox, docker cp).
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV

# Like make run: .env values (NAVA_APP_DB_PASSWORD, NAVA_IMAGE_TAG) apply as they do to compose.
if [ -f .env ]; then set -a; . ./.env; set +a; fi

PROJECT=${COMPOSE_PROJECT_NAME:-nava}
export COMPOSE_PROJECT_NAME="$PROJECT"
export DEDUPE_KEY=${DEDUPE_KEY:-demo-outbox}

net=$(docker network ls -q --filter "label=com.docker.compose.project=$PROJECT" --filter label=com.docker.compose.network=internal | head -n 1)
[ -n "$net" ] || { echo "demo-outbox: no internal network for compose project $PROJECT; is the stack up?" >&2; exit 1; }

# tenant-a's tenants row, as nava_onboard would have left it; a rerun keeps it.
TENANT_A=00000000-0000-4000-8000-0000000000a1
printf "INSERT INTO tenants (id, state) VALUES ('%s', 'active') ON CONFLICT DO NOTHING;\n" "$TENANT_A" |
	docker compose exec -T postgres sh -c 'PGPASSWORD=$(cat /run/pg_superuser_password) exec psql -X -q -v ON_ERROR_STOP=1 -U pgadmin -d nava -f -'

arch=$(docker version -f '{{.Server.Arch}}')
mkdir -p bin
CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -o bin/demo-outbox-linux ./cmd/demo-outbox

# The nava-core image is on the host already; a stopped container takes the binary by copy.
ctr=$(docker create --network "$net" --entrypoint /demo-outbox \
    -e DEDUPE_KEY -e "NAVA_DATABASE_URL=postgres://nava_app:${NAVA_APP_DB_PASSWORD:-dev-only-app}@pgbouncer:5432/nava?sslmode=disable" \
    "nava/nava-core:${NAVA_IMAGE_TAG:-dev}")
status=0
{ docker cp bin/demo-outbox-linux "$ctr":/demo-outbox && docker start -a "$ctr"; } || status=$?
docker rm -f "$ctr" >/dev/null
[ "$status" -eq 0 ] || exit "$status"

cat <<TXT

The relay starts the workflow and delivers the two signals within a few seconds.
To see them in the operator Temporal UI:
  1. make operator-tunnel        (leave it running; needs COMPOSE_PROFILES=operator, the stack default)
  2. open http://127.0.0.1:18233, namespace "nava"
  3. open the workflow id above: its history shows the start and two "demo.note"
     signals in commit order. No worker runs this workflow type in the stack, so
     it stays Running.
Run it again with the same DEDUPE_KEY: the same ids, still one workflow.
A new DEDUPE_KEY=<key> makes a new record and a new workflow.
TXT
