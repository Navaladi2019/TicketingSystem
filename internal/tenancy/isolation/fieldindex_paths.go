package isolation

// The field-index seed (testdata/seed.sql, P7-01, ADR-006): both tenants flag
// priority (text) under the same field id SharedField, so a lookup by the id
// or the key alone would find tenant B's row. Every field of both tenants
// holds a value on record RecordA1, which both tenants hold, so a side-table
// probe by field and record alone would meet tenant B's rows.
// FieldNumB, FieldBoolB and FieldRemovingB are tenant B's only.
const (
	SharedField    = "0d000000-0000-4000-8000-0000000009e1"
	FieldNumB      = "0b000000-0000-4000-8000-0000000014b2"
	FieldBoolB     = "0b000000-0000-4000-8000-0000000014b4"
	FieldRemovingB = "0b000000-0000-4000-8000-0000000014b5"
	// fieldNumA is tenant A's num field.
	fieldNumA = "0a000000-0000-4000-8000-0000000014a2"
)

// fieldIndexPaths are P7-01's tables (TICK-741, ADR-006 D2, D3, D14): the
// registry and the four typed side tables, each run as tenant A and aimed at
// tenant B. nava_app and nava_worker both write records, so both run the
// side-table paths P7-02's trigger will take as the writer.
func fieldIndexPaths() []Path {
	sql := func(name, role string, expect Expect, query string, args ...any) Path {
		return Path{Name: name, Case: "CI-2", Role: role, SQL: &SQLCall{Query: query, Args: args, Expect: expect}}
	}
	const insertField = `INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, requested_version)
		VALUES ($1, $2, $3, 'iso_new', 'text', 'backfilling', 1)`
	paths := []Path{
		sql("record_index_fields: look a shared field id up in every tenant", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM record_index_fields WHERE id = $1", SharedField),
		sql("record_index_fields: look a shared field key up in every tenant", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM record_index_fields WHERE field_key = 'priority'"),
		// The trigger's lookup (P7-02), aimed at tenant B's entity type.
		sql("record_index_fields: list tenant B's flagged fields of its type", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM record_index_fields WHERE tenant_id = $1 AND entity_definition_id = $2 AND state IN ('backfilling', 'ready')",
			TenantB, EntityDefinitionB1),
		// entity_definition_id is tenant B's, so only the policy can refuse it.
		sql("record_index_fields: register a field for tenant B", "nava_app", Denied,
			insertField, TenantB, NewID, EntityDefinitionB1),
		// The composite foreign key holds the tenant.
		sql("record_index_fields: hang tenant A's field off tenant B's entity type", "nava_app", Denied,
			insertField, TenantA, NewID, EntityDefinitionB1),
		sql("record_index_fields: publish flips tenant B's fields", "nava_app", Denied,
			"UPDATE record_index_fields SET state = 'removing', requested_version = 9, updated_at = now() WHERE tenant_id = $1", TenantB),
		sql("record_index_fields: the backfill marks tenant B's field ready", "nava_worker", Denied,
			"UPDATE record_index_fields SET state = 'ready', progress = 100, updated_at = now() WHERE id = $1", FieldBoolB),
		sql("record_index_fields: removal deletes tenant B's removing field", "nava_worker", Denied,
			"DELETE FROM record_index_fields WHERE id = $1", FieldRemovingB),
	}
	for _, s := range []struct{ table, value, field, fieldA string }{
		{"rfv_text", "'high'", SharedField, SharedField},
		{"rfv_num", "1", FieldNumB, fieldNumA},
	} {
		insert := "INSERT INTO " + s.table + " (tenant_id, field_id, record_id, ord, value) VALUES ($1, $2, $3, 1, " + s.value + ")"
		paths = append(paths,
			// The filter's EXISTS probe by field and value, and the trigger's
			// per-record probe, without their tenant predicate.
			sql(s.table+": filter tenant B's field by value", "nava_app", OwnRowsOnly,
				"SELECT tenant_id FROM "+s.table+" WHERE field_id = $1 AND value = "+s.value, s.field),
			sql(s.table+": read tenant B's values of a shared record id", "nava_worker", OwnRowsOnly,
				"SELECT tenant_id FROM "+s.table+" WHERE record_id = $1", RecordA1),
			sql(s.table+": index a value for tenant B", "nava_app", Denied,
				insert, TenantB, s.field, RecordB1),
			sql(s.table+": the backfill writes a value for tenant B", "nava_worker", Denied,
				insert, TenantB, s.field, RecordB1),
			// The composite foreign keys hold the tenant.
			sql(s.table+": index tenant A's field on tenant B's record", "nava_worker", Denied,
				insert, TenantA, s.fieldA, RecordB1),
			sql(s.table+": purge tenant B's values of a shared record id", "nava_app", Denied,
				"DELETE FROM "+s.table+" WHERE record_id = $1 AND tenant_id <> $2", RecordA1, TenantA),
			sql(s.table+": removal deletes tenant B's field values", "nava_worker", Denied,
				"DELETE FROM "+s.table+" WHERE field_id = $1 AND tenant_id = $2", s.field, TenantB),
		)
	}
	// rfv_ts and rfv_bool: the same policies, one read and one write each;
	// the generic checks probe every granted command on all four.
	paths = append(paths,
		sql("rfv_ts: read tenant B's values of a shared record id", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM rfv_ts WHERE record_id = $1", RecordA1),
		sql("rfv_ts: index a value for tenant B", "nava_worker", Denied,
			"INSERT INTO rfv_ts (tenant_id, field_id, record_id, ord, value) VALUES ($1, $2, $3, 1, now())",
			TenantB, "0b000000-0000-4000-8000-0000000014b3", RecordB1),
		sql("rfv_bool: filter tenant B's bool field", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM rfv_bool WHERE field_id = $1 AND value = false", FieldBoolB),
		sql("rfv_bool: index a value for tenant B", "nava_app", Denied,
			"INSERT INTO rfv_bool (tenant_id, field_id, record_id, ord, value) VALUES ($1, $2, $3, 1, true)",
			TenantB, FieldBoolB, RecordB1),
	)
	return append(paths, fieldIndexAttackPaths()...)
}
