#!/bin/sh
# TICK-640: runs explain.sh over the acceptance matrix (agent, heaviest and
# lightest heavy requester, admin; page 1 and 20) for the given sorts, writes
# every plan to $OUT and prints one summary line per statement.
#   OUT=bin/x.md sh scratch/tick-640/matrix.sh <sort>...
set -eu
cd "$(dirname "$0")/../.."
OUT=${OUT:-bin/tick640-plans.md}
: > "$OUT"
for s in "$@"; do for a in agent req1 req8 admin; do for p in 1 20; do
    f=$(mktemp)
    sh scratch/tick-640/explain.sh "$a" "$s" "$p" ${COUNT:+count} > "$f"
    cat "$f" >> "$OUT"
    awk -v l="$a $s p$p${FILTER:+ [$FILTER]}" 'BEGIN {c=0}
        /^count:/ {c=1}
        / Scan using | Seq Scan on records/ && !scan[c] && /records/ {x=$0; sub(/^ *(->)? */,"",x); sub(/ +\(cost.*/,"",x); scan[c]=x}
        /Sort/ && /->|^Sort|^ *Sort/ {sort[c]="SORT"}
        /Buffers: shared hit=/ && !buf[c] {split($0,b,"hit="); split(b[2],bb," "); buf[c]=bb[1]}
        /Execution Time/ {ex[c]=$3}
        /^ *Timing: .*Total/ {j=$0; sub(/.*Total /,"",j); jit[c]=j}
        END {for (i=0;i<=c;i++) printf "%-40s %-6s %-60s buf=%-7s exec=%-9s %s\n", (i?"  count":l), sort[i]?sort[i]:"-", scan[i], buf[i], ex[i], (i in jit)?"jit " jit[i]:""}' "$f"
    rm -f "$f"
done; done; done
