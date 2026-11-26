-- S29 acceptance (TICK-754): the database behind criteria 2, 3 and 6, after
-- the Bruno folder. run-kit.sh checks each `check|...` line: the last column
-- must be `ok`.
\set ON_ERROR_STOP on
\pset format unaligned
\pset tuples_only on
CREATE TEMP TABLE s29_a AS
SELECT r.* FROM records r JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
 WHERE r.tenant_id = '00000000-0000-4000-8000-0000000000a1' AND d.key = 'incident';

-- AC2: after both jobs, only the poison record lacks priority and s29_tier.
SELECT 'check|tenant A incidents with no stored priority (only the poison record)|' || count(*) || '|'
       || CASE WHEN count(*) = 1 AND bool_and(id = md5('s29a/2000')::uuid) THEN 'ok' ELSE 'FAIL' END
  FROM s29_a WHERE NOT data ? 'priority';
SELECT 'check|tenant A incidents with no stored s29_tier (only the poison record)|' || count(*) || '|'
       || CASE WHEN count(*) = 1 THEN 'ok' ELSE 'FAIL' END
  FROM s29_a WHERE NOT data ? 's29_tier';

-- AC2: the rewrite keeps updated_at (D5): no seeded record's updated_at moved.
SELECT 'check|seeded records whose updated_at moved|' || count(*) || '|' || CASE WHEN count(*) = 0 THEN 'ok' ELSE 'FAIL' END
  FROM s29_a a, generate_series(1, 2000) n
 WHERE a.id = md5('s29a/' || n)::uuid AND a.updated_at <> timestamptz '2026-11-01 00:00:00+00' - n * interval '1 minute';

-- AC2: exactly one schema-backfill audit entry per rewrite: each seeded,
-- non-poison record was rewritten twice (priority, then s29_tier).
WITH per AS (
  SELECT a.id, (SELECT count(*) FROM audit_log l WHERE l.tenant_id = a.tenant_id AND l.subject_id = a.id AND l.actor_system = 'schema-backfill') AS n
    FROM s29_a a, generate_series(1, 1999) g WHERE a.id = md5('s29a/' || g)::uuid)
SELECT 'check|seeded records with exactly 2 schema-backfill audit entries (of 1999)|' || count(*) FILTER (WHERE n = 2) || '|'
       || CASE WHEN count(*) FILTER (WHERE n = 2) = 1999 THEN 'ok' ELSE 'FAIL' END FROM per;
SELECT 'check|schema-backfill audit entries on the poison record|' || count(*) || '|' || CASE WHEN count(*) = 0 THEN 'ok' ELSE 'FAIL' END
  FROM audit_log WHERE tenant_id = '00000000-0000-4000-8000-0000000000a1' AND subject_id = md5('s29a/2000')::uuid AND actor_system = 'schema-backfill';

-- AC3: the failures hold an id and a code, never a value.
SELECT 'check|tenant A schema_backfill_failures rows, all the poison record|' || count(*) || ' ' || coalesce(string_agg(DISTINCT code, ','), '-') || '|'
       || CASE WHEN count(*) >= 1 AND bool_and(record_id = md5('s29a/2000')::uuid) THEN 'ok' ELSE 'FAIL' END
  FROM schema_backfill_failures WHERE tenant_id = '00000000-0000-4000-8000-0000000000a1';

-- AC6: tenant B has no job, no queue and no failure row; its incidents are unchanged.
SELECT 'check|tenant B schema_backfills / pending / failures rows|'
       || (SELECT count(*) FROM schema_backfills WHERE tenant_id = '00000000-0000-4000-8000-0000000000b1') || ' '
       || (SELECT count(*) FROM schema_backfill_pending WHERE tenant_id = '00000000-0000-4000-8000-0000000000b1') || ' '
       || (SELECT count(*) FROM schema_backfill_failures WHERE tenant_id = '00000000-0000-4000-8000-0000000000b1') || '|'
       || CASE WHEN (SELECT count(*) FROM schema_backfills WHERE tenant_id = '00000000-0000-4000-8000-0000000000b1')
              + (SELECT count(*) FROM schema_backfill_pending WHERE tenant_id = '00000000-0000-4000-8000-0000000000b1')
              + (SELECT count(*) FROM schema_backfill_failures WHERE tenant_id = '00000000-0000-4000-8000-0000000000b1') = 0 THEN 'ok' ELSE 'FAIL' END;
SELECT 'check|tenant B incidents changed by schema-backfill|' || count(*) || '|' || CASE WHEN count(*) = 0 THEN 'ok' ELSE 'FAIL' END
  FROM audit_log WHERE tenant_id = '00000000-0000-4000-8000-0000000000b1' AND actor_system = 'schema-backfill';

-- The jobs, for the evidence.
SELECT 'job|' || row_to_json(j)::text FROM schema_backfills j WHERE tenant_id = '00000000-0000-4000-8000-0000000000a1';
