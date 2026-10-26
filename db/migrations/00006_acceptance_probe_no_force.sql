-- THROWAWAY (TICK-44 acceptance): a tenant table with ENABLE but without
-- FORCE ROW LEVEL SECURITY. CI must fail on it (ADR-001 CI-1). Do not merge.

-- +goose Up
CREATE TABLE acceptance_probe (
    tenant_id uuid NOT NULL REFERENCES tenants (id),
    id        uuid NOT NULL,
    PRIMARY KEY (tenant_id, id)
);
ALTER TABLE acceptance_probe ENABLE ROW LEVEL SECURITY;
CREATE POLICY acceptance_probe_select ON acceptance_probe FOR SELECT TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- +goose Down
DROP TABLE acceptance_probe;
