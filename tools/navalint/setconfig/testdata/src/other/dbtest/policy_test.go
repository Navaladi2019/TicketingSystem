package dbtest

const leakSQL = "SELECT set_config('app.tenant_id', $1, false)" // want `set_config outside internal/tenancy`

var _ = leakSQL
