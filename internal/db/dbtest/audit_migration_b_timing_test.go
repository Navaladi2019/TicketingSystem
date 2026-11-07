//go:build integration

package dbtest

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// P5-32 (TICK-442) evidence, opt-in: how long migration B holds its six
// ACCESS EXCLUSIVE locks (R27, R-A6) at realistic S8 volumes. Off unless
// NAVA_BENCH_MIGRATION_B lists tenant counts, e.g. "1000,10000";
// scratch/audit-migration-b/run.sh runs it and keeps the output. Each size
// loads, before B: the tenants, one principal each, 10 types per tenant with
// versions 1 to 3 (half the tenants' v1 with no publisher, as onboarding
// seeded it), migration A's backfill, and then a version 4 for 1% of the
// types with no entry (published between A and the deploy). It then times
// B alone, and reads the time of its two functions from
// pg_stat_user_functions.
func TestP532MigrationBTiming(t *testing.T) {
	sizes := os.Getenv("NAVA_BENCH_MIGRATION_B")
	if sizes == "" {
		t.Skip("NAVA_BENCH_MIGRATION_B not set (see scratch/audit-migration-b/run.sh)")
	}
	for _, s := range strings.Split(sizes, ",") {
		tenants, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil || tenants < 1 {
			t.Fatalf("NAVA_BENCH_MIGRATION_B: %q is not a tenant count", s)
		}
		t.Run(strconv.Itoa(tenants), func(t *testing.T) { timeMigrationB(t, tenants) })
	}
}

func timeMigrationB(t *testing.T, tenants int) {
	m := freshCluster(t)
	if err := migrateOnlyFS(t, m, migrationsThrough(t, auditBackfillMigration)); err != nil {
		t.Fatal(err)
	}
	super := superNava(t)
	exec(t, super, "INSERT INTO tenants (id, state) SELECT gen_random_uuid(), 'active' FROM generate_series(1, $1)", tenants)
	exec(t, super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, email, enabled, federated_links, synced_at)
		SELECT id, id, 'kc-' || id, 'u@example.test', true, '[]', now() FROM tenants`)
	exec(t, super, `INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version)
		SELECT t.id, gen_random_uuid(), 'kind_' || g, 'Kind', 'K' || chr(64 + g) || chr(64 + g), 3
		FROM tenants t, generate_series(1, 10) g`)
	exec(t, super, `INSERT INTO entity_definition_versions
		(tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by, published_at)
		SELECT d.tenant_id, gen_random_uuid(), d.id, v, '{"type":"object"}', sha256('{"type":"object"}'), '{}', '{}',
		       CASE WHEN v = 1 THEN 'initial' ELSE 'compatible' END,
		       CASE WHEN v = 1 AND hashtext(d.tenant_id::text) % 2 = 0 THEN NULL ELSE d.tenant_id END,
		       now() - make_interval(days => 100 - v)
		FROM entity_definitions d, generate_series(1, 3) v`)
	if err := migrateOnlyFS(t, m, migrationsBefore(t, migrationB)); err != nil {
		t.Fatal(err)
	}
	// Migration A's backfill already ran (on empty tables) inside the
	// migration; this is the run the history above would have had.
	start := time.Now()
	added := queryString(t, asRole(t, "nava_migrator"), "SELECT audit_backfill_s8()::text")
	t.Logf("tenants=%d migration A backfill: %s entries in %s", tenants, added, time.Since(start).Round(time.Millisecond))
	exec(t, super, `INSERT INTO entity_definition_versions
		(tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by)
		SELECT d.tenant_id, gen_random_uuid(), d.id, 4, '{"type":"object"}', sha256('{"type":"object"}'), '{}', '{}',
		       'compatible', d.tenant_id
		FROM entity_definitions d WHERE hashtext(d.id::text) % 100 = 0`)
	gap := queryString(t, super, "SELECT count(*)::text FROM entity_definition_versions WHERE version = 4")
	versions := queryString(t, super, "SELECT count(*)::text FROM entity_definition_versions")
	exec(t, super, "ANALYZE")
	exec(t, super, "ALTER DATABASE nava SET track_functions = 'pl'")
	exec(t, super, "SELECT pg_stat_reset()")

	start = time.Now()
	if err := migrateOnlyFS(t, m, migrationsThrough(t, migrationB)); err != nil {
		t.Fatal(err)
	}
	took := time.Since(start)

	// The migrator's session has ended; its function stats flush on exit.
	var fns string
	for range 50 {
		fns = queryString(t, super, `SELECT coalesce(string_agg(funcname || ' calls=' || calls || ' total_ms=' || round(total_time::numeric, 1),
			'; ' ORDER BY funcname), '') FROM pg_stat_user_functions
			WHERE funcname IN ('audit_backfill_s8', 'audit_seed_chain_heads', 'audit_seed_uuidv7')`)
		if strings.Count(fns, "calls=") == 3 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Logf("tenants=%d versions=%s (between A and B: %s) migration B: %s", tenants, versions, gap, took.Round(time.Millisecond))
	t.Logf("tenants=%d functions: %s", tenants, fns)
	t.Logf("tenants=%d rows: heads=%s seal_rows=%s backfilled_by_B=%s", tenants,
		queryString(t, super, "SELECT count(*)::text FROM audit_chain_head"),
		queryString(t, super, "SELECT count(*)::text FROM outbox WHERE kind = 'audit.seal'"),
		queryString(t, super, `SELECT count(*)::text FROM audit_log l JOIN entity_definition_versions v
			ON l.tenant_id = v.tenant_id AND l.subject_id = v.definition_id AND l.version_after = v.version WHERE v.version = 4`))
	t.Logf("tenants=%d storage: audit_chain_head=%s outbox=%s (heap and indexes)", tenants,
		queryString(t, super, "SELECT pg_size_pretty(pg_total_relation_size('audit_chain_head'))"),
		queryString(t, super, "SELECT pg_size_pretty(pg_total_relation_size('outbox'))"))
	if bad := queryString(t, super, entriesPerVersion); bad != "" {
		t.Errorf("CI-A9 after B: %s", bad)
	}
	exec(t, super, "ALTER DATABASE nava RESET track_functions")
	// The grants, policies and gate, as a real migrate run ends.
	migrateAll(t, m)
}
