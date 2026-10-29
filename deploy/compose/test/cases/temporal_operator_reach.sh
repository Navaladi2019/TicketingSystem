#!/bin/sh
# stack-case: job=stack order=10 mode=parallel
set -eu
. "$(dirname "$0")/../case-env.sh"

# Operator-network restriction (T27a handoff item 5), at runtime: the Temporal UI
# wields the nava-operator identity (list every tenant's workflows, read their
# history, terminate), so no platform process may reach it, by name or through
# the host. On Docker Desktop a container on any non-internal network reaches a
# port published on the host's 127.0.0.1 via host.docker.internal, which
# `docker compose config` cannot show. Needs `--profile operator`.
t_temporal_operator_reach() {
    [ -n "$(compose ps -q temporal-ui)" ] || fail "temporal-ui is not running: bring the stack up with --profile operator"
    ui_port=${TEMPORAL_UI_PORT:-18233}
    for target in "temporal-ui:8080" "host.docker.internal:$ui_port" "gateway.docker.internal:$ui_port" "172.17.0.1:$ui_port"; do
        if compose exec -T nava-core wget -q -O - -T 3 "http://$target/api/v1/cluster-info" >/dev/null 2>&1; then
            fail "nava-core reaches the Temporal UI (nava-operator identity) at $target"
        fi
    done
    for net in internal default; do
        id=$(docker network ls -q --filter label=com.docker.compose.project=$PROJECT \
            --filter label=com.docker.compose.network=$net | head -n 1)
        for target in "host.docker.internal:$ui_port" "gateway.docker.internal:$ui_port"; do
            if docker run --rm --network "$id" --entrypoint /usr/local/bin/temporal-server "$TEMPORAL_IMAGE" \
                health -tcp-only -addr "$target" -timeout 3s >/dev/null 2>&1; then
                fail "a container on the $net network reaches the Temporal UI at $target"
            fi
        done
    done
    pass "no platform process reaches the Temporal UI, by name or through the host"
}

t_temporal_operator_reach
