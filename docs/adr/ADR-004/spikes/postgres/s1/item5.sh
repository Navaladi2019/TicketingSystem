#!/usr/bin/env bash
# Item 5: export caps (R20/R21) on tenant 1 = 1,000,000 sealed entries, one of
# 50 tenants (1.98M entries in audit_log). Through PgBouncer as a reader role,
# each page its own transaction with SET LOCAL statement_timeout = '30s',
# keyset on (tenant_id, seq) WHERE seq IS NOT NULL, 5,000 rows per page, the
# client reading every byte. Two passes: unfiltered (every line full, worst
# case bytes) and filtered (one subject full, the rest stub columns only).
set -euo pipefail
export MSYS_NO_PATHCONV=1
cd "$(dirname "$0")"
DC="docker compose -f compose.yaml"
PAGE=5000
mkdir -p out/gen
bouncer() { $DC exec -T -e PGPASSWORD=spike pg psql -h pgbouncer -p 6432 -U spike_export -d spike -v ON_ERROR_STOP=1 "$@"; }
T="md5('tenant-1')::uuid"
SUBJ="md5('rec-1-77')::uuid"
FULL="seq, encode(prev_hash,'hex'), encode(entry_hash,'hex'), id, occurred_at, domain, action, subject_type, subject_id,
      entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id,
      causation_id, causation_depth, request_id, backfilled, encode(content_sha256,'hex'), encode(diff_sha256,'hex'),
      diff_truncated, diff::text, encode(diff_salt,'hex'), sealed_at, erased_at, erasure_ref"
# Filtered: the envelope is still read for every row (the exporter must hash it
# for the stub's envelope_sha256); only the diff is skipped for stubs.
FILT="seq, encode(prev_hash,'hex'), encode(entry_hash,'hex'), id, occurred_at, domain, action, subject_type, subject_id,
      entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id,
      causation_id, causation_depth, request_id, backfilled, encode(content_sha256,'hex'), encode(diff_sha256,'hex'),
      diff_truncated, CASE WHEN subject_id = $SUBJ THEN diff::text END, CASE WHEN subject_id = $SUBJ THEN encode(diff_salt,'hex') END,
      sealed_at, erased_at, erasure_ref"

echo "== R20 anchor: first sealed seq in a time window, regardless of filter"
for w in "first day|now() - interval '90 days'|now() - interval '89 days'" \
         "middle day|now() - interval '45 days'|now() - interval '44 days'" \
         "last day|now() - interval '2 days'|now() - interval '1 day'" \
         "whole 90 days|now() - interval '91 days'|now()"; do
  IFS='|' read -r label from to <<<"$w"
  echo "-- window: $label"
  bouncer -q <<SQL
BEGIN;
SET LOCAL statement_timeout = '30s';
SELECT set_config('app.tenant_id', $T::text, true) \g /dev/null
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT min(seq) FROM audit_log WHERE tenant_id = $T AND occurred_at >= $from AND occurred_at < $to AND seq IS NOT NULL;
COMMIT;
SQL
done

echo "== one deep page (seq > 900000), plan"
bouncer -q <<SQL
BEGIN;
SET LOCAL statement_timeout = '30s';
SELECT set_config('app.tenant_id', $T::text, true) \g /dev/null
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT $FULL FROM audit_log WHERE tenant_id = $T AND seq IS NOT NULL AND seq > 900000 AND seq <= 1000000 ORDER BY seq LIMIT $PAGE;
COMMIT;
SQL

for pass in unfiltered filtered; do
  if [ $pass = unfiltered ]; then COLS=$FULL; else COLS=$FILT; fi
  gen=out/gen/export-$pass.sql
  {
    echo "\pset tuples_only on"
    echo "\pset format unaligned"
    echo "BEGIN;"
    echo "SELECT set_config('app.tenant_id', $T::text, true) \g /dev/null"
    echo "SELECT seq AS head FROM audit_chain_head WHERE tenant_id = $T \gset"
    echo "COMMIT;"
    echo "\set last 0"
    echo "\timing on"
    for p in $(seq 1 200); do
      echo "BEGIN;"
      echo "SET LOCAL statement_timeout = '30s';"
      echo "SELECT set_config('app.tenant_id', $T::text, true) \g /dev/null"
      echo "\o /out/page.txt"
      echo "SELECT $COLS FROM audit_log WHERE tenant_id = $T AND seq IS NOT NULL AND seq > :last AND seq <= :head ORDER BY seq LIMIT $PAGE;"
      echo "\o"
      echo "COMMIT;"
      echo "\set last \`tail -n1 /out/page.txt | cut -d'|' -f1\`"
      echo "\! wc -c < /out/page.txt >> /out/gen/bytes-$pass.txt"
    done
    echo "\timing off"
    echo "\echo last_seq=:last head_seq=:head"
  } > "$gen"
  rm -f "out/gen/bytes-$pass.txt"
  echo "== export pass: $pass (200 pages x $PAGE rows)"
  start=$(date +%s.%N)
  bouncer -q -f "/out/gen/export-$pass.sql" > "out/gen/export-$pass.out"
  end=$(date +%s.%N)
  grep -o 'last_seq=.*' "out/gen/export-$pass.out"
  # \timing lines in order: BEGIN, SET, set_config, SELECT page, COMMIT per page -> take the page SELECTs (4th of 5)
  grep '^Time:' "out/gen/export-$pass.out" | awk '{print $2}' | awk 'NR % 5 == 4' | sort -n | awk '
     {a[NR]=$1; s+=$1} END {printf "  page SELECT ms: n=%d mean=%.1f p50=%.1f p95=%.1f max=%.1f sum=%.1f\n", NR, s/NR, a[int(NR*0.5)], a[int(NR*0.95)], a[NR], s}'
  awk '{s+=$1} END {printf "  bytes received: %.1f MB\n", s/1048576}' "out/gen/bytes-$pass.txt"
  awk -v a="$start" -v b="$end" 'BEGIN {printf "  wall clock, 1,000,000 rows: %.1f s (cap 900 s)\n", b - a}'
done
