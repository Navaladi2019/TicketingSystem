#!/bin/sh
# stack-case: job=stack order=10 mode=parallel
set -eu
. "$(dirname "$0")/../case-env.sh"

# The compose smoke (T05): GET /healthz on nava-core's admin listener answers
# 200 from another container on the compose network, the way a probe would
# reach it (the admin port is never published to the host).
t_smoke() {
    body=$(compose exec -T otel-collector wget -q -T 10 -O - http://nava-core:9090/healthz)         || fail "GET http://nava-core:9090/healthz from the compose network was not 200"
    printf '%s
' "$body"
    printf '%s
' "$body" | grep -q '"postgres":"ok"' || fail "/healthz does not report postgres ok"
    pass "GET /healthz 200 from inside the compose network"
}

t_smoke
