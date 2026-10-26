-- T13 follow-up (TICK-146): indexes for nava-purge's batch DELETEs
-- (internal/purge), PR #51's non-blocking item 2. Without them every batch,
-- and always the last one of a run, reads the whole table. Evidence:
-- docs/data/purge-indexes.md.
--
-- One index per purge predicate, each partial on "column IS NOT NULL" so the
-- open rows (pending outbox rows, running workflows) are never in it. outbox
-- needs both of its indexes: its predicate is "dispatched_at < ... OR dead_at
-- < ...", which only a BitmapOr of the two can serve.
--
-- No new grants and no policy changes: nava_purge already reads these
-- columns, and the *_purge policies are unchanged.
--
-- The tables may hold rows, so CONCURRENTLY, which cannot run in a
-- transaction. A failed CREATE INDEX CONCURRENTLY leaves an INVALID index
-- under the name, which IF NOT EXISTS would then keep, so each index is
-- dropped first: a re-run rebuilds it (TestNoTransactionMigrationsAreIdempotent).

-- +goose NO TRANSACTION
-- +goose Up

DROP INDEX CONCURRENTLY IF EXISTS outbox_purge_dispatched;
CREATE INDEX CONCURRENTLY IF NOT EXISTS outbox_purge_dispatched ON outbox (dispatched_at)
    WHERE dispatched_at IS NOT NULL;

DROP INDEX CONCURRENTLY IF EXISTS outbox_purge_dead;
CREATE INDEX CONCURRENTLY IF NOT EXISTS outbox_purge_dead ON outbox (dead_at)
    WHERE dead_at IS NOT NULL;

DROP INDEX CONCURRENTLY IF EXISTS workflow_ref_purge_closed;
CREATE INDEX CONCURRENTLY IF NOT EXISTS workflow_ref_purge_closed ON workflow_ref (closed_at)
    WHERE closed_at IS NOT NULL;

DROP INDEX CONCURRENTLY IF EXISTS idempotency_keys_purge_expires;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idempotency_keys_purge_expires ON idempotency_keys (expires_at);
