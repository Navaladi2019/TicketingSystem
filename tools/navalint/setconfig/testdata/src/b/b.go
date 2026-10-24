// Package b is the deparsed catalogue predicate outside internal/db/catalog:
// the read exemption is by exact package path.
package b

const pred = "(tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)" // want `app.tenant_id outside internal/tenancy`

const setLocal = `SET LOCAL app.tenant_id = 'x'` // want `app.tenant_id outside internal/tenancy`

var _, _ = pred, setLocal
