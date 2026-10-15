-- Proposed outbox shape + relay claim queries, benchmarked.
-- Volume: 200 tenants, 3,000,000 dispatched rows (retention window), 50,000 pending:
-- tenant #1 floods 45,000; the other 199 tenants hold 5,000 between them.
\set ON_ERROR_STOP 1
SET ROLE nava_migrator;
CREATE TABLE app.outbox2 (
  id            uuid        NOT NULL,                 -- UUIDv7 in prod; doubles as Temporal request id
  tenant_id     uuid        NOT NULL,
  kind          text        NOT NULL,                 -- e.g. 'workflow.start', 'workflow.signal'
  kind_version  smallint    NOT NULL DEFAULT 1,       -- message contract version (relay decodes by it)
  dedupe_key    text        NOT NULL,                 -- producer idempotency key (I7)
  target_key    text        NOT NULL,                 -- <kind>:<key> part of workflow id t:<tenant>:<kind>:<key>
  payload       jsonb       NOT NULL DEFAULT '{}',    -- references only (CI-7)
  trace_context text,                                 -- W3C traceparent
  created_at    timestamptz NOT NULL DEFAULT now(),
  available_at  timestamptz NOT NULL DEFAULT now(),   -- backoff: next attempt not before
  attempts      int         NOT NULL DEFAULT 0,
  last_error    text,
  dispatched_at timestamptz,
  dead_at       timestamptz,                          -- parked after max attempts
  PRIMARY KEY (tenant_id, id),
  UNIQUE (tenant_id, dedupe_key)
);
CREATE INDEX outbox2_pending ON app.outbox2 (tenant_id, available_at) WHERE dispatched_at IS NULL AND dead_at IS NULL;
ALTER TABLE app.outbox2 ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.outbox2 FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON app.outbox2 TO nava_app, nava_worker
  USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY relay_select ON app.outbox2 FOR SELECT TO nava_relay USING (true);
CREATE POLICY relay_update ON app.outbox2 FOR UPDATE TO nava_relay USING (true) WITH CHECK (true);
CREATE POLICY relay_purge  ON app.outbox2 FOR DELETE TO nava_relay USING (dispatched_at < now() - interval '7 days' OR dead_at < now() - interval '30 days');
GRANT INSERT ON app.outbox2 TO nava_app, nava_worker;
GRANT SELECT (id, tenant_id, kind, kind_version, target_key, payload, trace_context, created_at, available_at, attempts, dispatched_at, dead_at) ON app.outbox2 TO nava_relay;
GRANT UPDATE (available_at, attempts, last_error, dispatched_at, dead_at) ON app.outbox2 TO nava_relay;
GRANT DELETE ON app.outbox2 TO nava_relay;
RESET ROLE;

-- synthetic data (superuser load)
CREATE TEMP TABLE t AS SELECT gen_random_uuid() AS id, row_number() over () AS n FROM generate_series(1,200);
INSERT INTO app.outbox2 (id, tenant_id, kind, dedupe_key, target_key, payload, created_at, available_at, dispatched_at)
SELECT gen_random_uuid(), (SELECT id FROM t WHERE n = 1 + (g % 200)), 'workflow.start', 'd'||g, 'approval:'||g,
       jsonb_build_object('record_id', gen_random_uuid()), now() - interval '6 days' + g * interval '100 ms',
       now() - interval '6 days' + g * interval '100 ms', now() - interval '6 days' + g * interval '100 ms' + interval '1 s'
FROM generate_series(1, 3000000) g;
INSERT INTO app.outbox2 (id, tenant_id, kind, dedupe_key, target_key, payload, created_at, available_at)
SELECT gen_random_uuid(), (SELECT id FROM t WHERE n = 1), 'workflow.start', 'p'||g, 'approval:p'||g, '{}', now() - interval '10 min' + g * interval '1 ms', now() - interval '10 min' + g * interval '1 ms'
FROM generate_series(1, 45000) g;
INSERT INTO app.outbox2 (id, tenant_id, kind, dedupe_key, target_key, payload, created_at, available_at)
SELECT gen_random_uuid(), (SELECT id FROM t WHERE n = 2 + (g % 199)), 'workflow.start', 'q'||g, 'approval:q'||g, '{}', now() - interval '5 min' + g * interval '10 ms', now() - interval '5 min' + g * interval '10 ms'
FROM generate_series(1, 5000) g;
VACUUM ANALYZE app.outbox2;
SELECT pg_size_pretty(pg_relation_size('app.outbox2')) AS heap, pg_size_pretty(pg_relation_size('app.outbox2_pending')) AS pending_idx,
       pg_size_pretty(pg_indexes_size('app.outbox2')) AS all_idx;
