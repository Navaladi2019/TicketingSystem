package isolation

const emptyTenantSQL = "SELECT set_config('app.tenant_id', '', true)"

var _ = emptyTenantSQL
