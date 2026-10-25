//go:build integration

package dbtest

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak/keycloaktest"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/pgstore"
)

// T17 (TICK-72): tenant resolution (ADR-001 A.1 steps 2-4) through the real
// Keycloak adapter and pgstore, reading as nava_app under FORCE RLS.

const (
	t17PrincipalA   = "0190a000-0000-7000-8000-0000000003a1"
	t17GroupA1      = "0190a000-0000-7000-8000-0000000004a1"
	t17GroupA2      = "0190a000-0000-7000-8000-0000000004a2"
	t17PrincipalB   = "0190a000-0000-7000-8000-0000000003b1"
	t17BrokeredA    = "0190a000-0000-7000-8000-0000000003a2"
	t17OnlyInB      = "0190a000-0000-7000-8000-0000000003b2"
	t17DisabledA    = "0190a000-0000-7000-8000-0000000003a3"
	t17LinksNullA   = "0190a000-0000-7000-8000-0000000003a4"
	t17ForeignLinkA = "0190a000-0000-7000-8000-0000000003a5"
)

// seedResolution binds org-a to tenant A (IdP alias idp-a) and org-b to
// tenant B (idp-b) under issuer, and syncs principals into both tenants. The
// subject kc-shared exists in both tenants, as Keycloak ids may.
func seedResolution(t *testing.T, super *pgx.Conn, issuer string) {
	t.Helper()
	exec(t, super, "INSERT INTO tenants (id, state) VALUES ($1, 'active'), ($2, 'active')", tenantA, tenantB)
	exec(t, super, `INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id, idp_aliases) VALUES
		($1, $3, 'org-a', '{idp-a}'), ($2, $3, 'org-b', '{idp-b}')`, tenantA, tenantB, issuer)
	for _, p := range []struct{ tenant, id, sub, links string }{
		{tenantA, t17PrincipalA, "kc-shared", "[]"},
		{tenantB, t17PrincipalB, "kc-shared", "[]"},
		{tenantA, t17BrokeredA, "kc-brokered", `[{"identityProvider": "idp-a", "userId": "x", "userName": "x"}]`},
		{tenantB, t17OnlyInB, "kc-only-b", "[]"},
		{tenantA, t17DisabledA, "kc-disabled", "[]"},
		{tenantA, t17ForeignLinkA, "kc-foreign-link", `[{"identityProvider": "idp-b", "userId": "y", "userName": "y"}]`},
	} {
		exec(t, super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, federated_links, synced_at)
			VALUES ($1, $2, $3, true, $4::jsonb, now())`, p.tenant, p.id, p.sub, p.links)
	}
	// federated_links omitted: the column has no default, so this is NULL,
	// "unknown", as a sync writer that skipped it would leave it.
	exec(t, super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at)
		VALUES ($1, $2, 'kc-links-null', true, now())`, tenantA, t17LinksNullA)
	exec(t, super, "UPDATE principals SET enabled = false WHERE id = $1", t17DisabledA)
	exec(t, super, `INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES
		($1::uuid, $2, 'kc-g1', '/t/' || $1::text || '/g1'), ($1::uuid, $3, 'kc-g2', '/t/' || $1::text || '/g2')`,
		tenantA, t17GroupA2, t17GroupA1)
	exec(t, super, "INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1, $2, $4), ($1, $3, $4)",
		tenantA, t17GroupA2, t17GroupA1, t17PrincipalA)
}

// appPool is a pgxpool logged in as nava_app, the role nava-core runs as.
func appPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig("")
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig = roleConfig(t, "nava_app", "nava")
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func newT17Resolver(t *testing.T) (*keycloaktest.Realm, *keycloak.Adapter) {
	t.Helper()
	migratedPlatform(t)
	realm := keycloaktest.New(t)
	seedResolution(t, superNava(t), realm.Issuer())
	return realm, realm.Resolver(t, pgstore.New(appPool(t)))
}

func TestT17ResolvesFromBindingAndPrincipal(t *testing.T) {
	realm, r := newT17Resolver(t)
	orgA := map[string]string{"tenant-a": "org-a"}

	got, err := r.Resolve(t.Context(), realm.Mint(t, keycloaktest.Token{Subject: "kc-shared", Orgs: orgA}))
	if err != nil {
		t.Fatal(err)
	}
	// Groups in id order, as the measured query sorts them.
	want := identity.Principal{
		TenantID: uuid.MustParse(tenantA), PrincipalID: uuid.MustParse(t17PrincipalA),
		Groups: []uuid.UUID{uuid.MustParse(t17GroupA1), uuid.MustParse(t17GroupA2)},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("kc-shared via org-a = %+v, want %+v", got, want)
	}

	// The same subject under org-b is tenant B's principal: the binding, not
	// the subject, picks the tenant.
	got, err = r.Resolve(t.Context(), realm.Mint(t, keycloaktest.Token{Subject: "kc-shared", Orgs: map[string]string{"tenant-b": "org-b"}}))
	if err != nil {
		t.Fatal(err)
	}
	if got.TenantID != uuid.MustParse(tenantB) || got.PrincipalID != uuid.MustParse(t17PrincipalB) || len(got.Groups) != 0 || got.Groups == nil {
		t.Fatalf("kc-shared via org-b = %+v, want tenant B's principal with no groups", got)
	}

	got, err = r.Resolve(t.Context(), realm.Mint(t, keycloaktest.Token{Subject: "kc-brokered", Orgs: orgA, IdP: "idp-a"}))
	if err != nil {
		t.Fatal(err)
	}
	if got.PrincipalID != uuid.MustParse(t17BrokeredA) || got.IdPAlias != "idp-a" {
		t.Fatalf("brokered login = %+v", got)
	}
}

// The plan's acceptance matrix, plus the TICK-69 carry-forward cases, against
// real rows: each must be 401 tenant_unresolved.
func TestT17RefusalsAgainstPostgres(t *testing.T) {
	realm, r := newT17Resolver(t)
	orgA := map[string]string{"tenant-a": "org-a"}
	for name, tok := range map[string]keycloaktest.Token{
		"zero organizations":                {Subject: "kc-shared"},
		"two organizations":                 {Subject: "kc-shared", Orgs: map[string]string{"tenant-a": "org-a", "tenant-b": "org-b"}},
		"unbound organization":              {Subject: "kc-shared", Orgs: map[string]string{"tenant-x": "org-x"}},
		"unbound idp claim":                 {Subject: "kc-shared", Orgs: orgA, IdP: "idp-evil"},
		"idp claim bound to another tenant": {Subject: "kc-shared", Orgs: orgA, IdP: "idp-b"},
		"federated link to a foreign IdP":   {Subject: "kc-foreign-link", Orgs: orgA},
		// Adversarial, TICK-69 security review: NULL is unknown, not "no links".
		"federated_links IS NULL":                  {Subject: "kc-links-null", Orgs: orgA},
		"federated_links IS NULL, bound idp claim": {Subject: "kc-links-null", Orgs: orgA, IdP: "idp-a"},
		// Adversarial, TICK-69 security review: no principals row at all.
		"no principals row for the subject": {Subject: "kc-never-synced", Orgs: orgA},
		"principal only in another tenant":  {Subject: "kc-only-b", Orgs: orgA},
		"principal disabled by sync":        {Subject: "kc-disabled", Orgs: orgA},
	} {
		got, err := r.Resolve(t.Context(), realm.Mint(t, tok))
		if !errors.Is(err, identity.ErrTenantUnresolved) || !reflect.DeepEqual(got, identity.Principal{}) {
			t.Errorf("%s: %+v, %v; want ErrTenantUnresolved and no principal", name, got, err)
		}
	}
}

// pgstore reads the principal under RLS: even a store call naming tenant A
// with tenant B's subject finds nothing, and NULL links come back as unknown.
func TestT17StoreReadsOneTenant(t *testing.T) {
	migratedPlatform(t)
	seedResolution(t, superNava(t), "https://kc.example/realms/nava")
	s := pgstore.New(appPool(t))

	if _, err := s.Principal(t.Context(), uuid.MustParse(tenantA), "kc-only-b"); !errors.Is(err, identity.ErrNoPrincipal) {
		t.Errorf("tenant B's subject read in tenant A: err = %v, want ErrNoPrincipal", err)
	}
	p, err := s.Principal(t.Context(), uuid.MustParse(tenantA), "kc-links-null")
	if err != nil {
		t.Fatal(err)
	}
	if p.LinksKnown || p.FederatedIdPs != nil {
		t.Errorf("NULL federated_links = %+v, want LinksKnown false", p)
	}
	p, err = s.Principal(t.Context(), uuid.MustParse(tenantA), "kc-foreign-link")
	if err != nil {
		t.Fatal(err)
	}
	if !p.LinksKnown || !reflect.DeepEqual(p.FederatedIdPs, []string{"idp-b"}) {
		t.Errorf("links = %+v, want [idp-b]", p)
	}
	if _, err := s.Binding(t.Context(), "https://kc.example/realms/nava", "org-x"); !errors.Is(err, identity.ErrNoBinding) {
		t.Errorf("unbound org: err = %v, want ErrNoBinding", err)
	}
	b, err := s.Binding(t.Context(), "https://kc.example/realms/nava", "org-b")
	if err != nil || b.TenantID != uuid.MustParse(tenantB) || !reflect.DeepEqual(b.IdPAliases, []string{"idp-b"}) {
		t.Errorf("org-b = %+v, %v", b, err)
	}
	// The binding is per issuer.
	if _, err := s.Binding(t.Context(), "https://kc.example/realms/other", "org-b"); !errors.Is(err, identity.ErrNoBinding) {
		t.Errorf("org-b under another issuer: err = %v, want ErrNoBinding", err)
	}
}
