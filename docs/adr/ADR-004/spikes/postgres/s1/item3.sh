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
  su -F ' | ' -c "SELECT resource_manager, count, record_size, fpi_size, combined_size
                    FROM pg_get_wal_stats('$2', '$3') WHERE count > 0 ORDER BY combined_size DESC" | sed 's/^/  /'
  su -c "SELECT '  TOTAL bytes=' || pg_wal_lsn_diff('$3', '$2') || '  bytes/entry=' || round(pg_wal_lsn_diff('$3', '$2') / $N)"
}
insert_n() {
  $DC exec -T -e PGPASSWORD=spike pg pgbench -h pgbouncer -p 6432 -U nava_app -n -M prepared \
     -f /spike/bench/audit_only.pgbench -c 1 -t $N spike > /dev/null
}
seal_all() {
  for i in $(seq 1 $((N / 1000))); do
    $DC exec -T pg psql -U nava_worker -d spike -qc "CALL spike_seal_batch(md5('tenant-3')::uuid, 1000)"
  done
}
su -c "CHECKPOINT"
a=$(lsn); insert_n; b=$(lsn); seal_all; c=$(lsn)
stats "insert" "$a" "$b"
stats "seal, warm (no checkpoint since insert)" "$b" "$c"
d=$(lsn); insert_n; e=$(lsn); su -c "CHECKPOINT"; f=$(lsn); seal_all; g=$(lsn)
stats "insert (second run)" "$d" "$e"
stats "seal, cold (CHECKPOINT between insert and seal)" "$f" "$g"
su -c "SELECT '== check: unsealed left=' || count(*) FILTER (WHERE sealed_at IS NULL) || ', head seq=' || (SELECT seq FROM audit_chain_head WHERE tenant_id = md5('tenant-3')::uuid) FROM audit_log WHERE tenant_id = md5('tenant-3')::uuid"
su -c "SELECT '== HOT updates on audit_log_default: ' || n_tup_hot_upd || ' of ' || n_tup_upd FROM pg_stat_user_tables WHERE relname = 'audit_log_default'"
