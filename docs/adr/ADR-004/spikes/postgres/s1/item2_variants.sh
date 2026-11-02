#!/usr/bin/env bash
# Item 2 follow-up: commit time of a 10k-subject transaction for each form of the
# commit-check function (same pinned predicate, different plan shape).
set -euo pipefail
export MSYS_NO_PATHCONV=1
cd "$(dirname "$0")"
DC="docker compose -f compose.yaml"
N=${N:-10000}
su() { $DC exec -T pg psql -U postgres -d spike -q -v ON_ERROR_STOP=1 "$@"; }
body() { # <variant>
  case $1 in
    v0-as-pinned) cat <<'SQL'
    PERFORM 1 FROM audit_log
     WHERE tenant_id = v_tenant AND occurred_at = now() AND txid = pg_current_xact_id()
       AND subject_type = TG_ARGV[0] AND subject_id = v_id
     LIMIT 1;
SQL
    ;;
    v1-locals) cat <<'SQL'
    v_type := TG_ARGV[0]; v_txid := pg_current_xact_id(); v_at := now();
    PERFORM 1 FROM audit_log
     WHERE tenant_id = v_tenant AND occurred_at = v_at AND txid = v_txid
       AND subject_type = v_type AND subject_id = v_id
     LIMIT 1;
SQL
    ;;
    v2-locals-version) cat <<'SQL'
    v_type := TG_ARGV[0]; v_txid := pg_current_xact_id(); v_at := now();
    PERFORM 1 FROM audit_log
     WHERE tenant_id = v_tenant AND occurred_at = v_at AND txid = v_txid
       AND subject_type = v_type AND subject_id = v_id AND version_after IS NOT DISTINCT FROM v_ver
     LIMIT 1;
SQL
    ;;
    v3-row-compare) cat <<'SQL'
    v_type := TG_ARGV[0]; v_txid := pg_current_xact_id(); v_at := now();
    PERFORM 1 FROM audit_log
     WHERE (tenant_id, occurred_at, txid, subject_type, subject_id) = (v_tenant, v_at, v_txid, v_type, v_id)
       AND tenant_id = v_tenant AND occurred_at = v_at
     LIMIT 1;
SQL
    ;;
    v4-order-by-r2-key) cat <<'SQL'
    v_type := TG_ARGV[0]; v_txid := pg_current_xact_id(); v_at := now();
    PERFORM 1 FROM audit_log
     WHERE tenant_id = v_tenant AND occurred_at = v_at AND txid = v_txid
       AND subject_type = v_type AND subject_id = v_id
     ORDER BY tenant_id, occurred_at, txid, subject_type, subject_id
     LIMIT 1;
SQL
    ;;
    v5-pinned-order-by) cat <<'SQL'
    PERFORM 1 FROM audit_log
     WHERE tenant_id = v_tenant AND occurred_at = now() AND txid = pg_current_xact_id()
       AND subject_type = TG_ARGV[0] AND subject_id = v_id
     ORDER BY tenant_id, occurred_at, txid, subject_type, subject_id
     LIMIT 1;
SQL
    ;;
  esac
}
install() { # <variant>
  su <<SQL
CREATE OR REPLACE FUNCTION audit_commit_check() RETURNS trigger LANGUAGE plpgsql SECURITY INVOKER AS \$fn\$
DECLARE
    v_tenant uuid; v_id uuid; v_type text; v_txid xid8; v_at timestamptz; v_ver bigint;
BEGIN
    IF TG_OP = 'DELETE' THEN v_tenant := OLD.tenant_id; v_id := OLD.id; v_ver := NULL;
    ELSE v_tenant := NEW.tenant_id; v_id := NEW.id; v_ver := NEW.version; END IF;
$(body "$1")
    IF NOT FOUND THEN
        RAISE EXCEPTION 'audit_missing: % % written without an audit entry in this transaction', TG_ARGV[0], v_id
            USING ERRCODE = '23514';
    END IF;
    RETURN NULL;
END \$fn\$;
SQL
}
run_txn() { # <variant> <tenant>
  $DC exec -T pg psql -U nava_app -d spike -q -v ON_ERROR_STOP=1 <<SQL
BEGIN;
SELECT set_config('app.tenant_id', md5('tenant-$2')::uuid::text, true) \g /dev/null
WITH r AS (UPDATE records SET version = version + 1, updated_at = now()
            WHERE tenant_id = md5('tenant-$2')::uuid AND number <= $N RETURNING id, version)
INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
SELECT md5('tenant-$2')::uuid, gen_random_uuid(), 'record', 'update', 'record', r.id, r.version, 'system', 'system' FROM r;
\echo commit, variant $1, $N subjects:
\timing on
COMMIT;
SQL
}
plan() { # <variant> <tenant>: plan of the probe after 6 calls (generic plan)
  $DC exec -T pg psql -U postgres -d spike -q <<SQL 2>&1 | sed -n '/Query Text: SELECT 1/,/^psql.*duration\|^$/p' | grep -E "Index Cond|Filter:|Scan|Rows Removed" | tail -4
LOAD 'auto_explain';
SET auto_explain.log_min_duration = 0; SET auto_explain.log_nested_statements = on; SET auto_explain.log_analyze = on;
SET auto_explain.log_level = notice; SET client_min_messages = notice;
BEGIN;
SET LOCAL ROLE nava_app;
SELECT set_config('app.tenant_id', md5('tenant-$2')::uuid::text, true) \g /dev/null
WITH r AS (UPDATE records SET version = version + 1 WHERE tenant_id = md5('tenant-$2')::uuid AND number <= 8 RETURNING id, version)
INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
SELECT md5('tenant-$2')::uuid, gen_random_uuid(), 'record', 'update', 'record', r.id, r.version, 'system', 'system' FROM r;
COMMIT;
SQL
}
t=5
for v in ${VARIANTS:-v0-as-pinned v1-locals v2-locals-version v3-row-compare}; do
  install "$v"
  echo "== $v: probe plan (8th call, cached generic plan)"
  plan "$v" $t | sed 's/^/   /'
  run_txn "$v" $t
  t=$((t + 1))
done
install v1-locals
