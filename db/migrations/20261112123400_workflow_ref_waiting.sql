-- P3-40 (TICK-709), D-3-29: the sink's lookup of the open runs on a record
-- that wait on record_changed (P3-42), on every record write:
--   SELECT outbox_id, record_changed_signal_id FROM workflow_ref
--   WHERE tenant_id = $1 AND record_id = $2
--     AND closed_at IS NULL AND record_changed_signal_id IS NOT NULL
-- Partial: only waiting runs are indexed, so the index stays the size of the
-- waiting set, not of workflow_ref, and a start or close that leaves
-- record_changed_signal_id NULL writes no entry.
-- Evidence: docs/data/workflow-definitions.md.
--
-- Online: CREATE INDEX CONCURRENTLY, alone in a NO TRANSACTION file. A
-- failed build leaves an INVALID index, which the DO block drops on the
-- re-run (TestNoTransactionMigrationsAreIdempotent); a valid one is kept.

-- +goose NO TRANSACTION
-- +goose Up

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_index WHERE indexrelid = to_regclass('workflow_ref_waiting') AND NOT indisvalid) THEN
        DROP INDEX workflow_ref_waiting;
    END IF;
END
$$;
-- +goose StatementEnd
CREATE INDEX CONCURRENTLY IF NOT EXISTS workflow_ref_waiting
    ON workflow_ref (tenant_id, record_id)
    WHERE closed_at IS NULL AND record_changed_signal_id IS NOT NULL;

-- +goose Down

DROP INDEX CONCURRENTLY IF EXISTS workflow_ref_waiting;
