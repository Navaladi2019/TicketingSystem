-- P5-10a (TICK-427), PLAN-005 D-5-1: S9's interim audit table becomes
-- audit_log_v0, so ADR-004's audit_log (P5-10b) can take the name. The rows,
-- grants, RLS and every constraint stay as they are; only names change.
--
-- RENAME TO renames the table alone. Its constraints, indexes and policies
-- keep their audit_log_* names, and audit_log_pkey would collide with the new
-- table's primary key, so each is renamed here too. Renaming a PRIMARY KEY
-- constraint renames its index. On PostgreSQL 16 a NOT NULL is not a named
-- constraint, so there is nothing else to rename. The migration test checks
-- that no constraint, index or policy on audit_log_v0 keeps an audit_log_*
-- name.
--
-- Grants are on the table's oid and survive the rename; db/grants.yaml moves
-- to the new name in the same change so db-roles reconciles the same table.
--
-- Catalog-only: no rewrite, no scan. Each statement takes ACCESS EXCLUSIVE
-- on audit_log for a catalog update, under lock_timeout so it never queues
-- the pipeline behind a long transaction. The code that names the table ships
-- in the same release (no compatibility view: P5-10b needs the name).

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE audit_log RENAME TO audit_log_v0;

ALTER TABLE audit_log_v0 RENAME CONSTRAINT audit_log_pkey TO audit_log_v0_pkey;
ALTER TABLE audit_log_v0 RENAME CONSTRAINT audit_log_tenant_fkey TO audit_log_v0_tenant_fkey;
ALTER TABLE audit_log_v0 RENAME CONSTRAINT audit_log_record_fkey TO audit_log_v0_record_fkey;
ALTER TABLE audit_log_v0 RENAME CONSTRAINT audit_log_source_check TO audit_log_v0_source_check;
ALTER TABLE audit_log_v0 RENAME CONSTRAINT audit_log_causation_depth_check TO audit_log_v0_causation_depth_check;

ALTER INDEX audit_log_record RENAME TO audit_log_v0_record;
ALTER INDEX audit_log_idempotency_key RENAME TO audit_log_v0_idempotency_key;

ALTER POLICY audit_log_select ON audit_log_v0 RENAME TO audit_log_v0_select;
ALTER POLICY audit_log_insert ON audit_log_v0 RENAME TO audit_log_v0_insert;
