-- S22 acceptance (TICK-754): 20,000 records of tenant A's type `s22_asset`,
-- written in one statement as the bootstrap superuser, as S21's seed does.
-- Run by run-kit.sh after kit.mjs setup created the type; `s22_code` is not
-- flagged yet, so there is nothing for the field-index trigger to copy, and
-- session_replication_role = replica skips it, the audit commit check and the
-- foreign keys for the bulk load. Nothing is random:
--
--   id         md5('s22/' || n)::uuid   (Bruno computes the same: lib/kit.js md5uuid)
--   s22_code   'K' || lpad(n % 20, 2, '0'): K00..K19, exactly 1,000 records each
--   version    1
--
-- The removal in Bruno S22.10 then has 20,000 side rows to delete in batches
-- of 1,000, long enough that S22.11 and S22.12 run while it is in progress.
\set ON_ERROR_STOP on
\timing on
SET session_replication_role = replica;

CREATE TEMP TABLE s22_ctx AS
SELECT d.tenant_id, d.id AS def, d.current_version AS ver,
       (SELECT id FROM principals WHERE tenant_id = d.tenant_id AND email = 'admin@tenant-a.test') AS admin_id
  FROM entity_definitions d
 WHERE d.tenant_id = '00000000-0000-4000-8000-0000000000a1' AND d.key = 's22_asset';

DO $$
BEGIN
  IF (SELECT count(*) FROM s22_ctx WHERE admin_id IS NOT NULL) <> 1 THEN
    RAISE EXCEPTION 's22 seed: tenant A needs the s22_asset type (kit.mjs setup) and admin@tenant-a (tools/seed.sh)';
  END IF;
  IF EXISTS (SELECT 1 FROM records r JOIN s22_ctx c ON r.tenant_id = c.tenant_id AND r.entity_definition_id = c.def) THEN
    RAISE EXCEPTION 's22 seed: s22_asset already has records; the kit seeds a fresh stack only';
  END IF;
END $$;

INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state,
                     requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_depth)
SELECT c.tenant_id, md5('s22/' || n)::uuid, 1, c.def, c.ver, 'SAST' || lpad(n::text, 7, '0'), 'new',
       c.admin_id, NULL, NULL, c.admin_id,
       timestamptz '2026-11-01 00:00:00+00' - n * interval '1 minute' - interval '1 day',
       timestamptz '2026-11-01 00:00:00+00' - n * interval '1 minute',
       jsonb_build_object('name', 'S22 seed ' || n, 'state', 'new', 's22_code', 'K' || lpad((n % 20)::text, 2, '0')),
       0
  FROM s22_ctx c, generate_series(1, 20000) n;

INSERT INTO record_counters (tenant_id, entity_definition_id, next)
SELECT tenant_id, def, 20001 FROM s22_ctx
ON CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next = GREATEST(record_counters.next, EXCLUDED.next);

SET session_replication_role = origin;
ANALYZE records;

SELECT r.data->>'s22_code' AS code, count(*)
  FROM records r JOIN s22_ctx c ON r.tenant_id = c.tenant_id AND r.entity_definition_id = c.def
 WHERE r.data->>'s22_code' IN ('K07', 'K08')
 GROUP BY 1 ORDER BY 1;
