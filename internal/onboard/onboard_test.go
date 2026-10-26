package onboard

import (
	"context"
	"errors"
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
	failOnce  error
	lockHeld  map[uuid.UUID]*sync.Mutex
}

func newStore() *fakeStore {
	return &fakeStore{rows: map[uuid.UUID][]byte{}, active: map[uuid.UUID]bool{}, bindings: map[uuid.UUID]string{}, lockHeld: map[uuid.UUID]*sync.Mutex{}}
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

func (s *fakeStore) Begin(_ context.Context, id uuid.UUID, hash []byte) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if h, ok := s.rows[id]; ok {
		if string(h) != string(hash) {
			return State{}, ErrKeyReuse
		}
		return State{Active: s.active[id]}, nil
	}
	s.rows[id] = hash
	return State{}, nil
}

func (s *fakeStore) Complete(_ context.Context, id uuid.UUID, _, orgID, _ string, _ []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failOnce != nil {
		err := s.failOnce
		s.failOnce = nil
		return err
	}
	s.bindings[id] = orgID
	s.active[id] = true
	s.completes++
	return nil
}

type fakeDir struct {
	mu        sync.Mutex
	orgs      map[uuid.UUID]string
	creates   int
	invites   []string
	inviteErr error
	specSeen  keycloak.IdPSpec
	specs     *Specs
}

func (d *fakeDir) CreateTenantOrg(ctx context.Context, id uuid.UUID, _ string, _ []string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
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
var okResolver = fakeResolver{"idp.example.org": {"203.0.113.10"}, "internal.example.org": {"10.0.0.5"}, "meta.example.org": {"169.254.169.254"}}

func newSvc(t *testing.T) (*Service, *fakeStore, *fakeDir) {
	t.Helper()
	st := newStore()
	specs := NewSpecs()
	dir := &fakeDir{orgs: map[uuid.UUID]string{}, specs: specs}
	svc, err := New(Options{Store: st, Directory: dir, Finder: dir, Specs: specs, Resolver: okResolver,
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

func TestOnboardReplayInvitesAgainAndCreatesNothing(t *testing.T) {
	svc, _, dir := newSvc(t)
	if _, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq()); err != nil {
		t.Fatal(err)
	}
	res, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq())
	if err != nil || !res.Replayed || dir.creates != 1 {
		t.Fatalf("replay: %+v, %v, creates=%d", res, err, dir.creates)
	}
}

func TestOnboardSameKeyDifferentBodyIsReuseAndCreatesNothing(t *testing.T) {
	svc, st, dir := newSvc(t)
	if _, err := svc.Onboard(t.Context(), op1, "k", []byte("h1"), validReq()); err != nil {
		t.Fatal(err)
	}
	creates, completes := dir.creates, st.completes
	_, err := svc.Onboard(t.Context(), op1, "k", []byte("h2"), validReq())
	if !errors.Is(err, ErrKeyReuse) || dir.creates != creates || st.completes != completes {
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
	r2, err := svc.Onboard(t.Context(), identity.Operator{PrincipalID: "op-2"}, "shared", []byte("h"), validReq())
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

func TestOnboardInviteFailureIsAnErrorAndRetryInvites(t *testing.T) {
	svc, _, dir := newSvc(t)
	dir.inviteErr = errors.New("keycloak down")
	if _, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq()); err == nil {
		t.Fatal("want error")
	}
	dir.inviteErr = nil
	if _, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq()); err != nil {
		t.Fatal(err)
	}
	if len(dir.invites) != 2 {
		t.Errorf("invites = %d, want 2", len(dir.invites))
	}
}

func TestOnboardDomainClaimed(t *testing.T) {
	svc, st, _ := newSvc(t)
	st.failOnce = ErrDomainClaimed
	_, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq())
	if !errors.Is(err, ErrDomainClaimed) {
		t.Fatalf("err = %v", err)
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
		"userinfo in url":       {func(r *Request) { r.IdP.Config["tokenUrl"] = "https://u:p@idp.example.org/t" }, "/identity_provider/config"},
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
