-- P3-40 (TICK-709): workflow definitions, their published immutable versions,
-- and workflow_cause (PLAN-003 D-3-12, D-3-13, D-3-15; S11b contract D-3-29).
-- Three new tenant tables: tenant_id leads every key, index and foreign key
-- (PostgreSQL runs foreign-key checks as the table owner and so ignores RLS,
-- D-3-2), RLS is ENABLEd and FORCEd, every policy is FOR one command, TO named
-- roles, and tests exactly the canonical tenant predicate.
--
-- Who may do what (db/grants.yaml):
--   nava_app     reads both definition tables, creates a definition, moves its
--                current_version / updated_at, and publishes versions (P3-41).
--   nava_worker  reads both definition tables (DefinitionStore.Load, the
--                sink's trigger lookup when a workflow write fires it).
--   both         read and insert workflow_cause.
--   nobody       UPDATEs or DELETEs a version, or a workflow_cause row: no
--                grant and no policy. No DELETE on any of the three.
--
-- Audit (db/audited-tables.yaml): both definition tables are audited under
-- the subject type workflow_definition (registered by the next migration),
-- not attached yet: nothing writes them until P3-41's publish route, which
-- writes its entry in the same transaction and attaches audit_commit_check
-- (the record_shares precedent). workflow_cause is exempt: a copy of the
-- causation of an outbox row whose write is audited.
--
-- New, empty tables: plain CREATE TABLE and CREATE INDEX hold no lock on any
-- existing table but the catalogue rows of the ones they reference (SHARE ROW
-- EXCLUSIVE on tenants, principals and entity_definitions for the foreign
-- keys, held for milliseconds). lock_timeout makes the migration fail and be
-- re-run instead of queueing reads behind a long write.
-- Design and EXPLAIN evidence: docs/data/workflow-definitions.md.

-- +goose Up

SET LOCAL lock_timeout = '5s';

-- One row per workflow. key is unique per tenant and set once.
-- current_version is the version new starts use; nava_app moves it forward
-- when it publishes, in the publish transaction. entity_definition_id is the
-- entity type the workflow belongs to. created_by names a principal of the
-- same tenant through the composite foreign key; it is NULL only for a
-- definition the platform seeds (a template, a demo), which no principal
-- wrote, as entity_definition_versions.published_by (P3-02). A NULL skips the
-- foreign key (MATCH SIMPLE) and names no other tenant's row.
CREATE TABLE workflow_definitions (
    tenant_id            uuid        NOT NULL,
    id                   uuid        NOT NULL,
    key                  text        NOT NULL,
    entity_definition_id uuid        NOT NULL,
    current_version      integer     NOT NULL,
    created_by           uuid,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT workflow_definitions_pkey PRIMARY KEY (tenant_id, id),
    CONSTRAINT workflow_definitions_key_key UNIQUE (tenant_id, key),
    CONSTRAINT workflow_definitions_tenant_fkey FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT workflow_definitions_entity_definition_fkey FOREIGN KEY (tenant_id, entity_definition_id)
        REFERENCES entity_definitions (tenant_id, id),
    CONSTRAINT workflow_definitions_created_by_fkey FOREIGN KEY (tenant_id, created_by)
        REFERENCES principals (tenant_id, id),
    CONSTRAINT workflow_definitions_key_check CHECK (key ~ '^[a-z][a-z0-9_]{1,40}$'),
    CONSTRAINT workflow_definitions_current_version_check CHECK (current_version >= 1)
);
ALTER TABLE workflow_definitions ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_definitions FORCE ROW LEVEL SECURITY;

-- No DELETE policy: there is nothing to allow.
CREATE POLICY workflow_definitions_select ON workflow_definitions FOR SELECT TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY workflow_definitions_insert ON workflow_definitions FOR INSERT TO nava_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY workflow_definitions_update ON workflow_definitions FOR UPDATE TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- One row per published version. Immutable: no role has UPDATE or DELETE, so
-- a running workflow pinned to a version id (D-3-12) always reads what was
-- published.
--
-- body is json, not jsonb, with the same sha256 CHECK as
-- entity_definition_versions.schema (D-3-4): jsonb would reorder keys, drop
-- duplicates and rewrite whitespace, and the bytes the publisher validated
-- are the bytes the interpreter reads.
--
-- step_ids is the step-id map AssignStepIDs returns at publish (D-3-15):
-- step key -> UUIDv4, plus "signal:<name>" -> UUIDv4 for each signal. jsonb,
-- because it is read whole and never hashed. trigger_on and
-- trigger_entity_definition_id copy body's trigger (def.Trigger.On, and the
-- entity its key resolves to at publish) so the sink can find the versions a
-- write fires without parsing body. published_by is NULL only as created_by
-- is.
CREATE TABLE workflow_definition_versions (
    tenant_id                    uuid        NOT NULL,
    id                           uuid        NOT NULL,
    definition_id                uuid        NOT NULL,
    version                      integer     NOT NULL,
    body                         json        NOT NULL,
    body_sha256                  bytea       NOT NULL,
    step_ids                     jsonb       NOT NULL,
    trigger_on                   text        NOT NULL,
    trigger_entity_definition_id uuid        NOT NULL,
    published_by                 uuid,
    published_at                 timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT workflow_definition_versions_pkey PRIMARY KEY (tenant_id, definition_id, version),
    CONSTRAINT workflow_definition_versions_id_key UNIQUE (tenant_id, id),
    CONSTRAINT workflow_definition_versions_definition_fkey FOREIGN KEY (tenant_id, definition_id)
        REFERENCES workflow_definitions (tenant_id, id),
    CONSTRAINT workflow_definition_versions_trigger_entity_fkey FOREIGN KEY (tenant_id, trigger_entity_definition_id)
        REFERENCES entity_definitions (tenant_id, id),
    CONSTRAINT workflow_definition_versions_published_by_fkey FOREIGN KEY (tenant_id, published_by)
        REFERENCES principals (tenant_id, id),
    CONSTRAINT workflow_definition_versions_version_check CHECK (version >= 1),
    CONSTRAINT workflow_definition_versions_body_sha256_check
        CHECK (body_sha256 = sha256(convert_to(body::text, 'UTF8'))),
    CONSTRAINT workflow_definition_versions_step_ids_check CHECK (jsonb_typeof(step_ids) = 'object'),
    CONSTRAINT workflow_definition_versions_trigger_on_check
        CHECK (trigger_on IN ('record.created', 'record.updated'))
);
ALTER TABLE workflow_definition_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_definition_versions FORCE ROW LEVEL SECURITY;

-- No UPDATE or DELETE policy: there is nothing to allow.
CREATE POLICY workflow_definition_versions_select ON workflow_definition_versions FOR SELECT TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY workflow_definition_versions_insert ON workflow_definition_versions FOR INSERT TO nava_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- The sink's trigger lookup (P3-42), on every record write:
--   SELECT v.id, v.definition_id FROM workflow_definition_versions v
--   JOIN workflow_definitions d ON d.tenant_id = v.tenant_id AND d.id = v.definition_id
--                              AND d.current_version = v.version
--   WHERE v.tenant_id = $1 AND v.trigger_entity_definition_id = $2 AND v.trigger_on = $3
-- "Current" lives on workflow_definitions, so no partial index can hold it;
-- the INCLUDE columns let the scan hand the join its probe keys without a
-- heap visit, and the join is a primary-key probe per version. Also the
-- index behind workflow_definition_versions_trigger_entity_fkey.
CREATE INDEX workflow_definition_versions_trigger ON workflow_definition_versions
    (tenant_id, trigger_entity_definition_id, trigger_on) INCLUDE (definition_id, version, id);

-- One row per start or signal the relay dispatches to a nava.interpreter run,
-- copied from that outbox row (D-3-29). Durable for the run's life: outbox
-- rows are purged 7 days after dispatch (config.MinOutboxDispatchedRetention)
-- and a wait can last 30 days. The RecordWriter adapter (P3-43) reads it by
-- (tenant_id, outbox_id), a primary-key probe, under RLS, and refuses it
-- unless ref_outbox_id is the run's own ref.
--
-- No foreign key to outbox or workflow_ref: both are purged, and a NO ACTION
-- key would make their purge fail. Nobody updates or deletes a row; its
-- purge, alongside workflow_ref's, is a later task (see the PR).
CREATE TABLE workflow_cause (
    tenant_id           uuid        NOT NULL,
    outbox_id           uuid        NOT NULL,
    ref_outbox_id       uuid        NOT NULL,
    causation_origin_id uuid        NOT NULL,
    causation_depth     smallint    NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT workflow_cause_pkey PRIMARY KEY (tenant_id, outbox_id),
    CONSTRAINT workflow_cause_tenant_fkey FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT workflow_cause_causation_depth_check CHECK (causation_depth >= 0)
);
ALTER TABLE workflow_cause ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_cause FORCE ROW LEVEL SECURITY;

-- No UPDATE or DELETE policy: there is nothing to allow.
CREATE POLICY workflow_cause_select ON workflow_cause FOR SELECT TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY workflow_cause_insert ON workflow_cause FOR INSERT TO nava_app, nava_worker
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- +goose Down

SET LOCAL lock_timeout = '5s';

DROP TABLE workflow_cause;
DROP TABLE workflow_definition_versions;
DROP TABLE workflow_definitions;
