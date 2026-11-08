-- P5-17 (TICK-435): drops S9's audit_log_v0 (PLAN-005 D-5-1, moved here from
-- P5-13 by the TICK-449 grooming). Since P5-13b (TICK-452) nothing writes it;
-- this task rebuilds the per-record history (P3-15) on audit_log in the same
-- pull request, so the history endpoint never reads a table that is gone.
--
-- DROP TABLE removes the table's indexes (audit_log_v0_pkey, _record,
-- _idempotency_key), its policies (audit_log_v0_select, _insert), its
-- constraints and every grant on it. Nothing else references it: no foreign
-- key points at it, and its own two point at tenants and records. Its rows
-- are S9-era history of records written before P5-13b. They are not copied:
-- ADR-004 R25 backfills configuration versions only, and S9 ran on
-- development stacks. db/grants.yaml loses its two audit_log_v0 lines in
-- the same change.
--
-- Rollout order. Binary first, then this migration, back to back. The new
-- binary reads only audit_log, and the reader's SELECT on audit_log comes
-- from the grants that nava-db migrate applies. Between the binary deploy and
-- nava-db migrate, GET /api/v1/audit/entries and the record history fail;
-- record writes do not. An old binary breaks once the table is gone, so the
-- migration never runs first. Every stack today is development. For a stack
-- that must have zero downtime, split the release: migrate with the reader
-- grant, deploy the binary, then drop (and remove the audit_log_v0 grant).
-- Runbook: docs/architecture/record-write-pipeline.md, "Rollout order".

-- +goose Up
SET LOCAL lock_timeout = '5s';
DROP TABLE audit_log_v0;

-- +goose Down
-- Not reversible: the rows are gone. Restore from backup if they are needed.
SELECT 1/0;
