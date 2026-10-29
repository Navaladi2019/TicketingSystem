#!/bin/sh
# stack-case: job=stack-db order=62 mode=serial
set -eu
. "$(dirname "$0")/../case-env.sh"

# A stray membership does not survive db-roles (T02 security review, ADR-001
# I3 / CI-3), and the migrator owns the platform database.
t_dbroles() {
    psql_super postgres "GRANT pg_read_all_data TO keycloak" >/dev/null
    compose run --rm --no-deps db-roles >/dev/null
    [ "$(psql_super postgres "SELECT count(*) FROM pg_auth_members WHERE member = 'keycloak'::regrole")" = 0 ]         || fail "the stray membership survived db-roles"
    [ "$(psql_super postgres "SELECT pg_get_userbyid(datdba) FROM pg_database WHERE datname = 'nava'")" = nava_migrator ]         || fail "nava is not owned by nava_migrator"
    [ "$(psql_super postgres "SELECT count(*) FROM pg_auth_members WHERE roleid = 'nava_migrator'::regrole")" = 0 ]         || fail "nava_migrator is granted to a role"
    pass "stray membership revoked; datdba = nava_migrator, granted to no one"

    # T08 security review: a second superuser fails db-roles, which names it.
    psql_super postgres "CREATE ROLE nava_test_super SUPERUSER" >/dev/null
    if out=$(compose run --rm --no-deps db-roles 2>&1); then
        psql_super postgres "DROP ROLE nava_test_super" >/dev/null
        fail "db-roles succeeded with a second superuser"
    fi
    psql_super postgres "DROP ROLE nava_test_super" >/dev/null
    printf '%s\n' "$out" | grep -q nava_test_super || fail "db-roles failed without naming the second superuser: $out"
    compose run --rm --no-deps db-roles >/dev/null || fail "db-roles fails once the second superuser is gone"
    pass "db-roles refuses a second superuser and names it"
}

t_dbroles
