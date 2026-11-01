#!/bin/sh
# stack-case: job=stack order=52 mode=serial
set -eu
. "$(dirname "$0")/../case-env.sh"

# One request to nava-core produces exactly one trace in the Collector's debug
# exporter. The test stack must run with OTEL_TAIL_SAMPLE_PERCENT=100, since
# an ordinary 200 is not an error or slow and is otherwise sampled.
#
# Only traces carrying this request's id count (nava.request_id on the server
# span, from the response's X-Request-Id). Counting every trace id in the
# Collector's output flaked (TICK-371): any other traced request during the
# wait, such as a background job or a probe, added one. The path cannot
# identify the span: url.path is on the deny-list and an unmatched route gets
# no http.route.
traces_for() { # request id -> distinct trace ids of spans carrying it, one per line
    compose logs --no-log-prefix otel-collector |
        awk -v rid="nava.request_id: Str($1)" '
            /(^|[ \t])Span #[0-9]/ {span = 1; tid = ""; next}
            /LogRecord #[0-9]|ResourceSpans #[0-9]|ResourceLog #[0-9]/ {span = 0; tid = ""; next}
            span && tid == "" && /Trace ID[ ]*:/ {tid = $NF; next}
            span && tid != "" && index($0, rid) {print tid}' |
        sort -u
}
count_traces() { traces_for "$1" | grep -c . || true; }
request() { # one traced GET to nava-core -> its X-Request-Id
    compose exec -T nava-core wget -S -O /dev/null "http://127.0.0.1:8080/api/v1/openapi.yaml" 2>&1 |
        tr -d '\r' | sed -n 's/^ *[Xx]-[Rr]equest-[Ii]d: *//p' | head -1
}
wait_trace() { # request id, label -> waits up to 60s for its first trace
    deadline=$(( $(date +%s) + 60 ))
    while [ "$(count_traces "$1")" -lt 1 ]; do
        [ "$(date +%s)" -lt "$deadline" ] || fail "no trace for $2 request $1 reached the Collector within 60s"
        sleep 2
    done
}
t_trace() {
    pct=$(compose exec -T otel-collector printenv OTEL_TAIL_SAMPLE_PERCENT | tr -d '\r')
    [ "$pct" = 100 ] || fail "OTEL_TAIL_SAMPLE_PERCENT is $pct; start the stack with OTEL_TAIL_SAMPLE_PERCENT=100 for this test"
    rid=$(request)
    [ -n "$rid" ] || fail "no X-Request-Id in the response"
    wait_trace "$rid" probe
    # Regression check (TICK-377): an unrelated request inside the wait window
    # must not change the probe's count. Its own trace must reach the Collector,
    # so the window really held a second trace, one that counting every new
    # trace would have seen.
    other=$(request)
    [ -n "$other" ] || fail "no X-Request-Id in the unrelated request's response"
    [ "$other" != "$rid" ] || fail "the unrelated request reused request id $rid"
    sleep 8 # a second trace for the probe would show up within the sampler's wait
    wait_trace "$other" unrelated
    n=$(count_traces "$rid")
    [ "$n" -eq 1 ] || fail "request $rid: expected 1 trace, saw $n: $(traces_for "$rid" | tr '\n' ' ')"
    pass "request $rid produced one trace in the Collector, with unrelated request $other traced in the same window"
}

t_trace
