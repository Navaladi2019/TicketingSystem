//go:build integration

// TICK-1036 (ADR-010-A2 A2-5, A2-5a, A2-6, A2-9) against a real PostgreSQL as
// nava_app, under RLS: the list picks C3 or W0 for a region term's page from
// the capped probe P3, skips the probe for a rare type, keeps the count on
// TICK-1034's CC, and returns exactly the rows Allows admits in every shape.
// The plans of W0, C3 and P3 are checked on the pinned PostgreSQL (A2-R4).
package pipeline_test

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
)

// shapeWorld is one tenant with an org tree and records spread over it:
//
//	R ── H      hot: hot records (the switch's district)
//	├─── Th     thin: 3 records
//	└─── N ── N1  2 records each
//	O           outside R: 5 records
//
// plus 5 records with no unit. u1 is a member of Th, N and N1 (nested caller
// units). Every tenth record names u1 as requester; state alternates.
type shapeWorld struct {
	tenant, def uuid.UUID
	u1          identity.Principal
	unit        map[string]uuid.UUID
	seq         int
}

func seedShapeWorld(t *testing.T, e env, hot int) *shapeWorld {
	t.Helper()
	f := seedTenantWith(t, e, aclDefinition())
	w := &shapeWorld{tenant: f.id, u1: f.principal, unit: map[string]uuid.UUID{}}
	if err := e.super.QueryRow(context.Background(), `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2`, w.tenant, aclKey).Scan(&w.def); err != nil {
		t.Fatal(err)
	}
	w.units(t, e, w.tenant, nil)
	for _, u := range []string{"Th", "N", "N1"} {
		mustExec(t, e.super, `INSERT INTO org_unit_members (tenant_id, org_unit_id, principal_id) VALUES ($1, $2, $3)`, w.tenant, w.unit[u], w.u1.PrincipalID)
	}
	rebuildClosure(t, e)
	for _, r := range []struct {
		unit string
		n    int
	}{{"H", hot}, {"Th", 3}, {"N", 2}, {"N1", 2}, {"O", 5}, {"", 5}} {
		w.add(t, e, w.tenant, r.unit, r.n)
	}
	mustExec(t, e.super, `ANALYZE records`)
	mustExec(t, e.super, `ANALYZE org_unit_closure`)
	return w
}

// units inserts the tree into tenant, with the ids of ids when set (another
// tenant reusing this one's unit ids), and records them in w.unit when nil.
func (w *shapeWorld) units(t *testing.T, e env, tenant uuid.UUID, ids map[string]uuid.UUID) {
	t.Helper()
	for _, u := range []struct{ name, parent string }{{"R", ""}, {"H", "R"}, {"Th", "R"}, {"N", "R"}, {"N1", "N"}, {"O", ""}} {
		id := uuid.New()
		if ids != nil {
			id = ids[u.name]
		} else {
			w.unit[u.name] = id
		}
		var parent any
		if u.parent != "" {
			parent = w.unit[u.parent]
			if ids != nil {
				parent = ids[u.parent]
			}
		}
		mustExec(t, e.super, `INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name) VALUES ($1, $2, $3, $4, $5)`,
			tenant, id, parent, "kc-ou-"+tenant.String()+"-"+u.name, u.name)
	}
}

// add inserts n records of tenant under unit ("" for none), past the audit
// commit check (replica role): these tests read rows and plans, not audit.
// Timestamps spread so the updated_at and created_at orders differ.
func (w *shapeWorld) add(t *testing.T, e env, tenant uuid.UUID, unit string, n int) {
	t.Helper()
	ctx := context.Background()
	var ou any
	if unit != "" {
		ou = w.unit[unit]
	}
	def := w.def
	if tenant != w.tenant {
		if err := e.super.QueryRow(ctx, `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2`, tenant, aclKey).Scan(&def); err != nil {
			t.Fatal(err)
		}
	}
	var requester any
	if tenant == w.tenant {
		requester = w.u1.PrincipalID
	}
	w.seq++
	tx, err := e.super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SET LOCAL session_replication_role = replica"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO records (tenant_id, id, entity_definition_id, version, schema_version, number, state, org_unit_id,
			requester_id, created_at, updated_at, data)
		SELECT $1, gen_random_uuid(), $2, 1, 1, 'SW-' || $3::text || '-' || g, CASE WHEN g % 2 = 0 THEN 'open' ELSE 'closed' END, $4::uuid,
			CASE WHEN g % 10 = 0 THEN $5::uuid END,
			now() - ((g * 7919 + $7::int) % 100003) * interval '1 second', now() - (g * 13 + $7::int) * interval '1 millisecond', '{}'
		FROM generate_series(1, $6::int) g`, tenant, def, fmt.Sprint(w.seq), ou, requester, n, w.seq); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func (w *shapeWorld) of(names ...string) string {
	ids := make([]string, len(names))
	for i, n := range names {
		ids[i] = `"` + w.unit[n].String() + `"`
	}
	return `{"under":{"of":[` + strings.Join(ids, ",") + `]}}`
}

// setEstimate sets N̂_e for the world's type (n < 0 deletes the row), as the
// refresh job's upsert would.
func (w *shapeWorld) setEstimate(t *testing.T, e env, n int64) {
	t.Helper()
	mustExec(t, e.super, `DELETE FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definition_id = $2`, w.tenant, w.def)
	if n >= 0 {
		mustExec(t, e.super, `INSERT INTO entity_record_estimates (tenant_id, entity_definition_id, live_records, refreshed_at) VALUES ($1, $2, $3, now())`,
			w.tenant, w.def, n)
	}
}

// probeTracer counts the P3 statements a pool runs.
type probeTracer struct {
	mu sync.Mutex
	n  int
}

func (p *probeTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	if strings.Contains(d.SQL, ") acl_x LIMIT ") && strings.HasSuffix(d.SQL, ") acl_p") {
		p.mu.Lock()
		p.n++
		p.mu.Unlock()
	}
	return ctx
}

func (p *probeTracer) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func (p *probeTracer) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.n
}

// tracedEnv is e with its nava_app pool replaced by one whose connections
// count P3 statements.
func tracedEnv(t *testing.T, e env) (env, *probeTracer) {
	t.Helper()
	tr := &probeTracer{}
	cfg := serverPoolConfig(t, appDSN(t, e.platform))
	cfg.ConnConfig.Tracer = tr
	e.app = newPool(t, cfg)
	return e, tr
}

// listing is what one list read returned: the ids in page order, the
// capped total, and the access SQL it ran.
type listing struct {
	ids    []uuid.UUID
	total  int64
	capped bool
	read   authz.ListRead
}

// how picks the shapes of a list read: the probe (ChooseListShapes), or one
// shape forced on every region term.
type how struct {
	name  string
	probe bool
	shape authz.ListShape
}

func hows() []how {
	return []how{{"probe", true, 0}, {"C3 forced", false, authz.ShapeC3}, {"W0 forced", false, authz.ShapeWalk}}
}

// list runs the list's page (LIMIT limit) and count for pol as the list
// route builds them, with sort -updated_at or -created_at, and mutate applied
// to the page and count SQL when set.
func (w *shapeWorld) list(ctx context.Context, tx pgx.Tx, pol *authz.Policy, p identity.Principal, h how, sort string, limit int) (listing, error) {
	bound, err := authz.ResolveUnder(ctx, tx, pol, p)
	if err != nil {
		return listing{}, err
	}
	shapes := authz.FixedListShapes(h.shape)
	if h.probe {
		if shapes, err = authz.ChooseListShapes(ctx, tx, bound, p, w.def, limit); err != nil {
			return listing{}, err
		}
	}
	read := authz.ReadListShaped(bound, p, "r", 2, shapes)
	q := listQuery(w.tenant, w.def, read, sort, limit)
	page, err := pipeline.List(ctx, tx, q)
	if err != nil {
		return listing{}, err
	}
	out := listing{total: page.Total, capped: page.TotalCapped, read: read}
	for _, r := range page.Records {
		out.ids = append(out.ids, r.ID)
	}
	return out, nil
}

// listQuery is the store query the list route builds from read, with no
// request filter.
func listQuery(tenant, def uuid.UUID, read authz.ListRead, sort string, limit int) pipeline.ListQuery {
	q := pipeline.ListQuery{TenantID: tenant, DefinitionID: def, Where: "true", Args: read.Args, Branches: read.Branches,
		CountBranches: read.CountBranches, Materialize: read.Materialize, SortColumn: sort, Desc: true, Limit: limit}
	if read.Branches == nil {
		q.Where, q.CountWhere = read.Where, read.CountWhere
	}
	for _, d := range read.Drivers {
		q.CountDrivers = append(q.CountDrivers, pipeline.CountDriver{From: d.From, Key: d.Key})
	}
	return q
}

// allowed is every record of the world's type Allows admits for pol and p,
// by id, the binding read with every record's row.
func (w *shapeWorld) allowed(ctx context.Context, tx pgx.Tx, pol *authz.Policy, p identity.Principal) ([]uuid.UUID, error) {
	rows, err := tx.Query(ctx, `SELECT id, coalesce(state, ''), requester_id, assignee_id, assignment_group_id, org_unit_id FROM records
		WHERE tenant_id = $1 AND entity_definition_id = $2`, w.tenant, w.def)
	if err != nil {
		return nil, err
	}
	recs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (authz.Row, error) {
		var x authz.Row
		err := r.Scan(&x.ID, &x.State, &x.RequesterID, &x.AssigneeID, &x.AssignmentGroupID, &x.OrgUnitID)
		return x, err
	})
	if err != nil {
		return nil, err
	}
	bound, err := authz.ResolveUnder(ctx, tx, pol, p, recs...)
	if err != nil {
		return nil, err
	}
	var out []uuid.UUID
	for _, r := range recs {
		if _, _, ok := authz.Allows(bound, p, authz.OpRead, r); ok {
			out = append(out, r.ID)
		}
	}
	return sortedUUIDs(out), nil
}

func sortedUUIDs(ids []uuid.UUID) []uuid.UUID {
	out := slices.Clone(ids)
	slices.SortFunc(out, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	return out
}

// agrees checks one list read against Allows: every row, and the capped
// total, and returns the first page's ids (limit 51) for comparing shapes.
func (w *shapeWorld) agrees(ctx context.Context, t *testing.T, tx pgx.Tx, pol *authz.Policy, p identity.Principal, h how, sort string) (listing, []uuid.UUID) {
	t.Helper()
	want, err := w.allowed(ctx, tx, pol, p)
	if err != nil {
		t.Fatal(err)
	}
	all, err := w.list(ctx, tx, pol, p, h, sort, 100000)
	if err != nil {
		t.Fatalf("%s: list: %v", h.name, err)
	}
	if got := sortedUUIDs(all.ids); !slices.Equal(got, want) {
		t.Errorf("%s, %s: the list returns %d rows, Allows admits %d", h.name, sort, len(got), len(want))
	}
	wantTotal := min(int64(len(want)), store.TotalCap)
	if all.total != wantTotal || all.capped != (len(want) > store.TotalCap) {
		t.Errorf("%s, %s: total %d capped %v, want %d capped %v", h.name, sort, all.total, all.capped, wantTotal, len(want) > store.TotalCap)
	}
	page, err := w.list(ctx, tx, pol, p, h, sort, 51)
	if err != nil {
		t.Fatalf("%s: page: %v", h.name, err)
	}
	if !slices.Equal(page.ids, all.ids[:min(51, len(all.ids))]) {
		t.Errorf("%s, %s: the 51-row page is not the head of the full list", h.name, sort)
	}
	return page, page.ids
}

// materialized reports whether any region term of read is C3.
func materialized(read authz.ListRead) bool { return slices.Contains(read.Materialize, true) }

// Acceptance tests 1 (a missing row: K* = 1,000 and P3 runs), 2 (the switch
// on both sides of K*, the hot and thin districts, a missing or wrong
// estimate) and 5's shape part: every case agrees with Allows in every shape.
func TestUnderListSwitchesAtKStar(t *testing.T) {
	base := migrated(t)
	e, tr := tracedEnv(t, base)
	w := seedShapeWorld(t, e, authz.KStarMin) // H holds exactly K* = 1,000
	ctx := context.Background()
	self := `{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}`
	open := `{"cmp":{"column":"state","op":"eq","values":["open"]}}`
	cases := []struct {
		name, when string
		c3         bool // the probe's choice, with no estimate row (K* = 1,000)
	}{
		{"hot district at m = K*", w.of("H"), true},
		{"thin district", w.of("Th"), true},
		{"nested caller units", `{"under":{"of":"caller"}}`, true},
		{"region or requester", `{"any":[` + w.of("H") + `,` + self + `]}`, true},
		{"requester or open in the thin district", `{"any":[` + self + `,{"all":[` + open + `,` + w.of("Th") + `]}]}`, true},
		{"the whole tree: m > K*", w.of("R"), false},
		{"not under the hot district: no region term", `{"not":` + w.of("H") + `}`, false},
	}
	run := func(t *testing.T, label string, want map[string]bool) {
		for _, tc := range cases {
			pol := underDoc(t, tc.when)
			inTx(t, e, w.tenant, func(tx pgx.Tx) error {
				var first []uuid.UUID
				for _, h := range hows() {
					for _, sort := range []string{"updated_at", "created_at"} {
						page, ids := w.agrees(ctx, t, tx, pol, w.u1, h, sort)
						if h.probe && want != nil && materialized(page.read) != want[tc.name] {
							t.Errorf("%s, %s: the probe chose C3 = %v, want %v", label, tc.name, materialized(page.read), want[tc.name])
						}
						if sort == "updated_at" {
							if first == nil {
								first = ids
							} else if !slices.Equal(ids, first) {
								t.Errorf("%s, %s: %s's first page differs from the probe's", label, tc.name, h.name)
							}
						}
					}
				}
				return nil
			})
		}
	}

	// A missing row: K* = 1,000, not rare, so P3 runs.
	w.setEstimate(t, e, -1)
	want := map[string]bool{}
	for _, tc := range cases {
		want[tc.name] = tc.c3
	}
	before := tr.count()
	run(t, "missing row, m = K*", want)
	if tr.count() == before {
		t.Error("a missing estimate row ran no P3")
	}

	// m = K* + 1: the hot district walks.
	w.add(t, e, w.tenant, "H", 1)
	want["hot district at m = K*"], want["region or requester"] = false, false
	run(t, "missing row, m = K* + 1", want)

	// The estimate right, 100x too high and 100x too low: the rows never
	// change (A2-6). 100x too low makes the type rare (A2-5a).
	var live int64
	if err := e.super.QueryRow(ctx, `SELECT count(*) FROM records WHERE tenant_id = $1 AND entity_definition_id = $2`, w.tenant, w.def).Scan(&live); err != nil {
		t.Fatal(err)
	}
	for _, n := range []int64{live, live * 100, live / 100} {
		w.setEstimate(t, e, n)
		run(t, fmt.Sprintf("estimate %d of %d", n, live), nil)
	}
}

// Acceptance test 3 (A2-5a): N̂_e ≤ K* issues no P3, renders W0 for the page
// and the count with no driver, and returns the rows and total of the probed
// path with the estimate removed.
func TestUnderListRareTypeSkipsTheProbe(t *testing.T) {
	base := migrated(t)
	e, tr := tracedEnv(t, base)
	w := seedShapeWorld(t, e, 300)
	ctx := context.Background()
	for _, when := range []string{w.of("H"), w.of("R"), `{"under":{"of":"caller"}}`,
		`{"any":[` + w.of("H") + `,{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}]}`} {
		pol := underDoc(t, when)
		w.setEstimate(t, e, -1)
		var probed listing
		inTx(t, e, w.tenant, func(tx pgx.Tx) error {
			var err error
			probed, err = w.list(ctx, tx, pol, w.u1, hows()[0], "updated_at", 100000)
			return err
		})
		w.setEstimate(t, e, 500) // ≤ K* = 1,000
		before := tr.count()
		inTx(t, e, w.tenant, func(tx pgx.Tx) error {
			rare, _ := w.agrees(ctx, t, tx, pol, w.u1, hows()[0], "updated_at")
			if materialized(rare.read) || slices.ContainsFunc(rare.read.Drivers, func(d authz.RegionDriver) bool { return d.From != "" }) {
				t.Errorf("%s: rare type materialized %v, drivers %+v; want W0 and no driver", when, rare.read.Materialize, rare.read.Drivers)
			}
			cc := pipeline.CountStatement(listQuery(w.tenant, w.def, rare.read, "updated_at", 51))
			if !strings.Contains(cc, "OFFSET 0") || strings.Contains(cc, "acl_d.descendant_id") {
				t.Errorf("%s: the rare count is not W0: %s", when, cc)
			}
			all, err := w.list(ctx, tx, pol, w.u1, hows()[0], "updated_at", 100000)
			if err != nil {
				return err
			}
			if !slices.Equal(sortedUUIDs(all.ids), sortedUUIDs(probed.ids)) || all.total != probed.total || all.capped != probed.capped {
				t.Errorf("%s: rare path %d rows total %d, probed path %d rows total %d", when, len(all.ids), all.total, len(probed.ids), probed.total)
			}
			return nil
		})
		if n := tr.count() - before; n != 0 {
			t.Errorf("%s: a rare type ran %d P3 statements, want 0", when, n)
		}
	}
}

// probeCount runs P3 for U = units with K = k.
func probeCount(ctx context.Context, tx pgx.Tx, sql string, tenant, def uuid.UUID, units []string) (int64, error) {
	var m int64
	err := tx.QueryRow(ctx, sql, tenant, units, def).Scan(&m)
	return m, err
}

// Acceptance tests 4 (R2-N1 through P3, C3, W0 and CC) and 6: the mutants
// u_not_canonical (a nested id and a duplicate id) and probe_changes_rows
// are caught.
func TestUnderListDuplicateAndNestedU(t *testing.T) {
	e := migrated(t)
	w := seedShapeWorld(t, e, 40)
	ctx := context.Background()
	th, n, n1 := w.unit["Th"].String(), w.unit["N"].String(), w.unit["N1"].String()
	inTx(t, e, w.tenant, func(tx pgx.Tx) error {
		p3 := authz.ProbeStatement(authz.KStarMin)
		for _, tc := range []struct {
			name  string
			units []string
			want  int64
		}{{"{Th}", []string{th}, 3}, {"{Th, Th}", []string{th, th}, 3}, {"{N, N1}", []string{n, n1}, 4}, {"{N1, N, N1}", []string{n1, n, n1}, 4}} {
			m, err := probeCount(ctx, tx, p3, w.tenant, w.def, tc.units)
			if err != nil {
				return err
			}
			if m != tc.want {
				t.Errorf("P3 over %s = %d, want %d", tc.name, m, tc.want)
			}
		}
		// u_not_canonical: no DISTINCT, and no nested-id drop.
		noDistinct := strings.Replace(p3, "(SELECT DISTINCT acl_ud.id FROM", "(SELECT acl_ud.id FROM", 1)
		if m, _ := probeCount(ctx, tx, noDistinct, w.tenant, w.def, []string{th, th}); m == 3 {
			t.Error("mutant u_not_canonical (duplicate id, P3) is not caught")
		}
		nested := regexp.MustCompile(` WHERE NOT EXISTS \(SELECT 1 FROM org_unit_closure acl_up .*?acl_up\.descendant_id = acl_u\.id\)\)`)
		noDrop := nested.ReplaceAllString(p3, ")")
		if noDrop == p3 {
			t.Fatal("P3 has no nested-id drop to remove")
		}
		if m, err := probeCount(ctx, tx, noDrop, w.tenant, w.def, []string{n, n1}); err != nil || m == 4 {
			t.Errorf("mutant u_not_canonical (nested id, P3) is not caught: %d, %v", m, err)
		}

		// U = {Th, Th} through C3, W0 and CC: the rows and total of {Th}.
		pol := underDoc(t, w.of("Th"))
		bound, err := authz.ResolveUnder(ctx, tx, pol, w.u1)
		if err != nil {
			return err
		}
		for _, shape := range []authz.ListShape{authz.ShapeC3, authz.ShapeWalk} {
			read := authz.ReadListShaped(bound, w.u1, "r", 2, authz.FixedListShapes(shape))
			one := listQuery(w.tenant, w.def, read, "updated_at", 100)
			two := one
			two.Args = dupU(one.Args, w.unit["Th"])
			a, err := pipeline.List(ctx, tx, one)
			if err != nil {
				return err
			}
			b, err := pipeline.List(ctx, tx, two)
			if err != nil {
				return err
			}
			if len(a.Records) != 3 || len(b.Records) != 3 || a.Total != 3 || b.Total != 3 {
				t.Errorf("shape %d: U = {Th} %d rows total %d, U = {Th, Th} %d rows total %d; want 3",
					shape, len(a.Records), a.Total, len(b.Records), b.Total)
			}
		}

		// probe_changes_rows: a C3 leaf that reads U itself instead of its
		// closure descendants changes the rows of of [N] (N1's go missing).
		polN := underDoc(t, w.of("N"))
		boundN, err := authz.ResolveUnder(ctx, tx, polN, w.u1)
		if err != nil {
			return err
		}
		read := authz.ReadListShaped(boundN, w.u1, "r", 2, authz.FixedListShapes(authz.ShapeC3))
		q := listQuery(w.tenant, w.def, read, "updated_at", 100)
		mut := regexp.MustCompile(`= ANY\(ARRAY\(SELECT acl_ou\.descendant_id FROM org_unit_closure acl_ou WHERE acl_ou\.tenant_id = (\$[0-9]+)::uuid AND acl_ou\.ancestor_id = ANY\((\$[0-9]+)::uuid\[\]\)\)\)`)
		mq := q
		mq.Where = mut.ReplaceAllString(q.Where, "= ANY(${2}::uuid[])")
		if mq.Where == q.Where {
			t.Fatalf("no C3 leaf to mutate: %s", q.Where)
		}
		good, err := pipeline.List(ctx, tx, q)
		if err != nil {
			return err
		}
		bad, err := pipeline.List(ctx, tx, mq)
		if err != nil {
			return err
		}
		want, err := w.allowed(ctx, tx, polN, w.u1)
		if err != nil {
			return err
		}
		if len(good.Records) != len(want) || len(want) != 4 {
			t.Errorf("C3 of [N] returns %d rows, Allows %d; want 4", len(good.Records), len(want))
		}
		if len(bad.Records) == len(want) {
			t.Error("mutant probe_changes_rows is not caught")
		}
		return nil
	})
}

// planNode is one node of a JSON EXPLAIN plan.
type planNode struct {
	Type     string     `json:"Node Type"`
	Relation string     `json:"Relation Name"`
	Index    string     `json:"Index Name"`
	Parent   string     `json:"Parent Relationship"`
	SubName  string     `json:"Subplan Name"`
	CTE      string     `json:"CTE Name"`
	Join     string     `json:"Join Type"`
	Plans    []planNode `json:"Plans"`
}

func explain(ctx context.Context, t *testing.T, tx pgx.Tx, sql string, args []any) planNode {
	t.Helper()
	var raw string
	if err := tx.QueryRow(ctx, "EXPLAIN (FORMAT JSON) "+sql, args...).Scan(&raw); err != nil {
		t.Fatalf("explain %s: %v", sql, err)
	}
	var plans []struct {
		Plan planNode `json:"Plan"`
	}
	if err := json.Unmarshal([]byte(raw), &plans); err != nil || len(plans) != 1 {
		t.Fatalf("plan %s: %v", raw, err)
	}
	return plans[0].Plan
}

// walk calls fn on every node with its ancestors, outermost first.
func (n planNode) walk(path []planNode, fn func(n planNode, path []planNode)) {
	fn(n, path)
	for _, k := range n.Plans {
		k.walk(append(slices.Clone(path), n), fn)
	}
}

// w0Faults is what a W0 page plan gets wrong (A2-5, A2-R4): the records are
// read from the sort index, the closure probe is a SubPlan on
// org_unit_closure_pkey, and nothing is a semi-join.
func w0Faults(plan planNode, sortIndex string) []string {
	var out []string
	var walked, sub bool
	plan.walk(nil, func(n planNode, path []planNode) {
		if n.Relation == "records" && n.Index == sortIndex {
			walked = true
		}
		if n.Index == "org_unit_closure_pkey" && slices.ContainsFunc(append(path, n), func(a planNode) bool { return a.Parent == "SubPlan" }) {
			sub = true
		}
		if strings.Contains(n.Join, "Semi") {
			out = append(out, "a "+n.Join+" join")
		}
	})
	if !walked {
		out = append(out, "records not read from "+sortIndex)
	}
	if !sub {
		out = append(out, "no SubPlan on org_unit_closure_pkey")
	}
	return out
}

// c3Faults: C3 plans a CTE Scan over the materialized acl_m.
func c3Faults(plan planNode) []string {
	found := false
	plan.walk(nil, func(n planNode, _ []planNode) { found = found || (n.Type == "CTE Scan" && n.CTE == "acl_m") })
	if !found {
		return []string{"no CTE Scan of acl_m"}
	}
	return nil
}

// Acceptance tests 7 (the CI plan test on the pinned PostgreSQL, A2-R4) and
// 6: the mutants c3_without_materialized and walk_without_offset are caught.
func TestUnderListShapePlans(t *testing.T) {
	e := migrated(t)
	w := seedShapeWorld(t, e, 200)
	w.add(t, e, w.tenant, "O", 30000) // a tenant large enough to walk an index
	mustExec(t, e.super, `ANALYZE records`)
	ctx := context.Background()
	pol := underDoc(t, w.of("R"))
	inTx(t, e, w.tenant, func(tx pgx.Tx) error {
		bound, err := authz.ResolveUnder(ctx, tx, pol, w.u1)
		if err != nil {
			return err
		}
		for _, sort := range []struct{ col, index string }{{"updated_at", "records_list_updated"}, {"created_at", "records_list_created"}} {
			read := authz.ReadListShaped(bound, w.u1, "r", 2, authz.FixedListShapes(authz.ShapeWalk))
			sql, args, err := pipeline.PageStatement(listQuery(w.tenant, w.def, read, sort.col, 51))
			if err != nil {
				return err
			}
			if f := w0Faults(explain(ctx, t, tx, sql, args), sort.index); len(f) > 0 {
				t.Errorf("W0, sort -%s: %v\n%s", sort.col, f, sql)
			}
			mut := strings.ReplaceAll(sql, " OFFSET 0))", "))")
			if mut == sql || len(w0Faults(explain(ctx, t, tx, mut, args), sort.index)) == 0 {
				t.Errorf("mutant walk_without_offset is not caught, sort -%s", sort.col)
			}

			read = authz.ReadListShaped(bound, w.u1, "r", 2, authz.FixedListShapes(authz.ShapeC3))
			sql, args, err = pipeline.PageStatement(listQuery(w.tenant, w.def, read, sort.col, 51))
			if err != nil {
				return err
			}
			if f := c3Faults(explain(ctx, t, tx, sql, args)); len(f) > 0 {
				t.Errorf("C3, sort -%s: %v\n%s", sort.col, f, sql)
			}
			// Acceptance test 10: at C3 the page holds no OFFSET 0 closure probe.
			if strings.Contains(sql, "OFFSET 0") {
				t.Errorf("C3 page holds an OFFSET 0 walk: %s", sql)
			}
			mut = strings.Replace(sql, "AS MATERIALIZED (", "AS (", 1)
			if mut == sql || len(c3Faults(explain(ctx, t, tx, mut, args))) == 0 {
				t.Errorf("mutant c3_without_materialized is not caught, sort -%s", sort.col)
			}
		}

		// P3: the records are read per unit, under a Limit, inside the
		// nested loop over the closure.
		plan := explain(ctx, t, tx, authz.ProbeStatement(authz.KStarMin), []any{w.tenant, []string{w.unit["R"].String()}, w.def})
		var scans, limited int
		plan.walk(nil, func(n planNode, path []planNode) {
			if n.Relation != "records" {
				return
			}
			scans++
			for i := len(path) - 1; i > 0; i-- {
				if path[i].Type == "Limit" && path[i-1].Type == "Nested Loop" {
					limited++
					break
				}
			}
		})
		if scans == 0 || limited != scans {
			t.Errorf("P3 reads records %d times, %d of them under a per-unit Limit in a Nested Loop", scans, limited)
		}
		return nil
	})
}

// Acceptance test 8 (A2-6 requirement 4): tenant B holds closure rows with
// A's unit ids and more than K* records under them. A's P3 count, C3, W0 and
// CC see none of them, and A's probe count is what it was before B existed.
func TestUnderListShapesStayInTheTenant(t *testing.T) {
	e := migrated(t)
	w := seedShapeWorld(t, e, 600)
	ctx := context.Background()
	pol := underDoc(t, w.of("R"))
	type seen struct {
		m      int64
		rows   map[string][]uuid.UUID
		totals map[string]int64
	}
	look := func() seen {
		s := seen{rows: map[string][]uuid.UUID{}, totals: map[string]int64{}}
		inTx(t, e, w.tenant, func(tx pgx.Tx) error {
			var err error
			if s.m, err = probeCount(ctx, tx, authz.ProbeStatement(authz.KStarMin), w.tenant, w.def, []string{w.unit["R"].String()}); err != nil {
				return err
			}
			for _, h := range hows() {
				l, err := w.list(ctx, tx, pol, w.u1, h, "updated_at", 100000)
				if err != nil {
					return err
				}
				s.rows[h.name], s.totals[h.name] = sortedUUIDs(l.ids), l.total
			}
			return nil
		})
		return s
	}
	before := look()
	if before.m != 607 {
		t.Fatalf("A's P3 under R = %d, want 607", before.m)
	}

	other := seedTenantWith(t, e, aclDefinition())
	w.units(t, e, other.id, w.unit) // B's units carry A's ids
	rebuildClosure(t, e)
	w.add(t, e, other.id, "H", authz.KStarMin+500)
	w.add(t, e, other.id, "N1", authz.KStarMin+500)
	mustExec(t, e.super, `ANALYZE records`)
	if n := count(t, e.super, `SELECT count(*) FROM org_unit_closure WHERE tenant_id = $1 AND ancestor_id = $2`, other.id, w.unit["R"]); n < 5 {
		t.Fatalf("B's closure holds %d rows under A's R id", n)
	}

	after := look()
	if after.m != before.m {
		t.Errorf("A's P3 count moved from %d to %d when B was added", before.m, after.m)
	}
	for _, h := range hows() {
		if !slices.Equal(after.rows[h.name], before.rows[h.name]) || after.totals[h.name] != before.totals[h.name] {
			t.Errorf("%s: A's rows %d -> %d, total %d -> %d when B was added", h.name,
				len(before.rows[h.name]), len(after.rows[h.name]), before.totals[h.name], after.totals[h.name])
		}
	}
}
