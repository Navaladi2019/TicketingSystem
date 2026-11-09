-- TICK-640 (P3-34b): the default sort (-updated_at) and sort=updated_at. A
-- descending list reads it forward (updated_at DESC NULLS FIRST, id DESC), an
-- ascending one backward.
-- Index set and evidence (EXPLAIN ANALYZE on P3-01's 800k shape):
-- docs/data/records-list-indexes.md. B-tree on fixed columns only: no
-- expression index and nothing on data (PLAN-003 D25 cut).
--
-- Online: CREATE INDEX CONCURRENTLY, alone in a NO TRANSACTION file. A
-- failed build leaves an INVALID index, which the DO block drops on the
-- re-run (TestNoTransactionMigrationsAreIdempotent); a valid one is kept.

-- +goose NO TRANSACTION
-- +goose Up

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_index WHERE indexrelid = to_regclass('records_list_updated') AND NOT indisvalid) THEN
        DROP INDEX records_list_updated;
    END IF;
END
$$;
-- +goose StatementEnd
CREATE INDEX CONCURRENTLY IF NOT EXISTS records_list_updated
    ON records (tenant_id, entity_definition_id, updated_at DESC, id DESC);

-- +goose Down

DROP INDEX CONCURRENTLY IF EXISTS records_list_updated;
