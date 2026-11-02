#!/usr/bin/env bash
# Post-hoc (disclosed in RESULT.md): pooled p95 per cell after dropping, from
# BOTH arms, every block pair where either arm's block p95 exceeds 2x the
# cell's median block p95 (a host stall: WSL2 fsync, checkpoint, autovacuum).
set -euo pipefail
export MSYS_NO_PATHCONV=1
cd "$(dirname "$0")"
docker compose -f compose.yaml exec -T pg sh -c '
p95() { cat "$@" | awk "{print \$3}" | sort -n | awk "{x[NR]=\$1} END {print x[int(NR*0.95)]}"; }
for d in /out/item1/raw/*/; do
  cell=$(basename $d); cd $d
  n=$(ls off-b*.* | sed "s/off-b\([0-9]*\)\..*/\1/" | sort -un | wc -l)
  for b in $(seq 1 $n); do echo "$b $(p95 off-b$b.*) $(p95 on-b$b.*)"; done > /tmp/blocks
  med=$(awk "{print \$2; print \$3}" /tmp/blocks | sort -n | awk "{x[NR]=\$1} END {print x[int((NR+1)/2)]}")
  keep=$(awk -v m=$med "\$2 <= 2*m && \$3 <= 2*m {print \$1}" /tmp/blocks)
  dropped=$(awk -v m=$med "\$2 > 2*m || \$3 > 2*m {printf \"b%s \", \$1}" /tmp/blocks)
  off=$(p95 $(for b in $keep; do ls off-b$b.*; done)); on=$(p95 $(for b in $keep; do ls on-b$b.*; done))
  awk -v c=$cell -v a=$off -v b=$on -v dr="$dropped" -v k=$(echo $keep | wc -w) -v n=$n "BEGIN {printf \"%-22s kept %2d/%d pairs  p95 %d->%d us (%+.1f%%)  dropped: %s\n\", c, k, n, a, b, (b-a)*100/a, (dr==\"\" ? \"none\" : dr)}"
done'
