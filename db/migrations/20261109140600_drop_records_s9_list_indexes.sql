-- TICK-640 (P3-34b): drops P3-09's list indexes, which the records_list_*
-- set (20261109140000 to 20261109140500) supersedes. None leads with
-- entity_definition_id, which every list statement binds (P3-36, PR #534).
--   records_tenant_updated     -> records_list_updated
--   records_tenant_requester   -> records_list_requester
--   records_tenant_group_open  -> records_list_group (not partial, so it
--                                 also serves closed records and the
--                                 records_assignment_group_fkey RI check)
-- No other statement uses them. store.Get, the write and cascade locks and
-- the outbox lock read records_pkey; the unique-value check and numbering
-- read records_number_key; the audit history reads audit_log. Plans before
-- and after: docs/data/records-list-indexes.md.
--
-- Online: DROP INDEX CONCURRENTLY, which cannot run in a transaction block,
-- so this file is NO TRANSACTION. IF EXISTS makes it re-runnable after a
-- partial failure. The down migration rebuilds them CONCURRENTLY, dropping
-- an INVALID leftover first.

-- +goose NO TRANSACTION
-- +goose Up

DROP INDEX CONCURRENTLY IF EXISTS records_tenant_updated;
DROP INDEX CONCURRENTLY IF EXISTS records_tenant_requester;
DROP INDEX CONCURRENTLY IF EXISTS records_tenant_group_open;

-- +goose Down

-- +goose StatementBegin
DO $$
DECLARE
    i text;
BEGIN
    FOREACH i IN ARRAY ARRAY['records_tenant_updated', 'records_tenant_requester', 'records_tenant_group_open'] LOOP
        IF EXISTS (SELECT 1 FROM pg_index WHERE indexrelid = to_regclass(i) AND NOT indisvalid) THEN
            EXECUTE format('DROP INDEX %I', i);
        END IF;
    END LOOP;
END
$$;
-- +goose StatementEnd
CREATE INDEX CONCURRENTLY IF NOT EXISTS records_tenant_updated ON records (tenant_id, updated_at DESC, id DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS records_tenant_requester ON records (tenant_id, requester_id, updated_at DESC, id DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS records_tenant_group_open ON records (tenant_id, assignment_group_id, updated_at DESC, id DESC)
    WHERE state <> 'closed';
