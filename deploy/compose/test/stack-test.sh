#!/bin/sh
# Stack tests for T02. Usage: stack-test.sh <healthy|pgbouncer|temporal|trace|idempotent|internal>
# Needs the stack up (task stack:up); `internal` brings up its own overlay.
set -eu
. "$(dirname "$0")/lib.sh"

# Every container healthy, db-init done.
t_healthy() { wait_healthy "${WAIT_SECONDS:-300}"; }

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
}

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

# One request to nava-core produces exactly one trace in the Collector's debug
# exporter. The test stack must run with OTEL_TAIL_SAMPLE_PERCENT=100, since
# an ordinary 404 is not an error or slow and is otherwise sampled.
distinct_traces() { compose logs --no-log-prefix otel-collector | sed -n 's/.*Trace ID[ ]*: *\([0-9a-f]*\).*/\1/p' | sort -u | wc -l | tr -d ' '; }
t_trace() {
    before=$(distinct_traces)
    compose exec -T nava-core wget -q -O /dev/null "http://127.0.0.1:8080/trace-probe" 2>/dev/null || true
    deadline=$(( $(date +%s) + 60 ))
    while [ "$(distinct_traces)" -lt $((before + 1)) ]; do
        [ "$(date +%s)" -lt "$deadline" ] || fail "no trace reached the Collector within 60s"
        sleep 2
    done
    sleep 8 # a second trace would show up within the sampler's wait
    after=$(distinct_traces)
    [ "$after" -eq $((before + 1)) ] || fail "expected 1 new trace, saw $((after - before))"
    pass "one request produced one trace in the Collector"
}

# The init is a no-op on a used volume: same role and database ids, data kept,
# nothing created, and the stack stays healthy after a second `up`.
t_idempotent() {
    snapshot() { psql_super postgres "SELECT (SELECT string_agg(oid::text, ',' ORDER BY rolname) FROM pg_roles WHERE rolname IN ('keycloak','temporal','pgbouncer_auth')) || '/' || (SELECT string_agg(oid::text, ',' ORDER BY datname) FROM pg_database WHERE datname IN ('keycloak','temporal','temporal_visibility','nava'))"; }
    psql_super nava "CREATE TABLE IF NOT EXISTS idempotency_marker (id int); INSERT INTO idempotency_marker SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM idempotency_marker)" >/dev/null
    first=$(snapshot)
    compose up -d
    out=$(compose run --rm db-init) || fail "db-init failed on the second run"
    printf '%s\n' "$out"
    printf '%s\n' "$out" | grep -q 'done: 0 created' || fail "second run created something"
    [ "$(snapshot)" = "$first" ] || fail "role or database ids changed"
    [ "$(psql_super nava 'SELECT count(*) FROM idempotency_marker')" = 1 ] || fail "data in nava was lost"
    wait_healthy 120
    pass "second up on a used volume is a no-op"
}

# With the internal overlay the stack still comes up healthy and has no egress.
t_internal() {
    COMPOSE_FILE_ARGS="-f compose.yaml -f deploy/compose/compose.internal.yaml"
    export COMPOSE_FILE_ARGS
    compose up -d
    wait_healthy "${WAIT_SECONDS:-300}"
    for net in $(docker network ls -q --filter label=com.docker.compose.project=nava); do
        [ "$(docker network inspect -f '{{.Internal}}' "$net")" = true ] || fail "a network is not internal"
    done
    if compose exec -T nava-core wget -q -T 5 -O /dev/null http://example.com 2>/dev/null; then
        fail "nava-core reached the internet"
    fi
    pass "overlay up, every network internal, no egress"
}

[ $# -eq 1 ] || fail "usage: stack-test.sh <healthy|pgbouncer|temporal|trace|idempotent|internal>"
"t_$1"
