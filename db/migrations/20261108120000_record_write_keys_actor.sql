-- TICK-626 (AR-TICK-480-01, carried onto D-5-3): an idempotency key belongs
-- to one principal, not to the whole tenant. Before this, record_write_keys
-- was keyed on (tenant_id, idempotency_key). Member B, writing under a key
-- member A already held, got A's write replayed or a 409. Either one told B
-- that the key existed. This migration adds the writer's principal and
-- backfills it. 20261108120100 makes the key unique per principal.
--
-- actor_principal_id is the principal the pipeline's audit entry names
-- (audit_log.actor_principal_id): NULL for a system writer with no
-- principal, as audit.Actor has it.
--
-- Backfill. Every row gets its writer, so a member who retries across the
-- deploy finds its own key and does not write twice. The source is
-- the write's primary audit entry, which audit_entry_id points at. It is
-- reached by (tenant_id, subject_type, subject_id) on audit_log_subject,
-- because audit_log's key leads with occurred_at. Keys written between
-- P5-13a and P5-13b point at audit_log_v0 entries, and that table is gone
-- (20261104100000). For those, a create falls back to records.created_by,
-- which is set from the same principal. An update key from that window has
-- no source left. It stays NULL, and the function counts it in a NOTICE.
-- Every stack that ran that window is a development stack.
--
-- Access path (ADR-004 R26). Temporary canonical policies for nava_migrator,
-- created and dropped in this one transaction. The only cross-tenant one is
-- SELECT on tenants, which lists the ids. Each tenant then runs under
-- set_config('app.tenant_id', ..., true). The R26.3 check below fails the
-- migration if a nava_migrator policy is left over.
--
-- Locks. ADD COLUMN of a nullable column with no default only changes the
-- catalogue. The UPDATE takes row locks on record_write_keys only, and
-- nothing reads it except replay. audit_log and records are only read.
--
-- Rollout. The migrate job runs before the new binary starts (compose
-- service_completed_successfully). A key written by an old binary after this
-- backfill and before the swap would keep a NULL actor. On a stack that
-- rolls without stopping writers, run SELECT record_write_keys_backfill_actor()
-- again as nava_migrator once the old binary is gone. It only fills NULLs.
-- Design and figures: docs/data/record-write-keys-per-principal.md.

-- +goose Up

ALTER TABLE record_write_keys ADD COLUMN actor_principal_id uuid;

-- +goose StatementBegin
CREATE FUNCTION record_write_keys_backfill_actor() RETURNS bigint
    LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path = pg_catalog, public AS $$
DECLARE
    tid uuid;
    filled bigint := 0;
    n bigint;
    unresolved bigint := 0;
BEGIN
    IF current_user <> 'nava_migrator' THEN
        RAISE EXCEPTION 'record_write_keys_backfill_actor: only nava_migrator may run it, not %', current_user
            USING ERRCODE = '42501';
    END IF;

    CREATE POLICY rwk_actor_tenants ON tenants FOR SELECT TO nava_migrator
        USING (true);
    CREATE POLICY rwk_actor_keys_select ON record_write_keys FOR SELECT TO nava_migrator
        USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
    CREATE POLICY rwk_actor_keys_update ON record_write_keys FOR UPDATE TO nava_migrator
        USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
        WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
    CREATE POLICY rwk_actor_log_select ON audit_log FOR SELECT TO nava_migrator
        USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
    CREATE POLICY rwk_actor_records_select ON records FOR SELECT TO nava_migrator
        USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

    FOR tid IN SELECT id FROM tenants ORDER BY id LOOP
        PERFORM set_config('app.tenant_id', tid::text, true);
        -- The primary audit entry: same id, same record, a principal actor.
        UPDATE record_write_keys k
        SET actor_principal_id = a.actor_principal_id
        FROM audit_log a
        WHERE k.tenant_id = tid
          AND k.actor_principal_id IS NULL
          AND a.tenant_id = k.tenant_id
          AND a.subject_type = 'record'
          AND a.subject_id = k.record_id
          AND a.id = k.audit_entry_id
          AND a.actor_principal_id IS NOT NULL;
        GET DIAGNOSTICS n = ROW_COUNT;
        filled := filled + n;
        -- A create whose entry was in audit_log_v0: the record's creator.
        UPDATE record_write_keys k
        SET actor_principal_id = r.created_by
        FROM records r
        WHERE k.tenant_id = tid
          AND k.actor_principal_id IS NULL
          AND k.action = 'create'
          AND r.tenant_id = k.tenant_id
          AND r.id = k.record_id
          AND r.created_by IS NOT NULL
          AND NOT EXISTS (
              SELECT 1 FROM audit_log a
              WHERE a.tenant_id = k.tenant_id
                AND a.subject_type = 'record'
                AND a.subject_id = k.record_id
                AND a.id = k.audit_entry_id);
        GET DIAGNOSTICS n = ROW_COUNT;
        filled := filled + n;
        -- What is still NULL: system writers (their entry names no
        -- principal) and update keys whose entry is gone.
        SELECT count(*) INTO n
        FROM record_write_keys k
        WHERE k.tenant_id = tid
          AND k.actor_principal_id IS NULL
          AND NOT EXISTS (
              SELECT 1 FROM audit_log a
              WHERE a.tenant_id = k.tenant_id
                AND a.subject_type = 'record'
                AND a.subject_id = k.record_id
                AND a.id = k.audit_entry_id);
        unresolved := unresolved + n;
    END LOOP;
    PERFORM set_config('app.tenant_id', '', true);

    DROP POLICY rwk_actor_records_select ON records;
    DROP POLICY rwk_actor_log_select ON audit_log;
    DROP POLICY rwk_actor_keys_update ON record_write_keys;
    DROP POLICY rwk_actor_keys_select ON record_write_keys;
    DROP POLICY rwk_actor_tenants ON tenants;

    IF unresolved > 0 THEN
        RAISE NOTICE 'record_write_keys_backfill_actor: % key(s) have no audit entry and no creator; left NULL', unresolved;
    END IF;
    RETURN filled;
END
$$;
-- +goose StatementEnd

SELECT record_write_keys_backfill_actor();

-- R26.3: no nava_migrator policy outlives the migration.
-- +goose StatementBegin
DO $$
DECLARE
    left_over text;
BEGIN
    SELECT string_agg(format('%s.%I', polrelid::regclass, polname), ', ') INTO left_over
    FROM pg_policy WHERE 'nava_migrator'::regrole = ANY (polroles);
    IF left_over IS NOT NULL THEN
        RAISE EXCEPTION 'record_write_keys actor backfill: nava_migrator policies left over: %', left_over;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down

DROP FUNCTION IF EXISTS record_write_keys_backfill_actor();
ALTER TABLE record_write_keys DROP COLUMN IF EXISTS actor_principal_id;
