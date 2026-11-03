-- TICK-498 (AR-TICK-480-01): audit_log's idempotency key is unique per
-- principal, not per tenant. TICK-480 scoped the replay lookup
-- (store.AuditByKey) to the actor, but the unique index stayed on
-- (tenant_id, idempotency_key): member B writing under member A's key got a
-- 409, which told B the key existed in the tenant. Now each principal has its
-- own key space, so B's write succeeds and reveals nothing.
--
-- NULLS NOT DISTINCT: a system writer with no principal (actor_id NULL) still
-- conflicts with itself, so its retry replays instead of writing twice.
-- Column order (tenant_id, idempotency_key, actor_id), not
-- (tenant_id, actor_id, idempotency_key): AuditByKey matches the actor with
-- IS NOT DISTINCT FROM, which a btree cannot search on, so the key must come
-- before the actor for the lookup to stay a two-column index probe.
-- Uniqueness is the same either way. Figures:
-- docs/data/audit-idempotency-per-actor.md.
--
-- Online: CREATE UNIQUE INDEX CONCURRENTLY under a temporary name, then DROP
-- INDEX CONCURRENTLY of the old one, then a rename (SHARE UPDATE EXCLUSIVE,
-- catalogue only). The new index is strictly finer than the old one, so the
-- existing rows cannot violate it. Between the drop and the rename a
-- concurrent duplicate is reported under the temporary name; Write turns it
-- into a 500 instead of a 409 for that instant, and nothing is written twice.
-- The name stays audit_log_idempotency_key, the one store.InsertAudit maps to
-- ErrDuplicateKey.
--
-- Re-runnable after a partial failure (TestNoTransactionMigrationsAreIdempotent):
-- an INVALID index from a failed CONCURRENTLY build is dropped first, and
-- each later step is IF EXISTS.
--
-- An ON CONFLICT target must now name all three columns and the predicate:
-- ON CONFLICT (tenant_id, idempotency_key, actor_id) WHERE idempotency_key IS NOT NULL.

-- +goose NO TRANSACTION
-- +goose Up

DROP INDEX CONCURRENTLY IF EXISTS audit_log_idempotency_key_actor;
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS audit_log_idempotency_key_actor
    ON audit_log (tenant_id, idempotency_key, actor_id) NULLS NOT DISTINCT
    WHERE idempotency_key IS NOT NULL;
DROP INDEX CONCURRENTLY IF EXISTS audit_log_idempotency_key;
ALTER INDEX IF EXISTS audit_log_idempotency_key_actor RENAME TO audit_log_idempotency_key;

-- +goose Down

-- Fails on the CREATE, leaving the per-actor index in place, if two actors of
-- one tenant already hold the same key: the tenant-wide index cannot hold
-- both rows. Those rows must be resolved by hand first; the INVALID
-- audit_log_idempotency_key_tenant the failure leaves is dropped by the
-- re-run.
DROP INDEX CONCURRENTLY IF EXISTS audit_log_idempotency_key_tenant;
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS audit_log_idempotency_key_tenant
    ON audit_log (tenant_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
DROP INDEX CONCURRENTLY IF EXISTS audit_log_idempotency_key;
ALTER INDEX IF EXISTS audit_log_idempotency_key_tenant RENAME TO audit_log_idempotency_key;
