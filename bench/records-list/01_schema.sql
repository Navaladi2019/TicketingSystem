-- Synthetic records table shaped like PLAN-003 `records`. Not a migration.
CREATE EXTENSION IF NOT EXISTS pg_prewarm;
CREATE EXTENSION IF NOT EXISTS pgstattuple;
SET ROLE bench_owner;
CREATE TABLE records (
    tenant_id            uuid        NOT NULL,
    id                   uuid        NOT NULL,
    entity_definition_id uuid        NOT NULL,
    number               bigint      NOT NULL,
    state                text        NOT NULL CHECK (state IN ('new', 'in_progress', 'on_hold', 'closed')),
    requester_id         uuid        NOT NULL,
    assignee_id          uuid,
    assignment_group_id  uuid,
    created_at           timestamptz NOT NULL,
    updated_at           timestamptz NOT NULL,
    data                 jsonb       NOT NULL,
    CONSTRAINT records_pkey PRIMARY KEY (tenant_id, id)
);
CREATE TABLE record_shares (
    tenant_id    uuid NOT NULL,
    record_id    uuid NOT NULL,
    principal_id uuid NOT NULL,
    CONSTRAINT record_shares_pkey PRIMARY KEY (tenant_id, principal_id, record_id)
);
-- The platform pattern (db/migrations/00002_platform_tables.sql): forced RLS,
-- the tenant from the transaction-local app.tenant_id.
ALTER TABLE records ENABLE ROW LEVEL SECURITY;
ALTER TABLE records FORCE ROW LEVEL SECURITY;
ALTER TABLE record_shares ENABLE ROW LEVEL SECURITY;
ALTER TABLE record_shares FORCE ROW LEVEL SECURITY;
CREATE POLICY records_select ON records FOR SELECT TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY records_insert ON records FOR INSERT TO nava_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY records_update ON records FOR UPDATE TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY record_shares_select ON record_shares FOR SELECT TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
GRANT SELECT, INSERT, UPDATE ON records TO nava_app;
GRANT SELECT ON record_shares TO nava_app;
RESET ROLE;
