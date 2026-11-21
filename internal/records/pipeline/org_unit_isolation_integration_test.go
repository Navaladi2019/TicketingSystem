//go:build integration

package pipeline_test

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

// TICK-952 (S31 P11-19) adversarial isolation tests, Security & Multi-Tenancy
// Reviewer. The attacker holds rows no runtime role can write: the superuser
// with session_replication_role = replica, which skips every foreign key and
// trigger, forges org-unit rows across the tenant wall. That is strictly more
// than the owner can do (the owner is held by FORCE RLS and cannot disable
// the RI triggers), so a pass here covers attack 4's "forged by the owner
// role". What must hold is that the closure, the under leaf and the
// pipeline's primary-unit lookup never cross the wall on their own
// predicates, whatever the rows say. The closure is rebuilt from the forged
// rows (org_unit_closure_rebuild_all walks each tenant under its own RLS), so
// the forged parents and memberships are what the under leaf reads.

// forge runs stmts as the superuser with foreign keys and triggers off.
func forge(t *testing.T, e env, stmts ...func(tx pgx.Tx) error) {
	t.Helper()
	ctx := context.Background()
	tx, err := e.super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SET LOCAL session_replication_role = replica"); err != nil {
		t.Fatal(err)
	}
	for _, s := range stmts {
		if err := s(tx); err != nil {
			t.Fatalf("forge: %v", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func execf(sql string, args ...any) func(tx pgx.Tx) error {
	return func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), sql, args...)
		return err
	}
}

// forgedUnderWorld is underWorld (tenant f with R ── A ── A1, R ── T† ── T1,
// R ── L ── Lx†; tenant o with live unit X) plus forged rows:
//
//   - f's membership of u1 in X, o's unit (attack 4): the caller seed names a
//     foreign unit;
//   - f's membership of u1 in tombstoned T (attack 3): the row the live FK
//     forbids, as if the sync had tombstoned T before removing it;
//   - Y, a unit of o whose parent is f's unit A (attack 4): the walk down
//     from A must not cross into o;
//   - Z, a unit of f whose parent is o's unit X: no walk reads o's tree, Z is
//     no descendant of X in f, and since no walk from f's live roots reaches
//     Z, the closure has no row for it: Z is in no region, not even its own
//     (fail closed, as under a tombstone; Security & Multi-Tenancy Reviewer,
//     TICK-1034);
//   - records of f in X and in Y (org_unit_id forged past the FK), named
//     "X" and "Y".
func forgedUnderWorld(t *testing.T, e env) (underWorld, uuid.UUID) {
	t.Helper()
	w := seedUnderWorld(t, e)
	var other uuid.UUID
	if err := e.super.QueryRow(context.Background(), `SELECT tenant_id FROM org_units WHERE id = $1`, w.unit["X"]).Scan(&other); err != nil {
		t.Fatal(err)
	}
	w.unit["Y"], w.unit["Z"] = uuid.New(), uuid.New()
	ctx := context.Background()
	sys := identity.Principal{TenantID: w.f.id}
	for _, name := range []string{"X", "Y", "Z"} {
		res, err := write(ctx, e, w.p, w.f.id, pipelineCreate(sys, map[string]any{"title": "forged-" + name}))
		if err != nil {
			t.Fatalf("seed record %s: %v", name, err)
		}
		w.record[name] = res.RecordID
		w.records = append(w.records, res.RecordID)
	}
	forge(t, e,
		execf(`INSERT INTO org_unit_members (tenant_id, org_unit_id, principal_id) VALUES ($1, $2, $3)`, w.f.id, w.unit["X"], w.u1),
		execf(`INSERT INTO org_unit_members (tenant_id, org_unit_id, principal_id) VALUES ($1, $2, $3)`, w.f.id, w.unit["T"], w.u1),
		execf(`INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name) VALUES ($1, $2, $3, 'kc-forged-y', 'Y')`, other, w.unit["Y"], w.unit["A"]),
		execf(`INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name) VALUES ($1, $2, $3, 'kc-forged-z', 'Z')`, w.f.id, w.unit["Z"], w.unit["X"]),
		execf(`UPDATE records SET org_unit_id = $3 WHERE tenant_id = $1 AND id = $2`, w.f.id, w.record["X"], w.unit["X"]),
		execf(`UPDATE records SET org_unit_id = $3 WHERE tenant_id = $1 AND id = $2`, w.f.id, w.record["Y"], w.unit["Y"]),
		execf(`UPDATE records SET org_unit_id = $3 WHERE tenant_id = $1 AND id = $2`, w.f.id, w.record["Z"], w.unit["Z"]),
	)
	rebuildClosure(t, e)
	return w, other
}

// Attacks 3 and 4: forged membership and parent rows never put a unit of
// another tenant, or a tombstoned unit or its subtree, into set S, in SQL, in
// Go or in the branch split; the recursive query filters by tenant at every
// step, on top of RLS.
func TestUnderSetIgnoresForgedCrossTenantAndTombstonedRows(t *testing.T) {
	e := migrated(t)
	w, _ := forgedUnderWorld(t, e)
	of := func(names ...string) string {
		s := `{"under":{"of":[`
		for i, n := range names {
			if i > 0 {
				s += ","
			}
			s += `"` + w.unit[n].String() + `"`
		}
		return s + `]}}`
	}
	caller := `{"under":{"of":"caller"}}`
	u1 := identity.Principal{TenantID: w.f.id, PrincipalID: w.u1}
	requester := `{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}`
	for _, tc := range []struct {
		name, when string
		want       []string
	}{
		// u1's real units are A and L. The forged X and T memberships add
		// nothing: X is not f's, T is tombstoned (and so is T1 below it).
		{"caller: forged memberships in X and tombstoned T add nothing", caller, []string{"A", "A1", "L"}},
		// The walk down from A does not reach o's Y, though Y's parent is A.
		{"listed A: o's child Y is not in S", of("A"), []string{"A", "A1"}},
		{"listed R: the whole live tree, nothing of o's", of("R"), []string{"A", "A1", "L", "R"}},
		// X is o's unit: as a listed root it is empty; Z, f's unit whose
		// forged parent is X, is not below it in f.
		{"listed X: empty, Z is not its descendant in f", of("X"), []string{}},
		// Y is o's unit: empty as a listed root too.
		{"listed Y: empty", of("Y"), []string{}},
		// Z is f's live unit, but its forged parent is o's X: no walk from f's live roots reaches it, so the closure has no (Z, Z) row and Z is in no region (fail closed, as under a tombstone).
		{"listed Z: orphaned by a foreign parent, outside every region", of("Z"), []string{}},
		{"not under listed Z: the orphan reads as outside", `{"not":` + of("Z") + `}`, []string{"A", "A1", "L", "Lx", "R", "T", "T1", "X", "Y", "Z", "none"}},
		{"listed T, tombstoned with a forged member: empty", of("T"), []string{}},
		// not under reads the forged records as outside the region; it never
		// errors or voids because of them.
		{"not under caller", `{"not":` + caller + `}`, []string{"Lx", "R", "T", "T1", "X", "Y", "Z", "none"}},
		{"under caller or requester, split", `{"any":[` + caller + `,` + requester + `]}`, []string{"A", "A1", "L"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bySQL, byGo, byBranches := w.rows(t, e, underDoc(t, tc.when), u1)
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
}

// Attack 4, the binding: S resolved in tenant f's transaction for a principal
// that claims tenant o (same principal id) is never used for o. Every policy,
// not under included, grants nothing: the scope is bound to f, the planner
// sees o, and the policy is voided rather than read as an empty S.
func TestUnderSetBoundInOneTenantGrantsNothingToAnother(t *testing.T) {
	e := migrated(t)
	w, other := forgedUnderWorld(t, e)
	intruder := identity.Principal{TenantID: other, PrincipalID: w.u1}
	for _, when := range []string{
		`{"under":{"of":"caller"}}`,
		`{"not":{"under":{"of":"caller"}}}`,
		`{"under":{"of":["` + w.unit["R"].String() + `"]}}`,
		`{"not":{"under":{"of":["` + w.unit["X"].String() + `"]}}}`,
	} {
		bySQL, byGo, _ := w.rows(t, e, underDoc(t, when), intruder)
		if len(bySQL) != 0 || len(byGo) != 0 {
			t.Errorf("%s bound in f, evaluated for o: SQL admits %v, Allows %v; want nothing", when, bySQL, byGo)
		}
	}
}

// Attack 2, the pipeline's lookup: a forged primary membership of the
// requester in another tenant's unit never becomes the record's org_unit_id.
// The create fails closed (the tenant-carrying FK refuses it), writes
// nothing, and no record of f names g's unit. A workflow update_record (and a
// user update) naming g's unit as org_unit_id is refused before any write.
func TestRecordNeverGetsAnotherTenantsOrgUnit(t *testing.T) {
	e := migrated(t)
	f, g := seedTenant(t, e), seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()
	foreign := seedOrgUnit(t, e, g.id, "Foreign")
	res := mustCreate(t, e, p, f, map[string]any{"title": "before the forgery"})

	forge(t, e, execf(`INSERT INTO org_unit_members (tenant_id, org_unit_id, principal_id, is_primary) VALUES ($1, $2, $3, true)`,
		f.id, foreign, f.principal.PrincipalID))
	records := func() int { return count(t, e.super, `SELECT count(*) FROM records WHERE tenant_id = $1`, f.id) }
	before := records()
	_, err := write(ctx, e, p, f.id, createCmd(f, map[string]any{"title": "lands in g's unit?"}))
	if pe, ok := errors.AsType[*pgconn.PgError](err); !ok || pe.Code != "23503" || pe.ConstraintName != "records_org_unit_fkey" {
		t.Errorf("a create whose requester's forged primary unit is another tenant's: err = %v, want 23503 on records_org_unit_fkey", err)
	}
	if n := records(); n != before {
		t.Errorf("records of f: %d before, %d after the refused create", before, n)
	}

	var version int64
	if err := e.super.QueryRow(ctx, `SELECT version FROM records WHERE id = $1`, res.RecordID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	for name, cmd := range map[string]pipeline.Command{
		"user update": updateCmd(f, res.RecordID, version, map[string]any{"org_unit_id": foreign.String()}),
		"workflow update_record": {Op: pipeline.OpUpdate, EntityKey: entityKey, RecordID: res.RecordID, ExpectedVersion: version,
			Data: map[string]any{"org_unit_id": foreign.String()}, Actor: identity.Principal{TenantID: f.id},
			Source: pipeline.SourceWorkflow, Causation: workflowCausation()},
	} {
		_, err := write(ctx, e, p, f.id, cmd)
		var nw *authz.ErrFieldNotWritable
		if !errors.As(err, &nw) {
			t.Errorf("%s naming g's unit: err = %v, want ErrFieldNotWritable", name, err)
		}
	}
	if n := count(t, e.super, `SELECT count(*) FROM records WHERE tenant_id = $1 AND org_unit_id = $2`, f.id, foreign); n != 0 {
		t.Errorf("%d records of f name g's unit", n)
	}
}

// orgDir is an identity.DirectoryAdmin whose only method in use is
// ChangesSince: one fixed snapshot per tenant, empty for the rest.
type orgDir struct {
	identity.DirectoryAdmin
	snaps map[uuid.UUID]identity.DirectoryChanges
}

func (d orgDir) ChangesSince(_ context.Context, id uuid.UUID, _ string) (identity.DirectoryChanges, string, error) {
	return d.snaps[id], "", nil
}

// groupsyncPool is a nava_groupsync pool on the platform database, built as
// cmd/nava-groupsync builds its own (QueryExecModeExec).
func groupsyncPool(t *testing.T, e env) *pgxpool.Pool {
	t.Helper()
	u, err := url.Parse(os.Getenv("NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	pw, _ := testPassword("nava_groupsync")
	u.User = url.UserPassword("nava_groupsync", pw)
	u.Path = "/" + e.platform
	cfg, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// Attack 7 end to end (QA review item 3): the real group sync applies tenant
// f's directory, where the requester's nava_primary_org_unit names a path
// that exists only in tenant g and the requester is in two units of f; then
// the real pipeline creates the requester's ticket. Its org_unit_id is NULL.
// The same create with the attribute naming one of f's units stores that
// unit, so the NULL is the attack's doing, not a broken lookup.
func TestCreateAfterSyncWithForeignPrimaryPathLeavesOrgUnitNull(t *testing.T) {
	e := migrated(t)
	f, g := seedTenant(t, e), seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()
	for i, tn := range []tenantFixture{f, g} {
		mustExec(t, e.super, `INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id) VALUES ($1, 'https://kc/realms/nava', $2)`,
			tn.id, "org-tick952-"+string(rune('a'+i)))
	}
	pathOf := func(tn tenantFixture, rest string) string { return "/t/" + tn.id.String() + rest }
	user := func(tn tenantFixture, primary string) identity.DirectoryUser {
		return identity.DirectoryUser{KeycloakUserID: "kc-" + tn.principal.PrincipalID.String(), Enabled: true, PrimaryOrgUnitPath: primary}
	}
	unit := func(tn tenantFixture, kc, rest string) identity.DirectoryGroup {
		return identity.DirectoryGroup{KeycloakGroupID: kc, Path: pathOf(tn, rest), Name: kc,
			MemberKeycloakUserIDs: []string{"kc-" + tn.principal.PrincipalID.String()}}
	}
	sync := func(fPrimary string) {
		t.Helper()
		dir := orgDir{snaps: map[uuid.UUID]identity.DirectoryChanges{
			f.id: {Users: []identity.DirectoryUser{user(f, fPrimary)},
				OrgUnits: []identity.DirectoryGroup{unit(f, "ou-f-1", "/ou/one"), unit(f, "ou-f-2", "/ou/two")}},
			g.id: {Users: []identity.DirectoryUser{user(g, pathOf(g, "/ou/only-g"))},
				OrgUnits: []identity.DirectoryGroup{unit(g, "ou-only-g", "/ou/only-g")}},
		}}
		s, err := groupsync.New(groupsyncPool(t, e), dir, noop.NewMeterProvider(), slog.New(slog.DiscardHandler), groupsync.Options{MaxOrgUnitDepth: 8})
		if err != nil {
			t.Fatal(err)
		}
		rs, err := s.Once(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range rs {
			if r.Err != nil {
				t.Fatalf("sync tenant %s: %v", r.TenantID, r.Err)
			}
		}
	}

	sync(pathOf(g, "/ou/only-g"))
	res := mustCreate(t, e, p, f, map[string]any{"title": "primary path only in g"})
	if got := recordUnit(t, e, res.RecordID); got != nil {
		t.Errorf("org_unit_id = %s, want NULL: the primary attribute named a path only tenant g has", got)
	}

	// Control: the attribute names f's own unit, and the same create stores it.
	sync(pathOf(f, "/ou/two"))
	var want uuid.UUID
	if err := e.super.QueryRow(ctx, `SELECT id FROM org_units WHERE tenant_id = $1 AND keycloak_group_id = 'ou-f-2'`, f.id).Scan(&want); err != nil {
		t.Fatal(err)
	}
	res = mustCreate(t, e, p, f, map[string]any{"title": "primary path in f"})
	if got := recordUnit(t, e, res.RecordID); got == nil || *got != want {
		t.Errorf("control: org_unit_id = %v, want f's unit %s", got, want)
	}
}
