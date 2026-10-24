#!/bin/sh
# make ci-isolated ID=<pr-or-sha> (TICK-147): the four CI stages on a compose
# stack of its own, so several checkouts run CI at once on one Docker host.
#   project      nava-<id>        (COMPOSE_PROJECT_NAME: containers, networks, volumes)
#   image tag    <id>             (NAVA_IMAGE_TAG on every nava/* image)
#   host ports   derived from id  (NAVA_CORE_PORT, KEYCLOAK_PORT, TEMPORAL_TEST_PORT, UI_PORT)
# Caches stay shared: the Go build, module and lint caches, the pinned tools,
# the BuildKit layer cache and pulled registry images. On exit it removes only
# its own project and image tags, then fails if anything of nava-<id> is left.
# One run per checkout at a time (certificates and ci6-report.md live in it).
# The stages run at the same time; STAGES="ci-unit ci-stack" runs a subset,
# and PARALLEL=0 runs them one after another, in the given order.
# --down <id> (make ci-isolated-down) only tears down, for a run that was
# killed before its EXIT trap ran.
set -eu
cd "$(dirname "$0")/../.."

down=
[ "${1:-}" = --down ] && { down=1; shift; }
[ $# -eq 1 ] || { echo "usage: $0 [--down] <pr-or-sha>" >&2; exit 2; }
# Docker tags and compose project names: lowercase letters, digits, - and _.
id=$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | tr -c 'a-z0-9_-' '-' | sed 's/^[^a-z0-9]*//' | cut -c 1-40)
case "$id" in
    "" | dev) echo "ci-isolated: ID '$1' is empty or the shared tag dev" >&2; exit 2 ;;
esac

COMPOSE_PROJECT_NAME="nava-$id"
NAVA_IMAGE_TAG="$id"
IMAGE_CACHE=local
export COMPOSE_PROJECT_NAME NAVA_IMAGE_TAG IMAGE_CACHE

# Four host ports from a slot in 20000-21999 chosen by the id's checksum; the
# next slot when a running container already publishes one of them.
taken() { [ -n "$(docker ps -q --filter "publish=$1")" ]; }
slot=$(( $(printf '%s' "$id" | cksum | cut -d ' ' -f 1) % 500 ))
tries=0
while :; do
    base=$(( 20000 + slot * 4 ))
    taken $base || taken $((base + 1)) || taken $((base + 2)) || taken $((base + 3)) || break
    tries=$((tries + 1))
    [ $tries -lt 500 ] || { echo "ci-isolated: no free port slot in 20000-21999" >&2; exit 1; }
    slot=$(( (slot + 1) % 500 ))
done
NAVA_CORE_PORT=${NAVA_CORE_PORT:-$base}
KEYCLOAK_PORT=${KEYCLOAK_PORT:-$((base + 1))}
TEMPORAL_TEST_PORT=${TEMPORAL_TEST_PORT:-$((base + 2))}
UI_PORT=${UI_PORT:-$((base + 3))}
TEMPORAL_UI_PORT=$UI_PORT
export NAVA_CORE_PORT KEYCLOAK_PORT TEMPORAL_TEST_PORT UI_PORT TEMPORAL_UI_PORT

images="nava/nava-core nava/nava-db nava/otel-collector nava/temporal-server"
stack_files="-f compose.yaml -f deploy/compose/compose.internal.yaml"

teardown() {
    # shellcheck disable=SC2086
    COMPOSE_PROFILES=operator docker compose $stack_files down -v --remove-orphans >/dev/null 2>&1 || true
    # Untags only: layers another tag uses and the build cache stay.
    for i in $images; do docker image rm "$i:$id" >/dev/null 2>&1 || true; done
    left=""
    for kind in container volume network; do
        case $kind in
            container) out=$(docker ps -a -q --filter "label=com.docker.compose.project=$COMPOSE_PROJECT_NAME") ;;
            *) out=$(docker $kind ls -q --filter "label=com.docker.compose.project=$COMPOSE_PROJECT_NAME") ;;
        esac
        [ -z "$out" ] || left="$left $kind:$(echo $out)"
    done
    for i in $images; do
        [ -z "$(docker images -q "$i:$id")" ] || left="$left image:$i:$id"
    done
    if [ -n "$left" ]; then
        echo "ci-isolated: teardown left$left" >&2
        return 1
    fi
    echo "ci-isolated: teardown clean, nothing of $COMPOSE_PROJECT_NAME left"
}
finish() {
    status=$?
    trap - EXIT
    teardown || [ "$status" -ne 0 ] || status=1
    echo "ci-isolated: $COMPOSE_PROJECT_NAME $( [ "$status" -eq 0 ] && echo PASS || echo "FAIL ($status)" ) in $(( $(date +%s) - start ))s"
    exit "$status"
}
[ -z "$down" ] || { teardown; exit; }

start=$(date +%s)
trap finish EXIT
trap 'exit 130' INT TERM

echo "ci-isolated: project $COMPOSE_PROJECT_NAME, image tag $NAVA_IMAGE_TAG, ports nava-core $NAVA_CORE_PORT keycloak $KEYCLOAK_PORT temporal $TEMPORAL_TEST_PORT ui $UI_PORT"

# The pinned tools install once per version, not on every run.
make --no-print-directory check-tools >/dev/null 2>&1 || make --no-print-directory tools

stages=${STAGES:-ci-lint ci-unit ci-integration ci-stack}
if [ "${PARALLEL:-1}" = 0 ]; then
    for stage in $stages; do
        t=$(date +%s)
        make --no-print-directory "$stage"
        echo "ci-isolated: $stage passed in $(( $(date +%s) - t ))s"
    done
    exit 0
fi

# The stages run at the same time (TICK-153 O4), so the run takes as long as
# its slowest stage. Each writes its own log, printed whole when it ends; the
# logs stay in $logs after the run.
logs="${TMPDIR:-/tmp}/ci-isolated-$id"
rm -rf "$logs"
mkdir -p "$logs"
echo "ci-isolated: stages $stages at once; logs in $logs"
procs=""
for stage in $stages; do
    ( t=$(date +%s)
      if make --no-print-directory "$stage" >"$logs/$stage.log" 2>&1; then r=0; else r=$?; fi
      echo "$r $(( $(date +%s) - t ))" >"$logs/$stage.status"
      exit "$r" ) &
    procs="$procs $!:$stage"
done
# A run cut short stops its stages before the teardown.
stop_stages() { for p in $procs; do kill "${p%%:*}" 2>/dev/null || true; done; }
trap 'stop_stages; exit 130' INT TERM

failed=""
for p in $procs; do
    stage=${p#*:}
    wait "${p%%:*}" || failed="$failed $stage"
    read -r r took <"$logs/$stage.status" || { r=1; took=?; }
    echo "=== ci-isolated: $stage log"
    cat "$logs/$stage.log"
    if [ "$r" -eq 0 ]; then
        echo "ci-isolated: $stage passed in ${took}s"
    else
        echo "ci-isolated: $stage FAILED ($r) in ${took}s"
    fi
done
[ -z "$failed" ] || { echo "ci-isolated: failed stages:$failed" >&2; exit 1; }
