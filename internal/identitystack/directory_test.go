//go:build identitystack

package identitystack

// T18 (TICK-73): the Keycloak DirectoryAdmin adapter against the real realm
// `nava`, with the two service accounts the realm export defines. Each test
// that creates something removes it with the master admin.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/contracttest"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

var (
	tenantA = uuid.MustParse("00000000-0000-4000-8000-0000000000a1")
	tenantB = uuid.MustParse("00000000-0000-4000-8000-0000000000b1")
)

// memRegistry is the Registry a caller would back with RLS reads.
type memRegistry struct {
	mu     sync.Mutex
	users  map[uuid.UUID]string // principal id -> keycloak user id (all tenants; key carries the tenant below)
	groups map[uuid.UUID]string
	owner  map[uuid.UUID]uuid.UUID // principal or group id -> tenant
	bound  map[uuid.UUID][]string
}

func newRegistry() *memRegistry {
	return &memRegistry{
		users: map[uuid.UUID]string{}, groups: map[uuid.UUID]string{}, owner: map[uuid.UUID]uuid.UUID{},
		bound: map[uuid.UUID][]string{tenantA: {"idp-tenant-a"}, tenantB: {"idp-tenant-b"}},
	}
}

func (r *memRegistry) addUser(tenant uuid.UUID, kc string) uuid.UUID {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := uuid.New()
	r.users[id], r.owner[id] = kc, tenant
	return id
}

func (r *memRegistry) addGroup(tenant uuid.UUID, kc string) uuid.UUID {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := uuid.New()
	r.groups[id], r.owner[id] = kc, tenant
	return id
}

// grant lets tenant read id although another tenant owns it: it models a
// wrong or hostile caller, not a Registry behaviour.
func (r *memRegistry) UserKeycloakID(_ context.Context, tenant, id uuid.UUID) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if kc, ok := r.users[id]; ok && r.owner[id] == tenant {
		return kc, nil
	}
	return "", keycloak.ErrRegistryNotFound
}

func (r *memRegistry) GroupKeycloakID(_ context.Context, tenant, id uuid.UUID) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if kc, ok := r.groups[id]; ok && r.owner[id] == tenant {
		return kc, nil
	}
	return "", keycloak.ErrRegistryNotFound
}

func (r *memRegistry) BoundIdPAliases(_ context.Context, tenant uuid.UUID) ([]string, error) {
	return r.bound[tenant], nil
}

// hostileRegistry answers for any tenant: the registry is wrong (a bug, or a
// forged row), so the adapter's own subtree check is the only defence left.
type hostileRegistry struct{ *memRegistry }

func (h hostileRegistry) UserKeycloakID(_ context.Context, _, id uuid.UUID) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if kc, ok := h.users[id]; ok {
		return kc, nil
	}
	return "", keycloak.ErrRegistryNotFound
}

func (h hostileRegistry) GroupKeycloakID(_ context.Context, _, id uuid.UUID) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if kc, ok := h.groups[id]; ok {
		return kc, nil
	}
	return "", keycloak.ErrRegistryNotFound
}

func newOutbound(t *testing.T) *telemetry.OutboundClient {
	t.Helper()
	cfg, err := config.Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	tel, err := telemetry.New(t.Context(), cfg.Telemetry)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	out, err := telemetry.NewOutboundClient(tel, telemetry.OutboundOptions{Timeout: 20 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

var (
	runtimeCreds    = keycloak.ClientCredentials{ClientID: "nava-platform", ClientSecret: "dev-only-platform-secret"}
	onboardingCreds = keycloak.ClientCredentials{ClientID: "nava-onboarding", ClientSecret: "dev-only-onboarding-secret"}
)

func newDirectory(t *testing.T, reg keycloak.Registry, onboarding bool, idps keycloak.IdPSource) *keycloak.Directory {
	t.Helper()
	cfg := keycloak.DirectoryConfig{BaseURL: baseURL(), Runtime: runtimeCreds, Registry: reg, IdPs: idps}
	if onboarding {
		cfg.Onboarding = &onboardingCreds
	}
	d, err := keycloak.NewDirectory(cfg, newOutbound(t))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func kcGroupID(t *testing.T, path string) string {
	t.Helper()
	code, b := adminRealm(t, "nava", http.MethodGet, "/group-by-path"+path, nil)
	var g struct{ ID string }
	if code != http.StatusOK || json.Unmarshal(b, &g) != nil || g.ID == "" {
		t.Fatalf("group %s: %d %s", path, code, b)
	}
	return g.ID
}

func kcUserByEmail(t *testing.T, email string) string {
	t.Helper()
	_, b := adminRealm(t, "nava", http.MethodGet, "/users?exact=true&email="+url.QueryEscape(email), nil)
	var us []struct{ ID string }
	_ = json.Unmarshal(b, &us)
	if len(us) != 1 {
		return ""
	}
	return us[0].ID
}

func deleteUserAfter(t *testing.T, email string) {
	t.Helper()
	t.Cleanup(func() {
		if id := kcUserByEmail(t, email); id != "" {
			adminRealm(t, "nava", http.MethodDelete, "/users/"+id, nil)
		}
	})
}

func userInGroup(t *testing.T, userID, groupID string) bool {
	t.Helper()
	_, b := adminRealm(t, "nava", http.MethodGet, "/groups/"+groupID+"/members", nil)
	var us []struct{ ID string }
	_ = json.Unmarshal(b, &us)
	for _, u := range us {
		if u.ID == userID {
			return true
		}
	}
	return false
}

// ---- the contract harness --------------------------------------------------

func TestDirectoryContract(t *testing.T) {
	reg := newRegistry()
	d := newDirectory(t, reg, false, nil)
	contracttest.Suite{Directory: d, DirectoryCases: contracttest.CommonDirectoryCases()}.Run(t)
}

// ---- ChangesSince ----------------------------------------------------------

func TestChangesSinceSnapshotAndCursor(t *testing.T) {
	d := newDirectory(t, newRegistry(), false, nil)
	snap, cursor, err := d.ChangesSince(t.Context(), tenantA, "")
	if err != nil {
		t.Fatal(err)
	}
	if snap.Unchanged || cursor == "" {
		t.Fatalf("first snapshot: Unchanged=%v cursor=%q", snap.Unchanged, cursor)
	}
	emails := map[string]bool{}
	for _, u := range snap.Users {
		emails[u.Email] = true
	}
	if !emails["alice@tenant-a.test"] || !emails["admin@tenant-a.test"] || len(emails) != 2 {
		t.Errorf("tenant A users = %v, want alice and admin only", emails)
	}
	var units []string
	for _, g := range snap.OrgUnits {
		units = append(units, g.Path)
		if !strings.HasPrefix(g.Path, "/t/"+tenantA.String()+"/ou/") {
			t.Errorf("org unit outside /ou/: %s", g.Path)
		}
	}
	if len(units) != 3 {
		t.Errorf("org units = %v, want eng, eng/platform and support", units)
	}
	for _, g := range snap.Groups {
		if strings.HasPrefix(g.Path, "/t/"+tenantA.String()+"/ou") {
			t.Errorf("group under /ou/: %s", g.Path)
		}
	}

	again, cursor2, err := d.ChangesSince(t.Context(), tenantA, cursor)
	if err != nil || cursor2 != cursor || !again.Unchanged || len(again.Users)+len(again.Groups)+len(again.OrgUnits) != 0 {
		t.Fatalf("same cursor: %+v %q %v", again, cursor2, err)
	}
	if other, c3, err := d.ChangesSince(t.Context(), tenantA, "stale"); err != nil || other.Unchanged || c3 != cursor {
		t.Fatalf("stale cursor: unchanged=%v cursor=%q err=%v", other.Unchanged, c3, err)
	}

	// Tenant B sees none of tenant A.
	b, _, err := d.ChangesSince(t.Context(), tenantB, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range b.Users {
		if strings.HasSuffix(u.Email, "tenant-a.test") {
			t.Errorf("tenant B snapshot holds %s", u.Email)
		}
	}

	// A tenant with no group has an empty directory, not an error.
	empty, _, err := d.ChangesSince(t.Context(), uuid.New(), "")
	if err != nil || len(empty.Users) != 0 {
		t.Errorf("unknown tenant: %+v %v", empty, err)
	}
}

// ---- InviteUser / DisableUser / SetGroupMembers ----------------------------

func TestInviteAndDisableUser(t *testing.T) {
	reg := newRegistry()
	d := newDirectory(t, reg, false, nil)
	email := "invitee-" + uuid.NewString()[:8] + "@tenant-a.test"
	deleteUserAfter(t, email)

	if err := d.InviteUser(t.Context(), tenantA, email); err != nil {
		t.Fatalf("InviteUser: %v", err)
	}
	uid := kcUserByEmail(t, email)
	if uid == "" || !userInGroup(t, uid, kcGroupID(t, "/t/"+tenantA.String())) {
		t.Fatal("invitee not created or not in the tenant group")
	}
	if err := d.InviteUser(t.Context(), tenantA, strings.ToUpper(email)); err != nil {
		t.Errorf("second invite (idempotent, case-insensitive): %v", err)
	}
	snap, _, _ := d.ChangesSince(t.Context(), tenantA, "")
	found := false
	for _, u := range snap.Users {
		found = found || (u.Email == email && u.Enabled)
	}
	if !found {
		t.Error("invitee missing from the snapshot")
	}

	pid := reg.addUser(tenantA, uid)
	if err := d.DisableUser(t.Context(), tenantA, pid); err != nil {
		t.Fatalf("DisableUser: %v", err)
	}
	_, b := adminRealm(t, "nava", http.MethodGet, "/users/"+uid, nil)
	var u struct{ Enabled bool }
	_ = json.Unmarshal(b, &u)
	if u.Enabled {
		t.Error("user still enabled")
	}
	for _, bad := range []string{"", "x", "a b@c.test", "<a@b.test>"} {
		if err := d.InviteUser(t.Context(), tenantA, bad); !errors.Is(err, identity.ErrInvalidInput) {
			t.Errorf("InviteUser(%q) = %v, want ErrInvalidInput", bad, err)
		}
	}
}

func TestSetGroupMembers(t *testing.T) {
	reg := newRegistry()
	d := newDirectory(t, reg, false, nil)
	eng := kcGroupID(t, "/t/"+tenantA.String()+"/ou/eng")
	gid := reg.addGroup(tenantA, eng)
	alice := reg.addUser(tenantA, userID(t, serviceToken(t), "alice@tenant-a.test"))
	adm := reg.addUser(tenantA, userID(t, serviceToken(t), "admin@tenant-a.test"))

	if err := d.SetGroupMembers(t.Context(), tenantA, gid, []uuid.UUID{alice, adm, alice}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if !userInGroup(t, userID(t, serviceToken(t), "alice@tenant-a.test"), eng) {
		t.Error("alice not added")
	}
	if err := d.SetGroupMembers(t.Context(), tenantA, gid, []uuid.UUID{adm}); err != nil {
		t.Fatalf("replace: %v", err)
	}
	if userInGroup(t, userID(t, serviceToken(t), "alice@tenant-a.test"), eng) {
		t.Error("alice not removed")
	}
	if err := d.SetGroupMembers(t.Context(), tenantA, gid, nil); err != nil {
		t.Fatalf("empty: %v", err)
	}
	if err := d.SetGroupMembers(t.Context(), tenantA, uuid.New(), nil); !errors.Is(err, identity.ErrNotFound) {
		t.Errorf("unknown group: %v", err)
	}
}

// ---- guardrail 2: the federated-link guard ---------------------------------

func TestForeignFederatedUserIsRejected(t *testing.T) {
	// A user whose IdP is tenant B's lives in realm nava, federated to
	// idp-tenant-b, after one brokered login.
	email := "frank-" + uuid.NewString()[:8] + "@tenant-b.test"
	username := "frank-" + uuid.NewString()[:8]
	seedIdPUser(t, "idp-tenant-b", username, email, "dev-only-frank")
	deleteUserAfter(t, email)
	// One brokered login creates the nava user, federated to idp-tenant-b.
	_, page := brokerAttempt(t, "idp-tenant-b", username, "dev-only-frank")
	uid := kcUserByEmail(t, email)
	if uid == "" {
		t.Fatalf("brokered user not created: %.600s", page)
	}

	reg := newRegistry()
	d := newDirectory(t, reg, false, nil)
	rootA := kcGroupID(t, "/t/"+tenantA.String())

	if err := d.InviteUser(t.Context(), tenantA, email); !errors.Is(err, identity.ErrIdPNotBound) {
		t.Errorf("InviteUser into tenant A of a tenant-B-federated user = %v, want ErrIdPNotBound", err)
	}
	if userInGroup(t, uid, rootA) {
		t.Error("the foreign-federated user was added to tenant A's group")
	}

	pid := reg.addUser(tenantA, uid)
	gid := reg.addGroup(tenantA, kcGroupID(t, "/t/"+tenantA.String()+"/ou/eng"))
	if err := d.SetGroupMembers(t.Context(), tenantA, gid, []uuid.UUID{pid}); !errors.Is(err, identity.ErrIdPNotBound) {
		t.Errorf("SetGroupMembers = %v, want ErrIdPNotBound", err)
	}
	if userInGroup(t, uid, kcGroupID(t, "/t/"+tenantA.String()+"/ou/eng")) {
		t.Error("the foreign-federated user was added through SetGroupMembers")
	}

	// The same user is accepted for tenant B, whose IdP it is bound to.
	if err := d.InviteUser(t.Context(), tenantB, email); err != nil {
		t.Errorf("InviteUser into tenant B: %v", err)
	}
}

// ---- guardrail 1: the subtree is server-derived ----------------------------

func TestTenantACannotReachTenantBGroups(t *testing.T) {
	base := newRegistry()
	// The registry is deliberately hostile: it hands tenant B's group and user
	// ids to a tenant A call. The adapter's own subtree check must still hold.
	d := newDirectory(t, hostileRegistry{base}, false, nil)
	bRoot := kcGroupID(t, "/t/"+tenantB.String())
	bOU := kcGroupID(t, "/t/"+tenantB.String()+"/ou/eng")
	bUser := userID(t, serviceToken(t), "bob@tenant-b.test")
	aUser := userID(t, serviceToken(t), "alice@tenant-a.test")
	bGroup := base.addGroup(tenantB, bOU)
	bRootGroup := base.addGroup(tenantB, bRoot)
	bPrincipal := base.addUser(tenantB, bUser)
	aPrincipal := base.addUser(tenantA, aUser)

	before := membersOf(t, bOU)
	for name, gid := range map[string]uuid.UUID{"org unit": bGroup, "root": bRootGroup} {
		if err := d.SetGroupMembers(t.Context(), tenantA, gid, []uuid.UUID{aPrincipal}); !errors.Is(err, identity.ErrNotFound) {
			t.Errorf("write to tenant B %s as tenant A = %v, want ErrNotFound", name, err)
		}
	}
	if got := membersOf(t, bOU); got != before {
		t.Errorf("tenant B group changed: %q -> %q", before, got)
	}
	if err := d.DisableUser(t.Context(), tenantA, bPrincipal); !errors.Is(err, identity.ErrNotFound) {
		t.Errorf("DisableUser of tenant B's user as tenant A = %v, want ErrNotFound", err)
	}
	_, b := adminRealm(t, "nava", http.MethodGet, "/users/"+bUser, nil)
	if !strings.Contains(string(b), `"enabled":true`) {
		t.Errorf("tenant B's user was disabled: %s", b)
	}
	snap, _, err := d.ChangesSince(t.Context(), tenantA, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range append(snap.Groups, snap.OrgUnits...) {
		if !strings.HasPrefix(g.Path, "/t/"+tenantA.String()+"/") {
			t.Errorf("tenant A snapshot holds group %s", g.Path)
		}
		for _, m := range g.MemberKeycloakUserIDs {
			if m == bUser {
				t.Errorf("tenant A snapshot lists tenant B's user in %s", g.Path)
			}
		}
	}
	for _, u := range snap.Users {
		if u.KeycloakUserID == bUser {
			t.Error("tenant A snapshot holds tenant B's user")
		}
	}
}

func membersOf(t *testing.T, groupID string) string {
	t.Helper()
	_, b := adminRealm(t, "nava", http.MethodGet, "/groups/"+groupID+"/members", nil)
	var us []struct{ ID string }
	_ = json.Unmarshal(b, &us)
	ids := make([]string, 0, len(us))
	for _, u := range us {
		ids = append(ids, u.ID)
	}
	return strings.Join(ids, ",")
}

// ---- guardrail 3: the privilege floor --------------------------------------

func accountToken(t *testing.T, c keycloak.ClientCredentials) string {
	t.Helper()
	return mustToken(t, url.Values{
		"grant_type": {"client_credentials"}, "client_id": {c.ClientID}, "client_secret": {c.ClientSecret},
	})
}

func asAccount(t *testing.T, tok, method, path string, body any) int {
	t.Helper()
	code, _ := admin(t, tok, method, path, body)
	return code
}

func TestRuntimeAccountHasNoRealmOrIdPReach(t *testing.T) {
	tok := accountToken(t, runtimeCreds)
	a := orgID(t, masterToken(t), "tenant-a")
	for _, c := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPut, "", map[string]any{"displayName": "pwned"}},
		{http.MethodGet, "/identity-provider/instances", nil},
		{http.MethodGet, "/identity-provider/instances/idp-tenant-a", nil},
		{http.MethodPost, "/identity-provider/instances", map[string]any{"alias": "x", "providerId": "oidc"}},
		{http.MethodPost, "/identity-provider/instances/idp-tenant-a/mappers", map[string]any{
			"name": "x", "identityProviderAlias": "idp-tenant-a", "identityProviderMapper": "oidc-hardcoded-role-idp-mapper",
			"config": map[string]string{"role": "nava-operator"}}},
		{http.MethodGet, "/organizations", nil},
		{http.MethodPost, "/organizations/" + a + "/identity-providers", "idp-tenant-b"},
		{http.MethodGet, "/organizations/" + a + "/members", nil},
		{http.MethodGet, "/roles", nil},
		{http.MethodGet, "/authentication/flows", nil},
	} {
		if code := asAccount(t, tok, c.method, c.path, c.body); code != http.StatusForbidden {
			t.Errorf("runtime account %s %s = %d, want 403", c.method, c.path, code)
		}
	}
	// GET on the realm answers any admin with the same short public document;
	// it must carry none of the configuration view-realm would add.
	code, body := admin(t, tok, http.MethodGet, "", nil)
	if code == http.StatusOK {
		for _, k := range []string{"defaultRole", "smtpServer", "identityProviders", "adminPermissionsEnabled", "browserFlow", "accessTokenLifespan", "attributes"} {
			if strings.Contains(string(body), `"`+k+`"`) {
				t.Errorf("runtime account GET realm exposes %s: %.300s", k, body)
			}
		}
	} else if code != http.StatusForbidden {
		t.Errorf("runtime account GET realm = %d, want 403 or the short public document", code)
	}
	// It does hold the four roles.
	for _, p := range []string{"/users?max=1", "/groups?max=1", "/users/count"} {
		if code := asAccount(t, tok, http.MethodGet, p, nil); code != http.StatusOK {
			t.Errorf("runtime account GET %s = %d, want 200", p, code)
		}
	}
	// And holds exactly those client roles on realm-management, no realm role.
	checkAccountRoles(t, "service-account-nava-platform", []string{"manage-users", "query-groups", "query-users", "view-users"})
}

func TestOnboardingAccountHasNoUserReach(t *testing.T) {
	tok := accountToken(t, onboardingCreds)
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/users?max=1"},
		{http.MethodGet, "/users/count"},
		{http.MethodGet, "/groups?max=1"},
		{http.MethodPost, "/users"},
		{http.MethodGet, "/organizations/" + orgID(t, masterToken(t), "tenant-a") + "/members"},
	} {
		var body any
		if c.method == http.MethodPost {
			body = map[string]any{"username": "should-not-exist"}
		}
		if code := asAccount(t, tok, c.method, c.path, body); code != http.StatusForbidden {
			t.Errorf("onboarding account %s %s = %d, want 403", c.method, c.path, code)
		}
	}
	checkAccountRoles(t, "service-account-nava-onboarding", []string{"manage-identity-providers", "manage-realm"})
}

func checkAccountRoles(t *testing.T, username string, want []string) {
	t.Helper()
	_, b := adminRealm(t, "nava", http.MethodGet, "/users?exact=true&username="+username, nil)
	var us []struct{ ID string }
	if json.Unmarshal(b, &us) != nil || len(us) != 1 {
		t.Fatalf("%s: %s", username, b)
	}
	_, cb := adminRealm(t, "nava", http.MethodGet, "/clients?clientId=realm-management", nil)
	var cs []struct{ ID string }
	_ = json.Unmarshal(cb, &cs)
	_, rb := adminRealm(t, "nava", http.MethodGet, "/users/"+us[0].ID+"/role-mappings/clients/"+cs[0].ID, nil)
	var roles []struct{ Name string }
	_ = json.Unmarshal(rb, &roles)
	var got []string
	for _, r := range roles {
		got = append(got, r.Name)
	}
	sortStrings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("%s realm-management roles = %v, want %v", username, got, want)
	}
	_, rr := adminRealm(t, "nava", http.MethodGet, "/users/"+us[0].ID+"/role-mappings/realm", nil)
	var realmRoles []struct{ Name string }
	_ = json.Unmarshal(rr, &realmRoles)
	for _, r := range realmRoles {
		if r.Name == "nava-operator" || r.Name == "tenant-admin" {
			t.Errorf("%s holds realm role %s", username, r.Name)
		}
	}
}

func sortStrings(s []string) {
	for i := range s {
		for j := i + 1; j < len(s); j++ {
			if s[j] < s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

// ---- CreateTenantOrg -------------------------------------------------------

func oidcSpec(context.Context, uuid.UUID) (keycloak.IdPSpec, error) {
	return keycloak.IdPSpec{ProviderID: "oidc", DisplayName: "Onboarded IdP", Config: map[string]string{
		"clientId": "nava-broker", "clientSecret": "dev-only-broker-secret", "clientAuthMethod": "client_secret_post",
		"authorizationUrl": "http://localhost:18081/realms/idp-tenant-a/protocol/openid-connect/auth",
		"tokenUrl":         "http://localhost:8080/realms/idp-tenant-a/protocol/openid-connect/token",
		"defaultScope":     "openid email profile", "syncMode": "IMPORT",
	}}, nil
}

func cleanupTenantOrg(t *testing.T, tenant uuid.UUID, alias string) {
	t.Helper()
	t.Cleanup(func() {
		_, b := adminRealm(t, "nava", http.MethodGet, "/organizations?exact=true&search="+alias, nil)
		var os []struct{ ID, Alias string }
		_ = json.Unmarshal(b, &os)
		for _, o := range os {
			if o.Alias == alias {
				adminRealm(t, "nava", http.MethodDelete, "/organizations/"+o.ID, nil)
			}
		}
		adminRealm(t, "nava", http.MethodDelete, "/identity-provider/instances/"+keycloak.TenantIdPAlias(tenant), nil)
	})
}

func TestCreateTenantOrg(t *testing.T) {
	d := newDirectory(t, newRegistry(), true, oidcSpec)
	tenant := uuid.New()
	alias := "t-" + tenant.String()[:8]
	cleanupTenantOrg(t, tenant, alias)

	id, err := d.CreateTenantOrg(t.Context(), tenant, alias, []string{alias + ".test"})
	if err != nil {
		t.Fatalf("CreateTenantOrg: %v", err)
	}
	idpAlias := keycloak.TenantIdPAlias(tenant)
	if idpAlias != "idp-"+tenant.String() {
		t.Errorf("idp alias = %s", idpAlias)
	}
	_, b := adminRealm(t, "nava", http.MethodGet, "/identity-provider/instances/"+idpAlias, nil)
	var idp struct {
		TrustEmail                bool
		FirstBrokerLoginFlowAlias string
		PostBrokerLoginFlowAlias  string
		LinkOnly                  bool
		StoreToken                bool
		OrganizationLinks         []struct{ OrganizationID string }
	}
	if err := json.Unmarshal(b, &idp); err != nil {
		t.Fatal(err)
	}
	if idp.TrustEmail || idp.FirstBrokerLoginFlowAlias != "nava-first-broker-login" || idp.PostBrokerLoginFlowAlias != "" || idp.StoreToken {
		t.Errorf("idp policy wrong: %s", b)
	}
	if len(idp.OrganizationLinks) != 1 || idp.OrganizationLinks[0].OrganizationID != id {
		t.Errorf("idp links = %+v, want only org %s", idp.OrganizationLinks, id)
	}
	_, mb := adminRealm(t, "nava", http.MethodGet, "/identity-provider/instances/"+idpAlias+"/mappers", nil)
	if strings.TrimSpace(string(mb)) != "[]" {
		t.Errorf("new idp has mappers: %s", mb)
	}
	code, ob := adminRealm(t, "nava", http.MethodGet, "/organizations/"+id, nil)
	if code != http.StatusOK || !strings.Contains(string(ob), alias+".test") {
		t.Errorf("organization: %d %s", code, ob)
	}

	// The same tenant again: refused, and the first stays intact.
	if _, err := d.CreateTenantOrg(t.Context(), tenant, alias, nil); !errors.Is(err, identity.ErrAlreadyExists) {
		t.Errorf("second CreateTenantOrg = %v, want ErrAlreadyExists", err)
	}
	if code, _ := adminRealm(t, "nava", http.MethodGet, "/organizations/"+id, nil); code != http.StatusOK {
		t.Error("first organization lost")
	}
}

// An IdP that already exists under the derived alias, linked to another
// organization, is refused and nothing is left behind.
func TestCreateTenantOrgRefusesExistingLinkedIdP(t *testing.T) {
	d := newDirectory(t, newRegistry(), true, oidcSpec)
	tenant := uuid.New()
	alias := "t-" + tenant.String()[:8]
	cleanupTenantOrg(t, tenant, alias)
	idpAlias := keycloak.TenantIdPAlias(tenant)

	spec, _ := oidcSpec(t.Context(), tenant)
	code, b := adminRealm(t, "nava", http.MethodPost, "/identity-provider/instances", map[string]any{
		"alias": idpAlias, "providerId": "oidc", "enabled": true, "config": spec.Config,
	})
	if code != http.StatusCreated {
		t.Fatalf("pre-create idp: %d %s", code, b)
	}
	adminRealm(t, "nava", http.MethodPost, "/organizations/"+orgID(t, masterToken(t), "tenant-b")+"/identity-providers", idpAlias)
	defer adminRealm(t, "nava", http.MethodDelete, "/organizations/"+orgID(t, masterToken(t), "tenant-b")+"/identity-providers/"+idpAlias, nil)

	if _, err := d.CreateTenantOrg(t.Context(), tenant, alias, nil); !errors.Is(err, identity.ErrAlreadyExists) {
		t.Fatalf("CreateTenantOrg over a linked idp = %v, want ErrAlreadyExists", err)
	}
	_, ob := adminRealm(t, "nava", http.MethodGet, "/organizations?exact=true&search="+alias, nil)
	if strings.Contains(string(ob), `"alias":"`+alias+`"`) {
		t.Error("the organization was left behind")
	}
	_, ib := adminRealm(t, "nava", http.MethodGet, "/identity-provider/instances/"+idpAlias, nil)
	if !strings.Contains(string(ib), orgID(t, masterToken(t), "tenant-b")) {
		t.Errorf("the pre-existing link was changed: %s", ib)
	}
}

func TestCreateTenantOrgRefusesBadInput(t *testing.T) {
	d := newDirectory(t, newRegistry(), true, oidcSpec)
	for name, c := range map[string]struct {
		alias   string
		domains []string
	}{
		"empty alias":  {"", nil},
		"upper alias":  {"Tenant", nil},
		"slash alias":  {"a/b", nil},
		"bad domain":   {"ok-alias", []string{"not a domain"}},
		"query domain": {"ok-alias", []string{"a.test?x=1"}},
	} {
		if _, err := d.CreateTenantOrg(t.Context(), uuid.New(), c.alias, c.domains); !errors.Is(err, identity.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
	bad := newDirectory(t, newRegistry(), true, func(context.Context, uuid.UUID) (keycloak.IdPSpec, error) {
		return keycloak.IdPSpec{ProviderID: "ldap"}, nil
	})
	if _, err := bad.CreateTenantOrg(t.Context(), uuid.New(), "ok-alias", nil); !errors.Is(err, identity.ErrInvalidInput) {
		t.Errorf("provider ldap: %v", err)
	}
}

// InviteUser's error path against a closed port, end to end with the real
// outbound client: no email, no query.
func TestInviteUserClosedPortLive(t *testing.T) {
	l, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	d, err := keycloak.NewDirectory(keycloak.DirectoryConfig{
		BaseURL: "http://" + addr, Runtime: runtimeCreds, Registry: newRegistry(),
	}, newOutbound(t))
	if err != nil {
		t.Fatal(err)
	}
	err = d.InviteUser(t.Context(), tenantA, "private@tenant-a.test")
	if err == nil || strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), "?") {
		t.Errorf("err = %v", err)
	}
	_ = fmt.Sprint()
}
