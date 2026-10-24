//go:build integration

package dbtest

import (
	"context"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
)

// T10 (TICK-66): the catalogue gate, ADR-001 Annex B CI-1, CI-3, CI-4. The
// migrator runs it after every run (R2), so every test here that migrates
// proves the real migrations pass it; the tests below prove it fails.

// wantProblem fails the test unless one of problems contains want.
func wantProblem(t *testing.T, problems []string, want string) {
	t.Helper()
	for _, p := range problems {
		if strings.Contains(p, want) {
			return
		}
	}
	t.Errorf("no problem contains %q; got %q", want, problems)
}

// The real migrations pass the gate, which reports nothing.
func TestT10GatePassesRealMigrations(t *testing.T) {
	migratedPlatform(t)
	problems, err := catalog.Check(context.Background(), superNava(t), manifest(t), globalTables(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range problems {
		t.Error(p)
	}
}

// Negative fixture migrations for CI-1: each is a migration the migrator
// applies, after which the run fails on the gate with that problem. Tenant
// tables are discovered fail-closed, so a table in a new schema counts too.
func TestT10GateFailsMigratorRun(t *testing.T) {
	for _, c := range []struct{ name, up, want string }{
		{"no_rls",
			"CREATE TABLE t10_fixture (tenant_id uuid NOT NULL);",
			"CI-1: tenant table public.t10_fixture does not ENABLE ROW LEVEL SECURITY"},
		{"no_force",
			"CREATE TABLE t10_fixture (tenant_id uuid NOT NULL);\nALTER TABLE t10_fixture ENABLE ROW LEVEL SECURITY;",
			"CI-1: tenant table public.t10_fixture does not FORCE ROW LEVEL SECURITY"},
		{"no_tenant_key",
			"CREATE TABLE t10_fixture (id int);\nALTER TABLE t10_fixture ENABLE ROW LEVEL SECURITY;\nALTER TABLE t10_fixture FORCE ROW LEVEL SECURITY;",
			"CI-1: table public.t10_fixture has no tenant key (tenant_id uuid NOT NULL) and is not in db/global-tables.yaml"},
		{"nullable_tenant_key",
			"CREATE TABLE t10_fixture (tenant_id uuid);\nALTER TABLE t10_fixture ENABLE ROW LEVEL SECURITY;\nALTER TABLE t10_fixture FORCE ROW LEVEL SECURITY;",
			"CI-1: table public.t10_fixture has no tenant key"},
		{"other_schema",
			"CREATE SCHEMA t10s;\nCREATE TABLE t10s.fixture (tenant_id uuid NOT NULL);",
			"CI-1: tenant table t10s.fixture does not FORCE ROW LEVEL SECURITY"},
		{"policy_off_allow_list",
			"CREATE POLICY t10_leak ON records FOR SELECT TO nava_app USING (true);",
			"CI-1: policy t10_leak on records"},
		{"security_definer",
			"CREATE FUNCTION t10_definer() RETURNS int LANGUAGE sql SECURITY DEFINER AS 'SELECT 1';",
			"CI-1: function t10_definer() is SECURITY DEFINER"},
		{"view_not_security_invoker",
			"CREATE VIEW t10_view AS SELECT id FROM records;",
			"CI-1: view public.t10_view is not security_invoker"},
		{"materialized_view",
			"CREATE MATERIALIZED VIEW t10_mv AS SELECT id FROM records;",
			"CI-1: materialized view public.t10_mv exists"},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := freshCluster(t)
			_, err := migrateFS(t, m, withExtraMigration(t, "99999_t10_fixture.sql", "-- +goose Up\n"+c.up+"\n"))
			if err == nil {
				t.Fatalf("migrator passed the gate with %s", c.up)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("migrate error does not contain %q:\n%v", c.want, err)
			}
		})
	}
}

// A security_invoker view passes; the gate is not just failing every view.
func TestT10GatePassesSecurityInvokerView(t *testing.T) {
	m := freshCluster(t)
	fsys := withExtraMigration(t, "99999_t10_fixture.sql",
		"-- +goose Up\nCREATE VIEW t10_view WITH (security_invoker = true) AS SELECT id FROM records;\n")
	if _, err := migrateFS(t, m, fsys); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}

// R2: a migrator run with nothing to apply still runs the gate, so a fault
// made outside the migrations fails the next run.
func TestT10GateRunsWhenNothingToApply(t *testing.T) {
	m := freshCluster(t)
	migrateAll(t, m)
	exec(t, superNava(t), "CREATE TABLE t10_drift (tenant_id uuid NOT NULL)")
	_, err := migrateFS(t, m, db.Migrations())
	if err == nil {
		t.Fatal("migrator passed the gate with an unprotected table")
	}
	if !strings.Contains(err.Error(), "CI-1: tenant table public.t10_drift does not ENABLE ROW LEVEL SECURITY") {
		t.Fatalf("migrate error: %v", err)
	}
}

// Faults the migrator cannot make, or that its grant manifest step heals on
// the objects it owns (CI-3, CI-4): each is made as the superuser in a
// rolled-back transaction, and the gate must report it.
func TestT10GateCatchesCatalogueDrift(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	ctx := context.Background()
	m, g := manifest(t), globalTables(t)
	for _, c := range []struct{ name, sql, want string }{
		// CI-3: memberships, either way round, and owning a platform object.
		{"runtime_member_of_runtime", "GRANT nava_relay TO nava_app",
			"CI-3: nava_app is a member of nava_relay"},
		{"member_of_migrator", "CREATE ROLE t10_intruder; GRANT nava_migrator TO t10_intruder",
			"CI-3: t10_intruder is a member of nava_migrator"},
		{"runtime_owns_table", "ALTER TABLE records OWNER TO nava_app",
			"CI-3: nava_app is a member of nava_app, owner of relation records"},
		// CI-3: PUBLIC, through every acl the gate explodes.
		{"public_table", "GRANT SELECT ON records TO PUBLIC",
			"CI-3: PUBLIC holds SELECT on relation records"},
		{"public_column", "GRANT SELECT (id) ON records TO PUBLIC",
			"CI-3: PUBLIC holds SELECT on column records.id"},
		{"public_database", "GRANT CONNECT ON DATABASE nava TO PUBLIC",
			"CI-3: PUBLIC holds CONNECT on database nava"},
		{"public_schema", "GRANT CREATE ON SCHEMA public TO PUBLIC",
			"CI-3: PUBLIC holds CREATE on schema public"},
		{"public_function_default_acl", "CREATE FUNCTION t10_fn() RETURNS int LANGUAGE sql AS 'SELECT 1'",
			"CI-3: PUBLIC holds EXECUTE on function t10_fn()"},
		{"public_default_privilege", "ALTER DEFAULT PRIVILEGES FOR ROLE nava_migrator GRANT SELECT ON TABLES TO PUBLIC",
			"CI-3: PUBLIC holds SELECT on default privilege of nava_migrator on tables"},
		// CI-4: effective privileges equal db/grants.yaml.
		{"extra_table_privilege", "GRANT DELETE ON records TO nava_app",
			"CI-4: nava_app has DELETE on table public.records, which db/grants.yaml does not list"},
		{"missing_table_privilege", "REVOKE SELECT ON records FROM nava_app",
			"CI-4: nava_app lacks SELECT on table public.records, which db/grants.yaml lists"},
		{"extra_column_privilege", "GRANT UPDATE (state) ON tenants TO nava_app",
			"CI-4: nava_app has UPDATE on column public.tenants.state, which db/grants.yaml does not list"},
		{"grant_option", "GRANT SELECT ON records TO nava_app WITH GRANT OPTION",
			"CI-4: nava_app holds SELECT on table public.records WITH GRANT OPTION"},
		{"privilege_through_public", "GRANT SELECT ON idempotency_keys TO PUBLIC",
			"CI-4: nava_relay has SELECT on table public.idempotency_keys, which db/grants.yaml does not list"},
	} {
		t.Run(c.name, func(t *testing.T) {
			tx, err := super.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(ctx) }()
			if _, err := tx.Exec(ctx, c.sql); err != nil {
				t.Fatalf("%s: %v", c.sql, err)
			}
			problems, err := catalog.Check(ctx, tx, m, g)
			if err != nil {
				t.Fatal(err)
			}
			wantProblem(t, problems, c.want)
		})
	}
}

// db/global-tables.yaml exempts a listed table from RLS and the tenant key,
// nothing else, and an entry for a table that does not exist fails.
func TestT10GateGlobalTables(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	ctx := context.Background()
	m := manifest(t)
	g, err := catalog.ParseGlobalTables([]byte("tables:\n  t10_global: { reason: test }\n  public.t10_gone: { reason: test }\n"))
	if err != nil {
		t.Fatal(err)
	}
	tx, err := super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "CREATE TABLE t10_global (code text); ALTER TABLE t10_global OWNER TO nava_migrator"); err != nil {
		t.Fatal(err)
	}
	problems, err := catalog.Check(ctx, tx, m, g)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range problems {
		if strings.Contains(p, "t10_global") {
			t.Errorf("listed table reported: %s", p)
		}
	}
	wantProblem(t, problems, "CI-1: db/global-tables.yaml lists public.t10_gone, which does not exist")

	// Listed, but granted off the manifest: CI-4 still applies.
	if _, err := tx.Exec(ctx, "GRANT SELECT ON t10_global TO nava_app"); err != nil {
		t.Fatal(err)
	}
	if problems, err = catalog.Check(ctx, tx, m, g); err != nil {
		t.Fatal(err)
	}
	wantProblem(t, problems, "CI-4: nava_app has SELECT on table public.t10_global")
}
