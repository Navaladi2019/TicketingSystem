//go:build integration

package groupsync_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// TICK-948 (ADR-010 D7, D-11-6): groupsync writes org_unit_members and one
// primary unit per user, in the same transaction as the tombstones.

func memberUnit(id, parent, p string, members ...string) identity.DirectoryGroup {
	g := unitG(id, parent, p, id)
	g.MemberKeycloakUserIDs = members
	return g
}

func orgUser(kc, primaryPath string) identity.DirectoryUser {
	return identity.DirectoryUser{KeycloakUserID: kc, Enabled: true, PrimaryOrgUnitPath: primaryPath}
}

// membership is "unit:user:primary" rows, sorted, by Keycloak ids.
func membership(t *testing.T, super *pgx.Conn, tid uuid.UUID) string {
	t.Helper()
	var s *string
	if err := super.QueryRow(context.Background(), `SELECT string_agg(o.keycloak_group_id || ':' || p.keycloak_user_id || ':' || m.is_primary::text, ',' ORDER BY o.keycloak_group_id, p.keycloak_user_id)
		FROM org_unit_members m
		JOIN org_units o ON o.tenant_id = m.tenant_id AND o.id = m.org_unit_id
		JOIN principals p ON p.tenant_id = m.tenant_id AND p.id = m.principal_id
		WHERE m.tenant_id = $1`, tid).Scan(&s); err != nil {
		t.Fatal(err)
	}
	if s == nil {
		return ""
	}
	return *s
}

func TestOrgMembersAndPrimaryWritten(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)
	a, b := path(tid, "/ou/a"), path(tid, "/ou/b")
	sync := func(users []identity.DirectoryUser, units ...identity.DirectoryGroup) {
		t.Helper()
		dir.set(tid, identity.DirectoryChanges{Users: users, OrgUnits: units})
		if r := once(t, s); r.Err != nil {
			t.Fatal(r.Err)
		}
	}
	// u1 in a and b, no attribute: no primary. u2 in a only: a is primary.
	// u3 in a, attribute names b, which u3 is not in: falls back to a.
	// u4 in a and b, attribute names b.
	units := func() []identity.DirectoryGroup {
		return []identity.DirectoryGroup{
			memberUnit("a", "", a, "u1", "u2", "u3", "u4"),
			memberUnit("b", "", b, "u1", "u4"),
		}
	}
	sync([]identity.DirectoryUser{orgUser("u1", ""), orgUser("u2", ""), orgUser("u3", b), orgUser("u4", b), orgUser("nobody", "")}, units()...)
	want := "a:u1:false,a:u2:true,a:u3:true,a:u4:false,b:u1:false,b:u4:true"
	if got := membership(t, super, tid); got != want {
		t.Fatalf("membership =\n %s\nwant\n %s", got, want)
	}

	// Changing the attribute moves is_primary, and the one-primary index holds.
	sync([]identity.DirectoryUser{orgUser("u1", a), orgUser("u2", ""), orgUser("u3", b), orgUser("u4", a), orgUser("nobody", "")}, units()...)
	want = "a:u1:true,a:u2:true,a:u3:true,a:u4:true,b:u1:false,b:u4:false"
	if got := membership(t, super, tid); got != want {
		t.Fatalf("after the attribute changed =\n %s\nwant\n %s", got, want)
	}
	// An attribute naming a unit that does not exist leaves u1 in two units: none.
	sync([]identity.DirectoryUser{orgUser("u1", path(tid, "/ou/zzz")), orgUser("u2", ""), orgUser("u3", ""), orgUser("u4", a)}, units()...)
	want = "a:u1:false,a:u2:true,a:u3:true,a:u4:true,b:u1:false,b:u4:false"
	if got := membership(t, super, tid); got != want {
		t.Fatalf("after a dangling attribute =\n %s\nwant\n %s", got, want)
	}
	// Leaving a unit removes the row; u4 is left in a alone and becomes primary.
	sync([]identity.DirectoryUser{orgUser("u1", ""), orgUser("u2", ""), orgUser("u3", ""), orgUser("u4", "")},
		memberUnit("a", "", a, "u1", "u2", "u3", "u4"), memberUnit("b", "", b, "u1"))
	want = "a:u1:false,a:u2:true,a:u3:true,a:u4:true,b:u1:false"
	if got := membership(t, super, tid); got != want {
		t.Fatalf("after u4 left b =\n %s\nwant\n %s", got, want)
	}
}

// A unit that still has members is removed from the directory: the sync
// succeeds (the memberships go before the tombstone), the unit is tombstoned,
// the record keeps its org_unit_id, descendants are tombstoned too, and the
// unit comes back on the same id.
func TestOrgUnitWithMembersRemovedAndRevived(t *testing.T) {
	super, pool := cluster(t)
	ctx := context.Background()
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)
	users := []identity.DirectoryUser{orgUser("u1", ""), orgUser("u2", "")}
	p, c := path(tid, "/ou/p"), path(tid, "/ou/p/c")
	snap := func(units ...identity.DirectoryGroup) groupsync.Result {
		t.Helper()
		dir.set(tid, identity.DirectoryChanges{Users: users, OrgUnits: units})
		return once(t, s)
	}
	if r := snap(memberUnit("p", "", p, "u1"), memberUnit("c", "p", c, "u1", "u2"), memberUnit("q", "", path(tid, "/ou/q"), "u2")); r.Err != nil {
		t.Fatal(r.Err)
	}
	var cID uuid.UUID
	if err := super.QueryRow(ctx, "SELECT id FROM org_units WHERE tenant_id = $1 AND keycloak_group_id = 'c'", tid).Scan(&cID); err != nil {
		t.Fatal(err)
	}
	exec(t, super, `WITH r AS (INSERT INTO records (tenant_id, id, org_unit_id) VALUES ($1, $2, $3) RETURNING tenant_id, id, version)
		INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
		SELECT tenant_id, gen_random_uuid(), 'record', 'create', 'record', id, version, 'system', 'system' FROM r`,
		tid, uuid.New(), cID)

	// p vanishes with its child c; both have members.
	if r := snap(memberUnit("q", "", path(tid, "/ou/q"), "u2")); r.Err != nil {
		t.Fatalf("removing units that have members: %v", r.Err)
	}
	if n := count(t, super, "SELECT count(*) FROM org_units WHERE tenant_id = $1 AND deleted_at IS NOT NULL AND keycloak_group_id IN ('p','c')", tid); n != 2 {
		t.Errorf("%d of p, c tombstoned, want 2", n)
	}
	if got, want := membership(t, super, tid), "q:u2:true"; got != want {
		t.Errorf("membership after the removal = %s, want %s", got, want)
	}
	if n := count(t, super, "SELECT count(*) FROM records WHERE tenant_id = $1 AND org_unit_id = $2", tid, cID); n != 1 {
		t.Errorf("the record's org_unit_id was changed or the record lost (%d rows name c)", n)
	}

	// They come back on the same ids, and their members with them.
	if r := snap(memberUnit("p", "", p, "u1"), memberUnit("c", "p", c, "u2"), memberUnit("q", "", path(tid, "/ou/q"), "u2")); r.Err != nil {
		t.Fatal(r.Err)
	}
	var back uuid.UUID
	if err := super.QueryRow(ctx, "SELECT id FROM org_units WHERE tenant_id = $1 AND keycloak_group_id = 'c' AND deleted_at IS NULL", tid).Scan(&back); err != nil || back != cID {
		t.Fatalf("c revived as %v (err %v), want %s", back, err, cID)
	}
	if got, want := membership(t, super, tid), "c:u2:false,p:u1:true,q:u2:false"; got != want {
		t.Errorf("membership after the revive = %s, want %s", got, want)
	}
}

// A failure mid-apply rolls back the membership writes and the tombstones
// together: here the tombstone and the membership removal are sent, then the
// chain check refuses a unit that came back too deep.
func TestOrgMembershipAndTombstoneRollBackTogether(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir) // MaxOrgUnitDepth 8
	users := []identity.DirectoryUser{orgUser("u1", ""), orgUser("u2", "")}
	dir.set(tid, identity.DirectoryChanges{Users: users, OrgUnits: []identity.DirectoryGroup{
		memberUnit("x", "", path(tid, "/ou/x"), "u1"), memberUnit("y", "", path(tid, "/ou/y"), "u2")}})
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	before := membership(t, super, tid)

	// x vanishes (tombstone + membership delete), u2 moves to z (insert), and
	// a chain 9 deep fails the depth check last.
	units := []identity.DirectoryGroup{memberUnit("y", "", path(tid, "/ou/y")), memberUnit("z", "", path(tid, "/ou/z"), "u1", "u2")}
	parent := ""
	for i := 1; i <= 9; i++ {
		k := "d" + string(rune('0'+i))
		units = append(units, unitG(k, parent, path(tid, "/ou/"+k), k))
		parent = k
	}
	dir.set(tid, identity.DirectoryChanges{Users: users, OrgUnits: units})
	r := once(t, s)
	if !errors.Is(r.Err, groupsync.ErrOrgUnitDepth) {
		t.Fatalf("err = %v, want ErrOrgUnitDepth", r.Err)
	}
	if got := membership(t, super, tid); got != before {
		t.Errorf("membership after the rollback = %s, want %s", got, before)
	}
	if n := count(t, super, "SELECT count(*) FROM org_units WHERE tenant_id = $1 AND (deleted_at IS NOT NULL OR keycloak_group_id IN ('z', 'd1'))", tid); n != 0 {
		t.Errorf("%d org units kept a tombstone or a new row after the rollback", n)
	}
}

// groupsync_org_subtree_max on a seeded tree: r has children a (with child
// a1) and b; u1 is in r, so r's subtree of 4 is the largest u1 reaches.
// Only users' units count: with u1 in a alone the value is 2.
func TestOrgSubtreeMaxReported(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	reader := metric.NewManualReader(metric.WithTemporalitySelector(func(metric.InstrumentKind) metricdata.Temporality { return metricdata.DeltaTemporality }))
	s, err := groupsync.New(pool, dir, metric.NewMeterProvider(metric.WithReader(reader)), slog.New(slog.DiscardHandler), groupsync.Options{MaxOrgUnitDepth: 8})
	if err != nil {
		t.Fatal(err)
	}
	last := func() int64 {
		t.Helper()
		var rm metricdata.ResourceMetrics
		if err := reader.Collect(context.Background(), &rm); err != nil {
			t.Fatal(err)
		}
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name != "groupsync_org_subtree_max" {
					continue
				}
				h := m.Data.(metricdata.Histogram[int64])
				if len(h.DataPoints) != 1 {
					t.Fatalf("%d data points, want 1 (no tenant attribute)", len(h.DataPoints))
				}
				if h.DataPoints[0].Attributes.Len() != 0 {
					t.Error("the metric carries attributes; a tenant id must never be one")
				}
				v, _ := h.DataPoints[0].Max.Value()
				return v
			}
		}
		t.Fatal("groupsync_org_subtree_max not reported")
		return 0
	}
	tree := func(rMembers, aMembers []string) []identity.DirectoryGroup {
		return []identity.DirectoryGroup{
			memberUnit("r", "", path(tid, "/ou/r"), rMembers...),
			memberUnit("a", "r", path(tid, "/ou/r/a"), aMembers...),
			memberUnit("a1", "a", path(tid, "/ou/r/a/a1")),
			memberUnit("b", "r", path(tid, "/ou/r/b")),
		}
	}
	users := []identity.DirectoryUser{orgUser("u1", "")}
	dir.set(tid, identity.DirectoryChanges{Users: users, OrgUnits: tree([]string{"u1"}, nil)})
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	if got := last(); got != 4 {
		t.Errorf("subtree max with u1 in r = %d, want 4 (r, a, a1, b)", got)
	}
	dir.set(tid, identity.DirectoryChanges{Users: users, OrgUnits: tree(nil, []string{"u1"})})
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	if got := last(); got != 2 {
		t.Errorf("subtree max with u1 in a = %d, want 2 (a, a1)", got)
	}
}
