#!/bin/sh
# stack-case: job=stack-db order=60 mode=serial
set -eu
. "$(dirname "$0")/../case-env.sh"

# db-roles and migrate are no-ops on a used volume: same role and database
# ids, data kept, nothing created, changed or revoked, and the stack is
# healthy again after `down` (volumes kept) and a second `up`.
t_idempotent() {
    snapshot() { psql_super postgres "SELECT (SELECT string_agg(oid::text || rolpassword, ',' ORDER BY rolname) FROM pg_authid WHERE rolname IN ('nava_migrator','nava_app','nava_worker','nava_relay','nava_purge','nava_onboard','nava_groupsync','nava_backup','keycloak','temporal','pgbouncer_auth')) || '/' || (SELECT string_agg(oid::text, ',' ORDER BY datname) FROM pg_database WHERE datname IN ('keycloak','temporal','temporal_visibility','nava'))"; }
    # The marker is a comment on the nava database, not a table: the
    # catalogue gate (CI-1) rejects any platform table without a tenant key
    # and FORCE RLS, and a comment is not a relation it inspects. It lives in
    # the cluster's catalogue, so it goes with the volume like any row would.
    marker="idempotency-marker-$(od -An -N8 -tx1 /dev/urandom | tr -d ' \n')"
    psql_super nava "COMMENT ON DATABASE nava IS '$marker'" >/dev/null
    first=$(snapshot)
    compose down
    compose up -d
    wait_healthy 300
    roles=$(compose logs --no-log-prefix db-roles | grep '"db-roles done"' | tail -n 1)
    printf '%s
' "$roles"
    printf '%s
' "$roles" | grep -q '"created":0,"changed":0,"revoked":0' || fail "second db-roles run changed something"
    mig=$(compose logs --no-log-prefix migrate | grep '"migrate done"' | tail -n 1)
    printf '%s
' "$mig"
    printf '%s
' "$mig" | grep -q '"migrations_applied":0,"grants_granted":0,"grants_revoked":0' || fail "second migrate run changed something"
    [ "$(snapshot)" = "$first" ] || fail "role ids, passwords or database ids changed"
    [ "$(psql_super nava "SELECT pg_catalog.shobj_description(oid, 'pg_database') FROM pg_catalog.pg_database WHERE datname = 'nava'")" = "$marker" ] || fail "data in nava was lost"
    pass "second up on a used volume is a no-op"
}

t_idempotent
