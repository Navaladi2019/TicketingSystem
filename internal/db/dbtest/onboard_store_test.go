//go:build integration

package dbtest

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
)

// T21 (TICK-81): onboard.PGStore as the real nava_onboard role, under the real
// row-level-security policies and grants.

func rolePool(t *testing.T, role string, maxConns int32) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig("")
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig = roleConfig(t, role, "nava")
	cfg.MaxConns = maxConns
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func onboardStore(t *testing.T, maxConns int32) *onboard.PGStore {
	t.Helper()
	s, err := onboard.NewPGStore(rolePool(t, "nava_onboard", maxConns), onboard.NewRegistry(rolePool(t, "nava_app", 2)))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func countRows(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := superNava(t).QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestT21BeginInsertsReplaysAndDetectsKeyReuse(t *testing.T) {
	migratedPlatform(t)
	s := onboardStore(t, 4)
	id := uuid.New()
	if st, err := s.Begin(t.Context(), id, []byte("h1"), nil); err != nil || st.Active {
		t.Fatalf("first Begin: %+v, %v", st, err)
	}
	if st, err := s.Begin(t.Context(), id, []byte("h1"), nil); err != nil || st.Active {
		t.Fatalf("replay Begin: %+v, %v", st, err)
	}
	if _, err := s.Begin(t.Context(), id, []byte("h2"), nil); !errors.Is(err, onboard.ErrKeyReuse) {
		t.Fatalf("different hash: %v, want ErrKeyReuse", err)
	}
	if n := countRows(t, "SELECT count(*) FROM tenants WHERE id = $1", id); n != 1 {
		t.Errorf("tenants rows = %d, want 1", n)
	}
	var stored []byte
	if err := superNava(t).QueryRow(context.Background(), "SELECT onboarding_request_hash FROM tenants WHERE id = $1", id).Scan(&stored); err != nil || !bytes.Equal(stored, []byte("h1")) {
		t.Errorf("stored hash = %q, %v", stored, err)
	}
}

// Bind, then Activate, under the provisioning WITH CHECKs; Bind a second time
// is a no-op (the existing binding row, read as nava_app, skips the inserts).
func TestT21BindIsIdempotentAndActivationIsLast(t *testing.T) {
	migratedPlatform(t)
	s := onboardStore(t, 4)
	id := uuid.New()
	if _, err := s.Begin(t.Context(), id, []byte("h"), []string{"acme.test", "acme.example"}); err != nil {
		t.Fatal(err)
	}
	for i := range 2 {
		if err := s.Bind(t.Context(), id, "https://kc/realms/nava", "org-"+id.String(), "idp-x"); err != nil {
			t.Fatalf("Bind #%d: %v", i+1, err)
		}
	}
	if n := countRows(t, "SELECT count(*) FROM tenant_identity_binding WHERE tenant_id = $1", id); n != 1 {
		t.Errorf("bindings = %d, want 1", n)
	}
	if n := countRows(t, "SELECT count(*) FROM tenant_email_domain WHERE tenant_id = $1", id); n != 2 {
		t.Errorf("domains = %d, want 2", n)
	}
	if st := queryString(t, superNava(t), "SELECT state FROM tenants WHERE id = $1", id); st != "provisioning" {
		t.Fatalf("state after Bind = %s, want provisioning", st)
	}
	if err := s.Activate(t.Context(), id); err != nil {
		t.Fatal(err)
	}
	if st := queryString(t, superNava(t), "SELECT state FROM tenants WHERE id = $1", id); st != "active" {
		t.Fatalf("state after Activate = %s", st)
	}
	if st, err := s.Begin(t.Context(), id, []byte("h"), nil); err != nil || !st.Active {
		t.Errorf("Begin after Activate: %+v, %v", st, err)
	}
	// An active tenant is never bound or activated again.
	if err := s.Activate(t.Context(), id); !errors.Is(err, onboard.ErrNotProvisioning) {
		t.Errorf("Activate on an active tenant: %v, want ErrNotProvisioning", err)
	}
	if err := s.Bind(t.Context(), id, "https://kc/realms/nava", "org-other", "idp-x"); !errors.Is(err, onboard.ErrNotProvisioning) {
		t.Errorf("Bind on an active tenant = %v, want ErrNotProvisioning", err)
	}
	// Refused even when the binding row exists (the retry path): same arguments.
	if err := s.Bind(t.Context(), id, "https://kc/realms/nava", "org-"+id.String(), "idp-x"); !errors.Is(err, onboard.ErrNotProvisioning) {
		t.Errorf("same-args Bind on an active tenant = %v, want ErrNotProvisioning", err)
	}
	if n := countRows(t, "SELECT count(*) FROM tenant_identity_binding WHERE tenant_id = $1", id); n != 1 {
		t.Errorf("bindings after refused Bind = %d, want 1", n)
	}
}

// A domain owned by another tenant fails Begin with ErrDomainClaimed and rolls
// the whole transaction back: no tenants row and no row for the tenant's other
// domain. The other tenant's row is untouched. nava_app still reads the domain
// as not owned (the second check).
func TestT21BeginRefusesADomainAnotherTenantOwns(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	exec(t, super, insertDomain, tenantA, "taken.test")
	s := onboardStore(t, 4)
	id := uuid.New()
	_, err := s.Begin(t.Context(), id, []byte("h"), []string{"mine.test", "taken.test"})
	if !errors.Is(err, onboard.ErrDomainClaimed) {
		t.Fatalf("Begin = %v, want ErrDomainClaimed", err)
	}
	if n := countRows(t, "SELECT count(*) FROM tenants WHERE id = $1", id); n != 0 {
		t.Errorf("a refused Begin left %d tenants rows, want 0", n)
	}
	if n := countRows(t, "SELECT count(*) FROM tenant_email_domain WHERE tenant_id = $1", id); n != 0 {
		t.Errorf("tenant has %d domain rows after a refused Begin (mine.test must roll back), want 0", n)
	}
	reg := onboard.NewRegistry(rolePool(t, "nava_app", 2))
	if got, err := reg.TenantOwnsEmailDomain(t.Context(), id, "taken.test"); err != nil || got {
		t.Errorf("TenantOwnsEmailDomain(taken.test) = %v, %v; want false", got, err)
	}
	if owner := queryString(t, super, "SELECT tenant_id::text FROM tenant_email_domain WHERE domain = 'taken.test'"); owner != tenantA {
		t.Errorf("taken.test owner = %s, want %s", owner, tenantA)
	}
}

// A same-key retry finds its tenants row, inserts nothing and keeps its
// domains (a retry after a Keycloak failure).
func TestT21BeginRetryKeepsItsDomains(t *testing.T) {
	migratedPlatform(t)
	s := onboardStore(t, 4)
	id := uuid.New()
	for i := range 2 {
		if _, err := s.Begin(t.Context(), id, []byte("h"), []string{"k1.test", "k2.test"}); err != nil {
			t.Fatalf("Begin #%d: %v", i+1, err)
		}
	}
	if n := countRows(t, "SELECT count(*) FROM tenant_email_domain WHERE tenant_id = $1", id); n != 2 {
		t.Errorf("domains = %d, want 2", n)
	}
}

// Tenants claim one domain at once: the primary key serialises them, one wins
// and the others are ErrDomainClaimed with no tenants row.
func TestT21BeginConcurrentClaimsOneWinner(t *testing.T) {
	migratedPlatform(t)
	s := onboardStore(t, 8)
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	errs := make([]error, len(ids))
	var wg sync.WaitGroup
	for i, id := range ids {
		wg.Go(func() {
			_, errs[i] = s.Begin(t.Context(), id, []byte("h"), []string{"race.test"})
		})
	}
	wg.Wait()
	wins := 0
	for i, err := range errs {
		switch {
		case err == nil:
			wins++
		case errors.Is(err, onboard.ErrDomainClaimed):
			if n := countRows(t, "SELECT count(*) FROM tenants WHERE id = $1", ids[i]); n != 0 {
				t.Errorf("loser %d left a tenants row", i)
			}
		default:
			t.Errorf("Begin %d: %v", i, err)
		}
	}
	if wins != 1 {
		t.Errorf("winners = %d, want 1", wins)
	}
}

// A same-arguments retry of a successful Bind succeeds and changes nothing:
// the tenant's domain rows stay as Begin wrote them.
func TestT21BindRetryChangesNothing(t *testing.T) {
	migratedPlatform(t)
	s := onboardStore(t, 4)
	id := uuid.New()
	if _, err := s.Begin(t.Context(), id, []byte("h"), []string{"r1.test", "r2.test"}); err != nil {
		t.Fatal(err)
	}
	for i := range 3 {
		if err := s.Bind(t.Context(), id, "https://kc/realms/nava", "org-r", "idp-r"); err != nil {
			t.Fatalf("Bind #%d: %v", i+1, err)
		}
	}
	if b, d := countRows(t, "SELECT count(*) FROM tenant_identity_binding WHERE tenant_id = $1", id),
		countRows(t, "SELECT count(*) FROM tenant_email_domain WHERE tenant_id = $1", id); b != 1 || d != 2 {
		t.Errorf("bindings=%d domains=%d, want 1 and 2", b, d)
	}
}

// A retry that resolved a different org than the stored binding names is
// refused: the binding and domains are unchanged and the tenant stays
// provisioning, so it is never activated against an org that is gone.
func TestT21BindRefusesARetryWithADifferentOrg(t *testing.T) {
	migratedPlatform(t)
	s := onboardStore(t, 4)
	id := uuid.New()
	if _, err := s.Begin(t.Context(), id, []byte("h"), []string{"a.test"}); err != nil {
		t.Fatal(err)
	}
	const issuer = "https://kc/realms/nava"
	if err := s.Bind(t.Context(), id, issuer, "org-a", "idp-a"); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ issuer, org string }{{issuer, "org-b"}, {"https://other/realms/nava", "org-a"}} {
		if err := s.Bind(t.Context(), id, c.issuer, c.org, "idp-b"); !errors.Is(err, onboard.ErrConflict) {
			t.Fatalf("Bind(%s, %s) after a binding to org-a: %v, want ErrConflict", c.issuer, c.org, err)
		}
	}
	if st := queryString(t, superNava(t), "SELECT state FROM tenants WHERE id = $1", id); st != "provisioning" {
		t.Errorf("tenant state = %s, want provisioning", st)
	}
	if got := queryString(t, superNava(t), "SELECT issuer || '|' || org_id || '|' || idp_aliases[1] FROM tenant_identity_binding WHERE tenant_id = $1", id); got != issuer+"|org-a|idp-a" {
		t.Errorf("binding = %q, want unchanged", got)
	}
	if d := countRows(t, "SELECT count(*) FROM tenant_email_domain WHERE tenant_id = $1", id); d != 1 {
		t.Errorf("domains = %d, want 1", d)
	}
}

// Binding reads are scoped by WHERE tenant_id: another tenant's binding
// (nava_app can read every row) must not make a first Bind look like a retry.
func TestT21BindIgnoresAnotherTenantsBinding(t *testing.T) {
	migratedPlatform(t)
	s := onboardStore(t, 4)
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if _, err := s.Begin(t.Context(), id, []byte("h"), nil); err != nil {
			t.Fatal(err)
		}
	}
	const issuer = "https://kc/realms/nava"
	if err := s.Bind(t.Context(), a, issuer, "org-a", "idp-a"); err != nil {
		t.Fatal(err)
	}
	if err := s.Bind(t.Context(), b, issuer, "org-b", "idp-b"); err != nil {
		t.Fatalf("first Bind of tenant B after tenant A's: %v, want nil", err)
	}
	if got := queryString(t, superNava(t), "SELECT org_id FROM tenant_identity_binding WHERE tenant_id = $1", a); got != "org-a" {
		t.Errorf("tenant A's org = %q, want org-a", got)
	}
	if got := queryString(t, superNava(t), "SELECT org_id FROM tenant_identity_binding WHERE tenant_id = $1", b); got != "org-b" {
		t.Errorf("tenant B's org = %q, want org-b", got)
	}
}

// The same (issuer, org id) cannot be bound to a second tenant: a different
// unique key, so it fails with ErrConflict and rolls back.
func TestT21BindRefusesAnOrgBoundToAnotherTenant(t *testing.T) {
	migratedPlatform(t)
	s := onboardStore(t, 4)
	a, b := uuid.New(), uuid.New()
	for id, domains := range map[uuid.UUID][]string{a: nil, b: {"b-keeps.test"}} {
		if _, err := s.Begin(t.Context(), id, []byte("h"), domains); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Bind(t.Context(), a, "https://kc/realms/nava", "org-shared", "idp-a"); err != nil {
		t.Fatal(err)
	}
	if err := s.Bind(t.Context(), b, "https://kc/realms/nava", "org-shared", "idp-b"); !errors.Is(err, onboard.ErrConflict) {
		t.Fatalf("Bind of an org bound elsewhere: %v, want ErrConflict", err)
	}
	if n := countRows(t, "SELECT count(*) FROM tenant_email_domain WHERE tenant_id = $1", b); n != 1 {
		t.Errorf("a refused Bind left %d domain rows of B, want the 1 Begin claimed", n)
	}
}

// ADR-001-E3: inside operator 2's transaction nava_onboard sees 0 rows of
// operator 1's tenant, and operator 1's tenant is unchanged by operator 2's
// replay of the same Idempotency-Key.
func TestT21OperatorTwoSeesNothingOfOperatorOnesTenant(t *testing.T) {
	migratedPlatform(t)
	s := onboardStore(t, 4)
	t1 := onboard.TenantID("operator-1", "shared-key")
	t2 := onboard.TenantID("operator-2", "shared-key")
	if t1 == t2 {
		t.Fatal("tenant ids collide")
	}
	if _, err := s.Begin(t.Context(), t1, []byte("h"), nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Bind(t.Context(), t1, "https://kc/realms/nava", "org-1", "idp-1"); err != nil {
		t.Fatal(err)
	}
	if err := s.Activate(t.Context(), t1); err != nil {
		t.Fatal(err)
	}
	if st, err := s.Begin(t.Context(), t2, []byte("h"), nil); err != nil || st.Active {
		t.Fatalf("operator 2 Begin: %+v, %v", st, err)
	}
	onb := asRole(t, "nava_onboard")
	for _, q := range []string{
		"SELECT count(*) FROM tenants WHERE id = '" + t1.String() + "'",
		"SELECT count(*) FROM tenants WHERE id <> current_setting('app.tenant_id')::uuid",
	} {
		var n int
		err := inTenantTx(onb, t2.String(), func(tx pgx.Tx) error {
			return tx.QueryRow(context.Background(), q).Scan(&n)
		})
		if err != nil || n != 0 {
			t.Errorf("%s as operator 2's tenant: %d rows, %v; want 0", q, n, err)
		}
	}
	if st := queryString(t, superNava(t), "SELECT state FROM tenants WHERE id = $1", t1); st != "active" {
		t.Errorf("operator 1's tenant state = %s, want active", st)
	}
}

// Each onboarding holds a lock connection and takes a second for its
// statements. With a pool of 2 and 2 concurrent tenants nothing deadlocks, and
// a pool of 1 is refused up front.
func TestT21LockPoolDoesNotSelfDeadlock(t *testing.T) {
	migratedPlatform(t)
	if _, err := onboard.NewPGStore(rolePool(t, "nava_onboard", 1), onboard.NewRegistry(rolePool(t, "nava_app", 2))); err == nil {
		t.Error("a 1-connection pool was accepted")
	}
	s := onboardStore(t, 2)
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			id := uuid.New()
			release, err := s.Lock(ctx, id)
			if err != nil {
				t.Errorf("Lock: %v", err)
				return
			}
			defer release()
			if _, err := s.Begin(ctx, id, []byte("h"), nil); err != nil {
				t.Errorf("Begin under the lock: %v", err)
			}
		})
	}
	wg.Wait()
}

// The per-tenant lock excludes a second holder for the same tenant.
func TestT21LockIsPerTenant(t *testing.T) {
	migratedPlatform(t)
	s := onboardStore(t, 8)
	id := uuid.New()
	release, err := s.Lock(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()
	if r2, err := s.Lock(ctx, id); err == nil {
		r2()
		t.Fatal("second Lock for the same tenant succeeded while the first was held")
	}
	other, err := s.Lock(t.Context(), uuid.New())
	if err != nil {
		t.Fatalf("Lock of another tenant: %v", err)
	}
	other()
	release()
	r3, err := s.Lock(t.Context(), id)
	if err != nil {
		t.Fatalf("Lock after release: %v", err)
	}
	r3()
}

// ReleaseDomains frees a stuck provisioning tenant's domains, frees them for
// another tenant to claim, and touches neither an active tenant's rows nor
// another provisioning tenant's.
func TestT21ReleaseDomainsFreesOnlyTheStuckTenantsRows(t *testing.T) {
	migratedPlatform(t)
	s := onboardStore(t, 4)
	stuck, active, other := uuid.New(), uuid.New(), uuid.New()
	for id, d := range map[uuid.UUID][]string{
		stuck:  {"stuck1.test", "stuck2.test"},
		active: {"active.test"},
		other:  {"other.test"},
	} {
		if _, err := s.Begin(t.Context(), id, []byte("h"), d); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Bind(t.Context(), active, "https://kc/realms/nava", "org-active", "idp-active"); err != nil {
		t.Fatal(err)
	}
	if err := s.Activate(t.Context(), active); err != nil {
		t.Fatal(err)
	}

	n, err := s.ReleaseDomains(t.Context(), stuck)
	if err != nil || n != 2 {
		t.Fatalf("ReleaseDomains(stuck) = %d, %v, want 2, nil", n, err)
	}
	if c := countRows(t, "SELECT count(*) FROM tenant_email_domain WHERE tenant_id = $1", stuck); c != 0 {
		t.Errorf("stuck tenant still holds %d domains", c)
	}
	if c := countRows(t, "SELECT count(*) FROM tenant_email_domain WHERE tenant_id = $1", other); c != 1 {
		t.Errorf("another provisioning tenant holds %d domains, want 1", c)
	}
	// The freed domain can be claimed by someone else.
	if _, err := s.Begin(t.Context(), uuid.New(), []byte("h"), []string{"stuck1.test"}); err != nil {
		t.Errorf("claim of a released domain: %v", err)
	}

	// An active tenant is out of reach, and a second release is a no-op.
	if n, err := s.ReleaseDomains(t.Context(), active); err != nil || n != 0 {
		t.Errorf("ReleaseDomains(active) = %d, %v, want 0, nil", n, err)
	}
	if c := countRows(t, "SELECT count(*) FROM tenant_email_domain WHERE tenant_id = $1", active); c != 1 {
		t.Errorf("active tenant holds %d domains, want 1", c)
	}
	if n, err := s.ReleaseDomains(t.Context(), stuck); err != nil || n != 0 {
		t.Errorf("second ReleaseDomains(stuck) = %d, %v, want 0, nil", n, err)
	}
}
