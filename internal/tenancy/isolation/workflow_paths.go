package isolation

// The workflow definition seed (testdata/seed.sql, P3-40): both tenants own
// the key triage (tenant A's is WorkflowDefinitionA, tenant B's is
// SharedDefinition, the definition of tenant B's seeded cascade claim), and
// only tenant B owns escalate. SharedCause is a delivered outbox id in both
// tenants' workflow_cause, and SharedSignal the record_changed signal id both
// tenants' open refs wait on, so a lookup by any of them alone would find
// tenant B's rows.
const (
	WorkflowDefinitionA = "0d000000-0000-4000-8000-0000000009d3"
	WorkflowVersionB2   = "0b000000-0000-4000-8000-0000000012b2"
	SharedCause         = "0d000000-0000-4000-8000-0000000009d5"
	SharedSignal        = "0d000000-0000-4000-8000-0000000009d6"
	// refA1 is tenant A's seeded workflow_ref (outbox row a1).
	refA1 = "0a000000-0000-4000-8000-0000000001a1"
	// entityTypeA1 is tenant A's seeded ticket type.
	entityTypeA1 = "0a000000-0000-4000-8000-0000000006a1"
)

// workflowPaths are P3-40's tables (TICK-709, S11b contract D-3-29): the
// definitions and their versions, workflow_cause, cascade_start's
// definition key and workflow_ref's waiting state, each run as tenant A and
// aimed at tenant B.
func workflowPaths() []Path {
	sql := func(name, role string, expect Expect, query string, args ...any) Path {
		return Path{Name: name, Case: "CI-2", Role: role, SQL: &SQLCall{Query: query, Args: args, Expect: expect}}
	}
	const (
		insertVersion = `INSERT INTO workflow_definition_versions (tenant_id, id, definition_id, version, body, body_sha256,
			step_ids, trigger_on, trigger_entity_definition_id, published_by)
			VALUES ($1, $2, $3, 3, '{}'::json, sha256(convert_to('{}', 'UTF8')), '{}', 'record.created', $4, $5)`
		insertCause = `INSERT INTO workflow_cause (tenant_id, outbox_id, ref_outbox_id, causation_origin_id, causation_depth)
			VALUES ($1, $2, $3, $4, 1)`
	)
	return []Path{
		sql("workflow_definitions: look a shared key up in every tenant", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM workflow_definitions WHERE key = 'triage'"),
		sql("workflow_definitions: read tenant B's only key", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM workflow_definitions WHERE key = 'escalate'"),
		sql("workflow_definitions: read tenant B's definition by its full key", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM workflow_definitions WHERE tenant_id = $1 AND id = $2", TenantB, SharedDefinition),
		// entity_definition_id and created_by are tenant B's, so only the
		// policy can refuse the row.
		sql("workflow_definitions: create a definition for tenant B", "nava_app", Denied,
			`INSERT INTO workflow_definitions (tenant_id, id, key, entity_definition_id, current_version, created_by)
			 VALUES ($1, $2, 'iso_new', $3, 1, $4)`, TenantB, NewID, EntityDefinitionB1, PrincipalB1),
		// The composite foreign key holds the tenant: tenant A cannot hang a
		// definition off tenant B's entity type.
		sql("workflow_definitions: hang tenant A's definition off tenant B's entity type", "nava_app", Denied,
			`INSERT INTO workflow_definitions (tenant_id, id, key, entity_definition_id, current_version, created_by)
			 VALUES ($1, $2, 'iso_new', $3, 1, $4)`, TenantA, NewID, EntityDefinitionB1, principalA1),
		sql("workflow_definitions: move tenant B's current versions", "nava_app", Denied,
			"UPDATE workflow_definitions SET current_version = 1, updated_at = now() WHERE tenant_id = $1", TenantB),

		sql("workflow_definition_versions: read tenant B's versions by definition id", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM workflow_definition_versions WHERE definition_id = $1", SharedDefinition),
		sql("workflow_definition_versions: load tenant B's version by its id", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM workflow_definition_versions WHERE id = $1", WorkflowVersionB2),
		// The sink's trigger lookup, aimed at tenant B's entity type.
		sql("workflow_definition_versions: match tenant B's triggers", "nava_app", OwnRowsOnly,
			`SELECT v.tenant_id FROM workflow_definition_versions v
			 JOIN workflow_definitions d ON d.tenant_id = v.tenant_id AND d.id = v.definition_id AND d.current_version = v.version
			 WHERE v.tenant_id = $1 AND v.trigger_entity_definition_id = $2 AND v.trigger_on = 'record.created'`,
			TenantB, EntityDefinitionB1),
		sql("workflow_definition_versions: publish a version for tenant B", "nava_app", Denied,
			insertVersion, TenantB, NewID, SharedDefinition, EntityDefinitionB1, PrincipalB1),
		sql("workflow_definition_versions: hang tenant A's version off tenant B's definition", "nava_app", Denied,
			insertVersion, TenantA, NewID, SharedDefinition, entityTypeA1, principalA1),
		sql("workflow_definition_versions: trigger tenant A's version on tenant B's entity type", "nava_app", Denied,
			insertVersion, TenantA, NewID, WorkflowDefinitionA, EntityDefinitionB1, principalA1),
		sql("workflow_definition_versions: rewrite tenant B's versions", "nava_worker", Denied,
			"UPDATE workflow_definition_versions SET step_ids = '{}' WHERE tenant_id = $1", TenantB),

		sql("workflow_cause: look a shared delivery up in every tenant", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM workflow_cause WHERE outbox_id = $1", SharedCause),
		sql("workflow_cause: read tenant B's cause by its full key", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM workflow_cause WHERE tenant_id = $1 AND outbox_id = $2", TenantB, SharedCause),
		sql("workflow_cause: record a cause for tenant B", "nava_worker", Denied,
			insertCause, TenantB, NewID, refA1, SharedOrigin),
		sql("workflow_cause: rewrite tenant B's causation", "nava_worker", Denied,
			"UPDATE workflow_cause SET causation_depth = 0 WHERE tenant_id = $1", TenantB),

		sql("workflow_ref: look a waiting signal id up in every tenant", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM workflow_ref WHERE record_changed_signal_id = $1", SharedSignal),
		sql("workflow_ref: clear tenant B's waiting state", "nava_worker", Denied,
			"UPDATE workflow_ref SET record_changed_signal_id = NULL WHERE tenant_id = $1", TenantB),

		// The composite foreign key holds the tenant: tenant A cannot claim a
		// start of tenant B's definition on its own record.
		sql("cascade_start: claim a start of tenant B's definition from tenant A", "nava_worker", Denied,
			`INSERT INTO cascade_start (tenant_id, causation_origin_id, record_id, definition_id, outbox_id)
			 VALUES ($1, $2, $3, $4, $5)`, TenantA, SharedOrigin, RecordA2, SharedDefinition, NewID),
	}
}
