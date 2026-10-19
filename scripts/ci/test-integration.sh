#!/bin/sh
# Integration stage (T05): every package with `//go:build integration` tests,
# against a throwaway PostgreSQL the script starts and removes. The same on a
# laptop and in CI; needs Docker. Extra arguments go to `go test`
# (e.g. -run TestCICanary).
set -eu
cd "$(dirname "$0")/../.."

# The image tag comes from the `postgres` row of docs/pins.md, never from here
# (TICK-94 item 16); tests that start their own container read it too.
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' docs/pins.md)
[ -n "$tag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
NAVA_TEST_PG_IMAGE="postgres:$tag"
export NAVA_TEST_PG_IMAGE
# The PgBouncer load test (T06, internal/db/pooltest) uses the compose pin.
bouncer_tag=$(sed -n 's/^| `edoburu\/pgbouncer` | \([^ |]*\) |.*/\1/p' docs/pins.md)
[ -n "$bouncer_tag" ] || { echo "no edoburu/pgbouncer row in docs/pins.md" >&2; exit 1; }
bouncer_image="edoburu/pgbouncer:$bouncer_tag"
# Git Bash would rewrite the container paths given to docker cp.
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV

name="nava-it-pg-$$"
net="nava-it-net-$$"
pooled="nava-it-pgb-pooled-$$"
noprep="nava-it-pgb-noprep-$$"
bouncers="$pooled $noprep"
# This script owns the shared containers and network: remove them on any exit.
# INT and TERM exit through the EXIT trap, which dash skips on a bare signal.
trap 'docker rm -f "$name" $bouncers >/dev/null 2>&1 || true; docker network rm "$net" >/dev/null 2>&1 || true' EXIT
trap 'exit 130' INT TERM

docker network create "$net" >/dev/null
docker run -d --name "$name" --network "$net" --network-alias pg \
    -e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=test-only \
    --health-cmd 'pg_isready -h 127.0.0.1 -U pgadmin -d postgres' \
    --health-interval 1s --health-timeout 5s --health-retries 90 \
    -p 127.0.0.1::5432 "$NAVA_TEST_PG_IMAGE" >/dev/null

# Two PgBouncers in front of it (T06): pooled.ini is the setting under test,
# noprep.ini the same with max_prepared_statements = 0 (the negative case).
for b in $bouncers; do
    docker create --name "$b" --network "$net"         --health-cmd 'pg_isready -h 127.0.0.1 -p 5432'         --health-interval 1s --health-timeout 5s --health-retries 90         -p 127.0.0.1::5432 "$bouncer_image" >/dev/null
    if [ "$b" = "$pooled" ]; then ini=pooled.ini; else ini=noprep.ini; fi
    docker cp scripts/ci/pgbouncer/$ini "$b:/etc/pgbouncer/pgbouncer.ini"
    docker cp scripts/ci/pgbouncer/userlist.txt "$b:/etc/pgbouncer/userlist.txt"
    docker start "$b" >/dev/null
done

# Gate on each container's health check (postgres: TCP, since the image's
# init server listens on the socket only, so a TCP answer means the real
# server is up).
deadline=$(( $(date +%s) + 120 ))
for c in "$name" $bouncers; do
    until [ "$(docker inspect -f '{{.State.Health.Status}}' "$c")" = healthy ]; do
        [ "$(date +%s)" -lt "$deadline" ] || { docker logs "$c"; echo "$c not healthy within 120s" >&2; exit 1; }
        sleep 1
    done
done
hostport() { docker port "$1" 5432/tcp | head -n 1 | sed 's/.*://'; }
NAVA_TEST_PG_URL="postgres://pgadmin:test-only@127.0.0.1:$(hostport "$name")/postgres"
# The role and database nava_it_pool are created by the load test itself.
NAVA_TEST_PGBOUNCER_URL="postgres://nava_it_pool:test-only-pool@127.0.0.1:$(hostport "$pooled")/nava_it_pool"
NAVA_TEST_PGBOUNCER_NOPREP_URL="postgres://nava_it_pool:test-only-pool@127.0.0.1:$(hostport "$noprep")/nava_it_pool"
export NAVA_TEST_PG_URL NAVA_TEST_PGBOUNCER_URL NAVA_TEST_PGBOUNCER_NOPREP_URL

pkgs=$(grep -rl --include='*_test.go' '^//go:build integration' cmd internal tools 2>/dev/null |
    xargs -n 1 dirname | sort -u | sed 's|^|./|')
echo "integration packages ($NAVA_TEST_PG_IMAGE):" $pkgs
# -p 1: the dbtest suites share the cluster's roles.
# -v: the PgBouncer load test logs its SHOW POOLS / SHOW STATS evidence (T06).
# shellcheck disable=SC2086
go test -tags integration -p 1 -count=1 -v "$@" $pkgs
