#!/bin/sh
# make test-integration-cleanup (TICK-185): scripts/ci/test-integration.sh
# leaves none of its nava-it-* containers or its network behind when a run
# with go test arguments ends (the path that once ended in `exec` and dropped
# the EXIT trap), or when TERM, sent to the script alone, stops a run part way:
# its go commands get no signal, so the script must stop them itself. INT runs
# the same trap, but a background job starts with INT ignored, which a shell
# cannot trap, so it is not tested here. Each run filters for a test that does
# not exist, so it starts the shared cluster and its PgBouncers and runs
# nothing. NAVA_IT_ID names each run's resources. Needs Docker.
#
# TICK-182: a run beside the TERM-stopped one still passes (a run's teardown
# removes only its own nava-it-* objects), the cluster and network carry the
# run's labels, and a run whose cluster is removed under it fails at once
# with a clear message instead of polling a missing container.
set -eu
cd "$(dirname "$0")/../.."

fail=0
pids=""
ids=""
base=$(date +%s)
check() { if eval "$2"; then echo "ok:   $1"; else echo "FAIL: $1" >&2; fail=1; fi; }
# left <id>: the containers and network of the run with that NAVA_IT_ID.
left() {
    { docker ps -a --format '{{.Names}}'; docker network ls --format '{{.Name}}'; } |
        grep -E "^nava-it-(pg|pgb-pooled|pgb-noprep|net)-$1(-s[0-9]+)?\$" || true
}
# Whatever a failed check left is this test's own: remove it.
cleanup() {
    for p in $pids; do kill "$p" 2>/dev/null || true; done
    for i in $ids; do
        for r in $(left "$i"); do docker rm -f "$r" >/dev/null 2>&1 || docker network rm "$r" >/dev/null 2>&1 || true; done
    done
}
trap cleanup EXIT
trap 'exit 130' INT TERM

start() { # start <n> [log]: run n in the background, NAVA_IT_ID <base><n>
    id="$base$1"
    ids="$ids $id"
    NAVA_IT_ID=$id sh scripts/ci/test-integration.sh -run '^TestTICK185NoSuchTest$' >"${2:-/dev/null}" 2>&1 &
    pid=$!
    pids="$pids $pid"
}

start 1
st=0
wait "$pid" || st=$?
check "a run with go test arguments exits 0 (got $st)" '[ "$st" -eq 0 ]'
check "a run with go test arguments leaves nothing: $(left "$id" | tr '\n' ' ')" '[ -z "$(left "$id")" ]'

# wait_cluster <id>: until the run with that NAVA_IT_ID has its shared
# cluster; it is then starting the rest or compiling, with something to remove.
wait_cluster() { # wait_cluster <id> [container, default the cluster]
    deadline=$(( $(date +%s) + 180 ))
    until docker ps -a --format '{{.Names}}' | grep -qx "${2:-nava-it-pg-$1}"; do
        [ "$(date +%s)" -lt "$deadline" ] || { echo "FAIL: run $1 started no cluster within 180s" >&2; exit 1; }
        sleep 1
    done
}

# Run 3 beside run 2 (TICK-182): run 2's TERM teardown leaves run 3 alone.
start 3
pid3=$pid
id3=$id
sig=TERM
start 2
wait_cluster "$id"
wait_cluster "$id3"
lbl=$(docker inspect -f '{{index .Config.Labels "nava.ci.run"}} {{index .Config.Labels "nava.ci.owner"}}' "nava-it-pg-$id")
check "the cluster carries the run's labels (got $lbl)" '[ "$lbl" = "nava-it-$id ${COMPOSE_PROJECT_NAME:-none}" ]'
nl=$(docker network inspect -f '{{index .Labels "nava.ci.run"}}' "nava-it-net-$id")
check "the network carries the run's label (got $nl)" '[ "$nl" = "nava-it-$id" ]'
kill "-$sig" "$pid" || { echo "FAIL: run $id ended before $sig" >&2; exit 1; }
st=0
wait "$pid" || st=$?
check "$sig stops the run with a non-zero status (got $st)" '[ "$st" -ne 0 ]'
check "$sig leaves nothing: $(left "$id" | tr '\n' ' ')" '[ -z "$(left "$id")" ]'
st=0
wait "$pid3" || st=$?
check "the run beside it still exits 0 (got $st)" '[ "$st" -eq 0 ]'
check "the run beside it leaves nothing: $(left "$id3" | tr '\n' ' ')" '[ -z "$(left "$id3")" ]'

# Run 4's cluster removed under it (TICK-182): it fails at once and says so,
# well before the 120 s health deadline.
log4=$(mktemp)
start 4 "$log4"
# Its last PgBouncer exists: the docker run of the cluster has returned, so
# removing it hits the health gate, not docker run itself.
wait_cluster "$id" "nava-it-pgb-noprep-$id"
t=$(date +%s)
docker rm -f "nava-it-pg-$id" >/dev/null
st=0
wait "$pid" || st=$?
took=$(( $(date +%s) - t ))
check "a removed cluster fails the run (got $st)" '[ "$st" -ne 0 ]'
check "a removed cluster is reported as vanished: $(tail -n 2 "$log4" | tr '\n' ' ')" \
    'grep -q "container nava-it-pg-$id of run $id vanished" "$log4"'
check "a removed cluster fails within 100s (took ${took}s)" '[ "$took" -lt 100 ]'
check "a removed cluster's run leaves nothing: $(left "$id" | tr '\n' ' ')" '[ -z "$(left "$id")" ]'
rm -f "$log4"
exit "$fail"
