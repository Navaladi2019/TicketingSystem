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

start() { # start <n>: run n in the background, NAVA_IT_ID <base><n>
    id="$base$1"
    ids="$ids $id"
    NAVA_IT_ID=$id sh scripts/ci/test-integration.sh -run '^TestTICK185NoSuchTest$' >/dev/null 2>&1 &
    pid=$!
    pids="$pids $pid"
}

start 1
st=0
wait "$pid" || st=$?
check "a run with go test arguments exits 0 (got $st)" '[ "$st" -eq 0 ]'
check "a run with go test arguments leaves nothing: $(left "$id" | tr '\n' ' ')" '[ -z "$(left "$id")" ]'

sig=TERM
start 2
# Signal once the run has its cluster: it is then starting the rest or
# compiling, with something to remove.
deadline=$(( $(date +%s) + 180 ))
until docker ps -a --format '{{.Names}}' | grep -qx "nava-it-pg-$id"; do
    [ "$(date +%s)" -lt "$deadline" ] || { echo "FAIL: run $id started no cluster within 180s" >&2; exit 1; }
    sleep 1
done
kill "-$sig" "$pid" || { echo "FAIL: run $id ended before $sig" >&2; exit 1; }
st=0
wait "$pid" || st=$?
check "$sig stops the run with a non-zero status (got $st)" '[ "$st" -ne 0 ]'
check "$sig leaves nothing: $(left "$id" | tr '\n' ' ')" '[ -z "$(left "$id")" ]'
exit "$fail"
