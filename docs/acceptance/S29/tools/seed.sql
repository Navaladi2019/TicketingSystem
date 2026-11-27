-- S29 acceptance (TICK-754, formerly TICK-928): 2,000 tenant A incidents
-- with no `s29_level`, written in one statement as the bootstrap superuser, as
-- S21's seed does. Run by run-kit.sh after kit.mjs setup, so the incident type
-- is at the version with a flagged `s29_level` with no default and an optional `s29_tier`;
-- every record is written at that version and holds neither key. No stored
-- value is a flagged field's (s29_level is absent), so the field-index trigger
-- has nothing to copy, and session_replication_role = replica skips it, the
-- audit commit check and the foreign keys for the bulk load.
--
--   id                 md5('s29a/' || n)::uuid
--   number             INC9000001..INC9002000 (clear of the counter's own numbers)
--   assignment_group   service-desk (Alice, agent@tenant-a, reads every one)
--   n = 2000           the poison record: its short_description is 200
--                      characters, over the schema's maxLength of 160, so the
--                      backfill's validation refuses it (criterion 3)
--
-- The 50 records that already hold an s29_level are created through the API
-- afterwards (kit.mjs held), so they have their create history.
\set ON_ERROR_STOP on
\timing on
SET session_replication_role = replica;

CREATE TEMP TABLE s29_ctx AS
SELECT d.tenant_id, d.id AS def, d.current_version AS ver,
       (SELECT id FROM principals WHERE tenant_id = d.tenant_id AND email = 'admin@tenant-a.test') AS admin_id,
       (SELECT id FROM groups WHERE tenant_id = d.tenant_id AND path = '/t/' || d.tenant_id || '/service-desk' AND deleted_at IS NULL) AS sd
  FROM entity_definitions d
 WHERE d.tenant_id = '00000000-0000-4000-8000-0000000000a1' AND d.key = 'incident';

DO $$
BEGIN
  IF (SELECT count(*) FROM s29_ctx WHERE admin_id IS NOT NULL AND sd IS NOT NULL) <> 1 THEN
    RAISE EXCEPTION 's29 seed: tenant A needs its incident type, admin@tenant-a and the service-desk group (run tools/seed.sh)';
  END IF;
  -- Other incidents may exist (a CI stack run's earlier cases); expect.sql
  -- counts whatever is there. Only the kit's own ids must be new.
  IF EXISTS (SELECT 1 FROM records r JOIN s29_ctx c ON r.tenant_id = c.tenant_id AND r.id = md5('s29a/1')::uuid) THEN
    RAISE EXCEPTION 's29 seed: the kit already ran on this stack; it runs once per stack';
  END IF;
END $$;

INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state,
                     requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_depth)
SELECT c.tenant_id, md5('s29a/' || n)::uuid, 1, c.def, c.ver, 'INC9' || lpad(n::text, 6, '0'), 'new',
       c.admin_id, NULL, c.sd, c.admin_id,
       timestamptz '2026-11-01 00:00:00+00' - n * interval '1 minute' - interval '1 day',
       timestamptz '2026-11-01 00:00:00+00' - n * interval '1 minute',
       jsonb_build_object('short_description',
                          CASE WHEN n = 2000 THEN 'S29 poison ' || repeat('x', 189) ELSE 'S29 seed ' || n END,
                          'state', 'new'),
       0
  FROM s29_ctx c, generate_series(1, 2000) n;


SET session_replication_role = origin;
ANALYZE records;

SELECT count(*) AS incidents, count(*) FILTER (WHERE length(r.data->>'short_description') > 160) AS poison
  FROM records r JOIN s29_ctx c ON r.tenant_id = c.tenant_id AND r.entity_definition_id = c.def;
