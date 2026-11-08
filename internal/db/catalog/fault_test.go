package catalog

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// The gate's error paths, without a database: a scripted Querier answers
// each query by its SQL and fails the one a test names. The checks against
// a real catalogue are in internal/db/dbtest (catalog_test.go).

var errInjected = errors.New("injected failure")

// fault is how the scripted Querier fails the query whose SQL contains match.
type fault struct {
	match string
	kind  string // "query": Query fails; "scan": the first row's Scan fails; "rows": rows.Err fails; "norow": no row, no error
}

// fakeQ answers "SELECT current_database()" with db, the datdba lookup with
// owner, a query matching rows' key with those rows, every other query with
// no rows, and fails the query that f matches.
type fakeQ struct {
	db, owner string
	rows      map[string][][]any
	f         fault
	queries   []string
}

func (q *fakeQ) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	q.queries = append(q.queries, sql)
	if q.f.match != "" && strings.Contains(sql, q.f.match) {
		switch q.f.kind {
		case "query":
			return nil, errInjected
		case "scan":
			return &fakeRows{rows: [][]any{{"x"}}, scanErr: errInjected}, nil
		case "rows":
			return &fakeRows{err: errInjected}, nil
		case "norow":
			return &fakeRows{}, nil
		}
	}
	switch {
	case strings.TrimSpace(sql) == "SELECT current_database()::text":
		return &fakeRows{rows: [][]any{{q.db}}}, nil
	case strings.Contains(sql, "SELECT pg_catalog.pg_get_userbyid(datdba)::text FROM pg_catalog.pg_database"):
		return &fakeRows{rows: [][]any{{q.owner}}}, nil
	}
	for k, rows := range q.rows {
		if strings.Contains(sql, k) {
			return &fakeRows{rows: rows}, nil
		}
	}
	return &fakeRows{}, nil
}

// fakeRows is a pgx.Rows over fixed values; Scan assigns them by position.
type fakeRows struct {
	rows    [][]any
	i       int
	scanErr error
	err     error
}

func (r *fakeRows) Close()                                       {}
func (r *fakeRows) Err() error                                   { return r.err }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) RawValues() [][]byte                          { return nil }
func (r *fakeRows) Conn() *pgx.Conn                              { return nil }
func (r *fakeRows) TypeMap() *pgtype.Map                         { return pgtype.NewMap() }
func (r *fakeRows) Values() ([]any, error)                       { return r.rows[r.i-1], nil }

func (r *fakeRows) Next() bool {
	if r.i >= len(r.rows) {
		return false
	}
	r.i++
	return true
}

func (r *fakeRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	row := r.rows[r.i-1]
	if len(dest) != len(row) {
		return errors.New("fakeRows: column count mismatch")
	}
	for i, d := range dest {
		reflect.ValueOf(d).Elem().Set(reflect.ValueOf(row[i]))
	}
	return nil
}

// testManifest is a minimal manifest: owner nava_migrator, one runtime role.
func testManifest() *grants.Manifest {
	return &grants.Manifest{
		PlatformDatabase: "nava",
		Databases:        map[string]grants.Database{"nava": {Owner: "nava_migrator"}},
		Roles: map[string]grants.Role{
			"nava_migrator": {},
			"nava_app":      {Database: []string{"CONNECT"}, Tables: map[string]map[string][]string{"tickets": {"SELECT": nil}}},
		},
	}
}

func newFakeQ(f fault) *fakeQ { return &fakeQ{db: "nava", owner: "nava_migrator", f: f} }

func TestCheckFailsWhenTheGateCannotRun(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		f    fault
		want string // prefix the error must carry, besides wrapping errInjected
	}{
		{"current database query", fault{"SELECT current_database()::text", "query"}, ""},
		{"current database scan", fault{"SELECT current_database()::text", "scan"}, ""},
		{"ci1 relations query", fault{"security_invoker", "query"}, "catalogue gate: "},
		{"ci1 relations scan", fault{"security_invoker", "scan"}, "catalogue gate: "},
		{"ci1 relations rows", fault{"security_invoker", "rows"}, "catalogue gate: "},
		{"ci1 policies query", fault{"pg_catalog.pg_policies", "query"}, "catalogue gate: "},
		{"ci1 policies scan", fault{"pg_catalog.pg_policies", "scan"}, "catalogue gate: "},
		{"ci1 policies rows", fault{"pg_catalog.pg_policies", "rows"}, "catalogue gate: "},
		{"ci1 security definer query", fault{"is SECURITY DEFINER", "query"}, "catalogue gate: "},
		{"ci3 owner lookup", fault{"SELECT pg_catalog.pg_get_userbyid(datdba)::text FROM pg_catalog.pg_database", "query"}, "catalogue gate: "},
		{"ci4 role lookup", fault{"SELECT rolname::text FROM pg_catalog.pg_roles WHERE rolname = ANY", "query"}, "catalogue gate: "},
		{"ci4 missing objects", fault{"which does not exist'", "query"}, "catalogue gate: "},
		{"ci4 privileges query", fault{"has_database_privilege", "query"}, "catalogue gate: "},
		{"ci4 privileges scan", fault{"has_database_privilege", "scan"}, "catalogue gate: "},
		{"ci4 privileges rows", fault{"has_database_privilege", "rows"}, "catalogue gate: "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			problems, err := Check(context.Background(), newFakeQ(c.f), testManifest(), nil)
			if !errors.Is(err, errInjected) {
				t.Fatalf("err = %v, want the injected failure", err)
			}
			if !strings.HasPrefix(err.Error(), c.want) {
				t.Errorf("err = %q, want prefix %q", err, c.want)
			}
			// An error means the gate did not run: no partial problem list
			// that a caller could mistake for a verdict.
			if problems != nil {
				t.Errorf("problems = %v, want nil with an error", problems)
			}
		})
	}
}

func TestCheckRefusesAnotherDatabase(t *testing.T) {
	t.Parallel()
	q := newFakeQ(fault{})
	q.db = "keycloak"
	_, err := Check(context.Background(), q, testManifest(), nil)
	if err == nil || err.Error() != `catalogue gate: connected to "keycloak", the platform database is "nava"` {
		t.Fatalf("err = %v", err)
	}
	if len(q.queries) != 1 {
		t.Fatalf("ran %d queries on the wrong database, want only the database lookup", len(q.queries))
	}
}

func TestCheckReportsWrongDatabaseOwner(t *testing.T) {
	t.Parallel()
	q := newFakeQ(fault{})
	q.owner = "pgadmin"
	problems, err := Check(context.Background(), q, testManifest(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(problems, "CI-3: datdba of nava is pgadmin, want nava_migrator") {
		t.Fatalf("problems = %v, want the datdba problem", problems)
	}
}

func TestScanOneWithoutARow(t *testing.T) {
	t.Parallel()
	var s string
	err := scanOne(context.Background(), newFakeQ(fault{"SELECT 1", "norow"}), "SELECT 1", nil, &s)
	if err == nil || err.Error() != "no row: SELECT 1" {
		t.Fatalf("err = %v, want no row", err)
	}
	q := &fakeQ{}
	q.f = fault{"SELECT 1", "rows"}
	if err := scanOne(context.Background(), q, "SELECT 1", nil, &s); !errors.Is(err, errInjected) {
		t.Fatalf("err = %v, want the rows error, not \"no row\"", err)
	}
}

func TestCheckRejectsMalformedExceptionFunction(t *testing.T) {
	t.Parallel()
	m := testManifest()
	m.InfrastructureExceptions = map[string]grants.Exception{
		"broken": {Schema: "pgbouncer", Function: "not a signature", SecurityDefiner: true},
	}
	_, err := Check(context.Background(), newFakeQ(fault{}), m, nil)
	if err == nil || !strings.Contains(err.Error(), `function "not a signature" is not schema.name(argtypes)`) {
		t.Fatalf("err = %v, want the malformed function refused", err)
	}
}

func TestCI4RejectsMalformedManifestObjects(t *testing.T) {
	t.Parallel()
	for name, r := range map[string]grants.Role{
		"table":    {Tables: map[string]map[string][]string{"Not.A.Table": {"SELECT": nil}}},
		"function": {Functions: map[string][]string{"no_parens": {"EXECUTE"}}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m := testManifest()
			m.Roles["nava_app"] = r
			_, err := ci4Privileges(context.Background(), newFakeQ(fault{}), m, nil)
			if err == nil || !strings.Contains(err.Error(), "is not") {
				t.Fatalf("err = %v, want the malformed %s refused", err, name)
			}
		})
	}
}

func TestCI4ReportsMissingRole(t *testing.T) {
	t.Parallel()
	// The fake knows no role, so every runtime role is missing.
	problems, err := ci4Privileges(context.Background(), newFakeQ(fault{}), testManifest(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(problems, "CI-4: role nava_app of db/grants.yaml does not exist") {
		t.Fatalf("problems = %v", problems)
	}
}

func TestTenantTablesErrors(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"query", "scan"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			got, err := TenantTables(context.Background(), newFakeQ(fault{"Key, set below", kind}), nil)
			if !errors.Is(err, errInjected) || got != nil {
				t.Fatalf("TenantTables = %v, %v; want nil and the injected failure", got, err)
			}
		})
	}
}

func TestTenantTablesKeysAndSkipsGlobals(t *testing.T) {
	t.Parallel()
	q := newFakeQ(fault{})
	q.rows = map[string][][]any{"Key, set below": {
		{"public", "tenants", ""}, {"public", "tickets", ""}, {"public", "global_thing", ""},
	}}
	g := &GlobalTables{Tables: map[string]GlobalTable{"global_thing": {}}}
	got, err := TenantTables(context.Background(), q, g)
	if err != nil {
		t.Fatal(err)
	}
	want := []TenantTable{{"public", "tenants", "id"}, {"public", "tickets", "tenant_id"}}
	if !slices.Equal(got, want) {
		t.Fatalf("TenantTables = %v, want %v", got, want)
	}
}

func TestTenantTableQualifiedNameQuotes(t *testing.T) {
	t.Parallel()
	if got := (TenantTable{Schema: "public", Name: `we"ird`}).QualifiedName(); got != `"public"."we""ird"` {
		t.Fatalf("QualifiedName = %s", got)
	}
}

func TestNilGlobalTablesHaveNoNames(t *testing.T) {
	t.Parallel()
	var g *GlobalTables
	if got := g.Names(); got == nil || len(got) != 0 {
		t.Fatalf("Names() = %#v, want an empty, non-nil slice", got)
	}
}

// policyRow is a pg_policies row as PolicyProblems scans it.
func policyRow(key string, p Policy, roles ...string) []any {
	parts := strings.SplitN(key, ".", 3)
	return []any{parts[0], parts[1], parts[2], p.Cmd, roles, p.Qual, p.Check}
}

func TestPolicyProblemsComparesPinnedForms(t *testing.T) {
	t.Parallel()
	allowKey := "public.outbox.outbox_relay_select"
	pinnedKey := "public.tenants.tenants_onboard_insert"
	allowed, pinned := AllowList()[allowKey], PinnedTenantPolicies()[pinnedKey]
	widened := allowed
	widened.Qual = "(1 = 1)"
	drifted := pinned
	drifted.Check = "true"

	q := newFakeQ(fault{})
	q.rows = map[string][][]any{"pg_catalog.pg_policies": {
		policyRow(allowKey, widened, "nava_relay"),
		policyRow(pinnedKey, drifted, "nava_onboard"),
		policyRow("public.tickets.tickets_public", Policy{Cmd: "SELECT", Qual: TenantPred("tenant_id")}, "public"),
	}}
	problems, untested, err := PolicyProblems(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`outbox_relay_select (allow-listed) = {"SELECT" "nava_relay" "(1 = 1)" ""}, want {"SELECT" "nava_relay" "true" ""}`,
		"tenants_onboard_insert = ",
		"tickets_public on tickets is TO PUBLIC",
		"public.outbox.outbox_purge is missing",
	} {
		if !slices.ContainsFunc(problems, func(p string) bool { return strings.HasPrefix(p, want) }) {
			t.Errorf("no problem starting %q in\n%s", want, strings.Join(problems, "\n"))
		}
	}
	if slices.ContainsFunc(problems, func(p string) bool { return strings.HasPrefix(p, "public.tenants.tenants_onboard_insert is missing") }) {
		t.Error("a drifted pinned policy is reported missing; it exists")
	}
	// The allow-listed policy does not test the tenant; the TO PUBLIC one does.
	if !slices.Equal(untested, []string{allowKey}) {
		t.Errorf("untested = %v, want [%s]", untested, allowKey)
	}
}
