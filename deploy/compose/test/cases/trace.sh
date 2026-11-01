#!/bin/sh
# stack-case: job=stack order=52 mode=serial
set -eu
. "$(dirname "$0")/../case-env.sh"

# One request to nava-core produces exactly one trace in the Collector's debug
# exporter. The probe's trace is found by its request id: nava-core sets
# X-Request-Id on every response and stamps the same id on the server span as
# nava.request_id, so traces from earlier cases that land late are not counted.
# The test stack must run with OTEL_TAIL_SAMPLE_PERCENT=100, since an ordinary
# 404 is not an error or slow and is otherwise sampled.
traces_for() { # request id -> distinct trace ids of the Collector's spans carrying it
    compose logs --no-log-prefix otel-collector |
        awk -v want="nava.request_id: Str($1)" '
            function flush() { if (span && hit && tid != "") print tid; span = 0; hit = 0; tid = "" }
            /(Resource|Scope)(Spans|Logs) #|LogRecord #|Span #/ { flush(); span = ($0 ~ /^ *Span #/) }
            span && /Trace ID/ { t = $0; sub(/.*Trace ID[ ]*: */, "", t); sub(/[^0-9a-f].*/, "", t); tid = t }
            span && index($0, want) { hit = 1 }
            END { flush() }' |
        sort -u
}
t_trace() {
    pct=$(compose exec -T otel-collector printenv OTEL_TAIL_SAMPLE_PERCENT | tr -d '\r')
    [ "$pct" = 100 ] || fail "OTEL_TAIL_SAMPLE_PERCENT is $pct; start the stack with OTEL_TAIL_SAMPLE_PERCENT=100 for this test"
    # The probe 404s, so wget exits non-zero; -S still prints the headers.
    rid=$(compose exec -T nava-core wget -S -O /dev/null "http://127.0.0.1:8080/trace-probe" 2>&1 |
        tr -d '\r' | sed -n 's/^ *[Xx]-[Rr]equest-[Ii]d: *//p' | head -1) || true
    [ -n "$rid" ] || fail "no X-Request-Id in the probe's response"
    deadline=$(( $(date +%s) + 60 ))
    until [ -n "$(traces_for "$rid")" ]; do
        [ "$(date +%s)" -lt "$deadline" ] || fail "no trace for request $rid reached the Collector within 60s"
        sleep 2
    done
    sleep 8 # a second trace for the same request would show up within the sampler's wait
    tids=$(traces_for "$rid")
    n=$(printf '%s\n' "$tids" | grep -c .)
    [ "$n" -eq 1 ] || fail "expected 1 trace for request $rid, saw $n: $(printf '%s' "$tids" | tr '\n' ' ')"
    pass "request $rid produced one trace in the Collector ($tids)"
}

t_trace
