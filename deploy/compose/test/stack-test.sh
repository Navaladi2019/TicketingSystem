#!/bin/sh
# Stack tests for T02 and T03. Usage: stack-test.sh <healthy|pgbouncer|temporal|dependencies|trace|redact|idempotent|dbroles|internal|temporal_handshake|temporal_isolation|temporal_custody|temporal_health>
# Needs the stack up (task stack:up); `internal` brings up its own overlay.
set -eu
. "$(dirname "$0")/lib.sh"

# Every container healthy, db-roles and migrate done.
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

# nava-core's dependency probes against the real containers: the Keycloak
# management port and path, and Temporal's GetSystemInfo. With Keycloak stopped
# /healthz shows it failed and the degraded metric is 1, while /readyz answers
# as before (only PostgreSQL gates readiness; until T05 sets NAVA_DATABASE_URL
# in compose it is 503 both times). The admin port is never published, so the
# requests go through `compose exec`. nc, not wget: /healthz is 503 here while
# PostgreSQL is unconfigured, and busybox wget drops the body of an error reply.
admin_get() { # <path>: the full HTTP response from the admin listener
    printf 'GET %s HTTP/1.0\r\nHost: localhost\r\n\r\n' "$1" | compose exec -T nava-core nc 127.0.0.1 9090 | tr -d '\r'
}
admin_code() { admin_get "$1" | sed -n '1s/^HTTP\/[0-9.]* \([0-9]*\).*/\1/p'; }
# await_check <name> <ok|fail> <seconds>: wait for /healthz to report the state.
await_check() {
    deadline=$(( $(date +%s) + $3 ))
    until admin_get /healthz | grep -q "\"$1\":\"$2\""; do
        [ "$(date +%s)" -lt "$deadline" ] || fail "/healthz did not report $1 $2 within $3s"
        sleep 1
    done
}
degraded_metric() { admin_get /metrics | grep "^nava_dependency_degraded{" | grep "dependency=\"$1\"" | sed 's/.* //'; }
t_dependencies() {
    await_check keycloak ok 120
    await_check temporal ok 120
    [ "$(degraded_metric keycloak)" = 0 ] || fail "nava_dependency_degraded for keycloak is not 0 while it is up"
    pass "/healthz reports keycloak and temporal ok against the real containers"

    ready_before=$(admin_code /readyz)
    compose stop keycloak
    trap 'compose start keycloak' EXIT
    await_check keycloak fail 60
    deadline=$(( $(date +%s) + 30 ))
    until [ "$(degraded_metric keycloak)" = 1 ]; do
        [ "$(date +%s)" -lt "$deadline" ] || { compose start keycloak; fail "nava_dependency_degraded{dependency=\"keycloak\"} did not reach 1"; }
        sleep 1
    done
    ready_during=$(admin_code /readyz)
    [ "$ready_during" = "$ready_before" ] || { compose start keycloak; fail "/readyz went from $ready_before to $ready_during with Keycloak stopped"; }
    pass "keycloak stopped: /healthz fail, degraded metric 1, /readyz unchanged ($ready_during)"

    compose start keycloak
    await_check keycloak ok 180
    [ "$(degraded_metric keycloak)" = 0 ] || fail "nava_dependency_degraded for keycloak did not return to 0"
    pass "keycloak back: /healthz ok again"
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

# db-roles and migrate are no-ops on a used volume: same role and database
# ids, data kept, nothing created, changed or revoked, and the stack is
# healthy again after `down` (volumes kept) and a second `up`.
t_idempotent() {
    snapshot() { psql_super postgres "SELECT (SELECT string_agg(oid::text || rolpassword, ',' ORDER BY rolname) FROM pg_authid WHERE rolname IN ('nava_migrator','nava_app','keycloak','temporal','pgbouncer_auth')) || '/' || (SELECT string_agg(oid::text, ',' ORDER BY datname) FROM pg_database WHERE datname IN ('keycloak','temporal','temporal_visibility','nava'))"; }
    psql_super nava "CREATE TABLE IF NOT EXISTS idempotency_marker (id int); INSERT INTO idempotency_marker SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM idempotency_marker)" >/dev/null
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
    [ "$(psql_super nava 'SELECT count(*) FROM idempotency_marker')" = 1 ] || fail "data in nava was lost"
    pass "second up on a used volume is a no-op"
}

# A stray membership does not survive db-roles (T02 security review, ADR-001
# I3 / CI-3), and the migrator owns the platform database.
t_dbroles() {
    psql_super postgres "GRANT pg_read_all_data TO keycloak" >/dev/null
    compose run --rm --no-deps db-roles >/dev/null
    [ "$(psql_super postgres "SELECT count(*) FROM pg_auth_members WHERE member = 'keycloak'::regrole")" = 0 ]         || fail "the stray membership survived db-roles"
    [ "$(psql_super postgres "SELECT pg_get_userbyid(datdba) FROM pg_database WHERE datname = 'nava'")" = nava_migrator ]         || fail "nava is not owned by nava_migrator"
    [ "$(psql_super postgres "SELECT count(*) FROM pg_auth_members WHERE roleid = 'nava_migrator'::regrole")" = 0 ]         || fail "nava_migrator is granted to a role"
    pass "stray membership revoked; datdba = nava_migrator, granted to no one"
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

# --- T28: Temporal mTLS and network isolation -------------------------------
# Needs the stack up with certificates from `make dev-ca`. These run from the
# platform network (`internal`), the way nava-core and the other platform
# processes see Temporal.
CERTS="$ROOT/deploy/compose/certs"
TEMPORAL_IMAGE=nava/temporal-server:dev

# platform_net: the compose network that carries the platform processes.
platform_net() {
    docker network ls -q --filter label=com.docker.compose.project=nava \
        --filter label=com.docker.compose.network=internal | head -n 1
}

# dial_as <client-cert-name|none> <cert-dir>: run `temporal-server health` from a
# throwaway container on the platform network, through the gateway, with the named
# certificate. Prints the command's output; exit status is the probe's.
dial_as() {
    export MSYS_NO_PATHCONV=1 # Git Bash on Windows would rewrite the /c/ paths
    if [ "$1" = none ]; then
        certs="-ca /c/ca.pem -cert= -key="
        mounts="-v $2/ca.pem:/c/ca.pem:ro"
    else
        certs="-ca /c/ca.pem -cert /c/$1.pem -key /c/$1.key"
        mounts="-v $2/ca.pem:/c/ca.pem:ro -v $2/$1.pem:/c/$1.pem:ro -v $2/$1.key:/c/$1.key:ro"
    fi
    # shellcheck disable=SC2086
    docker run --rm --network "$(platform_net)" $mounts --entrypoint /usr/local/bin/temporal-server \
        "$TEMPORAL_IMAGE" health -addr temporal:7233 -server-name temporal $certs 2>&1
}

# Rogue-CA certificate fails the handshake: a nava-relay certificate from another
# CA is refused, as is no certificate, while the real nava-core certificate works.
t_temporal_handshake() {
    [ -f "$CERTS/ca.pem" ] || fail "no certificates in $CERTS: run make dev-ca"
    dial_as nava-core "$CERTS" | grep -q SERVING || fail "the nava-core certificate could not dial Temporal"
    rogue=$(mktemp -d)
    (cd temporal-server && go run ./cmd/devcerts -rogue -out "$rogue") >/dev/null || fail "cannot generate the rogue certificate"
    cp "$CERTS/ca.pem" "$rogue/ca.pem"
    chmod -R a+rX "$rogue"
    if out=$(dial_as rogue-nava-relay "$rogue"); then rm -rf "$rogue"; fail "a rogue-CA certificate completed the handshake: $out"; fi
    case $out in *"health check"*PermissionDenied*) rm -rf "$rogue"; fail "rogue certificate got past the handshake: $out" ;; esac
    rm -rf "$rogue"
    if out=$(dial_as none "$CERTS"); then fail "a client without a certificate was served: $out"; fi
    pass "nava-core dials; a rogue-CA certificate and no certificate fail the handshake"
}

# A container on the platform network reaches 7233 through the gateway and none of
# the internode (7234, 7235, 7239) or membership (6933-6939) ports, and cannot
# even resolve the server.
t_temporal_isolation() {
    nc_ok() { compose exec -T nava-core nc -w 3 "$1" "$2" </dev/null >/dev/null 2>&1; }
    nc_ok temporal 7233 || fail "the gateway does not answer on 7233"
    for port in 7234 7235 7239 6933 6934 6935 6936 6937 6938 6939 9090; do
        nc_ok temporal "$port" && fail "port $port is reachable from the platform network through the gateway"
    done
    if compose exec -T nava-core nslookup temporal-server >/dev/null 2>&1; then
        fail "the platform network can resolve temporal-server"
    fi
    gw=$(docker inspect -f '{{range $n, $c := .NetworkSettings.Networks}}{{$n}} {{end}}' "$(container_of temporal-server)")
    case $gw in *_internal*) fail "temporal-server is on the platform network: $gw" ;; esac
    pass "7233 reachable; 7234, 7235, 7239, 6933-6939 and 9090 are not; temporal-server is not on the platform network"
}

# Each certificate is mounted only into the service that owns it. temporal-system
# is never mounted into a platform service.
t_temporal_custody() {
    bad=""
    for cid in $(compose ps -a -q); do
        svc=$(docker inspect -f '{{index .Config.Labels "com.docker.compose.service"}}' "$cid")
        for src in $(docker inspect -f '{{range .Mounts}}{{.Source}} {{end}}' "$cid"); do
            case $src in
                */certs/temporal-system.*|*/certs/internode*|*/certs/server.*) [ "$svc" = temporal-server ] || bad="$bad $svc:$src" ;;
                */certs/nava-core.*) [ "$svc" = nava-core ] || bad="$bad $svc:$src" ;;
                */certs/nava-operator.*) [ "$svc" = temporal-ui ] || bad="$bad $svc:$src" ;;
                */certs/nava-relay.*|*/certs/nava-worker.*) bad="$bad $svc:$src" ;;
            esac
        done
    done
    [ -z "$bad" ] || fail "certificates mounted into the wrong service:$bad"
    pass "every certificate is mounted only into its own service"
}

# The server is up on the Postgres config with namespace nava and NavaTenantId
# (the compose health check requires both), and nava-core reports Temporal ok
# while dialing with its certificate.
t_temporal_health() {
    [ "$(health_of temporal-server)" = healthy ] || fail "temporal-server is not healthy"
    await_check temporal ok 120
    pass "temporal-server bootstrapped (nava, NavaTenantId); /healthz reports temporal ok over mTLS"
}

[ $# -eq 1 ] || fail "usage: stack-test.sh <healthy|pgbouncer|temporal|dependencies|trace|redact|idempotent|dbroles|internal|temporal_handshake|temporal_isolation|temporal_custody|temporal_health>"
"t_$1"
