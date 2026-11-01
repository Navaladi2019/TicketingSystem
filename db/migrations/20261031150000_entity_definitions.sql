-- P3-02 (TICK-340): entity types and their published, immutable versions
-- (PLAN-003 D-3-2, D-3-4; groomed on TICK-360 and TICK-363). Both tables are
-- tenant tables under ENABLE + FORCE ROW LEVEL SECURITY; every key and
-- foreign key leads with tenant_id, because PostgreSQL runs foreign-key
-- checks as the table owner and so ignores RLS (D-3-2).
--
-- Who may do what (db/grants.yaml):
--   nava_app      reads both, publishes versions (INSERT), creates types and
--                 moves a type's current_version / updated_at. Nothing else
--                 on a type changes: key and number_prefix are set once.
--   nava_worker   reads both.
--   nava_onboard  inserts both while its own tenant is provisioning (seed
--                 templates, P3-06, deterministic UUIDv5 ids), under the
--                 pinned policies in catalog.PinnedTenantPolicies.
--   nobody        UPDATEs or DELETEs a version (D-3-4): no grant, no policy.
--
-- Every policy is FOR one command and TO named roles and tests exactly the
-- canonical tenant predicate, except the two nava_onboard INSERT policies,
-- which add "and that tenant is provisioning" (pinned, not allow-listed).
--
-- Empty new tables, so plain CREATE TABLE / index builds hold no lock worth
-- planning for. Later CHECK constraints are added NOT VALID, then validated.
-- Design and EXPLAIN evidence: docs/data/platform-tables.md.

-- +goose Up

-- One row per entity type. number_prefix lives on the type, not the version,
-- so it is unique per tenant and record numbers never collide across types
-- (TICK-363). current_version is the version new records are written under;
-- nava_app moves it forward when it publishes.
CREATE TABLE entity_definitions (
    tenant_id       uuid        NOT NULL,
    id              uuid        NOT NULL,
    key             text        NOT NULL,
    name            text        NOT NULL,
    number_prefix   text        NOT NULL,
    current_version integer     NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT entity_definitions_pkey PRIMARY KEY (tenant_id, id),
    CONSTRAINT entity_definitions_key_key UNIQUE (tenant_id, key),
    CONSTRAINT entity_definitions_number_prefix_key UNIQUE (tenant_id, number_prefix),
    CONSTRAINT entity_definitions_tenant_fkey FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT entity_definitions_key_check CHECK (key ~ '^[a-z][a-z0-9_]{1,40}$'),
    CONSTRAINT entity_definitions_number_prefix_check CHECK (number_prefix ~ '^[A-Z]{2,6}$'),
    CONSTRAINT entity_definitions_current_version_check CHECK (current_version >= 1)
);
ALTER TABLE entity_definitions ENABLE ROW LEVEL SECURITY;
ALTER TABLE entity_definitions FORCE ROW LEVEL SECURITY;

CREATE POLICY entity_definitions_select ON entity_definitions FOR SELECT TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY entity_definitions_insert ON entity_definitions FOR INSERT TO nava_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY entity_definitions_update ON entity_definitions FOR UPDATE TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
-- tenants_onboard_select shows nava_onboard only the tenant of its
-- transaction, so the subquery adds "and that tenant is provisioning".
CREATE POLICY entity_definitions_onboard ON entity_definitions FOR INSERT TO nava_onboard
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
                AND tenant_id IN (SELECT id FROM tenants WHERE state = 'provisioning'));

-- One row per published version of a type. Immutable: no role has UPDATE or
-- DELETE (D-3-4).
--
-- schema is json, not jsonb: jsonb reorders keys, drops duplicate keys and
-- rewrites whitespace (checked on postgres:16.10, TICK-360), and the bytes
-- the publisher validated are the bytes the interpreter must read. Go reads
-- it as []byte. schema_sha256 is the SHA-256 of those exact bytes; the CHECK
-- keeps the two from ever disagreeing. Duplicate-key rejection is P3-03's
-- (TICK-341), before the insert.
--
-- id names a version on its own (the interpreter and P3-41, D-3-12).
-- published_by carries tenant_id into principals, so a version cannot name
-- another tenant's principal (D-3-2). It is NULL only for a version
-- onboarding seeds (P3-06): the tenant is still provisioning and group sync
-- has written no principal yet, so there is no one to name. A NULL skips the
-- foreign key (MATCH SIMPLE) and names no other tenant's row. Version 1, and
-- only version 1, is 'initial'.
CREATE TABLE entity_definition_versions (
    tenant_id     uuid        NOT NULL,
    id            uuid        NOT NULL,
    definition_id uuid        NOT NULL,
    version       integer     NOT NULL,
    schema        json        NOT NULL,
    schema_sha256 bytea       NOT NULL,
    fixed_columns jsonb       NOT NULL,
    states        jsonb       NOT NULL,
    compat        text        NOT NULL,
    published_by  uuid,
    published_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT entity_definition_versions_pkey PRIMARY KEY (tenant_id, definition_id, version),
    CONSTRAINT entity_definition_versions_id_key UNIQUE (tenant_id, id),
    CONSTRAINT entity_definition_versions_definition_fkey FOREIGN KEY (tenant_id, definition_id)
        REFERENCES entity_definitions (tenant_id, id),
    CONSTRAINT entity_definition_versions_published_by_fkey FOREIGN KEY (tenant_id, published_by)
        REFERENCES principals (tenant_id, id),
    CONSTRAINT entity_definition_versions_version_check CHECK (version >= 1),
    CONSTRAINT entity_definition_versions_schema_sha256_check
        CHECK (schema_sha256 = sha256(convert_to(schema::text, 'UTF8'))),
    CONSTRAINT entity_definition_versions_compat_check
        CHECK (compat IN ('initial', 'compatible', 'needs_backfill')),
    CONSTRAINT entity_definition_versions_initial_check CHECK ((version = 1) = (compat = 'initial'))
);
ALTER TABLE entity_definition_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE entity_definition_versions FORCE ROW LEVEL SECURITY;

-- No UPDATE or DELETE policy: there is nothing to allow (D-3-4).
CREATE POLICY entity_definition_versions_select ON entity_definition_versions FOR SELECT TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY entity_definition_versions_insert ON entity_definition_versions FOR INSERT TO nava_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY entity_definition_versions_onboard ON entity_definition_versions FOR INSERT TO nava_onboard
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
                AND tenant_id IN (SELECT id FROM tenants WHERE state = 'provisioning'));
