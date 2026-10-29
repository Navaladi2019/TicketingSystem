#!/bin/sh
# stack-case: job=stack order=54 mode=serial
set -eu
. "$(dirname "$0")/../case-env.sh"

# A log record from nava-core reaches the Collector's debug exporter with the
# trace id of the request that produced it. One request, then the request id
# from the response, the trace id from nava-core's own stdout line for that
# request (it carries both), then the Collector's LogRecord for that trace id.
# The Collector's logs pipeline is not tail sampled, so any
# OTEL_TAIL_SAMPLE_PERCENT works.
log_records_for() { # trace id -> the debug exporter's LogRecord blocks for it
    compose logs --no-log-prefix otel-collector |
        awk -v id="$1" '/LogRecord #/ {if (p) printf "%s", blk; blk = ""; p = 0} {blk = blk $0 "\n"} index($0, "Trace ID") && index($0, id) {p = 1} END {if (p) printf "%s", blk}'
}
t_logs() {
    rid=$(compose exec -T nava-core wget -S -O /dev/null "http://127.0.0.1:8080/api/v1/openapi.yaml" 2>&1 |
        tr -d '\r' | sed -n 's/^ *[Xx]-[Rr]equest-[Ii]d: *//p' | head -1)
    [ -n "$rid" ] || fail "no X-Request-Id in the response"
    deadline=$(( $(date +%s) + 60 ))
    tid=
    until [ -n "$tid" ]; do
        tid=$(compose logs --no-log-prefix nava-core | grep "\"request_id\":\"$rid\"" |
            sed -n 's/.*"trace_id":"\([0-9a-f]\{32\}\)".*/\1/p' | head -1)
        [ -n "$tid" ] || { [ "$(date +%s)" -lt "$deadline" ] || fail "no stdout log line for request $rid"; sleep 1; }
    done
    until [ -n "$(log_records_for "$tid")" ]; do
        [ "$(date +%s)" -lt "$deadline" ] || fail "no log record with trace id $tid reached the Collector within 60s"
        sleep 2
    done
    rec=$(log_records_for "$tid")
    printf '%s\n' "$rec" | grep -q 'Body: Str(http request)' || fail "the record for trace $tid is not the request's access log line"
    printf '%s\n' "$rec" | grep -q "request_id: Str($rid)" || fail "the record lacks request_id $rid"
    if printf '%s\n' "$rec" | grep -Eq 'url\.path|db\.statement|exception\.message|http\.request\.header|client\.address|user_agent\.original'; then
        fail "a denied attribute is on the request's log record"
    fi
    pass "request $rid: its log record reached the Collector with trace id $tid, no denied attributes"

    # The deny-list reaches log records: post one carrying denied keys.
    rand_hex() { od -An -N"$1" -tx1 /dev/urandom | tr -d ' \n'; }
    ptid=$(rand_hex 16)
    sed -e "s/__TRACE_ID__/$ptid/g" -e "s/__SPAN_ID__/$(rand_hex 8)/" -e "s/__NOW__/$(date +%s)000000000/" \
        "$ROOT/deploy/compose/test/redact-log.json" |
        compose exec -T nava-core sh -c 'cat > /tmp/redact-log.json'
    compose exec -T nava-core wget -q -O /dev/null \
        --header 'Content-Type: application/json' --post-file /tmp/redact-log.json \
        http://otel-collector:4318/v1/logs || fail "the Collector rejected the log record"
    until [ -n "$(log_records_for "$ptid")" ]; do
        [ "$(date +%s)" -lt "$deadline" ] || fail "log record $ptid did not reach the Collector's debug output"
        sleep 2
    done
    leaks=$(compose logs --no-log-prefix otel-collector | grep -o 'LEAKY-[A-Z]*' | sort -u | tr '\n' ' ')
    [ -z "$leaks" ] || fail "denied values reached the debug exporter's log records: $leaks"
    log_records_for "$ptid" | grep -q "keep.me: Str($ptid)" || fail "marker attribute keep.me missing from the log record"
    pass "denied log attributes redacted; marker kept"
}

t_logs
