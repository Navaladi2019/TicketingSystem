#!/bin/sh
# Markdown tables from the RESULT lines of a run.sh output file:
#   sh bench/records-list/table.sh bin/records-list-bench-output.txt
# One table per pass (plan_cache_mode) and per index state (the "###" step
# header before the lines). Cells: p50 / p95 / p99 in ms.
awk '
function cell(k) { return (k in v) ? v[k] : "n/a" }
function flush(  i, key) {
    if (n == 0) return
    print ""
    print "#### " title
    print ""
    print "| Query | Tenant | Page | RLS alone | Explicit `tenant_id = $1` |"
    print "| --- | --- | --- | --- | --- |"
    for (i = 1; i <= n; i++) {
        key = rows[i]
        split(key, f, SUBSEP)
        print "| " f[1] " | " f[2] " | " f[3] " | " cell(key SUBSEP "rls") " | " cell(key SUBSEP "explicit") " |"
    }
    n = 0; delete seen; delete v; delete rows
}
/^### / { flush(); title = substr($0, 5); next }
/^RESULT/ {
    plan = ""; q = ""; mode = ""; t = ""; p = ""; pg = ""; p50 = ""; p95 = ""; p99 = ""
    for (i = 2; i <= NF; i++) {
        split($i, kv, "=")
        if (kv[1] == "plan") plan = kv[2]
        if (kv[1] == "q") q = kv[2]
        if (kv[1] == "mode") mode = kv[2]
        if (kv[1] == "tenant") t = kv[2]
        if (kv[1] == "pred") p = kv[2]
        if (kv[1] == "page") pg = kv[2]
        if (kv[1] == "p50_ms") p50 = kv[2]
        if (kv[1] == "p95_ms") p95 = kv[2]
        if (kv[1] == "p99_ms") p99 = kv[2]
    }
    if (p50 == "") next
    name = "(" q ")" (mode == "count" ? " capped count" : " list")
    if (plan != "" && index(title, plan) == 0) title = title " [plan_cache_mode=" plan "]"
    key = name SUBSEP t SUBSEP (mode == "count" ? "-" : pg)
    if (!(key in seen)) { seen[key] = 1; rows[++n] = key }
    v[key SUBSEP p] = p50 " / **" p95 "** / " p99
}
END { flush() }
' "$1"
