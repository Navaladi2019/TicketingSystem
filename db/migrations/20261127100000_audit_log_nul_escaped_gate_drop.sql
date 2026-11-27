-- TICK-1170 step 2 (ADR-004-E10 E10-6): diff_nul_escaped is now an envelope
-- member (internal/audit/chain, emitted only when true), the sealer reads the
-- column, and the verifier and the export carry it, all in this change. The
-- step 1 gate that refused diff_nul_escaped = true on INSERT is no longer
-- needed, so it goes. UPDATE stays refused by audit_log_guard_update.
--
-- DROP TRIGGER takes an ACCESS EXCLUSIVE lock on audit_log for the catalog
-- update only; lock_timeout fails the migration rather than queue the audit
-- writers behind it.

-- +goose Up

SET LOCAL lock_timeout = '5s';

DROP TRIGGER audit_log_nul_escaped_gate ON audit_log;
DROP FUNCTION audit_log_nul_escaped_gate();

-- +goose Down

SET LOCAL lock_timeout = '5s';

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
