package isolation

// The S18 number-format fixtures (testdata/seed.sql, P6-06).
const (
	// numberFormatA1 is tenant A's format; numberFormatB1 is tenant B's
	// format under the same key, at versions 1 and 2.
	numberFormatA1 = "0a000000-0000-4000-8000-0000000012a1"
	numberFormatB1 = "0b000000-0000-4000-8000-0000000012b1"
)

// numberFormatPaths are the ADR-005 §5 rows "required on the implementation PR
// that adds the format tables" (P6-06, TICK-671; §3.7 C8, C9): number_formats,
// number_format_versions and numbering_epochs are per tenant, the versions
// and epochs insert-only, and the pinned link on entity_definition_versions is
// a composite foreign key that leads with tenant_id. The generic CI-2 checks
// cover the three tables as every role; these name the lookups the format
// admin routes and the definition publish make.
func numberFormatPaths() []Path {
	sql := func(name, role string, expect Expect, query string, args ...any) Path {
		return Path{Name: name, Case: "CI-2", Role: role, SQL: &SQLCall{Query: query, Args: args, Expect: expect}}
	}
	// linkVersion publishes tenant A's ticket version 3 pinned to
	// (tenant_id $1, format $2, version 1).
	const linkVersion = `INSERT INTO entity_definition_versions (tenant_id, id, definition_id, version, schema, schema_sha256,
		fixed_columns, states, compat, published_by, number_format_id, number_format_version, number_compiled)
		VALUES ($1, $3, $4, 3, '{}'::json, sha256(convert_to('{}', 'UTF8')), '{}', '{}', 'compatible', $5, $2, 1, '{}')`
	var paths []Path
	for _, role := range []string{"nava_app", "nava_worker"} {
		paths = append(paths,
			sql("number_formats: read tenant B's format by id as "+role, role, OwnRowsOnly,
				"SELECT tenant_id FROM number_formats WHERE id = $1", numberFormatB1),
			sql("number_formats: look a shared key up in every tenant as "+role, role, OwnRowsOnly,
				"SELECT tenant_id FROM number_formats WHERE key = 'monthly'"),
			sql("number_format_versions: read tenant B's versions by format id as "+role, role, OwnRowsOnly,
				"SELECT tenant_id FROM number_format_versions WHERE format_id = $1", numberFormatB1),
			sql("number_format_versions: read tenant B's version by its full key as "+role, role, OwnRowsOnly,
				"SELECT tenant_id FROM number_format_versions WHERE tenant_id = $1 AND format_id = $2 AND version = 1",
				TenantB, numberFormatB1),
			sql("numbering_epochs: read tenant B's epochs by type as "+role, role, OwnRowsOnly,
				"SELECT tenant_id FROM numbering_epochs WHERE entity_definition_id = $1", EntityDefinitionB1),
			sql("numbering_epochs: look a shared compiled hash up in every tenant as "+role, role, OwnRowsOnly,
				"SELECT tenant_id FROM numbering_epochs WHERE compiled_sha256 = sha256('monthly'::bytea)"),
			sql("entity_definition_versions: list the versions that pin tenant B's format as "+role, role, OwnRowsOnly,
				"SELECT tenant_id FROM entity_definition_versions WHERE number_format_id = $1", numberFormatB1),
		)
	}
	return append(paths,
		// Inserts for tenant B: the policies' WITH CHECK refuses nava_app; nava_worker
		// has no INSERT grant at all, even for its own tenant.
		sql("number_formats: create a format for tenant B", "nava_app", Denied,
			"INSERT INTO number_formats (tenant_id, id, key, current_version) VALUES ($1, $2, 'iso_new', 1)", TenantB, NewID),
		sql("number_format_versions: publish a version for tenant B", "nava_app", Denied,
			`INSERT INTO number_format_versions (tenant_id, format_id, version, document, compiled, compiled_sha256, created_by)
			 VALUES ($1, $2, 3, '{}', '{}', sha256('iso'::bytea), $3)`, TenantB, numberFormatB1, PrincipalB1),
		sql("numbering_epochs: take an epoch for tenant B's type", "nava_app", Denied,
			"INSERT INTO numbering_epochs (tenant_id, entity_definition_id, epoch, compiled_sha256, literal_head) VALUES ($1, $2, 3, sha256('iso'::bytea), 'TCK/')",
			TenantB, EntityDefinitionB1),
		sql("number_formats: worker creates a format", "nava_worker", Denied,
			"INSERT INTO number_formats (tenant_id, id, key, current_version) VALUES ($1, $2, 'iso_new', 1)", TenantA, NewID),
		sql("number_format_versions: worker publishes a version", "nava_worker", Denied,
			`INSERT INTO number_format_versions (tenant_id, format_id, version, document, compiled, compiled_sha256, created_by)
			 VALUES ($1, $2, 2, '{}', '{}', sha256('iso'::bytea), $3)`, TenantA, numberFormatA1, principalA1),
		sql("numbering_epochs: worker takes an epoch", "nava_worker", Denied,
			"INSERT INTO numbering_epochs (tenant_id, entity_definition_id, epoch, compiled_sha256, literal_head) VALUES ($1, $2, 2, sha256('iso'::bytea), 'TCK/')",
			TenantA, definitionA1),
		// The composite foreign keys hold the tenant, though the policy passes
		// the row: tenant A cannot hang a format version or an epoch off tenant
		// B's format or type.
		sql("number_format_versions: publish in tenant A a version of tenant B's format", "nava_app", Denied,
			`INSERT INTO number_format_versions (tenant_id, format_id, version, document, compiled, compiled_sha256, created_by)
			 VALUES ($1, $2, 3, '{}', '{}', sha256('iso'::bytea), $3)`, TenantA, numberFormatB1, principalA1),
		sql("numbering_epochs: take an epoch in tenant A for tenant B's type", "nava_app", Denied,
			"INSERT INTO numbering_epochs (tenant_id, entity_definition_id, epoch, compiled_sha256, literal_head) VALUES ($1, $2, 3, sha256('iso'::bytea), 'TCK/')",
			TenantA, EntityDefinitionB1),
		// C8, the FK case: a tenant A version pinned to tenant B's format id
		// fails as one pinned to a made-up id does (23503 on
		// entity_definition_versions_number_format_fkey; the error class is
		// compared in internal/db/dbtest, TestP606LinkToAnotherTenantsFormat).
		sql("entity_definition_versions: pin tenant B's format from tenant A", "nava_app", Denied,
			linkVersion, TenantA, numberFormatB1, NewID, definitionA1, principalA1),
		sql("entity_definition_versions: pin a made-up format from tenant A", "nava_app", Denied,
			linkVersion, TenantA, NewID, NewID, definitionA1, principalA1),
		// No UPDATE or DELETE grant on versions or epochs (C9), aimed at
		// tenant A's own rows, so only the missing grant can stop them.
		sql("number_format_versions: rewrite tenant A's own version", "nava_app", Denied,
			"UPDATE number_format_versions SET compiled = '{}' WHERE tenant_id = $1", TenantA),
		sql("number_format_versions: delete tenant A's own versions", "nava_app", Denied,
			"DELETE FROM number_format_versions WHERE tenant_id = $1", TenantA),
		sql("numbering_epochs: rewrite tenant A's own epoch", "nava_app", Denied,
			"UPDATE numbering_epochs SET literal_head = 'TCK/' WHERE tenant_id = $1", TenantA),
		sql("numbering_epochs: delete tenant A's own epochs", "nava_app", Denied,
			"DELETE FROM numbering_epochs WHERE tenant_id = $1", TenantA),
		sql("number_formats: move tenant B's current versions", "nava_app", Denied,
			"UPDATE number_formats SET current_version = 1, updated_at = now() WHERE tenant_id = $1", TenantB),
		sql("number_formats: rename tenant A's own format", "nava_app", Denied,
			"UPDATE number_formats SET key = 'iso_renamed' WHERE tenant_id = $1", TenantA),
		sql("number_formats: delete tenant A's own formats", "nava_app", Denied,
			"DELETE FROM number_formats WHERE tenant_id = $1", TenantA),
	)
}
