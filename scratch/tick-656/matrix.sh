#!/bin/sh
# TICK-656: every scenario of the acceptance criteria through explain.sh, run
# twice (the first warms), keeping the second; plans to $OUT, one summary line
# per statement: the first records index it scans and how, how many Sort
# nodes it has (a Merge Append's Sort Key is not a Sort), the largest Rows
# Removed by Filter, shared buffers and execution time.
#   OUT=scratch/tick-656/plans-x.md sh scratch/tick-656/matrix.sh
set -eu
cd "$(dirname "$0")/../.."
OUT=${OUT:-bin/tick656-plans.md}
: > "$OUT"
for sp in "agent-state-new 1" "agent 1" "agent 20" "agent-group-desc 1" "agent-group-desc 20" "agent-group-asc 1" "admin-state-in 1"; do
    set -- $sp
    sh scratch/tick-656/explain.sh "$1" "$2" > /dev/null
    f=$(mktemp)
    sh scratch/tick-656/explain.sh "$1" "$2" > "$f"
    cat "$f" >> "$OUT"
    awk -v l="$1 p$2" '
        /^count:/ {s="count"; n++; ord[n]=s} /^page:/ {s="page"; n++; ord[n]=s}
        /-> +Sort  |^Sort  / {srt[s]++}
        /Scan (using|on) records_[a-z_]+/ && !scan[s] {match($0, /Scan (using|on) records_[a-z_]+/); scan[s]=substr($0, RSTART, RLENGTH); sub(/^Scan (using|on) /, "", scan[s]); kind[s]=($0 ~ /Bitmap/) ? "bitmap" : (($0 ~ /Only/) ? "index-only" : "index")}
        /Rows Removed by Filter/ {split($0,a,": "); if (a[2]+0 > rr[s]) rr[s]=a[2]+0}
        /Buffers: shared hit=/ && !buf[s] {split($0,b,"hit="); split(b[2],bb," "); buf[s]=bb[1]}
        /Execution Time/ {ex[s]=$3}
        END {for (i=1;i<=n;i++) {k=ord[i]; printf "%-22s %-5s sorts=%-2s %-26s %-10s max-removed=%-6s buf=%-6s exec=%s\n", l, k, srt[k]+0, scan[k], kind[k], rr[k]+0, buf[k], ex[k]}}' "$f"
    rm -f "$f"
done
