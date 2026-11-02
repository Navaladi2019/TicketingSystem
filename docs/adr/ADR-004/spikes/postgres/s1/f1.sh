#!/usr/bin/env bash
# F1 evidence on clean statistics (TICK-460 review, must-change 2 and 3).
# Run on a FRESH stack right after seed: ./run.sh down; ./run.sh up; ./run.sh seed; ./run.sh f1
#   1. n_distinct override OFF (as pinned), clean ANALYZE:
#      the trigger's cached generic plan for a 1-subject transaction (item 1's shape),
#      then the five probe forms (plan + 2,000-subject commit)      -> out/f1-fix-off.txt
#   2. delete every post-seed audit row (all unsealed), override ON, VACUUM ANALYZE:
#      the same two measurements                                  -> out/f1-fix-on.txt
#   3. delete post-seed rows again, ANALYZE, then item 2 with the override ON
#      (10k update, 10k insert, 100k insert)                        -> out/item2.txt
set -euo pipefail
export MSYS_NO_PATHCONV=1
cd "$(dirname "$0")"
DC="docker compose -f compose.yaml"
su() { $DC exec -T pg psql -U postgres -d spike -qAt -v ON_ERROR_STOP=1 "$@"; }
fix() {
  if [ "$1" = on ]; then v="SET (n_distinct = -0.0001)"; else v="RESET (n_distinct)"; fi
  su -c "ALTER TABLE audit_log ALTER COLUMN occurred_at $v" -c "ALTER TABLE audit_log_default ALTER COLUMN occurred_at $v"
}
clean() { # remove every row written after seed (seed rows are all sealed), then fresh stats
  su -c "DELETE FROM audit_log WHERE seq IS NULL"
  su -c "VACUUM (ANALYZE) audit_log_default"
}
stats() {
  su -c "SELECT '  pg_stats occurred_at: n_distinct=' || n_distinct || ', top MCV freq=' || coalesce(most_common_freqs[1]::text, 'none')
           FROM pg_stats WHERE tablename = 'audit_log_default' AND attname = 'occurred_at'"
  su -c "SELECT '  attoptions: ' || coalesce(attoptions::text, 'none') FROM pg_attribute WHERE attrelid = 'audit_log_default'::regclass AND attname = 'occurred_at'"
  su -c "SELECT '  max rows sharing one occurred_at: ' || max(c) FROM (SELECT count(*) c FROM audit_log_default GROUP BY occurred_at) x"
}
# The trigger's cached plan for item 1's shape: 8 one-subject transactions in one
# session as nava_app; auto_explain prints the probe of the 8th (generic plan).
trigger_plan() {
  {
    echo "LOAD 'auto_explain';"
    echo "SET auto_explain.log_min_duration = 0; SET auto_explain.log_nested_statements = on;"
    echo "SET auto_explain.log_analyze = on; SET auto_explain.log_buffers = on; SET auto_explain.log_level = notice;"
    echo "SET client_min_messages = notice;"
    for k in 1 2 3 4 5 6 7 8; do
      [ $k = 8 ] && echo "\echo ---- 8th one-subject transaction: the probe below runs on the cached generic plan"
      cat <<SQL
BEGIN;
SET LOCAL ROLE nava_app;
SELECT set_config('app.tenant_id', md5('tenant-9')::uuid::text, true) \g /dev/null
UPDATE records SET version = version + 1 WHERE tenant_id = md5('tenant-9')::uuid AND id = md5('rec-9-$k')::uuid;
INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
VALUES (md5('tenant-9')::uuid, gen_random_uuid(), 'record', 'update', 'record', md5('rec-9-$k')::uuid, 99, 'system', 'system');
COMMIT;
SQL
    done
  } | $DC exec -T pg psql -U postgres -d spike -q 2>&1 | sed -n '/---- 8th/,$p' | awk '/Query Text: SELECT 1 FROM audit_log/{p=1} p' | sed -n '1,/^psql.*NOTICE:  duration/p' | grep -v "NOTICE:  duration" | sed 's/^/    /'
}
section() { # <fix> <file>
  {
    echo "######## n_distinct override: $1"
    stats
    echo "== trigger's cached generic plan, one-subject transaction (item 1 shape)"
    trigger_plan
    echo "== probe forms, N=2000 subjects per commit"
    N=2000 bash ./item2_variants.sh 2>&1 | grep -v '^$'
  } | tee "$2"
}
fix off; clean
section off out/f1-fix-off.txt
clean; fix on; su -c "ANALYZE audit_log_default"
section on out/f1-fix-on.txt
clean
ITEM2_SPECS="on update 10000|on insert 10000|on insert 100000" bash ./item2.sh 2>&1 | tee out/item2.txt
