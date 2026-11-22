//go:build integration

// ADR-010-E3 (TICK-1084): a caller with exactly one assignment group gets the
// scalar "assignment_group_id = $n::uuid" from the row-ACL compiler, so the
// band (ii) list page (a ready num filter, the onboard default incident ACL)
// walks records_list_group in index order and stops after the page, instead
// of reading the whole group and sorting it (TICK-1049).
//
// The fixture is TICK-1049's shape where it matters: 404,137 records of one
// type, a service-desk group of about 5% hashed evenly over updated_at as the
// S21 kit does, two small groups, and the kit's priority distribution
// (1/1/48/50%). The statements are the real ones: authz.ReadPredicate, the
// list's own EXISTS (ExistsTermForTest) and pipeline.PageStatement, run as
// nava_app on the server's pool (QueryExecModeExec).

package server_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// sgRecords is the tenant's size, TICK-1049's (S21 kit: 400,000 plus the
// kit's own); the groups are hashed over it as the kit hashes its
// service-desk group, so each is spread evenly over updated_at: big 5%,
// small 0.75%, tiny 0.25%, the rest unassigned.
const sgRecords = 404137

// sgEnv is the fixture: tenant A's asset type under the onboard default
// incident ACL, its three groups, the ready priority field and the policy as
// the handler loads it.
type sgEnv struct {
	*etEnv
	big, small, tiny uuid.UUID
	priority, def    uuid.UUID
	size             map[uuid.UUID]int // records per group
	pol              *authz.Policy
}

func newSGEnv(t *testing.T) *sgEnv {
	t.Helper()
	e := &sgEnv{etEnv: newFilterEnv(t), big: uuid.New(), small: uuid.New(), tiny: uuid.New()}
	ctx := t.Context()
	tn := e.a.id
	prefix := "/t/" + tn.String()
	for name, g := range map[string]uuid.UUID{"service-desk": e.big, "desk-2": e.small, "desk-3": e.tiny} {
		if _, err := e.super.Exec(ctx, "INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1, $2, $3, $4)",
			tn, g, "a-"+name, prefix+"/"+name); err != nil {
			t.Fatal(err)
		}
	}
	rules := onboard.DefaultIncidentRules(e.a.adminGroup, []uuid.UUID{e.big, e.small, e.tiny})
	e.publishACL(t, tn, "asset", rules...)
	e.pol = &authz.Policy{Version: 1, EntityType: "asset", Document: authz.ConvertFormat1(rules)}

	// Records written as the bulk seeds are (replica role: no triggers, no
	// audit); the priority side table is filled by flagField below, through
	// record_field_coerce as the backfill does.
	tx, err := e.super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SET LOCAL session_replication_role = replica"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO records (tenant_id, id, entity_definition_id, version, schema_version, number, state,
			assignment_group_id, created_at, updated_at, data)
		SELECT d.tenant_id, gen_random_uuid(), d.id, 1, 1, 'SG-' || g, 'open',
			CASE WHEN x.hg < 500 THEN $3::uuid WHEN x.hg < 575 THEN $4::uuid WHEN x.hg < 600 THEN $5::uuid END,
			now() - make_interval(secs => g * 4), now() - make_interval(secs => g * 4),
			jsonb_build_object('short_description', 'sg', 'priority',
				CASE WHEN x.hp = 0 THEN 1 WHEN x.hp = 1 THEN 2 WHEN x.hp < 50 THEN 3 ELSE 4 END)
		FROM entity_definitions d, generate_series(1, $6::int) g,
			LATERAL (SELECT (hashtext('g' || g) & 2147483647) % 10000 AS hg, (hashtext('p' || g) & 2147483647) % 100 AS hp) x
		WHERE d.tenant_id = $1 AND d.key = $2`, tn, "asset", e.big, e.small, e.tiny, sgRecords); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	e.priority = e.flagField(t, tn, "asset", "priority", "num", "ready", 100)
	if err := e.super.QueryRow(ctx, "SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = 'asset'", tn).Scan(&e.def); err != nil {
		t.Fatal(err)
	}
	e.size = map[uuid.UUID]int{}
	for _, g := range []uuid.UUID{e.big, e.small, e.tiny} {
		var n int
		if err := e.super.QueryRow(ctx, "SELECT count(*) FROM records WHERE tenant_id = $1 AND assignment_group_id = $2", tn, g).Scan(&n); err != nil {
			t.Fatal(err)
		}
		e.size[g] = n
	}
	for _, tbl := range []string{"records", "rfv_num", "groups", "entity_definitions"} {
		if _, err := e.super.Exec(ctx, "VACUUM (ANALYZE) "+tbl); err != nil {
			t.Fatal(err)
		}
	}
	return e
}

// agent is a principal of tenant A in groups, registered as a token.
func (e *sgEnv) agent(t *testing.T, name string, groups ...uuid.UUID) identity.Principal {
	t.Helper()
	p := identity.Principal{TenantID: e.a.id, PrincipalID: uuid.New(), Groups: groups}
	if _, err := e.super.Exec(t.Context(), "INSERT INTO principals (tenant_id, id, keycloak_user_id, email, enabled, federated_links, synced_at) VALUES ($1, $2, $3, $4, true, '[]', now())",
		p.TenantID, p.PrincipalID, "a-"+name, name+"@a.test"); err != nil {
		t.Fatal(err)
	}
	for _, g := range groups {
		if _, err := e.super.Exec(t.Context(), "INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1, $2, $3)", p.TenantID, g, p.PrincipalID); err != nil {
			t.Fatal(err)
		}
	}
	e.tokens[name] = p
	return p
}

var sgPlaceholder = regexp.MustCompile(`\$(\d+)`)

// query is the band (ii) list query the handler builds for p with
// filter[priority]=3, sorted by updated_at desc, a page of 50: the ACL from
// authz.ReadPredicate, the filter from the list's own EXISTS, its
// placeholders numbered after the ACL's as planList's where(base) numbers them.
func (e *sgEnv) query(p identity.Principal) pipeline.ListQuery {
	branches, readSQL, readArgs := authz.ReadPredicate(e.pol, p, "r", 2)
	filt, filtArgs := server.ExistsTermForTest(fieldindex.TypeNum, e.priority, "eq", 3.0)
	base := 2 + len(readArgs)
	filt = sgPlaceholder.ReplaceAllStringFunc(filt, func(m string) string {
		n, _ := strconv.Atoi(m[1:])
		return "$" + strconv.Itoa(base+n)
	})
	q := pipeline.ListQuery{
		TenantID: e.a.id, DefinitionID: e.def, Where: filt, Args: append(slices.Clone(readArgs), filtArgs...),
		Branches: branches, SortColumn: "updated_at", Desc: true, Limit: 51, BudgetCount: true, SkipCount: true,
	}
	if branches == nil {
		q.Where = readSQL + " AND " + filt
	}
	return q
}

// arrayForm is q with main's rendering of a one-group my_groups condition:
// "= ANY($n::uuid[])" bound to the one-element []string, the form E3 replaced.
func arrayForm(t *testing.T, q pipeline.ListQuery, group uuid.UUID) pipeline.ListQuery {
	t.Helper()
	out := q
	out.Args = slices.Clone(q.Args)
	out.Branches = slices.Clone(q.Branches)
	at := slices.IndexFunc(q.Args, func(a any) bool { id, ok := a.(uuid.UUID); return ok && id == group })
	if at < 0 {
		t.Fatalf("no uuid.UUID argument for the group in %v", q.Args)
	}
	n := "$" + strconv.Itoa(3+at)
	scalar := "r.assignment_group_id = " + n + "::uuid)"
	out.Args[at] = authz.UUIDTexts([]uuid.UUID{group})
	rewrote := false
	for i, b := range out.Branches {
		if strings.Contains(b, scalar) {
			out.Branches[i] = strings.ReplaceAll(b, scalar, "r.assignment_group_id = ANY("+n+"::uuid[]))")
			rewrote = true
		}
	}
	if !rewrote {
		t.Fatalf("no scalar %s in %v", scalar, q.Branches)
	}
	return out
}

// explain is EXPLAIN (ANALYZE, FORMAT JSON) of q's page statement as nava_app
// on the server's pool.
func (e *sgEnv) explain(t *testing.T, q pipeline.ListQuery) map[string]any {
	t.Helper()
	sql, args, err := pipeline.PageStatement(q)
	if err != nil {
		t.Fatal(err)
	}
	var raw string
	if err := tenancy.InTenantTx(t.Context(), e.etEnv.pool, e.a.id, func(tx pgx.Tx) error {
		return tx.QueryRow(t.Context(), "EXPLAIN (ANALYZE, FORMAT JSON) "+sql, args...).Scan(&raw)
	}); err != nil {
		t.Fatalf("explain: %v\n%s", err, sql)
	}
	var doc []map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatal(err)
	}
	return doc[0]
}

func sgKids(n map[string]any) []map[string]any {
	var out []map[string]any
	if ks, ok := n["Plans"].([]any); ok {
		for _, k := range ks {
			if m, ok := k.(map[string]any); ok {
				out = append(out, m)
			}
		}
	}
	return out
}

// groupScans is every index or bitmap index scan on records_list_group with
// its ancestors, root first.
func groupScans(root map[string]any) [][]map[string]any {
	var out [][]map[string]any
	var walk func(n map[string]any, path []map[string]any)
	walk = func(n map[string]any, path []map[string]any) {
		path = append(slices.Clone(path), n)
		if (n["Node Type"] == "Index Scan" || n["Node Type"] == "Bitmap Index Scan") && n["Index Name"] == "records_list_group" {
			out = append(out, path)
		}
		for _, k := range sgKids(n) {
			walk(k, path)
		}
	}
	walk(root, nil)
	return out
}

// TICK-1084 on one fixture (seeding it takes most of the run): AC 2, AC 3
// and the RESULT timings.
func TestOneGroupACLOnTheTICK1049Shape(t *testing.T) {
	e := newSGEnv(t)
	t.Logf("TICK-1084 fixture: %d records; groups big %d, small %d, tiny %d", sgRecords, e.size[e.big], e.size[e.small], e.size[e.tiny])
	t.Run("plan", e.testWalksRecordsListGroupInOrder)
	t.Run("equivalence", e.testScalarAndArrayFormsListTheSameRows)
	t.Run("timings", e.testPageTimingsByGroupCount)
}

// TICK-1084 AC 2, as ADR-010-E3 E3-5 states it: for a one-group agent the
// group branch reads records_list_group under its own Limit with no Sort in
// between, and reads under 10% of the group's rows. The array form, the
// control, fails both.
func (e *sgEnv) testWalksRecordsListGroupInOrder(t *testing.T) {
	// check reports whether the plan's records_list_group scan sits under its
	// branch's Limit with no Sort between, and the rows it read; a plan with no
	// such scan is not ordered.
	check := func(q pipeline.ListQuery) (ordered bool, read float64, idx []string) {
		root := e.explain(t, q)["Plan"].(map[string]any)
		scans := groupScans(root)
		if len(scans) != 1 {
			return false, 0, indexNames(root)
		}
		path := scans[0]
		scan := path[len(path)-1]
		read = scan["Actual Rows"].(float64) * scan["Actual Loops"].(float64)
		for i := len(path) - 2; i >= 0; i-- {
			switch path[i]["Node Type"] {
			case "Sort", "Incremental Sort":
				return false, read, indexNames(root)
			case "Limit":
				return true, read, indexNames(root)
			}
		}
		return false, read, indexNames(root)
	}
	sortedControl := 0
	for _, g := range []uuid.UUID{e.big, e.small, e.tiny} {
		agent := e.agent(t, "plan-"+g.String()[:8], g)
		q := e.query(agent)
		if len(q.Branches) < 2 {
			t.Fatalf("branches %v: the onboard ACL reads as one branch per subject", q.Branches)
		}
		ordered, read, idx := check(q)
		// E3-5's bound, under 10% of the group, where it is meaningful: a page
		// of 51 at the 48% priority needs about 107 rows, so a group of 1,060
		// can only be held to the order.
		limit := 0.10 * float64(e.size[g])
		if !ordered || (e.size[g] >= 2000 && read >= limit) {
			t.Errorf("group of %d, scalar: ordered %t, %.0f rows read (want under %.0f), indexes %v", e.size[g], ordered, read, limit, idx)
		}
		// Control: main's rendering. The planner may walk records_list_updated
		// instead, which is fast; when it uses records_list_group it reads the
		// group and sorts it (TICK-1049).
		aOrdered, aRead, aIdx := check(arrayForm(t, q, g))
		t.Logf("group of %d: scalar ordered %t read %.0f; array ordered %t read %.0f %v", e.size[g], ordered, read, aOrdered, aRead, aIdx)
		if aOrdered {
			t.Errorf("group of %d, array form: records_list_group read in order; the scalar fix would not be needed", e.size[g])
		}
		if aRead >= limit {
			sortedControl++
		}
	}
	if sortedControl == 0 {
		t.Errorf("no one-group array form read its group and sorted: the fixture no longer reproduces TICK-1049")
	}
}

type sgPage struct {
	ids  []uuid.UUID
	keys []string
}

// pages lists n pages of q from the start and from a cursor deep in the
// result, carrying the keyset cursor as the handler does for updated_at desc.
func (e *sgEnv) pages(t *testing.T, q pipeline.ListQuery, n int, from *sgCursor) []sgPage {
	t.Helper()
	var out []sgPage
	cur := from
	for range n {
		lq := q
		if cur != nil {
			b := 2 + len(q.Args)
			lq.Keyset = fmt.Sprintf("((r.updated_at, r.id) < ($%d::timestamptz, $%d::uuid))", b+1, b+2)
			lq.KeysetArgs = []any{cur.ts, cur.id.String()}
		}
		var page *pipeline.ListPage
		if err := tenancy.InTenantTx(t.Context(), e.etEnv.pool, e.a.id, func(tx pgx.Tx) error {
			var err error
			page, err = pipeline.List(t.Context(), tx, lq)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		var p sgPage
		for i, r := range page.Records[:min(len(page.Records), 50)] {
			p.ids = append(p.ids, r.ID)
			p.keys = append(p.keys, *page.SortKeys[i])
		}
		out = append(out, p)
		if len(page.Records) <= 50 {
			break
		}
		cur = &sgCursor{ts: p.keys[49], id: p.ids[49]}
	}
	return out
}

type sgCursor struct {
	ts string
	id uuid.UUID
}

// TICK-1084 AC 3 (ADR-010-E3 E3-4 (2)): the scalar and the array form give the
// same rows and the same keyset cursors, page by page, from the start and from
// deep in the result, for a one-group agent of the big group and of a small
// one. The handler's own pages are the scalar form's.
func (e *sgEnv) testScalarAndArrayFormsListTheSameRows(t *testing.T) {
	for _, c := range []struct {
		name  string
		group uuid.UUID
	}{{"agent-big", e.big}, {"agent-tiny", e.tiny}} {
		agent := e.agent(t, c.name, c.group)
		q := e.query(agent)
		arr := arrayForm(t, q, c.group)
		var deep sgCursor
		if err := e.super.QueryRow(t.Context(), `SELECT r.updated_at::text, r.id FROM records r JOIN rfv_num v ON v.tenant_id = r.tenant_id AND v.record_id = r.id
			WHERE r.tenant_id = $1 AND r.assignment_group_id = $2 AND v.field_id = $3 AND v.value = 3
			ORDER BY r.updated_at DESC, r.id DESC OFFSET $4 LIMIT 1`, e.a.id, c.group, e.priority, e.size[c.group]/4).Scan(&deep.ts, &deep.id); err != nil {
			t.Fatal(err)
		}
		for _, from := range []*sgCursor{nil, &deep} {
			got, want := e.pages(t, q, 4, from), e.pages(t, arr, 4, from)
			if len(got) != len(want) || len(got) == 0 {
				t.Fatalf("%s: %d scalar pages, %d array pages", c.name, len(got), len(want))
			}
			for i := range got {
				if !slices.Equal(got[i].ids, want[i].ids) || !slices.Equal(got[i].keys, want[i].keys) {
					t.Errorf("%s (deep %t) page %d: the scalar and the array form differ", c.name, from != nil, i)
				}
				if len(got[i].ids) == 0 {
					t.Errorf("%s page %d is empty", c.name, i)
				}
			}
		}
		// The handler serves the scalar form: its first page is the same rows.
		first := e.pages(t, q, 1, nil)[0]
		page := e.mustList(t, c.name, "asset", "filter[priority]=3&limit=50")
		var hids []uuid.UUID
		for _, it := range page.Items {
			hids = append(hids, uuid.MustParse(it.ID))
		}
		if !slices.Equal(hids, first.ids) {
			t.Errorf("%s: the handler's first page differs from the scalar form's", c.name)
		}
		// Cross-group: nothing of another group, nothing unassigned.
		for _, id := range hids {
			var g *uuid.UUID
			if err := e.super.QueryRow(t.Context(), "SELECT assignment_group_id FROM records WHERE tenant_id = $1 AND id = $2", e.a.id, id).Scan(&g); err != nil {
				t.Fatal(err)
			}
			if g == nil || *g != c.group {
				t.Errorf("%s: row %s of group %v is in the page", c.name, id, g)
			}
		}
	}
}

// TICK-1084 RESULT figures, not a gate: the page statement's execution time,
// single client, for agents in 1, 2 and 3 groups (with and without the big
// group). Median of 5 runs after a warm-up, from EXPLAIN (ANALYZE).
func (e *sgEnv) testPageTimingsByGroupCount(t *testing.T) {
	for _, c := range []struct {
		name   string
		groups []uuid.UUID
	}{
		{"1 group (big)", []uuid.UUID{e.big}},
		{"2 groups (big + small)", []uuid.UUID{e.big, e.small}},
		{"3 groups (big + small + tiny)", []uuid.UUID{e.big, e.small, e.tiny}},
		{"2 groups (small + tiny)", []uuid.UUID{e.small, e.tiny}},
	} {
		agent := e.agent(t, "timing-"+strconv.Itoa(len(c.groups))+"-"+c.groups[0].String()[:4], c.groups...)
		q := e.query(agent)
		var ms []float64
		var read float64
		for i := range 6 {
			plan := e.explain(t, q)
			if i == 0 {
				continue
			}
			ms = append(ms, plan["Execution Time"].(float64))
			read = 0
			for _, path := range groupScans(plan["Plan"].(map[string]any)) {
				s := path[len(path)-1]
				read += s["Actual Rows"].(float64) * s["Actual Loops"].(float64)
			}
		}
		sort.Float64s(ms)
		t.Logf("TICK-1084 timing | %s | median %.2f ms | min %.2f ms | records_list_group rows read %.0f | indexes %v", c.name, ms[len(ms)/2], ms[0], read, indexNames(e.explain(t, q)["Plan"].(map[string]any)))
		// The same statement when the planner reads records_list_group, as
		// TICK-1049's stack chose for the big group: records_list_updated is
		// dropped inside a transaction that is rolled back.
		ms = ms[:0]
		var idx []string
		for i := range 6 {
			plan := e.explainWithoutUpdatedIndex(t, q)
			idx = indexNames(plan["Plan"].(map[string]any))
			if i == 0 {
				continue
			}
			ms = append(ms, plan["Execution Time"].(float64))
			read = 0
			for _, path := range groupScans(plan["Plan"].(map[string]any)) {
				s := path[len(path)-1]
				read += s["Actual Rows"].(float64) * s["Actual Loops"].(float64)
			}
		}
		sort.Float64s(ms)
		t.Logf("TICK-1084 timing | %s, records_list_updated dropped | median %.2f ms | min %.2f ms | records_list_group rows read %.0f | indexes %v", c.name, ms[len(ms)/2], ms[0], read, idx)
	}
}

// indexNames is every index the plan scans, in plan order.
func indexNames(n map[string]any) []string {
	var out []string
	if s, ok := n["Index Name"].(string); ok {
		out = append(out, s)
	}
	for _, k := range sgKids(n) {
		out = append(out, indexNames(k)...)
	}
	return out
}

// errRollback ends explainWithoutUpdatedIndex's transaction.
var errRollback = errors.New("rollback")

// explainWithoutUpdatedIndex is explain with records_list_updated dropped in
// a tenant transaction that is rolled back: the superuser drops it, then the
// statement runs as nava_app (row-level security on) in the pool's exec mode.
func (e *sgEnv) explainWithoutUpdatedIndex(t *testing.T, q pipeline.ListQuery) map[string]any {
	t.Helper()
	sql, args, err := pipeline.PageStatement(q)
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	pc, err := pgxpool.ParseConfig("")
	if err != nil {
		t.Fatal(err)
	}
	pc.ConnConfig = e.super.Config().Copy()
	pc.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	pc.MaxConns = 1
	super, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		t.Fatal(err)
	}
	defer super.Close()
	var raw string
	err = tenancy.InTenantTx(ctx, super, e.a.id, func(tx pgx.Tx) error {
		for _, s := range []string{"DROP INDEX records_list_updated", "SET LOCAL ROLE nava_app"} {
			if _, err := tx.Exec(ctx, s); err != nil {
				return err
			}
		}
		if err := tx.QueryRow(ctx, "EXPLAIN (ANALYZE, FORMAT JSON) "+sql, args...).Scan(&raw); err != nil {
			return err
		}
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatalf("explain: %v", err)
	}
	var doc []map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatal(err)
	}
	return doc[0]
}
