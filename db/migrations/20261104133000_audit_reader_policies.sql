-- ADR-004-A1 A1-3 (TICK-558): nava_audit_reader, the masking reader's own
-- login (db/grants.yaml), reads audit_log and the masker's
-- entity_definition_versions rows under the canonical predicate, unchanged.
-- Without the second policy its masker sees no version row and fails closed
-- (A1-4: a 500, never an unmasked diff).
--
-- nava_app stays out of audit_log_select: its audit_log rows come only from
-- the generated audit_log_writer_select (internal/db/audited/policies.go).
-- CI-A3 asserts this exact role list (A1-9).
--
-- Rollout (A1-11): one release, with the role, its password, the PgBouncer
-- line, the revoke of nava_app's diff and diff_salt (applied by nava-db
-- migrate from db/grants.yaml after this migration) and the nava-core binary
-- that reads through the reader's pool. ALTER POLICY takes an ACCESS
-- EXCLUSIVE lock on each table for a catalogue update only: no rewrite, no
-- scan, so the lock is held for milliseconds; lock_timeout bounds the wait
-- behind a long transaction.

-- +goose Up
SET LOCAL lock_timeout = '5s';
ALTER POLICY audit_log_select ON audit_log TO nava_worker, nava_erasure, nava_audit_reader;
ALTER POLICY entity_definition_versions_select ON entity_definition_versions TO nava_app, nava_worker, nava_audit_reader;

-- +goose Down
SET LOCAL lock_timeout = '5s';
ALTER POLICY entity_definition_versions_select ON entity_definition_versions TO nava_app, nava_worker;
ALTER POLICY audit_log_select ON audit_log TO nava_worker, nava_erasure;
