//go:build integration

// TICK-950 P11-17 and TICK-1034 (ADR-010 D7 as amended, D5a) against a real
// PostgreSQL as nava_app, under RLS: the under leaf reads org_unit_closure,
// filled from a real org-unit tree, and gives the same rows in SQL
// (CompileRead, ReadBranches) and in Go (Allows over the single-record loader,
// which fills Row.OrgUnitID, bound by authz.ResolveUnder's closure probe). A
// tombstoned unit drops out with its whole subtree, live descendants included;
// a NULL org_unit_id is never under; a region of more than 10,000 units is a
// normal answer, and its count reads at most K + 1 records.
package pipeline_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

// underWorld is the aclWorld tenant with an org tree and one record per unit:
//
//	R ── A ── A1
//	├─── T† ── T1        T is tombstoned, T1 is live below it
//	└─── L ── Lx†        Lx is tombstoned
//
// plus one record with no unit, and another tenant's live unit X. u1 is a
// member of A and L, u2 of T1 (a live unit under a tombstoned one, so out of
// every S: D7 drops a tombstoned unit's whole subtree).
type underWorld struct {
	aclWorld
	unit   map[string]uuid.UUID // name → unit id
	record map[string]uuid.UUID // unit name (or "none") → record id
}

func seedUnderWorld(t *testing.T, e env) underWorld {
	t.Helper()
	ctx := context.Background()
	w := underWorld{aclWorld: seedACLWorld(t, e), unit: map[string]uuid.UUID{}, record: map[string]uuid.UUID{}}
	other := seedTenantWith(t, e, aclDefinition())
	tree := []struct {
		name, parent string
		dead         bool
	}{
		{"R", "", false}, {"A", "R", false}, {"A1", "A", false},
		{"T", "R", true}, {"T1", "T", false}, {"L", "R", false}, {"Lx", "L", true},
	}
	for _, u := range tree {
		id := uuid.New()
		w.unit[u.name] = id
		var parent any
		if u.parent != "" {
			parent = w.unit[u.parent]
		}
		mustExec(t, e.super, `INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name, deleted_at)
			VALUES ($1, $2, $3, $4, $5, CASE WHEN $6 THEN now() END)`, w.f.id, id, parent, "kc-ou-"+id.String(), u.name, u.dead)
	}
	w.unit["X"] = uuid.New()
	mustExec(t, e.super, `INSERT INTO org_units (tenant_id, id, keycloak_group_id, name) VALUES ($1, $2, $3, 'X')`,
		other.id, w.unit["X"], "kc-ou-"+w.unit["X"].String())
	for _, m := range []struct {
		unit string
		who  uuid.UUID
	}{{"A", w.u1}, {"L", w.u1}, {"T1", w.u2}} {
		mustExec(t, e.super, `INSERT INTO org_unit_members (tenant_id, org_unit_id, principal_id) VALUES ($1, $2, $3)`, w.f.id, w.unit[m.unit], m.who)
	}
	// One record per unit of the tenant and one with none. The write
	// pipeline sets org_unit_id on create only from TICK-949, so the test
	// sets it directly, past the audit commit check (replica role).
	sys := identity.Principal{TenantID: w.f.id}
	tx, err := e.super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SET LOCAL session_replication_role = replica"); err != nil {
		t.Fatal(err)
	}
	w.records = nil
	for _, name := range []string{"R", "A", "A1", "T", "T1", "L", "Lx", "none"} {
		res, err := write(ctx, e, w.p, w.f.id, pipelineCreate(sys, map[string]any{"title": "u-" + name}))
		if err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
		w.record[name] = res.RecordID
		w.records = append(w.records, res.RecordID)
		if name == "none" {
			continue
		}
		if _, err := tx.Exec(ctx, `UPDATE records SET org_unit_id = $3 WHERE tenant_id = $1 AND id = $2`, w.f.id, res.RecordID, w.unit[name]); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	rebuildClosure(t, e)
	return w
}

// rebuildClosure fills org_unit_closure from org_units, for every tenant, as
// the migration's backfill does (org_unit_closure_rebuild_all, nava_migrator
// only). The tests seed org_units directly, so they seed the closure the same
// way; group sync keeps it current in production (TICK-1033).
func rebuildClosure(t *testing.T, e env) {
	t.Helper()
	ctx := context.Background()
	tx, err := e.super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE nava_migrator"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "SELECT org_unit_closure_rebuild_all()"); err != nil {
		t.Fatalf("rebuild the closure: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

// names maps record ids back to their unit names, sorted.
func (w underWorld) names(ids []uuid.UUID) []string {
	out := []string{}
	for name, id := range w.record {
		if slices.Contains(ids, id) {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out
}

// rows returns, for pol bound by ResolveUnder in a tenant transaction (with
// the loader's row of every record), the records the SQL predicate admits,
// those Allows admits over the loader's rows, and those the ReadBranches
// UNION ALL admits (nil when it does not split). It also checks that the list
// count with the closure-driven region term equals the count without it.
func (w underWorld) rows(t *testing.T, e env, pol *authz.Policy, p identity.Principal) (bySQL, byGo, byBranches []string) {
	t.Helper()
	ctx := context.Background()
	inTx(t, e, w.f.id, func(tx pgx.Tx) error {
		recs := make([]authz.Row, len(w.records))
		for i, id := range w.records {
			rec, err := pipelineGet(ctx, tx, w.f.id, id)
			if err != nil {
				return err
			}
			recs[i] = rec
		}
		bound, err := authz.ResolveUnder(ctx, tx, pol, p, recs...)
		if err != nil {
			return err
		}
		if err := w.countsAgree(ctx, tx, bound, p); err != nil {
			return err
		}
		const from = `SELECT r.id FROM records r JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
			WHERE r.tenant_id = $1 AND d.key = $2 AND `
		pred, args := authz.CompileRead(bound, p, "r", 2)
		rows, err := tx.Query(ctx, from+pred, append([]any{w.f.id, aclKey}, args...)...)
		if err != nil {
			return fmt.Errorf("predicate %s: %w", pred, err)
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		if err != nil {
			return err
		}
		bySQL = w.names(ids)
		var goIDs []uuid.UUID
		for i, id := range w.records {
			if _, _, ok := authz.Allows(bound, p, authz.OpRead, recs[i]); ok {
				goIDs = append(goIDs, id)
			}
		}
		byGo = w.names(goIDs)
		branches, bargs := authz.ReadBranches(bound, p, "r", 2)
		if len(branches) < 2 {
			return nil
		}
		arms := make([]string, len(branches))
		for i, b := range branches {
			arms[i] = from + b
		}
		rows, err = tx.Query(ctx, strings.Join(arms, " UNION ALL "), append([]any{w.f.id, aclKey}, bargs...)...)
		if err != nil {
			return fmt.Errorf("branches %v: %w", branches, err)
		}
		ids, err = pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		if err != nil {
			return err
		}
		for i := 1; i < len(ids); i++ {
			if slices.Contains(ids[:i], ids[i]) {
				t.Errorf("record %s is admitted by two branches", ids[i])
			}
		}
		byBranches = w.names(ids)
		return nil
	})
	return bySQL, byGo, byBranches
}

// countsAgree runs the list's capped count for pol twice, as the list route
// builds it (authz.ReadList): with the closure drivers and without them. The
// driven region term must count exactly the rows the plain one does.
func (w underWorld) countsAgree(ctx context.Context, tx pgx.Tx, pol *authz.Policy, p identity.Principal) error {
	read := authz.ReadList(pol, p, "r", 2)
	q := pipeline.ListQuery{TenantID: w.f.id, DefinitionID: w.def(ctx, tx), Where: "true", Args: read.Args, Branches: read.Branches}
	if read.Branches == nil {
		q.Where = read.Where
	}
	args := append([]any{q.TenantID, q.DefinitionID}, q.Args...)
	var plain, driven int64
	if err := tx.QueryRow(ctx, pipeline.CountStatement(q), args...).Scan(&plain); err != nil {
		return fmt.Errorf("plain count: %w", err)
	}
	for _, d := range read.Drivers {
		q.CountDrivers = append(q.CountDrivers, pipeline.CountDriver{From: d.From, Key: d.Key})
	}
	if err := tx.QueryRow(ctx, pipeline.CountStatement(q), args...).Scan(&driven); err != nil {
		return fmt.Errorf("driven count %s: %w", pipeline.CountStatement(q), err)
	}
	if plain != driven {
		return fmt.Errorf("driven count %d, plain count %d: %s", driven, plain, pipeline.CountStatement(q))
	}
	return nil
}

// def is the world's entity definition id.
func (w underWorld) def(ctx context.Context, tx pgx.Tx) uuid.UUID {
	var id uuid.UUID
	_ = tx.QueryRow(ctx, `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2`, w.f.id, aclKey).Scan(&id)
	return id
}

func underDoc(t *testing.T, when string) *authz.Policy {
	t.Helper()
	d, err := authz.Decode([]byte(`{"format":2,"conditions":[],"policies":[{"name":"p","ops":["read"],"when":` + when + `}]}`))
	if err != nil {
		t.Fatalf("decode %s: %v", when, err)
	}
	return &authz.Policy{Version: 1, EntityType: aclKey, Document: d}
}

func TestUnderSetFromARealTreeAgreesInSQLAndGo(t *testing.T) {
	e := migrated(t)
	w := seedUnderWorld(t, e)
	of := func(names ...string) string {
		ids := make([]string, len(names))
		for i, n := range names {
			ids[i] = `"` + w.unit[n].String() + `"`
		}
		return `{"under":{"of":[` + strings.Join(ids, ",") + `]}}`
	}
	caller := `{"under":{"of":"caller"}}`
	u1 := identity.Principal{TenantID: w.f.id, PrincipalID: w.u1}
	u2 := identity.Principal{TenantID: w.f.id, PrincipalID: w.u2}
	// u3 is a member of A and of A1 below it: overlapping caller units.
	u3 := identity.Principal{TenantID: w.f.id, PrincipalID: uuid.New()}
	mustExec(t, e.super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at) VALUES ($1, $2, $3, true, now())`,
		w.f.id, u3.PrincipalID, "kc-"+u3.PrincipalID.String())
	for _, unit := range []string{"A", "A1"} {
		mustExec(t, e.super, `INSERT INTO org_unit_members (tenant_id, org_unit_id, principal_id) VALUES ($1, $2, $3)`, w.f.id, w.unit[unit], u3.PrincipalID)
	}
	noID := identity.Principal{TenantID: w.f.id, Groups: []uuid.UUID{w.g1}}
	requester := `{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}`
	for _, tc := range []struct {
		name string
		when string
		p    identity.Principal
		want []string
	}{
		{"direct unit, a leaf of the tree", of("A1"), u1, []string{"A1"}},
		{"descendant", of("A"), u1, []string{"A", "A1"}},
		{"tombstoned ancestor: T and its live child T1 are out", of("R"), u1, []string{"A", "A1", "L", "R"}},
		{"tombstoned leaf is out", of("L"), u1, []string{"L"}},
		{"a tombstoned unit listed: no closure row, false", of("Lx"), u1, []string{}},
		{"tombstoned root listed: its live child is out too", of("T"), u1, []string{}},
		{"two roots, overlapping", of("A", "A1", "L"), u1, []string{"A", "A1", "L"}},
		{"another tenant's unit: no closure row in this tenant", of("X"), u1, []string{}},
		{"NULL org_unit_id and the excluded units are not under", `{"not":` + of("R") + `}`, u1, []string{"Lx", "T", "T1", "none"}},
		{"caller: u1's units A and L", caller, u1, []string{"A", "A1", "L"}},
		// The AC's EMEA and EMEA/Germany: rows and count as for A alone (the
		// count driver drops A1, which is below A; countsAgree checks it).
		{"caller: u3 in A and A1, same as A alone", caller, u3, []string{"A", "A1"}},
		// D7 (fail closed): T1 dropped out with tombstoned T's subtree,
		// so being a direct member of it grants nothing; listing it neither.
		{"caller: u2 is a member of live T1 below tombstoned T: excluded", caller, u2, []string{}},
		{"a listed live unit below a tombstoned one: excluded", of("T1"), u1, []string{}},
		{"caller with no id: false", caller, noID, []string{}},
		{"not under caller", `{"not":` + caller + `}`, u1, []string{"Lx", "R", "T", "T1", "none"}},
		{"under or requester: split into branches", `{"any":[` + caller + `,` + requester + `]}`, u1, []string{"A", "A1", "L"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bySQL, byGo, byBranches := w.rows(t, e, underDoc(t, tc.when), tc.p)
			if !slices.Equal(bySQL, tc.want) {
				t.Errorf("SQL admits %v, want %v", bySQL, tc.want)
			}
			if !slices.Equal(byGo, bySQL) {
				t.Errorf("Allows admits %v, SQL %v", byGo, bySQL)
			}
			if byBranches != nil && !slices.Equal(byBranches, bySQL) {
				t.Errorf("ReadBranches admit %v, SQL %v", byBranches, bySQL)
			}
		})
	}

	// Unbound, the same policies grant nothing: the planner voids a policy
	// whose under leaf has no binding for the caller (fails closed), even
	// under a not.
	for _, when := range []string{of("R"), `{"not":` + of("R") + `}`, caller} {
		pol := underDoc(t, when)
		inTx(t, e, w.f.id, func(tx pgx.Tx) error {
			for _, id := range w.records {
				rec, err := pipelineGet(context.Background(), tx, w.f.id, id)
				if err != nil {
					return err
				}
				if _, _, ok := authz.Allows(pol, u1, authz.OpRead, rec); ok {
					t.Errorf("unbound %s admits %s", when, id)
				}
			}
			return nil
		})
	}

	// of [ids] that became nested after a tree move (the AC): L moves under
	// A, the closure is rebuilt, and of [A, L] gives the rows and the capped
	// count of of [A] alone.
	mustExec(t, e.super, `UPDATE org_units SET parent_id = $3 WHERE tenant_id = $1 AND id = $2`, w.f.id, w.unit["L"], w.unit["A"])
	rebuildClosure(t, e)
	alone, nested := underDoc(t, of("A")), underDoc(t, of("A", "L"))
	wantRows, _, _ := w.rows(t, e, alone, u1)
	gotRows, gotGo, _ := w.rows(t, e, nested, u1)
	if want := []string{"A", "A1", "L"}; !slices.Equal(wantRows, want) {
		t.Fatalf("of [A] after the move admits %v, want %v", wantRows, want)
	}
	if !slices.Equal(gotRows, wantRows) || !slices.Equal(gotGo, wantRows) {
		t.Errorf("of [A, L] after the move: SQL %v, Allows %v; want of [A]'s %v", gotRows, gotGo, wantRows)
	}
	if a, n := w.drivenCount(t, e, alone, u1), w.drivenCount(t, e, nested, u1); a != n || a != int64(len(wantRows)) {
		t.Errorf("capped count of [A] %d, of [A, L] %d; want both %d", a, n, len(wantRows))
	}
}

// drivenCount is the list's capped count for pol, with the closure drivers,
// as the list route runs it.
func (w underWorld) drivenCount(t *testing.T, e env, pol *authz.Policy, p identity.Principal) int64 {
	t.Helper()
	ctx := context.Background()
	var n int64
	inTx(t, e, w.f.id, func(tx pgx.Tx) error {
		bound, err := authz.ResolveUnder(ctx, tx, pol, p)
		if err != nil {
			return err
		}
		read := authz.ReadList(bound, p, "r", 2)
		q := pipeline.ListQuery{TenantID: w.f.id, DefinitionID: w.def(ctx, tx), Where: "true", Args: read.Args, Branches: read.Branches}
		if read.Branches == nil {
			q.Where = read.Where
		}
		for _, d := range read.Drivers {
			q.CountDrivers = append(q.CountDrivers, pipeline.CountDriver{From: d.From, Key: d.Key})
		}
		return tx.QueryRow(ctx, pipeline.CountStatement(q), append([]any{q.TenantID, q.DefinitionID}, q.Args...)...).Scan(&n)
	})
	return n
}

// A region of more than 10,000 units (the old MaxUnderSet) is a normal
// answer now (TICK-1034): Allows, the list predicate and the count answer for
// a record deep in it. The driven count reads at most K + 1 = 1,001 records of
// a region holding more, so its cost does not grow with the region.
func TestUnderAboveTenThousandUnitsIsANormalAnswer(t *testing.T) {
	e := migrated(t)
	w := seedACLWorld(t, e)
	ctx := context.Background()
	root := uuid.New()
	mustExec(t, e.super, `INSERT INTO org_units (tenant_id, id, keycloak_group_id, name) VALUES ($1, $2, 'kc-cap-root', 'root')`, w.f.id, root)
	mustExec(t, e.super, `INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name)
		SELECT $1, gen_random_uuid(), $2, 'kc-cap-' || n, 'c' || n FROM generate_series(1, 10001) n`, w.f.id, root)
	rebuildClosure(t, e)
	var def uuid.UUID
	if err := e.super.QueryRow(ctx, `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2`, w.f.id, aclKey).Scan(&def); err != nil {
		t.Fatal(err)
	}
	// 1,500 records spread over the region's units, past the audit commit
	// check (replica role): this test reads answers and plans, not audit.
	tx, err := e.super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SET LOCAL session_replication_role = replica"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO records (tenant_id, id, entity_definition_id, version, schema_version, number, state, org_unit_id, created_at, updated_at, data)
		SELECT $1, gen_random_uuid(), $2, 1, 1, 'BIG-' || g, 'open', u.id, now(), now(), '{}'
		FROM generate_series(1, 1500) g
		JOIN LATERAL (SELECT id FROM org_units WHERE tenant_id = $1 AND keycloak_group_id = 'kc-cap-' || (g % 10001 + 1)) u ON true`, w.f.id, def); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	mustExec(t, e.super, `ANALYZE records`)
	mustExec(t, e.super, `ANALYZE org_unit_closure`)

	pol := underDoc(t, `{"under":{"of":["`+root.String()+`"]}}`)
	p := identity.Principal{TenantID: w.f.id, PrincipalID: w.u1}
	inTx(t, e, w.f.id, func(tx pgx.Tx) error {
		var deep uuid.UUID
		var unit uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT id, org_unit_id FROM records WHERE tenant_id = $1 AND number = 'BIG-1500'`, w.f.id).Scan(&deep, &unit); err != nil {
			return err
		}
		row := authz.Row{ID: deep, State: "open", OrgUnitID: &unit}
		bound, err := authz.ResolveUnder(ctx, tx, pol, p, row)
		if err != nil {
			return fmt.Errorf("ResolveUnder over 10,002 units: %w", err)
		}
		if _, _, ok := authz.Allows(bound, p, authz.OpRead, row); !ok {
			t.Error("a record in the region is refused")
		}
		read := authz.ReadList(bound, p, "r", 2)
		q := pipeline.ListQuery{TenantID: w.f.id, DefinitionID: def, Where: read.Where, Args: read.Args}
		for _, d := range read.Drivers {
			q.CountDrivers = append(q.CountDrivers, pipeline.CountDriver{From: d.From, Key: d.Key})
		}
		if len(q.CountDrivers) != 1 || q.CountDrivers[0].From == "" {
			t.Fatalf("the region term has no count driver: %+v", read.Drivers)
		}
		args := append([]any{w.f.id, def}, q.Args...)
		var n int64
		if err := tx.QueryRow(ctx, pipeline.CountStatement(q), args...).Scan(&n); err != nil {
			return fmt.Errorf("count: %w", err)
		}
		if n != 1001 {
			t.Errorf("capped count = %d, want 1001 (TotalCap + 1)", n)
		}
		var plan string
		if err := tx.QueryRow(ctx, "EXPLAIN (ANALYZE, FORMAT JSON) "+pipeline.CountStatement(q), args...).Scan(&plan); err != nil {
			return fmt.Errorf("explain: %w", err)
		}
		if got := recordsRead(t, plan); got > 1001 {
			t.Errorf("the driven count read %d records, want at most 1001:\n%s", got, plan)
		}
		return nil
	})
}

// recordsRead sums, over every scan of records in a JSON EXPLAIN ANALYZE
// plan, the rows it returned times its loops: the records the statement read.
func recordsRead(t *testing.T, plan string) int64 {
	t.Helper()
	var nodes []struct {
		Plan map[string]any `json:"Plan"`
	}
	if err := json.Unmarshal([]byte(plan), &nodes); err != nil || len(nodes) != 1 {
		t.Fatalf("plan: %v", err)
	}
	var walk func(n map[string]any) int64
	walk = func(n map[string]any) int64 {
		var sum int64
		if n["Relation Name"] == "records" {
			rows, _ := n["Actual Rows"].(float64)
			loops, _ := n["Actual Loops"].(float64)
			sum += int64(rows * loops)
		}
		kids, _ := n["Plans"].([]any)
		for _, k := range kids {
			if m, ok := k.(map[string]any); ok {
				sum += walk(m)
			}
		}
		return sum
	}
	return walk(nodes[0].Plan)
}

func pipelineCreate(actor identity.Principal, data map[string]any) pipeline.Command {
	return pipeline.Command{Op: pipeline.OpCreate, EntityKey: aclKey, Data: data, Actor: actor, Source: pipeline.SourceSystem}
}

// pipelineGet is the record as the single-record loader reads it, as the
// ACL sees it (these records have no shares).
func pipelineGet(ctx context.Context, tx pgx.Tx, tenant, id uuid.UUID) (authz.Row, error) {
	rec, err := pipeline.Get(ctx, tx, tenant, id)
	if err != nil {
		return authz.Row{}, err
	}
	return pipeline.RowWith(rec, nil), nil
}

// A create under a policy whose only write grant is under (TICK-950): the
// pipeline resolves the record's unit (the requester's primary unit) before
// the write check, so CanWrite sees the row's real OrgUnitID. The create
// succeeds when that unit is in S and stores it; it is refused, writing
// nothing, when the unit is outside S or the requester has none.
func TestCreateUnderAnUnderOnlyWriteGrant(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()
	in, out := seedOrgUnit(t, e, f.id, "In"), seedOrgUnit(t, e, f.id, "Out")
	child := uuid.New()
	mustExec(t, e.super, `INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name) VALUES ($1, $2, $3, $4, 'In child')`,
		f.id, child, in, "kc-"+child.String())
	rebuildClosure(t, e)
	inTx(t, e, f.id, func(tx pgx.Tx) error {
		prep, ps := authz.Prepare([]byte(`{"rules":{"format":2,"conditions":[],"policies":[{"name":"region","ops":["read","write"],"when":{"under":{"of":["`+in.String()+`"]}}}]}}`), authz.DefaultMaxDocumentBytes)
		if len(ps) > 0 {
			return fmt.Errorf("prepare: %v", ps)
		}
		cur, err := authz.PolicyStore{}.Newest(ctx, tx, entityKey)
		if err != nil {
			return err
		}
		_, err = authz.PolicyStore{}.PublishPrepared(ctx, tx, entityKey, cur.Version, authz.Publisher{System: "tick-950", Source: audit.SourceSystem}, prep)
		return err
	})
	records := func() int {
		return count(t, e.super, `SELECT count(*) FROM records WHERE tenant_id = $1`, f.id)
	}

	// The requester's primary unit is below the listed one: in S.
	seedMember(t, e, f.id, child, f.principal.PrincipalID, true)
	res, err := write(ctx, e, p, f.id, createCmd(f, map[string]any{"title": "in S"}))
	if err != nil {
		t.Fatalf("create with the requester's unit in S: %v", err)
	}
	if got := recordUnit(t, e, res.RecordID); got == nil || *got != child {
		t.Fatalf("org_unit_id = %v, want %s (the unit the check saw)", got, child)
	}

	// Primary unit outside S: refused, nothing written.
	mustExec(t, e.super, `UPDATE org_unit_members SET is_primary = false WHERE tenant_id = $1 AND principal_id = $2`, f.id, f.principal.PrincipalID)
	seedMember(t, e, f.id, out, f.principal.PrincipalID, true)
	before := records()
	if _, err := write(ctx, e, p, f.id, createCmd(f, map[string]any{"title": "outside S"})); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("create with the requester's unit outside S: %v, want ErrForbidden", err)
	}
	// No primary unit: NULL is never under.
	mustExec(t, e.super, `UPDATE org_unit_members SET is_primary = false WHERE tenant_id = $1 AND principal_id = $2`, f.id, f.principal.PrincipalID)
	if _, err := write(ctx, e, p, f.id, createCmd(f, map[string]any{"title": "no unit"})); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("create with no primary unit: %v, want ErrForbidden", err)
	}
	if n := records(); n != before {
		t.Fatalf("refused creates wrote: %d records, want %d", n, before)
	}

	// The requester is not the actor (resolveUnit reads store.PrimaryUnit):
	// the record's unit is the requester's primary unit, not the actor's. The
	// actor now has none, so only the requester's unit can grant.
	other := uuid.New()
	mustExec(t, e.super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at) VALUES ($1, $2, $3, true, now())`,
		f.id, other, "kc-"+other.String())
	seedMember(t, e, f.id, child, other, true)
	res, err = write(ctx, e, p, f.id, createCmd(f, map[string]any{"title": "for another, in S", "requester": other.String()}))
	if err != nil {
		t.Fatalf("create for a requester whose unit is in S: %v", err)
	}
	if got := recordUnit(t, e, res.RecordID); got == nil || *got != child {
		t.Fatalf("org_unit_id = %v, want the requester's unit %s", got, child)
	}
	// The actor's primary unit is in S, the requester's is not: refused, so
	// the check reads the requester's unit, never the actor's.
	mustExec(t, e.super, `UPDATE org_unit_members SET is_primary = (org_unit_id = $3) WHERE tenant_id = $1 AND principal_id = $2`, f.id, f.principal.PrincipalID, child)
	mustExec(t, e.super, `UPDATE org_unit_members SET is_primary = false WHERE tenant_id = $1 AND principal_id = $2`, f.id, other)
	seedMember(t, e, f.id, out, other, true)
	before = records()
	if _, err := write(ctx, e, p, f.id, createCmd(f, map[string]any{"title": "for another, outside S", "requester": other.String()})); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("create for a requester whose unit is outside S (the actor's is in S): %v, want ErrForbidden", err)
	}
	if n := records(); n != before {
		t.Fatalf("a refused create wrote: %d records, want %d", n, before)
	}
}

// dupU returns args with every bound U equal to {x} replaced by {x, x}: the
// U a caller that skipped Go's sortedIDs dedup would bind (R2-N1).
func dupU(args []any, x uuid.UUID) []any {
	out := slices.Clone(args)
	for i, a := range out {
		if s, ok := a.([]string); ok && len(s) == 1 && s[0] == x.String() {
			out[i] = []string{s[0], s[0]}
		}
	}
	return out
}

// ADR-010-A2 A2-4, R2-N1: U = {A, A} gives the same rows and capped count as
// U = {A}. Go binds U without repeats, so the test binds the repeat itself;
// the SQL DISTINCT on the unnest(U) source is what holds then. With the
// DISTINCT removed (the u_not_canonical mutant, duplicate id) the driven
// count reads every record of A's subtree twice, and the test sees it.
func TestUnderDuplicateUCountsOnce(t *testing.T) {
	e := migrated(t)
	w := seedUnderWorld(t, e)
	ctx := context.Background()
	p := identity.Principal{TenantID: w.f.id, PrincipalID: w.u1}
	pol := underDoc(t, `{"under":{"of":["`+w.unit["A"].String()+`"]}}`)
	inTx(t, e, w.f.id, func(tx pgx.Tx) error {
		bound, err := authz.ResolveUnder(ctx, tx, pol, p)
		if err != nil {
			return err
		}
		read := authz.ReadList(bound, p, "r", 2)
		q := pipeline.ListQuery{TenantID: w.f.id, DefinitionID: w.def(ctx, tx), Where: read.Where, Args: read.Args}
		for _, d := range read.Drivers {
			q.CountDrivers = append(q.CountDrivers, pipeline.CountDriver{From: d.From, Key: d.Key})
		}
		if len(q.CountDrivers) != 1 || q.CountDrivers[0].From == "" {
			t.Fatalf("the region term has no count driver: %+v", read.Drivers)
		}
		count := func(sql string, args []any) int64 {
			var n int64
			if err := tx.QueryRow(ctx, sql, append([]any{q.TenantID, q.DefinitionID}, args...)...).Scan(&n); err != nil {
				t.Fatalf("count %s: %v", sql, err)
			}
			return n
		}
		rows := func(args []any) []string {
			r, err := tx.Query(ctx, `SELECT r.id FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND `+read.Where,
				append([]any{q.TenantID, q.DefinitionID}, args...)...)
			if err != nil {
				t.Fatal(err)
			}
			ids, err := pgx.CollectRows(r, pgx.RowTo[uuid.UUID])
			if err != nil {
				t.Fatal(err)
			}
			return w.names(ids)
		}
		dup := dupU(q.Args, w.unit["A"])
		if fmt.Sprint(dup) == fmt.Sprint(q.Args) {
			t.Fatalf("no U {A} bound in %v", q.Args)
		}
		want := []string{"A", "A1"}
		if got := rows(q.Args); !slices.Equal(got, want) {
			t.Fatalf("U = {A} admits %v, want %v", got, want)
		}
		if got := rows(dup); !slices.Equal(got, want) {
			t.Errorf("U = {A, A} admits %v, want %v", got, want)
		}
		cc := pipeline.CountStatement(q)
		if a, aa := count(cc, q.Args), count(cc, dup); a != 2 || aa != 2 {
			t.Errorf("capped count: U = {A} %d, U = {A, A} %d; want both 2", a, aa)
		}
		// Mutant u_not_canonical (duplicate id): no DISTINCT on U.
		mut := strings.Replace(cc, "(SELECT DISTINCT acl_ud.id FROM unnest(", "(SELECT acl_ud.id FROM unnest(", 1)
		if mut == cc {
			t.Fatalf("the count statement has no DISTINCT U to remove: %s", cc)
		}
		if n := count(mut, dup); n == 2 {
			t.Errorf("mutant without DISTINCT counts %d for U = {A, A}; the test cannot see a repeat", n)
		}
		return nil
	})
}
