#!/usr/bin/env bash
# Item 2: one transaction touching N subjects. Commit time and backend memory of
# the deferred-trigger queue, trigger off vs on. Each case runs on a fresh
# backend as nava_app (unix socket, RLS applies). Not through PgBouncer: a
# fresh backend is needed so VmHWM (peak RSS) belongs to this transaction only.
set -euo pipefail
export MSYS_NO_PATHCONV=1
cd "$(dirname "$0")"
DC="docker compose -f compose.yaml"
case_sql() { # <kind> <n> <tag>
  local kind=$1 n=$2 tag=$3
  local write
  if [ "$kind" = update ]; then
    write="WITH r AS (UPDATE records SET version = version + 1, data = jsonb_set(data, '{status}', '4'), updated_at = now()
                      WHERE tenant_id = md5('tenant-2')::uuid AND number <= $n RETURNING id, version)"
  else
    write="WITH r AS (INSERT INTO records (tenant_id, id, entity_definition_id, number, version, data)
                      SELECT md5('tenant-2')::uuid, gen_random_uuid(), md5('def-2')::uuid, 2000000 + g, 1,
                             jsonb_build_object('title', 'bulk ' || g, 'status', g % 5)
                        FROM generate_series(1, $n) g RETURNING id, version)"
  fi
  cat <<SQL
\pset footer off
BEGIN;
SELECT set_config('app.tenant_id', md5('tenant-2')::uuid::text, true) \g /dev/null
SELECT pg_backend_pid() AS pid \gset
\setenv BPID :pid
\timing on
$write
INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_principal_id, source, request_id,
                       content_sha256, diff, diff_salt, diff_sha256)
SELECT md5('tenant-2')::uuid, gen_random_uuid(), 'record', CASE '$kind' WHEN 'insert' THEN 'create' ELSE '$kind' END, 'record', r.id, r.version, md5('user-1')::uuid, 'user', '$tag',
       '\x8a4f1c3e5b7d9f0a2c4e6a8b0d2f4a6c8e0a2c4e6a8b0d2f4a6c8e0a2c4e6a8b',
       '[{"op":"replace","path":"/status","value":4}]', '\x00112233445566778899aabbccddeeff',
       '\x1b3d5f7091b3d5f7091b3d5f7091b3d5f7091b3d5f7091b3d5f7091b3d5f7091'
  FROM r;
\timing off
SELECT '$tag' AS "case", coalesce(sum(total_bytes) FILTER (WHERE name = 'AfterTriggerEvents'), 0) AS after_trigger_events_bytes,
       sum(total_bytes) AS backend_total_bytes
  FROM pg_backend_memory_contexts;
\echo COMMIT time for $tag:
\timing on
COMMIT;
\timing off
\! grep -E 'VmHWM|RssAnon' /proc/\$BPID/status | tr -s ' \t' ' ' | sed 's/^/  $tag /'
SQL
}
# Spike only: lets nava_app read its own pg_backend_memory_contexts. Revoked at the end.
$DC exec -T pg psql -U postgres -d spike -qc "GRANT pg_read_all_stats TO nava_app"
fix() { # off: reset the n_distinct override (as pinned); on: apply it (schema.sql default)
  if [ "$1" = on ]; then v="SET (n_distinct = -0.0001)"; else v="RESET (n_distinct)"; fi
  $DC exec -T pg psql -U postgres -d spike -qc "ALTER TABLE audit_log ALTER COLUMN occurred_at $v; ALTER TABLE audit_log_default ALTER COLUMN occurred_at $v; ANALYZE audit_log_default"
}
# The O(n^2) case is only run at 10k; at 100k it would take about 100x as long.
SPECS=${ITEM2_SPECS:-"off update 10000|on update 10000|on insert 10000|on insert 100000"}
IFS="|" read -r -a SPECARR <<<"$SPECS"
for spec in "${SPECARR[@]}"; do
  set -- $spec
  f=$1; shift
  echo "#### n_distinct fix: $f"
  fix "$f"
  for arm in off on; do
    tag="$1-$2-fix-$f-trigger-$arm"
    if [ "$arm" = on ]; then act=ENABLE; else act=DISABLE; fi
    $DC exec -T pg psql -U postgres -d spike -qc "ALTER TABLE records $act TRIGGER records_audit_commit_check"
    $DC exec -T pg psql -U postgres -d spike -qc "CHECKPOINT"
    echo "== $tag"
    case_sql "$1" "$2" "$tag" | $DC exec -T pg psql -U nava_app -d spike -q -v ON_ERROR_STOP=1
  done
done
$DC exec -T pg psql -U postgres -d spike -qc "ALTER TABLE records ENABLE TRIGGER records_audit_commit_check"
$DC exec -T pg psql -U postgres -d spike -qc "REVOKE pg_read_all_stats FROM nava_app"
