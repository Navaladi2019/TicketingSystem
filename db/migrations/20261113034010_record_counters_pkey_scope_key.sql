-- S18 P6-05 (TICK-670), ADR-005 §4.2 step M3: record_counters_pkey becomes
-- (tenant_id, entity_definition_id, scope_key), promoted from S17's unique
-- index record_counters_scoped_key (20261112193200). Runbook:
-- docs/data/record-counters-scope-key.md.
--
-- THE POINT OF NO RETURN. After it no 2-column unique constraint exists, so
-- an R0 (pre-S17) binary's ON CONFLICT (tenant_id, entity_definition_id)
-- fails every create with 42P10. Run it (release R2) only once every API
-- and worker replica runs R1. Once a non-empty scope_key row exists, the
-- Down below refuses to run.
--
-- One ALTER TABLE: DROP CONSTRAINT drops the 2-column index, and ADD PRIMARY
-- KEY USING INDEX renames record_counters_scoped_key to record_counters_pkey
-- (the name S18's publish gate and every 23505 report keep). No build, no
-- scan: the index already exists, is unique and its columns are NOT NULL.
-- The FK record_counters_entity_definition_fkey is not touched; its
-- referencing lookup (tenant_id, entity_definition_id) is still the leading
-- prefix of the key. Measured 1.6 ms at 20k rows (ADR-005 spike).
--
-- Lock and retry (ADR-005 §4.2: lock_timeout 2s, retried up to 5 times with
-- backoff). The ALTER needs ACCESS EXCLUSIVE on record_counters, which every
-- create touches, so a queued request would stall every create behind it.
-- The retry is in the DO block, not the migrator: each attempt runs in a
-- subtransaction, so a 55P03 rolls back only that attempt and releases what
-- it took, then waits 0.2, 0.4, 0.8, 1.6, 3.2 s. At most 6 attempts, about
-- 18 s. When the last one times out the block raises 55006 (object_in_use),
-- which the migrator does not retry: its own 55P03 retry (TICK-579) would
-- otherwise multiply the attempts (the reason audit migration B keeps no
-- retry of its own). The transaction rolls back, the table is unchanged, no
-- version row is written; re-run the migrate job when the table is quieter.
--
-- Re-runnable: when the primary key already covers scope_key (a re-run, or
-- an operator who ran it by hand) the block does nothing. It refuses to run
-- when record_counters_scoped_key is missing or INVALID (M2 not applied, or
-- a failed CONCURRENTLY build left over): re-run the migrate job, whose M2
-- step repairs it, first.
--
-- After M3, do not re-run 20261112193200 (M2) by hand: its name is free
-- again, so it would build a second, redundant unique index on the same
-- columns (correct, but one more btree entry per counter row). The migrator
-- never re-runs a recorded migration. If it happens, DROP INDEX CONCURRENTLY
-- record_counters_scoped_key.
--
-- Grants, RLS, audit: unchanged (the policies are per row, the grants per
-- column; the table is exempt in db/audited-tables.yaml).

-- +goose Up

SET LOCAL lock_timeout = '2s';

-- +goose StatementBegin
DO $$
DECLARE
    retries CONSTANT int := 5;
    attempt int := 0;
BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint c
               JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY (c.conkey)
               WHERE c.conrelid = 'record_counters'::regclass AND c.contype = 'p' AND a.attname = 'scope_key') THEN
        RAISE NOTICE 'M3: record_counters_pkey already covers scope_key, nothing to do';
        RETURN;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_index
                   WHERE indexrelid = to_regclass('record_counters_scoped_key') AND indisunique AND indisvalid AND indisready) THEN
        RAISE EXCEPTION 'M3: record_counters_scoped_key is missing or INVALID'
            USING ERRCODE = 'object_not_in_prerequisite_state',
                  HINT = 'Re-run the migrate job so 20261112193200 (M2) builds it, then M3 runs.';
    END IF;
    LOOP
        BEGIN
            ALTER TABLE record_counters
                DROP CONSTRAINT record_counters_pkey,
                ADD CONSTRAINT record_counters_pkey PRIMARY KEY USING INDEX record_counters_scoped_key;
            RETURN;
        EXCEPTION WHEN lock_not_available THEN
            IF attempt >= retries THEN
                RAISE EXCEPTION 'M3: no ACCESS EXCLUSIVE lock on record_counters after % attempts (lock_timeout %)',
                    attempt + 1, current_setting('lock_timeout')
                    USING ERRCODE = 'object_in_use',
                          DETAIL = SQLERRM,
                          HINT = 'Nothing changed. Re-run the migrate job when fewer long transactions hold record_counters.';
            END IF;
            attempt := attempt + 1;
            RAISE NOTICE 'M3: lock_timeout on record_counters, retry % of % after % s', attempt, retries, 0.1 * 2 ^ attempt;
            PERFORM pg_sleep(0.1 * 2 ^ attempt);
        END;
    END LOOP;
END
$$;
-- +goose StatementEnd

-- +goose Down

-- Back to S17's schema: the 2-column primary key and record_counters_scoped_key
-- beside it. Only while every scope_key is '': a second scope of a type
-- would break the 2-column key, and an R1 binary never reads a scope row.
-- The check is a CHECK constraint added and dropped in this transaction,
-- because constraint validation sees every tenant's rows and the migrator,
-- under FORCE ROW LEVEL SECURITY, sees none. Builds both indexes under the
-- ACCESS EXCLUSIVE lock: the table holds one row per type and scope.

SET LOCAL lock_timeout = '2s';

LOCK TABLE record_counters IN ACCESS EXCLUSIVE MODE;

-- +goose StatementBegin
DO $$
BEGIN
    ALTER TABLE record_counters ADD CONSTRAINT record_counters_m3_down_default_scope CHECK (scope_key = '');
EXCEPTION WHEN check_violation THEN
    RAISE EXCEPTION 'M3 down refused: record_counters holds rows with a non-empty scope_key'
        USING ERRCODE = 'object_not_in_prerequisite_state',
              HINT = 'M3 is the point of no return once scoped counters exist (docs/data/record-counters-scope-key.md).';
END
$$;
-- +goose StatementEnd

ALTER TABLE record_counters
    DROP CONSTRAINT record_counters_m3_down_default_scope,
    DROP CONSTRAINT record_counters_pkey,
    ADD CONSTRAINT record_counters_pkey PRIMARY KEY (tenant_id, entity_definition_id);
CREATE UNIQUE INDEX record_counters_scoped_key ON record_counters (tenant_id, entity_definition_id, scope_key);
