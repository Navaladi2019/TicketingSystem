-- TICK-248 claim-plan evidence. Run as a superuser on an empty database:
--   psql -f seed.sql. Only the outbox table and its pending indexes from
-- 00002_platform_tables.sql; RLS is not enabled, so the plans are the
-- superuser's (nava_relay's outbox_relay_* policies add no predicate on
-- pending rows).
DROP TABLE IF EXISTS outbox;
CREATE TABLE outbox (
    id              uuid        NOT NULL,
    tenant_id       uuid        NOT NULL,
    kind            text        NOT NULL,
    kind_version    smallint    NOT NULL DEFAULT 1,
    dedupe_key      text        NOT NULL,
    target_key      text        NOT NULL,
    target_seq      bigint      NOT NULL,
    causation_id    uuid,
    causation_depth smallint    NOT NULL DEFAULT 0,
    payload         jsonb       NOT NULL DEFAULT '{}',
    trace_context   text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    available_at    timestamptz NOT NULL DEFAULT now(),
    attempts        integer     NOT NULL DEFAULT 0,
    last_error      text,
    dispatched_at   timestamptz,
    dead_at         timestamptz,
    CONSTRAINT outbox_pkey PRIMARY KEY (tenant_id, id),
    CONSTRAINT outbox_dedupe_key UNIQUE (tenant_id, dedupe_key)
);
CREATE INDEX outbox_pending_available ON outbox (tenant_id, available_at)
    WHERE dispatched_at IS NULL AND dead_at IS NULL;
CREATE INDEX outbox_pending_target ON outbox (tenant_id, target_key, target_seq)
    WHERE dispatched_at IS NULL AND dead_at IS NULL;

-- Rows of one tenant: targets targets, depth rows each, consecutive in
-- available_at one microsecond apart from first (BenchmarkClaim's shape).
DROP FUNCTION IF EXISTS seed(uuid, int, int, timestamptz, text);
CREATE FUNCTION seed(tenant uuid, targets int, depth int, first timestamptz, tag text) RETURNS void
LANGUAGE sql AS $$
    INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq, available_at)
    SELECT gen_random_uuid(), tenant, 'ticket.approval', tag || '-' || t || '-' || s,
           'ticket.approval:' || tag || '-' || t, s,
           first + (((t - 1) * depth + s) * interval '1 microsecond')
    FROM generate_series(1, targets) t, generate_series(1, depth) s
$$;

-- S1, BenchmarkClaim's shape: one flood tenant, 1,000 targets 45 deep, and
-- 199 tenants with 25 one-row targets each (49,975 pending rows).
SELECT count(seed('00000000-0000-0000-0000-00000000000f', 1000, 45, now() - interval '1 hour', 'flood'));
SELECT count(seed(gen_random_uuid(), 25, 1, now() - interval '1 hour', 'small')) FROM generate_series(1, 199);

-- S2, T33's stall: tenant a, target X (head + 250 successors) then target Y.
-- X's head was claimed (attempts 1) and its lease ended with no settle, so
-- its available_at is later than its successors'.
SELECT count(seed('00000000-0000-0000-0000-00000000000a', 1, 251, now() - interval '1 hour', 'x'));
SELECT count(seed('00000000-0000-0000-0000-00000000000a', 1, 1, now() - interval '59 minutes', 'y'));
UPDATE outbox SET attempts = 1, available_at = now() - interval '1 second'
WHERE tenant_id = '00000000-0000-0000-0000-00000000000a' AND target_key = 'ticket.approval:x-1' AND target_seq = 1;

-- S3, the stranded step's worst case: tenant b, 200 targets 3 deep whose
-- heads are all still leased (an hour-long lease, so it outlasts the runs), so its full window is 400 blocked successors
-- over 200 targets and every probe finds a leased head.
SELECT count(seed('00000000-0000-0000-0000-00000000000b', 200, 3, now() - interval '1 hour', 'b'));
UPDATE outbox SET attempts = 1, available_at = now() + interval '1 hour'
WHERE tenant_id = '00000000-0000-0000-0000-00000000000b' AND target_seq = 1;

VACUUM ANALYZE outbox;
SELECT count(*) AS pending_rows, count(DISTINCT tenant_id) AS tenants FROM outbox;
