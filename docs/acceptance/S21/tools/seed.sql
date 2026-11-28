-- S21 acceptance (TICK-749): 400,000 incidents in tenant A and 50 in tenant B,
-- written in one statement per tenant as the bootstrap superuser, as the
-- P7-07 measurement seeds its 400k tenant (internal/server/
-- records_fieldindex_perf_integration_test.go, perfSeedSQLWith). The API would
-- take about 35 minutes at the measured 190 creates/s; this takes about one.
--
-- Run by run-kit.sh after kit.mjs setup, so tenant A's incident type is at the
-- version that has `due_at` (and the mask on `description`); every record is
-- written at that schema version. No field is flagged yet: the trigger has
-- nothing to copy, and session_replication_role = replica skips it, the
-- audit commit check and the foreign keys for the bulk load (every id below
-- is valid anyway). Nothing here is random: each value is a hash of the
-- record's ordinal n, so expect.sql and every rerun see the same rows.
--
--   priority  (hash 'p' || n) % 100: 0 -> 1 (1%), 1 -> 2 (1%), 2..49 -> 3 (48%), 50..99 -> 4 (50%)
--   due_at    2026-01-01T00:00:00Z + ((hash 'd' || n) % 525600) minutes: whole minutes over 2026
--   assignment_group  service-desk for (hash 'g' || n) % 100 < 5 (5%), else none
--                     (Alice, agent@tenant-a, is in service-desk: she sees about 5%)
--   requester / created_by  admin@tenant-a (never Alice)
--
-- Two marked records, for criterion 5 (seconds :59 never come from the hash):
--   n = 400000  due_at 2026-12-31T23:59:59Z, no group: tenant A has it, Alice cannot see it
--   n = 399999  due_at 2026-12-31T23:57:59Z, service-desk: Alice can see it (the control)
-- The value 2026-12-31T23:58:59Z exists nowhere.
--
-- Tenant B: 50 incidents, all priority 1, created by kit.mjs setup through the API.
\set ON_ERROR_STOP on
\timing on
SET max_parallel_workers_per_gather = 0;
SET session_replication_role = replica;

CREATE TEMP TABLE s21_ctx AS
SELECT d.tenant_id, d.id AS def, d.current_version AS ver,
       (SELECT id FROM principals WHERE tenant_id = d.tenant_id AND email = 'admin@tenant-a.test') AS admin_id,
       (SELECT id FROM groups WHERE tenant_id = d.tenant_id AND path = '/t/' || d.tenant_id || '/service-desk' AND deleted_at IS NULL) AS sd
  FROM entity_definitions d
 WHERE d.tenant_id = '00000000-0000-4000-8000-0000000000a1' AND d.key = 'incident';

DO $$
BEGIN
  IF (SELECT count(*) FROM s21_ctx WHERE admin_id IS NOT NULL AND sd IS NOT NULL) <> 1 THEN
    RAISE EXCEPTION 's21 seed: tenant A needs its incident type, admin@tenant-a and the service-desk group (run tools/seed.sh)';
  END IF;
  IF EXISTS (SELECT 1 FROM records r JOIN s21_ctx c ON r.tenant_id = c.tenant_id AND r.entity_definition_id = c.def) THEN
    RAISE EXCEPTION 's21 seed: tenant A already has incidents; the kit seeds a fresh stack only';
  END IF;
END $$;

INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state,
                     requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_depth)
SELECT c.tenant_id,
       md5('s21a/' || x.n)::uuid,
       1, c.def, c.ver, 'INC' || lpad(x.n::text, 7, '0'),
       (ARRAY['new', 'in_progress', 'resolved', 'closed'])[1 + x.hs % 4],
       c.admin_id, NULL,
       CASE WHEN x.n = 399999 OR (x.n <> 400000 AND x.hg % 100 < 5) THEN c.sd END,
       c.admin_id,
       x.upd - interval '1 day', x.upd,
       jsonb_build_object(
         'short_description', 'S21 seed ' || x.n,
         'description', 'S21 internal note ' || x.n,
         'priority', CASE WHEN x.hp % 100 = 0 THEN 1 WHEN x.hp % 100 = 1 THEN 2 WHEN x.hp % 100 < 50 THEN 3 ELSE 4 END,
         'due_at', CASE x.n
                     WHEN 400000 THEN '2026-12-31T23:59:59Z'
                     WHEN 399999 THEN '2026-12-31T23:57:59Z'
                     ELSE to_char((timestamptz '2026-01-01 00:00:00+00' + (x.hd % 525600) * interval '1 minute') AT TIME ZONE 'UTC',
                                  'YYYY-MM-DD"T"HH24:MI:SS"Z"')
                   END),
       0
  FROM s21_ctx c,
       LATERAL (
         SELECT n,
                hashtextextended('p' || n, 0) & 2147483647 AS hp,
                hashtextextended('d' || n, 0) & 2147483647 AS hd,
                hashtextextended('g' || n, 0) & 2147483647 AS hg,
                hashtextextended('s' || n, 0) & 2147483647 AS hs,
                timestamptz '2026-11-01 00:00:00+00' - n * interval '1 minute' AS upd
           FROM generate_series(1, 400000) n
       ) x;

INSERT INTO record_counters (tenant_id, entity_definition_id, next)
SELECT tenant_id, def, 400001 FROM s21_ctx
ON CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next = GREATEST(record_counters.next, EXCLUDED.next);

-- Tenant B's 50 incidents (all priority 1) are created by kit.mjs setup
-- through the API, as Bob: since S22 (TICK-752) the seed template flags
-- priority in tenant B too, and a bulk write here would skip its index.

SET session_replication_role = origin;
RESET max_parallel_workers_per_gather;
VACUUM (ANALYZE) records;

SELECT r.tenant_id, count(*) AS incidents,
       count(*) FILTER (WHERE r.data->>'priority' = '1') AS priority_1,
       count(*) FILTER (WHERE r.assignment_group_id IS NOT NULL) AS service_desk
  FROM records r JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id AND d.key = 'incident'
 GROUP BY 1 ORDER BY 1;
