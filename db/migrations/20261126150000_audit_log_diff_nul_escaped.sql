-- TICK-1170 (unblocks TICK-1121 criterion 4; ADR-004-E10, TICK-1169):
-- audit_log.diff_nul_escaped says the previous schema in this entry's diff had
-- each NUL character escaped to the text \u0000 before the diff was built,
-- because jsonb refuses NUL. It sits next to diff_truncated and is written
-- through the same R10 INSERT path (audit.Entry.DiffNULEscaped).
--
-- Online: a constant DEFAULT makes ADD COLUMN metadata-only on the parent and
-- on every partition (PostgreSQL 11+ fast default): no rewrite, no scan, the
-- ACCESS EXCLUSIVE lock is held for the catalog update only. lock_timeout
-- fails the migration rather than queue the audit writers behind it. Every
-- existing row reads false, which is true of every row written before the
-- TICK-1121 escape exists.
--
-- R11: the guard's envelope comparison gains the column, so neither the seal
-- nor the erase transition can flip it, whatever TICK-1169 decides about the
-- hash. Grants (db/grants.yaml, internal/db/audited): INSERT for every R10
-- writer, SELECT wherever diff_truncated is readable. Whether the flag is an
-- envelope member is TICK-1169's verdict and a later change (TICK-1170 step 2).
--
-- ADR-004-E10 E10-6 (ACCEPTED): until step 2 puts the member in the envelope,
-- the sealer's read and the verifier, a row sealed with the flag true would
-- read tampered forever. audit_log_nul_escaped_gate refuses such an INSERT
-- with 23514; UPDATE is already refused by the guard below. Step 2's
-- migration drops the gate. A trigger, not a CHECK: ADD CONSTRAINT CHECK
-- scans every partition under ACCESS EXCLUSIVE; the trigger looks at new rows
-- only.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE audit_log ADD COLUMN diff_nul_escaped boolean NOT NULL DEFAULT false;

-- +goose StatementBegin
CREATE FUNCTION audit_log_nul_escaped_gate() RETURNS trigger
    LANGUAGE plpgsql SECURITY INVOKER SET search_path = pg_catalog, public AS $$
BEGIN
    IF NEW.diff_nul_escaped THEN
        RAISE EXCEPTION 'audit_log: diff_nul_escaped = true is refused until it is an envelope member (ADR-004-E10 E10-6)'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER audit_log_nul_escaped_gate BEFORE INSERT ON audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_nul_escaped_gate();

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION audit_log_guard_update() RETURNS trigger
    LANGUAGE plpgsql SECURITY INVOKER SET search_path = pg_catalog, public AS $$
DECLARE
    same_envelope boolean :=
        (NEW.tenant_id, NEW.id, NEW.occurred_at, NEW.txid, NEW.domain, NEW.action, NEW.subject_type, NEW.subject_id,
         NEW.entity_definition_id, NEW.version_before, NEW.version_after, NEW.actor_principal_id, NEW.actor_system,
         NEW.source, NEW.source_ref, NEW.change_set_id, NEW.causation_id, NEW.causation_depth, NEW.request_id,
         NEW.backfilled, NEW.content_sha256, NEW.diff_sha256, NEW.diff_truncated, NEW.diff_nul_escaped)
        IS NOT DISTINCT FROM
        (OLD.tenant_id, OLD.id, OLD.occurred_at, OLD.txid, OLD.domain, OLD.action, OLD.subject_type, OLD.subject_id,
         OLD.entity_definition_id, OLD.version_before, OLD.version_after, OLD.actor_principal_id, OLD.actor_system,
         OLD.source, OLD.source_ref, OLD.change_set_id, OLD.causation_id, OLD.causation_depth, OLD.request_id,
         OLD.backfilled, OLD.content_sha256, OLD.diff_sha256, OLD.diff_truncated, OLD.diff_nul_escaped);
BEGIN
    IF current_user = 'nava_worker' AND same_envelope
       AND OLD.sealed_at IS NULL AND OLD.seq IS NULL AND OLD.prev_hash IS NULL AND OLD.entry_hash IS NULL
       AND NEW.sealed_at IS NOT NULL AND NEW.seq IS NOT NULL AND NEW.prev_hash IS NOT NULL AND NEW.entry_hash IS NOT NULL
       AND NEW.diff::text IS NOT DISTINCT FROM OLD.diff::text AND NEW.diff_salt IS NOT DISTINCT FROM OLD.diff_salt
       AND (NEW.erased_at, NEW.erasure_ref) IS NOT DISTINCT FROM (OLD.erased_at, OLD.erasure_ref) THEN
        RETURN NEW; -- seal
    END IF;
    IF current_user = 'nava_erasure' AND same_envelope
       AND OLD.erased_at IS NULL AND OLD.diff IS NOT NULL
       AND NEW.diff IS NULL AND NEW.diff_salt IS NULL AND NEW.erased_at IS NOT NULL AND NEW.erasure_ref IS NOT NULL
       AND (NEW.seq, NEW.prev_hash, NEW.entry_hash, NEW.sealed_at)
           IS NOT DISTINCT FROM (OLD.seq, OLD.prev_hash, OLD.entry_hash, OLD.sealed_at) THEN
        RETURN NEW; -- erase
    END IF;
    RAISE EXCEPTION 'audit_log: update by % is neither the seal nor the erase transition', current_user
        USING ERRCODE = '42501';
END
$$;
-- +goose StatementEnd

-- +goose Down

SET LOCAL lock_timeout = '5s';

DROP TRIGGER audit_log_nul_escaped_gate ON audit_log;
DROP FUNCTION audit_log_nul_escaped_gate();

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION audit_log_guard_update() RETURNS trigger
    LANGUAGE plpgsql SECURITY INVOKER SET search_path = pg_catalog, public AS $$
DECLARE
    same_envelope boolean :=
        (NEW.tenant_id, NEW.id, NEW.occurred_at, NEW.txid, NEW.domain, NEW.action, NEW.subject_type, NEW.subject_id,
         NEW.entity_definition_id, NEW.version_before, NEW.version_after, NEW.actor_principal_id, NEW.actor_system,
         NEW.source, NEW.source_ref, NEW.change_set_id, NEW.causation_id, NEW.causation_depth, NEW.request_id,
         NEW.backfilled, NEW.content_sha256, NEW.diff_sha256, NEW.diff_truncated)
        IS NOT DISTINCT FROM
        (OLD.tenant_id, OLD.id, OLD.occurred_at, OLD.txid, OLD.domain, OLD.action, OLD.subject_type, OLD.subject_id,
         OLD.entity_definition_id, OLD.version_before, OLD.version_after, OLD.actor_principal_id, OLD.actor_system,
         OLD.source, OLD.source_ref, OLD.change_set_id, OLD.causation_id, OLD.causation_depth, OLD.request_id,
         OLD.backfilled, OLD.content_sha256, OLD.diff_sha256, OLD.diff_truncated);
BEGIN
    IF current_user = 'nava_worker' AND same_envelope
       AND OLD.sealed_at IS NULL AND OLD.seq IS NULL AND OLD.prev_hash IS NULL AND OLD.entry_hash IS NULL
       AND NEW.sealed_at IS NOT NULL AND NEW.seq IS NOT NULL AND NEW.prev_hash IS NOT NULL AND NEW.entry_hash IS NOT NULL
       AND NEW.diff::text IS NOT DISTINCT FROM OLD.diff::text AND NEW.diff_salt IS NOT DISTINCT FROM OLD.diff_salt
       AND (NEW.erased_at, NEW.erasure_ref) IS NOT DISTINCT FROM (OLD.erased_at, OLD.erasure_ref) THEN
        RETURN NEW; -- seal
    END IF;
    IF current_user = 'nava_erasure' AND same_envelope
       AND OLD.erased_at IS NULL AND OLD.diff IS NOT NULL
       AND NEW.diff IS NULL AND NEW.diff_salt IS NULL AND NEW.erased_at IS NOT NULL AND NEW.erasure_ref IS NOT NULL
       AND (NEW.seq, NEW.prev_hash, NEW.entry_hash, NEW.sealed_at)
           IS NOT DISTINCT FROM (OLD.seq, OLD.prev_hash, OLD.entry_hash, OLD.sealed_at) THEN
        RETURN NEW; -- erase
    END IF;
    RAISE EXCEPTION 'audit_log: update by % is neither the seal nor the erase transition', current_user
        USING ERRCODE = '42501';
END
$$;
-- +goose StatementEnd

ALTER TABLE audit_log DROP COLUMN diff_nul_escaped;
