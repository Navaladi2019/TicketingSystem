-- P3-09 (TICK-353): records gains what the write pipeline needs, audit_log is
-- append-only and record_counters backs per-tenant numbering (PLAN-003 D-3-2,
-- D-3-9, D-3-13; ADR-001 I1, I8, I9; groomed on TICK-360 and TICK-363). Both
-- new tables are tenant tables under ENABLE + FORCE ROW LEVEL SECURITY, and
-- every key and foreign key leads with tenant_id, because PostgreSQL runs
-- foreign-key checks as the table owner and so ignores RLS (TICK-360).
--
-- Additive only, so a deployed records table is not rewritten on PG16: every
-- new column is nullable or has a constant or stable default (now() is
-- evaluated once and stored as the column's missing value). The integration
-- test pins this with pg_class.relfilenode. The all-or-none CHECK keeps the
-- PLAN-001 bare (tenant_id, id) rows valid; SET NOT NULL is a later contract
-- step, once no bare row remains.
--
-- Who may do what (db/grants.yaml):
--   nava_app, nava_worker  records: INSERT, SELECT, and UPDATE of every
--                          column but tenant_id, id, entity_definition_id,
--                          number, created_by and created_at, which are set
--                          once. audit_log: INSERT, SELECT; nobody updates
--                          or deletes an audit row. record_counters: INSERT,
--                          SELECT, UPDATE (next), the D-3-9 upsert.
--   nava_relay             still nothing on records (CI-9).
--
-- Index set: P3-01 (TICK-339, docs/data/records-list-bench.md), every one led
-- by tenant_id. No assignee index, no JSONB expression index and no GIN on
-- data: under RLS the planner may not push ->> or @> (not leakproof) below
-- the policy, so neither is usable. No JSONB operator is ever marked
-- LEAKPROOF (Security, TICK-339). Plain CREATE INDEX: records holds only
-- PLAN-001 test rows in M1; any later index on it uses CONCURRENTLY.
--
-- plan_cache_mode = force_custom_plan for nava_app is set by db-roles from
-- the role's settings in db/grants.yaml, not here: nava_migrator is
-- NOCREATEROLE and so may not ALTER ROLE another role.
--
-- No partitioning (D23). No policy on the A.3 allow-list. Design and
-- evidence: docs/data/platform-tables.md.

-- +goose Up

ALTER TABLE records
    ADD COLUMN entity_definition_id uuid,
    ADD COLUMN schema_version       integer,
    ADD COLUMN number               text,
    ADD COLUMN state                text,
    ADD COLUMN requester_id         uuid,
    ADD COLUMN assignee_id          uuid,
    ADD COLUMN assignment_group_id  uuid,
    ADD COLUMN created_by           uuid,
    -- Keyset paging on (updated_at, id) breaks on NULL (TICK-363).
    ADD COLUMN created_at           timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN updated_at           timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN data                 jsonb       NOT NULL DEFAULT '{}',
    ADD COLUMN causation_origin_id  uuid,
    ADD COLUMN causation_depth      smallint,
    ADD CONSTRAINT records_number_key UNIQUE (tenant_id, entity_definition_id, number),
    ADD CONSTRAINT records_entity_definition_fkey FOREIGN KEY (tenant_id, entity_definition_id)
        REFERENCES entity_definitions (tenant_id, id),
    ADD CONSTRAINT records_requester_fkey FOREIGN KEY (tenant_id, requester_id)
        REFERENCES principals (tenant_id, id),
    ADD CONSTRAINT records_assignee_fkey FOREIGN KEY (tenant_id, assignee_id)
        REFERENCES principals (tenant_id, id),
    ADD CONSTRAINT records_created_by_fkey FOREIGN KEY (tenant_id, created_by)
        REFERENCES principals (tenant_id, id),
    -- Group sync deletes a group that Keycloak no longer has
    -- (internal/groupsync). NO ACTION would then fail the tenant's whole
    -- sync, deprovisioning included, for as long as one record names the
    -- group, so the delete clears the record's group instead. The column
    -- list leaves tenant_id alone (PG15+).
    ADD CONSTRAINT records_assignment_group_fkey FOREIGN KEY (tenant_id, assignment_group_id)
        REFERENCES groups (tenant_id, id) ON DELETE SET NULL (assignment_group_id),
    -- No half-formed pipeline row while PLAN-001's bare rows remain (TICK-363).
    ADD CONSTRAINT records_pipeline_all_or_none_check CHECK (
        (entity_definition_id IS NULL) = (schema_version IS NULL)
        AND (entity_definition_id IS NULL) = (number IS NULL)
        AND (entity_definition_id IS NULL) = (state IS NULL)),
    ADD CONSTRAINT records_schema_version_check CHECK (schema_version >= 1),
    ADD CONSTRAINT records_causation_depth_check CHECK (causation_depth >= 0);

-- records_pkey (tenant_id, id) exists since T09.
CREATE INDEX records_tenant_updated ON records (tenant_id, updated_at DESC, id DESC);
CREATE INDEX records_tenant_requester ON records (tenant_id, requester_id, updated_at DESC, id DESC);
CREATE INDEX records_tenant_group_open ON records (tenant_id, assignment_group_id, updated_at DESC, id DESC)
    WHERE state <> 'closed';

-- One row per change to a record, and per pipeline event that changed none
-- (cascade_rejected on a create, cascade_cycle): record_id is NULL only when
-- there is no record to name. Append-only: no role has UPDATE or DELETE.
--
-- idempotency_key (D-3-13) is on exactly one row per write, the primary row;
-- any other row from the same write carries NULL, or a write would conflict
-- with itself (TICK-363). The replay lookup and ON CONFLICT use the partial
-- unique index, so an ON CONFLICT target must repeat its predicate:
-- ON CONFLICT (tenant_id, idempotency_key) WHERE idempotency_key IS NOT NULL.
--
-- actor_id has no foreign key: a system actor (rule, workflow, function, ai,
-- system) names no principal. An id with no foreign key reaches no other
-- tenant's row.
CREATE TABLE audit_log (
    tenant_id           uuid        NOT NULL,
    id                  uuid        NOT NULL, -- UUIDv7
    record_id           uuid,
    action              text        NOT NULL,
    actor_id            uuid,
    source              text        NOT NULL,
    before              jsonb,
    after               jsonb,
    causation_origin_id uuid,
    causation_depth     smallint,
    idempotency_key     text,
    at                  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT audit_log_pkey PRIMARY KEY (tenant_id, id),
    CONSTRAINT audit_log_tenant_fkey FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT audit_log_record_fkey FOREIGN KEY (tenant_id, record_id) REFERENCES records (tenant_id, id),
    CONSTRAINT audit_log_source_check
        CHECK (source IN ('user', 'rule', 'workflow', 'function', 'ai', 'system')),
    CONSTRAINT audit_log_causation_depth_check CHECK (causation_depth >= 0)
);
-- A record's history in order; also the index behind audit_log_record_fkey.
CREATE INDEX audit_log_record ON audit_log (tenant_id, record_id, at, id);
CREATE UNIQUE INDEX audit_log_idempotency_key ON audit_log (tenant_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log FORCE ROW LEVEL SECURITY;

-- No UPDATE or DELETE policy: there is nothing to allow.
CREATE POLICY audit_log_select ON audit_log FOR SELECT TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY audit_log_insert ON audit_log FOR INSERT TO nava_app, nava_worker
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- Per-tenant, per-type record numbers (D-3-9, I8: no sequences). One upsert
-- in the write transaction, as its last statement before the audit and
-- outbox inserts:
--   INSERT INTO record_counters (tenant_id, entity_definition_id, next)
--   VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id)
--   DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1
-- next is the number the following create gets. No onboarding or publish
-- step writes a row here.
CREATE TABLE record_counters (
    tenant_id            uuid   NOT NULL,
    entity_definition_id uuid   NOT NULL,
    next                 bigint NOT NULL,
    CONSTRAINT record_counters_pkey PRIMARY KEY (tenant_id, entity_definition_id),
    CONSTRAINT record_counters_entity_definition_fkey FOREIGN KEY (tenant_id, entity_definition_id)
        REFERENCES entity_definitions (tenant_id, id),
    CONSTRAINT record_counters_next_check CHECK (next >= 2)
);
ALTER TABLE record_counters ENABLE ROW LEVEL SECURITY;
ALTER TABLE record_counters FORCE ROW LEVEL SECURITY;

-- The upsert needs all three: INSERT, then on conflict UPDATE, and SELECT for
-- the conflict row and RETURNING. No DELETE.
CREATE POLICY record_counters_select ON record_counters FOR SELECT TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY record_counters_insert ON record_counters FOR INSERT TO nava_app, nava_worker
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY record_counters_update ON record_counters FOR UPDATE TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
