package isolation

// tenantLimitsPaths are TICK-751's table: tenant B has a tenant_limits row
// (testdata/seed.sql), tenant A none. nava_app reads its own tenant's row for
// publish and writes it only on the operator route, in the target tenant's
// transaction; every path runs as tenant A and aims at tenant B.
func tenantLimitsPaths() []Path {
	sql := func(name, role string, expect Expect, query string, args ...any) Path {
		return Path{Name: name, Case: "CI-2", Role: role, SQL: &SQLCall{Query: query, Args: args, Expect: expect}}
	}
	return []Path{
		sql("tenant_limits: read every tenant's limits", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM tenant_limits"),
		sql("tenant_limits: read tenant B's limit by its key", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM tenant_limits WHERE tenant_id = $1", TenantB),
		sql("tenant_limits: raise tenant B's limit", "nava_app", Denied,
			"UPDATE tenant_limits SET filterable_fields_per_type = 10, updated_at = now(), updated_by = 'x' WHERE tenant_id = $1", TenantB),
		sql("tenant_limits: write a row for tenant B", "nava_app", Denied,
			"INSERT INTO tenant_limits (tenant_id, filterable_fields_per_type, updated_by) VALUES ($1, 10, 'x')", TenantB),
		sql("tenant_limits: delete tenant B's row", "nava_app", Denied,
			"DELETE FROM tenant_limits WHERE tenant_id = $1", TenantB),
		sql("tenant_limits: the worker reads tenant B's limit", "nava_worker", Denied,
			"SELECT tenant_id FROM tenant_limits WHERE tenant_id = $1", TenantB),
	}
}
