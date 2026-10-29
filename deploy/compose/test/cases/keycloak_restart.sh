#!/bin/sh
# stack-case: job=stack order=51 mode=serial
# Restarts PostgreSQL, so it runs after dependencies and before the cases that count on a settled stack.
set -eu
. "$(dirname "$0")/../case-env.sh"

# TICK-328 (R4): restart PostgreSQL. Keycloak must issue tokens again within
# RECOVER_SECONDS (90) with no restart of its own, and /healthz must not report
# keycloak ok while issuance fails: a sample counts when issuance failed just
# before it and just after it, and had been failing for more than
# STALE_SECONDS (5). /healthz serves the last probe round, which lags by up to
# the round interval plus the probe timeout (1 s + 2 s, health.Options), so a
# sample in the first seconds after the restart can still be the round before
# it. Issuance is a client_credentials grant for
# nava-platform (dev secret from the realm import), sent from nava-core's
# container.
issue_code() { # HTTP status of a token request; empty when none came back
    compose exec -T nava-core wget -q -S -O /dev/null -T 5         --post-data 'grant_type=client_credentials&client_id=nava-platform&client_secret=dev-only-platform-secret'         http://keycloak:8080/realms/nava/protocol/openid-connect/token 2>&1 |
        sed -n 's/^ *HTTP\/[0-9.]* \([0-9]*\).*/\1/p' | head -n 1 || true
}
t_keycloak_restart() {
    await_check keycloak ok 120
    [ "$(issue_code)" = 200 ] || fail "no token before the restart"
    compose restart postgres
    start=$(date +%s)
    deadline=$(( start + ${RECOVER_SECONDS:-90} ))
    failing_since=
    while :; do
        before=$(issue_code)
        [ "$before" != 200 ] || break
        [ -n "$failing_since" ] || failing_since=$(date +%s)
        healthz_ok=0
        admin_get /healthz | grep -q '"keycloak":"ok"' && healthz_ok=1
        after=$(issue_code)
        [ "$after" != 200 ] || break
        [ "$healthz_ok" = 0 ] || [ $(( $(date +%s) - failing_since )) -le "${STALE_SECONDS:-5}" ] || fail "/healthz reported keycloak ok while token issuance failed (HTTP ${before:-none}, then ${after:-none})"
        [ "$(date +%s)" -lt "$deadline" ] || fail "no token ${RECOVER_SECONDS:-90}s after the PostgreSQL restart (last HTTP ${after:-none})"
        sleep 1
    done
    pass "token issuance recovered $(( $(date +%s) - start ))s after the PostgreSQL restart"
    await_check keycloak ok 60
    pass "/healthz keycloak ok again; it was never ok while issuance failed"
}

t_keycloak_restart
