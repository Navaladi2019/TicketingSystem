#!/bin/sh
# Shared helpers for the stack tests. Source it; run from anywhere.
# COMPOSE_FILE_ARGS selects the files, e.g. the internal overlay.

# The checkout root: the nearest parent holding compose.yaml, wherever the script sits.
ROOT=$(cd "$(dirname "$0")" && while [ ! -f compose.yaml ]; do cd ..; done; pwd)
cd "$ROOT"

# The compose project under test: `name: nava` in compose.yaml unless
# COMPOSE_PROJECT_NAME names another (make ci-isolated).
PROJECT=${COMPOSE_PROJECT_NAME:-nava}

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
JOBS="db-roles migrate temporal-schema"

# services: every long-running service.
services() { compose config --services | grep -v -x -e db-roles -e migrate -e temporal-schema; }

# epoch_of <RFC 3339 UTC time>: seconds since 1970. awk, not GNU date -d, so
# it runs on a macOS laptop too.
epoch_of() {
    printf '%s\n' "$1" | awk -F'[-T:.Z]' '{
        y = $1; m = $2 + 0
        if (m <= 2) { y--; m += 12 }
        days = 365 * y + int(y / 4) - int(y / 100) + int(y / 400) + int((153 * (m - 3) + 2) / 5) + $3 - 719469
        print days * 86400 + $4 * 3600 + $5 * 60 + int($6)
    }'
}

# stack_created: when the project's first container was created, in seconds
# since 1970.
stack_created() {
    first=$(docker inspect -f '{{.Created}}' $(compose ps -a -q) | sort | head -n 1)
    [ -n "$first" ] || fail "no container in project $PROJECT"
    epoch_of "$first"
}

# other_stacks: the other compose projects with a running container on this
# Docker host, one per line.
other_stacks() {
    docker ps --filter label=com.docker.compose.project --format '{{.Label "com.docker.compose.project"}}' \
        | sort -u | grep -v -x -e "$PROJECT" || true
}

# wait_healthy [seconds] [since] [contended]: block until every long-running
# service is healthy and every job has exited 0; print each one's state and
# fail on timeout. The deadline is seconds after since (seconds since 1970), or
# after now. With contended, the deadline is contended seconds after since
# instead whenever another compose stack ran on the host during the wait: the
# stacks share CPU, and Keycloak's first boot slows most (TICK-200). The pass
# line then says the budget was exceeded and why.
wait_healthy() {
    since=${2:-$(date +%s)}
    deadline=$(( since + ${1:-300} ))
    others=""
    while :; do
        if [ -n "${3:-}" ]; then
            now_others=$(other_stacks)
            [ -z "$now_others" ] || others=$(printf '%s\n%s\n' "$others" "$now_others" | grep -v '^$' | sort -u)
            [ -z "$others" ] || deadline=$(( since + $3 ))
        fi
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
    took=$(( $(date +%s) - since ))
    if [ -n "$others" ] && [ "$took" -gt "${1:-300}" ]; then
        pass "healthy $took s after the start: over the ${1:-300} s budget, allowed up to $3 s because other stacks shared the host:" $others
    else
        pass "healthy $took s after the start, budget ${1:-300} s"
    fi
    for s in $(services); do pass "$s healthy"; done
    for j in $JOBS; do pass "$j exited 0"; done
}

# psql_super <database> <sql>: run as the superuser over the Unix socket
# inside the postgres container, the only path db/pg_hba.conf opens to it (the
# password never leaves its secret file).
psql_super() {
    # shellcheck disable=SC2016 # expanded inside the container
    compose exec -T postgres sh -c 'PGPASSWORD=$(cat /run/pg_superuser_password) exec psql -X -q -t -A -U pgadmin -d "$0" -c "$1"' "$1" "$2"
}

# --- Helpers shared by several cases ---------------------------------------
admin_get() { # <path>: the full HTTP response from the admin listener
    printf 'GET %s HTTP/1.0\r\nHost: localhost\r\n\r\n' "$1" | compose exec -T nava-core nc 127.0.0.1 9090 | tr -d '\r'
}
admin_code() { admin_get "$1" | sed -n '1s/^HTTP\/[0-9.]* \([0-9]*\).*/\1/p'; }
# await_check <name> <ok|fail> <seconds>: wait for /healthz to report the state.
await_check() {
    deadline=$(( $(date +%s) + $3 ))
    until admin_get /healthz | grep -q "\"$1\":\"$2\""; do
        [ "$(date +%s)" -lt "$deadline" ] || fail "/healthz did not report $1 $2 within $3s"
        sleep 1
    done
}
degraded_metric() { admin_get /metrics | grep "^nava_dependency_degraded{" | grep "dependency=\"$1\"" | sed 's/.* //'; }

# --- T28: Temporal mTLS and network isolation -------------------------------
# Needs the stack up with certificates from `make dev-ca`. These run from the
# platform network (`internal`), the way nava-core and the other platform
# processes see Temporal.
CERTS="$ROOT/deploy/compose/certs"
TEMPORAL_IMAGE=nava/temporal-server:${NAVA_IMAGE_TAG:-dev}

# platform_net: the compose network that carries the platform processes.
platform_net() {
    docker network ls -q --filter label=com.docker.compose.project=$PROJECT \
        --filter label=com.docker.compose.network=internal | head -n 1
}

# dial_as <client-cert-name|none> <cert-dir>: run `temporal-server health` from a
# throwaway container on the platform network, through the gateway, with the named
# certificate. Prints the command's output; exit status is the probe's.
dial_as() {
    if [ "$1" = none ]; then
        certs="-ca /c/ca.pem -cert= -key="
        mounts="-v $2/ca.pem:/c/ca.pem:ro"
    else
        certs="-ca /c/ca.pem -cert /c/$1.pem -key /c/$1.key"
        mounts="-v $2/ca.pem:/c/ca.pem:ro -v $2/$1.pem:/c/$1.pem:ro -v $2/$1.key:/c/$1.key:ro"
    fi
    # shellcheck disable=SC2086
    docker run --rm --network "$(platform_net)" $mounts --entrypoint /usr/local/bin/temporal-server \
        "$TEMPORAL_IMAGE" health -addr temporal:7233 -server-name temporal $certs 2>&1
}

t_temporal_confinement() {
    nets_of() {
        cid=$(container_of "$1")
        [ -n "$cid" ] || fail "no container for $1"
        docker inspect -f '{{range $n, $c := .NetworkSettings.Networks}}{{$n}} {{end}}' "$cid"
    }
    srv=$(nets_of temporal-server)
    [ "$srv" = "${PROJECT}_temporal-backplane " ] || fail "temporal-server is on networks other than temporal-backplane: $srv"
    for svc in nava-core nava-worker nava-relay nava-purge; do
        # An assignment, so set -e stops on a failed lookup; a bare $(...) in
        # the for list would let a missing container pass with no networks.
        nets=$(nets_of "$svc")
        for n in $nets; do
            case " $srv " in *" $n "*) fail "temporal-server shares $n with $svc" ;; esac
        done
    done
    pass "temporal-server is only on temporal-backplane and shares no network with a platform process"
}
