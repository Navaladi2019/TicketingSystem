-- SP-1b (TICK-969, ADR-010 R1): bench-only closure table over SP-1's org tree
-- (region.sql must have run). NOT a migration. Runs as the superuser in the
-- kept bench database `nava`. The rows are inserted by closure/main.go, which
-- times the build; this file only creates the empty table, its RLS and grant.
\set ON_ERROR_STOP on
DROP TABLE IF EXISTS org_unit_closure;
CREATE TABLE org_unit_closure (
    tenant_id     uuid NOT NULL,
    ancestor_id   uuid NOT NULL,
    descendant_id uuid NOT NULL,
    depth         int  NOT NULL,
    CONSTRAINT org_unit_closure_pkey PRIMARY KEY (tenant_id, ancestor_id, descendant_id)
);
ALTER TABLE org_unit_closure ENABLE ROW LEVEL SECURITY;
ALTER TABLE org_unit_closure FORCE ROW LEVEL SECURITY;
-- Same shape as org_units_select (00003_identity_tables.sql:127).
CREATE POLICY org_unit_closure_select ON org_unit_closure FOR SELECT TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
GRANT SELECT ON org_unit_closure TO nava_app;
