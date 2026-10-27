package onboard

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
)

type fakeStore struct {
	mu        sync.Mutex
	rows      map[uuid.UUID][]byte
	active    map[uuid.UUID]bool
	bindings  map[uuid.UUID]string
	completes int
	binds     int
	domains   map[string]uuid.UUID
	failOnce  error
	lockHeld  map[uuid.UUID]*sync.Mutex
}

func newStore() *fakeStore {
	return &fakeStore{rows: map[uuid.UUID][]byte{}, active: map[uuid.UUID]bool{}, bindings: map[uuid.UUID]string{}, lockHeld: map[uuid.UUID]*sync.Mutex{}, domains: map[string]uuid.UUID{}}
}

func (s *fakeStore) Lock(_ context.Context, id uuid.UUID) (func(), error) {
	s.mu.Lock()
	m := s.lockHeld[id]
	if m == nil {
		m = &sync.Mutex{}
		s.lockHeld[id] = m
	}
	s.mu.Unlock()
	m.Lock()
	return m.Unlock, nil
}

func (s *fakeStore) Begin(_ context.Context, id uuid.UUID, hash []byte, domains []string) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if h, ok := s.rows[id]; ok {
		if string(h) != string(hash) {
			return State{}, ErrKeyReuse
		}
		return State{Active: s.active[id]}, nil
	}
	for _, d := range domains {
		if owner, taken := s.domains[d]; taken && owner != id {
			return State{}, ErrDomainClaimed
		}
	}
	for _, d := range domains {
		s.domains[d] = id
	}
	s.rows[id] = hash
	return State{}, nil
}

func (s *fakeStore) Bind(_ context.Context, id uuid.UUID, _, orgID, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failOnce != nil {
		err := s.failOnce
		s.failOnce = nil
		return err
	}
	s.bindings[id] = orgID
	s.binds++
	return nil
}

func (s *fakeStore) Activate(_ context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active[id] {
		return ErrNotProvisioning
	}
	s.active[id] = true
	s.completes++
	return nil
}

// TenantOwnsEmailDomain is the fake nava_app read: only the tenant's own row.
func (s *fakeStore) TenantOwnsEmailDomain(_ context.Context, id uuid.UUID, d string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.domains[d] == id, nil
}

type fakeDir struct {
	mu        sync.Mutex
	orgs      map[uuid.UUID]string
	creates   int
	invites   []string
	inviteErr error
	createErr error
	specSeen  keycloak.IdPSpec
	specs     *Specs
}

func (d *fakeDir) CreateTenantOrg(ctx context.Context, id uuid.UUID, _ string, _ []string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.createErr != nil {
		return "", d.createErr
	}
	if _, ok := d.orgs[id]; ok {
		return "", identity.ErrAlreadyExists
	}
	spec, err := d.specs.Source(ctx, id)
	if err != nil {
		return "", err
	}
	d.specSeen = spec
	d.creates++
	d.orgs[id] = "org-" + id.String()
	return d.orgs[id], nil
}

func (d *fakeDir) InviteUser(_ context.Context, _ uuid.UUID, email string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.invites = append(d.invites, email)
	return d.inviteErr
}

func (d *fakeDir) FindTenantOrg(_ context.Context, id uuid.UUID, _ string) (string, bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	o, ok := d.orgs[id]
	return o, ok, nil
}

type fakeResolver map[string][]string

func (r fakeResolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	var out []netip.Addr
	for _, a := range r[host] {
		out = append(out, netip.MustParseAddr(a))
	}
	if len(out) == 0 {
		return nil, errors.New("no such host")
	}
	return out, nil
}

//nolint:gochecknoglobals // test fixtures
var okResolver = fakeResolver{"idp.example.org": {"93.184.216.34"}, "internal.example.org": {"10.0.0.5"}, "meta.example.org": {"169.254.169.254"}}

func newSvc(t *testing.T) (*Service, *fakeStore, *fakeDir) {
	t.Helper()
	st := newStore()
	specs := NewSpecs()
	dir := &fakeDir{orgs: map[uuid.UUID]string{}, specs: specs}
	svc, err := New(Options{Store: st, Directory: dir, Finder: dir, Domains: st, Specs: specs, Resolver: okResolver,
		Issuer: "https://kc/realms/nava", Log: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatal(err)
	}
	return svc, st, dir
}

func validReq() Request {
	return Request{
		AdminEmail:   "admin@acme.test",
		EmailDomains: []string{"acme.test"},
		IdP: IdPRequest{ProviderID: "oidc", DisplayName: "Acme", Config: map[string]string{
			"clientId": "c", "clientSecret": "s", "tokenUrl": "https://idp.example.org/token",
			"authorizationUrl": "https://idp.example.org/auth", "jwksUrl": "https://idp.example.org/jwks",
		}},
	}
}

//nolint:gochecknoglobals // test fixtures
var op1 = identity.Operator{PrincipalID: "op-1"}

func TestOnboardCreatesAndActivates(t *testing.T) {
	svc, st, dir := newSvc(t)
	res, err := svc.Onboard(t.Context(), op1, "key-1", []byte("h"), validReq())
	if err != nil {
		t.Fatal(err)
	}
	if res.TenantID != TenantID("op-1", "key-1") || res.Replayed {
		t.Errorf("result = %+v", res)
	}
	if !st.active[res.TenantID] || st.bindings[res.TenantID] == "" || dir.creates != 1 || len(dir.invites) != 1 {
		t.Errorf("active=%v binding=%q creates=%d invites=%v", st.active[res.TenantID], st.bindings[res.TenantID], dir.creates, dir.invites)
	}
	if dir.specSeen.Config["validateSignature"] != "true" || dir.specSeen.Config["syncMode"] != "IMPORT" {
		t.Errorf("spec not pinned: %v", dir.specSeen.Config)
	}
}

func TestOnboardReplayOfActiveTenantChangesNothing(t *testing.T) {
	svc, _, dir := newSvc(t)
	if _, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq()); err != nil {
		t.Fatal(err)
	}
	res, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq())
	if err != nil || !res.Replayed || dir.creates != 1 || len(dir.invites) != 1 {
		t.Fatalf("replay: %+v, %v, creates=%d invites=%d", res, err, dir.creates, len(dir.invites))
	}
}

func TestOnboardSameKeyDifferentBodyIsReuseAndCreatesNothing(t *testing.T) {
	svc, st, dir := newSvc(t)
	if _, err := svc.Onboard(t.Context(), op1, "k", []byte("h1"), validReq()); err != nil {
		t.Fatal(err)
	}
	creates, completes := dir.creates, st.completes
	_, err := svc.Onboard(t.Context(), op1, "k", []byte("h2"), validReq())
	if !errors.Is(err, ErrKeyReuse) || dir.creates != creates || st.completes != completes || st.binds != 1 {
		t.Fatalf("err=%v creates %d->%d completes %d->%d", err, creates, dir.creates, completes, st.completes)
	}
}

// A failure between org creation and the binding insert leaves the tenant
// provisioning; the retry reuses the one org and writes the one binding.
func TestOnboardResumesAfterFailureBetweenOrgAndBinding(t *testing.T) {
	svc, st, dir := newSvc(t)
	st.failOnce = errors.New("killed")
	if _, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq()); err == nil {
		t.Fatal("first call should fail")
	}
	id := TenantID("op-1", "k")
	if st.active[id] || len(dir.orgs) != 1 {
		t.Fatalf("after failure: active=%v orgs=%d", st.active[id], len(dir.orgs))
	}
	if _, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq()); err != nil {
		t.Fatal(err)
	}
	if dir.creates != 1 || len(dir.orgs) != 1 || st.completes != 1 || !st.active[id] {
		t.Errorf("creates=%d orgs=%d completes=%d active=%v", dir.creates, len(dir.orgs), st.completes, st.active[id])
	}
}

// ADR-001-E3: operator 2 replaying operator 1's key gets another tenant id and
// never touches operator 1's tenant.
func TestOnboardOperatorsDoNotShareTenants(t *testing.T) {
	svc, st, _ := newSvc(t)
	r1, err := svc.Onboard(t.Context(), op1, "shared", []byte("h"), validReq())
	if err != nil {
		t.Fatal(err)
	}
	req2 := validReq()
	req2.AdminEmail, req2.EmailDomains = "admin@other.test", []string{"other.test"}
	r2, err := svc.Onboard(t.Context(), identity.Operator{PrincipalID: "op-2"}, "shared", []byte("h"), req2)
	if err != nil {
		t.Fatal(err)
	}
	if r1.TenantID == r2.TenantID || r2.Replayed {
		t.Fatalf("tenant ids %v %v replayed=%v", r1.TenantID, r2.TenantID, r2.Replayed)
	}
	if st.bindings[r1.TenantID] == st.bindings[r2.TenantID] {
		t.Error("tenants share an org")
	}
}

func TestOnboardInviteRefusedLooksLikeSuccess(t *testing.T) {
	svc, _, dir := newSvc(t)
	dir.inviteErr = identity.ErrInviteRefused
	res, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq())
	if err != nil || res.Replayed {
		t.Fatalf("refused invite: %+v, %v", res, err)
	}
}

// Activation is the last step: a failed invite leaves the tenant
// provisioning, and the retry completes it. A replay that fails again is an
// error, never a 200.
func TestOnboardInviteFailureLeavesProvisioningAndRetryCompletes(t *testing.T) {
	svc, st, dir := newSvc(t)
	id := TenantID("op-1", "k")
	dir.inviteErr = errors.New("keycloak down")
	if _, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq()); err == nil {
		t.Fatal("want error")
	}
	if st.active[id] {
		t.Fatal("tenant is active after a failed invite")
	}
	if _, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq()); err == nil {
		t.Fatal("a replay whose invite fails again must be an error")
	}
	if st.active[id] {
		t.Fatal("tenant is active after a second failed invite")
	}
	dir.inviteErr = nil
	res, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq())
	if err != nil || res.Replayed || !st.active[id] {
		t.Fatalf("retry: %+v, %v, active=%v", res, err, st.active[id])
	}
	if dir.creates != 1 || len(dir.invites) != 3 {
		t.Errorf("creates=%d invites=%d, want 1 and 3", dir.creates, len(dir.invites))
	}
}

// The domain is owned by another tenant: Bind skips it, the ownership read
// finds none of the tenant's own, nothing is invited and the tenant stays
// provisioning.
func TestOnboardDomainOwnedByAnotherTenant(t *testing.T) {
	svc, st, dir := newSvc(t)
	st.domains["acme.test"] = uuid.New()
	_, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq())
	if !errors.Is(err, ErrDomainClaimed) || len(dir.invites) != 0 || st.active[TenantID("op-1", "k")] {
		t.Fatalf("err=%v invites=%d", err, len(dir.invites))
	}
}

func TestOnboardBindConflict(t *testing.T) {
	svc, st, dir := newSvc(t)
	st.failOnce = ErrConflict
	_, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq())
	if !errors.Is(err, ErrConflict) || len(dir.invites) != 0 || st.active[TenantID("op-1", "k")] {
		t.Fatalf("err=%v invites=%d", err, len(dir.invites))
	}
}

func TestOnboardConcurrentSameTenantCreatesOneOrg(t *testing.T) {
	svc, _, dir := newSvc(t)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if _, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq()); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if dir.creates != 1 || len(dir.orgs) != 1 {
		t.Errorf("creates=%d orgs=%d, want 1", dir.creates, len(dir.orgs))
	}
}

func TestOnboardRejectsInvalidRequests(t *testing.T) {
	cases := map[string]struct {
		mut  func(*Request)
		want string
	}{
		"public domain":         {func(r *Request) { r.EmailDomains = []string{"gmail.com"}; r.AdminEmail = "a@gmail.com" }, "/email_domains"},
		"public domain upper":   {func(r *Request) { r.EmailDomains = []string{"GMAIL.com"}; r.AdminEmail = "a@gmail.com" }, "/email_domains"},
		"no domains":            {func(r *Request) { r.EmailDomains = nil }, "/email_domains"},
		"bad domain":            {func(r *Request) { r.EmailDomains = []string{"nodot"} }, "/email_domains"},
		"admin outside domains": {func(r *Request) { r.AdminEmail = "a@other.test" }, "/admin_email"},
		"bad admin":             {func(r *Request) { r.AdminEmail = "not an email" }, "/admin_email"},
		"bad provider":          {func(r *Request) { r.IdP.ProviderID = "ldap" }, "/identity_provider/provider_id"},
		"http url":              {func(r *Request) { r.IdP.Config["tokenUrl"] = "http://idp.example.org/token" }, "/identity_provider/config"},
		"private host":          {func(r *Request) { r.IdP.Config["jwksUrl"] = "https://internal.example.org/jwks" }, "/identity_provider/config"},
		"metadata ip":           {func(r *Request) { r.IdP.Config["authorizationUrl"] = "https://meta.example.org/x" }, "/identity_provider/config"},
		"literal loopback":      {func(r *Request) { r.IdP.Config["tokenUrl"] = "https://127.0.0.1/token" }, "/identity_provider/config"},
		"literal link-local":    {func(r *Request) { r.IdP.Config["tokenUrl"] = "https://[fe80::1]/token" }, "/identity_provider/config"},
		"unresolvable":          {func(r *Request) { r.IdP.Config["tokenUrl"] = "https://nope.example.org/t" }, "/identity_provider/config"},
		"localhost":             {func(r *Request) { r.IdP.Config["tokenUrl"] = "https://localhost/t" }, "/identity_provider/config"},
		"alias key":             {func(r *Request) { r.IdP.Config["alias"] = "idp-other-tenant" }, "/identity_provider/config"},
		"saml artifact url": {func(r *Request) {
			r.IdP.ProviderID = "saml"
			r.IdP.Config = map[string]string{"artifactResolutionServiceUrl": "https://internal.example.org/x"}
		}, "/identity_provider/config"},
		"unknown key":      {func(r *Request) { r.IdP.Config["kc.org.broker.redirect.mode.email-matches"] = "true" }, "/identity_provider/config"},
		"cgnat":            {func(r *Request) { r.IdP.Config["tokenUrl"] = "https://100.64.0.1/t" }, "/identity_provider/config"},
		"nat64 private v4": {func(r *Request) { r.IdP.Config["tokenUrl"] = "https://[64:ff9b::a00:1]/t" }, "/identity_provider/config"},
		"benchmark net":    {func(r *Request) { r.IdP.Config["tokenUrl"] = "https://198.18.0.1/t" }, "/identity_provider/config"},
		"zero net":         {func(r *Request) { r.IdP.Config["tokenUrl"] = "https://0.1.2.3/t" }, "/identity_provider/config"},
		"userinfo in url":  {func(r *Request) { r.IdP.Config["tokenUrl"] = "https://u:p@idp.example.org/t" }, "/identity_provider/config"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			svc, st, dir := newSvc(t)
			req := validReq()
			c.mut(&req)
			_, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), req)
			var fe *FieldError
			if !errors.As(err, &fe) || fe.Pointer != c.want || !errors.Is(err, ErrInvalid) {
				t.Fatalf("err = %v, want field %s", err, c.want)
			}
			if len(st.rows) != 0 || dir.creates != 0 {
				t.Error("something was created for an invalid request")
			}
		})
	}
}

func TestOnboardOverridesSignatureAndSyncMode(t *testing.T) {
	svc, _, dir := newSvc(t)
	req := validReq()
	req.IdP.Config["syncMode"] = "FORCE"
	req.IdP.Config["validateSignature"] = "false"
	if _, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), req); err != nil {
		t.Fatal(err)
	}
	if c := dir.specSeen.Config; c["syncMode"] != "IMPORT" || c["validateSignature"] != "true" {
		t.Errorf("config = %v", c)
	}
}

func TestTenantIDIsStableAndPerOperator(t *testing.T) {
	a := TenantID("a", "k")
	if a != TenantID("a", "k") || TenantID("a", "k") == TenantID("b", "k") || TenantID("a", "k") == TenantID("a", "j") {
		t.Fatal("tenant id derivation")
	}
	if TenantID("ab", "c") == TenantID("a", "bc") {
		t.Fatal("principal and key run together")
	}
	if got := TenantID("a", "k").Version(); got != 5 {
		t.Errorf("version = %d, want 5", got)
	}
}

func TestSpecsSourceWithoutSpecFails(t *testing.T) {
	if _, err := NewSpecs().Source(t.Context(), uuid.New()); err == nil {
		t.Fatal("want error")
	}
}

// Keycloak refuses an organization with a linked domain before Bind runs: the
// service answers ErrDomainClaimed, not an opaque create error.
func TestOnboardKeycloakDomainLinkedIsDomainClaimed(t *testing.T) {
	svc, st, dir := newSvc(t)
	dir.createErr = fmt.Errorf("keycloak admin: create: %w", keycloak.ErrDomainLinked)
	_, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq())
	if !errors.Is(err, ErrDomainClaimed) {
		t.Fatalf("err = %v, want ErrDomainClaimed", err)
	}
	if len(st.bindings) != 0 || len(dir.invites) != 0 {
		t.Errorf("bindings=%v invites=%v after a refused onboarding", st.bindings, dir.invites)
	}
}

// Another tenant owns the domain: Begin refuses it, Keycloak is never called
// and no row is left for the refused tenant.
func TestOnboardRefusesADomainAnotherTenantOwnsBeforeKeycloak(t *testing.T) {
	svc, st, dir := newSvc(t)
	if _, err := svc.Onboard(t.Context(), op1, "a", []byte("h"), validReq()); err != nil {
		t.Fatal(err)
	}
	creates := dir.creates
	_, err := svc.Onboard(t.Context(), op1, "b", []byte("h2"), validReq())
	if !errors.Is(err, ErrDomainClaimed) {
		t.Fatalf("err = %v, want ErrDomainClaimed", err)
	}
	if _, left := st.rows[TenantID("op-1", "b")]; left || dir.creates != creates {
		t.Errorf("row left = %v, creates %d -> %d", left, creates, dir.creates)
	}
}
