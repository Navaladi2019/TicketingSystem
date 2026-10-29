#!/bin/sh
# stack-case: job=stack order=10 mode=parallel
set -eu
. "$(dirname "$0")/../case-env.sh"

# Keycloak and Temporal have their own databases; Temporal's visibility is SQL.
t_temporal() {
    [ "$(psql_super postgres "SELECT string_agg(datname, ',' ORDER BY datname) FROM pg_database WHERE datname IN ('keycloak','temporal','temporal_visibility')")" = keycloak,temporal,temporal_visibility ] \
        || fail "a service database is missing"
    [ "$(psql_super temporal_visibility "SELECT count(*) FROM information_schema.tables WHERE table_name = 'executions_visibility'")" = 1 ] \
        || fail "no SQL visibility schema in temporal_visibility"
    [ "$(psql_super keycloak "SELECT count(*) FROM information_schema.tables WHERE table_name = 'realm'")" = 1 ] \
        || fail "Keycloak has not migrated its schema"
    pass "own databases present, Temporal SQL visibility and Keycloak schema in place"
}

t_temporal
