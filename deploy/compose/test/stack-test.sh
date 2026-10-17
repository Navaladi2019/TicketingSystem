#!/bin/sh
# Stack tests for T02. Usage: stack-test.sh <healthy|pgbouncer|temporal|trace|redact|idempotent|internal>
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
    # The image superuser must not be reachable through the pooler, while a
    # service role is (this also proves the auth_query lookup works).
    if compose exec -T -e PGPASSWORD="${PG_SUPERUSER_PASSWORD:-dev-only-superuser}" pgbouncer \
        psql -X -h 127.0.0.1 -U pgadmin nava -c 'SELECT 1' >/dev/null 2>&1; then
        fail "the superuser logged in through PgBouncer"
    fi
    compose exec -T -e PGPASSWORD="${KEYCLOAK_DB_PASSWORD:-dev-only-keycloak}" pgbouncer \
        psql -X -h 127.0.0.1 -U keycloak nava -c 'SELECT 1' >/dev/null 2>&1 \
        || fail "keycloak could not log in through PgBouncer"
    pass "superuser refused through PgBouncer, service role accepted"
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
    pct=$(compose exec -T otel-collector printenv OTEL_TAIL_SAMPLE_PERCENT | tr -d '\r')
    [ "$pct" = 100 ] || fail "OTEL_TAIL_SAMPLE_PERCENT is $pct; start the stack with OTEL_TAIL_SAMPLE_PERCENT=100 for this test"
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

# The Collector's second layer redacts what filter.go would: post one span that
# carries every kind of denied value (span attribute, event attribute, header
# key, status description) straight to the Collector, and check none comes out
# of the debug exporter. Status Error, so tail sampling keeps the trace at any
# OTEL_TAIL_SAMPLE_PERCENT.
t_redact() {
    rand_hex() { od -An -N"$1" -tx1 /dev/urandom | tr -d ' \n'; }
    tid=$(rand_hex 16)
    now=$(date +%s)
    # Streamed in, not copied: a host temp path is not one docker can read on Windows.
    sed -e "s/__TRACE_ID__/$tid/g" -e "s/__SPAN_ID__/$(rand_hex 8)/" \
        -e "s/__START__/${now}000000000/g" -e "s/__END__/$((now + 1))000000000/" \
        "$ROOT/deploy/compose/test/redact-span.json" |
        compose exec -T nava-core sh -c 'cat > /tmp/redact-span.json'
    MSYS_NO_PATHCONV=1 compose exec -T nava-core wget -q -O /dev/null \
        --header 'Content-Type: application/json' --post-file /tmp/redact-span.json \
        http://otel-collector:4318/v1/traces || fail "the Collector rejected the span"

    deadline=$(( $(date +%s) + 60 ))
    until compose logs --no-log-prefix otel-collector | grep -q "$tid"; do
        [ "$(date +%s)" -lt "$deadline" ] || fail "trace $tid did not reach the Collector's debug output"
        sleep 2
    done
    logs=$(compose logs --no-log-prefix otel-collector)
    # This trace's span, from its Trace ID line to the next one.
    span=$(printf '%s\n' "$logs" | awk -v id="$tid" '/Trace ID/ {p = index($0, id) > 0} p')

    leaks=$(printf '%s\n' "$logs" | grep -o 'LEAKY-[A-Z]*' | sort -u | tr '\n' ' ')
    [ -z "$leaks" ] || fail "denied values reached the debug exporter: $leaks"
    printf '%s\n' "$span" | grep -q "keep.me: Str($tid)" || fail "marker attribute keep.me missing from the span"
    printf '%s\n' "$span" | grep -q 'exception.type: Str(keep-type)' || fail "marker event attribute exception.type missing"
    printf '%s\n' "$span" | grep -Eq 'Status code +: Error' || fail "status code Error missing"
    pass "denied span, event and status values redacted; markers and Error status kept"
}

# The init is a no-op on a used volume: same role and database ids, data kept,
# nothing created, and the stack is healthy again after `down` (volumes kept)
# and a second `up`.
t_idempotent() {
    snapshot() { psql_super postgres "SELECT (SELECT string_agg(oid::text, ',' ORDER BY rolname) FROM pg_roles WHERE rolname IN ('keycloak','temporal','pgbouncer_auth')) || '/' || (SELECT string_agg(oid::text, ',' ORDER BY datname) FROM pg_database WHERE datname IN ('keycloak','temporal','temporal_visibility','nava'))"; }
    psql_super nava "CREATE TABLE IF NOT EXISTS idempotency_marker (id int); INSERT INTO idempotency_marker SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM idempotency_marker)" >/dev/null
    first=$(snapshot)
    compose down
    compose up -d
    wait_healthy 300
    out=$(compose logs --no-log-prefix db-init)
    printf '%s\n' "$out"
    printf '%s\n' "$out" | grep -q 'done: 0 created' || fail "second run created something"
    [ "$(snapshot)" = "$first" ] || fail "role or database ids changed"
    [ "$(psql_super nava 'SELECT count(*) FROM idempotency_marker')" = 1 ] || fail "data in nava was lost"
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

[ $# -eq 1 ] || fail "usage: stack-test.sh <healthy|pgbouncer|temporal|trace|redact|idempotent|internal>"
"t_$1"
