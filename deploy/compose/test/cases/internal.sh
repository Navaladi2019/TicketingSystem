#!/bin/sh
# stack-case: job=stack-internal order=99 mode=serial
set -eu
. "$(dirname "$0")/../case-env.sh"

# With the internal overlay the stack still comes up healthy and has no egress.
t_internal() {
    COMPOSE_FILE_ARGS="-f compose.yaml -f deploy/compose/compose.internal.yaml"
    export COMPOSE_FILE_ARGS
    compose up -d
    # 300 s: an overlay of its own, not the S1 stack.
    wait_healthy "${WAIT_SECONDS:-300}"
    for net in $(docker network ls -q --filter label=com.docker.compose.project=$PROJECT); do
        [ "$(docker network inspect -f '{{.Internal}}' "$net")" = true ] || fail "a network is not internal"
    done
    if compose exec -T nava-core wget -q -T 5 -O /dev/null http://example.com 2>/dev/null; then
        fail "nava-core reached the internet"
    fi
    pass "overlay up, every network internal, no egress"
}

t_internal
