package a

import "fmt"

const prefix = "set_"

const scoped = "SELECT set_config('app.tenant_id', $1, true)" // want `set_config outside internal/tenancy`

func calls() {
	_ = "SELECT set_config('app.tenant_id', $1, false)" // want `set_config outside internal/tenancy`
	_ = `SET LOCAL app.tenant_id = 'x'`                 // want `app.tenant_id outside internal/tenancy`
	_ = "select SET_CONFIG('x', $1, true)"              // want `set_config outside internal/tenancy`
	_ = "SELECT current_setting('APP.TENANT_ID')"       // want `app.tenant_id outside internal/tenancy`
	_ = "set_" + "config"                               // want `set_config outside internal/tenancy`
	_ = (prefix + "config")                             // want `set_config outside internal/tenancy`
	_ = fmt.Sprintf("%s", "app.tenant_id")              // want `app.tenant_id outside internal/tenancy`
	x := "y"
	_ = "set_config(" + x // want `set_config outside internal/tenancy`
	_ = scoped
}

// Not constants: built at run time, so not seen (defence in depth only).
func runtime(a, b string) string { return a + b }

var _ = runtime("set_", "config")

// Clean strings and comments are fine: set_config in a comment is not code.
const clean = "SELECT 1"
