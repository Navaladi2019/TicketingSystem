package isolation

// schemaBackfillPaths are P10-01's tables (TICK-916, ADR-009 D2, D7): the job
// row, its queue and its failures, each run as tenant A and aimed at tenant B
// (testdata/seed.sql). Both tenants queue and fail the shared record id
// RecordA1, so a probe by record alone would meet tenant B's rows. nava_worker
// runs the job's paths; nava_app only reads (D10, D12), so its writes are
// refused by grant before any policy.
func schemaBackfillPaths() []Path {
	sql := func(name, role string, expect Expect, query string, args ...any) Path {
		return Path{Name: name, Case: "CI-3", Role: role, SQL: &SQLCall{Query: query, Args: args, Expect: expect}}
	}
	const (
		insertJob = `INSERT INTO schema_backfills (tenant_id, entity_definition_id, target_version, default_keys)
			VALUES ($1, $2, 9, '{priority}')`
		insertPending = `INSERT INTO schema_backfill_pending (tenant_id, entity_definition_id, target_version, record_id)
			VALUES ($1, $2, 2, $3)`
		insertFailure = `INSERT INTO schema_backfill_failures (tenant_id, entity_definition_id, target_version, record_id, code)
			VALUES ($1, $2, 2, $3, 'validation_failed')`
	)
	return []Path{
		// schema_backfills: D2's check of the def and version under RLS, and
		// D8's look for a newer job, both aimed at tenant B's job.
		sql("schema_backfills: read tenant B's job by its full key", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM schema_backfills WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = 2",
			TenantB, EntityDefinitionB1),
		sql("schema_backfills: list every tenant's running jobs", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM schema_backfills WHERE state = 'running'"),
		sql("schema_backfills: queue a job for tenant B", "nava_worker", Denied,
			insertJob, TenantB, EntityDefinitionB1),
		// The composite foreign key holds the tenant.
		sql("schema_backfills: queue tenant A's job on tenant B's type", "nava_worker", Denied,
			insertJob, TenantA, EntityDefinitionB1),
		sql("schema_backfills: supersede tenant B's job", "nava_worker", Denied,
			"UPDATE schema_backfills SET state = 'superseded', finished_at = now() WHERE entity_definition_id = $1", EntityDefinitionB1),
		sql("schema_backfills: move tenant A's job to tenant B", "nava_worker", Denied,
			"UPDATE schema_backfills SET tenant_id = $2 WHERE tenant_id = $1", TenantA, TenantB),
		sql("schema_backfills: delete tenant B's job", "nava_worker", Denied,
			"DELETE FROM schema_backfills WHERE tenant_id = $1", TenantB),
		sql("schema_backfills: nava_app queues a job", "nava_app", Denied,
			insertJob, TenantA, definitionA1),
		sql("schema_backfills: nava_app finishes tenant B's job", "nava_app", Denied,
			"UPDATE schema_backfills SET state = 'done' WHERE tenant_id = $1", TenantB),

		// schema_backfill_pending: the page read (D7) and D12's join, by record.
		sql("schema_backfill_pending: read a shared record id in every tenant", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM schema_backfill_pending WHERE record_id = $1", RecordA1),
		sql("schema_backfill_pending: page tenant B's queue", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM schema_backfill_pending WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = 2 ORDER BY record_id LIMIT 100",
			TenantB, EntityDefinitionB1),
		sql("schema_backfill_pending: queue a record for tenant B", "nava_worker", Denied,
			insertPending, TenantB, EntityDefinitionB1, RecordB2),
		sql("schema_backfill_pending: queue under tenant B's job from tenant A", "nava_worker", Denied,
			insertPending, TenantA, EntityDefinitionB1, RecordA2),
		sql("schema_backfill_pending: drain tenant B's queue", "nava_worker", Denied,
			"DELETE FROM schema_backfill_pending WHERE record_id = $1 AND tenant_id <> $2", RecordA1, TenantA),
		sql("schema_backfill_pending: nava_app drains the queue", "nava_app", Denied,
			"DELETE FROM schema_backfill_pending WHERE record_id = $1", RecordA1),

		// schema_backfill_failures: D10's failures read and D12's join.
		sql("schema_backfill_failures: read a shared record id in every tenant", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM schema_backfill_failures WHERE record_id = $1", RecordA1),
		sql("schema_backfill_failures: list tenant B's failures", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM schema_backfill_failures WHERE tenant_id = $1", TenantB),
		sql("schema_backfill_failures: record a failure for tenant B", "nava_worker", Denied,
			insertFailure, TenantB, EntityDefinitionB1, RecordB2),
		sql("schema_backfill_failures: clear tenant B's failures", "nava_worker", Denied,
			"DELETE FROM schema_backfill_failures WHERE tenant_id = $1", TenantB),
		sql("schema_backfill_failures: nava_app records a failure", "nava_app", Denied,
			insertFailure, TenantA, definitionA1, RecordA2),
	}
}
