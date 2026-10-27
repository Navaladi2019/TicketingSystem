//go:build integration

package dbtest

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

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

// Adversarial (security review of PR #115): when an onboarding claim
// collides with a domain another tenant owns, the 23505 must not name the
// owner. The key is (domain) INCLUDE (tenant_id) and nava_onboard is under
// FORCE RLS, so PostgreSQL must leave the DETAIL ("Key (...)=(...)") out:
// neither the domain nor the owning tenant's id may appear in any field of
// the error.
func TestT18aDuplicateDomainErrorHidesOwner(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantB}, []string{tenantX})
	exec(t, super, insertDomain, tenantB, "tenant-b.test")

	_, err := execIn(asRole(t, "nava_onboard"), tenantX, insertDomain, tenantX, "tenant-b.test")
	wantSQLState(t, "X claims B's domain", err, "23505")
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("err = %v, want a *pgconn.PgError", err)
	}
	for _, field := range []string{pgErr.Message, pgErr.Detail, pgErr.Hint, pgErr.Where, pgErr.InternalQuery} {
		if strings.Contains(field, tenantB) || strings.Contains(field, "tenant-b.test") {
			t.Errorf("23505 leaks the owner or the key: message %q, detail %q, hint %q, where %q",
				pgErr.Message, pgErr.Detail, pgErr.Hint, pgErr.Where)
		}
	}
}

// releaseDomains is the statement a domain release runs (TICK-280, AR-036)
// in the tenant transaction, with the tenant as $1.
const releaseDomains = "DELETE FROM tenant_email_domain WHERE tenant_id = $1"

// TICK-280: nava_onboard frees the domains of its own provisioning tenant,
// and no other: not those of an active tenant, its own or another's, nor
// another provisioning tenant's, nor any with no tenant set. Activated in
// the same transaction, its own tenant releases nothing more.
func TestTICK280OnlyProvisioningTenantReleasesDomains(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, []string{tenantX, tenantY})
	for _, d := range []struct{ tenant, domain string }{
		{tenantA, "a.test"}, {tenantB, "b.test"}, {tenantX, "x.test"}, {tenantX, "x2.test"}, {tenantY, "y.test"},
	} {
		exec(t, super, insertDomain, d.tenant, d.domain)
	}
	onboard := asRole(t, "nava_onboard")
	owned := func() string {
		return queryString(t, super, "SELECT string_agg(domain, ',' ORDER BY domain COLLATE \"C\") FROM tenant_email_domain")
	}

	// Each attempt reaches no row, without an error.
	for _, w := range []struct{ ctxTenant, target string }{
		{tenantA, tenantA}, // its own, but active
		{tenantA, tenantB}, // another active tenant
		{tenantX, tenantA}, // an active tenant from X's transaction
		{tenantX, tenantY}, // another provisioning tenant
		{"", tenantX},      // no tenant set
	} {
		if n, err := execIn(onboard, w.ctxTenant, releaseDomains, w.target); err != nil || n != 0 {
			t.Errorf("nava_onboard in tenant %q releases %d of %s's domains, %v; want 0, nil", w.ctxTenant, n, w.target, err)
		}
	}
	if got := owned(); got != "a.test,b.test,x.test,x2.test,y.test" {
		t.Fatalf("after the refused releases the domains are %s, want all five", got)
	}

	// Control: X releases both of its own, and Y's stays.
	if n, err := execIn(onboard, tenantX, releaseDomains, tenantX); err != nil || n != 2 {
		t.Fatalf("control: X releases %d domains, %v; want 2, nil", n, err)
	}
	if got := owned(); got != "a.test,b.test,y.test" {
		t.Errorf("after X's release the domains are %s, want a.test,b.test,y.test", got)
	}

	// Y activated in the same transaction keeps its domain.
	err := inTenantTx(onboard, tenantY, func(tx pgx.Tx) error {
		if _, err := tx.Exec(t.Context(), "UPDATE tenants SET state = 'active' WHERE id = $1", tenantY); err != nil {
			return err
		}
		tag, err := tx.Exec(t.Context(), releaseDomains, tenantY)
		if err == nil && tag.RowsAffected() != 0 {
			t.Errorf("Y after activation releases %d domains, want 0", tag.RowsAffected())
		}
		return err
	})
	if err != nil {
		t.Fatalf("Y activates and releases: %v", err)
	}
	if got := owned(); got != "a.test,b.test,y.test" {
		t.Errorf("after Y's activation the domains are %s, want a.test,b.test,y.test", got)
	}
}

// TICK-280: nava_onboard reads tenant_id alone, and only its own tenant's;
// a release or read that names the domain is refused. The other roles
// cannot release at all.
func TestTICK280OnboardReadsOnlyOwnTenantID(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, []string{tenantX})
	exec(t, super, insertDomain, tenantX, "x.test")
	exec(t, super, insertDomain, tenantA, "a.test")
	onboard := asRole(t, "nava_onboard")

	for _, tenant := range []string{tenantX, tenantA} {
		var n int
		err := inTenantTx(onboard, tenant, func(tx pgx.Tx) error {
			return tx.QueryRow(t.Context(), "SELECT count(*) FROM tenant_email_domain WHERE tenant_id <> $1", tenant).Scan(&n)
		})
		if err != nil || n != 0 {
			t.Errorf("nava_onboard in tenant %s sees %d other tenants' rows, %v; want 0, nil", tenant, n, err)
		}
	}
	for _, c := range []struct {
		q    string
		args []any
	}{
		{"DELETE FROM tenant_email_domain WHERE domain = 'x.test'", nil},
		{releaseDomains + " RETURNING domain", []any{tenantX}},
		{"SELECT domain FROM tenant_email_domain", nil},
	} {
		_, err := execIn(onboard, tenantX, c.q, c.args...)
		wantSQLState(t, "nava_onboard: "+c.q, err, "42501")
	}
	for _, role := range []string{"nava_app", "nava_groupsync"} {
		_, err := execIn(asRole(t, role), tenantA, releaseDomains, tenantA)
		wantSQLState(t, role+" releases A's domains", err, "42501")
	}
	if n := queryString(t, super, "SELECT count(*)::text FROM tenant_email_domain"); n != "2" {
		t.Errorf("tenant_email_domain holds %s rows, want 2", n)
	}
}
