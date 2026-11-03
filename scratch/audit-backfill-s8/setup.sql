-- P5-15 (TICK-433) backfill timing: roles, stub parents owned by
-- nava_migrator, then the real audit_log migration and the real backfill
-- migration via \i (its backfill runs on empty tables here).
\set ON_ERROR_STOP on
CREATE ROLE nava_app; CREATE ROLE nava_worker; CREATE ROLE nava_erasure; CREATE ROLE nava_migrator;
GRANT CREATE, USAGE ON SCHEMA public TO nava_migrator;
SET ROLE nava_migrator;
CREATE TABLE tenants (id uuid PRIMARY KEY);
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY; ALTER TABLE tenants FORCE ROW LEVEL SECURITY;
CREATE TABLE entity_definitions (tenant_id uuid, id uuid, PRIMARY KEY (tenant_id, id));
CREATE TABLE principals (tenant_id uuid, id uuid, PRIMARY KEY (tenant_id, id));
CREATE TABLE records (tenant_id uuid, id uuid, PRIMARY KEY (tenant_id, id));
-- entity_definition_versions as 20261031150000 creates it, keys included.
CREATE TABLE entity_definition_versions (
    tenant_id uuid NOT NULL, id uuid NOT NULL, definition_id uuid NOT NULL, version integer NOT NULL,
    schema json NOT NULL, schema_sha256 bytea NOT NULL, published_by uuid,
    published_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, definition_id, version), UNIQUE (tenant_id, id),
    FOREIGN KEY (tenant_id, definition_id) REFERENCES entity_definitions (tenant_id, id),
    FOREIGN KEY (tenant_id, published_by) REFERENCES principals (tenant_id, id));
ALTER TABLE entity_definition_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE entity_definition_versions FORCE ROW LEVEL SECURITY;
BEGIN;
\i /work/audit_log.sql
COMMIT;
BEGIN;
\i /work/backfill.sql
COMMIT;
RESET ROLE;
