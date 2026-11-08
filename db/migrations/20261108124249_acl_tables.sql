-- P3-30 (TICK-615): the S10 authorization tables (PLAN-003 D-3-2, the S10
-- rows; ADR-001 I5: one authorization layer, inside the query).
--
--   acl_versions   an entity type's published ACL: role/group rules and field
--                  masks, one immutable row per publish, numbered per type
--                  like entity_definition_versions. The evaluator (P3-31)
--                  reads the newest version of a type: a backward primary-key
--                  probe, (tenant_id, entity_definition_id, version DESC).
--   record_shares  a record shared with one user or group. The list query
--                  (P3-34) joins it on the caller's principal ids; the
--                  record view probes it by record.
--
-- Both are tenant tables (D-3-2): tenant_id leads every key and index, RLS is
-- ENABLEd and FORCEd, and every foreign key is composite, because PostgreSQL
-- runs foreign-key checks as the table owner and so ignores RLS: tenant A
-- cannot publish an ACL for tenant B's type or share tenant B's record.
--
-- Immutable: nava_app and nava_worker get SELECT and INSERT only, and there is
-- no UPDATE or DELETE policy. Publishing an ACL inserts version n + 1; an
-- unshare is out of scope for S10 (a later task adds a DELETE grant and
-- policy with it). No sequence (I8).
--
-- published_by and granted_by carry tenant_id into principals (D-3-2), as
-- entity_definition_versions.published_by does. NULL is allowed for a row no
-- principal wrote (a system or onboarding default); a NULL skips the foreign
-- key (MATCH SIMPLE) and names no other tenant's row.
--
-- record_shares.principal_id has no foreign key: it names a user (principals)
-- or a group (groups), by principal_kind, and one column cannot reference
-- both. A foreign principal id is harmless: the evaluator matches it only
-- against the caller's own principal and group ids, read under tenant A's
-- RLS, so it matches nothing. principal_kind sits before principal_id in the
-- key so a user and a group with the same uuid stay two shares.
--
-- rules and masks have no shape CHECK yet: their format is the ACL compiler's
-- (P3-31), which adds one, NOT VALID then VALIDATE, with the format.
--
-- Index set: the two primary keys and record_shares_principal. The ACL lookup
-- and the share probe by record are primary-key scans; the records foreign
-- key on record_shares is backed by its primary key's (tenant_id, record_id)
-- prefix. record_shares_principal (tenant_id, principal_id) is the list join:
-- "records shared with any of these principal ids" under one tenant. A
-- covering (tenant_id, principal_id, record_id) was measured and rejected:
-- 129 MB against 18 MB at 2,000,000 shares (B-tree deduplication collapses
-- the repeated (tenant, principal) keys of the two-column index), and no
-- faster, 12.4 to 16.5 ms against 12.0 to 13.0 ms, because the join's
-- records_pkey probes dominate both (docs/data/acl-tables.md,
-- scratch/acl-tables-bench). Plain CREATE INDEX: the tables are created in
-- this migration and are empty.
--
-- Who may do what (db/grants.yaml): nava_app and nava_worker SELECT and
-- INSERT on both; no other role has anything. No policy on the A.3
-- allow-list.

-- +goose Up

CREATE TABLE acl_versions (
    tenant_id            uuid        NOT NULL,
    entity_definition_id uuid        NOT NULL,
    version              integer     NOT NULL,
    rules                jsonb       NOT NULL,
    masks                jsonb       NOT NULL,
    published_by         uuid,
    published_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT acl_versions_pkey PRIMARY KEY (tenant_id, entity_definition_id, version),
    CONSTRAINT acl_versions_entity_definition_fkey FOREIGN KEY (tenant_id, entity_definition_id)
        REFERENCES entity_definitions (tenant_id, id),
    CONSTRAINT acl_versions_published_by_fkey FOREIGN KEY (tenant_id, published_by)
        REFERENCES principals (tenant_id, id),
    CONSTRAINT acl_versions_version_check CHECK (version >= 1)
);
ALTER TABLE acl_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE acl_versions FORCE ROW LEVEL SECURITY;

-- No UPDATE or DELETE policy: there is nothing to allow.
CREATE POLICY acl_versions_select ON acl_versions FOR SELECT TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY acl_versions_insert ON acl_versions FOR INSERT TO nava_app, nava_worker
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

CREATE TABLE record_shares (
    tenant_id      uuid        NOT NULL,
    record_id      uuid        NOT NULL,
    principal_id   uuid        NOT NULL,
    principal_kind text        NOT NULL,
    granted_by     uuid,
    granted_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT record_shares_pkey PRIMARY KEY (tenant_id, record_id, principal_kind, principal_id),
    CONSTRAINT record_shares_record_fkey FOREIGN KEY (tenant_id, record_id)
        REFERENCES records (tenant_id, id),
    CONSTRAINT record_shares_granted_by_fkey FOREIGN KEY (tenant_id, granted_by)
        REFERENCES principals (tenant_id, id),
    CONSTRAINT record_shares_principal_kind_check CHECK (principal_kind IN ('user', 'group'))
);
CREATE INDEX record_shares_principal ON record_shares (tenant_id, principal_id);
ALTER TABLE record_shares ENABLE ROW LEVEL SECURITY;
ALTER TABLE record_shares FORCE ROW LEVEL SECURITY;

-- No UPDATE or DELETE policy: there is nothing to allow.
CREATE POLICY record_shares_select ON record_shares FOR SELECT TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY record_shares_insert ON record_shares FOR INSERT TO nava_app, nava_worker
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- +goose Down

-- Drops the tables with their policies, indexes and grants. Safe only before
-- S10 ships: afterwards it deletes every published ACL and share.
DROP TABLE IF EXISTS record_shares;
DROP TABLE IF EXISTS acl_versions;
