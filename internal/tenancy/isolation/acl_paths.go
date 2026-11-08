package isolation

// SharedSharePrincipal is tenant A's principal 0a…3a1, with whom tenant B
// has also shared its record RecordB2 (testdata/seed.sql, P3-30).
// record_shares.principal_id has no foreign key, so a list join by principal
// id alone would meet tenant B's share.
const SharedSharePrincipal = "0a000000-0000-4000-8000-0000000003a1"

// aclPaths are the S10 authorization tables (P3-30, TICK-615): acl_versions
// and record_shares are per tenant, insert-only, and their composite foreign
// keys hold the tenant. The generic CI-2 checks cover them as every role; these
// name the lookups the evaluator (P3-31) and the list join (P3-34) make.
func aclPaths() []Path {
	sql := func(name, role string, expect Expect, query string, args ...any) Path {
		return Path{Name: name, Case: "CI-2", Role: role, SQL: &SQLCall{Query: query, Args: args, Expect: expect}}
	}
	return []Path{
		sql("acl_versions: read tenant B's ACL by its type", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM acl_versions WHERE entity_definition_id = $1", EntityDefinitionB1),
		sql("acl_versions: read every type's version 1", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM acl_versions WHERE version = 1"),
		sql("acl_versions: publish an ACL for tenant B", "nava_app", Denied,
			"INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks) VALUES ($1, $2, 3, '{}', '{}')",
			TenantB, EntityDefinitionB1),
		// The composite foreign key holds the tenant: tenant A cannot publish an
		// ACL for tenant B's type, though the policy passes the row.
		sql("acl_versions: publish an ACL in tenant A for tenant B's type", "nava_worker", Denied,
			"INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks) VALUES ($1, $2, 3, '{}', '{}')",
			TenantA, EntityDefinitionB1),
		sql("acl_versions: rewrite tenant B's ACL", "nava_app", Denied,
			"UPDATE acl_versions SET rules = '{}' WHERE tenant_id = $1", TenantB),
		sql("acl_versions: delete tenant B's ACL", "nava_worker", Denied,
			"DELETE FROM acl_versions WHERE tenant_id = $1", TenantB),
		sql("record_shares: list the records shared with a principal id in every tenant", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM record_shares WHERE principal_id = $1", SharedSharePrincipal),
		sql("record_shares: read tenant B's shares by record", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM record_shares WHERE record_id = $1", RecordB2),
		sql("record_shares: share a record for tenant B", "nava_app", Denied,
			"INSERT INTO record_shares (tenant_id, record_id, principal_id, principal_kind) VALUES ($1, $2, $3, 'user')",
			TenantB, RecordB2, NewID),
		// The composite foreign key holds the tenant: tenant A cannot share
		// tenant B's record.
		sql("record_shares: share tenant B's record from tenant A", "nava_worker", Denied,
			"INSERT INTO record_shares (tenant_id, record_id, principal_id, principal_kind) VALUES ($1, $2, $3, 'user')",
			TenantA, RecordB2, NewID),
		sql("record_shares: rewrite tenant B's shares", "nava_app", Denied,
			"UPDATE record_shares SET principal_id = $2 WHERE tenant_id = $1", TenantB, NewID),
		sql("record_shares: delete tenant B's shares", "nava_worker", Denied,
			"DELETE FROM record_shares WHERE tenant_id = $1", TenantB),
	}
}
