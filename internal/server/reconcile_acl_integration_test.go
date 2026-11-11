//go:build integration

// P3-35 (TICK-620), contract amendment A5: the default-ACL backfill
// (nava-onboard reconcile-default-acl) runs each tenant in its own
// transaction. A tenant that fails, here for want of an admins group row,
// gets nothing and stays fail-closed, and the batch goes on; the summary says
// who failed and the run is an error. A default is never published without the
// admins rule.

package server_test

import (
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
)

func (e *etEnv) defaultACLRows(t *testing.T, tenant uuid.UUID) (versions, entries int) {
	t.Helper()
	if err := e.super.QueryRow(t.Context(), "SELECT count(*) FROM acl_versions WHERE tenant_id = $1", tenant).Scan(&versions); err != nil {
		t.Fatal(err)
	}
	if err := e.super.QueryRow(t.Context(),
		"SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND subject_type = 'acl_policy'", tenant).Scan(&entries); err != nil {
		t.Fatal(err)
	}
	return versions, entries
}

func TestReconcileDefaultACLOneTenantFailsTheBatchGoesOn(t *testing.T) {
	e := newETEnv(t)
	e.noTestACL = true // these tenants start with no ACL, as before the backfill
	mid := etSeedTenant(t, e.super, "c")
	e.tokens["admin-c"] = mid.admin
	for _, tc := range []struct{ token, key string }{{"admin-a", "a"}, {"admin-c", "c"}, {"admin-b", "b"}} {
		if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", tc.token, "reconcile-"+tc.key, etBody("incident", "INC")); rec.Code != http.StatusCreated {
			t.Fatalf("create incident for %s = %d: %s", tc.token, rec.Code, rec.Body)
		}
	}
	// The middle tenant has no admins group row.
	if _, err := e.super.Exec(t.Context(), "DELETE FROM tenant_admin_group WHERE tenant_id = $1", mid.id); err != nil {
		t.Fatal(err)
	}
	pub := onboard.ACLStore{Pool: e.pool}
	log := slog.New(slog.DiscardHandler)
	batch := []uuid.UUID{e.a.id, mid.id, e.b.id}

	sum, err := onboard.ReconcileDefaultACLs(t.Context(), pub, log, batch)
	if err == nil || !strings.Contains(err.Error(), "1 of 3") {
		t.Fatalf("batch error = %v, want a failure for 1 of 3 tenants", err)
	}
	if want := (onboard.ACLReconcileSummary{Published: 2, Failed: 1}); sum != want {
		t.Fatalf("summary = %+v, want %+v", sum, want)
	}
	for name, tc := range map[string]struct {
		id       uuid.UUID
		versions int
	}{"first": {e.a.id, 1}, "middle": {mid.id, 0}, "last": {e.b.id, 1}} {
		if v, n := e.defaultACLRows(t, tc.id); v != tc.versions || n != tc.versions {
			t.Errorf("%s tenant: %d ACL versions and %d audit entries, want %d of each", name, v, n, tc.versions)
		}
	}
	// Every published default has the admins rule, naming the tenant's own group.
	for _, tn := range []etTenant{e.a, e.b} {
		var rules string
		if err := e.super.QueryRow(t.Context(), "SELECT rules::text FROM acl_versions WHERE tenant_id = $1 AND version = 1", tn.id).Scan(&rules); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(rules, `"default-admins"`) || !strings.Contains(rules, tn.adminGroup.String()) {
			t.Errorf("tenant %s default %s lacks the admins rule for %s", tn.id, rules, tn.adminGroup)
		}
	}

	// A rerun after the row is added publishes only the middle tenant.
	if _, err := e.super.Exec(t.Context(), "INSERT INTO tenant_admin_group (tenant_id, group_id) VALUES ($1, $2)", mid.id, mid.adminGroup); err != nil {
		t.Fatal(err)
	}
	sum, err = onboard.ReconcileDefaultACLs(t.Context(), pub, log, batch)
	if err != nil {
		t.Fatalf("rerun: %v", err)
	}
	if want := (onboard.ACLReconcileSummary{Published: 1, Unchanged: 2}); sum != want {
		t.Fatalf("rerun summary = %+v, want %+v", sum, want)
	}
	for _, id := range []uuid.UUID{e.a.id, mid.id, e.b.id} {
		if v, n := e.defaultACLRows(t, id); v != 1 || n != 1 {
			t.Errorf("tenant %s after the rerun: %d ACL versions and %d audit entries, want 1 of each", id, v, n)
		}
	}
}
