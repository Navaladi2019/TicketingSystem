package dbtest

const helperSQL = "SELECT set_config('app.tenant_id', $1, true)" // want `set_config outside internal/tenancy`
