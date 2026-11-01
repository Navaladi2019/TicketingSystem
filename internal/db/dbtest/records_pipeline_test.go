//go:build integration

package dbtest

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
)

// P3-09 (TICK-353): records extended additively for the write pipeline,
// append-only audit_log and record_counters, all under FORCE RLS with
// composite tenant keys, grants from db/grants.yaml, and nava_app's
// plan_cache_mode from db-roles.

const recordsPipelineMigration = "20261101190000_records_pipeline.sql"

const (
	// insertRecord: tenant $1, id $2, type $3, number $4, requester $5,
	// assignee $6, group $7, created_by $8. schema_version 1, state 'new'.
	insertRecord = `INSERT INTO records (tenant_id, id, entity_definition_id, schema_version, number, state,
		requester_id, assignee_id, assignment_group_id, created_by)
		VALUES ($1, $2, $3, 1, $4, 'new', $5, $6, $7, $8)`
	// insertAudit: tenant $1, record $2, idempotency key $3.
	insertAudit = `INSERT INTO audit_log (tenant_id, id, record_id, action, actor_id, source, after, idempotency_key)
		VALUES ($1, gen_random_uuid(), $2, 'create', NULL, 'workflow', '{}', $3)`
	// nextNumber is the D-3-9 upsert: tenant $1, type $2.
	nextNumber = `INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2)
		ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1
		RETURNING next - 1`
)

// seedPipeline writes, as the superuser, for each tenant: the identity rows
// and entity type of seedEntityTypes (all with id = the tenant id), one
// pipeline record (pipelineRecord: the same id in every tenant, unlike
// seedTenants' bare record, whose id is the tenant id), one audit row keyed
// seed-key and one counter.
func seedPipeline(t *testing.T, super *pgx.Conn, tenants ...string) {
	t.Helper()
	seedEntityTypes(t, super, tenants...)
	for _, id := range tenants {
		rec := pipelineRecord(id)
		exec(t, super, insertRecord, id, rec, id, "TCK0000001", id, id, id, id)
		exec(t, super, insertAudit, id, rec, "seed-key")
		exec(t, super, "INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $1, 2)", id)
	}
}

func pipelineRecord(tenant string) string { return tenant[:len(tenant)-3] + "999" }

// Acceptance: the migration adds columns, constraints and indexes to a
// records table that holds PLAN-001 bare (tenant_id, id) rows without
// rewriting it, and the bare rows stay valid with the new defaults.
func TestP309RecordsNotRewritten(t *testing.T) {
	m := freshCluster(t)
	if err := migrateOnlyFS(t, m, migrationsBefore(t, recordsPipelineMigration)); err != nil {
		t.Fatalf("migrate to before %s: %v", recordsPipelineMigration, err)
	}
	super := superNava(t)
	exec(t, super, "INSERT INTO tenants (id, state) VALUES ($1, 'active'), ($2, 'active')", tenantA, tenantB)
	exec(t, super, `INSERT INTO records (tenant_id, id)
		SELECT t, gen_random_uuid() FROM unnest($1::uuid[]) t, generate_series(1, 500)`, []string{tenantA, tenantB})
	before := queryString(t, super, "SELECT relfilenode::text FROM pg_class WHERE oid = 'records'::regclass")

	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if after := queryString(t, super, "SELECT relfilenode::text FROM pg_class WHERE oid = 'records'::regclass"); after != before {
		t.Fatalf("records relfilenode %s -> %s: the migration rewrote the table", before, after)
	}
	if got := queryString(t, super, `SELECT count(*) || ':' || count(*) FILTER (WHERE created_at IS NOT NULL AND updated_at IS NOT NULL
		AND data = '{}' AND entity_definition_id IS NULL AND number IS NULL) FROM records`); got != "1000:1000" {
		t.Fatalf("bare rows after the migration (all:defaulted) = %s, want 1000:1000", got)
	}
}

// Acceptance: the P3-01 index set, every index led by tenant_id, and no GIN
// or expression index on records.
func TestP309RecordsIndexSet(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	got := queryString(t, super, `SELECT string_agg(indexname || ' ' || regexp_replace(indexdef, '^.* USING ', ''), E'\n' ORDER BY indexname)
		FROM pg_indexes WHERE schemaname = 'public' AND tablename = 'records'`)
	want := strings.Join([]string{
		"records_number_key btree (tenant_id, entity_definition_id, number)",
		"records_pkey btree (tenant_id, id)",
		"records_tenant_group_open btree (tenant_id, assignment_group_id, updated_at DESC, id DESC) WHERE (state <> 'closed'::text)",
		"records_tenant_requester btree (tenant_id, requester_id, updated_at DESC, id DESC)",
		"records_tenant_updated btree (tenant_id, updated_at DESC, id DESC)",
	}, "\n")
	if got != want {
		t.Fatalf("records indexes:\n%s\nwant:\n%s", got, want)
	}
	// No index on any P3-09 table leads with anything but tenant_id.
	if bad := queryString(t, super, `SELECT coalesce(string_agg(i.indexrelid::regclass::text, ','), '')
		FROM pg_index i JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = i.indkey[0]
		WHERE i.indrelid IN ('records'::regclass, 'audit_log'::regclass, 'record_counters'::regclass)
		  AND a.attname <> 'tenant_id'`); bad != "" {
		t.Fatalf("indexes not led by tenant_id: %s", bad)
	}
}

// Acceptance: audit_log is append-only for every runtime role, and the
// columns a record sets once (entity_definition_id, number, created_by,
// created_at) cannot be updated; the other pipeline columns can (control).
func TestP309AuditAppendOnlyAndRecordNumberFixed(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedPipeline(t, super, tenantA)
	rec := pipelineRecord(tenantA)

	for _, role := range []string{"nava_app", "nava_worker", "nava_onboard", "nava_groupsync", "nava_relay", "nava_purge"} {
		conn := asRole(t, role)
		for _, q := range []string{
			"UPDATE audit_log SET action = 'forged' WHERE tenant_id = $1",
			"UPDATE audit_log SET idempotency_key = NULL WHERE tenant_id = $1",
			"DELETE FROM audit_log WHERE tenant_id = $1",
			"DELETE FROM records WHERE tenant_id = $1",
			"DELETE FROM record_counters WHERE tenant_id = $1",
		} {
			_, err := execIn(conn, tenantA, q, tenantA)
			wantSQLState(t, role+": "+q, err, "42501")
		}
	}
	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		for _, q := range []string{
			"UPDATE records SET number = 'TCK0000002' WHERE tenant_id = $1 AND id = $2",
			"UPDATE records SET entity_definition_id = $1 WHERE tenant_id = $1 AND id = $2",
			"UPDATE records SET created_by = $1 WHERE tenant_id = $1 AND id = $2",
			"UPDATE records SET created_at = now() WHERE tenant_id = $1 AND id = $2",
			"UPDATE records SET id = gen_random_uuid() WHERE tenant_id = $1 AND id = $2",
		} {
			_, err := execIn(conn, tenantA, q, tenantA, rec)
			wantSQLState(t, role+": "+q, err, "42501")
		}
		n, err := execIn(conn, tenantA, `UPDATE records SET state = 'open', data = '{"title": "x"}', updated_at = now(),
			version = version + 1, schema_version = 1, requester_id = $1, assignee_id = NULL, assignment_group_id = $1,
			causation_origin_id = $1, causation_depth = 1 WHERE tenant_id = $1 AND id = $2`, tenantA, rec)
		if err != nil || n != 1 {
			t.Errorf("control: %s updates the mutable columns: %d rows, %v; want 1, nil", role, n, err)
		}
	}
	if got := queryString(t, super, "SELECT number || ':' || count(*) OVER () FROM audit_log a JOIN records r USING (tenant_id) WHERE r.id = $1", rec); got != "TCK0000001:1" {
		t.Errorf("tenant A's record number and audit rows = %s, want TCK0000001:1", got)
	}
}

// Acceptance: tenant B cannot see tenant A's audit rows or counters; with no
// tenant, or an empty one, a role sees none. Writes for another tenant are
// refused by the policies.
func TestP309TenantRowsInvisible(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB)

	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		for _, table := range []string{"audit_log", "record_counters", "records"} {
			for _, mode := range []string{"B", "empty", "unset"} {
				var own, other int
				err := inTenantTx(conn, "", func(tx pgx.Tx) error {
					if mode != "unset" {
						v := map[string]string{"B": tenantB, "empty": ""}[mode]
						if _, err := tx.Exec(context.Background(), "SELECT set_config('app.tenant_id', $1, true)", v); err != nil {
							return err
						}
					}
					return tx.QueryRow(context.Background(),
						"SELECT count(*) FILTER (WHERE tenant_id = $1), count(*) FILTER (WHERE tenant_id <> $1) FROM "+table,
						tenantB).Scan(&own, &other)
				})
				if err != nil {
					t.Fatalf("%s reads %s (tenant %s): %v", role, table, mode, err)
				}
				wantOwn := 0
				if mode == "B" {
					wantOwn = 1
					if table == "records" {
						wantOwn = 2 // seedTenants' bare row and the pipeline row
					}
				}
				if own != wantOwn || other != 0 {
					t.Errorf("%s reads %s with tenant %s: own %d, other tenants %d; want %d, 0", role, table, mode, own, other, wantOwn)
				}
			}
		}

		_, err := execIn(conn, tenantB, insertAudit, tenantA, pipelineRecord(tenantA), "iso-new")
		wantSQLState(t, role+" in tenant B writes an audit row for A", err, "42501")
		_, err = execIn(conn, tenantB, nextNumber, tenantA, tenantA)
		wantSQLState(t, role+" in tenant B takes a number from A's counter", err, "42501")
		if n, err := execIn(conn, tenantB, "UPDATE record_counters SET next = next + 1 WHERE tenant_id = $1", tenantA); err != nil || n != 0 {
			t.Errorf("%s in tenant B moves A's counter: %d rows, %v; want 0, nil", role, n, err)
		}
		_, err = execIn(conn, "", insertAudit, tenantA, pipelineRecord(tenantA), "iso-unset")
		wantSQLState(t, role+" with no tenant writes an audit row", err, "42501")
	}
	if got := queryString(t, super, "SELECT string_agg(next::text, ',' ORDER BY tenant_id) FROM record_counters"); got != "2,2" {
		t.Errorf("counters after the cross-tenant attempts = %s, want 2,2", got)
	}
}

// Acceptance (TICK-360): a record in tenant A naming tenant B's principal,
// group or type, an audit row naming B's record, and a counter for B's type
// are refused by the composite foreign keys, though the policies pass them.
func TestP309ForeignKeysRejectCrossTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB)
	app := asRole(t, "nava_app")
	a, b := tenantA, tenantB // seeded ids equal the tenant id

	for _, tc := range []struct {
		what                                string
		typ, requester, assignee, group, by any
	}{
		{"B's type", b, a, a, a, a},
		{"B's principal as requester", a, b, a, a, a},
		{"B's principal as assignee", a, a, b, a, a},
		{"B's group", a, a, a, b, a},
		{"B's principal as created_by", a, a, a, a, b},
	} {
		_, err := execIn(app, a, insertRecord, a, tenantX, tc.typ, "TCK0000002", tc.requester, tc.assignee, tc.group, tc.by)
		wantSQLState(t, "A's record naming "+tc.what, err, "23503")
	}
	_, err := execIn(app, a, "INSERT INTO records (tenant_id, id, assignment_group_id) VALUES ($1, $2, $3)", a, tenantX, b)
	wantSQLState(t, "A's bare record naming B's group", err, "23503")
	_, err = execIn(app, a, "UPDATE records SET assignment_group_id = $2 WHERE tenant_id = $1", a, b)
	wantSQLState(t, "A moves its records to B's group", err, "23503")
	_, err = execIn(app, a, "UPDATE records SET requester_id = $2 WHERE tenant_id = $1", a, b)
	wantSQLState(t, "A names B's principal as requester", err, "23503")
	_, err = execIn(app, a, insertAudit, a, b, nil) // B's bare record (seedTenants)
	wantSQLState(t, "A's audit row naming B's record", err, "23503")
	_, err = execIn(app, a, nextNumber, a, b)
	wantSQLState(t, "A's counter for B's type", err, "23503")

	// Controls: the same rows with A's own ids are accepted.
	if _, err := execIn(app, a, insertRecord, a, tenantX, a, "TCK0000002", a, a, a, a); err != nil {
		t.Fatalf("control: A's record naming A's own rows: %v", err)
	}
	if _, err := execIn(app, a, insertAudit, a, tenantX, nil); err != nil {
		t.Fatalf("control: A's audit row for A's record: %v", err)
	}
	if _, err := execIn(app, a, insertAudit, a, nil, nil); err != nil {
		t.Fatalf("control: an audit row with no record (cascade_rejected on a create): %v", err)
	}
}

// Acceptance (TICK-363): a pipeline column set without the others is refused
// by the all-or-none CHECK; a bare row and a full row are accepted.
func TestP309AllOrNoneCheck(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedEntityTypes(t, super, tenantA)
	app := asRole(t, "nava_app")

	for _, tc := range []struct{ what, cols, vals string }{
		{"type but no number", "entity_definition_id, schema_version, state", "$1, 1, 'new'"},
		{"type but no schema_version", "entity_definition_id, number, state", "$1, 'TCK0000009', 'new'"},
		{"type but no state", "entity_definition_id, schema_version, number", "$1, 1, 'TCK0000009'"},
		{"number but no type", "number", "'TCK0000009'"},
		{"state but no type", "state", "'new'"},
		{"schema_version but no type", "schema_version", "1"},
	} {
		_, err := execIn(app, tenantA, "INSERT INTO records (tenant_id, id, "+tc.cols+") VALUES ($1, gen_random_uuid(), "+tc.vals+")", tenantA)
		wantSQLState(t, "a record with "+tc.what, err, "23514")
	}
	_, err := execIn(app, tenantA, "UPDATE records SET state = 'new' WHERE tenant_id = $1 AND id = $1", tenantA)
	wantSQLState(t, "a bare record given a state alone", err, "23514")
	_, err = execIn(app, tenantA, insertRecord, tenantA, tenantX, tenantA, "TCK0000001", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("control: a full pipeline record: %v", err)
	}
	if _, err := execIn(app, tenantA, "INSERT INTO records (tenant_id, id) VALUES ($1, gen_random_uuid())", tenantA); err != nil {
		t.Fatalf("control: a bare PLAN-001 record: %v", err)
	}
	_, err = execIn(app, tenantA, "UPDATE records SET schema_version = 0 WHERE tenant_id = $1 AND id = $2", tenantA, tenantX)
	wantSQLState(t, "schema_version 0", err, "23514")
}

// Acceptance (D-3-13): a second audit row with the same idempotency_key in
// one tenant is refused, the same key in another tenant is allowed, and one
// write's rows with only the primary row keyed are allowed.
func TestP309IdempotencyKeyPerTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB) // each tenant holds "seed-key"
	worker := asRole(t, "nava_worker")
	recA := pipelineRecord(tenantA)

	_, err := execIn(worker, tenantA, insertAudit, tenantA, recA, "seed-key")
	wantSQLState(t, "a second audit row keyed seed-key in tenant A", err, "23505")

	key := "wf:0190a000-0000-7000-8000-0000000000aa:step-1"
	if _, err := execIn(worker, tenantB, insertAudit, tenantB, pipelineRecord(tenantB), key); err != nil {
		t.Fatalf("tenant B's key: %v", err)
	}
	// One write, two rows: the primary row keyed, the cascade_rejected row not.
	err = inTenantTx(worker, tenantA, func(tx pgx.Tx) error {
		ctx := context.Background()
		if _, err := tx.Exec(ctx, insertAudit, tenantA, recA, key); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO audit_log (tenant_id, id, record_id, action, source, idempotency_key)
			VALUES ($1, gen_random_uuid(), $2, 'cascade_rejected', 'workflow', NULL)`, tenantA, recA)
		return err
	})
	if err != nil {
		t.Fatalf("the same key in tenant A, primary row only: %v", err)
	}

	// The replay lookup (D-3-13) through the partial unique index's predicate.
	var n int
	err = inTenantTx(worker, tenantA, func(tx pgx.Tx) error {
		tag, err := tx.Exec(context.Background(), `INSERT INTO audit_log (tenant_id, id, record_id, action, source, idempotency_key)
			VALUES ($1, gen_random_uuid(), $2, 'update', 'workflow', $3)
			ON CONFLICT (tenant_id, idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING`, tenantA, recA, key)
		n = int(tag.RowsAffected())
		return err
	})
	if err != nil || n != 0 {
		t.Fatalf("replay with ON CONFLICT DO NOTHING: %d rows, %v; want 0, nil", n, err)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM audit_log WHERE idempotency_key = $1", key); got != "2" {
		t.Errorf("rows keyed %s = %s, want 2 (one per tenant)", key, got)
	}
	_, err = execIn(worker, tenantA, "INSERT INTO audit_log (tenant_id, id, action, source) VALUES ($1, gen_random_uuid(), 'x', 'operator')", tenantA)
	wantSQLState(t, "an audit row with an unknown source", err, "23514")
}

// The D-3-9 upsert works under the grants and policies, numbers are unique
// per type, not per tenant, and the same number in another type is fine.
func TestP309NumberingUpsertAndUniqueness(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedEntityTypes(t, super, tenantA)
	exec(t, super, "INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version) VALUES ($1, $2, 'incident', 'Incident', 'INC', 1)", tenantA, tenantY)
	app := asRole(t, "nava_app")

	for want := 1; want <= 3; want++ {
		var got int
		err := inTenantTx(app, tenantA, func(tx pgx.Tx) error {
			return tx.QueryRow(context.Background(), nextNumber, tenantA, tenantA).Scan(&got)
		})
		if err != nil || got != want {
			t.Fatalf("number %d from the upsert: got %d, %v", want, got, err)
		}
	}
	if _, err := execIn(app, tenantA, insertRecord, tenantA, tenantX, tenantA, "TCK0000001", nil, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	_, err := execIn(app, tenantA, insertRecord, tenantA, tenantY, tenantA, "TCK0000001", nil, nil, nil, nil)
	wantSQLState(t, "the same number twice in one type", err, "23505")
	if _, err := execIn(app, tenantA, insertRecord, tenantA, tenantY, tenantY, "TCK0000001", nil, nil, nil, nil); err != nil {
		t.Errorf("the same number in another type: %v; want accepted", err)
	}
}

// Group sync deletes a group Keycloak no longer has. A record naming it
// must not block that delete (it would stall the tenant's sync,
// deprovisioning included): the record's group is cleared, its tenant_id
// kept.
func TestP309GroupDeleteClearsRecordGroup(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedPipeline(t, super, tenantA)
	sync := asRole(t, "nava_groupsync")

	n, err := execIn(sync, tenantA, "DELETE FROM groups WHERE tenant_id = $1 AND id = ANY($2::uuid[])", tenantA, []string{tenantA})
	if err != nil || n != 1 {
		t.Fatalf("group sync deletes a group a record names: %d rows, %v; want 1, nil", n, err)
	}
	if got := queryString(t, super, "SELECT tenant_id || ':' || coalesce(assignment_group_id::text, 'null') FROM records WHERE id = $1",
		pipelineRecord(tenantA)); got != tenantA+":null" {
		t.Errorf("record after its group was deleted = %s, want %s:null", got, tenantA)
	}
}

// Acceptance (P3-01 verdict condition): on a fresh cluster nava_app plans
// with custom plans, from db-roles alone, and no other role's plan mode
// changes. Drift, role-wide or in one database, is converged and logged,
// and a converged cluster changes nothing.
func TestP309PlanCacheModeFromDBRoles(t *testing.T) {
	m := freshCluster(t)
	show := func(role string) string {
		return queryString(t, connect(t, roleConfig(t, role, "nava")), "SHOW plan_cache_mode")
	}
	migrateAll(t, m)
	if got := show("nava_app"); got != "force_custom_plan" {
		t.Fatalf("nava_app plan_cache_mode on a fresh cluster = %s, want force_custom_plan", got)
	}
	if got := show("nava_worker"); got != "auto" {
		t.Errorf("nava_worker plan_cache_mode = %s, want the default auto", got)
	}

	conn := connect(t, superConfig(t))
	exec(t, conn, "ALTER ROLE nava_app SET plan_cache_mode = auto")
	exec(t, conn, "ALTER ROLE nava_app IN DATABASE nava SET plan_cache_mode = force_generic_plan")
	exec(t, conn, "ALTER ROLE nava_worker SET work_mem = '64MB'")
	log, buf := newLog()
	res, err := roles.Run(context.Background(), superConfig(t), m, passwords, log)
	if err != nil {
		t.Fatal(err)
	}
	if res.Changed != 3 {
		t.Fatalf("changed = %d, want 3 (nava_app role-wide and in nava, nava_worker)\n%s", res.Changed, buf)
	}
	if !strings.Contains(buf.String(), `"msg":"reset role settings in database"`) {
		t.Errorf("per-database reset not logged:\n%s", buf)
	}
	if got := show("nava_app"); got != "force_custom_plan" {
		t.Errorf("nava_app plan_cache_mode after converging = %s, want force_custom_plan", got)
	}
	if got := queryString(t, conn, `SELECT coalesce(string_agg(r.rolname || coalesce('@' || d.datname, '') || '=' || array_to_string(s.setconfig, ','), ' ' ORDER BY 1), '')
		FROM pg_db_role_setting s JOIN pg_roles r ON r.oid = s.setrole LEFT JOIN pg_database d ON d.oid = s.setdatabase
		WHERE r.rolname = ANY ($1)`, m.RoleNames()); got != "nava_app=plan_cache_mode=force_custom_plan" {
		t.Errorf("manifest role settings = %q, want only nava_app=plan_cache_mode=force_custom_plan", got)
	}
	if res, err := roles.Run(context.Background(), superConfig(t), m, passwords, log); err != nil || res.Total() != 0 {
		t.Fatalf("run after converging: %+v, %v; want no changes", res, err)
	}
}
