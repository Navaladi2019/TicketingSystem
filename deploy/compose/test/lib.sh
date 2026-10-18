#!/bin/sh
# Shared helpers for the stack tests. Source it; run from anywhere.
# COMPOSE_FILE_ARGS selects the files, e.g. the internal overlay.

ROOT=$(cd "$(dirname "$0")/../../.." && pwd)
cd "$ROOT"

compose() {
    # shellcheck disable=SC2086
    docker compose ${COMPOSE_FILE_ARGS:-} "$@"
}

fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }
pass() { printf 'ok: %s\n' "$*"; }

# container_of <service>: the service's container id.
container_of() { compose ps -a -q "$1" | head -n 1; }

# health_of <service>: healthy, unhealthy, starting, or none.
health_of() {
    cid=$(container_of "$1")
    [ -n "$cid" ] || { echo missing; return; }
    docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$cid"
}

# The one-shot jobs, which must exit 0 instead of staying healthy.
JOBS="db-roles migrate"

# services: every long-running service.
services() { compose config --services | grep -v -x -e db-roles -e migrate; }

# wait_healthy [seconds]: block until every long-running service is healthy
# and every job has exited 0; print each one's state and fail on timeout.
wait_healthy() {
    deadline=$(( $(date +%s) + ${1:-300} ))
    while :; do
        bad=""
        for s in $(services); do
            [ "$(health_of "$s")" = healthy ] || bad="$bad $s"
        done
        for j in $JOBS; do
            state=$(docker inspect -f '{{.State.Status}} {{.State.ExitCode}}' "$(container_of "$j")" 2>/dev/null || echo missing)
            [ "$state" = "exited 0" ] || bad="$bad $j($state)"
        done
        [ -z "$bad" ] && break
        if [ "$(date +%s)" -ge "$deadline" ]; then
            compose ps -a
            fail "not healthy before the deadline:$bad"
        fi
        sleep 3
    done
    for s in $(services); do pass "$s healthy"; done
    for j in $JOBS; do pass "$j exited 0"; done
}

# psql_super <database> <sql>: run as the superuser from inside the postgres
# container (the password never leaves its secret file).
psql_super() {
    compose exec -T postgres psql -X -q -t -A -U pgadmin -d "$1" -c "$2"
}
