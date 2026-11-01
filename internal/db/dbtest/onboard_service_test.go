//go:build integration

package dbtest

import (
	"context"
	"log/slog"
	"net/netip"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
)

// P3-06b (TICK-382): the whole onboard.Service.Onboard against the real
// database as nava_onboard (domain reads as nava_app), with a fake directory in
// place of Keycloak. TICK-344's tests call SeedEntityTypes directly; this one
// proves Onboard wires the seed in, before activation, and that a replay of
// the same request writes nothing.

// e2eDir is the Keycloak side of onboarding: one org per tenant, a stable
// admins group id, and a count of every call so a replay can be shown to make
// none.
type e2eDir struct {
	mu      sync.Mutex
	orgs    map[uuid.UUID]string
	creates int
	ensures int
	invites []string
}

func (d *e2eDir) CreateTenantOrg(_ context.Context, id uuid.UUID, _ string, _ []string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.orgs[id]; ok {
		return "", identity.ErrAlreadyExists
	}
	d.creates++
	d.orgs[id] = "org-" + id.String()
	return d.orgs[id], nil
}

func (d *e2eDir) InviteUser(_ context.Context, _ uuid.UUID, email string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.invites = append(d.invites, email)
	return nil
}

func (d *e2eDir) EnsureAdminGroup(_ context.Context, id uuid.UUID) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensures++
	return "kc-admins-" + id.String(), nil
}

func (d *e2eDir) FindTenantOrg(_ context.Context, id uuid.UUID, _ string) (string, bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	o, ok := d.orgs[id]
	return o, ok, nil
}

func (d *e2eDir) calls() (creates, ensures, invites int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.creates, d.ensures, len(d.invites)
}

// e2eResolver answers every host with one public address, so the IdP URL
// checks pass without DNS.
type e2eResolver struct{}

func (e2eResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
}

func onboardService(t *testing.T) (*onboard.Service, *e2eDir) {
	t.Helper()
	specs := onboard.NewSpecs()
	dir := &e2eDir{orgs: map[uuid.UUID]string{}}
	svc, err := onboard.New(onboard.Options{
		Store:     onboardStore(t, 4),
		Directory: dir,
		Finder:    dir,
		Domains:   onboard.NewRegistry(rolePool(t, "nava_app", 2)),
		Specs:     specs,
		Resolver:  e2eResolver{},
		Issuer:    "https://kc/realms/nava",
		Log:       slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc, dir
}

func onboardRequest(domain string) onboard.Request {
	return onboard.Request{
		AdminEmail:   "admin@" + domain,
		EmailDomains: []string{domain},
		IdP: onboard.IdPRequest{ProviderID: "oidc", DisplayName: "E2E", Config: map[string]string{
			"clientId": "c", "clientSecret": "fixture-not-a-secret",
			"authorizationUrl": "https://idp.example.org/auth", "tokenUrl": "https://idp.example.org/token",
			"jwksUrl": "https://idp.example.org/jwks",
		}},
	}
}

// tenantFootprint counts every row onboarding writes for a tenant.
func tenantFootprint(t *testing.T, id uuid.UUID) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, table := range []string{
		"tenants", "tenant_identity_binding", "tenant_email_domain", "groups",
		"tenant_admin_group", "entity_definitions", "entity_definition_versions",
	} {
		col := "tenant_id"
		if table == "tenants" {
			col = "id"
		}
		out[table] = countRows(t, "SELECT count(*) FROM "+table+" WHERE "+col+" = $1", id)
	}
	return out
}

func TestP306bServiceOnboardSeedsThreeTypesAndActivates(t *testing.T) {
	migratedPlatform(t)
	svc, dir := onboardService(t)
	op := identity.Operator{PrincipalID: "op-" + uuid.NewString()}
	key := uuid.NewString()
	want := onboard.TenantID(op.PrincipalID, key)
	req := onboardRequest("e2e-" + want.String()[:8] + ".test")

	res, err := svc.Onboard(t.Context(), op, key, []byte("h"), req)
	if err != nil {
		t.Fatalf("Onboard: %v", err)
	}
	if res.TenantID != want || res.Replayed {
		t.Fatalf("Onboard = %+v, want tenant %s, not replayed", res, want)
	}
	super := superNava(t)
	if st := queryString(t, super, "SELECT state FROM tenants WHERE id = $1", want); st != "active" {
		t.Fatalf("tenant state = %s, want active", st)
	}
	for _, w := range seedWant {
		defID, verID := onboard.EntityDefinitionID(want, w.key), onboard.EntityVersionID(want, w.key, 1)
		got := queryString(t, super, `SELECT d.id::text || ' ' || d.number_prefix || ' ' || d.name || ' ' || d.current_version
			FROM entity_definitions d WHERE d.tenant_id = $1 AND d.key = $2`, want, w.key)
		if exp := defID.String() + " " + w.prefix + " " + w.name + " 1"; got != exp {
			t.Errorf("%s definition = %q, want %q", w.key, got, exp)
		}
		got = queryString(t, super, `SELECT v.id::text || ' ' || v.version || ' ' || v.compat
			FROM entity_definition_versions v WHERE v.tenant_id = $1 AND v.definition_id = $2`, want, defID)
		if exp := verID.String() + " 1 initial"; got != exp {
			t.Errorf("%s version = %q, want %q", w.key, got, exp)
		}
	}
	fp := tenantFootprint(t, want)
	if fp["entity_definitions"] != 3 || fp["entity_definition_versions"] != 3 {
		t.Errorf("tenant has %d types, %d versions; want 3 and 3", fp["entity_definitions"], fp["entity_definition_versions"])
	}
	if fp["tenant_identity_binding"] != 1 || fp["tenant_email_domain"] != 1 || fp["tenant_admin_group"] != 1 {
		t.Errorf("footprint = %v, want one binding, one domain, one admin group row", fp)
	}
	if c, e, i := dir.calls(); c != 1 || e != 1 || i != 1 {
		t.Errorf("directory calls: %d creates, %d ensures, %d invites; want 1 each", c, e, i)
	}
}

// A replay of the same request (same operator, key and hash) after the tenant
// is active is Replayed and writes nothing: no row in any onboarding table and
// no directory call.
func TestP306bServiceOnboardReplayInsertsNothing(t *testing.T) {
	migratedPlatform(t)
	svc, dir := onboardService(t)
	op := identity.Operator{PrincipalID: "op-" + uuid.NewString()}
	key := uuid.NewString()
	id := onboard.TenantID(op.PrincipalID, key)
	req := onboardRequest("replay-" + id.String()[:8] + ".test")

	if _, err := svc.Onboard(t.Context(), op, key, []byte("h"), req); err != nil {
		t.Fatalf("first Onboard: %v", err)
	}
	before := tenantFootprint(t, id)
	all := countRows(t, "SELECT count(*) FROM entity_definition_versions")
	c0, e0, i0 := dir.calls()

	res, err := svc.Onboard(t.Context(), op, key, []byte("h"), req)
	if err != nil {
		t.Fatalf("replay Onboard: %v", err)
	}
	if res.TenantID != id || !res.Replayed {
		t.Fatalf("replay = %+v, want tenant %s, replayed", res, id)
	}
	after := tenantFootprint(t, id)
	for table, n := range before {
		if after[table] != n {
			t.Errorf("%s rows for the tenant: %d before the replay, %d after", table, n, after[table])
		}
	}
	if n := countRows(t, "SELECT count(*) FROM entity_definition_versions"); n != all {
		t.Errorf("entity_definition_versions holds %d rows after the replay, want %d", n, all)
	}
	if c, e, i := dir.calls(); c != c0 || e != e0 || i != i0 {
		t.Errorf("the replay called the directory: creates %d->%d, ensures %d->%d, invites %d->%d", c0, c, e0, e, i0, i)
	}
}
