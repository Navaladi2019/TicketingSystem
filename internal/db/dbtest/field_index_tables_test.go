//go:build integration

package dbtest

import (
	"context"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pressly/goose/v3"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
)

// P7-01 (TICK-741, ADR-006 D2, D3, D14): record_index_fields and the four
// typed side tables rfv_text, rfv_num, rfv_ts, rfv_bool.

const (
	fieldIndexMigration = "20261127110000_field_index_tables.sql"
	// beforeFieldIndex is the version just before fieldIndexMigration.
	beforeFieldIndex = 20261125164959

	fieldA = "0e000000-0000-4000-8000-0000000007a1" // tenant A's registry row
	fieldB = "0e000000-0000-4000-8000-0000000007b1" // tenant B's registry row

	// insertField: tenant $1, id $2, entity type $3, value_type $4. The
	// field key is shared by every tenant's row, so a lookup by it alone
	// would find tenant B's.
	insertField = `INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, requested_version)
		VALUES ($1, $2, $3, 'priority', $4, 'backfilling', 1)`
)

// fieldIndexTables are the tables fieldIndexMigration creates, registry first.
var fieldIndexTables = []string{"record_index_fields", "rfv_text", "rfv_num", "rfv_ts", "rfv_bool"} //nolint:gochecknoglobals // read-only test/bench fixture, set once

// sideTables maps each value type to its side table and a literal of its type.
//
//nolint:gochecknoglobals // read-only test/bench fixture, set once
var sideTables = []struct {
	vt    fieldindex.ValueType
	table string
	value string
}{
	{fieldindex.TypeText, "rfv_text", "'high'"},
	{fieldindex.TypeNum, "rfv_num", "1.5"},
	{fieldindex.TypeTS, "rfv_ts", "'2026-11-15T00:00:00Z'"},
	{fieldindex.TypeBool, "rfv_bool", "true"},
}

// insertSide is the INSERT of one side row into table: tenant $1, field $2,
// record $3, ord 0.
func insertSide(table, value string) string {
	return "INSERT INTO " + table + " (tenant_id, field_id, record_id, ord, value) VALUES ($1, $2, $3, 0, " + value + ")"
}

// sideField is the registry id of value type vt's field in the tenant whose
// field id is base: one registry row per side table, so each side row names a
// field of its own type.
// The last digit is c, d, e or f, never base's own.
func sideField(base string, i int) string { return base[:len(base)-1] + string("cdef"[i]) }

// beforeFieldIndexTables is m and a as a run of fsys sees them when fsys stops
// before fieldIndexMigration: without the five tables in either manifest.
// beforeNumberingMode calls it, so every older-schema run gets it.
func beforeFieldIndexTables(fsys fs.FS, m *grants.Manifest, a *audited.Manifest) (*grants.Manifest, *audited.Manifest) {
	if _, err := fs.Stat(fsys, fieldIndexMigration); err == nil {
		return m, a
	}
	gm := *m
	gm.Roles = make(map[string]grants.Role, len(m.Roles))
	for name, r := range m.Roles {
		tables := make(map[string]map[string][]string, len(r.Tables))
		for tbl, privs := range r.Tables {
			if !slices.Contains(fieldIndexTables, tbl) {
				tables[tbl] = privs
			}
		}
		r.Tables = tables
		gm.Roles[name] = r
	}
	am := *a
	am.Audited = maps.Clone(a.Audited)
	am.Exempt = maps.Clone(a.Exempt)
	for _, tbl := range fieldIndexTables {
		delete(am.Audited, tbl)
		delete(am.Exempt, tbl)
	}
	return &gm, &am
}

// seedFieldIndex writes, as the superuser, for each tenant: one registry row
// per value type on seedEntityTypes' type (ids sideField(base, i), base being
// fieldA for tenant A and fieldB for any other) and one side row for each on
// seedTenants' record (id = the tenant id).
func seedFieldIndex(t *testing.T, super *pgx.Conn, tenants ...string) {
	t.Helper()
	for _, tenant := range tenants {
		base := fieldB
		if tenant == tenantA {
			base = fieldA
		}
		for i, s := range sideTables {
			id := sideField(base, i)
			exec(t, super, strings.Replace(insertField, "'priority'", "'f_"+string(s.vt)+"'", 1), tenant, id, tenant, string(s.vt))
			exec(t, super, insertSide(s.table, s.value), tenant, id, tenant)
		}
	}
}

// Acceptance (CI-2, the leakproof catalog): every comparison operator a side
// table's filter uses is leakproof, so the planner may push it below forced
// RLS as an index condition; the value columns have exactly the D3 types and
// collation, and no side table has a numeric column.
func TestP701LeakproofCatalog(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)

	for _, typ := range []string{"text", "float8", "timestamptz", "bool"} {
		rows, err := super.Query(t.Context(), `SELECT o.oprname::text, p.proname::text, p.proleakproof
			FROM pg_operator o JOIN pg_proc p ON p.oid = o.oprcode
			WHERE o.oprleft = $1::regtype AND o.oprright = $1::regtype AND o.oprname IN ('=', '<', '<=', '>', '>=')
			ORDER BY o.oprname`, typ)
		if err != nil {
			t.Fatal(err)
		}
		var ops []string
		for rows.Next() {
			var op, fn string
			var leakproof bool
			if err := rows.Scan(&op, &fn, &leakproof); err != nil {
				t.Fatal(err)
			}
			ops = append(ops, op)
			if !leakproof {
				t.Errorf("%s %s (%s) is not leakproof: a filter on it stays a Filter above RLS", typ, op, fn)
			}
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		if want := []string{"<", "<=", "=", ">", ">="}; !slices.Equal(ops, want) {
			t.Errorf("%s comparison operators = %v, want %v", typ, ops, want)
		}
	}
	// The control: numeric's comparisons are not leakproof, which is why
	// num is double precision. A server where they were would not need it,
	// but this test would still pass for the side tables; it fails here so
	// a changed catalog is looked at.
	if queryBool(t, super, `SELECT bool_or(p.proleakproof) FROM pg_operator o JOIN pg_proc p ON p.oid = o.oprcode
		WHERE o.oprleft = 'numeric'::regtype AND o.oprright = 'numeric'::regtype AND o.oprname IN ('=', '<')`) {
		t.Error("control: numeric = or < is leakproof on this server; revisit ADR-006 D3")
	}

	for _, s := range sideTables {
		got := queryString(t, super, `SELECT format_type(a.atttypid, a.atttypmod) || ' ' || coalesce(c.collname, '-') || ' ' || a.attnotnull::text
			FROM pg_attribute a LEFT JOIN pg_collation c ON c.oid = a.attcollation
			WHERE a.attrelid = $1::regclass AND a.attname = 'value'`, s.table)
		want := map[string]string{
			"rfv_text": "text C true", "rfv_num": "double precision - true",
			"rfv_ts": "timestamp with time zone - true", "rfv_bool": "boolean - true",
		}[s.table]
		if got != want {
			t.Errorf("%s.value = %s, want %s", s.table, got, want)
		}
	}
	if got := queryString(t, super, `SELECT coalesce(string_agg(attrelid::regclass::text || '.' || attname, ', '), '') FROM pg_attribute
		WHERE attrelid = ANY ('{rfv_text,rfv_num,rfv_ts,rfv_bool,record_index_fields}'::regclass[]) AND attnum > 0
		  AND NOT attisdropped AND atttypid = 'numeric'::regtype`); got != "" {
		t.Errorf("numeric columns: %s, want none (numeric comparisons are not leakproof)", got)
	}
}

// Acceptance: the keys, indexes, foreign keys and statistics are D3's, with
// the contract's constraint names, and RLS is enabled and forced on all five.
func TestP701Shape(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)

	for _, s := range sideTables {
		tbl := s.table
		for name, want := range map[string]string{
			tbl + "_pkey":      "PRIMARY KEY (tenant_id, record_id, field_id, ord)",
			tbl + "_record_fk": "FOREIGN KEY (tenant_id, record_id) REFERENCES records(tenant_id, id)",
			tbl + "_field_fk":  "FOREIGN KEY (tenant_id, field_id) REFERENCES record_index_fields(tenant_id, id)",
		} {
			if got := queryString(t, super, `SELECT coalesce((SELECT pg_get_constraintdef(oid) FROM pg_constraint
				WHERE conrelid = $1::regclass AND conname = $2), '')`, tbl, name); got != want {
				t.Errorf("%s = %q, want %q", name, got, want)
			}
		}
		// NO ACTION both ways, never CASCADE: a record or a registry row is
		// not deleted under its side rows.
		if got := queryString(t, super, `SELECT string_agg(confdeltype::text || confupdtype::text, ',' ORDER BY conname)
			FROM pg_constraint WHERE conrelid = $1::regclass AND contype = 'f'`, tbl); got != "aa,aa" {
			t.Errorf("%s foreign-key actions = %s, want NO ACTION (aa,aa)", tbl, got)
		}
		if got := queryString(t, super, `SELECT pg_get_indexdef($1::regclass)`, tbl+"_value"); !strings.HasSuffix(got,
			"USING btree (tenant_id, field_id, value, record_id)") {
			t.Errorf("%s_value = %s", tbl, got)
		}
		wantStats := "rfv_X_field_mcv:m:tenant_id,field_id:-1"
		if s.vt == fieldindex.TypeText {
			wantStats += " rfv_X_value_mcv:m:tenant_id,field_id,value:-1"
		}
		if s.vt == fieldindex.TypeBool {
			wantStats += " rfv_X_value_mcv:m:tenant_id,field_id,value:10000"
		}
		wantStats = strings.ReplaceAll(wantStats, "rfv_X", tbl)
		if got := queryString(t, super, `SELECT string_agg(x.stxname || ':' || array_to_string(x.stxkind, '') || ':' ||
				(SELECT string_agg(a.attname, ',' ORDER BY k.ord) FROM unnest(x.stxkeys) WITH ORDINALITY k(attnum, ord)
				 JOIN pg_attribute a ON a.attrelid = x.stxrelid AND a.attnum = k.attnum) || ':' || coalesce(x.stxstattarget, -1)::text,
				' ' ORDER BY x.stxname)
			FROM pg_statistic_ext x WHERE x.stxrelid = $1::regclass`, tbl); got != wantStats {
			t.Errorf("%s statistics = %s, want %s", tbl, got, wantStats)
		}
	}
	if got := queryString(t, super, `SELECT pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid = 'record_index_fields'::regclass AND conname = 'record_index_fields_key'`); got !=
		"UNIQUE (tenant_id, entity_definition_id, field_key)" {
		t.Errorf("record_index_fields_key = %s", got)
	}
	if got := queryString(t, super, `SELECT pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid = 'record_index_fields'::regclass AND conname = 'record_index_fields_entity_definition_fkey'`); got !=
		"FOREIGN KEY (tenant_id, entity_definition_id) REFERENCES entity_definitions(tenant_id, id)" {
		t.Errorf("record_index_fields_entity_definition_fkey = %s", got)
	}
	for _, tbl := range fieldIndexTables {
		if got := queryString(t, super, `SELECT relrowsecurity::text || relforcerowsecurity::text FROM pg_class
			WHERE oid = $1::regclass`, tbl); got != "truetrue" {
			t.Errorf("%s RLS enabled and forced = %s, want truetrue", tbl, got)
		}
	}

	// One policy per command and per role (D14), none FOR ALL; the catalogue
	// gate already pins each predicate to the canonical one.
	want := map[string]string{
		"record_index_fields": "DELETE:nava_worker INSERT:nava_app SELECT:nava_app SELECT:nava_worker UPDATE:nava_app UPDATE:nava_worker",
	}
	for _, s := range sideTables {
		want[s.table] = "DELETE:nava_app DELETE:nava_worker INSERT:nava_app INSERT:nava_worker SELECT:nava_app SELECT:nava_worker"
	}
	for tbl, w := range want {
		if got := queryString(t, super, `SELECT string_agg(cmd || ':' || array_to_string(roles, ','), ' ' ORDER BY cmd, roles::text)
			FROM pg_policies WHERE tablename = $1`, tbl); got != w {
			t.Errorf("%s policies = %s, want %s", tbl, got, w)
		}
	}
}

// Acceptance: the value_type and state CHECKs accept exactly the
// internal/fieldindex constants, and progress is 0 to 100.
func TestP701RegistryChecksMatchFieldindex(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedEntityTypes(t, super, tenantA)

	const ins = `INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, progress, requested_version)
		VALUES ($1, gen_random_uuid(), $1, $2, $3, $4, $5, 1)`
	n := 0
	for _, vt := range fieldindex.ValueTypes() {
		for _, st := range fieldindex.States() {
			n++
			if _, err := super.Exec(t.Context(), ins, tenantA, "k"+string(rune('a'+n)), string(vt), string(st), 0); err != nil {
				t.Errorf("value_type %s, state %s: %v", vt, st, err)
			}
		}
	}
	for name, args := range map[string][]any{
		"value_type numeric":    {"x1", "numeric", "ready", 0},
		"value_type empty":      {"x2", "", "ready", 0},
		"state gone":            {"x3", "text", "gone", 0},
		"progress 101":          {"x4", "text", "ready", 101},
		"progress -1":           {"x5", "text", "ready", -1},
		"an empty field key":    {"", "text", "ready", 0},
		"requested_version 0":   nil,
		"a duplicate field key": {"kb", "bool", "ready", 0},
	} {
		var err error
		if args == nil {
			_, err = super.Exec(t.Context(), `INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, requested_version)
				VALUES ($1, gen_random_uuid(), $1, 'x6', 'text', 'ready', 0)`, tenantA)
		} else {
			_, err = super.Exec(t.Context(), ins, append([]any{tenantA}, args...)...)
		}
		code := "23514"
		if name == "a duplicate field key" {
			code = "23505"
		}
		wantSQLState(t, name, err, code)
	}
	if got := queryString(t, super, "SELECT progress::text FROM record_index_fields WHERE field_key = 'kb'"); got != "0" {
		t.Errorf("progress default = %s, want 0", got)
	}
}

// Acceptance (CI-3): as nava_app and nava_worker in tenant A, every side
// table and the registry read and write only tenant A's rows; an INSERT with
// tenant B's tenant_id fails WITH CHECK; a side row whose field_id has no
// registry row (or names tenant B's, or tenant B's record) fails its foreign
// key; a registry DELETE with side rows left fails the foreign key.
func TestP701TenantIsolationAndKeys(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedEntityTypes(t, super, tenantA, tenantB)
	seedFieldIndex(t, super, tenantA, tenantB)

	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		for _, tbl := range fieldIndexTables {
			for _, c := range []struct {
				tenant, q string
				want      int
			}{
				{tenantA, "SELECT count(*) FROM " + tbl + " WHERE tenant_id <> $1", 0},
				{tenantA, "SELECT count(*) FROM " + tbl + " WHERE tenant_id = $1", map[bool]int{true: 4, false: 1}[tbl == "record_index_fields"]},
				{"", "SELECT count(*) FROM " + tbl + " WHERE tenant_id = $1 OR true", 0},
			} {
				var n int
				if err := inTenantTx(conn, c.tenant, func(tx pgx.Tx) error {
					return tx.QueryRow(t.Context(), c.q, tenantA).Scan(&n)
				}); err != nil || n != c.want {
					t.Errorf("%s in tenant %q: %s = %d, %v; want %d", role, c.tenant, c.q, n, err, c.want)
				}
			}
		}
		// The field key is the same in both tenants: a lookup by it alone
		// finds only tenant A's row.
		var owners string
		if err := inTenantTx(conn, tenantA, func(tx pgx.Tx) error {
			return tx.QueryRow(t.Context(), "SELECT string_agg(DISTINCT tenant_id::text, ',') FROM record_index_fields WHERE field_key = 'f_text'").Scan(&owners)
		}); err != nil || owners != tenantA {
			t.Errorf("%s in tenant A: registry rows keyed f_text belong to %s, %v; want only tenant A", role, owners, err)
		}

		for i, s := range sideTables {
			ins := insertSide(s.table, s.value)
			// WITH CHECK: tenant B's tenant_id, with tenant B's own field and record.
			_, err := execIn(conn, tenantA, strings.Replace(ins, "ord, value) VALUES ($1, $2, $3, 0", "ord, value) VALUES ($1, $2, $3, 1", 1),
				tenantB, sideField(fieldB, i), tenantB)
			wantSQLState(t, role+" in tenant A inserts a "+s.table+" row for tenant B", err, "42501")
			// The required registry key: a field nobody registered.
			_, err = execIn(conn, tenantA, ins, tenantA, newUUID(t, super), tenantA)
			wantSQLState(t, role+": a "+s.table+" row whose field has no registry row", err, "23503")
			// The keys hold the tenant: tenant B's field, tenant B's record.
			_, err = execIn(conn, tenantA, strings.Replace(ins, ", 0, ", ", 2, ", 1), tenantA, sideField(fieldB, i), tenantA)
			wantSQLState(t, role+": a "+s.table+" row naming tenant B's field", err, "23503")
			_, err = execIn(conn, tenantA, strings.Replace(ins, ", 0, ", ", 3, ", 1), tenantA, sideField(fieldA, i), tenantB)
			wantSQLState(t, role+": a "+s.table+" row naming tenant B's record", err, "23503")
			// The control: tenant A's own row, then its delete.
			if n, err := execIn(conn, tenantA, strings.Replace(ins, ", 0, ", ", 4, ", 1), tenantA, sideField(fieldA, i), tenantA); err != nil || n != 1 {
				t.Errorf("control: %s inserts tenant A's %s row: %d, %v", role, s.table, n, err)
			}
			if n, err := execIn(conn, tenantA, "DELETE FROM "+s.table+" WHERE ord = 4"); err != nil || n != 1 {
				t.Errorf("control: %s deletes tenant A's own %s row: %d rows, %v; want 1", role, s.table, n, err)
			}
			if n, err := execIn(conn, tenantA, "DELETE FROM "+s.table+" WHERE tenant_id = $1", tenantB); err != nil || n != 0 {
				t.Errorf("%s in tenant A deletes tenant B's %s rows: %d, %v; want 0", role, s.table, n, err)
			}
			if n, err := execIn(conn, "", "DELETE FROM "+s.table); err != nil || n != 0 {
				t.Errorf("%s with no tenant deletes %s rows: %d, %v; want 0", role, s.table, n, err)
			}
			// No UPDATE for anyone (D14): a value is rewritten as DELETE then INSERT.
			_, err = execIn(conn, tenantA, "UPDATE "+s.table+" SET ord = ord WHERE tenant_id = $1", tenantA)
			wantSQLState(t, role+" updates "+s.table, err, "42501")
		}
	}

	app, worker := asRole(t, "nava_app"), asRole(t, "nava_worker")
	// The registry as nava_app: a row for tenant B fails WITH CHECK, a row
	// on tenant B's entity type fails the foreign key, the control commits.
	_, err := execIn(app, tenantA, insertField, tenantB, newUUID(t, super), tenantB, "text")
	wantSQLState(t, "nava_app in tenant A registers a field for tenant B", err, "42501")
	_, err = execIn(app, tenantA, strings.Replace(insertField, "'priority'", "'iso_b'", 1), tenantA, newUUID(t, super), tenantB, "text")
	wantSQLState(t, "nava_app in tenant A registers a field on tenant B's entity type", err, "23503")
	if n, err := execIn(app, tenantA, insertField, tenantA, fieldA, tenantA, "num"); err != nil || n != 1 {
		t.Errorf("control: nava_app registers tenant A's field: %d, %v", n, err)
	}
	_, err = execIn(worker, tenantA, strings.Replace(insertField, "'priority'", "'iso_w'", 1), tenantA, newUUID(t, super), tenantA, "text")
	wantSQLState(t, "nava_worker registers a field", err, "42501")
	if n, err := execIn(app, tenantA, "UPDATE record_index_fields SET state = 'ready', progress = 100, updated_at = now() WHERE tenant_id = $1", tenantB); err != nil || n != 0 {
		t.Errorf("nava_app in tenant A flips tenant B's fields: %d, %v; want 0", n, err)
	}
	if n, err := execIn(worker, tenantA, "UPDATE record_index_fields SET state = 'ready', progress = 100, updated_at = now() WHERE tenant_id = $1", tenantB); err != nil || n != 0 {
		t.Errorf("nava_worker in tenant A flips tenant B's fields: %d, %v; want 0", n, err)
	}
	// UPDATE ... SET tenant_id is refused by the column grant before WITH CHECK.
	_, err = execIn(worker, tenantA, "UPDATE record_index_fields SET tenant_id = $2 WHERE tenant_id = $1", tenantA, tenantB)
	wantSQLState(t, "nava_worker moves tenant A's field to tenant B", err, "42501")

	// Registry DELETE: only nava_worker, only tenant A's, and never while
	// side rows remain (D11 step 5).
	_, err = execIn(app, tenantA, "DELETE FROM record_index_fields WHERE id = $1", fieldA)
	wantSQLState(t, "nava_app deletes a registry row", err, "42501")
	if n, err := execIn(worker, tenantA, "DELETE FROM record_index_fields WHERE tenant_id = $1", tenantB); err != nil || n != 0 {
		t.Errorf("nava_worker in tenant A deletes tenant B's registry rows: %d, %v; want 0", n, err)
	}
	for i, s := range sideTables {
		_, err := execIn(worker, tenantA, "DELETE FROM record_index_fields WHERE tenant_id = $1 AND id = $2", tenantA, sideField(fieldA, i))
		wantSQLState(t, "nava_worker deletes tenant A's "+string(s.vt)+" field with side rows left", err, "23503")
		if n, err := execIn(worker, tenantA, "DELETE FROM "+s.table+" WHERE tenant_id = $1 AND field_id = $2", tenantA, sideField(fieldA, i)); err != nil || n != 1 {
			t.Fatalf("nava_worker clears the %s field's side rows: %d, %v; want 1", s.vt, n, err)
		}
		if n, err := execIn(worker, tenantA, "DELETE FROM record_index_fields WHERE tenant_id = $1 AND id = $2", tenantA, sideField(fieldA, i)); err != nil || n != 1 {
			t.Errorf("control: nava_worker deletes the emptied %s field: %d, %v; want 1", s.vt, n, err)
		}
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM record_index_fields WHERE tenant_id = $1", tenantB); got != "4" {
		t.Errorf("tenant B's registry rows after tenant A's writes = %s, want 4", got)
	}

	// A record is not deleted under its side rows (NO ACTION; P7-02's
	// BEFORE DELETE purge is what clears them).
	_, err = super.Exec(t.Context(), "DELETE FROM records WHERE tenant_id = $1 AND id = $1", tenantB)
	wantSQLState(t, "a record with side rows is deleted", err, "23503")

	for _, role := range []string{"nava_onboard", "nava_groupsync", "nava_relay", "nava_purge", "nava_audit_reader", "nava_erasure"} {
		for _, tbl := range fieldIndexTables {
			_, err := execIn(asRole(t, role), tenantA, "SELECT count(*) FROM "+tbl)
			wantSQLState(t, role+" reads "+tbl, err, "42501")
		}
	}
}

// Acceptance (CI-5b): the catalog holds exactly the D14 grants on the five
// tables, for every role: no UPDATE on a side table, the registry's UPDATE by
// column, and nothing for any other role.
func TestP701Grants(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)

	got := strings.Fields(queryString(t, super, `
		WITH t AS (SELECT oid, relname, relacl FROM pg_class WHERE oid = ANY ('{record_index_fields,rfv_text,rfv_num,rfv_ts,rfv_bool}'::regclass[])),
		g AS (
			SELECT t.relname::text AS tbl, r.rolname::text AS role, a.privilege_type AS priv, '' AS col
			FROM t, aclexplode(t.relacl) a JOIN pg_roles r ON r.oid = a.grantee
			WHERE r.rolname <> 'nava_migrator'
			UNION ALL
			SELECT t.relname::text, r.rolname::text, a.privilege_type, att.attname::text
			FROM t JOIN pg_attribute att ON att.attrelid = t.oid AND att.attacl IS NOT NULL, aclexplode(att.attacl) a
			JOIN pg_roles r ON r.oid = a.grantee
			WHERE r.rolname <> 'nava_migrator'
			UNION ALL
			SELECT t.relname::text, 'PUBLIC', a.privilege_type, ''
			FROM t, aclexplode(t.relacl) a WHERE a.grantee = 0
		)
		SELECT coalesce(string_agg(tbl || ':' || role || ':' || priv || CASE WHEN col <> '' THEN '(' || col || ')' ELSE '' END,
			' ' ORDER BY tbl, role, priv, col), '') FROM g`))

	var want []string
	for _, tbl := range []string{"rfv_bool", "rfv_num", "rfv_text", "rfv_ts"} {
		for _, role := range []string{"nava_app", "nava_worker"} {
			for _, priv := range []string{"DELETE", "INSERT", "SELECT"} {
				want = append(want, tbl+":"+role+":"+priv)
			}
		}
	}
	want = append(want,
		"record_index_fields:nava_app:INSERT",
		"record_index_fields:nava_app:SELECT",
		"record_index_fields:nava_app:UPDATE(progress)",
		"record_index_fields:nava_app:UPDATE(requested_version)",
		"record_index_fields:nava_app:UPDATE(state)",
		"record_index_fields:nava_app:UPDATE(updated_at)",
		"record_index_fields:nava_worker:DELETE",
		"record_index_fields:nava_worker:SELECT",
		"record_index_fields:nava_worker:UPDATE(progress)",
		"record_index_fields:nava_worker:UPDATE(state)",
		"record_index_fields:nava_worker:UPDATE(updated_at)",
	)
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("grants on the field-index tables:\n got  %v\n want %v", got, want)
	}
}

// Acceptance: the migration goes down and up again on real PostgreSQL, and
// down leaves none of its objects behind.
func TestP701MigrationDownUp(t *testing.T) {
	m := freshCluster(t)
	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedEntityTypes(t, super, tenantA)

	sqlDB := openSQL(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, db.Migrations(),
		goose.WithAllowOutofOrder(true), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := p.DownTo(ctx, beforeFieldIndex); err != nil {
		t.Fatalf("down: %v", err)
	}
	if got := queryString(t, super, `SELECT coalesce(string_agg(relname, ','), '') FROM pg_class
		WHERE relname LIKE 'rfv\_%' OR relname LIKE 'record\_index\_fields%'`); got != "" {
		t.Errorf("after down: relations left: %s", got)
	}
	if got := queryString(t, super, `SELECT coalesce(string_agg(stxname, ','), '') FROM pg_statistic_ext WHERE stxname LIKE 'rfv\_%'`); got != "" {
		t.Errorf("after down: statistics left: %s", got)
	}
	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("up again: %v", err)
	}
	if _, err := execIn(asRole(t, "nava_app"), tenantA, insertField, tenantA, fieldA, tenantA, "text"); err != nil {
		t.Errorf("after up again: nava_app registers a field: %v", err)
	}
}
