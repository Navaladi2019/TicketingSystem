-- S22 acceptance (TICK-754): the database behind criteria 1, 2 and 4, after
-- the Bruno folder. run-kit.sh checks each `check|...` line: the last column
-- must be `ok`.
--
--   AC4  no side row of any type belongs to a field with no registry row
--        (a removal that missed a row written during it leaves exactly that);
--        s22_code's re-built index equals the JSON of every s22_asset record.
--   AC1  s22_code has exactly one registry row, `ready`.
--   AC2  tenant A's tenant_limits row is 10, and its change has exactly one
--        audit entry (the replay S22.20 added none), naming the operator and
--        the reason. Tenant B has no row (its limit is still the default 5).
\set ON_ERROR_STOP on
\pset format unaligned
\pset tuples_only on
CREATE TEMP TABLE s22_f AS
SELECT f.* FROM record_index_fields f JOIN entity_definitions d ON d.tenant_id = f.tenant_id AND d.id = f.entity_definition_id
 WHERE f.tenant_id = '00000000-0000-4000-8000-0000000000a1' AND d.key = 's22_asset' AND f.field_key = 's22_code';

SELECT 'check|s22_code registry rows (1, ready)|' || count(*) || ' ' || coalesce(string_agg(state, ','), '-') || '|'
       || CASE WHEN count(*) = 1 AND bool_and(state = 'ready') THEN 'ok' ELSE 'FAIL' END FROM s22_f;

WITH orphans AS (
  SELECT 'rfv_text' AS t, count(*) AS n FROM rfv_text s WHERE NOT EXISTS (SELECT 1 FROM record_index_fields f WHERE f.tenant_id = s.tenant_id AND f.id = s.field_id)
  UNION ALL SELECT 'rfv_num', count(*) FROM rfv_num s WHERE NOT EXISTS (SELECT 1 FROM record_index_fields f WHERE f.tenant_id = s.tenant_id AND f.id = s.field_id)
  UNION ALL SELECT 'rfv_ts', count(*) FROM rfv_ts s WHERE NOT EXISTS (SELECT 1 FROM record_index_fields f WHERE f.tenant_id = s.tenant_id AND f.id = s.field_id)
  UNION ALL SELECT 'rfv_bool', count(*) FROM rfv_bool s WHERE NOT EXISTS (SELECT 1 FROM record_index_fields f WHERE f.tenant_id = s.tenant_id AND f.id = s.field_id))
SELECT 'check|orphan side rows (no registry row), every tenant and table|' || sum(n) || '|' || CASE WHEN sum(n) = 0 THEN 'ok' ELSE 'FAIL' END FROM orphans;

WITH want AS (
  SELECT r.id, r.data->>'s22_code' AS v FROM records r JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
   WHERE r.tenant_id = '00000000-0000-4000-8000-0000000000a1' AND d.key = 's22_asset' AND r.data ? 's22_code'),
got AS (SELECT s.record_id AS id, s.value AS v FROM rfv_text s JOIN s22_f f ON f.tenant_id = s.tenant_id AND f.id = s.field_id),
diff AS (SELECT (SELECT count(*) FROM want w WHERE NOT EXISTS (SELECT 1 FROM got g WHERE g.id = w.id AND g.v = w.v)) AS missing,
                (SELECT count(*) FROM got g WHERE NOT EXISTS (SELECT 1 FROM want w WHERE w.id = g.id AND w.v = g.v)) AS surplus,
                (SELECT count(*) FROM want) AS records)
SELECT 'check|s22_code side rows against the JSON (records missing surplus)|' || records || ' ' || missing || ' ' || surplus || '|'
       || CASE WHEN missing = 0 AND surplus = 0 AND records = 20003 THEN 'ok' ELSE 'FAIL' END FROM diff;

SELECT 'check|tenant A tenant_limits.filterable_fields_per_type|' || coalesce(max(filterable_fields_per_type)::text, 'none') || '|'
       || CASE WHEN max(filterable_fields_per_type) = 10 THEN 'ok' ELSE 'FAIL' END
  FROM tenant_limits WHERE tenant_id = '00000000-0000-4000-8000-0000000000a1';
SELECT 'check|tenant B tenant_limits rows|' || count(*) || '|' || CASE WHEN count(*) = 0 THEN 'ok' ELSE 'FAIL' END
  FROM tenant_limits WHERE tenant_id = '00000000-0000-4000-8000-0000000000b1';

-- The audit entry of the raise. Its exact domain/action names are TICK-916's;
-- the kit matches any entry whose subject is tenant A's limits row or whose
-- action names tenant_limits, and prints it so the evidence shows actor and reason.
SELECT 'check|tenant_limits audit entries for tenant A (exactly 1)|' || count(*) || '|' || CASE WHEN count(*) = 1 THEN 'ok' ELSE 'FAIL' END
  FROM audit_log a
 WHERE a.tenant_id = '00000000-0000-4000-8000-0000000000a1'
   AND (a.subject_type ILIKE '%limit%' OR a.action ILIKE '%limit%' OR a.domain ILIKE '%limit%');
SELECT 'audit|' || a.occurred_at || '|' || a.domain || '|' || a.action || '|' || a.subject_type || '|' || row_to_json(a)::text
  FROM audit_log a
 WHERE a.tenant_id = '00000000-0000-4000-8000-0000000000a1'
   AND (a.subject_type ILIKE '%limit%' OR a.action ILIKE '%limit%' OR a.domain ILIKE '%limit%');
