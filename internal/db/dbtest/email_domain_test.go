//go:build integration

package dbtest

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/identity/pgstore"
)

// T18a (TICK-194): tenant_email_domain behind keycloak.Registry's
// TenantOwnsEmailDomain. A domain belongs to one tenant realm-wide; nava_app
// reads it under the tenant predicate; nava_onboard claims it only for its
// own provisioning tenant.

const insertDomain = "INSERT INTO tenant_email_domain (tenant_id, domain) VALUES ($1, $2)"

// Acceptance: the same domain for two tenants fails on the unique
// constraint, through the onboarding role and even as the superuser.
func TestT18aDomainUniqueAcrossTenants(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, []string{tenantX, tenantY})
	onboard := asRole(t, "nava_onboard")

	if _, err := execIn(onboard, tenantX, insertDomain, tenantX, "shared.test"); err != nil {
		t.Fatalf("X claims shared.test: %v", err)
	}
	_, err := execIn(onboard, tenantY, insertDomain, tenantY, "shared.test")
	wantSQLState(t, "Y claims X's domain", err, "23505")
	_, err = super.Exec(t.Context(), insertDomain, tenantA, "shared.test")
	wantSQLState(t, "the superuser gives X's domain to A", err, "23505")
	_, err = execIn(onboard, tenantX, insertDomain, tenantX, "shared.test")
	wantSQLState(t, "X claims shared.test twice", err, "23505")
}

// Acceptance: under tenant A's RLS context, tenant B's domain reads as not
// owned, the same as a domain nobody owns, and without an error.
func TestT18aCrossTenantDomainNotOwned(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	exec(t, super, insertDomain, tenantA, "tenant-a.test")
	exec(t, super, insertDomain, tenantB, "tenant-b.test")

	s := pgstore.New(appPool(t))
	a, b := uuid.MustParse(tenantA), uuid.MustParse(tenantB)
	for _, c := range []struct {
		tenant uuid.UUID
		domain string
		want   bool
	}{
		{a, "tenant-a.test", true},
		{a, "Tenant-A.TEST", true},
		{a, "tenant-b.test", false},
		{a, "nobody.test", false},
		{b, "tenant-b.test", true},
		{b, "tenant-a.test", false},
	} {
		owns, err := s.TenantOwnsEmailDomain(t.Context(), c.tenant, c.domain)
		if err != nil || owns != c.want {
			t.Errorf("TenantOwnsEmailDomain(%s, %q) = %v, %v; want %v, nil", c.tenant, c.domain, owns, err, c.want)
		}
	}

	// The row itself is invisible to A, whatever the query names, and to a
	// transaction with no tenant.
	app := asRole(t, "nava_app")
	for _, tenant := range []string{tenantA, ""} {
		var n int
		err := inTenantTx(app, tenant, func(tx pgx.Tx) error {
			return tx.QueryRow(t.Context(), "SELECT count(*) FROM tenant_email_domain WHERE domain = 'tenant-b.test' OR tenant_id = $1", tenantB).Scan(&n)
		})
		if err != nil || n != 0 {
			t.Errorf("nava_app in tenant %q sees %d of B's domain rows, %v; want 0, nil", tenant, n, err)
		}
	}
}

// nava_onboard claims domains only for the tenant of its transaction, and
// only while it is provisioning, as binding_onboard binds; nava_app cannot
// claim one at all.
func TestT18aOnlyProvisioningTenantClaimsDomains(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, []string{tenantX, tenantY})
	onboard := asRole(t, "nava_onboard")

	if _, err := execIn(onboard, tenantX, insertDomain, tenantX, "x.test"); err != nil {
		t.Fatalf("control: X claims x.test: %v", err)
	}
	for _, w := range []struct{ ctxTenant, target, domain string }{
		{tenantX, tenantY, "y.test"}, // another provisioning tenant
		{tenantA, tenantA, "a.test"}, // its own, but active
		{tenantX, tenantA, "a.test"}, // an active tenant from X's transaction
		{"", tenantX, "x2.test"},     // no tenant set
	} {
		_, err := execIn(onboard, w.ctxTenant, insertDomain, w.target, w.domain)
		wantSQLState(t, "nava_onboard in tenant "+w.ctxTenant+" claims "+w.domain+" for "+w.target, err, "42501")
	}
	_, err := execIn(asRole(t, "nava_app"), tenantA, insertDomain, tenantA, "a.test")
	wantSQLState(t, "nava_app claims a domain", err, "42501")
	if n := queryString(t, super, "SELECT count(*)::text FROM tenant_email_domain"); n != "1" {
		t.Errorf("tenant_email_domain holds %s rows, want only X's", n)
	}
}

// The CHECK refuses a domain the lower-cased lookup could never match.
func TestT18aDomainCheck(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, nil, []string{tenantX})
	onboard := asRole(t, "nava_onboard")
	for _, d := range []string{"Tenant-X.test", "x@x.test", "localhost", "", " x.test", "x.test.", "-x.test", "x..test"} {
		_, err := execIn(onboard, tenantX, insertDomain, tenantX, d)
		wantSQLState(t, "claim "+d, err, "23514")
	}
	for _, d := range []string{"x.test", "mail.x-1.example", "xn--bcher-kva.example"} {
		if _, err := execIn(onboard, tenantX, insertDomain, tenantX, d); err != nil {
			t.Errorf("claim %s: %v", d, err)
		}
	}
}
