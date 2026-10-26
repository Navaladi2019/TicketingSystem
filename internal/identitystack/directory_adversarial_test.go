//go:build identitystack

package identitystack

// Security review of T18 (TICK-73, PR #101): a tenant admin, through the
// DirectoryAdmin adapter, must never reach a Keycloak user that belongs to
// another tenant or to the platform. The adapter's user accounts are realm
// wide, so whatever lets tenant A put a user in /t/<A> also hands tenant A
// that user's email (ChangesSince) and DisableUser over them.

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// enrolAttack invites email into tenant A and, if the adapter accepted it,
// shows what tenant A can then do: read the user and disable them realm wide.
func enrolAttack(t *testing.T, email string) {
	t.Helper()
	victim := userID(t, serviceToken(t), email)
	rootA := kcGroupID(t, "/t/"+tenantA.String())
	t.Cleanup(func() {
		adminRealm(t, "nava", http.MethodDelete, "/users/"+victim+"/groups/"+rootA, nil)
		_, b := adminRealm(t, "nava", http.MethodGet, "/users/"+victim, nil)
		var rep map[string]any
		if json.Unmarshal(b, &rep) == nil {
			rep["enabled"] = true
			adminRealm(t, "nava", http.MethodPut, "/users/"+victim, rep)
		}
	})

	reg := newRegistry()
	d := newDirectory(t, reg, false, nil)
	err := d.InviteUser(t.Context(), tenantA, email)
	if err == nil {
		t.Errorf("InviteUser(tenant A, %s) = nil: a user of another tenant or of the platform was enrolled without consent", email)
	}
	if !userInGroup(t, victim, rootA) {
		return
	}
	t.Errorf("%s was placed in /t/%s", email, tenantA)

	snap, _, _ := d.ChangesSince(t.Context(), tenantA, "")
	for _, u := range snap.Users {
		if u.KeycloakUserID == victim {
			t.Errorf("tenant A's directory snapshot now holds %s (email %s)", victim, u.Email)
		}
	}
	// T25 would sync the user into tenant A's principals; model that row.
	pid := reg.addUser(tenantA, victim)
	if err := d.DisableUser(t.Context(), tenantA, pid); err == nil {
		t.Errorf("tenant A disabled %s realm wide", email)
	}
}

// bob is a local user of tenant B (org tenant-b, no federated link), so the
// federated-link guard does not stop him.
func TestInviteCannotEnrolAnotherTenantsUser(t *testing.T) {
	enrolAttack(t, "bob@tenant-b.test")
}

// The platform operator has no tenant at all.
func TestInviteCannotEnrolPlatformOperator(t *testing.T) {
	enrolAttack(t, "operator@nava.test")
}

// Tenant A invites an address in tenant B's domain before tenant B's user has
// ever logged in. nava-first-broker-login never links by email, so the user
// tenant A created blocks tenant B's user at first brokered login.
func TestInviteCannotSquatAnotherTenantsEmail(t *testing.T) {
	email := "squat-" + uuid.NewString()[:8] + "@tenant-b.test"
	username := "squat-" + uuid.NewString()[:8]
	deleteUserAfter(t, email)

	d := newDirectory(t, newRegistry(), false, nil)
	if err := d.InviteUser(t.Context(), tenantA, email); err != nil {
		return // refused: safe
	}
	seedIdPUser(t, "idp-tenant-b", username, email, "dev-only-squat")
	if tok, page := brokerAttempt(t, "idp-tenant-b", username, "dev-only-squat"); tok == "" {
		t.Errorf("tenant A's invite of %s locked tenant B's federated user out of first login: %.300s", email, page)
	}
}

// GET /admin/realms/nava answers any admin token with a short public
// document. A deny-list of config keys misses whatever a Keycloak upgrade
// adds; pin the exact key set instead, so any new key fails here.
func TestRuntimeRealmReadIsPublicDocumentOnly(t *testing.T) {
	code, body := admin(t, accountToken(t, runtimeCreds), http.MethodGet, "", nil)
	if code == http.StatusForbidden {
		return
	}
	if code != http.StatusOK {
		t.Fatalf("runtime GET realm = %d, want 200 public document or 403", code)
	}
	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatalf("realm document: %v", err)
	}
	allowed := map[string]bool{
		"realm": true, "registrationEmailAsUsername": true, "bruteForceProtected": true,
		"supportedLocales": true, "organizationsEnabled": true,
	}
	for k := range doc {
		if !allowed[k] {
			t.Errorf("runtime GET realm exposes key %q", k)
		}
	}
}

// The runtime account's manage-users can map nava-operator (AR-030). The
// adapter must never be the code that does it: no role-mapping route in it.
func TestAdapterNeverMapsRoles(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "identity", "keycloak", "*.go"))
	if err != nil || len(files) == 0 {
		t.Fatalf("adapter sources: %v", err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, route := range []string{"role-mappings", "/roles", "impersonation", "/credentials", "reset-password", "execute-actions-email"} {
			if strings.Contains(string(b), route) {
				t.Errorf("%s calls %s: the adapter must not map roles or touch credentials", filepath.Base(f), route)
			}
		}
	}
}
