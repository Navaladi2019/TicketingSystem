-- P10-01 (TICK-916, ADR-009 D2, D7, grants): the three tables of the schema
-- backfill. schema_backfills holds one row per job, schema_backfill_pending is
-- the job's queue of record ids still to rewrite, and schema_backfill_failures
-- holds the ids that could not be rewritten, with an error code and never a
-- value (D7). records.data stays the only source of truth: these tables are
-- job state.
--
-- Keys: tenant_id leads every key, index and foreign key. The job row's key
-- (tenant_id, entity_definition_id, target_version) is the key the queue and
-- the failures share, and both cascade from the job: a superseded job drops
-- its queue and its failures with one DELETE (D8). The (tenant_id, record_id)
-- index on queue and failures backs D12's join from records. record_id has no
-- foreign key to records: a record delete is never blocked or slowed by a
-- queued id, and OpMaterialize treats an id whose record is gone as nothing to
-- write.
--
-- One active job per tenant (D7, Solution Architect's CHANGES on TICK-914):
-- schema_backfills_one_active_per_tenant admits one filling or running job per
-- tenant. The job that loses the race to flip a row to filling gets 23505 and
-- stays pending (P10-06).
--
-- Policies: one per command and per role, never FOR ALL, each the canonical
-- tenant predicate. No new role, no BYPASSRLS, no RLS allow-list entry.
--
-- Who may do what (db/grants.yaml, ADR-009 Grants):
--   nava_worker  SELECT, INSERT, UPDATE, DELETE on all three (the job).
--   nava_app     SELECT on all three (D10 status and failures, D12 filter
--                shapes) and INSERT of a pending schema_backfills row only,
--                the publish transaction's job (D1, D-10-11). No UPDATE or
--                DELETE anywhere, nothing else on the queue or failures.
--
-- Audit (db/audited-tables.yaml): all three are exempt, job state. Every
-- record rewrite is audited by the pipeline (D4).
--
-- New, empty tables: only the foreign key to entity_definitions takes a lock
-- on an existing table (SHARE ROW EXCLUSIVE, milliseconds). lock_timeout makes
-- the migration fail and be re-run rather than queue writes behind it.

-- +goose Up

SET LOCAL lock_timeout = '5s';

CREATE TABLE schema_backfills (
    tenant_id            uuid        NOT NULL,
    entity_definition_id uuid        NOT NULL,
    target_version       integer     NOT NULL,
    state                text        NOT NULL DEFAULT 'pending',
    default_keys         text[]      NOT NULL,
    causation_audit_id   uuid,
    rewritten            bigint      NOT NULL DEFAULT 0,
    skipped              bigint      NOT NULL DEFAULT 0,
    failed               bigint      NOT NULL DEFAULT 0,
    created_at           timestamptz NOT NULL DEFAULT now(),
    started_at           timestamptz,
    finished_at          timestamptz,
    CONSTRAINT schema_backfills_pkey PRIMARY KEY (tenant_id, entity_definition_id, target_version),
    CONSTRAINT schema_backfills_entity_definition_fkey FOREIGN KEY (tenant_id, entity_definition_id)
        REFERENCES entity_definitions (tenant_id, id),
    CONSTRAINT schema_backfills_state_check
        CHECK (state IN ('pending', 'filling', 'running', 'done', 'failed_records', 'superseded')),
    CONSTRAINT schema_backfills_target_version_check CHECK (target_version >= 1),
    CONSTRAINT schema_backfills_counts_check CHECK (rewritten >= 0 AND skipped >= 0 AND failed >= 0)
);
CREATE UNIQUE INDEX schema_backfills_one_active_per_tenant ON schema_backfills (tenant_id)
    WHERE state IN ('filling', 'running');
ALTER TABLE schema_backfills ENABLE ROW LEVEL SECURITY;
ALTER TABLE schema_backfills FORCE ROW LEVEL SECURITY;
CREATE POLICY schema_backfills_select_app ON schema_backfills FOR SELECT TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY schema_backfills_select_worker ON schema_backfills FOR SELECT TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
-- D-10-11 (PLAN-010, TICK-916): publish queues the job in its own
-- transaction as nava_app (ADR-009 D1), and only a fresh one: pending, no
-- counts, never started or finished. This is nava_app's only INSERT policy on
-- the table, so the conjunction is the ruling's tenant policy plus its
-- restrictive pending check, written as one pinned policy
-- (catalog.PinnedTenantPolicies): the catalogue gate admits a policy that is
-- the tenant predicate AND more only when it is pinned, and the A.3
-- allow-list stays at 11.
CREATE POLICY schema_backfills_insert_app_pending ON schema_backfills FOR INSERT TO nava_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
                AND state = 'pending' AND rewritten = 0 AND skipped = 0 AND failed = 0
                AND started_at IS NULL AND finished_at IS NULL);
CREATE POLICY schema_backfills_insert_worker ON schema_backfills FOR INSERT TO nava_worker
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY schema_backfills_update_worker ON schema_backfills FOR UPDATE TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY schema_backfills_delete_worker ON schema_backfills FOR DELETE TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

CREATE TABLE schema_backfill_pending (
    tenant_id            uuid    NOT NULL,
    entity_definition_id uuid    NOT NULL,
    target_version       integer NOT NULL,
    record_id            uuid    NOT NULL,
    CONSTRAINT schema_backfill_pending_pkey PRIMARY KEY (tenant_id, entity_definition_id, target_version, record_id),
    CONSTRAINT schema_backfill_pending_entity_definition_fkey FOREIGN KEY (tenant_id, entity_definition_id)
        REFERENCES entity_definitions (tenant_id, id),
    CONSTRAINT schema_backfill_pending_job_fkey FOREIGN KEY (tenant_id, entity_definition_id, target_version)
        REFERENCES schema_backfills (tenant_id, entity_definition_id, target_version) ON DELETE CASCADE
);
CREATE INDEX schema_backfill_pending_record ON schema_backfill_pending (tenant_id, record_id);
ALTER TABLE schema_backfill_pending ENABLE ROW LEVEL SECURITY;
ALTER TABLE schema_backfill_pending FORCE ROW LEVEL SECURITY;
CREATE POLICY schema_backfill_pending_select_app ON schema_backfill_pending FOR SELECT TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY schema_backfill_pending_select_worker ON schema_backfill_pending FOR SELECT TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY schema_backfill_pending_insert_worker ON schema_backfill_pending FOR INSERT TO nava_worker
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY schema_backfill_pending_update_worker ON schema_backfill_pending FOR UPDATE TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY schema_backfill_pending_delete_worker ON schema_backfill_pending FOR DELETE TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- No value column, ever (D7): an id and a stable lower-snake code.
CREATE TABLE schema_backfill_failures (
    tenant_id            uuid        NOT NULL,
    entity_definition_id uuid        NOT NULL,
    target_version       integer     NOT NULL,
    record_id            uuid        NOT NULL,
    code                 text        NOT NULL,
    failed_at            timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT schema_backfill_failures_pkey PRIMARY KEY (tenant_id, entity_definition_id, target_version, record_id),
    CONSTRAINT schema_backfill_failures_entity_definition_fkey FOREIGN KEY (tenant_id, entity_definition_id)
        REFERENCES entity_definitions (tenant_id, id),
    CONSTRAINT schema_backfill_failures_job_fkey FOREIGN KEY (tenant_id, entity_definition_id, target_version)
        REFERENCES schema_backfills (tenant_id, entity_definition_id, target_version) ON DELETE CASCADE,
    CONSTRAINT schema_backfill_failures_code_check CHECK (code ~ '^[a-z_]{1,64}$')
);
CREATE INDEX schema_backfill_failures_record ON schema_backfill_failures (tenant_id, record_id);
ALTER TABLE schema_backfill_failures ENABLE ROW LEVEL SECURITY;
ALTER TABLE schema_backfill_failures FORCE ROW LEVEL SECURITY;
CREATE POLICY schema_backfill_failures_select_app ON schema_backfill_failures FOR SELECT TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY schema_backfill_failures_select_worker ON schema_backfill_failures FOR SELECT TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY schema_backfill_failures_insert_worker ON schema_backfill_failures FOR INSERT TO nava_worker
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY schema_backfill_failures_update_worker ON schema_backfill_failures FOR UPDATE TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY schema_backfill_failures_delete_worker ON schema_backfill_failures FOR DELETE TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- +goose Down

SET LOCAL lock_timeout = '5s';

DROP TABLE schema_backfill_failures;
DROP TABLE schema_backfill_pending;
DROP TABLE schema_backfills;
