package keycloak

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// memStore is an in-memory identity.Store. The PostgreSQL store runs the same
// cases against a real database in internal/db/dbtest (TestT17*).
type memStore struct {
	mu         sync.Mutex
	bindings   map[[2]string][]identity.Binding // (issuer, org id) → rows
	principals map[principalKey]identity.PrincipalRecord
	fail       error // returned by every call when set
}

type principalKey struct {
	tenant  uuid.UUID
	subject string
}

func newMemStore() *memStore {
	return &memStore{bindings: map[[2]string][]identity.Binding{}, principals: map[principalKey]identity.PrincipalRecord{}}
}

func (m *memStore) bind(issuer, orgID string, b identity.Binding) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := [2]string{issuer, orgID}
	m.bindings[k] = append(m.bindings[k], b)
}

func (m *memStore) addPrincipal(tenant uuid.UUID, subject string, p identity.PrincipalRecord) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.principals[principalKey{tenant, subject}] = p
}

func (m *memStore) Binding(_ context.Context, issuer, orgID string) (identity.Binding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return identity.Binding{}, m.fail
	}
	rows := m.bindings[[2]string{issuer, orgID}]
	switch len(rows) {
	case 0:
		return identity.Binding{}, identity.ErrNoBinding
	case 1:
		return rows[0], nil
	}
	return identity.Binding{}, errors.New("memstore: more than one binding")
}

func (m *memStore) Principal(_ context.Context, tenant uuid.UUID, subject string) (identity.PrincipalRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return identity.PrincipalRecord{}, m.fail
	}
	p, ok := m.principals[principalKey{tenant, subject}]
	if !ok {
		return identity.PrincipalRecord{}, identity.ErrNoPrincipal
	}
	return p, nil
}

// The fixture every resolution test starts from: the harness's default
// tenant token (org 0b6f9a8e-org-a, sub ...0001) bound to tenantA with IdP
// alias idp-a, and its principal synced with no federated links.
//
//nolint:gochecknoglobals // fixed test ids, never written
var (
	tenantA    = uuid.MustParse("0a000000-0000-4000-8000-00000000000a")
	tenantB    = uuid.MustParse("0b000000-0000-4000-8000-00000000000b")
	principalA = uuid.MustParse("0a000000-0000-4000-8000-0000000003a1")
	groupA     = uuid.MustParse("0a000000-0000-4000-8000-0000000004a1")
)

const (
	defaultSub = "6f1c2a7e-0000-4000-8000-000000000001"
	defaultOrg = "0b6f9a8e-org-a"
)

func bindDefault(h *harness) {
	h.store.bind(h.idp.issuer(realmNava), defaultOrg, identity.Binding{TenantID: tenantA, IdPAliases: []string{"idp-a"}})
	h.store.addPrincipal(tenantA, defaultSub, identity.PrincipalRecord{
		ID: principalA, Enabled: true, LinksKnown: true, FederatedIdPs: []string{}, Groups: []uuid.UUID{groupA},
	})
}

func wantUnresolved(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, identity.ErrTenantUnresolved) || errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("err = %v, want only ErrTenantUnresolved", err)
	}
}

func TestResolveHappyPath(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{realmNava})
	bindDefault(h)
	s := h.idp.signer(realmNava)

	got, err := h.a.Resolve(t.Context(), sign(t, s, h.idp.tenantClaims(realmNava, h.clock.now())))
	if err != nil {
		t.Fatal(err)
	}
	want := identity.Principal{TenantID: tenantA, PrincipalID: principalA, Groups: []uuid.UUID{groupA}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("local login: %+v, want %+v", got, want)
	}

	// A brokered login through the bound IdP, with its link recorded.
	h.store.addPrincipal(tenantA, defaultSub, identity.PrincipalRecord{
		ID: principalA, Enabled: true, LinksKnown: true, FederatedIdPs: []string{"idp-a"},
	})
	c := h.idp.tenantClaims(realmNava, h.clock.now())
	c["idp"] = "idp-a"
	got, err = h.a.Resolve(t.Context(), sign(t, s, c))
	if err != nil {
		t.Fatal(err)
	}
	want = identity.Principal{TenantID: tenantA, PrincipalID: principalA, Groups: []uuid.UUID{}, IdPAlias: "idp-a"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("brokered login: %+v, want %+v", got, want)
	}
}

// TestResolveRefusals is the A.1 steps 2-4 matrix: each case is the happy
// path with one thing changed, and each must be 401 tenant_unresolved.
func TestResolveRefusals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		claims func(c claims)
		store  func(h *harness)
	}{
		{name: "zero organizations", claims: func(c claims) { delete(c, "organization") }},
		{name: "empty organization object", claims: func(c claims) { c["organization"] = map[string]any{} }},
		{name: "two organizations", claims: func(c claims) {
			c["organization"] = map[string]any{"tenant-a": map[string]any{"id": defaultOrg}, "tenant-b": map[string]any{"id": "org-b"}}
		}},
		{name: "organization without id", claims: func(c claims) { c["organization"] = map[string]any{"tenant-a": map[string]any{}} }},
		{name: "organization as a list", claims: func(c claims) { c["organization"] = []string{"tenant-a"} }},
		{name: "unbound organization", claims: func(c claims) {
			c["organization"] = map[string]any{"tenant-x": map[string]any{"id": "org-x"}}
		}},
		{name: "unbound idp claim", claims: func(c claims) { c["idp"] = "idp-evil" }},
		{name: "idp claim of another tenant", claims: func(c claims) { c["idp"] = "idp-b" }},
		{name: "empty idp claim", claims: func(c claims) { c["idp"] = "" }},
		{name: "idp claim not a string", claims: func(c claims) { c["idp"] = []string{"idp-a"} }},
		{name: "federated link to a foreign IdP", store: func(h *harness) {
			h.store.addPrincipal(tenantA, defaultSub, identity.PrincipalRecord{ID: principalA, Enabled: true, LinksKnown: true, FederatedIdPs: []string{"idp-b"}})
		}},
		{name: "federated link to a foreign IdP beside a bound one", claims: func(c claims) { c["idp"] = "idp-a" }, store: func(h *harness) {
			h.store.addPrincipal(tenantA, defaultSub, identity.PrincipalRecord{ID: principalA, Enabled: true, LinksKnown: true, FederatedIdPs: []string{"idp-a", "idp-b"}})
		}},
		// TICK-69 security review carry-forward: NULL federated_links means
		// unknown, never "no links".
		{name: "federated links NULL (unknown)", store: func(h *harness) {
			h.store.addPrincipal(tenantA, defaultSub, identity.PrincipalRecord{ID: principalA, Enabled: true, LinksKnown: false})
		}},
		{name: "federated links NULL with a bound idp claim", claims: func(c claims) { c["idp"] = "idp-a" }, store: func(h *harness) {
			h.store.addPrincipal(tenantA, defaultSub, identity.PrincipalRecord{ID: principalA, Enabled: true, LinksKnown: false})
		}},
		// TICK-69 carry-forward: no principals row for the subject.
		{name: "no principal for the subject", claims: func(c claims) { c["sub"] = "kc-user-never-synced" }},
		{name: "principal only in another tenant", claims: func(c claims) { c["sub"] = "kc-user-of-b" }, store: func(h *harness) {
			h.store.addPrincipal(tenantB, "kc-user-of-b", identity.PrincipalRecord{ID: principalA, Enabled: true, LinksKnown: true})
		}},
		{name: "principal disabled", store: func(h *harness) {
			h.store.addPrincipal(tenantA, defaultSub, identity.PrincipalRecord{ID: principalA, Enabled: false, LinksKnown: true})
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, []string{realmNava})
			bindDefault(h)
			if c.store != nil {
				c.store(h)
			}
			cl := h.idp.tenantClaims(realmNava, h.clock.now())
			if c.claims != nil {
				c.claims(cl)
			}
			got, err := h.a.Resolve(t.Context(), sign(t, h.idp.signer(realmNava), cl))
			wantUnresolved(t, err)
			if !reflect.DeepEqual(got, identity.Principal{}) {
				t.Errorf("refused token returned %+v", got)
			}
		})
	}
}

// A tenant comes only from the binding of the token's own (iss, org id): the
// same org id under another issuer is not that tenant.
func TestResolveBindingIsPerIssuer(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{realmNava, "other"})
	bindDefault(h)
	c := h.idp.tenantClaims("other", h.clock.now())
	_, err := h.a.Resolve(t.Context(), sign(t, h.idp.signer("other"), c))
	wantUnresolved(t, err)
}

// More than one binding row is an error (A.1 step 4), never a pick. UNIQUE
// (issuer, org_id) makes it a broken database, so it is a 500, not a 401.
func TestResolveTwoBindingsRefused(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{realmNava})
	bindDefault(h)
	h.store.bind(h.idp.issuer(realmNava), defaultOrg, identity.Binding{TenantID: tenantB})
	got, err := h.a.Resolve(t.Context(), sign(t, h.idp.signer(realmNava), h.idp.tenantClaims(realmNava, h.clock.now())))
	if err == nil || !reflect.DeepEqual(got, identity.Principal{}) {
		t.Fatalf("two bindings: %+v, %v; want an error and no principal", got, err)
	}
}

// A store failure is neither sentinel (a 500, not a 401), and its error text
// reaches the caller only wrapped, never a claim value.
func TestResolveStoreFailure(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{realmNava})
	bindDefault(h)
	h.store.fail = errors.New("connection refused")
	_, err := h.a.Resolve(t.Context(), sign(t, h.idp.signer(realmNava), h.idp.tenantClaims(realmNava, h.clock.now())))
	if err == nil || errors.Is(err, identity.ErrInvalidToken) || errors.Is(err, identity.ErrTenantUnresolved) {
		t.Fatalf("err = %v, want a non-sentinel error", err)
	}
}

// Refusal reasons are fixed text: no claim value (org id, subject, idp alias)
// appears in the error.
func TestResolveErrorsQuoteNoClaim(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{realmNava})
	bindDefault(h)
	s := h.idp.signer(realmNava)
	for _, edit := range []func(c claims){
		func(c claims) { c["idp"] = "idp-secret-alias" },
		func(c claims) { c["sub"] = "secret-subject" },
		func(c claims) { c["organization"] = map[string]any{"secret-alias": map[string]any{"id": "secret-org"}} },
	} {
		c := h.idp.tenantClaims(realmNava, h.clock.now())
		edit(c)
		_, err := h.a.Resolve(t.Context(), sign(t, s, c))
		if err == nil || strings.Contains(err.Error(), "secret") {
			t.Errorf("err = %v: must be a fixed reason", err)
		}
	}
}
