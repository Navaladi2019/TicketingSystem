-- TICK-626 (AR-TICK-480-01): record_write_keys is unique per principal.
-- record_write_keys_pkey (tenant_id, idempotency_key) is replaced by
-- record_write_keys_actor_key, a unique index on
-- (tenant_id, idempotency_key, actor_principal_id) NULLS NOT DISTINCT.
-- Member B's write under member A's key now gets B's own row, and nothing
-- tells B that A's key exists. NULLS NOT DISTINCT keeps a system writer with
-- no principal in conflict with itself, so its retry replays and does not
-- write twice. The index cannot be a primary key, because the actor is
-- nullable. store.idempotencyIndex names the new index, and a 23505 on it
-- is ErrDuplicateKey.
--
-- Column order: key, then actor. Replay matches the actor with IS NOT
-- DISTINCT FROM, which a btree cannot search on. With the actor last, the
-- lookup is a two-column index probe plus a filter over that key's few rows.
-- Plan: docs/data/record-write-keys-per-principal.md.
--
-- Online. CREATE UNIQUE INDEX CONCURRENTLY first. The new index is strictly
-- finer than the primary key, so the existing rows fit it. Then the primary
-- key is dropped. That is ACCESS EXCLUSIVE, catalogue only, under
-- lock_timeout. Between the two steps both are enforced, which is stricter,
-- never looser. Uniqueness is never off.
--
-- Re-runnable after a partial failure (TestNoTransactionMigrationsAreIdempotent).
-- A failed CONCURRENTLY build leaves an INVALID index, and the DO block drops
-- it. A valid index is kept. Every later step is IF EXISTS / IF NOT EXISTS.

-- +goose NO TRANSACTION
-- +goose Up

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_index WHERE indexrelid = to_regclass('record_write_keys_actor_key') AND NOT indisvalid) THEN
        DROP INDEX record_write_keys_actor_key;
    END IF;
END
$$;
-- +goose StatementEnd
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS record_write_keys_actor_key
    ON record_write_keys (tenant_id, idempotency_key, actor_principal_id) NULLS NOT DISTINCT;
SET lock_timeout = '5s';
ALTER TABLE record_write_keys DROP CONSTRAINT IF EXISTS record_write_keys_pkey;
RESET lock_timeout;

-- +goose Down

-- This fails on the CREATE if two principals of one tenant already hold the
-- same key, because the tenant-wide key cannot hold both rows. Resolve those
-- rows by hand first. The re-run drops the INVALID index the failure leaves.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_index WHERE indexrelid = to_regclass('record_write_keys_tenant_key') AND NOT indisvalid) THEN
        DROP INDEX record_write_keys_tenant_key;
    END IF;
END
$$;
-- +goose StatementEnd
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS record_write_keys_tenant_key
    ON record_write_keys (tenant_id, idempotency_key);
SET lock_timeout = '5s';
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'record_write_keys'::regclass AND conname = 'record_write_keys_pkey') THEN
        ALTER TABLE record_write_keys ADD CONSTRAINT record_write_keys_pkey PRIMARY KEY USING INDEX record_write_keys_tenant_key;
    END IF;
END
$$;
-- +goose StatementEnd
RESET lock_timeout;
DROP INDEX CONCURRENTLY IF EXISTS record_write_keys_actor_key;
