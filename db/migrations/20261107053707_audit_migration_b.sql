-- P5-32 (TICK-442): migration B (ADR-004 R25 B half, R26, R27; R16
-- seeding). One transaction, which:
--
--   1. takes the six R27 ACCESS EXCLUSIVE locks up front, in the fixed order
--      tenants, entity_definitions, entity_definition_versions, audit_log,
--      audit_chain_head, outbox, under lock_timeout; the migrator retries
--      a timeout (TICK-579, below);
--   2. re-runs migration A's backfill (audit_backfill_s8, 20261103120000).
--      Its NOT EXISTS dedupe adds only the versions published between A and
--      the deploy that made the S8 publish path call audit.Append (P5-14),
--      so every S8 version has exactly one entry (CI-A9);
--   3. attaches the audit_commit_check constraint trigger to
--      entity_definitions and entity_definition_versions. The UPDATE OF lists
--      are db/audited-tables.yaml's audited_columns (CI-A3);
--      entity_definition_versions is insert-only, so it has no UPDATE event.
--      From here a raw write to either table fails at COMMIT with
--      audit_missing (CI-A2);
--   4. seeds, for every tenant that has none, the audit_chain_head row and the
--      first audit.seal start row, the same two rows onboarding writes
--      (onboard.PGStore.SeedAuditChain, P5-33, ADR-001-A6.2): head at the
--      column defaults (version 1, seq 0, zero entry_hash, seal_interval_s 10)
--      with next_seal_outbox_id = the start row and next_seal_at = now(); the
--      start row in the C5 shape of outbox.EnqueueSeal(ctx, tx, 1) with no
--      option: kind audit.seal, kind_version 1, dedupe_key audit.seal:1,
--      target_key audit.seal:<own id>, target_seq 1, causation_id = own id,
--      depth 0, payload {}, no trace context, available_at = now(), no
--      workflow_ref. Both ids are UUIDv7. A tenant onboarding already seeded
--      keeps its rows (the head's primary key, ON CONFLICT DO NOTHING); a
--      tenant still provisioning gets its rows here, and onboarding's later
--      SeedAuditChain then writes nothing. TestP532SealRowMatchesEnqueueSeal
--      compares the row with EnqueueSeal's own output (TICK-449 grooming);
--   5. asserts no nava_migrator policy is left (R26.3).
--
-- Access path (R26). Each function, run by nava_migrator in this
-- transaction (no NO TRANSACTION directive), creates its temporary policies,
-- sets app.tenant_id with is_local = true per tenant, and drops them before it
-- returns. The only cross-tenant policy is FOR SELECT TO nava_migrator
-- USING (true) on tenants, to list ids; the head and outbox inserts go
-- through canonical tenant policies. An error anywhere rolls everything back,
-- policies included.
--
-- Locks (R27, R-A6). Every later statement needs at most the lock already
-- held, so nothing here can wait once step 1 holds. The retry has one owner,
-- the migrator (internal/db/migrate, TICK-579, ADR-004-E5): a LOCK TABLE
-- that waits past lock_timeout fails with 55P03, the whole transaction rolls
-- back and releases whatever part of the set it took, and the runner re-runs
-- B after a doubling backoff (1 s to 30 s, 10 attempts, about 3.4 min worst
-- case before the run fails). So a retry never holds a lock while it waits.
-- B keeps no retry of its own: with both, the attempts multiplied (5 x 10,
-- about 8.4 min). The locks block tenant resolution, S8, audit writes and the
-- relay until COMMIT: measured on PG 16.10, 0.48 s for 1,000 tenants and
-- 30,093 versions, 4.29 s for 10,000 and 300,972, about 0.13 ms per tenant
-- plus 9.5 us per version (docs/data/audit-migration-b-bench/RESULT.md).

-- +goose Up

SET LOCAL lock_timeout = '5s';

-- R27: all six locks, in order, before any other work. A timeout fails the
-- migration with 55P03, and the migrator retries it.
LOCK TABLE tenants, entity_definitions, entity_definition_versions, audit_log, audit_chain_head, outbox
    IN ACCESS EXCLUSIVE MODE;

-- R25: the re-run. The number it added is in the migration log.
SELECT audit_backfill_s8();

CREATE CONSTRAINT TRIGGER entity_definitions_audit_commit_check
    AFTER INSERT OR DELETE OR UPDATE OF current_version, updated_at ON entity_definitions
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION audit_commit_check('entity_definition', 'id');

CREATE CONSTRAINT TRIGGER entity_definition_versions_audit_commit_check
    AFTER INSERT OR DELETE ON entity_definition_versions
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION audit_commit_check('entity_definition', 'definition_id');

-- RFC 9562 UUIDv7: 48-bit Unix milliseconds of clock_timestamp(), then the
-- random bits of gen_random_uuid() (which already carry the RFC variant),
-- version nibble 7. Go's uuid.NewV7 also fills rand_a from a sub-millisecond
-- counter; RFC 9562 allows either, and readers only sort by the time prefix.
-- +goose StatementBegin
CREATE FUNCTION audit_seed_uuidv7() RETURNS uuid
    LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path = pg_catalog AS $$
DECLARE
    b bytea := uuid_send(gen_random_uuid());
BEGIN
    b := overlay(b PLACING substring(int8send(floor(extract(epoch FROM clock_timestamp()) * 1000)::bigint) FROM 3)
                 FROM 1 FOR 6);
    b := set_byte(b, 6, (get_byte(b, 6) & 15) | 112); -- version 7
    RETURN encode(b, 'hex')::uuid;
END
$$;
-- +goose StatementEnd

-- R16 seeding for existing tenants. Returns the number of tenants seeded.
-- Like audit_backfill_s8, only nava_migrator may run it (it creates
-- policies), it has no EXECUTE grant, and a re-run seeds nothing new.
-- +goose StatementBegin
CREATE FUNCTION audit_seed_chain_heads() RETURNS bigint
    LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path = pg_catalog, public AS $$
DECLARE
    tid uuid;
    head_id uuid;
    seal_id uuid;
    seeded bigint := 0;
    n bigint;
BEGIN
    IF current_user <> 'nava_migrator' THEN
        RAISE EXCEPTION 'audit_seed_chain_heads: only nava_migrator may seed chain heads, not %', current_user
            USING ERRCODE = '42501';
    END IF;

    CREATE POLICY audit_seed_tenants ON tenants FOR SELECT TO nava_migrator
        USING (true);
    CREATE POLICY audit_seed_head_insert ON audit_chain_head FOR INSERT TO nava_migrator
        WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
    CREATE POLICY audit_seed_outbox_insert ON outbox FOR INSERT TO nava_migrator
        WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

    FOR tid IN SELECT id FROM tenants ORDER BY id LOOP
        PERFORM set_config('app.tenant_id', tid::text, true);
        head_id := audit_seed_uuidv7();
        seal_id := audit_seed_uuidv7();
        -- onboard.insertHeadSQL. The head is the idempotency record: if the
        -- tenant has one, it has its lineage, and nothing else is written.
        INSERT INTO audit_chain_head (tenant_id, id, next_seal_outbox_id, next_seal_at)
        VALUES (tid, head_id, seal_id, now())
        ON CONFLICT DO NOTHING;
        GET DIAGNOSTICS n = ROW_COUNT;
        CONTINUE WHEN n = 0;
        -- onboard.insertSealStartSQL, which equals outbox.insertSealSQL's
        -- column list. No ON CONFLICT: a seal row with no head is corruption,
        -- and the unique dedupe key fails the migration on it.
        INSERT INTO outbox (id, tenant_id, kind, kind_version, dedupe_key, target_key, target_seq,
            causation_id, causation_depth, payload, trace_context)
        VALUES (seal_id, tid, 'audit.seal', 1, 'audit.seal:1', 'audit.seal:' || seal_id::text, 1,
            seal_id, 0, '{}', NULL);
        seeded := seeded + 1;
    END LOOP;
    PERFORM set_config('app.tenant_id', '', true);

    DROP POLICY audit_seed_outbox_insert ON outbox;
    DROP POLICY audit_seed_head_insert ON audit_chain_head;
    DROP POLICY audit_seed_tenants ON tenants;
    RETURN seeded;
END
$$;
-- +goose StatementEnd

SELECT audit_seed_chain_heads();

-- R26.3: no nava_migrator policy outlives the migration.
-- +goose StatementBegin
DO $$
DECLARE
    left_over text;
BEGIN
    SELECT string_agg(format('%s.%I', polrelid::regclass, polname), ', ') INTO left_over
    FROM pg_policy WHERE 'nava_migrator'::regrole = ANY (polroles);
    IF left_over IS NOT NULL THEN
        RAISE EXCEPTION 'migration B: nava_migrator policies left over: %', left_over;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- The seeded heads and start rows stay: they are the rows onboarding writes,
-- and removing a live lineage would stop the tenant's sealer.
DROP FUNCTION audit_seed_chain_heads();
DROP FUNCTION audit_seed_uuidv7();
DROP TRIGGER entity_definition_versions_audit_commit_check ON entity_definition_versions;
DROP TRIGGER entity_definitions_audit_commit_check ON entity_definitions;
