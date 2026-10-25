-- T16 (TICK-69): the identity tables group sync (T25) fills from Keycloak and
-- tenant resolution (T17, ADR-001 A.1 step 3) reads. Each is under ENABLE +
-- FORCE ROW LEVEL SECURITY, every key and foreign key leads with tenant_id,
-- and every policy is the canonical tenant predicate, FOR one command and TO
-- named roles: nava_groupsync writes (one transaction per tenant, D-P13),
-- nava_app reads. None is on the A.3 allow-list. Grants: db/grants.yaml.
--
-- Empty new tables, so plain CREATE INDEX holds no lock worth planning for.
-- Design and EXPLAIN evidence: docs/data/identity.md.

-- +goose Up

-- One row per Keycloak user who is a member of the tenant's organization.
-- federated_links is the user's federated identities from the Keycloak admin
-- API, as it returns them: [{"identityProvider": alias, "userId", "userName"}].
--   NULL  = unknown (never synced). The resolver returns 401 (A.1 step 3).
--   '[]'  = synced, no federated identity.
-- It is nullable with no default on purpose: NOT NULL DEFAULT '[]' would turn
-- "not synced yet" into "synced, no links" and fail open. The CHECK refuses a
-- non-array and any element without a string identityProvider, so the
-- resolver never meets a link it cannot compare with the bound aliases.
CREATE TABLE principals (
    tenant_id        uuid        NOT NULL,
    id               uuid        NOT NULL,
    keycloak_user_id text        NOT NULL,
    email            text,
    enabled          boolean     NOT NULL,
    federated_links  jsonb,
    synced_at        timestamptz NOT NULL,
    CONSTRAINT principals_pkey PRIMARY KEY (tenant_id, id),
    CONSTRAINT principals_keycloak_user_key UNIQUE (tenant_id, keycloak_user_id),
    CONSTRAINT principals_tenant_fkey FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT principals_federated_links_check CHECK (
        jsonb_typeof(federated_links) = 'array'
        AND NOT jsonb_path_exists(federated_links,
            '$[*] ? (!exists(@.identityProvider) || @.identityProvider.type() != "string")'))
);
ALTER TABLE principals ENABLE ROW LEVEL SECURITY;
ALTER TABLE principals FORCE ROW LEVEL SECURITY;

CREATE POLICY principals_select ON principals FOR SELECT TO nava_app, nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY principals_insert ON principals FOR INSERT TO nava_groupsync
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY principals_update ON principals FOR UPDATE TO nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY principals_delete ON principals FOR DELETE TO nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- Tenant groups, from Keycloak groups under /t/<tenant_id>/ (A.1 platform
-- rules). The CHECK keeps another tenant's subtree out even if sync is wrong.
CREATE TABLE groups (
    tenant_id         uuid NOT NULL,
    id                uuid NOT NULL,
    keycloak_group_id text NOT NULL,
    path              text NOT NULL,
    CONSTRAINT groups_pkey PRIMARY KEY (tenant_id, id),
    CONSTRAINT groups_keycloak_group_key UNIQUE (tenant_id, keycloak_group_id),
    CONSTRAINT groups_tenant_fkey FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT groups_path_check CHECK (starts_with(path, '/t/' || tenant_id::text || '/'))
);
ALTER TABLE groups ENABLE ROW LEVEL SECURITY;
ALTER TABLE groups FORCE ROW LEVEL SECURITY;

CREATE POLICY groups_select ON groups FOR SELECT TO nava_app, nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY groups_insert ON groups FOR INSERT TO nava_groupsync
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY groups_update ON groups FOR UPDATE TO nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY groups_delete ON groups FOR DELETE TO nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- Membership. Both foreign keys carry tenant_id, so a row cannot join a
-- principal to another tenant's group. Removing a principal or a group
-- removes its memberships (same tenant only: the key includes tenant_id).
-- The primary key serves the group side; group_members_principal serves the
-- resolver (a principal's groups) and principal_fkey's cascade. It is two
-- columns, not three with group_id: B-tree deduplication folds a principal's
-- entries into one, so it is about a fifth of the size for ~0.07 ms more per
-- lookup (docs/data/identity.md).
CREATE TABLE group_members (
    tenant_id    uuid NOT NULL,
    group_id     uuid NOT NULL,
    principal_id uuid NOT NULL,
    CONSTRAINT group_members_pkey PRIMARY KEY (tenant_id, group_id, principal_id),
    CONSTRAINT group_members_group_fkey FOREIGN KEY (tenant_id, group_id)
        REFERENCES groups (tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT group_members_principal_fkey FOREIGN KEY (tenant_id, principal_id)
        REFERENCES principals (tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX group_members_principal ON group_members (tenant_id, principal_id);
ALTER TABLE group_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE group_members FORCE ROW LEVEL SECURITY;

-- No UPDATE: a membership is inserted or deleted, never rewritten.
CREATE POLICY group_members_select ON group_members FOR SELECT TO nava_app, nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY group_members_insert ON group_members FOR INSERT TO nava_groupsync
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY group_members_delete ON group_members FOR DELETE TO nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- The org-unit tree, from Keycloak groups under /t/<tenant_id>/ou/... A root
-- has parent_id NULL. The self-FK carries tenant_id, so a unit cannot hang
-- under another tenant's unit; NO ACTION lets sync delete a whole subtree in
-- one statement. org_units_parent backs that FK's check on delete and the
-- children-of lookup.
CREATE TABLE org_units (
    tenant_id         uuid NOT NULL,
    id                uuid NOT NULL,
    parent_id         uuid,
    keycloak_group_id text NOT NULL,
    name              text NOT NULL,
    CONSTRAINT org_units_pkey PRIMARY KEY (tenant_id, id),
    CONSTRAINT org_units_keycloak_group_key UNIQUE (tenant_id, keycloak_group_id),
    CONSTRAINT org_units_tenant_fkey FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT org_units_parent_fkey FOREIGN KEY (tenant_id, parent_id) REFERENCES org_units (tenant_id, id),
    CONSTRAINT org_units_parent_check CHECK (parent_id <> id)
);
CREATE INDEX org_units_parent ON org_units (tenant_id, parent_id);
ALTER TABLE org_units ENABLE ROW LEVEL SECURITY;
ALTER TABLE org_units FORCE ROW LEVEL SECURITY;

CREATE POLICY org_units_select ON org_units FOR SELECT TO nava_app, nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY org_units_insert ON org_units FOR INSERT TO nava_groupsync
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY org_units_update ON org_units FOR UPDATE TO nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY org_units_delete ON org_units FOR DELETE TO nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
