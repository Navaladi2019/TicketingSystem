-- TICK-1032 (S31 fix; SP-1d decision, critique B1 on TICK-946, accepted by
-- the Solution Architect): records_org_unit becomes
-- (tenant_id, org_unit_id, entity_definition_id) WHERE org_unit_id IS NOT NULL.
-- The name stays records_org_unit, so the region bench and its plans name the
-- same index. Its (tenant_id, org_unit_id) prefix still backs the NO ACTION
-- check of records_org_unit_fkey.
--
-- Online, and records always has one of the two indexes:
--   1. A failed earlier build leaves an INVALID records_org_unit_new; drop it.
--   2. CREATE INDEX CONCURRENTLY records_org_unit_new, alone (no transaction).
--   3. One DO block, one transaction: DROP INDEX records_org_unit and rename
--      records_org_unit_new to records_org_unit. Other sessions see either
--      the old index under the name or the new one, never neither. The DROP
--      takes ACCESS EXCLUSIVE on records for a catalogue change only: no
--      scan, no build. lock_timeout 2s, retried up to 5 times with backoff
--      (0.2 to 3.2 s), as 20261113210000 does; the last timeout raises 55006,
--      which the migrator does not retry, so re-run the migrate job when
--      records is quieter. records_org_unit_new is kept for that re-run.
--
-- Re-runnable (TestNoTransactionMigrationsAreIdempotent): once
-- records_org_unit has the new definition, step 3 does nothing but drop a
-- records_org_unit_new that a re-run of step 2 built again. That rebuild is
-- the cost of a re-run after a completed swap; the migrator never re-runs a
-- recorded migration, so it only happens when the version row was not
-- written.

-- +goose NO TRANSACTION
-- +goose Up

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_index WHERE indexrelid = to_regclass('records_org_unit_new') AND NOT indisvalid) THEN
        DROP INDEX records_org_unit_new;
    END IF;
END
$$;
-- +goose StatementEnd
CREATE INDEX CONCURRENTLY IF NOT EXISTS records_org_unit_new
    ON records (tenant_id, org_unit_id, entity_definition_id) WHERE org_unit_id IS NOT NULL;

-- +goose StatementBegin
DO $$
DECLARE
    target CONSTANT text := 'CREATE INDEX records_org_unit ON public.records USING btree (tenant_id, org_unit_id, entity_definition_id) WHERE (org_unit_id IS NOT NULL)';
    retries CONSTANT int := 5;
    attempt int := 0;
BEGIN
    IF pg_get_indexdef(to_regclass('records_org_unit')) = target THEN
        DROP INDEX IF EXISTS records_org_unit_new;
        RETURN;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_index
                   WHERE indexrelid = to_regclass('records_org_unit_new') AND indisvalid AND indisready) THEN
        RAISE EXCEPTION 'records_org_unit swap: records_org_unit_new is missing or INVALID'
            USING ERRCODE = 'object_not_in_prerequisite_state',
                  HINT = 'Re-run the migrate job; its first two statements rebuild it.';
    END IF;
    PERFORM set_config('lock_timeout', '2s', true);
    LOOP
        BEGIN
            DROP INDEX IF EXISTS records_org_unit;
            ALTER INDEX records_org_unit_new RENAME TO records_org_unit;
            RETURN;
        EXCEPTION WHEN lock_not_available THEN
            IF attempt >= retries THEN
                RAISE EXCEPTION 'records_org_unit swap: no ACCESS EXCLUSIVE lock on records after % attempts (lock_timeout %)',
                    attempt + 1, current_setting('lock_timeout')
                    USING ERRCODE = 'object_in_use', DETAIL = SQLERRM,
                          HINT = 'Re-run the migrate job when records is quieter; records_org_unit_new is kept.';
            END IF;
            PERFORM pg_sleep(0.2 * power(2, attempt));
            attempt := attempt + 1;
        END;
    END LOOP;
END
$$;
-- +goose StatementEnd

-- +goose Down

-- The same steps back to (tenant_id, org_unit_id).
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_index WHERE indexrelid = to_regclass('records_org_unit_old') AND NOT indisvalid) THEN
        DROP INDEX records_org_unit_old;
    END IF;
END
$$;
-- +goose StatementEnd
CREATE INDEX CONCURRENTLY IF NOT EXISTS records_org_unit_old
    ON records (tenant_id, org_unit_id) WHERE org_unit_id IS NOT NULL;
-- +goose StatementBegin
DO $$
BEGIN
    PERFORM set_config('lock_timeout', '5s', true);
    DROP INDEX IF EXISTS records_org_unit;
    ALTER INDEX records_org_unit_old RENAME TO records_org_unit;
END
$$;
-- +goose StatementEnd
