//go:build integration

// Adversarial tenant-isolation tests for T25 (TICK-78), from the security
// review of PR #113. Threat: tenant A's directory snapshot is hostile (it
// reuses tenant B's Keycloak ids, names B's org units as parents, or is
// empty so that every row "should go"). None of it may read, change or
// delete a row of tenant B, and A's lock or failure must not stop B's sync.

package groupsync_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// tenantRows is every identity row of one tenant, as text, read as the
// superuser so RLS does not hide anything.
func tenantRows(t *testing.T, super *pgx.Conn, tid uuid.UUID) string {
	t.Helper()
	var s string
	err := super.QueryRow(context.Background(), `SELECT concat_ws(' | ',
		(SELECT string_agg(id || ':' || keycloak_user_id || ':' || coalesce(email, '') || ':' || enabled || ':' || federated_links::text, ',' ORDER BY id) FROM principals WHERE tenant_id = $1),
		(SELECT string_agg(id || ':' || keycloak_group_id || ':' || path, ',' ORDER BY id) FROM groups WHERE tenant_id = $1),
		(SELECT string_agg(group_id || ':' || principal_id, ',' ORDER BY group_id, principal_id) FROM group_members WHERE tenant_id = $1),
		(SELECT string_agg(id || ':' || keycloak_group_id || ':' || coalesce(parent_id::text, '-') || ':' || name, ',' ORDER BY id) FROM org_units WHERE tenant_id = $1))`,
		tid).Scan(&s)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// victimSnapshot is tenant B's directory. Tenant A's hostile snapshots reuse
// its Keycloak ids.
func victimSnapshot(b uuid.UUID) identity.DirectoryChanges {
	return identity.DirectoryChanges{
		Users: []identity.DirectoryUser{
			{KeycloakUserID: "u-shared", Email: "shared@example.test", Enabled: true},
			{KeycloakUserID: "u-b", Email: "b@example.test", Enabled: true},
		},
		Groups: []identity.DirectoryGroup{
			{KeycloakGroupID: "g-shared", Path: path(b, "/agents"), Name: "agents", MemberKeycloakUserIDs: []string{"u-shared", "u-b"}},
		},
		OrgUnits: []identity.DirectoryGroup{
			unit("ou-b-root", "", path(b, "/ou/emea")),
			unit("ou-b-child", "ou-b-root", path(b, "/ou/emea/sales")),
		},
	}
}

// twoTenants returns two tenant ids with a sorting first. Once visits tenants
// ORDER BY tenant_id, so "A fails or is locked, B still syncs" only proves
// something when A is visited before B. Postgres orders uuids as their
// lowercase hex strings.
func twoTenants() (a, b uuid.UUID) {
	a, b = uuid.New(), uuid.New()
	if a.String() > b.String() {
		a, b = b, a
	}
	return a, b
}

func syncAll(t *testing.T, s *groupsync.Syncer) map[uuid.UUID]groupsync.Result {
	t.Helper()
	rs, err := s.Once(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	out := map[uuid.UUID]groupsync.Result{}
	for _, r := range rs {
		out[r.TenantID] = r
	}
	return out
}

// Tenant A's snapshot reuses B's Keycloak user and group ids and B's paths,
// then turns empty. B's rows never change, and A's rows are A's own.
func TestHostileSnapshotNeverTouchesAnotherTenant(t *testing.T) {
	super, pool := cluster(t)
	a, b := twoTenants()
	seedTenant(t, super, a, 1)
	seedTenant(t, super, b, 2)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)

	dir.set(b, victimSnapshot(b))
	dir.set(a, identity.DirectoryChanges{})
	for id, r := range syncAll(t, s) {
		if r.Err != nil {
			t.Fatalf("tenant %s: %v", id, r.Err)
		}
	}
	victim := tenantRows(t, super, b)
	if victim == "" {
		t.Fatal("tenant B was not seeded")
	}

	// 1. A claims B's ids: the shared user, B's group id under B's path and
	//    under A's path, and B's org-unit ids under A's ou subtree.
	dir.set(a, identity.DirectoryChanges{
		Users: []identity.DirectoryUser{
			{KeycloakUserID: "u-shared", Email: "attacker@example.test", Enabled: false},
			{KeycloakUserID: "u-b", Email: "attacker2@example.test", Enabled: false},
		},
		Groups: []identity.DirectoryGroup{
			{KeycloakGroupID: "g-shared", Path: path(a, "/agents"), Name: "agents", MemberKeycloakUserIDs: []string{"u-shared", "u-b"}},
			{KeycloakGroupID: "g-b-path", Path: path(b, "/admins"), Name: "admins", MemberKeycloakUserIDs: []string{"u-b"}},
		},
		OrgUnits: []identity.DirectoryGroup{
			unit("ou-b-root", "", path(a, "/ou/emea")),
			unit("ou-b-child", "ou-b-root", path(a, "/ou/emea/sales")),
		},
	})
	if r := syncAll(t, s)[a]; r.Err != nil {
		t.Fatalf("tenant A: %v", r.Err)
	}
	if got := tenantRows(t, super, b); got != victim {
		t.Fatalf("tenant A's sync changed tenant B:\n before %s\n after  %s", victim, got)
	}
	if n := count(t, super, "SELECT count(*) FROM groups WHERE tenant_id = $1 AND path NOT LIKE '/t/' || $1::text || '/%'", a); n != 0 {
		t.Errorf("tenant A holds %d groups outside /t/<A>/", n)
	}
	// Every membership and parent link of A points at A's rows: the
	// composite foreign keys hold, and none reaches B's ids.
	if n := count(t, super, `SELECT count(*) FROM group_members m
		WHERE m.tenant_id = $1 AND (m.group_id IN (SELECT id FROM groups WHERE tenant_id = $2)
		OR m.principal_id IN (SELECT id FROM principals WHERE tenant_id = $2))`, a, b); n != 0 {
		t.Errorf("tenant A has %d memberships that point at tenant B's rows", n)
	}
	if n := count(t, super, `SELECT count(*) FROM org_units WHERE tenant_id = $1
		AND parent_id IN (SELECT id FROM org_units WHERE tenant_id = $2)`, a, b); n != 0 {
		t.Errorf("tenant A has %d org units under tenant B's org units", n)
	}

	// 2. A names an org-unit parent it does not have (B's only): refused.
	//    A rolls back alone: A's rows are unchanged (apply had already
	//    deleted A's principals and groups before the org units failed), and
	//    B, visited after A, still syncs.
	attacker := tenantRows(t, super, a)
	dir.set(a, identity.DirectoryChanges{OrgUnits: []identity.DirectoryGroup{
		unit("ou-a", "ou-b-only", path(a, "/ou/x")),
	}})
	dir.set(b, identity.DirectoryChanges{
		Users: victimSnapshot(b).Users, Groups: victimSnapshot(b).Groups,
		OrgUnits: append(victimSnapshot(b).OrgUnits, unit("ou-b-only", "", path(b, "/ou/apac"))),
	})
	rs := syncAll(t, s)
	if !errors.Is(rs[a].Err, groupsync.ErrBadSnapshot) {
		t.Errorf("tenant A parent in tenant B: err = %v, want ErrBadSnapshot", rs[a].Err)
	}
	if got := tenantRows(t, super, a); got != attacker {
		t.Errorf("tenant A's refused sync did not roll back:\n before %s\n after  %s", attacker, got)
	}
	if rs[b].Err != nil {
		t.Fatalf("tenant B must still sync when tenant A fails: %v", rs[b].Err)
	}
	victim = tenantRows(t, super, b)

	// 3. A's directory empties: every A row goes, no B row does.
	dir.set(a, identity.DirectoryChanges{})
	if r := syncAll(t, s)[a]; r.Err != nil {
		t.Fatalf("tenant A: %v", r.Err)
	}
	if got := tenantRows(t, super, b); got != victim {
		t.Fatalf("tenant A's empty snapshot deleted tenant B's rows:\n before %s\n after  %s", victim, got)
	}
	if got := tenantRows(t, super, a); got != "" {
		t.Errorf("tenant A rows after an empty snapshot = %q, want none", got)
	}
}

// The tenant setting is transaction-local: after a sync, the same pooled
// connections see no tenant's rows outside InTenantTx, and a transaction for
// tenant A sees none of B's.
func TestTenantContextDoesNotLeakAcrossPooledConnections(t *testing.T) {
	super, pool := cluster(t)
	a, b := twoTenants()
	seedTenant(t, super, a, 1)
	seedTenant(t, super, b, 2)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	dir.set(a, identity.DirectoryChanges{})
	dir.set(b, victimSnapshot(b))
	s := newSyncer(t, pool, dir)
	for id, r := range syncAll(t, s) {
		if r.Err != nil {
			t.Fatalf("tenant %s: %v", id, r.Err)
		}
	}
	ctx := context.Background()
	// Hold every pooled connection at once, so each one is checked, the
	// connections the sync used included: none may still carry B's setting.
	conns := make([]*pgxpool.Conn, 0, pool.Config().MaxConns)
	release := func() {
		for _, c := range conns {
			c.Release()
		}
		conns = nil
	}
	defer release()
	for range pool.Config().MaxConns {
		c, err := pool.Acquire(ctx)
		if err != nil {
			t.Fatal(err)
		}
		conns = append(conns, c)
	}
	for i, c := range conns {
		for _, table := range []string{"principals", "groups", "group_members", "org_units"} {
			var n int
			if err := c.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n); err != nil {
				t.Fatal(err)
			}
			if n != 0 {
				t.Fatalf("pooled connection %d: nava_groupsync with no tenant set sees %d %s rows", i, n, table)
			}
		}
	}
	release() // InTenantTx below needs a connection
	err := tenancy.InTenantTx(ctx, pool, a, func(tx pgx.Tx) error {
		for _, table := range []string{"principals", "groups", "group_members", "org_units"} {
			var n int
			if err := tx.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE tenant_id = $1", b).Scan(&n); err != nil {
				return err
			}
			if n != 0 {
				t.Errorf("a transaction for tenant A sees %d of tenant B's %s rows", n, table)
			}
		}
		// A write naming B's tenant id, with a path valid for B so that
		// groups_path_check passes: only the RLS WITH CHECK can refuse it.
		_, err := tx.Exec(ctx, `INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1, $2, 'x', $3)`,
			b, uuid.New(), path(b, "/y"))
		var pgErr *pgconn.PgError
		switch {
		case err == nil:
			t.Error("a transaction for tenant A inserted a group for tenant B")
		case !errors.As(err, &pgErr) || pgErr.Code != "42501":
			t.Errorf("cross-tenant insert: err = %v, want the RLS refusal (SQLSTATE 42501)", err)
		}
		return errors.New("rollback")
	})
	if err == nil {
		t.Fatal("expected the probe transaction to roll back")
	}
}

// Holding tenant A's advisory lock skips A only; B still syncs.
func TestOneTenantsLockDoesNotBlockAnother(t *testing.T) {
	super, pool := cluster(t)
	a, b := twoTenants()
	seedTenant(t, super, a, 1)
	seedTenant(t, super, b, 2)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	dir.set(a, identity.DirectoryChanges{})
	dir.set(b, victimSnapshot(b))
	s := newSyncer(t, pool, dir)

	ctx := context.Background()
	tx, err := super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var locked bool
	if err := tx.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock(hashtextextended($1, 0))", groupsync.LockName(a)).Scan(&locked); err != nil || !locked {
		t.Fatalf("hold tenant A's lock: %v, %v", locked, err)
	}
	rs := syncAll(t, s)
	if !rs[a].Skipped {
		t.Errorf("tenant A synced while its lock was held: %+v", rs[a])
	}
	if rs[b].Skipped || rs[b].Err != nil {
		t.Errorf("tenant B was blocked by tenant A's lock: %+v", rs[b])
	}
	if n := count(t, super, "SELECT count(*) FROM principals WHERE tenant_id = $1", b); n != 2 {
		t.Errorf("tenant B principals = %d, want 2", n)
	}
}
