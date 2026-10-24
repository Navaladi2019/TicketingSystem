package isolation

const registeredSQL = "SELECT set_config('app.tenant_id', $1, true)" // want `set_config outside internal/tenancy`
