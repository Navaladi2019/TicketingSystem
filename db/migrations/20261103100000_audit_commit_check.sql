-- P5-11 (TICK-428): ADR-004 decision 2's commit check, as a function any
-- audited table can attach, and the hand-off of the writers' audit_log
-- policies to the generator (R12; PLAN-005 TICK-449 grooming; S1 finding F1).
--
-- This migration attaches the trigger to no table. P5-13b (TICK-452)
-- attaches it to records, P5-32 (migration B) to entity_definitions and
-- entity_definition_versions, each with
--
--   CREATE CONSTRAINT TRIGGER <table>_audit_commit_check
--       AFTER INSERT OR DELETE OR UPDATE OF <audited_columns> ON <table>
--       DEFERRABLE INITIALLY DEFERRED
--       FOR EACH ROW EXECUTE FUNCTION audit_commit_check('<subject_type>', '<subject_id_column>');
--
-- and flips the table to attached: true in db/audited-tables.yaml; CI-A3
-- (internal/db/audited.Check) fails on any other shape, and on a trigger on
-- a table that is not marked attached. An insert-only table (no audited
-- columns) leaves out OR UPDATE OF.
--
-- At COMMIT the trigger requires at least one audit_log entry with the
-- written row's tenant_id, this transaction's txid and now(), the table's
-- subject_type and the row's subject id. The probe is the grooming's pinned
-- predicate
--   tenant_id = $1 AND occurred_at = now() AND txid = pg_current_xact_id()
--   AND subject_type = $2 AND subject_id = $3
-- with now() and pg_current_xact_id() read into plpgsql locals first (F1),
-- so the cached generic plan probes the full R2 key (an Index Only Scan) or
-- the R28 subject index, never a (tenant_id, occurred_at) prefix. That plan
-- also needs P5-10b's n_distinct = 1 on occurred_at; the plan and linearity
-- tests are TestP511CommitCheckPlan* (internal/db/dbtest).
--
-- SECURITY INVOKER: the writer's own SELECT grant (R12) and SELECT policy
-- decide what the probe sees, so a writer can only find its own tenant's
-- entries. Owned by nava_migrator; EXECUTE is granted to nobody (a trigger
-- function needs none at fire time). DELETE and the subject id of an
-- UPDATE come from the row as it was and as it is: the subject id column
-- is never updatable (CI-A3 requires every updatable column to be audited,
-- and the key columns are granted to no writer for UPDATE).
--
-- Policies: until now audit_log_select and audit_log_insert named nava_app
-- and nava_worker by hand. Their writer access now comes from
-- db/audited-tables.yaml, which nava-db migrate turns into
-- audit_log_writer_select and audit_log_writer_insert (canonical) and, for
-- nava_onboard, audit_log_onboard_select and audit_log_onboard_insert
-- (pinned provisioning predicate). The hand-written policies keep the
-- non-writer uses only: the sealer's read (nava_worker, R15) and the
-- erasure job (nava_erasure, R23). audit_log is empty until P5-13b, so the
-- moment between this migration and the generator's commit, when nava_app
-- has no audit_log policy, loses nothing.

-- +goose Up

-- +goose StatementBegin
CREATE FUNCTION audit_commit_check() RETURNS trigger
    LANGUAGE plpgsql SECURITY INVOKER SET search_path = pg_catalog, public AS $$
DECLARE
    v_tenant uuid;
    v_id     uuid;
    v_type   text        := TG_ARGV[0];
    v_txid   xid8        := pg_current_xact_id();
    v_at     timestamptz := now();
BEGIN
    IF TG_NARGS <> 2 THEN
        RAISE EXCEPTION 'audit_commit_check on %: want (subject_type, subject_id_column), got % arguments',
            TG_TABLE_NAME, TG_NARGS USING ERRCODE = '39P01';
    END IF;
    IF TG_OP = 'DELETE' THEN
        v_tenant := OLD.tenant_id;
        IF TG_ARGV[1] = 'id' THEN
            v_id := OLD.id;
        ELSE
            EXECUTE format('SELECT ($1).%I', TG_ARGV[1]) INTO v_id USING OLD;
        END IF;
    ELSE
        v_tenant := NEW.tenant_id;
        IF TG_ARGV[1] = 'id' THEN
            v_id := NEW.id;
        ELSE
            EXECUTE format('SELECT ($1).%I', TG_ARGV[1]) INTO v_id USING NEW;
        END IF;
    END IF;
    PERFORM 1 FROM audit_log
     WHERE tenant_id = v_tenant AND occurred_at = v_at AND txid = v_txid
       AND subject_type = v_type AND subject_id = v_id
     LIMIT 1;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'audit_missing: % % % written without an audit entry in this transaction',
            TG_TABLE_NAME, v_type, v_id
            USING ERRCODE = '23514', CONSTRAINT = TG_NAME, TABLE = TG_TABLE_NAME, SCHEMA = TG_TABLE_SCHEMA;
    END IF;
    RETURN NULL;
END
$$;
-- +goose StatementEnd

ALTER POLICY audit_log_select ON audit_log TO nava_worker, nava_erasure;
ALTER POLICY audit_log_insert ON audit_log TO nava_erasure;
