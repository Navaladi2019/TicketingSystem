//go:build integration

package dbtest

import (
	"context"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5"
	"github.com/pressly/goose/v3"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// S17 P6-01 (TICK-665), ADR-005 §4.1 and §4.2 steps M1 and M2:
// record_counters gains scope_key and period_end, and the scoped unique index
// is built concurrently beside the old primary key. Runbook:
// docs/data/record-counters-scope-key.md.

const (
	scopeColumnsMigration = "20261112193000_record_counters_scope_columns.sql"
	scopedKeyMigration    = "20261112193200_record_counters_scoped_key.sql"
	periodEndMigration    = "20261112193300_record_counters_period_end.sql"
	// m3Migration is S18's M3 (TICK-670). The S17 tests here run on the
	// schema before it: the expand window M1/M2 opened and M3 closes.
	m3Migration = "20261113034010_record_counters_pkey_scope_key.sql"
	// scopeColumnsBefore is the version just before M1.
	scopeColumnsBefore = 20261112192959

	// takeNumberR0 is the pre-S17 (R0) upsert: tenant $1, type $2. After M3
	// it fails with 42P10: no unique constraint matches its target.
	takeNumberR0 = `INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2)
		ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1
		RETURNING next - 1`
	// takeNumberR1 is R1's upsert, still writing the default scope: tenant
	// $1, type $2.
	takeNumberR1 = `INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES ($1, $2, '', 2)
		ON CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next = record_counters.next + 1
		RETURNING next - 1`
	// takeScoped is the S18 upsert of a non-default scope: tenant $1, type
	// $2, scope $3. Its INSERT arm sets period_end; its UPDATE arm never does.
	takeScoped = `INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, period_end, next)
		VALUES ($1, $2, $3, now() + interval '1 day', 2)
		ON CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next = record_counters.next + 1
		RETURNING next - 1`

	// counterIndexes lists record_counters' indexes with validity.
	counterIndexes = `SELECT string_agg(c.relname || ' valid=' || i.indisvalid || ' ' || regexp_replace(pg_get_indexdef(i.indexrelid), '^.* USING ', ''),
		E'\n' ORDER BY c.relname)
		FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid WHERE i.indrelid = 'record_counters'::regclass`
	counterColumns = `SELECT string_agg(a.attname || ' ' || format_type(a.atttypid, a.atttypmod) || ' notnull=' || a.attnotnull
		|| coalesce(' default=' || pg_get_expr(d.adbin, d.adrelid), ''), E'\n' ORDER BY a.attnum)
		FROM pg_attribute a LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		WHERE a.attrelid = 'record_counters'::regclass AND a.attnum > 0 AND NOT a.attisdropped`
)

// migratedBeforeM3 is a fresh cluster with every migration before M3 and the
// grants applied: the R1 schema, both unique indexes in place.
func migratedBeforeM3(t *testing.T) *grants.Manifest {
	t.Helper()
	m := freshCluster(t)
	if _, err := migrateFS(t, m, migrationsBefore(t, m3Migration)); err != nil {
		t.Fatalf("migrate to before %s: %v", m3Migration, err)
	}
	return m
}

// takeIn runs a numbering upsert in a tenant transaction and returns the
// number it took.
func takeIn(conn *pgx.Conn, tenant, sql string, args ...any) (int64, error) {
	var n int64
	err := inTenantTx(conn, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), sql, args...).Scan(&n)
	})
	return n, err
}

// Acceptance: M1 and M2 add the columns, the CHECK and both indexes to a
// table that already holds counters, without rewriting it; the existing
// counters read the default scope and keep their next; the old primary key
// stays; up, down and up again all succeed.
func TestP601CounterScopeMigration(t *testing.T) {
	m := freshCluster(t)
	if err := migrateOnlyFS(t, m, migrationsBefore(t, scopeColumnsMigration)); err != nil {
		t.Fatalf("migrate to before %s: %v", scopeColumnsMigration, err)
	}
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB)
	exec(t, super, "UPDATE record_counters SET next = 43 WHERE tenant_id = $1", tenantA)
	fileBefore := queryString(t, super, `SELECT pg_relation_filenode('record_counters')::text`)

	// With P6-10's purge policies out of order, as migrateFS applies them, so
	// the Down below knows every applied version.
	beforeM3 := withCounterPurge(t, migrationsBefore(t, m3Migration))
	if _, err := migrateFS(t, m, beforeM3); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if got := queryString(t, super, `SELECT pg_relation_filenode('record_counters')::text`); got != fileBefore {
		t.Errorf("record_counters filenode %s -> %s: the migration rewrote the table", fileBefore, got)
	}
	if got := queryString(t, super, counterColumns); got != strings.Join([]string{
		"tenant_id uuid notnull=true",
		"entity_definition_id uuid notnull=true",
		"next bigint notnull=true",
		"scope_key text notnull=true default=''::text",
		"period_end timestamp with time zone notnull=false",
	}, "\n") {
		t.Errorf("record_counters columns:\n%s", got)
	}
	if got := queryString(t, super, `SELECT string_agg(tenant_id || ' ' || next || ' [' || scope_key || '] ' || coalesce(period_end::text, 'NULL'), ',' ORDER BY tenant_id)
		FROM record_counters`); got != tenantA+" 43 [] NULL,"+tenantB+" 2 [] NULL" {
		t.Errorf("existing counters read %q", got)
	}
	if got := queryString(t, super, `SELECT convalidated || ' ' || pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid = 'record_counters'::regclass AND conname = 'record_counters_scope_key_len'`); got != "true CHECK ((length(scope_key) <= 128))" {
		t.Errorf("record_counters_scope_key_len: %s", got)
	}
	wantIndexes := strings.Join([]string{
		"record_counters_period_end valid=true btree (period_end) WHERE (period_end IS NOT NULL)",
		"record_counters_pkey valid=true btree (tenant_id, entity_definition_id)",
		"record_counters_scoped_key valid=true btree (tenant_id, entity_definition_id, scope_key)",
	}, "\n")
	if got := queryString(t, super, counterIndexes); got != wantIndexes {
		t.Errorf("record_counters indexes:\n%s\nwant:\n%s", got, wantIndexes)
	}
	if got := queryString(t, super, `SELECT pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid = 'record_counters'::regclass AND contype = 'p'`); got != "PRIMARY KEY (tenant_id, entity_definition_id)" {
		t.Errorf("primary key before M3: %s", got)
	}

	sqlDB := openSQL(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, beforeM3,
		goose.WithAllowOutofOrder(true), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := p.DownTo(ctx, scopeColumnsBefore); err != nil {
		t.Fatalf("down: %v", err)
	}
	if got := queryString(t, super, counterColumns); got != "tenant_id uuid notnull=true\nentity_definition_id uuid notnull=true\nnext bigint notnull=true" {
		t.Errorf("after down: columns\n%s", got)
	}
	if got := queryString(t, super, counterIndexes); got != "record_counters_pkey valid=true btree (tenant_id, entity_definition_id)" {
		t.Errorf("after down: indexes\n%s", got)
	}
	if got := queryString(t, super, `SELECT string_agg(next::text, ',' ORDER BY tenant_id) FROM record_counters`); got != "43,2" {
		t.Errorf("after down: next %s, want 43,2", got)
	}
	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up again: %v", err)
	}
	if got := queryString(t, super, counterIndexes); got != wantIndexes {
		t.Errorf("after up again: indexes\n%s", got)
	}
}

// Pinned test 3, first half (ADR-005 §4.2): between M2 and M3, R0's
// two-column upsert and R1's three-column upsert both succeed on the same row
// and advance the same next, as nava_app and nava_worker; a non-empty
// scope_key for a type that already has a row fails on the old primary key.
func TestP601CounterScopeExpandWindow(t *testing.T) {
	migratedBeforeM3(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA)
	seedEntityTypes(t, super, tenantB)

	want := int64(1)
	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		for _, q := range []struct{ name, sql string }{{"R0", takeNumberR0}, {"R1", takeNumberR1}, {"R0", takeNumberR0}, {"R1", takeNumberR1}} {
			want++ // the seeded row's next is 2: the first take returns 2
			got, err := takeIn(conn, tenantA, q.sql, tenantA, tenantA)
			if err != nil || got != want {
				t.Fatalf("%s %s upsert: %d, %v; want %d, nil", role, q.name, got, err, want)
			}
		}
	}
	if got := queryString(t, super, `SELECT count(*) || ' ' || max(next) || ' [' || max(scope_key) || ']' FROM record_counters WHERE tenant_id = $1`, tenantA); got != "1 10 []" {
		t.Errorf("tenant A counters (rows, next, scope) = %s, want 1 10 []", got)
	}

	// A scope row for a type that already has one: the old key refuses it,
	// whichever arbiter the statement names.
	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		_, err := takeIn(conn, tenantA, takeScoped, tenantA, tenantA, "e1|d:202611")
		if code, con := pgErrDetail(err); code != "23505" || con != "record_counters_pkey" {
			t.Errorf("%s: scoped upsert on a type with a row: %v, want 23505 on record_counters_pkey", role, err)
		}
		_, err = execIn(conn, tenantA, `INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES ($1, $1, 'e1|f:BLR', 2)`, tenantA)
		if code, con := pgErrDetail(err); code != "23505" || con != "record_counters_pkey" {
			t.Errorf("%s: plain scoped insert on a type with a row: %v, want 23505 on record_counters_pkey", role, err)
		}
	}

	// A type with no row yet: the INSERT arm writes scope_key and period_end
	// (table-level INSERT), and the UPDATE arm cannot touch either (UPDATE
	// (next) only).
	app := asRole(t, "nava_app")
	if got, err := takeIn(app, tenantB, takeScoped, tenantB, tenantB, "e1|d:202611"); err != nil || got != 1 {
		t.Fatalf("nava_app first scoped take: %d, %v; want 1, nil", got, err)
	}
	if got, err := takeIn(app, tenantB, takeScoped, tenantB, tenantB, "e1|d:202611"); err != nil || got != 2 {
		t.Fatalf("nava_app second scoped take: %d, %v; want 2, nil", got, err)
	}
	if got := queryBool(t, super, `SELECT period_end IS NOT NULL FROM record_counters WHERE tenant_id = $1`, tenantB); !got {
		t.Error("the INSERT arm did not set period_end")
	}
	for _, role := range []string{"nava_app", "nava_worker"} {
		for _, col := range []string{"scope_key = 'x'", "period_end = now()"} {
			_, err := execIn(asRole(t, role), tenantB, "UPDATE record_counters SET "+col+" WHERE tenant_id = $1", tenantB)
			wantSQLState(t, role+" updates "+col, err, "42501")
		}
	}

	// RLS holds on the new key: tenant A takes from tenant B's scope row.
	_, err := takeIn(app, tenantA, takeScoped, tenantB, tenantB, "e1|d:202611")
	wantSQLState(t, "tenant A takes from tenant B's scoped counter", err, "42501")
	if got := queryString(t, super, `SELECT next::text FROM record_counters WHERE tenant_id = $1`, tenantB); got != "3" {
		t.Errorf("tenant B's scoped next = %s, want 3", got)
	}
}

// Acceptance: the CHECK takes a 128-character scope_key and refuses a
// 129-character one, on insert and on update (the superuser bypasses RLS and
// grants, not CHECKs).
func TestP601CounterScopeKeyLength(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA)
	seedEntityTypes(t, super, tenantB)

	_, err := takeIn(asRole(t, "nava_app"), tenantB, takeScoped, tenantB, tenantB, strings.Repeat("x", 129))
	wantSQLState(t, "nava_app inserts a 129-character scope_key", err, "23514")
	if got, err := takeIn(asRole(t, "nava_app"), tenantB, takeScoped, tenantB, tenantB, strings.Repeat("x", 128)); err != nil || got != 1 {
		t.Fatalf("nava_app inserts a 128-character scope_key: %d, %v; want 1, nil", got, err)
	}
	_, err = super.Exec(context.Background(), "UPDATE record_counters SET scope_key = repeat('y', 129) WHERE tenant_id = $1", tenantA)
	if code, con := pgErrDetail(err); code != "23514" || con != "record_counters_scope_key_len" {
		t.Errorf("129-character update: %v, want 23514 on record_counters_scope_key_len", err)
	}
}

// Acceptance: an interrupted M2 leaves an INVALID index under the name; the
// re-run drops it and builds a valid one. A re-run over a valid index keeps it
// (the same relation), so a serving R1 never loses its arbiter.
func TestP601CounterScopeIndexRepair(t *testing.T) {
	m := migratedBeforeM3(t)
	super := superNava(t)
	sqlDB := openSQL(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	rerun := func(name string) {
		t.Helper()
		b, err := fs.ReadFile(db.Migrations(), name)
		if err != nil {
			t.Fatal(err)
		}
		p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, fstest.MapFS{name: {Data: b}},
			goose.WithDisableVersioning(true), goose.WithDisableGlobalRegistry(true))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.Up(context.Background()); err != nil {
			t.Fatalf("re-run %s: %v", name, err)
		}
	}
	for name, index := range map[string]string{scopedKeyMigration: "record_counters_scoped_key", periodEndMigration: "record_counters_period_end"} {
		oid := queryString(t, super, `SELECT to_regclass($1)::oid::text`, index)
		rerun(name)
		if got := queryString(t, super, `SELECT to_regclass($1)::oid::text`, index); got != oid {
			t.Errorf("%s: a re-run over a valid index rebuilt it (oid %s -> %s)", index, oid, got)
		}

		// What a CREATE INDEX CONCURRENTLY that failed after its catalogue
		// entry leaves behind.
		exec(t, super, `UPDATE pg_index SET indisvalid = false WHERE indexrelid = to_regclass($1)`, index)
		if queryBool(t, super, `SELECT indisvalid FROM pg_index WHERE indexrelid = to_regclass($1)`, index) {
			t.Fatalf("%s: could not mark the index invalid", index)
		}
		rerun(name)
		if !queryBool(t, super, `SELECT indisvalid AND indisready FROM pg_index WHERE indexrelid = to_regclass($1)`, index) {
			t.Errorf("%s: still invalid after the re-run", index)
		}
		if got := queryString(t, super, `SELECT to_regclass($1)::oid::text`, index); got == oid {
			t.Errorf("%s: the re-run kept the invalid index", index)
		}
	}
	if got := queryString(t, super, counterIndexes); !strings.Contains(got, "record_counters_scoped_key valid=true") || !strings.Contains(got, "record_counters_period_end valid=true") {
		t.Errorf("record_counters indexes after the repairs:\n%s", got)
	}
}
