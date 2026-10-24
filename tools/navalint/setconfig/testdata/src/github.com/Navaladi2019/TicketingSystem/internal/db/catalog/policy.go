package catalog

// The deparsed predicate the catalogue gate compares policies against: a
// read, so not reported.
func TenantPred(col string) string {
	return "(" + col + " = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)"
}

const handRead = "SELECT current_setting( 'APP.TENANT_ID', true)"

// Writes are reported here as anywhere outside internal/tenancy.
const (
	setConfig = "SELECT set_config('app.tenant_id', $1, true)"                       // want `set_config outside internal/tenancy`
	setLocal  = `SET LOCAL app.tenant_id = 'x'`                                      // want `app.tenant_id outside internal/tenancy`
	setPlain  = "set app.tenant_id to 'x'"                                           // want `app.tenant_id outside internal/tenancy`
	readSet   = "SELECT current_setting('app.tenant_id'); SET app.tenant_id = 'x'"   // want `app.tenant_id outside internal/tenancy`
	copyRead  = "SELECT set_config('app.x', current_setting('app.tenant_id'), true)" // want `set_config outside internal/tenancy`
)
