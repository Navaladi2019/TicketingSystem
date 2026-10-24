package dbtest

const leakSQL = "SELECT set_config('app.tenant_id', $1, false)"

var _ = leakSQL
