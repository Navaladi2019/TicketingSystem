-- TICK-1070 (S31; ADR-010-A2 A2-9, PLAN-011-A2 D-11A2-2): N-hat_e, the
-- estimated live records of one entity type in one tenant. The authz list
-- leaf reads it to set K* (A2-5, A2-5a); the hourly refresh job in
-- nava-worker (TICK-1071) writes it, one tenant at a time, under that
-- tenant's app.tenant_id. Platform DDL, no per-tenant DDL.
--
-- 1. One row per (tenant, entity type). live_records is count(*) of the
--    type's records at refreshed_at. A missing row means "no estimate": the
--    reader takes K* = 1,000 and runs the probe (A2-5a). A stale or wrong
--    value changes cost, never rows (A2-6).
--
-- 2. Keys. The primary key (tenant_id, entity_definition_id) is the only
--    index: the reader looks one row up by it, and the job's upsert arbitrates
--    on it. The FK carries tenant_id, so a row can only name an entity type
--    of its own tenant; no action, since no runtime role deletes
--    entity_definitions.
--
-- 3. FORCE RLS, every policy on the canonical app.tenant_id predicate, one
--    command each: SELECT for nava_app and nava_worker; INSERT and UPDATE for
--    nava_worker, the refresh job's role (D-11A2-2). No DELETE policy and no
--    DELETE grant: a type's row is overwritten, never removed.
--
-- 4. Derived data, not configuration: exempt from audit
--    (db/audited-tables.yaml), no config audit entry.
--
-- Locks: the table is new. The FK takes a SHARE ROW EXCLUSIVE lock on
-- entity_definitions for the ALTER, which blocks its writes (publishes) but
-- not reads; lock_timeout fails the migration instead of queueing behind a
-- long publish.

-- +goose Up

SET LOCAL lock_timeout = '5s';

CREATE TABLE entity_record_estimates (
    tenant_id            uuid        NOT NULL,
    entity_definition_id uuid        NOT NULL,
    live_records         bigint      NOT NULL,
    refreshed_at         timestamptz NOT NULL,
    CONSTRAINT entity_record_estimates_pkey PRIMARY KEY (tenant_id, entity_definition_id),
    CONSTRAINT entity_record_estimates_live_records_check CHECK (live_records >= 0),
    CONSTRAINT entity_record_estimates_definition_fkey FOREIGN KEY (tenant_id, entity_definition_id)
        REFERENCES entity_definitions (tenant_id, id)
);
ALTER TABLE entity_record_estimates ENABLE ROW LEVEL SECURITY;
ALTER TABLE entity_record_estimates FORCE ROW LEVEL SECURITY;

CREATE POLICY entity_record_estimates_select ON entity_record_estimates FOR SELECT TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY entity_record_estimates_insert ON entity_record_estimates FOR INSERT TO nava_worker
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY entity_record_estimates_update ON entity_record_estimates FOR UPDATE TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- +goose Down

DROP TABLE IF EXISTS entity_record_estimates;
