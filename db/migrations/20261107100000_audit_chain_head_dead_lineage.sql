-- P5-34f (TICK-590, ADR-004 R17): the dead-lineage gauge. nava-purge, as
-- nava_purge, counts the heads whose next_seal_at is more than 10 min
-- overdue, across tenants, and exports the count (audit_dead_lineages, no
-- tenant label). The policy is R17's predicate, the same one the chain:reseed
-- gate tests (internal/audit/lineage.Predicate), so nava_purge sees exactly
-- the lineages a reseed may restart and nothing else. Its grant is SELECT on
-- tenant_id and next_seal_at only (db/grants.yaml): no hash, seq, version or
-- lease, and no write.
--
-- A policy only: a catalogue update, no rewrite, no index (one row per
-- tenant). CREATE POLICY takes an ACCESS EXCLUSIVE lock on the head, and a
-- seal run holds head row locks for up to 10,000 entries, so lock_timeout
-- fails the migration after 5 s rather than queueing every seal behind it;
-- rerun it (TICK-590 Postgres review R1). Registered in
-- internal/db/catalog.AllowList, pinned to
-- lineage.PolicyQual.

-- +goose Up

SET LOCAL lock_timeout = '5s';

CREATE POLICY audit_chain_head_purge_select ON audit_chain_head FOR SELECT TO nava_purge
    USING (next_seal_at < now() - interval '10 minutes');

-- +goose Down

SET LOCAL lock_timeout = '5s';

DROP POLICY audit_chain_head_purge_select ON audit_chain_head;
