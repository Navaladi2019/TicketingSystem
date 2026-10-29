#!/bin/sh
# stack-case: job=stack order=56 mode=serial
set -eu
. "$(dirname "$0")/../case-env.sh"

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
    compose exec -T nava-core wget -q -O /dev/null \
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

t_redact
