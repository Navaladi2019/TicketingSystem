#!/bin/sh
# stack-case: job=stack order=50 mode=serial
set -eu
. "$(dirname "$0")/../case-env.sh"

# nava-core's dependency probes against the real containers: the Keycloak
# management port and path, and Temporal's GetSystemInfo. With Keycloak stopped
# /healthz shows it failed and the degraded metric is 1, while /readyz stays
# 200 (only PostgreSQL gates readiness, and compose sets NAVA_DATABASE_URL).
# The admin port is never published, so the requests go through `compose
# exec`. nc, not wget: busybox wget drops the body of an error reply, which is
# what a failure message needs.
t_dependencies() {
    await_check keycloak ok 120
    await_check temporal ok 120
    [ "$(degraded_metric keycloak)" = 0 ] || fail "nava_dependency_degraded for keycloak is not 0 while it is up"
    pass "/healthz reports keycloak and temporal ok against the real containers"

    ready_before=$(admin_code /readyz)
    [ "$ready_before" = 200 ] || fail "/readyz is $ready_before before the stop, want 200"
    compose stop keycloak
    trap 'compose start keycloak' EXIT
    await_check keycloak fail 60
    deadline=$(( $(date +%s) + 30 ))
    until [ "$(degraded_metric keycloak)" = 1 ]; do
        [ "$(date +%s)" -lt "$deadline" ] || { compose start keycloak; fail "nava_dependency_degraded{dependency=\"keycloak\"} did not reach 1"; }
        sleep 1
    done
    ready_during=$(admin_code /readyz)
    [ "$ready_during" = 200 ] || { compose start keycloak; fail "/readyz is $ready_during with Keycloak stopped, want 200"; }
    pass "keycloak stopped: /healthz fail, degraded metric 1, /readyz 200"

    compose start keycloak
    await_check keycloak ok 180
    [ "$(degraded_metric keycloak)" = 0 ] || fail "nava_dependency_degraded for keycloak did not return to 0"
    pass "keycloak back: /healthz ok again"
}

t_dependencies
