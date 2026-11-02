#!/usr/bin/env bash
# F1 follow-up: is the plpgsql-locals probe (v1) stable without the n_distinct
# override? Five fresh ANALYZE samples right after VACUUM (table ~100%
# all-visible) and five after ordinary scattered writes, with the override off
# and then on; reports the index of the cached generic plan each time.
# Output: out/f1-vm.txt. Leaves the override ON and v0 installed (schema.sql state).
set -euo pipefail
export MSYS_NO_PATHCONV=1
cd "$(dirname "$0")"
DC="docker compose -f compose.yaml"
su() { $DC exec -T pg psql -U postgres -d spike -qAt -v ON_ERROR_STOP=1 "$@"; }
vm() { su -c "SELECT 'relpages=' || relpages || ' relallvisible=' || relallvisible || ' (' || round(100.0 * relallvisible / relpages, 1) || '% all-visible)' FROM pg_class WHERE relname = 'audit_log_default'"; }
probe() { # index used by the 8th probe of an 8-subject transaction (generic plan)
  { echo "LOAD 'auto_explain'; SET auto_explain.log_min_duration = 0; SET auto_explain.log_nested_statements = on; SET auto_explain.log_level = notice; SET client_min_messages = notice;"
    echo "BEGIN; SET LOCAL ROLE nava_app; SELECT set_config('app.tenant_id', md5('tenant-45')::uuid::text, true) \g /dev/null"
    echo "WITH r AS (UPDATE records SET version = version + 1 WHERE tenant_id = md5('tenant-45')::uuid AND number <= 8 RETURNING id, version)
          INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
          SELECT md5('tenant-45')::uuid, gen_random_uuid(), 'record', 'update', 'record', r.id, r.version, 'system', 'system' FROM r; COMMIT;"
  } | $DC exec -T pg psql -U postgres -d spike -q 2>&1 | grep -oE "(Index Only Scan|Index Scan) using [a-z0-9_]+" | tail -1
}
for f in off on; do
  if [ $f = on ]; then v="SET (n_distinct = -0.0001)"; else v="RESET (n_distinct)"; fi
  su -c "ALTER TABLE audit_log ALTER COLUMN occurred_at $v" -c "ALTER TABLE audit_log_default ALTER COLUMN occurred_at $v"
  INSTALL_ONLY=v1-locals bash ./item2_variants.sh
  echo "######## n_distinct override: $f, probe form v1-locals"
  su -c "DELETE FROM audit_log WHERE seq IS NULL"; su -c "VACUUM (ANALYZE) audit_log_default"
  echo "  after VACUUM (ANALYZE): $(vm)"
  for k in 1 2 3 4 5; do su -c "ANALYZE audit_log_default"; echo "    ANALYZE $k: $(probe)"; done
  # scattered writes: 2,000 new entries over 10 tenants, plus touching 2,500 old pages of tenant 1
  for t in 11 12 13 14 15 16 17 18 19 20; do
    su -c "BEGIN; SET LOCAL ROLE nava_app; SELECT set_config('app.tenant_id', md5('tenant-$t')::uuid::text, true);
           INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source)
           SELECT md5('tenant-$t')::uuid, gen_random_uuid(), 'record', 'update', 'record', gen_random_uuid(), 'system', 'system' FROM generate_series(1, 200); COMMIT;" > /dev/null
  done
  su -c "SET session_replication_role = replica; UPDATE audit_log_default SET request_id = request_id WHERE tenant_id = md5('tenant-1')::uuid AND seq % 400 = 0"
  su -c "ANALYZE audit_log_default"
  echo "  after scattered writes + ANALYZE: $(vm)"
  for k in 1 2 3 4 5; do su -c "ANALYZE audit_log_default"; echo "    ANALYZE $k: $(probe)"; done
done
su -c "DELETE FROM audit_log WHERE seq IS NULL"; su -c "VACUUM (ANALYZE) audit_log_default"
INSTALL_ONLY=v0-as-pinned bash ./item2_variants.sh
