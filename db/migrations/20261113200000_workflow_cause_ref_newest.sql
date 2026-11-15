-- TICK-785 (S11b follow-up to TICK-712): back causeSQL's newest-cause check
-- (internal/workerguard/cause.go). ResolveCause refuses a cause row when the
-- same run has a later row, by (created_at, outbox_id). With only
-- workflow_cause_pkey (tenant_id, outbox_id) that EXISTS read every cause row
-- of the tenant. This index makes it one range probe of the run's own rows:
-- tenant_id leads, ref_outbox_id is the equality, and the row comparison
-- (created_at, outbox_id) > (...) is an index condition on the last two
-- columns, so the probe is index-only.
-- Evidence (EXPLAIN (ANALYZE, BUFFERS), 2.1M rows, postgres:16.10):
-- docs/data/workflow-definitions.md, "Newest-cause check (TICK-785)".
-- No grant, policy or isolation-registry change: the table's RLS and
-- grants are unchanged.
--
-- Online: CREATE INDEX CONCURRENTLY, alone in a NO TRANSACTION file. A
-- failed build leaves an INVALID index, which the DO block drops on the
-- re-run (TestNoTransactionMigrationsAreIdempotent); a valid one is kept.

-- +goose NO TRANSACTION
-- +goose Up

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_index WHERE indexrelid = to_regclass('workflow_cause_ref_newest') AND NOT indisvalid) THEN
        DROP INDEX workflow_cause_ref_newest;
    END IF;
END
$$;
-- +goose StatementEnd
CREATE INDEX CONCURRENTLY IF NOT EXISTS workflow_cause_ref_newest
    ON workflow_cause (tenant_id, ref_outbox_id, created_at, outbox_id);

-- +goose Down

DROP INDEX CONCURRENTLY IF EXISTS workflow_cause_ref_newest;
