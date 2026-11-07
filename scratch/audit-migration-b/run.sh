#!/bin/sh
# P5-32 (TICK-442) migration B timing. Needs Docker and Go.
# Usage: run.sh [tenant counts, comma-separated; default 1000,10000]
# Starts postgres (tag from docs/pins.md), then runs the opt-in
# TestP532MigrationBTiming (internal/db/dbtest/audit_migration_b_timing_test.go)
# against it, which builds each size on the real migrations and times B.
# Output: out/timing.txt.
set -eu
cd "$(dirname "$0")"
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' ../../docs/pins.md)
[ -n "$tag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
MSYS_NO_PATHCONV=1; export MSYS_NO_PATHCONV
name=nava-migration-b-$$
trap 'docker rm -f "$name" >/dev/null 2>&1 || true' EXIT
# The same throwaway settings as scripts/ci/test-integration.sh.
docker run -d --name "$name" -e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=test-only \
    -e POSTGRES_INITDB_ARGS=--no-sync -p 127.0.0.1::5432 "postgres:$tag" \
    -c fsync=off -c synchronous_commit=off -c full_page_writes=off >/dev/null
until docker exec "$name" pg_isready -h 127.0.0.1 -U pgadmin -d postgres >/dev/null 2>&1; do sleep 1; done
sleep 2
port=$(docker port "$name" 5432/tcp | head -n 1 | sed 's/.*://')
mkdir -p out
{
  echo "# $(date -u +%FT%TZ) postgres:$tag, host: $(uname -srm)"
  docker exec "$name" nproc | sed 's/^/# container cpus: /'
} > out/env.txt
cd ../..
NAVA_TEST_PG_URL="postgres://pgadmin:test-only@127.0.0.1:$port/postgres" \
NAVA_BENCH_MIGRATION_B="${1:-1000,10000}" \
    go test -tags integration -count=1 -timeout 30m -v -run '^TestP532MigrationBTiming$' ./internal/db/dbtest/ |
    tee scratch/audit-migration-b/out/timing.txt
