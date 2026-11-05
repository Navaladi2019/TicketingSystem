//go:build integration

package dbtest

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
)

// P5-23 (TICK-568): nava_onboard may create the auditors group of an active
// tenant that has none (migration 20261104191200), and nothing else. Tenant A
// and B are active; seedIdentity gives each an /agents group whose id is the
// tenant id.

const (
	insertAuditorsGroup = "INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1::uuid, $2, $3, '/t/' || $1::text || '/auditors')"
	insertGroupAtPath   = "INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1::uuid, $2, $3, $4)"
	insertAuditorRow    = "INSERT INTO tenant_auditor_group (tenant_id, group_id) VALUES ($1, $2)"
)

func activeAB(t *testing.T) *pgx.Conn {
	t.Helper()
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedIdentity(t, super, tenantA, tenantB)
	return super
}

// The reconcile path: RecordAuditorGroup on an active tenant without the
// group writes the groups row and the pointer, which names AuditorGroupID.
func TestP523RecordAuditorGroupForActiveTenant(t *testing.T) {
	super := activeAB(t)
	a := uuid.MustParse(tenantA)
	if err := onboardStore(t, 4).RecordAuditorGroup(context.Background(), a, "kc-auditors-a"); err != nil {
		t.Fatalf("RecordAuditorGroup on active A: %v", err)
	}
	if got := queryString(t, super, "SELECT group_id::text FROM tenant_auditor_group WHERE tenant_id = $1", tenantA); got != onboard.AuditorGroupID(a).String() {
		t.Errorf("A's pointer = %s, want %s", got, onboard.AuditorGroupID(a))
	}
	if got := queryString(t, super, "SELECT path FROM groups WHERE tenant_id = $1 AND id = $2", tenantA, onboard.AuditorGroupID(a)); got != "/t/"+tenantA+"/auditors" {
		t.Errorf("path = %s", got)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM tenant_auditor_group WHERE tenant_id = $1", tenantB); got != "0" {
		t.Errorf("B's pointer rows = %s, want 0", got)
	}
	// Once A has the pointer, a second run is refused, not a silent rewrite:
	// the reconcile checks tenant_auditor_group first.
	if err := onboardStore(t, 4).RecordAuditorGroup(context.Background(), a, "kc-auditors-a2"); sqlState(err) != "42501" {
		t.Errorf("second RecordAuditorGroup on A: %v, want SQLSTATE 42501", err)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM groups WHERE tenant_id = $1 AND path LIKE '%/auditors'", tenantA); got != "1" {
		t.Errorf("A's auditors groups = %s, want 1", got)
	}
}

// Group sync created the auditors group first, under an id of its own: the
// insert is a no-op, the SELECT policy shows the row, and the pointer names
// the sync's id.
func TestP523RecordAuditorGroupAdoptsSyncedGroup(t *testing.T) {
	super := activeAB(t)
	synced := uuid.New()
	exec(t, super, insertAuditorsGroup, tenantA, synced, "kc-auditors-a")
	if err := onboardStore(t, 4).RecordAuditorGroup(context.Background(), uuid.MustParse(tenantA), "kc-auditors-a"); err != nil {
		t.Fatalf("RecordAuditorGroup: %v", err)
	}
	if got := queryString(t, super, "SELECT group_id::text FROM tenant_auditor_group WHERE tenant_id = $1", tenantA); got != synced.String() {
		t.Errorf("A's pointer = %s, want the synced group %s", got, synced)
	}
}

// Every write the contract excludes is refused, for an active tenant in its
// own context.
func TestP523OnboardActiveTenantRefusals(t *testing.T) {
	super := activeAB(t)
	ob := asRole(t, "nava_onboard")
	a, b := uuid.MustParse(tenantA), uuid.MustParse(tenantB)
	refused := func(what, code, sql string, args ...any) {
		t.Helper()
		_, err := execIn(ob, tenantA, sql, args...)
		wantSQLState(t, what, err, code)
	}
	// groups: anything but A's own /t/A/auditors.
	for name, path := range map[string]string{
		"admins":    "/t/" + tenantA + "/admins",
		"other":     "/t/" + tenantA + "/g",
		"lookalike": "/t/" + tenantA + "/ou/auditors",
		"subgroup":  "/t/" + tenantA + "/auditors/night",
	} {
		refused("groups "+name, "42501", insertGroupAtPath, tenantA, uuid.New(), "kc-"+name, path)
	}
	refused("groups: B's auditors from A", "42501", insertAuditorsGroup, tenantB, onboard.AuditorGroupID(b), "kc-b")
	// tenant_auditor_group: only a live auditors group of A.
	refused("pointer to A's agents group", "42501", insertAuditorRow, tenantA, a)
	exec(t, super, insertAuditorsGroup, tenantB, onboard.AuditorGroupID(b), "kc-b")
	refused("pointer to B's auditors group", "42501", insertAuditorRow, tenantA, onboard.AuditorGroupID(b))
	refused("B's pointer from A", "42501", insertAuditorRow, tenantB, onboard.AuditorGroupID(b))
	tomb := uuid.New()
	exec(t, super, insertAuditorsGroup, tenantA, tomb, "kc-tomb")
	exec(t, super, tombstoneGroup, tenantA, tomb)
	refused("pointer to a tombstoned auditors group", "42501", insertAuditorRow, tenantA, tomb)

	// The allowed pair succeeds.
	if _, err := execIn(ob, tenantA, insertAuditorsGroup, tenantA, onboard.AuditorGroupID(a), "kc-a"); err != nil {
		t.Fatalf("A's auditors group: %v", err)
	}
	if _, err := execIn(ob, tenantA, insertAuditorRow, tenantA, onboard.AuditorGroupID(a)); err != nil {
		t.Fatalf("A's pointer: %v", err)
	}
	// Once A has its pointer: no second pointer, no second auditors group.
	refused("second pointer", "23505", insertAuditorRow, tenantA, onboard.AuditorGroupID(a))
	refused("second pointer, ON CONFLICT", "", insertAuditorRow+" ON CONFLICT DO NOTHING", tenantA, onboard.AuditorGroupID(a))
	refused("second auditors group", "42501", insertAuditorsGroup, tenantA, uuid.New(), "kc-a2")
	// No UPDATE and no DELETE on either table.
	for what, sql := range map[string]string{
		"UPDATE groups":               "UPDATE groups SET path = path WHERE tenant_id = $1",
		"DELETE groups":               "DELETE FROM groups WHERE tenant_id = $1",
		"UPDATE tenant_auditor_group": "UPDATE tenant_auditor_group SET group_id = group_id WHERE tenant_id = $1",
		"DELETE tenant_auditor_group": "DELETE FROM tenant_auditor_group WHERE tenant_id = $1",
	} {
		refused(what, "42501", sql, tenantA)
	}
	if got := queryString(t, super, "SELECT group_id::text FROM tenant_auditor_group WHERE tenant_id = $1", tenantA); got != onboard.AuditorGroupID(a).String() {
		t.Errorf("A's pointer = %s", got)
	}
}

// Under A's context nava_onboard sees A's auditors group and pointer, and
// nothing of B nor A's other groups.
func TestP523OnboardActiveTenantSelectIsolation(t *testing.T) {
	super := activeAB(t)
	for _, id := range []string{tenantA, tenantB} {
		g := onboard.AuditorGroupID(uuid.MustParse(id))
		exec(t, super, insertAuditorsGroup, id, g, "kc-auditors")
		exec(t, super, insertAuditorRow, id, g)
	}
	ob := asRole(t, "nava_onboard")
	count := func(ctxTenant, sql string) int {
		t.Helper()
		var n int
		if err := inTenantTx(ob, ctxTenant, func(tx pgx.Tx) error {
			return tx.QueryRow(context.Background(), sql).Scan(&n)
		}); err != nil {
			t.Fatalf("%s from '%s': %v", sql, ctxTenant, err)
		}
		return n
	}
	for sql, want := range map[string]int{
		"SELECT count(*)::int FROM groups":                                                                               1,
		"SELECT count(*)::int FROM groups WHERE tenant_id = '" + tenantB + "'":                                           0,
		"SELECT count(*)::int FROM groups WHERE id = '" + tenantA + "'":                                                  0,
		"SELECT count(*)::int FROM tenant_auditor_group":                                                                 1,
		"SELECT count(*)::int FROM tenant_auditor_group WHERE tenant_id = '" + tenantB + "'":                             0,
		"SELECT count(*)::int FROM groups WHERE id = '" + onboard.AuditorGroupID(uuid.MustParse(tenantA)).String() + "'": 1,
	} {
		if n := count(tenantA, sql); n != want {
			t.Errorf("from A, %s = %d, want %d", sql, n, want)
		}
	}
	for _, sql := range []string{"SELECT count(*)::int FROM groups", "SELECT count(*)::int FROM tenant_auditor_group"} {
		if n := count("", sql); n != 0 {
			t.Errorf("no tenant, %s = %d, want 0", sql, n)
		}
	}
	// The pointer's group_id stays ungranted.
	_, err := execIn(ob, tenantA, "SELECT group_id FROM tenant_auditor_group")
	wantSQLState(t, "nava_onboard SELECT group_id", err, "42501")
}
