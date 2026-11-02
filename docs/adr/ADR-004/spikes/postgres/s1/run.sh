#!/usr/bin/env bash
# Spike S1 (TICK-423) runner. Uses only the compose file in this folder.
#   ./run.sh up        start postgres:16.10 + pgbouncer v1.24.1-p1, load the schema
#   ./run.sh item4     R10 assert trigger on the partitioned table (empty table; run before seed)
#   ./run.sh seed      50 tenants, 1M records, 1.98M sealed audit entries
#   ./run.sh item1     p95 of the commit-check trigger, 10k iterations per arm, through PgBouncer
#                      (CELL=update-sync-c1 ./run.sh item1 reruns one cell and appends to the summary)
#   ./run.sh item2     10k- and 100k-subject transactions: commit time, deferred-queue memory
#   ./run.sh item3     WAL bytes per entry, insert vs seal UPDATE
#   ./run.sh item5     export paging on the 1M-entry tenant, through PgBouncer
#   ./run.sh all       up, item4, seed, item1, item2, item3, item5
#   ./run.sh down      stop and delete the volume
# Results land in ./out (summaries are committed; raw pgbench logs are not).
set -euo pipefail
export MSYS_NO_PATHCONV=1 MSYS2_ARG_CONV_EXCL='*'
cd "$(dirname "$0")"
DC="docker compose -f compose.yaml"
psql_su() { $DC exec -T pg psql -U postgres -d spike -v ON_ERROR_STOP=1 "$@"; }
psql_bouncer() { local role=$1; shift; $DC exec -T -e PGPASSWORD=spike pg psql -h pgbouncer -p 6432 -U "$role" -d spike -v ON_ERROR_STOP=1 "$@"; }

ITER=${ITER:-10000}
ROUNDS=${ROUNDS:-3}

up() {
  $DC up -d --wait
  psql_su -q -f /spike/sql/schema.sql
}

trigger() { psql_su -qc "ALTER TABLE records $1 TRIGGER records_audit_commit_check"; }

# pgbench_run <name> <script> <clients>; prints one summary line
pgbench_run() {
  local name=$1 script=$2 clients=$3 per=$(( ITER / $3 ))
  $DC exec -T pg sh -c "rm -f /out/item1/raw/$name.*"
  $DC exec -T -e PGPASSWORD=spike pg pgbench -h pgbouncer -p 6432 -U nava_app -n -M prepared \
      -f "$script" -c "$clients" -j "$clients" -t "$per" -l --log-prefix="/out/item1/raw/$name" spike \
      > "out/item1/pgbench-$name.txt" 2>&1
  # pgbench log column 3 = transaction latency in microseconds
  $DC exec -T pg sh -c "cat /out/item1/raw/$name.* | grep -v summary | awk '{print \$3}' | sort -n | awk -v n=$name '
      {a[NR]=\$1; s+=\$1} END {printf \"%-28s n=%d mean=%.0f p50=%d p95=%d p99=%d\\n\", n, NR, s/NR, a[int(NR*0.50)], a[int(NR*0.95)], a[int(NR*0.99)]}'" \
      | tee -a out/item1/runs.txt
}

BLOCKS=${BLOCKS:-20}   # blocks per arm; ITER/BLOCKS transactions per block

# Item 1. Per cell (mode x clients x scenario): trigger off and on alternate in
# BLOCKS short blocks each, in ABBA order (off,on,on,off,...), so host noise
# (WSL2 fsync jitter, autovacuum, checkpoints) falls on both arms evenly.
# Each arm totals ITER transactions. Verdict: pooled p95(on) vs p95(off).
item1() {
  mkdir -p out/item1/raw out/gen
  if [ -z "${CELL:-}" ]; then rm -f out/item1/pgbench-*.txt; : > out/item1/runs.txt; : > out/item1/summary.txt; fi
  for s in update insert; do
    sed 's/^BEGIN;$/BEGIN;\nSET LOCAL synchronous_commit = off;/' "bench/$s.pgbench" > "out/gen/$s-async.pgbench"
  done
  local per_block=$(( ITER / BLOCKS ))
  trigger ENABLE
  for s in update insert; do  # warm-up
    $DC exec -T -e PGPASSWORD=spike pg pgbench -h pgbouncer -p 6432 -U nava_app -n -M prepared -f "/spike/bench/$s.pgbench" -c 4 -j 4 -t 500 spike > /dev/null 2>&1
  done
  for mode in sync async; do
    for c in 1 8; do
      for s in update insert; do
        if [ "$mode" = sync ]; then f="/spike/bench/$s.pgbench"; else f="/out/gen/$s-async.pgbench"; fi
        cell="$s-$mode-c$c"
        if [ -n "${CELL:-}" ] && [ "$cell" != "$CELL" ]; then continue; fi
        $DC exec -T pg sh -c "rm -rf /out/item1/raw/$cell; mkdir -p /out/item1/raw/$cell"
        psql_su -qAtc "SELECT pg_stat_reset()" >/dev/null
        for b in $(seq 1 "$BLOCKS"); do
          if [ $((b % 2)) -eq 1 ]; then order="off on"; else order="on off"; fi
          for arm in $order; do
            if [ "$arm" = on ]; then trigger ENABLE; else trigger DISABLE; fi
            $DC exec -T -e PGPASSWORD=spike pg pgbench -h pgbouncer -p 6432 -U nava_app -n -M prepared \
              -f "$f" -c "$c" -j "$c" -t $(( per_block / c )) -l --log-prefix="/out/item1/raw/$cell/$arm-b$b" spike \
              >> "out/item1/pgbench-$cell.txt" 2>&1
          done
        done
        sleep 2  # let backends flush function stats
        psql_su -qAtc "SELECT '$cell commit check: calls=' || calls || ' self_ms=' || round(self_time::numeric,1) || ' us/call=' || round((self_time*1000/calls)::numeric,1) FROM pg_stat_user_functions WHERE funcname='audit_commit_check'" | tee -a out/item1/runs.txt
        summarize_cell "$cell" | tee -a out/item1/summary.txt
      done
    done
  done
  trigger ENABLE
  probe_plan | tee out/item1/probe-plan.txt
}

# pooled p50/p95/p99 per arm, plus the median of the per-block-pair p95 ratios
summarize_cell() {
  local cell=$1
  $DC exec -T pg sh -c "
    pct() { cat \"\$@\" | awk '{print \$3}' | sort -n | awk '{a[NR]=\$1; s+=\$1} END {printf \"%d %.0f %d %d %d\", NR, s/NR, a[int(NR*0.50)], a[int(NR*0.95)], a[int(NR*0.99)]}'; }
    cd /out/item1/raw/$cell
    off=\$(pct off-b*); on=\$(pct on-b*)
    ratios=\$(for b in \$(seq 1 $BLOCKS); do
      p0=\$(pct off-b\$b.* | cut -d' ' -f4); p1=\$(pct on-b\$b.* | cut -d' ' -f4); echo \"\$p1 \$p0\" | awk '{printf \"%.4f\n\", \$1/\$2}'; done | sort -n | awk '{a[NR]=\$1} END {print a[int((NR+1)/2)]}')
    echo \"\$off \$on \$ratios\" | awk -v cell=$cell '{
      d=(\$9-\$4)*100/\$4;
      printf \"%-18s n/arm=%d  p50 %d->%d us  p95 %d->%d us (%+.1f%%)  p99 %d->%d us  mean %+.1f%%  median block-pair p95 ratio %+.1f%%  %s\n\",
        cell, \$1, \$3, \$8, \$4, \$9, d, \$5, \$10, (\$7-\$2)*100/\$2, (\$11-1)*100, (d <= 10 ? \"PASS\" : \"FAIL\") }'"
}

probe_plan() {
  psql_bouncer nava_app <<'SQL'
BEGIN;
SELECT set_config('app.tenant_id', md5('tenant-1')::uuid::text, true);
UPDATE records SET version = version + 1 WHERE tenant_id = md5('tenant-1')::uuid AND id = md5('rec-1-7')::uuid;
INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
VALUES (md5('tenant-1')::uuid, gen_random_uuid(), 'record', 'update', 'record', md5('rec-1-7')::uuid, 2, 'system', 'system');
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT 1 FROM audit_log
 WHERE tenant_id = md5('tenant-1')::uuid AND occurred_at = now() AND txid = pg_current_xact_id()
   AND subject_type = 'record' AND subject_id = md5('rec-1-7')::uuid LIMIT 1;
ROLLBACK;
SQL
}

item2() { bash ./item2.sh 2>&1 | tee out/item2.txt; }
item3() { bash ./item3.sh 2>&1 | tee out/item3.txt; }
item4() { mkdir -p out; psql_su -f /spike/sql/item4.sql > out/item4.txt 2>&1; sed -n '/== item 4 cases/,$p' out/item4.txt; }
seed()  { mkdir -p out; psql_su -f /spike/sql/seed.sql 2>&1 | tee out/seed.txt; }
item5() { bash ./item5.sh 2>&1 | tee out/item5.txt; }
down()  { $DC down -v; }
all()   { up; item4; seed; item1; item2; item3; item5; }

"${1:-all}"
