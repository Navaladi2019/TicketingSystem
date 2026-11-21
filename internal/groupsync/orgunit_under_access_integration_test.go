//go:build integration

package groupsync_test

import (
	"context"
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// TICK-951 P11-18 (ADR-010 D7, S31 acceptance criterion 3): a unit that
// records still name is removed from the directory. The next sync succeeds,
// and access through the under leaf to that unit's records stops, for a
// caller leaf and for a leaf that lists the unit, in the SQL list predicate
// and in the Go single-record check alike. The records keep their
// org_unit_id. Then the caller's own unit goes: access through it stops too.
//
// The under leaf reads org_unit_closure (TICK-1034), and nothing here
// rebuilds it by hand: only group sync keeping the closure current (TICK-1033)
// makes this pass. It is the merge-order guard of the TICK-1032 security
// review: until then a removed unit would still grant its old subtree.
func TestOrgUnitRemovalEndsUnderAccessAndSyncSucceeds(t *testing.T) {
	super, pool := cluster(t)
	app := appPool(t, super)
	ctx := context.Background()
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)
	users := []identity.DirectoryUser{orgUser("agent", ""), orgUser("req", "")}
	emea, de, apac := path(tid, "/ou/emea"), path(tid, "/ou/emea/de"), path(tid, "/ou/apac")
	snap := func(units ...identity.DirectoryGroup) {
		t.Helper()
		dir.set(tid, identity.DirectoryChanges{Users: users, OrgUnits: units})
		if r := once(t, s); r.Err != nil {
			t.Fatalf("sync: %v", r.Err)
		}
	}
	snap(memberUnit("emea", "", emea, "agent"), memberUnit("de", "emea", de, "req"), memberUnit("apac", "", apac, "req"))

	unit := func(kc string) uuid.UUID {
		t.Helper()
		var id uuid.UUID
		if err := super.QueryRow(ctx, "SELECT id FROM org_units WHERE tenant_id = $1 AND keycloak_group_id = $2", tid, kc).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	var agent uuid.UUID
	if err := super.QueryRow(ctx, "SELECT id FROM principals WHERE tenant_id = $1 AND keycloak_user_id = 'agent'", tid).Scan(&agent); err != nil {
		t.Fatal(err)
	}
	emeaID, deID, apacID := unit("emea"), unit("de"), unit("apac")
	rows := map[string]*uuid.UUID{"in emea": &emeaID, "in de": &deID, "in apac": &apacID, "in no unit": nil}
	ids := map[string]uuid.UUID{}
	for name, u := range rows {
		ids[name] = uuid.New()
		exec(t, super, `WITH r AS (INSERT INTO records (tenant_id, id, org_unit_id) VALUES ($1, $2, $3) RETURNING tenant_id, id, version)
			INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
			SELECT tenant_id, gen_random_uuid(), 'record', 'create', 'record', id, version, 'system', 'system' FROM r`,
			tid, ids[name], u)
	}

	p := identity.Principal{TenantID: tid, PrincipalID: agent}
	caller := underPolicy(authz.Under{Caller: true})
	listed := underPolicy(authz.Under{Units: []uuid.UUID{deID}})
	check := func(when string, pol *authz.Policy, want ...string) {
		t.Helper()
		slices.Sort(want)
		sqlGot, goGot := underAccess(t, app, p, pol, rows, ids)
		if !slices.Equal(sqlGot, want) || !slices.Equal(goGot, want) {
			t.Errorf("%s: the list admits %v, Allows %v; want %v", when, sqlGot, goGot, want)
		}
	}
	check("before: under caller", caller, "in emea", "in de")
	check("before: under [de]", listed, "in de")

	// de goes from the directory; its records still name it.
	snap(memberUnit("emea", "", emea, "agent"), memberUnit("apac", "", apac, "req"))
	if n := count(t, super, "SELECT count(*) FROM org_units WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NOT NULL", tid, deID); n != 1 {
		t.Fatalf("de is not tombstoned (%d rows)", n)
	}
	if n := count(t, super, "SELECT count(*) FROM records WHERE tenant_id = $1 AND org_unit_id = $2", tid, deID); n != 1 {
		t.Fatalf("the record in de lost its org_unit_id (%d rows name de)", n)
	}
	check("after de is removed: under caller", caller, "in emea")
	check("after de is removed: under [de]", listed)

	// The caller's own unit goes: its membership with it.
	snap(memberUnit("apac", "", apac, "req"))
	check("after emea is removed: under caller", caller)
}

// underPolicy is a format-2 document with one read policy: the leaf.
func underPolicy(u authz.Under) *authz.Policy {
	return &authz.Policy{Version: 1, Document: authz.Document{
		Format:     authz.FormatV2,
		Conditions: []authz.Condition{},
		Policies:   []authz.NamedPolicy{{Name: "region", Ops: []string{authz.OpRead}, When: authz.Node{Under: &u}}},
	}}
}

// underAccess binds pol with authz.ResolveUnder (the caller's units and the
// closure probe for every record) as nava_app in p's tenant, then returns, by
// name, the records the CompileRead predicate admits and those authz.Allows
// admits, each sorted.
func underAccess(t *testing.T, app *pgxpool.Pool, p identity.Principal, pol *authz.Policy, units map[string]*uuid.UUID, ids map[string]uuid.UUID) ([]string, []string) {
	t.Helper()
	var bySQL, byGo []string
	err := tenancy.InTenantTx(context.Background(), app, p.TenantID, func(tx pgx.Tx) error {
		ctx := context.Background()
		recs := make([]authz.Row, 0, len(ids))
		for name, id := range ids {
			recs = append(recs, authz.Row{ID: id, OrgUnitID: units[name]})
		}
		bound, err := authz.ResolveUnder(ctx, tx, pol, p, recs...)
		if err != nil {
			return fmt.Errorf("ResolveUnder: %w", err)
		}
		pred, args := authz.CompileRead(bound, p, "r", 1)
		rows, err := tx.Query(ctx, "SELECT r.id FROM records r WHERE r.tenant_id = $1 AND "+pred, append([]any{p.TenantID}, args...)...)
		if err != nil {
			return fmt.Errorf("%s: %w", pred, err)
		}
		got, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		if err != nil {
			return err
		}
		for name, id := range ids {
			if slices.Contains(got, id) {
				bySQL = append(bySQL, name)
			}
			if _, _, ok := authz.Allows(bound, p, authz.OpRead, authz.Row{ID: id, OrgUnitID: units[name]}); ok {
				byGo = append(byGo, name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(bySQL)
	slices.Sort(byGo)
	return bySQL, byGo
}

// appPool logs in as nava_app to the platform database, the role the list
// and single-record reads run as.
func appPool(t *testing.T, super *pgx.Conn) *pgxpool.Pool {
	t.Helper()
	pc, err := pgxpool.ParseConfig(os.Getenv("NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	pc.ConnConfig.User, pc.ConnConfig.Password, pc.ConnConfig.Database = "nava_app", password("nava_app"), super.Config().Database
	pc.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	pc.MaxConns = 2
	pool, err := pgxpool.NewWithConfig(context.Background(), pc)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
