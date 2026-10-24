-- A row-level security policy as the db/ migrations write them.
CREATE POLICY tenant_isolation ON tickets
    USING (tenant_id = current_setting('app.tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);
SELECT set_config('app.tenant_id', '00000000-0000-0000-0000-000000000001', true);
