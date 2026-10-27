#!/bin/sh
# Integration stage (T05): every package with `//go:build integration` tests,
# against throwaway PostgreSQL clusters the script starts and removes. The same
# on a laptop and in CI; needs Docker. Extra arguments go to `go test`
# (e.g. -run TestCICanary) and run every package one after another on one
# cluster, as before TICK-153.
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

# NAVA_IT_ID (digits, default this shell's pid) names this run's containers
# and network; make test-integration-cleanup sets it to find them.
id=${NAVA_IT_ID:-$$}
case $id in '' | *[!0-9]*) echo "NAVA_IT_ID=$id: digits only" >&2; exit 2 ;; esac
name="nava-it-pg-$id"
net="nava-it-net-$id"
pooled="nava-it-pgb-pooled-$id"
noprep="nava-it-pgb-noprep-$id"
bouncers="$pooled $noprep"
# dbtest runs as DBTEST_SHARDS processes, each on a cluster of its own
# (TICK-153 O3): its tests drop and create cluster-wide roles, so no two of
# them may share a cluster.
shards=${DBTEST_SHARDS:-3}
[ $# -eq 0 ] || shards=0
shard_pgs=""
i=1
while [ "$i" -le "$shards" ]; do shard_pgs="$shard_pgs nava-it-pg-$id-s$i"; i=$((i + 1)); done
# Every go test gets a -timeout well below the integration job's
# timeout-minutes (10), so a hung test panics with Go's goroutine dump, which
# names it, before GitHub cancels the step with an empty log (TICK-225). The
# tests take under a minute and start about two minutes into the job. A later
# -timeout in the script's arguments wins.
timeout=6m
logs=$(mktemp -d)
pipes=$(mktemp -d)
# This script owns the clusters, the network and the logs: remove them on any
# exit. INT and TERM exit through the EXIT trap, which dash skips on a bare
# signal, after stopping the go commands still running. Never `exec` the last
# command: exec drops the EXIT trap, and every `ARGS=...` run left its
# clusters and network behind (TICK-185). Only SIGKILL skips the trap; make
# reap-integration removes what such a run left.
fgpid=""
procs=""
printers=""
stop() { for j in $fgpid $procs; do kill "${j%%:*}" 2>/dev/null || true; done; }
# shellcheck disable=SC2086
trap 'docker rm -f "$name" $shard_pgs $bouncers >/dev/null 2>&1 || true; docker network rm "$net" >/dev/null 2>&1 || true; rm -rf "$logs" "$pipes"' EXIT
trap 'stop; exit 130' INT
trap 'stop; exit 143' TERM
# fgrun <command>: runs it in the background and waits, so INT and TERM reach the
# trap at once; the shell runs a trap only after a foreground command ends.
fgrun() { "$@" & fgpid=$!; wait "$fgpid"; }

t0=$(date +%s)
docker network create "$net" >/dev/null
# Throwaway clusters, so no durability (TICK-153 O1): a crash loses nothing
# that matters, and fsync dominates the dbtest suites' time. Never set these
# on the dev or production compose stack.
start_pg() { # start_pg <name> [docker run options]
    n=$1
    shift
    docker run -d --name "$n" --network "$net" "$@" \
        -e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=test-only -e POSTGRES_INITDB_ARGS=--no-sync \
        --health-cmd 'pg_isready -h 127.0.0.1 -U pgadmin -d postgres' \
        --health-interval 1s --health-timeout 5s --health-retries 90 \
        -p 127.0.0.1::5432 "$NAVA_TEST_PG_IMAGE" \
        -c fsync=off -c synchronous_commit=off -c full_page_writes=off >/dev/null
}
start_pg "$name" --network-alias pg
for c in $shard_pgs; do start_pg "$c"; done

# Two PgBouncers in front of the shared cluster (T06): pooled.ini is the
# setting under test, noprep.ini the same with max_prepared_statements = 0
# (the negative case).
for b in $bouncers; do
    docker create --name "$b" --network "$net" \
        --health-cmd 'pg_isready -h 127.0.0.1 -p 5432' \
        --health-interval 1s --health-timeout 5s --health-retries 90 \
        -p 127.0.0.1::5432 "$bouncer_image" >/dev/null
    if [ "$b" = "$pooled" ]; then ini=pooled.ini; else ini=noprep.ini; fi
    docker cp scripts/ci/pgbouncer/$ini "$b:/etc/pgbouncer/pgbouncer.ini"
    docker cp scripts/ci/pgbouncer/userlist.txt "$b:/etc/pgbouncer/userlist.txt"
    docker start "$b" >/dev/null
done

pkgs=$(grep -rl --include='*_test.go' '^//go:build integration' cmd internal tools 2>/dev/null |
    xargs -n 1 dirname | sort -u | sed 's|^|./|' |
    # The isolation suite needs its own cluster and PgBouncer: make isolation.
    grep -v '^\./internal/tenancy/isolation$')
echo "integration packages ($NAVA_TEST_PG_IMAGE):" $pkgs
# Compile every test binary, and the log printer, while the clusters start;
# -run '^$' runs nothing.
# shellcheck disable=SC2086
fgrun go test -tags integration -count=1 -run '^$' $pkgs >/dev/null
fgrun go run ./tools/ci/testlog warm-up </dev/null

# Gate on each container's health check (postgres: TCP, since the image's
# init server listens on the socket only, so a TCP answer means the real
# server is up).
deadline=$(( $(date +%s) + 120 ))
for c in "$name" $shard_pgs $bouncers; do
    until [ "$(docker inspect -f '{{.State.Health.Status}}' "$c")" = healthy ]; do
        [ "$(date +%s)" -lt "$deadline" ] || { docker logs "$c"; echo "$c not healthy within 120s" >&2; exit 1; }
        sleep 1
    done
done
echo "integration: clusters healthy after $(( $(date +%s) - t0 ))s"
hostport() { docker port "$1" 5432/tcp | head -n 1 | sed 's/.*://'; }
pg_url() { echo "postgres://pgadmin:test-only@127.0.0.1:$(hostport "$1")/postgres"; }
NAVA_TEST_PG_URL=$(pg_url "$name")
# The role and database nava_it_pool are created by the load test itself.
NAVA_TEST_PGBOUNCER_URL="postgres://nava_it_pool:test-only-pool@127.0.0.1:$(hostport "$pooled")/nava_it_pool"
NAVA_TEST_PGBOUNCER_NOPREP_URL="postgres://nava_it_pool:test-only-pool@127.0.0.1:$(hostport "$noprep")/nava_it_pool"
export NAVA_TEST_PG_URL NAVA_TEST_PGBOUNCER_URL NAVA_TEST_PGBOUNCER_NOPREP_URL

# -v: the PgBouncer load test logs its SHOW POOLS / SHOW STATS evidence (T06).
if [ "$shards" -eq 0 ]; then
    # -p 1: the packages share the cluster's roles.
    # shellcheck disable=SC2086
    fgrun go test -tags integration -p 1 -count=1 -timeout "$timeout" -v "$@" $pkgs
    exit 0
fi

# Three groups run at the same time:
#   dbtest  its top-level tests dealt round-robin over the shard clusters
#   shared  the other packages that use the shared cluster or its PgBouncers,
#           -p 1 since they create the same roles
#   own     the packages that use neither (health starts its own container)
shared=""
own=""
for p in $pkgs; do
    [ "$p" = ./internal/db/dbtest ] && continue
    if grep -q -e NAVA_TEST_PG_URL -e NAVA_TEST_PGBOUNCER "$p"/*_test.go; then shared="$shared $p"; else own="$own $p"; fi
done
tests=$(go test -tags integration -list . ./internal/db/dbtest | grep '^Test')
total=$(echo "$tests" | wc -l | tr -d ' ')
echo "dbtest: $total tests over $shards clusters; shared cluster:$shared; own cluster:$own"

# run <group> <go test arguments>: in the background. The output streams to
# the log as it arrives, each line prefixed with [<group>] (tools/ci/testlog),
# so a cancelled step still shows it (TICK-225), and the -json stream is kept
# in $logs/<group> for the checks below. go test writes into a FIFO the
# printer reads, so $! is go test itself: wait gets its exit code and stop
# kills it, after which the printer reads EOF and ends.
run() {
    l=$1
    shift
    mkfifo "$pipes/$l"
    tee "$logs/$l" <"$pipes/$l" | go run ./tools/ci/testlog "$l" &
    printers="$printers $!:$l"
    go test -tags integration -count=1 -timeout "$timeout" -json "$@" >"$pipes/$l" 2>&1 &
    procs="$procs $!:$l"
}
i=1
for c in $shard_pgs; do
    # Tests i, i+shards, i+2*shards, ... of the sorted list.
    sel=$(echo "$tests" | awk -v n="$shards" -v i="$i" 'NR % n == i % n' | paste -s -d '|' -)
    NAVA_TEST_PG_URL=$(pg_url "$c") run "dbtest-$i" -run "^($sel)\$" ./internal/db/dbtest
    i=$((i + 1))
done
# shellcheck disable=SC2086
[ -z "$shared" ] || run shared -p 1 $shared
# shellcheck disable=SC2086
[ -z "$own" ] || run own $own

status=0
failed=""
for j in $procs; do
    wait "${j%%:*}" || { status=1; failed="$failed ${j#*:}"; }
done
for j in $printers; do
    wait "${j%%:*}" || { echo "integration: testlog failed for ${j#*:}" >&2; status=1; }
done
echo "integration: tests done after $(( $(date +%s) - t0 ))s"
# The groups' output was streamed above; this repeats, per group, the package
# results and the output of every failed test, then checks that every
# registered isolation suite in these packages ran a test (tools/ci/suiteran,
# TICK-230): the groups are checked together, since a dbtest shard runs only
# some of the package's tests.
fgrun go run ./tools/ci/suiteran -tags integration -in "$logs"/* || status=1
# Every dbtest test ran, exactly once over the shards.
ran=$(grep -h '"Action":"run"' "$logs"/dbtest-* | grep -vc '"Test":"[^"]*/' || true)
echo "dbtest: $ran of $total top-level tests ran"
[ "$ran" -eq "$total" ] || { echo "dbtest: the shards ran $ran tests, want $total" >&2; status=1; }
[ "$status" -eq 0 ] || echo "integration: failed groups:$failed" >&2
exit "$status"
