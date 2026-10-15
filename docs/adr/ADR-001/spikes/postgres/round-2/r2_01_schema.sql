-- Round 2: RFC-001 revision 1 §4.3/§4.4 as written. Run after 01_roles_as_written.sql (roles + schema app).
\set ON_ERROR_STOP 1
CREATE ROLE nava_purge LOGIN PASSWORD 'p';
GRANT CONNECT ON DATABASE nava TO nava_purge;
GRANT USAGE ON SCHEMA app TO nava_purge;
-- §4.4 global settings
REVOKE ALL ON DATABASE nava FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE nava_migrator REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;

SET ROLE nava_migrator;
CREATE TABLE app.outbox3 (
  id uuid NOT NULL, tenant_id uuid NOT NULL, kind text NOT NULL, kind_version smallint NOT NULL DEFAULT 1,
  dedupe_key text NOT NULL, target_key text NOT NULL, payload jsonb NOT NULL DEFAULT '{}', trace_context text,
  created_at timestamptz NOT NULL DEFAULT now(), available_at timestamptz NOT NULL DEFAULT now(),
  attempts int NOT NULL DEFAULT 0, last_error text, dispatched_at timestamptz, dead_at timestamptz,
  PRIMARY KEY (tenant_id, id), UNIQUE (tenant_id, dedupe_key));
CREATE INDEX outbox3_claim ON app.outbox3 (tenant_id, available_at) WHERE dispatched_at IS NULL AND dead_at IS NULL;
CREATE INDEX outbox3_fifo  ON app.outbox3 (tenant_id, target_key, id) WHERE dispatched_at IS NULL AND dead_at IS NULL;
ALTER TABLE app.outbox3 ENABLE ROW LEVEL SECURITY; ALTER TABLE app.outbox3 FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON app.outbox3 TO nava_app, nava_worker USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY outbox_relay_select ON app.outbox3 FOR SELECT TO nava_relay USING (true);
CREATE POLICY outbox_relay_update ON app.outbox3 FOR UPDATE TO nava_relay USING (true);
-- purge exactly as §4.4: FOR DELETE policy only, column SELECT on key + predicate columns
CREATE POLICY outbox_purge ON app.outbox3 FOR DELETE TO nava_purge USING (dispatched_at < now() - interval '7 days' OR dead_at < now() - interval '30 days');
GRANT INSERT ON app.outbox3 TO nava_app, nava_worker;
GRANT SELECT (id, tenant_id, kind, kind_version, target_key, payload, trace_context, created_at, available_at, attempts, dispatched_at, dead_at) ON app.outbox3 TO nava_relay;
GRANT UPDATE (available_at, attempts, last_error, dispatched_at, dead_at) ON app.outbox3 TO nava_relay;
GRANT SELECT (tenant_id, id, dispatched_at, dead_at), DELETE ON app.outbox3 TO nava_purge;

-- workflow_ref as §4.3: (tenant_id, record_id, kind, workflow_id)
CREATE TABLE app.workflow_ref (
  tenant_id uuid NOT NULL, workflow_id text NOT NULL, record_id uuid NOT NULL, kind text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, workflow_id));
CREATE INDEX workflow_ref_record ON app.workflow_ref (tenant_id, record_id, kind, created_at DESC);
ALTER TABLE app.workflow_ref ENABLE ROW LEVEL SECURITY; ALTER TABLE app.workflow_ref FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON app.workflow_ref TO nava_app, nava_worker USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
GRANT SELECT, INSERT ON app.workflow_ref TO nava_app, nava_worker;
RESET ROLE;

-- ---------- synthetic data ----------
-- 200 tenants; 3,000,000 dispatched rows (2,000,000 inside 7 days, 1,000,000 older = purgeable)
CREATE TABLE t AS SELECT gen_random_uuid() AS id, row_number() over () AS n FROM generate_series(1,200);
INSERT INTO app.outbox3 (id, tenant_id, kind, dedupe_key, target_key, created_at, available_at, dispatched_at)
SELECT gen_random_uuid(), (SELECT id FROM t WHERE n = 1 + (g % 200)), 'workflow.signal', 'd'||g, 'rec:'||(g % 400000),
       now() - interval '10 days' + g * interval '250 ms', now() - interval '10 days' + g * interval '250 ms',
       now() - interval '10 days' + g * interval '250 ms' + interval '1 s'
FROM generate_series(1, 3000000) g;
-- pending, flood tenant #1: 45,000 rows on 1,000 targets (45 deep each)
INSERT INTO app.outbox3 (id, tenant_id, kind, dedupe_key, target_key, created_at, available_at)
SELECT gen_random_uuid(), (SELECT id FROM t WHERE n = 1), 'workflow.signal', 'p'||g, 'rec:flood'||(g % 1000),
       now() - interval '10 min' + g * interval '1 ms', now() - interval '10 min' + g * interval '1 ms'
FROM generate_series(1, 45000) g;
-- pending, other 199 tenants: 5,000 rows, one per target
INSERT INTO app.outbox3 (id, tenant_id, kind, dedupe_key, target_key, created_at, available_at)
SELECT gen_random_uuid(), (SELECT id FROM t WHERE n = 2 + (g % 199)), 'workflow.signal', 'q'||g, 'rec:q'||g,
       now() - interval '5 min' + g * interval '10 ms', now() - interval '5 min' + g * interval '10 ms'
FROM generate_series(1, 5000) g;
-- ids must sort in arrival order per target for the FIFO rule (stand-in for UUIDv7)
UPDATE app.outbox3 SET id = (lpad(to_hex((extract(epoch FROM created_at)*1000)::bigint), 12, '0')
       || substr(replace(id::text,'-',''), 13))::uuid WHERE dispatched_at IS NULL;
-- workflow_ref: 3,000,000 rows (one per workflow ever started)
INSERT INTO app.workflow_ref (tenant_id, workflow_id, record_id, kind, created_at)
SELECT (SELECT id FROM t WHERE n = 1 + (g % 200)), 't:x:approval:'||gen_random_uuid(), md5((g % 1500000)::text)::uuid, 'approval',
       now() - interval '10 days' + g * interval '250 ms'
FROM generate_series(1, 3000000) g;
VACUUM ANALYZE app.outbox3; VACUUM ANALYZE app.workflow_ref;
SELECT 'outbox3' AS rel, pg_size_pretty(pg_relation_size('app.outbox3')) heap,
       pg_size_pretty(pg_relation_size('app.outbox3_claim')) claim_idx, pg_size_pretty(pg_relation_size('app.outbox3_fifo')) fifo_idx,
       pg_size_pretty(pg_indexes_size('app.outbox3')) all_idx
UNION ALL
SELECT 'workflow_ref', pg_size_pretty(pg_relation_size('app.workflow_ref')), '-', '-', pg_size_pretty(pg_indexes_size('app.workflow_ref'));
-- spike helper only: lets the relay session name tenant #n
GRANT USAGE ON SCHEMA public TO nava_relay; GRANT SELECT ON t TO nava_relay;
