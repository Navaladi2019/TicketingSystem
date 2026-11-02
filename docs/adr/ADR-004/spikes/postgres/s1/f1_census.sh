#!/usr/bin/env bash
# F1 plan census (TICK-460 review, must-change 3). For the pinned form (v0) and
# the plpgsql-locals form (v1), on clean statistics, with the n_distinct
# override off and then on:
#   - three 2,000-subject commits, timed (no auto_explain loaded)
#   - which plan each of the 2,000 probes used: auto_explain without ANALYZE,
#     then with ANALYZE (auto_explain changes nothing about plan choice; the two
#     runs are two independent sessions)
# Leaves the override ON (schema.sql default) and the v0 function installed.
set -euo pipefail
export MSYS_NO_PATHCONV=1
cd "$(dirname "$0")"
DC="docker compose -f compose.yaml"
N=2000
su() { $DC exec -T pg psql -U postgres -d spike -qAt -v ON_ERROR_STOP=1 "$@"; }
fix() {
  if [ "$1" = on ]; then v="SET (n_distinct = -0.0001)"; else v="RESET (n_distinct)"; fi
  su -c "ALTER TABLE audit_log ALTER COLUMN occurred_at $v" -c "ALTER TABLE audit_log_default ALTER COLUMN occurred_at $v"
}
clean() { su -c "DELETE FROM audit_log WHERE seq IS NULL"; su -c "VACUUM (ANALYZE) audit_log_default"; }
txn() { # <tenant>
  cat <<SQL
BEGIN;
SET LOCAL ROLE nava_app;
SELECT set_config('app.tenant_id', md5('tenant-$1')::uuid::text, true) \g /dev/null
WITH r AS (UPDATE records SET version = version + 1 WHERE tenant_id = md5('tenant-$1')::uuid AND number <= $N RETURNING id, version)
INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
SELECT md5('tenant-$1')::uuid, gen_random_uuid(), 'record', 'update', 'record', r.id, r.version, 'system', 'system' FROM r;
\timing on
COMMIT;
SQL
}
census() { # <tenant> <analyze on|off>
  { echo "LOAD 'auto_explain';"
    echo "SET auto_explain.log_min_duration = 0; SET auto_explain.log_nested_statements = on; SET auto_explain.log_level = notice; SET client_min_messages = notice;"
    [ "$2" = on ] && echo "SET auto_explain.log_analyze = on;"
    txn "$1"; } | $DC exec -T pg psql -U postgres -d spike -q 2>&1 \
    | awk '/Query Text: SELECT 1 FROM audit_log/{q=1} q && /(Index Only Scan|Index Scan) using/{
             match($0, /(Index Only Scan|Index Scan) using [a-z0-9_]+/); k=substr($0, RSTART, RLENGTH);
             g = (getline nxt) > 0 && nxt ~ /\$[0-9]/ ? "generic" : "custom"; c[k" ("g")"]++; q=0 }
           END { for (k in c) printf "      %5d probes: %s\n", c[k], k }'
}
t=11  # seeded tenants are 1..50; 24 transactions use 11..34
for f in off on; do
  fix "$f"; clean
  echo "######## n_distinct override: $f (clean stats; $N subjects per commit)"
  su -c "SELECT '  pg_stats occurred_at: n_distinct=' || n_distinct FROM pg_stats WHERE tablename = 'audit_log_default' AND attname = 'occurred_at'"
  for v in v0-as-pinned v1-locals; do
    INSTALL_ONLY=$v bash ./item2_variants.sh
    echo "  == $v"
    for r in 1 2 3; do
      echo "    commit $r: $(txn $t | $DC exec -T pg psql -U postgres -d spike -q 2>&1 | grep '^Time')"; t=$((t + 1))
    done
    echo "    plan census, auto_explain without ANALYZE:"; census $t off; t=$((t + 1))
    echo "    plan census, auto_explain with ANALYZE:";    census $t on;  t=$((t + 1))
    clean
  done
done
# Stability: does the plan depend on ANALYZE's random sample? For each form,
# K fresh ANALYZEs; after each, a new session runs an 8-subject transaction and
# auto_explain reports the index of the 8th probe (the cached generic plan).
K=${K:-10}
stability() { # <tenant>
  { echo "LOAD 'auto_explain';"
    echo "SET auto_explain.log_min_duration = 0; SET auto_explain.log_nested_statements = on; SET auto_explain.log_level = notice; SET client_min_messages = notice;"
    txn "$1" | sed "s/number <= $N/number <= 8/"; } | $DC exec -T pg psql -U postgres -d spike -q 2>&1     | grep -oE "(Index Only Scan|Index Scan) using [a-z0-9_]+" | tail -1
}
for f in off on; do
  fix "$f"
  echo "######## stability, n_distinct override: $f ($K fresh ANALYZE samples per form)"
  for v in v0-as-pinned v1-locals; do
    INSTALL_ONLY=$v bash ./item2_variants.sh
    for k in $(seq 1 "$K"); do
      su -c "ANALYZE audit_log_default"
      nd=$(su -c "SELECT n_distinct FROM pg_stats WHERE tablename = 'audit_log_default' AND attname = 'occurred_at'")
      echo "$v n_distinct=$nd $(stability 45)"
    done > out/gen/stability.txt
    awk -v K="$K" '{k=$0; sub(/^[^ ]+ [^ ]+ /, "", k); c[$1"|"k]++; nd[$1]=nd[$1]" "substr($2,12)}
      END {for (x in c) {split(x, a, "|"); printf "  %-14s %2d/%d probes: %s\n", a[1], c[x], K, a[2]}
           for (v in nd) printf "  %-14s n_distinct samples:%s\n", v, nd[v]}' out/gen/stability.txt | sort
    clean
  done
done
INSTALL_ONLY=v0-as-pinned bash ./item2_variants.sh
