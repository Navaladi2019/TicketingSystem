#!/bin/sh
# make isolation (T11, TICK-67): the tenant isolation suite
# (internal/tenancy/isolation) against a throwaway PostgreSQL and a PgBouncer
# in transaction mode that logs the runtime roles in with the real auth_query
# (scripts/ci/pgbouncer/isolation.ini). The suite rebuilds the roles and the
# platform database itself, seeds tenants A and B, and writes its report to
# ISOLATION_REPORT (default isolation-report.md at the repository root), the
# CI artifact. Needs Docker. Extra arguments go to `go test`.
set -eu
cd "$(dirname "$0")/../.."

# Images from docs/pins.md, as scripts/ci/test-integration.sh reads them.
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' docs/pins.md)
[ -n "$tag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
bouncer_tag=$(sed -n 's/^| `edoburu\/pgbouncer` | \([^ |]*\) |.*/\1/p' docs/pins.md)
[ -n "$bouncer_tag" ] || { echo "no edoburu/pgbouncer row in docs/pins.md" >&2; exit 1; }
# Git Bash would rewrite the container paths given to docker cp.
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV

pg="nava-iso-pg-$$"
bouncer="nava-iso-pgb-$$"
net="nava-iso-net-$$"
# INT and TERM stop the running go command and exit through the EXIT trap.
# Never `exec` the last command: exec drops the EXIT trap (TICK-185).
fgpid=""
trap 'docker rm -f "$pg" "$bouncer" >/dev/null 2>&1 || true; docker network rm "$net" >/dev/null 2>&1 || true' EXIT
trap '[ -z "$fgpid" ] || kill "$fgpid" 2>/dev/null; exit 130' INT
trap '[ -z "$fgpid" ] || kill "$fgpid" 2>/dev/null; exit 143' TERM
# fgrun <command>: as in test-integration.sh, so a signal reaches the trap at once.
fgrun() { "$@" & fgpid=$!; wait "$fgpid"; }

docker network create "$net" >/dev/null
# Throwaway cluster, so no durability (as test-integration.sh).
docker run -d --name "$pg" --network "$net" --network-alias pg \
    -e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=test-only -e POSTGRES_INITDB_ARGS=--no-sync \
    --health-cmd 'pg_isready -h 127.0.0.1 -U pgadmin -d postgres' \
    --health-interval 1s --health-timeout 5s --health-retries 90 \
    -p 127.0.0.1::5432 "postgres:$tag" \
    -c fsync=off -c synchronous_commit=off -c full_page_writes=off >/dev/null
docker create --name "$bouncer" --network "$net" \
    --health-cmd 'pg_isready -h 127.0.0.1 -p 5432' \
    --health-interval 1s --health-timeout 5s --health-retries 90 \
    -p 127.0.0.1::5432 "edoburu/pgbouncer:$bouncer_tag" >/dev/null
docker cp scripts/ci/pgbouncer/isolation.ini "$bouncer:/etc/pgbouncer/pgbouncer.ini"
docker cp scripts/ci/pgbouncer/isolation-userlist.txt "$bouncer:/etc/pgbouncer/userlist.txt"
docker start "$bouncer" >/dev/null

# Compile while the containers start (TestMain needs the cluster, so not
# go test -run '^$').
fgrun go vet -tags integration ./internal/tenancy/isolation

deadline=$(( $(date +%s) + 120 ))
for c in "$pg" "$bouncer"; do
    until [ "$(docker inspect -f '{{.State.Health.Status}}' "$c")" = healthy ]; do
        [ "$(date +%s)" -lt "$deadline" ] || { docker logs "$c"; echo "$c not healthy within 120s" >&2; exit 1; }
        sleep 1
    done
done
hostport() { docker port "$1" 5432/tcp | head -n 1 | sed 's/.*://'; }
NAVA_TEST_PG_URL="postgres://pgadmin:test-only@127.0.0.1:$(hostport "$pg")/postgres"
NAVA_ISOLATION_PGBOUNCER_ADDR="127.0.0.1:$(hostport "$bouncer")"
# pwd -W: a path Go on Windows (Git Bash) can open.
NAVA_ISOLATION_REPORT="${ISOLATION_REPORT:-$(pwd -W 2>/dev/null || pwd)/isolation-report.md}"
export NAVA_TEST_PG_URL NAVA_ISOLATION_PGBOUNCER_ADDR NAVA_ISOLATION_REPORT

# With no extra arguments, tools/ci/suiteran (TICK-230) also fails the run
# when a registered isolation suite ran zero tests. Extra arguments (a -run
# filter) may leave a suite out on purpose, so they skip that check.
if [ $# -eq 0 ]; then
    fgrun go run ./tools/ci/suiteran -tags integration -v -- go test -tags integration -count=1 -json ./internal/tenancy/isolation
    exit 0
fi
fgrun go test -tags integration -count=1 -v ./internal/tenancy/isolation "$@"
