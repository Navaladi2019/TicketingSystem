#!/bin/sh
# S25 acceptance walk (TICK-830). Run from the repository root on a fresh, healthy stack
# (docker compose down -v && docker compose up -d --wait; see README.md in this directory).
# Seeds tenants A and B, packs the SDK, installs the tarball into examples/sdk-node, runs the
# ten-line example and the criteria walk, and counts the replayed create's audit rows in Postgres.
# Prints no token or password. Exit status is non-zero on any failure.
set -eu
A=00000000-0000-4000-8000-0000000000a1
step() { printf '\n== %s\n' "$*"; }

step "readyz (nava-core admin listener)"
docker compose exec -T nava-core wget -qO- http://127.0.0.1:9090/readyz; echo

step "seed tenants A and B"
sh docs/acceptance/guide/tools/seed.sh | tail -3

step "pack the SDK (task sdk:build)"
(cd sdk/ts && npm ci --silent && npm run build --silent && npm pack --silent)

step "install the tarball into examples/sdk-node"
(cd examples/sdk-node && rm -rf node_modules && npm install --no-audit --no-fund --silent && npm ls @nava/sdk)

step "the ten-line example, as alice@tenant-a.test: list, create, get"
(cd examples/sdk-node && node list-read-create.mjs)
step "the example twice more with one idempotency key: the second create answers the same record"
(cd examples/sdk-node && node list-read-create.mjs "s25-kit-key-$$" | tail -1 && node list-read-create.mjs "s25-kit-key-$$" | tail -1)

step "criteria 1 to 4 and the isolation checks, tenants A and B"
out=$(mktemp)
trap 'rm -f "$out"' EXIT
(cd examples/sdk-node && S25_REPLAY_ID_OUT="$out" node criteria.mjs)

step "criterion 2 in Postgres: audit_log rows for the replayed create"
id=$(tr -d '\r\n ' <"$out")
# shellcheck disable=SC2016 # expanded inside the container
n=$(docker compose exec -T postgres sh -c 'PGPASSWORD=$(cat /run/pg_superuser_password) exec psql -X -q -t -A -U pgadmin -d nava -c "$0"' \
  "SELECT count(*) FROM audit_log WHERE tenant_id = '$A' AND subject_type = 'record' AND subject_id = '$id'" | tr -d '\r ')
echo "record $id: $n audit_log row(s)"
[ "$n" = 1 ] || { echo "FAIL: want 1"; exit 1; }
echo "PASS  C2 one audit_log row"
