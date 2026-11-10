-- TICK-656 (P3-36d): the group access path with state as an index condition.
-- An agent's list filtered on state reads the group branch of the ACL
-- (authz.ReadBranches) through this index's (assignment_group_id, state)
-- range, and no longer reads every row of its groups to drop the other states
-- (records_list_group: 22,950 of 24,118 rows removed by filter at 800k).
-- records_list_group stays: sort=[-]group orders assignment_group_id,
-- updated_at, id, which this index cannot give with state in between.
-- Evidence (EXPLAIN ANALYZE on P3-01's 800k shape):
-- docs/data/records-list-indexes.md. B-tree on fixed columns only: no
-- expression index, no partial index on "open" states and nothing on data
-- (PLAN-003 D25 cut; the CTO on TICK-336).
--
-- Online: CREATE INDEX CONCURRENTLY, alone in a NO TRANSACTION file. A
-- failed build leaves an INVALID index, which the DO block drops on the
-- re-run (TestNoTransactionMigrationsAreIdempotent); a valid one is kept.

-- +goose NO TRANSACTION
-- +goose Up

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_index WHERE indexrelid = to_regclass('records_list_group_state') AND NOT indisvalid) THEN
        DROP INDEX records_list_group_state;
    END IF;
END
$$;
-- +goose StatementEnd
CREATE INDEX CONCURRENTLY IF NOT EXISTS records_list_group_state
    ON records (tenant_id, entity_definition_id, assignment_group_id DESC, state DESC, updated_at DESC, id DESC);

-- +goose Down

DROP INDEX CONCURRENTLY IF EXISTS records_list_group_state;
