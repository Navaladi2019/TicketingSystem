#!/usr/bin/env bash
# Item 3: WAL bytes per audit entry, written vs sealed (R11 seal UPDATE).
# 10,000 entries for tenant 3, one INSERT per transaction through PgBouncer as
# nava_app; then sealed by nava_worker in 10 batches of 1,000 (R16 A2), one
# transaction per batch. WAL measured with pg_walinspect between LSNs.
#   warm: seal right after the inserts (pages already full-page-written since the checkpoint)
#   cold: CHECKPOINT between insert and seal (every touched page pays a full-page image)
set -euo pipefail
export MSYS_NO_PATHCONV=1
cd "$(dirname "$0")"
DC="docker compose -f compose.yaml"
N=10000
su() { $DC exec -T pg psql -U postgres -d spike -qAt -v ON_ERROR_STOP=1 "$@"; }
lsn() { su -c "SELECT pg_current_wal_insert_lsn()"; }
stats() { # <label> <from> <to>
  echo "== $1 ($N entries)"
  su -F ' | ' -c "SELECT rm, count, record_size, fpi_size, combined_size
                    FROM pg_get_wal_stats('$2', '$3') AS w(rm, count, count_pct, record_size, record_pct, fpi_size, fpi_pct, combined_size, combined_pct) WHERE count > 0 ORDER BY combined_size DESC" | sed 's/^/  /'
  su -c "SELECT '  TOTAL bytes=' || pg_wal_lsn_diff('$3', '$2') || '  bytes/entry=' || round(pg_wal_lsn_diff('$3', '$2') / $N)"
}
detail() { # per record type, Heap and Btree only
  echo "   -- $1 by record type"
  su -F ' | ' -c "SELECT rm, count, record_size, fpi_size, combined_size
                    FROM pg_get_wal_stats('$2', '$3', true) AS w(rm, count, count_pct, record_size, record_pct, fpi_size, fpi_pct, combined_size, combined_pct)
                   WHERE count > 0 AND (rm LIKE 'Heap%' OR rm LIKE 'Btree%') ORDER BY combined_size DESC" | sed 's/^/     /'
}
insert_n() {
  $DC exec -T -e PGPASSWORD=spike pg pgbench -h pgbouncer -p 6432 -U nava_app -n -M prepared \
     -f /spike/bench/audit_only.pgbench -c 1 -t $N spike > /dev/null
}
seal_all() {
  local i
  for i in $(seq 1 $((N / 1000))); do
    $DC exec -T pg psql -U nava_worker -d spike -qc "CALL spike_seal_batch(md5('tenant-3')::uuid, 1000)"
  done
}
# Drain: seal every older unsealed entry of tenant 3 (item 1 writes some), so
# each seal below seals exactly the 10,000 entries just inserted.
while [ "$(su -c "SELECT count(*) FROM audit_log WHERE tenant_id = md5('tenant-3')::uuid AND sealed_at IS NULL")" != 0 ]; do
  $DC exec -T pg psql -U nava_worker -d spike -qc "CALL spike_seal_batch(md5('tenant-3')::uuid, 10000)"
done
# A: insert right after a checkpoint (cold insert), then seal (warm seal)
su -c "CHECKPOINT"
a=$(lsn); insert_n; b=$(lsn); seal_all; c=$(lsn)
stats "A1 insert, first touch after CHECKPOINT" "$a" "$b"
detail "A1 insert" "$a" "$b"
stats "A2 seal, warm (same pages already logged since the checkpoint)" "$b" "$c"
detail "A2 seal, warm" "$b" "$c"
# B: warm insert, then CHECKPOINT, then seal (cold seal)
d=$(lsn); insert_n; e=$(lsn)
stats "B1 insert, warm" "$d" "$e"
su -c "CHECKPOINT"; f=$(lsn); seal_all; g=$(lsn)
stats "B2 seal, cold (CHECKPOINT between insert and seal)" "$f" "$g"
detail "B2 seal, cold" "$f" "$g"
# C: warm insert, warm seal with the R11 BEFORE UPDATE trigger disabled (attribution only)
h=$(lsn); insert_n; i=$(lsn)
su -c "ALTER TABLE audit_log DISABLE TRIGGER audit_log_guard_update"
j=$(lsn); seal_all; k=$(lsn)
su -c "ALTER TABLE audit_log ENABLE TRIGGER audit_log_guard_update"
stats "C1 insert, warm" "$h" "$i"
stats "C2 seal, warm, R11 trigger DISABLED (attribution only)" "$j" "$k"
detail "C2 seal, warm, no R11 trigger" "$j" "$k"
su -c "SELECT '== check: unsealed left=' || count(*) FILTER (WHERE sealed_at IS NULL) || ', head seq=' || (SELECT seq FROM audit_chain_head WHERE tenant_id = md5('tenant-3')::uuid) FROM audit_log WHERE tenant_id = md5('tenant-3')::uuid"
su -c "SELECT '== HOT updates on audit_log_default (cumulative): ' || n_tup_hot_upd || ' of ' || n_tup_upd FROM pg_stat_user_tables WHERE relname = 'audit_log_default'"
