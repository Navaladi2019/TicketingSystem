package tenancy

const setTenantSQL = "SELECT set_config('app.tenant_id', $1, true)"

// SQL returns the statement.
func SQL() string { return setTenantSQL }
