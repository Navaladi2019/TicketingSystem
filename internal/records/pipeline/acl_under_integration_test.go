//go:build integration

// TICK-950 P11-17 (ADR-010 D7, D5a) against a real PostgreSQL as nava_app,
// under RLS: authz.ExpandUnder computes set S from a real org-unit tree with
// one recursive query over live units, and the under leaf gives the same rows
// in SQL (CompileRead, ReadBranches) and in Go (Allows over the single-record
// loader, which fills Row.OrgUnitID). A tombstoned unit drops out of S with
// its whole subtree, live descendants included; a NULL org_unit_id is never
// under; more than authz.MaxUnderSet units is a typed error.
package pipeline_test

import (
	"context"
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
	return w
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

// rows returns, for pol bound by ExpandUnder in a tenant transaction, the
// records the SQL predicate admits, those Allows admits over the loader's
// rows, and those the ReadBranches UNION ALL admits (nil when it does not
// split).
func (w underWorld) rows(t *testing.T, e env, pol *authz.Policy, p identity.Principal) (bySQL, byGo, byBranches []string) {
	t.Helper()
	ctx := context.Background()
	inTx(t, e, w.f.id, func(tx pgx.Tx) error {
		bound, err := authz.ExpandUnder(ctx, tx, pol, p)
		if err != nil {
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
		for _, id := range w.records {
			rec, err := pipelineGet(ctx, tx, w.f.id, id)
			if err != nil {
				return err
			}
			if _, _, ok := authz.Allows(bound, p, authz.OpRead, rec); ok {
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
		{"a tombstoned unit listed: empty S, false", of("Lx"), u1, []string{}},
		{"tombstoned root listed: its live child is out too", of("T"), u1, []string{}},
		{"two roots, overlapping", of("A", "A1", "L"), u1, []string{"A", "A1", "L"}},
		{"another tenant's unit: empty S", of("X"), u1, []string{}},
		{"NULL org_unit_id and the excluded units are not under", `{"not":` + of("R") + `}`, u1, []string{"Lx", "T", "T1", "none"}},
		{"caller: u1's units A and L", caller, u1, []string{"A", "A1", "L"}},
		// D7 (fail closed): T1 dropped out of S with tombstoned T's subtree,
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
	// whose under leaf has no set S for the caller (fails closed), even under
	// a not.
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
}

// More than authz.MaxUnderSet units in S is *authz.UnderSetTooLargeError;
// exactly MaxUnderSet is fine.
func TestUnderSetAboveTheCapIsATypedError(t *testing.T) {
	e := migrated(t)
	w := seedACLWorld(t, e)
	ctx := context.Background()
	root := uuid.New()
	mustExec(t, e.super, `INSERT INTO org_units (tenant_id, id, keycloak_group_id, name) VALUES ($1, $2, 'kc-cap-root', 'root')`, w.f.id, root)
	// root plus MaxUnderSet-1 children: exactly MaxUnderSet units.
	mustExec(t, e.super, `INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name)
		SELECT $1, gen_random_uuid(), $2, 'kc-cap-' || n, 'c' || n FROM generate_series(1, $3::int) n`, w.f.id, root, authz.MaxUnderSet-1)
	pol := underDoc(t, `{"under":{"of":["`+root.String()+`"]}}`)
	p := identity.Principal{TenantID: w.f.id, PrincipalID: w.u1}
	inTx(t, e, w.f.id, func(tx pgx.Tx) error {
		if _, err := authz.ExpandUnder(ctx, tx, pol, p); err != nil {
			t.Errorf("S of exactly %d units: %v", authz.MaxUnderSet, err)
		}
		return nil
	})
	mustExec(t, e.super, `INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name) VALUES ($1, gen_random_uuid(), $2, 'kc-cap-over', 'over')`, w.f.id, root)
	inTx(t, e, w.f.id, func(tx pgx.Tx) error {
		_, err := authz.ExpandUnder(ctx, tx, pol, p)
		var big *authz.UnderSetTooLargeError
		if !errors.As(err, &big) {
			t.Errorf("S of %d units: err = %v, want *authz.UnderSetTooLargeError", authz.MaxUnderSet+1, err)
		}
		if err != nil && strings.Contains(err.Error(), root.String()) {
			t.Error("the error names a unit id")
		}
		return nil
	})
	// A tombstoned child brings S back under the cap.
	mustExec(t, e.super, `UPDATE org_units SET deleted_at = now() WHERE tenant_id = $1 AND keycloak_group_id = 'kc-cap-over'`, w.f.id)
	inTx(t, e, w.f.id, func(tx pgx.Tx) error {
		if _, err := authz.ExpandUnder(ctx, tx, pol, p); err != nil {
			t.Errorf("after tombstoning one unit: %v", err)
		}
		return nil
	})
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
}
