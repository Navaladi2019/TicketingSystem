//go:build integration

package groupsync_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// TICK-1035 (S31 fix; ADR-010 I5, D7 as amended by ADR-010-A2; PLAN-011): the
// under agreement on a closure that group sync writes, for the cases a
// closure can get wrong. Nothing here seeds org_unit_closure by hand: the
// tree comes from group sync runs, and the closure from its rebuild.
//
// Each check reads, as nava_app under RLS, the three renderings of a policy
// and compares them with the records of the expected units:
//
//   - Go: authz.Allows on every record, bound by authz.ResolveUnder;
//   - SQL: SELECT … WHERE authz.CompileRead;
//   - the list route: authz.ReadListShaped through pipeline.List, as
//     internal/server/records_list.go builds it (ResolveUnder with no rows,
//     the region count driver, the branches when the document splits), with
//     the page shape the probe picks and with C3 and W0 forced (TICK-1036,
//     ADR-010-A2 A2-5), the count always CC, and with R forced, page and
//     count (TICK-1104, ADR-010-A3 A3-1): every record on the page once, and
//     the count statement the readable count, capped at store.TotalCap.
//
// The sync-resilience test (a unit is removed and the sync still succeeds,
// S31 AC 3) is TestOrgUnitRemovalEndsUnderAccessAndSyncSucceeds.

// ucaRecords is how many records each unit holds, by Keycloak id; "" is no
// unit. The counts differ per unit so that a count which reads a subtree
// twice is told apart from one that reads another unit.
type ucaRecords map[string]int

// ucaUnits is every org unit of the tenant, tombstoned ones too, by Keycloak id.
func ucaUnits(t *testing.T, super *pgx.Conn, tid uuid.UUID) map[string]uuid.UUID {
	t.Helper()
	rows, err := super.Query(context.Background(), "SELECT keycloak_group_id, id FROM org_units WHERE tenant_id = $1", tid)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]uuid.UUID{}
	var (
		kc string
		id uuid.UUID
	)
	if _, err := pgx.ForEachRow(rows, []any{&kc, &id}, func() error { out[kc] = id; return nil }); err != nil {
		t.Fatal(err)
	}
	return out
}

// ucaPrincipal is the principal of Keycloak user kc.
func ucaPrincipal(t *testing.T, super *pgx.Conn, tid uuid.UUID, kc string) identity.Principal {
	t.Helper()
	var id uuid.UUID
	if err := super.QueryRow(context.Background(), "SELECT id FROM principals WHERE tenant_id = $1 AND keycloak_user_id = $2", tid, kc).Scan(&id); err != nil {
		t.Fatalf("principal %s: %v", kc, err)
	}
	return identity.Principal{TenantID: tid, PrincipalID: id}
}

// ucaRegion is a format-2 document whose one read policy is when; with
// shared, a second read policy (shared) makes ReadList split into branches,
// the region branch carrying the count driver.
func ucaRegion(when authz.Node, shared bool) *authz.Policy {
	pols := []authz.NamedPolicy{{Name: "region", Ops: []string{authz.OpRead}, When: when}}
	if shared {
		pols = append(pols, authz.NamedPolicy{Name: "shared", Ops: []string{authz.OpRead}, When: authz.Node{Shared: true}})
	}
	return &authz.Policy{Version: 1, Document: authz.Document{Format: authz.FormatV2, Conditions: []authz.Condition{}, Policies: pols}}
}

func ucaUnder(u authz.Under) authz.Node { return authz.Node{Under: &u} }

// ucaSeed commits an entity type of tid and recs, each write with its audit
// entry (the commit checks), and returns the type's id.
func ucaSeed(t *testing.T, super *pgx.Conn, tid uuid.UUID, recs ucaRecords) uuid.UUID {
	t.Helper()
	units := ucaUnits(t, super, tid)
	def := uuid.New()
	exec(t, super, `WITH d AS (INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version)
			VALUES ($1, $2, 'uca', 'UCA', 'UCA', 1) RETURNING tenant_id, id)
		INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, content_sha256, actor_system, source)
		SELECT tenant_id, gen_random_uuid(), 'config', 'publish', 'entity_definition', id, sha256('uca'::bytea), 'system', 'system' FROM d`,
		tid, def)
	keys := make([]string, 0, len(recs))
	for kc := range recs {
		keys = append(keys, kc)
	}
	slices.Sort(keys)
	next := 0
	for _, kc := range keys {
		var unit any
		if kc != "" {
			id, ok := units[kc]
			if !ok {
				t.Fatalf("no unit %q", kc)
			}
			unit = id
		}
		exec(t, super, `WITH r AS (INSERT INTO records (tenant_id, id, entity_definition_id, schema_version, number, state, org_unit_id)
				SELECT $1, gen_random_uuid(), $2, 1, 'UCA' || lpad((g + $3)::text, 7, '0'), 'open', $4::uuid FROM generate_series(1, $5::int) g
				RETURNING tenant_id, id, version)
			INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
			SELECT tenant_id, gen_random_uuid(), 'record', 'create', 'record', id, version, 'system', 'system' FROM r`,
			tid, def, next, unit, recs[kc])
		next += recs[kc]
	}
	return def
}

// ucaCheck reads, as nava_app in tid (tenancy.InTenantTx), the records of
// type def, and checks that Go, the SQL predicate and the list route each
// admit exactly the records of the units in want (Keycloak ids, "" for no
// unit). Both document shapes (one policy, and split into branches) are
// checked.
func ucaCheck(t *testing.T, super *pgx.Conn, app *pgxpool.Pool, tid, def uuid.UUID, what string, p identity.Principal, when authz.Node, want ...string) {
	t.Helper()
	ctx := context.Background()
	units := ucaUnits(t, super, tid)
	nameOf := map[uuid.UUID]string{}
	for kc, id := range units {
		nameOf[id] = kc
	}
	// A caller leaf of a principal in no unit folds to false: no region
	// term is left to drive or to split.
	folded := when.Under != nil && when.Under.Caller &&
		count(t, super, "SELECT count(*) FROM org_unit_members WHERE tenant_id = $1 AND principal_id = $2", tid, p.PrincipalID) == 0
	if err := tenancy.InTenantTx(ctx, app, tid, func(tx pgx.Tx) error {
		ucaCheckTx(ctx, t, tx, nameOf, tid, def, what, p, when, folded, want)
		return nil
	}); err != nil {
		t.Fatalf("%s: %v", what, err)
	}
}

func ucaCheckTx(ctx context.Context, t *testing.T, tx pgx.Tx, nameOf map[uuid.UUID]string, tid, def uuid.UUID, what string, p identity.Principal, when authz.Node, folded bool, want []string) {
	t.Helper()
	rows, err := tx.Query(ctx, "SELECT id, org_unit_id FROM records WHERE tenant_id = $1 AND entity_definition_id = $2", tid, def)
	if err != nil {
		t.Fatal(err)
	}
	var all []authz.Row
	var (
		id   uuid.UUID
		unit *uuid.UUID
	)
	if _, err := pgx.ForEachRow(rows, []any{&id, &unit}, func() error {
		all = append(all, authz.Row{ID: id, OrgUnitID: unit})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(all) == 0 {
		t.Fatalf("%s: nava_app reads none of the seeded records", what)
	}
	unitName := func(r authz.Row) string {
		if r.OrgUnitID == nil {
			return ""
		}
		return nameOf[*r.OrgUnitID]
	}
	wantIDs := map[uuid.UUID]bool{}
	for _, r := range all {
		if slices.Contains(want, unitName(r)) {
			wantIDs[r.ID] = true
		}
	}
	// byUnit is a set of records as "unit×n", for a readable failure.
	byUnit := func(ids map[uuid.UUID]int) string {
		n := map[string]int{}
		for _, r := range all {
			if ids[r.ID] > 0 {
				n[unitName(r)] += ids[r.ID]
			}
		}
		var out []string
		for kc, k := range n {
			if kc == "" {
				kc = "(none)"
			}
			out = append(out, fmt.Sprintf("%s×%d", kc, k))
		}
		slices.Sort(out)
		return "[" + strings.Join(out, " ") + "]"
	}
	same := func(side string, got map[uuid.UUID]int) {
		t.Helper()
		ok := len(got) == len(wantIDs)
		for id, n := range got {
			ok = ok && n == 1 && wantIDs[id]
		}
		if !ok {
			wantN := map[uuid.UUID]int{}
			for id := range wantIDs {
				wantN[id] = 1
			}
			t.Errorf("%s: %s admits %s, want %s", what, side, byUnit(got), byUnit(wantN))
		}
	}

	for _, shared := range []bool{false, true} {
		pol := ucaRegion(when, shared)
		shape := "one policy"
		if shared {
			shape = "branches"
		}
		// Go: the single-record check, bound for every record.
		bound, err := authz.ResolveUnder(ctx, tx, pol, p, all...)
		if err != nil {
			t.Fatalf("%s: ResolveUnder: %v", what, err)
		}
		goGot := map[uuid.UUID]int{}
		for _, r := range all {
			if _, _, ok := authz.Allows(bound, p, authz.OpRead, r); ok {
				goGot[r.ID]++
			}
		}
		same(shape+", Allows", goGot)

		// SQL: the CompileRead predicate.
		pred, args := authz.CompileRead(bound, p, "r", 2)
		q, err := tx.Query(ctx, "SELECT r.id FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND "+pred,
			append([]any{tid, def}, args...)...)
		if err != nil {
			t.Fatalf("%s: %s: %v", what, pred, err)
		}
		sqlIDs, err := pgx.CollectRows(q, pgx.RowTo[uuid.UUID])
		if err != nil {
			t.Fatal(err)
		}
		sqlGot := map[uuid.UUID]int{}
		for _, id := range sqlIDs {
			sqlGot[id]++
		}
		same(shape+", CompileRead", sqlGot)

		// The list route: page and total.
		listPol, err := authz.ResolveUnder(ctx, tx, pol, p)
		if err != nil {
			t.Fatalf("%s: ResolveUnder for the list: %v", what, err)
		}
		// The list route, as TICK-1036 builds it: each region term's page in
		// the shape the probe picks (ChooseListShapes, page 51), and forced
		// C3 and W0, each with the count on CC, and forced R; every one agrees
		// with Allows.
		probed, err := authz.ChooseListShapes(ctx, tx, listPol, p, def, 51)
		if err != nil {
			t.Fatalf("%s: ChooseListShapes: %v", what, err)
		}
		for _, sh := range []struct {
			how    string
			shapes authz.ListShapes
		}{{"probed", probed}, {"C3 forced", authz.FixedListShapes(authz.ShapeC3)}, {"W0 forced", authz.FixedListShapes(authz.ShapeWalk)},
			{"R forced", authz.FixedListShapes(authz.ShapeRecords)}} {
			how, shapes := sh.how, sh.shapes
			read := authz.ReadListShaped(listPol, p, "r", 2, shapes)
			if shared && !folded && len(read.Branches) < 2 {
				t.Fatalf("%s: the two-policy document did not split (%d branches)", what, len(read.Branches))
			}
			lq := pipeline.ListQuery{TenantID: tid, DefinitionID: def, Where: "true", Args: slices.Clone(read.Args),
				Branches: read.Branches, CountBranches: read.CountBranches, Materialize: read.Materialize,
				SortColumn: "updated_at", Desc: true, Limit: len(all) + 1}
			if read.Branches == nil {
				lq.Where, lq.CountWhere = read.Where+" AND true", read.CountWhere+" AND true"
			}
			for _, d := range read.Drivers {
				lq.CountDrivers = append(lq.CountDrivers, pipeline.CountDriver{From: d.From, Key: d.Key})
			}
			if driven := slices.ContainsFunc(lq.CountDrivers, func(d pipeline.CountDriver) bool { return d.From != "" }); how == "R forced" {
				// R reads the type's records: no driver, its own leaf text.
				r := strings.Contains(read.Where+strings.Join(read.Branches, " "), ") IS TRUE)")
				if driven || r != (ucaDriven(when) && !folded) {
					t.Errorf("%s: %s, %s: driven %v, R leaf %v", what, shape, how, driven, r)
				}
			} else if !driven && ucaDriven(when) && !folded {
				t.Errorf("%s: %s, %s: the region term has no count driver, so total is not read through the closure", what, shape, how)
			}
			page, err := pipeline.List(ctx, tx, lq)
			if err != nil {
				t.Fatalf("%s: %s: list: %v", what, how, err)
			}
			pageGot := map[uuid.UUID]int{}
			for _, r := range page.Records {
				pageGot[r.ID]++
			}
			same(shape+", "+how+", the list page", pageGot)
			n := int64(len(wantIDs))
			// This page holds every record, so List takes the total from it and
			// runs no count (ADR-006-A2 A2-4).
			if page.Total != n || page.TotalState != store.TotalExact {
				t.Errorf("%s: %s, %s: total = %d %q, want %d exact from the full page", what, shape, how, page.Total, page.TotalState, n)
			}
			// A page that does not hold them all runs the region count.
			var counted int64
			if err := tx.QueryRow(ctx, pipeline.CountStatement(lq), append([]any{tid, def}, lq.Args...)...).Scan(&counted); err != nil {
				t.Fatalf("%s: %s, %s: count: %v", what, shape, how, err)
			}
			if counted != min(n, store.TotalCap+1) {
				t.Errorf("%s: %s, %s: the count reads %d, want %d of %d readable", what, shape, how, counted, min(n, store.TotalCap+1), n)
			}
			if n > store.TotalCap {
				lq.Limit = 51
				capped, err := pipeline.List(ctx, tx, lq)
				if err != nil {
					t.Fatalf("%s: %s, %s: capped list: %v", what, shape, how, err)
				}
				if capped.Total != store.TotalCap || !capped.TotalCapped || capped.TotalState != store.TotalCapped {
					t.Errorf("%s: %s, %s: total = %d (capped %v, %q), want %d of %d readable, capped",
						what, shape, how, capped.Total, capped.TotalCapped, capped.TotalState, store.TotalCap, n)
				}
			}
		}
	}
}

// ucaDriven reports whether the list counts when's region term from the
// closure: a bare under leaf is driven, a not under is not.
func ucaDriven(when authz.Node) bool { return when.Under != nil }

// SP-1e decision (nested U), a unit moved by group sync, and a tombstoned
// intermediate unit whose child is still live.
func TestUnderClosureAgreementNestedMovedAndTombstoned(t *testing.T) {
	super, pool := cluster(t)
	ctx := context.Background()
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)
	p := func(rest string) string { return path(tid, "/ou"+rest) }
	users := []identity.DirectoryUser{orgUser("nested", ""), orgUser("de-agent", ""), orgUser("fr-agent", "")}
	sync := func(what string, units ...identity.DirectoryGroup) {
		t.Helper()
		dir.set(tid, identity.DirectoryChanges{Users: users, OrgUnits: units})
		if r := once(t, s); r.Err != nil {
			t.Fatalf("%s: %v", what, r.Err)
		}
	}
	emea := memberUnit("emea", "", p("/emea"), "nested")
	de := memberUnit("de", "emea", p("/emea/de"), "nested", "de-agent")
	fr := memberUnit("fr", "emea", p("/emea/fr"), "fr-agent")
	apac := memberUnit("apac", "", p("/apac"))
	berlinUnderDE := memberUnit("berlin", "de", p("/emea/de/berlin"))
	sync("initial tree", emea, de, berlinUnderDE, fr, apac)

	app := appPool(t, super)
	def := ucaSeed(t, super, tid, ucaRecords{"emea": 2, "de": 3, "berlin": 4, "fr": 5, "apac": 6, "": 1})
	nested := ucaPrincipal(t, super, tid, "nested")
	deAgent := ucaPrincipal(t, super, tid, "de-agent")
	frAgent := ucaPrincipal(t, super, tid, "fr-agent")
	units := ucaUnits(t, super, tid)
	caller := ucaUnder(authz.Under{Caller: true})
	listed := func(kc string) authz.Node { return ucaUnder(authz.Under{Units: []uuid.UUID{units[kc]}}) }
	notUnder := func(kc string) authz.Node { n := listed(kc); return authz.Node{Not: &n} }

	// Nested U: the caller is in emea and in de, below it. de and berlin
	// are reached from both; each record is admitted once, and total is 14,
	// not the 21 of a count that reads de's subtree twice.
	ucaCheck(t, super, app, tid, def, "nested U, under caller", nested, caller, "emea", "de", "berlin", "fr")
	ucaCheck(t, super, app, tid, def, "de agent, under caller", deAgent, caller, "de", "berlin")
	ucaCheck(t, super, app, tid, def, "under [emea]", nested, listed("emea"), "emea", "de", "berlin", "fr")
	ucaCheck(t, super, app, tid, def, "not under [de]", nested, notUnder("de"), "emea", "fr", "apac", "")

	// Group sync moves berlin under fr: the answer follows the new parent.
	sync("berlin moves under fr", emea, de, memberUnit("berlin", "fr", p("/emea/fr/berlin")), fr, apac)
	ucaCheck(t, super, app, tid, def, "after the move, de agent", deAgent, caller, "de")
	ucaCheck(t, super, app, tid, def, "after the move, fr agent", frAgent, caller, "fr", "berlin")
	ucaCheck(t, super, app, tid, def, "after the move, under [de]", nested, listed("de"), "de")
	ucaCheck(t, super, app, tid, def, "after the move, nested U", nested, caller, "emea", "de", "berlin", "fr")

	// And back under de.
	sync("berlin moves back under de", emea, de, berlinUnderDE, fr, apac)
	ucaCheck(t, super, app, tid, def, "after the move back, de agent", deAgent, caller, "de", "berlin")
	ucaCheck(t, super, app, tid, def, "after the move back, fr agent", frAgent, caller, "fr")

	// de is tombstoned while berlin, below it, stays live. No snapshot makes
	// that state (one that drops de and keeps berlin is ErrBadSnapshot), so
	// it is written directly, as a backfill or a hand edit leaves it, and the
	// closure is rebuilt with group sync's own query, as nava_groupsync.
	exec(t, super, "DELETE FROM org_unit_members WHERE tenant_id = $1 AND org_unit_id = $2", tid, units["de"])
	exec(t, super, "UPDATE org_units SET deleted_at = now() WHERE tenant_id = $1 AND id = $2", tid, units["de"])
	if err := tenancy.InTenantTx(ctx, pool, tid, func(tx pgx.Tx) error {
		_, err := groupsync.RebuildClosureForTest(ctx, tx, tid)
		return err
	}); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if n := count(t, super, "SELECT count(*) FROM org_units WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL AND parent_id = $3", tid, units["berlin"], units["de"]); n != 1 {
		t.Fatalf("berlin is not a live child of the tombstoned de (%d rows)", n)
	}
	if n := count(t, super, "SELECT count(*) FROM org_unit_closure WHERE tenant_id = $1 AND (ancestor_id = ANY($2) OR descendant_id = ANY($2))",
		tid, []uuid.UUID{units["de"], units["berlin"]}); n != 0 {
		t.Errorf("%d closure rows name de or berlin after de's tombstone", n)
	}
	ucaCheck(t, super, app, tid, def, "de tombstoned, under [emea]", nested, listed("emea"), "emea", "fr")
	ucaCheck(t, super, app, tid, def, "de tombstoned, nested caller (now in emea only)", nested, caller, "emea", "fr")
	ucaCheck(t, super, app, tid, def, "de tombstoned, under [berlin]", nested, listed("berlin"))
	ucaCheck(t, super, app, tid, def, "de tombstoned, not under [emea]", nested, notUnder("emea"), "de", "berlin", "apac", "")
	ucaCheck(t, super, app, tid, def, "de tombstoned, de agent", deAgent, caller)
}

// A skewed district (SP-1c's seed shape): almost every record of emea sits in
// one unit, berlin, past the count cap. Every record agrees on every side,
// the page holds each once, and total is the cap with TotalCapped set; the
// units beside berlin agree with exact totals.
func TestUnderClosureAgreementSkewedDistrict(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)
	p := func(rest string) string { return path(tid, "/ou"+rest) }
	dir.set(tid, identity.DirectoryChanges{
		Users: []identity.DirectoryUser{orgUser("emea-agent", ""), orgUser("berlin-agent", ""), orgUser("fr-agent", "")},
		OrgUnits: []identity.DirectoryGroup{
			memberUnit("emea", "", p("/emea"), "emea-agent"),
			memberUnit("de", "emea", p("/emea/de")),
			memberUnit("berlin", "de", p("/emea/de/berlin"), "berlin-agent"),
			memberUnit("munich", "de", p("/emea/de/munich")),
			memberUnit("fr", "emea", p("/emea/fr"), "fr-agent"),
			memberUnit("apac", "", p("/apac")),
		},
	})
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	app := appPool(t, super)
	def := ucaSeed(t, super, tid, ucaRecords{"berlin": store.TotalCap + 150, "munich": 3, "de": 2, "fr": 4, "emea": 1, "apac": 40, "": 2})
	units := ucaUnits(t, super, tid)
	caller := ucaUnder(authz.Under{Caller: true})
	emeaAgent := ucaPrincipal(t, super, tid, "emea-agent")
	ucaCheck(t, super, app, tid, def, "emea agent, under caller (past the cap)", emeaAgent, caller, "emea", "de", "berlin", "munich", "fr")
	ucaCheck(t, super, app, tid, def, "berlin agent, under caller (past the cap)", ucaPrincipal(t, super, tid, "berlin-agent"), caller, "berlin")
	ucaCheck(t, super, app, tid, def, "fr agent, under caller", ucaPrincipal(t, super, tid, "fr-agent"), caller, "fr")
	ucaCheck(t, super, app, tid, def, "under [munich], beside the hot unit", emeaAgent, ucaUnder(authz.Under{Units: []uuid.UUID{units["munich"]}}), "munich")
	ucaCheck(t, super, app, tid, def, "under [apac], another district", emeaAgent, ucaUnder(authz.Under{Units: []uuid.UUID{units["apac"]}}), "apac")
}
