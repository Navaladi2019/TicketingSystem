#!/bin/sh
# stack-case: job=stack order=10 mode=parallel
set -eu
. "$(dirname "$0")/../case-env.sh"

# The server is up on the Postgres config with namespace nava and NavaTenantId
# (the compose health check requires both), and nava-core reports Temporal ok
# while dialing with its certificate.
t_temporal_health() {
    [ "$(health_of temporal-server)" = healthy ] || fail "temporal-server is not healthy"
    await_check temporal ok 120
    pass "temporal-server bootstrapped (nava, NavaTenantId); /healthz reports temporal ok over mTLS"
}

t_temporal_health
