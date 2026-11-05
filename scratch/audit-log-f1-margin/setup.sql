-- P5-10b (TICK-450) Finding 5 margin spike: stub parents, the roles, then the
-- real migration (db/migrations/20261102233600_audit_log.sql) via \i.
\set ON_ERROR_STOP on
CREATE ROLE nava_app LOGIN PASSWORD 'spike';
CREATE ROLE nava_worker LOGIN;
CREATE ROLE nava_erasure LOGIN;
CREATE ROLE nava_migrator LOGIN;
CREATE TABLE tenants (id uuid PRIMARY KEY);
CREATE TABLE entity_definitions (tenant_id uuid, id uuid, PRIMARY KEY (tenant_id, id));
CREATE TABLE principals (tenant_id uuid, id uuid, PRIMARY KEY (tenant_id, id));
CREATE TABLE records (tenant_id uuid, id uuid, PRIMARY KEY (tenant_id, id));
\i /work/migration.sql
GRANT USAGE ON SCHEMA public TO nava_app;
GRANT SELECT (tenant_id, occurred_at, txid, subject_type, subject_id, version_after) ON audit_log TO nava_app;
