//go:build integration

package dbtest

import (
	"io/fs"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// P10-01 (TICK-916, ADR-009 D2, D7, grants): schema_backfills,
// schema_backfill_pending and schema_backfill_failures.

const (
	schemaBackfillMigration = "20261126163300_schema_backfill_tables.sql"

	// insertBackfill: tenant $1, entity type $2, target version $3, state $4.
	insertBackfill = `INSERT INTO schema_backfills (tenant_id, entity_definition_id, target_version, state, default_keys)
		VALUES ($1, $2, $3, $4, '{priority}')`
	// insertBackfillPending: tenant $1, entity type $2, version $3, record $4.
	insertBackfillPending = `INSERT INTO schema_backfill_pending (tenant_id, entity_definition_id, target_version, record_id)
		VALUES ($1, $2, $3, $4)`
	// insertBackfillFailure: tenant $1, entity type $2, version $3, record $4, code $5.
	insertBackfillFailure = `INSERT INTO schema_backfill_failures (tenant_id, entity_definition_id, target_version, record_id, code)
		VALUES ($1, $2, $3, $4, $5)`
)

// schemaBackfillTables are the tables schemaBackfillMigration creates, job first.
var schemaBackfillTables = []string{"schema_backfills", "schema_backfill_pending", "schema_backfill_failures"} //nolint:gochecknoglobals // read-only test fixture, set once

// beforeSchemaBackfillTables is m and a as a run of fsys sees them when fsys
// stops before schemaBackfillMigration: without the three tables in either
// manifest. beforeNumberingMode calls it, so every older-schema run gets it.
func beforeSchemaBackfillTables(fsys fs.FS, m *grants.Manifest, a *audited.Manifest) (*grants.Manifest, *audited.Manifest) {
	if _, err := fs.Stat(fsys, schemaBackfillMigration); err == nil {
		return m, a
	}
	gm := *m
	gm.Roles = make(map[string]grants.Role, len(m.Roles))
	for name, r := range m.Roles {
		tables := make(map[string]map[string][]string, len(r.Tables))
		for tbl, privs := range r.Tables {
			if !slices.Contains(schemaBackfillTables, tbl) {
				tables[tbl] = privs
			}
		}
		r.Tables = tables
		gm.Roles[name] = r
	}
	am := *a
	am.Audited = maps.Clone(a.Audited)
	am.Exempt = maps.Clone(a.Exempt)
	for _, tbl := range schemaBackfillTables {
		delete(am.Audited, tbl)
		delete(am.Exempt, tbl)
	}
	return &gm, &am
}

// seedSchemaBackfill writes, as the superuser, for each tenant one pending job
// on seedEntityTypes' type (version 2), one queued and one failed id: the
// tenant's seeded record (id = the tenant id).
func seedSchemaBackfill(t *testing.T, super *pgx.Conn, tenants ...string) {
	t.Helper()
	for _, tenant := range tenants {
		exec(t, super, insertBackfill, tenant, tenant, 2, "pending")
		exec(t, super, insertBackfillPending, tenant, tenant, 2, tenant)
		exec(t, super, insertBackfillFailure, tenant, tenant, 2, tenant, "validation_failed")
	}
}

// Acceptance: keys, foreign keys, the indexes and the active-job index are
// the contract's (C1), with RLS enabled and forced and one policy per command
// and role.
func TestP1001Shape(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)

	for name, want := range map[string]string{
		"schema_backfills_pkey":                          "PRIMARY KEY (tenant_id, entity_definition_id, target_version)",
		"schema_backfills_entity_definition_fkey":        "FOREIGN KEY (tenant_id, entity_definition_id) REFERENCES entity_definitions(tenant_id, id)",
		"schema_backfill_pending_pkey":                   "PRIMARY KEY (tenant_id, entity_definition_id, target_version, record_id)",
		"schema_backfill_pending_entity_definition_fkey": "FOREIGN KEY (tenant_id, entity_definition_id) REFERENCES entity_definitions(tenant_id, id)",
		"schema_backfill_pending_job_fkey": "FOREIGN KEY (tenant_id, entity_definition_id, target_version) " +
			"REFERENCES schema_backfills(tenant_id, entity_definition_id, target_version) ON DELETE CASCADE",
		"schema_backfill_failures_pkey":                   "PRIMARY KEY (tenant_id, entity_definition_id, target_version, record_id)",
		"schema_backfill_failures_entity_definition_fkey": "FOREIGN KEY (tenant_id, entity_definition_id) REFERENCES entity_definitions(tenant_id, id)",
		"schema_backfill_failures_job_fkey": "FOREIGN KEY (tenant_id, entity_definition_id, target_version) " +
			"REFERENCES schema_backfills(tenant_id, entity_definition_id, target_version) ON DELETE CASCADE",
		"schema_backfill_failures_code_check": "CHECK ((code ~ '^[a-z_]{1,64}$'::text))",
	} {
		if got := queryString(t, super, `SELECT coalesce((SELECT pg_get_constraintdef(oid) FROM pg_constraint
			WHERE conname = $1), '')`, name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	for idx, want := range map[string]string{
		"schema_backfills_one_active_per_tenant": "CREATE UNIQUE INDEX schema_backfills_one_active_per_tenant ON public.schema_backfills USING btree (tenant_id) " +
			"WHERE (state = ANY (ARRAY['filling'::text, 'running'::text]))",
		"schema_backfill_pending_record":  "CREATE INDEX schema_backfill_pending_record ON public.schema_backfill_pending USING btree (tenant_id, record_id)",
		"schema_backfill_failures_record": "CREATE INDEX schema_backfill_failures_record ON public.schema_backfill_failures USING btree (tenant_id, record_id)",
	} {
		if got := queryString(t, super, `SELECT pg_get_indexdef($1::regclass)`, idx); got != want {
			t.Errorf("%s = %s, want %s", idx, got, want)
		}
	}
	// No value column, ever (D7): failures hold ids, a code and a time.
	if got := queryString(t, super, `SELECT string_agg(attname, ',' ORDER BY attnum) FROM pg_attribute
		WHERE attrelid = 'schema_backfill_failures'::regclass AND attnum > 0 AND NOT attisdropped`); got !=
		"tenant_id,entity_definition_id,target_version,record_id,code,failed_at" {
		t.Errorf("schema_backfill_failures columns = %s", got)
	}
	// record_id has no foreign key: a record delete is never blocked by a
	// queued or failed id.
	if got := queryString(t, super, `SELECT count(*)::text FROM pg_constraint
		WHERE confrelid = 'records'::regclass AND conrelid = ANY ('{schema_backfill_pending,schema_backfill_failures}'::regclass[])`); got != "0" {
		t.Errorf("foreign keys from the queue or failures to records = %s, want 0", got)
	}
	for _, tbl := range schemaBackfillTables {
		if got := queryString(t, super, `SELECT relrowsecurity::text || relforcerowsecurity::text FROM pg_class
			WHERE oid = $1::regclass`, tbl); got != "truetrue" {
			t.Errorf("%s RLS enabled and forced = %s, want truetrue", tbl, got)
		}
		if got := queryString(t, super, `SELECT string_agg(cmd || ':' || array_to_string(roles, ','), ' ' ORDER BY cmd, roles::text)
			FROM pg_policies WHERE tablename = $1`, tbl); got !=
			map[bool]string{
				true:  "DELETE:nava_worker INSERT:nava_app INSERT:nava_worker SELECT:nava_app SELECT:nava_worker UPDATE:nava_worker",
				false: "DELETE:nava_worker INSERT:nava_worker SELECT:nava_app SELECT:nava_worker UPDATE:nava_worker",
			}[tbl == "schema_backfills"] {
			t.Errorf("%s policies = %s", tbl, got)
		}
	}
}

// Acceptance (CI-3 (5) and (6)): as nava_app and as nava_worker in tenant A,
// each table shows no tenant B row and an INSERT with tenant B's tenant_id is
// refused; with no tenant each table reads zero rows; nava_app's UPDATE and
// DELETE on all three and INSERT on the queue and failures are refused with
// 42501. nava_app inserts only a fresh pending job of its own tenant
// (PLAN-010 D-10-11): a running row, a count, a start or finish time, or
// tenant B's tenant_id is refused.
func TestP1001TenantIsolation(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedEntityTypes(t, super, tenantA, tenantB)
	seedSchemaBackfill(t, super, tenantA, tenantB)

	inserts := map[string]func(tenant string) (string, []any){
		"schema_backfills": func(tn string) (string, []any) { return insertBackfill, []any{tn, tn, 3, "pending"} },
		"schema_backfill_pending": func(tn string) (string, []any) {
			return insertBackfillPending, []any{tn, tn, 2, newUUID(t, super)}
		},
		"schema_backfill_failures": func(tn string) (string, []any) {
			return insertBackfillFailure, []any{tn, tn, 2, newUUID(t, super), "conflict"}
		},
	}
	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		for _, tbl := range schemaBackfillTables {
			for _, c := range []struct {
				tenant, q string
				want      int
			}{
				{tenantA, "SELECT count(*) FROM " + tbl + " WHERE tenant_id <> $1", 0},
				{tenantA, "SELECT count(*) FROM " + tbl + " WHERE tenant_id = $1", 1},
				{"", "SELECT count(*) FROM " + tbl + " WHERE tenant_id = $1 OR true", 0},
			} {
				var n int
				if err := inTenantTx(conn, c.tenant, func(tx pgx.Tx) error {
					return tx.QueryRow(t.Context(), c.q, tenantA).Scan(&n)
				}); err != nil || n != c.want {
					t.Errorf("%s in tenant %q: %s = %d, %v; want %d", role, c.tenant, c.q, n, err, c.want)
				}
			}
			q, args := inserts[tbl](tenantB)
			_, err := execIn(conn, tenantA, q, args...)
			wantSQLState(t, role+" in tenant A inserts a "+tbl+" row for tenant B", err, "42501")
		}
	}

	app, worker := asRole(t, "nava_app"), asRole(t, "nava_worker")
	for _, tbl := range schemaBackfillTables {
		q, args := inserts[tbl](tenantA)
		var err error
		if tbl == "schema_backfills" {
			// D-10-11: the publish transaction's pending job (another version
			// than the worker's control row below).
			if n, err := execIn(app, tenantA, insertBackfill, tenantA, tenantA, 30, "pending"); err != nil || n != 1 {
				t.Errorf("nava_app inserts tenant A's pending job: %d, %v; want 1", n, err)
			}
		} else {
			_, err = execIn(app, tenantA, q, args...)
			wantSQLState(t, "nava_app inserts tenant A's "+tbl+" row", err, "42501")
		}
		_, err = execIn(app, tenantA, "UPDATE "+tbl+" SET target_version = target_version WHERE tenant_id = $1", tenantA)
		wantSQLState(t, "nava_app updates "+tbl, err, "42501")
		_, err = execIn(app, tenantA, "DELETE FROM "+tbl+" WHERE tenant_id = $1", tenantA)
		wantSQLState(t, "nava_app deletes from "+tbl, err, "42501")

		// The worker: its own tenant's writes succeed, tenant B's touch nothing.
		if n, err := execIn(worker, tenantA, q, args...); err != nil || n != 1 {
			t.Errorf("control: nava_worker inserts tenant A's %s row: %d, %v", tbl, n, err)
		}
		if n, err := execIn(worker, tenantA, "UPDATE "+tbl+" SET target_version = target_version WHERE tenant_id = $1", tenantB); err != nil || n != 0 {
			t.Errorf("nava_worker in tenant A updates tenant B's %s rows: %d, %v; want 0", tbl, n, err)
		}
		if n, err := execIn(worker, tenantA, "DELETE FROM "+tbl+" WHERE tenant_id = $1", tenantB); err != nil || n != 0 {
			t.Errorf("nava_worker in tenant A deletes tenant B's %s rows: %d, %v; want 0", tbl, n, err)
		}
		_, err = execIn(worker, tenantA, "UPDATE "+tbl+" SET tenant_id = $2 WHERE tenant_id = $1", tenantA, tenantB)
		wantSQLState(t, "nava_worker moves tenant A's "+tbl+" rows to tenant B", err, "42501")
	}

	// D-10-11: nava_app's job insert is a fresh pending row or nothing.
	for name, q := range map[string]string{
		"a running row": `INSERT INTO schema_backfills (tenant_id, entity_definition_id, target_version, state, default_keys)
			VALUES ($1, $1, 31, 'running', '{priority}')`,
		"a filling row": `INSERT INTO schema_backfills (tenant_id, entity_definition_id, target_version, state, default_keys)
			VALUES ($1, $1, 32, 'filling', '{priority}')`,
		"a done row": `INSERT INTO schema_backfills (tenant_id, entity_definition_id, target_version, state, default_keys)
			VALUES ($1, $1, 33, 'done', '{priority}')`,
		"rewritten 1": `INSERT INTO schema_backfills (tenant_id, entity_definition_id, target_version, default_keys, rewritten)
			VALUES ($1, $1, 34, '{priority}', 1)`,
		"skipped 1": `INSERT INTO schema_backfills (tenant_id, entity_definition_id, target_version, default_keys, skipped)
			VALUES ($1, $1, 35, '{priority}', 1)`,
		"failed 1": `INSERT INTO schema_backfills (tenant_id, entity_definition_id, target_version, default_keys, failed)
			VALUES ($1, $1, 36, '{priority}', 1)`,
		"a start time": `INSERT INTO schema_backfills (tenant_id, entity_definition_id, target_version, default_keys, started_at)
			VALUES ($1, $1, 37, '{priority}', now())`,
		"a finish time": `INSERT INTO schema_backfills (tenant_id, entity_definition_id, target_version, default_keys, finished_at)
			VALUES ($1, $1, 38, '{priority}', now())`,
	} {
		_, err := execIn(app, tenantA, q, tenantA)
		wantSQLState(t, "nava_app inserts "+name, err, "42501")
	}
	_, err := execIn(app, tenantA, insertBackfill, tenantB, tenantB, 39, "pending")
	wantSQLState(t, "nava_app in tenant A inserts tenant B's pending job", err, "42501")
	if got := queryString(t, super, `SELECT string_agg(target_version::text, ',' ORDER BY target_version) FROM schema_backfills
		WHERE tenant_id = $1 AND target_version >= 30`, tenantA); got != "30" {
		t.Errorf("tenant A's jobs from nava_app = %s, want only the pending 30", got)
	}

	// The keys hold the tenant: a job on tenant B's type, or a queued id
	// under tenant B's job, fails its foreign key.
	_, err = execIn(worker, tenantA, insertBackfill, tenantA, tenantB, 4, "pending")
	wantSQLState(t, "nava_worker queues tenant A's job on tenant B's type", err, "23503")
	_, err = execIn(worker, tenantA, insertBackfillPending, tenantA, tenantA, 7, tenantA)
	wantSQLState(t, "nava_worker queues an id under a job that does not exist", err, "23503")

	// A superseded job drops its queue and its failures with it (D8).
	if n, err := execIn(worker, tenantA, `DELETE FROM schema_backfills WHERE tenant_id = $1 AND entity_definition_id = $1 AND target_version = 2`, tenantA); err != nil || n != 1 {
		t.Fatalf("nava_worker deletes tenant A's job: %d, %v", n, err)
	}
	if got := queryString(t, super, `SELECT (SELECT count(*) FROM schema_backfill_pending WHERE tenant_id = $1 AND target_version = 2) || ',' ||
		(SELECT count(*) FROM schema_backfill_failures WHERE tenant_id = $1 AND target_version = 2)`, tenantA); got != "0,0" {
		t.Errorf("tenant A's queue and failures after its job's delete = %s, want 0,0", got)
	}
	if got := queryString(t, super, `SELECT (SELECT count(*) FROM schema_backfill_pending WHERE tenant_id = $1) || ',' ||
		(SELECT count(*) FROM schema_backfill_failures WHERE tenant_id = $1)`, tenantB); got != "1,1" {
		t.Errorf("tenant B's queue and failures after tenant A's writes = %s, want 1,1", got)
	}

	for _, role := range []string{"nava_onboard", "nava_groupsync", "nava_relay", "nava_purge", "nava_audit_reader", "nava_erasure"} {
		for _, tbl := range schemaBackfillTables {
			_, err := execIn(asRole(t, role), tenantA, "SELECT count(*) FROM "+tbl)
			wantSQLState(t, role+" reads "+tbl, err, "42501")
		}
	}
}

// Acceptance: the failure code is lower snake, 1 to 64 characters; a space,
// a quote or a 65th character is refused. The state CHECK is the contract's.
func TestP1001Checks(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedEntityTypes(t, super, tenantA)
	seedSchemaBackfill(t, super, tenantA)
	worker := asRole(t, "nava_worker")

	for name, code := range map[string]string{
		"a space":       "version conflict",
		"a quote":       "it's",
		"65 characters": strings.Repeat("a", 65),
		"empty":         "",
		"upper case":    "Conflict",
		"a digit":       "e42501",
	} {
		_, err := execIn(worker, tenantA, insertBackfillFailure, tenantA, tenantA, 2, newUUID(t, super), code)
		wantSQLState(t, "a failure code with "+name, err, "23514")
	}
	if n, err := execIn(worker, tenantA, insertBackfillFailure, tenantA, tenantA, 2, newUUID(t, super), strings.Repeat("a", 64)); err != nil || n != 1 {
		t.Errorf("control: a 64-character code: %d, %v", n, err)
	}
	for i, st := range []string{"filling", "running", "done", "failed_records", "superseded"} {
		if _, err := execIn(worker, tenantA, insertBackfill, tenantA, tenantA, 10+i, st); err != nil {
			t.Errorf("state %s: %v", st, err)
		}
		// One active job per tenant: only the first of filling and running fits.
		if st == "filling" {
			exec(t, super, "UPDATE schema_backfills SET state = 'done' WHERE tenant_id = $1 AND target_version = 10", tenantA)
		}
	}
	_, err := execIn(worker, tenantA, insertBackfill, tenantA, tenantA, 20, "queued")
	wantSQLState(t, "state queued", err, "23514")
	_, err = execIn(worker, tenantA, "UPDATE schema_backfills SET rewritten = -1 WHERE tenant_id = $1", tenantA)
	wantSQLState(t, "a negative count", err, "23514")
	_, err = execIn(worker, tenantA, insertBackfill, tenantA, tenantA, 0, "pending")
	wantSQLState(t, "target_version 0", err, "23514")
	if got := queryString(t, super, `SELECT state || ',' || rewritten || ',' || skipped || ',' || failed || ',' ||
		(started_at IS NULL)::text || ',' || (finished_at IS NULL)::text FROM schema_backfills
		WHERE tenant_id = $1 AND target_version = 2`, tenantA); got != "pending,0,0,0,true,true" {
		t.Errorf("a new job's defaults = %s, want pending,0,0,0,true,true", got)
	}
}

// Acceptance (D7 admission): with tenant A's job in filling, moving a second
// tenant A job to filling or running fails with 23505 on
// schema_backfills_one_active_per_tenant; tenant B's job still moves to
// filling; once A's first job is done, A's second one moves.
func TestP1001OneActiveJobPerTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedEntityTypes(t, super, tenantA, tenantB)
	seedSchemaBackfill(t, super, tenantA, tenantB)
	worker := asRole(t, "nava_worker")
	exec(t, super, insertBackfill, tenantA, tenantA, 3, "pending")

	const flip = `UPDATE schema_backfills SET state = $3, started_at = now()
		WHERE tenant_id = $1 AND entity_definition_id = $1 AND target_version = $2`
	if n, err := execIn(worker, tenantA, flip, tenantA, 2, "filling"); err != nil || n != 1 {
		t.Fatalf("tenant A's first job to filling: %d, %v", n, err)
	}
	for _, st := range []string{"filling", "running"} {
		_, err := execIn(worker, tenantA, flip, tenantA, 3, st)
		wantSQLState(t, "tenant A's second job to "+st, err, "23505")
		if c := constraintName(err); c != "schema_backfills_one_active_per_tenant" {
			t.Errorf("tenant A's second job to %s: constraint = %q, want schema_backfills_one_active_per_tenant", st, c)
		}
	}
	// A fresh insert straight into running is refused the same way.
	_, err := execIn(worker, tenantA, insertBackfill, tenantA, tenantA, 4, "running")
	wantSQLState(t, "a new tenant A job inserted running", err, "23505")
	if n, err := execIn(worker, tenantB, flip, tenantB, 2, "filling"); err != nil || n != 1 {
		t.Errorf("tenant B's job to filling while tenant A's fills: %d, %v", n, err)
	}
	// filling to running keeps the one slot.
	if n, err := execIn(worker, tenantA, flip, tenantA, 2, "running"); err != nil || n != 1 {
		t.Errorf("tenant A's first job filling to running: %d, %v", n, err)
	}
	if n, err := execIn(worker, tenantA, flip, tenantA, 2, "done"); err != nil || n != 1 {
		t.Fatalf("tenant A's first job to done: %d, %v", n, err)
	}
	if n, err := execIn(worker, tenantA, flip, tenantA, 3, "filling"); err != nil || n != 1 {
		t.Errorf("tenant A's second job to filling after the first is done: %d, %v", n, err)
	}
}

// Acceptance (grants drift): the catalog holds exactly ADR-009's grants on the
// three tables: nava_worker's DML, nava_app's SELECT, nothing for any other
// role or PUBLIC, and no column grant.
func TestP1001Grants(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)

	got := strings.Fields(queryString(t, super, `
		WITH t AS (SELECT oid, relname, relacl FROM pg_class
			WHERE oid = ANY ('{schema_backfills,schema_backfill_pending,schema_backfill_failures}'::regclass[])),
		g AS (
			SELECT t.relname::text AS tbl, coalesce(r.rolname::text, 'PUBLIC') AS role, a.privilege_type AS priv
			FROM t, aclexplode(t.relacl) a LEFT JOIN pg_roles r ON r.oid = a.grantee
			WHERE coalesce(r.rolname, '') <> 'nava_migrator'
			UNION ALL
			SELECT t.relname::text, 'column:' || att.attname, ''
			FROM t JOIN pg_attribute att ON att.attrelid = t.oid AND att.attacl IS NOT NULL
		)
		SELECT coalesce(string_agg(tbl || ':' || role || ':' || priv, ' ' ORDER BY tbl, role, priv), '') FROM g`))
	var want []string
	for _, tbl := range []string{"schema_backfill_failures", "schema_backfill_pending", "schema_backfills"} {
		if tbl == "schema_backfills" {
			want = append(want, tbl+":nava_app:INSERT")
		}
		want = append(want, tbl+":nava_app:SELECT")
		for _, p := range []string{"DELETE", "INSERT", "SELECT", "UPDATE"} {
			want = append(want, tbl+":nava_worker:"+p)
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("grants on the backfill tables:\n got %v\nwant %v", got, want)
	}
}

// tenantLimitsMigration creates tenant_limits (TICK-751).
const tenantLimitsMigration = "20261126163700_tenant_limits.sql"

// beforeTenantLimits is m and a without tenant_limits when fsys stops before
// tenantLimitsMigration. beforeNumberingMode calls it.
func beforeTenantLimits(fsys fs.FS, m *grants.Manifest, a *audited.Manifest) (*grants.Manifest, *audited.Manifest) {
	if _, err := fs.Stat(fsys, tenantLimitsMigration); err == nil {
		return m, a
	}
	gm := *m
	gm.Roles = make(map[string]grants.Role, len(m.Roles))
	for name, r := range m.Roles {
		tables := maps.Clone(r.Tables)
		delete(tables, "tenant_limits")
		r.Tables = tables
		gm.Roles[name] = r
	}
	am := *a
	am.Audited = maps.Clone(a.Audited)
	am.Exempt = maps.Clone(a.Exempt)
	delete(am.Audited, "tenant_limits")
	delete(am.Exempt, "tenant_limits")
	return &gm, &am
}
