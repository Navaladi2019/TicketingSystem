package catalog_test

const readSQL = "SELECT current_setting('app.tenant_id', true)"

const leakSQL = "SELECT set_config('app.tenant_id', $1, false)" // want `set_config outside internal/tenancy`

var _, _ = readSQL, leakSQL
