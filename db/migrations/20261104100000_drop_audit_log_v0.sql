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
-- Rollout order (contract step, Postgres review of PR #432). Deploy the new
-- binary first, then apply this migration and the grants. An old binary
-- still reads audit_log_v0 for history and breaks once the table is gone;
-- it no longer writes it (INSERT was revoked in P5-13b). Runbook:
-- docs/architecture/record-write-pipeline.md, "Rollout order".

-- +goose Up
DROP TABLE audit_log_v0;

-- +goose Down
-- Not reversible: the rows are gone. Restore from backup if they are needed.
SELECT 1/0;
