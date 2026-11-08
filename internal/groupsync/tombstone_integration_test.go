//go:build integration

package groupsync_test

import (
	"bytes"
	"context"
	"errors"
	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"go.opentelemetry.io/otel/metric/noop"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

func tsUsers() []identity.DirectoryUser {
	return []identity.DirectoryUser{
		{KeycloakUserID: "u1", Email: "a@example.test", Enabled: true},
		{KeycloakUserID: "u2", Email: "b@example.test", Enabled: true},
	}
}

func tsGroups(tid uuid.UUID, withAdm, withAud bool) []identity.DirectoryGroup {
	gs := []identity.DirectoryGroup{
		{KeycloakGroupID: "g1", Path: path(tid, "/agents"), Name: "agents", MemberKeycloakUserIDs: []string{"u1", "u2"}},
	}
	if withAdm {
		gs = append(gs, identity.DirectoryGroup{KeycloakGroupID: "gadm", Path: path(tid, "/admins"), Name: "admins", MemberKeycloakUserIDs: []string{"u1"}})
	}
	if withAud {
		gs = append(gs, identity.DirectoryGroup{KeycloakGroupID: "gaud", Path: path(tid, "/auditors"), Name: "auditors", MemberKeycloakUserIDs: []string{"u2"}})
	}
	return gs
}

func gid(t *testing.T, super *pgx.Conn, tid uuid.UUID, kc string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := super.QueryRow(context.Background(), "SELECT id FROM groups WHERE tenant_id = $1 AND keycloak_group_id = $2", tid, kc).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func exec(t *testing.T, super *pgx.Conn, sql string, args ...any) {
	t.Helper()
	if _, err := super.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

// seedPointers records gadm / gaud as the tenant's admins / auditors groups.
func seedPointers(t *testing.T, super *pgx.Conn, tid uuid.UUID) {
	t.Helper()
	exec(t, super, "INSERT INTO tenant_admin_group (tenant_id, group_id) VALUES ($1, $2)", tid, gid(t, super, tid, "gadm"))
	exec(t, super, "INSERT INTO tenant_auditor_group (tenant_id, group_id) VALUES ($1, $2)", tid, gid(t, super, tid, "gaud"))
}

// CI-E5-1, CI-E5-4, CI-E5-5b, acceptance 2, and TICK-538's risk check
// (acceptance 1): the migrations revoke DELETE on groups from nava_groupsync
// (step ii-a), so any DELETE FROM groups would fail the sync, and a full sync
// that drops a group must still commit. A record names the dropped admins
// group; RESTRICT must not fire and the record must not change.
func TestTombstoneWithoutDeleteOnGroups(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	var logs bytes.Buffer
	s, err := groupsync.New(pool, dir, noop.NewMeterProvider(), slog.New(slog.NewTextHandler(&logs, nil)), groupsync.Options{MaxOrgUnitDepth: 8})
	if err != nil {
		t.Fatal(err)
	}

	dir.set(tid, identity.DirectoryChanges{Users: tsUsers(), Groups: tsGroups(tid, true, true)})
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	seedPointers(t, super, tid)
	// The record and its audit entry commit together, as the records
	// commit-check trigger requires (P5-13c, TICK-530).
	exec(t, super, `WITH r AS (INSERT INTO records (tenant_id, id, assignment_group_id) VALUES ($1, $2, $3) RETURNING tenant_id, id, version)
		INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
		SELECT tenant_id, gen_random_uuid(), 'record', 'create', 'record', id, version, 'system', 'system' FROM r`,
		tid, uuid.New(), gid(t, super, tid, "gadm"))
	// CI-E5-1 is the attached run (TICK-530): the commit-check trigger is on
	// records, so a group-sync write that reached a record would fail at COMMIT.
	if n := count(t, super, "SELECT count(*) FROM pg_trigger WHERE tgrelid = 'records'::regclass AND tgname = 'records_audit_commit_check' AND tgenabled = 'O'"); n != 1 {
		t.Fatal("records_audit_commit_check is not attached to records (P5-13c)")
	}
	if n := count(t, super, "SELECT count(*) FROM (SELECT 1 WHERE has_table_privilege('nava_groupsync', 'public.groups', 'DELETE')) x"); n != 0 {
		t.Fatal("nava_groupsync still holds DELETE on groups after the migrations (TICK-538 step ii-a)")
	}

	recState := func() string {
		var s string
		if err := super.QueryRow(context.Background(), "SELECT string_agg(id || ':' || version || ':' || coalesce(assignment_group_id::text, 'null') || ':' || xmin::text, ',') FROM records WHERE tenant_id = $1", tid).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	before := recState()
	adm := gid(t, super, tid, "gadm")

	dir.set(tid, identity.DirectoryChanges{Users: tsUsers(), Groups: tsGroups(tid, false, false)})
	r := once(t, s)
	if r.Err != nil {
		t.Fatalf("sync must commit with DELETE on groups revoked: %v", r.Err)
	}
	for _, kc := range []string{"gadm", "gaud"} {
		if n := count(t, super, "SELECT count(*) FROM groups WHERE tenant_id = $1 AND keycloak_group_id = $2 AND deleted_at IS NOT NULL", tid, kc); n != 1 {
			t.Errorf("%s not tombstoned", kc)
		}
		if n := count(t, super, "SELECT count(*) FROM group_members WHERE tenant_id = $1 AND group_id = $2", tid, gid(t, super, tid, kc)); n != 0 {
			t.Errorf("%s keeps members", kc)
		}
	}
	if n := count(t, super, "SELECT count(*) FROM tenant_admin_group WHERE tenant_id = $1", tid); n != 0 {
		t.Errorf("admin pointer remains")
	}
	if n := count(t, super, "SELECT count(*) FROM tenant_auditor_group WHERE tenant_id = $1", tid); n != 0 {
		t.Errorf("auditor pointer remains")
	}
	if !strings.Contains(logs.String(), "admins group was deleted") {
		t.Errorf("AdminGroupDeleted not reported")
	}
	if after := recState(); after != before {
		t.Errorf("records changed: %s -> %s", before, after)
	}

	// The groups come back: same ids, members restored, admin check still denies.
	dir.set(tid, identity.DirectoryChanges{Users: tsUsers(), Groups: tsGroups(tid, true, true)})
	if r := once(t, s); r.Err != nil {
		t.Fatalf("revive: %v", r.Err)
	}
	if got := gid(t, super, tid, "gadm"); got != adm {
		t.Errorf("revived with a new id %s, want %s", got, adm)
	}
	if n := count(t, super, "SELECT count(*) FROM groups WHERE tenant_id = $1 AND deleted_at IS NULL", tid); n != 3 {
		t.Errorf("live groups = %d, want 3", n)
	}
	if n := count(t, super, "SELECT count(*) FROM group_members WHERE tenant_id = $1", tid); n != 4 {
		t.Errorf("memberships = %d, want 4", n)
	}
	if after := recState(); after != before {
		t.Errorf("records changed on revive")
	}
	p := identity.Principal{TenantID: tid, Groups: []uuid.UUID{adm}}
	var got error
	if err := tenancy.InTenantTx(context.Background(), pool, tid, func(tx pgx.Tx) error {
		got = authz.RequireTenantAdminTx(context.Background(), tx, p)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(got, authz.ErrForbidden) {
		t.Errorf("RequireTenantAdminTx after revive = %v, want ErrForbidden", got)
	}
}

// Dropping the auditors group alone commits and does not report an admins loss.
func TestDropAuditorsGroupOnly(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)
	dir.set(tid, identity.DirectoryChanges{Users: tsUsers(), Groups: tsGroups(tid, true, true)})
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	seedPointers(t, super, tid)
	dir.set(tid, identity.DirectoryChanges{Users: tsUsers(), Groups: tsGroups(tid, true, false)})
	r := once(t, s)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if n := count(t, super, "SELECT count(*) FROM tenant_auditor_group WHERE tenant_id = $1", tid); n != 0 {
		t.Errorf("auditor pointer remains")
	}
	if n := count(t, super, "SELECT count(*) FROM tenant_admin_group WHERE tenant_id = $1", tid); n != 1 {
		t.Errorf("admin pointer lost")
	}
}

// Dropping groups in tenant A leaves tenant B untouched.
func TestTombstoneIsolatedPerTenant(t *testing.T) {
	super, pool := cluster(t)
	a, b := uuid.New(), uuid.New()
	seedTenant(t, super, a, 1)
	seedTenant(t, super, b, 2)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)
	cycle := func() {
		rs, err := s.Once(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range rs {
			if r.Err != nil {
				t.Fatal(r.Err)
			}
		}
	}
	for _, id := range []uuid.UUID{a, b} {
		dir.set(id, identity.DirectoryChanges{Users: tsUsers(), Groups: tsGroups(id, true, true)})
	}
	cycle()
	seedPointers(t, super, a)
	seedPointers(t, super, b)
	dir.set(a, identity.DirectoryChanges{Users: tsUsers(), Groups: tsGroups(a, false, false)})
	cycle()
	if n := count(t, super, "SELECT count(*) FROM groups WHERE tenant_id = $1 AND deleted_at IS NULL", b); n != 3 {
		t.Errorf("tenant B live groups = %d, want 3", n)
	}
	if n := count(t, super, "SELECT count(*) FROM group_members WHERE tenant_id = $1", b); n != 4 {
		t.Errorf("tenant B memberships = %d, want 4", n)
	}
	if n := count(t, super, "SELECT count(*) FROM tenant_admin_group WHERE tenant_id = $1", b) + count(t, super, "SELECT count(*) FROM tenant_auditor_group WHERE tenant_id = $1", b); n != 2 {
		t.Errorf("tenant B pointers = %d, want 2", n)
	}
	if n := count(t, super, "SELECT count(*) FROM tenant_admin_group WHERE tenant_id = $1", a); n != 0 {
		t.Errorf("tenant A admin pointer remains")
	}
}
