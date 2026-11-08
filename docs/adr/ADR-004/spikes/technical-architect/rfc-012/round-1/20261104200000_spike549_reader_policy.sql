-- SPIKE TICK-549, throwaway, never committed.
-- +goose Up
CREATE POLICY audit_log_select_reader ON audit_log FOR SELECT TO nava_audit_reader
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY entity_definition_versions_select_reader ON entity_definition_versions FOR SELECT TO nava_audit_reader
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
