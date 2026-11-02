-- P3-12 (TICK-88): cascade_start, the start key of ADR-001 amendment A5
-- (A5-2). A workflow definition starts at most once per (tenant, origin,
-- record): the EventSink claims the key with
--
--   INSERT INTO cascade_start (tenant_id, causation_origin_id, record_id, definition_id, outbox_id)
--   VALUES (...) ON CONFLICT ON CONSTRAINT cascade_start_pkey DO NOTHING RETURNING outbox_id
--
-- in the triggering write's transaction, after the write holds R's row lock
-- (A5-1). One row back: enqueue the start under outbox_id. None: a cycle, so
-- the start is suppressed and audited (A5-4). Only cascade.ClaimStart writes
-- here.
--
-- A dedicated table, not an index on outbox or workflow_ref: both are purged
-- and neither holds the definition id, and the key must outlive the longest
-- workflow. Rows are kept for the life of the record (records are never
-- deleted in M1); a future record purge deletes a record's rows first,
-- through a nava_purge DELETE policy, because the foreign key is NO ACTION.
--
-- Tenant table: tenant_id leads the key, RLS is ENABLEd and FORCEd, and the
-- record foreign key is composite, because PostgreSQL runs foreign-key checks
-- as the table owner and so ignores RLS: tenant A cannot claim a key on
-- tenant B's record. definition_id gets its composite foreign key to
-- workflow_definitions with P3-40, which creates that table.
--
-- Who may do what (db/grants.yaml): nava_app and nava_worker INSERT and
-- SELECT (RETURNING and ON CONFLICT need it). Nobody updates or deletes a
-- claim. No sequence (I8): outbox_id is a UUIDv7 minted by ClaimStart.
--
-- No extra index: the claim is a primary-key probe. The record foreign key's
-- referencing side has no index of its own; records are never deleted or
-- re-keyed in M1, so nothing scans it. The purge task that adds the DELETE
-- policy adds (tenant_id, record_id) with it.

-- +goose Up

CREATE TABLE cascade_start (
    tenant_id           uuid        NOT NULL,
    causation_origin_id uuid        NOT NULL,
    record_id           uuid        NOT NULL,
    definition_id       uuid        NOT NULL,
    outbox_id           uuid        NOT NULL, -- UUIDv7, the start row's id
    created_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT cascade_start_pkey PRIMARY KEY (tenant_id, causation_origin_id, record_id, definition_id),
    CONSTRAINT cascade_start_tenant_fkey FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT cascade_start_record_fkey FOREIGN KEY (tenant_id, record_id) REFERENCES records (tenant_id, id)
);
ALTER TABLE cascade_start ENABLE ROW LEVEL SECURITY;
ALTER TABLE cascade_start FORCE ROW LEVEL SECURITY;

-- No UPDATE or DELETE policy: there is nothing to allow.
CREATE POLICY cascade_start_select ON cascade_start FOR SELECT TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY cascade_start_insert ON cascade_start FOR INSERT TO nava_app, nava_worker
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
