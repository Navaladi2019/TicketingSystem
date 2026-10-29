#!/bin/sh
# stack-case: job=stack order=10 mode=parallel
set -eu
. "$(dirname "$0")/../case-env.sh"

# PgBouncer is in transaction mode, version >= 1.21, max_prepared_statements > 0.
t_pgbouncer() {
    pw=${PGBOUNCER_AUTH_PASSWORD:-dev-only-pgbouncer}
    show() {
        compose exec -T -e PGPASSWORD="$pw" pgbouncer \
            psql -X -t -A -F '|' -h 127.0.0.1 -U pgbouncer_auth pgbouncer -c "$1"
    }
    [ "$(show 'SHOW CONFIG' | awk -F'|' '$1=="pool_mode"{print $2}')" = transaction ] \
        || fail "pool_mode is not transaction"
    mps=$(show 'SHOW CONFIG' | awk -F'|' '$1=="max_prepared_statements"{print $2}')
    [ "${mps:-0}" -gt 0 ] || fail "max_prepared_statements is $mps"
    ver=$(show 'SHOW VERSION' | sed -n 's/.*PgBouncer \([0-9]*\)\.\([0-9]*\).*/\1.\2/p')
    major=${ver%.*}; minor=${ver#*.}
    { [ "$major" -gt 1 ] || [ "$minor" -ge 21 ]; } || fail "PgBouncer $ver is older than 1.21"
    pass "pgbouncer $ver, transaction mode, max_prepared_statements=$mps"
    # Only roles with pgbouncer_login in db/grants.yaml log in through the
    # pooler: never the superuser or the migrator (T02 security review).
    via_pooler() { # <role> <password>
        compose exec -T -e PGPASSWORD="$2" pgbouncer psql -X -h 127.0.0.1 -U "$1" nava -c 'SELECT 1' >/dev/null 2>&1
    }
    via_pooler pgadmin "${PG_SUPERUSER_PASSWORD:-dev-only-superuser}" && fail "the superuser logged in through PgBouncer"
    via_pooler nava_migrator "${NAVA_MIGRATOR_DB_PASSWORD:-dev-only-migrator}" && fail "the migrator logged in through PgBouncer"
    via_pooler keycloak "${KEYCLOAK_DB_PASSWORD:-dev-only-keycloak}" && fail "keycloak logged in through PgBouncer"
    via_pooler nava_app "${NAVA_APP_DB_PASSWORD:-dev-only-app}" || fail "nava_app could not log in through PgBouncer"
    pass "superuser, migrator and keycloak refused through PgBouncer, nava_app accepted"
}

t_pgbouncer
