package tenancy_test

const tenantSQL = "SELECT current_setting('app.tenant_id', true)"

var _ = tenantSQL
