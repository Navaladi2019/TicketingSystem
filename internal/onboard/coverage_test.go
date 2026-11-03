package onboard

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
)

func TestNewRequiresEveryOption(t *testing.T) {
	t.Parallel()
	st := newStore()
	specs := NewSpecs()
	dir := &fakeDir{orgs: map[uuid.UUID]string{}, specs: specs}
	full := Options{Store: st, Directory: dir, Domains: st, Specs: specs, Resolver: okResolver,
		Issuer: "https://kc/realms/nava", Log: slog.New(slog.DiscardHandler)}
	if _, err := New(full); err != nil {
		t.Fatal(err)
	}
	for name, mut := range map[string]func(*Options){
		"store":    func(o *Options) { o.Store = nil },
		"dir":      func(o *Options) { o.Directory = nil },
		"domains":  func(o *Options) { o.Domains = nil },
		"specs":    func(o *Options) { o.Specs = nil },
		"resolver": func(o *Options) { o.Resolver = nil },
		"log":      func(o *Options) { o.Log = nil },
		"issuer":   func(o *Options) { o.Issuer = "" },
	} {
		o := full
		mut(&o)
		if _, err := New(o); err == nil {
			t.Errorf("New without %s must fail", name)
		}
	}
}

func TestFieldErrorNamesThePointerNotTheValue(t *testing.T) {
	t.Parallel()
	var err error = &FieldError{Pointer: "/admin_email"}
	if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "/admin_email") {
		t.Fatalf("FieldError = %v", err)
	}
}

// errStore fails one step of the onboarding, so each step's error is returned
// and later steps never run.
type errStore struct {
	*fakeStore
	lockErr, beginErr, activateErr, domainErr error
	notOwner                                  bool
}

func (s *errStore) Lock(ctx context.Context, id uuid.UUID) (func(), error) {
	if s.lockErr != nil {
		return nil, s.lockErr
	}
	return s.fakeStore.Lock(ctx, id)
}

func (s *errStore) Begin(ctx context.Context, id uuid.UUID, h []byte, d []string) (State, error) {
	if s.beginErr != nil {
		return State{}, s.beginErr
	}
	return s.fakeStore.Begin(ctx, id, h, d)
}

func (s *errStore) Activate(ctx context.Context, id uuid.UUID) error {
	if s.activateErr != nil {
		return s.activateErr
	}
	return s.fakeStore.Activate(ctx, id)
}

func (s *errStore) TenantOwnsEmailDomain(ctx context.Context, id uuid.UUID, d string) (bool, error) {
	if s.domainErr != nil {
		return false, s.domainErr
	}
	if s.notOwner {
		return false, nil
	}
	return s.fakeStore.TenantOwnsEmailDomain(ctx, id, d)
}

func svcWith(t *testing.T) (*Service, *errStore, *fakeDir) {
	t.Helper()
	st := &errStore{fakeStore: newStore()}
	specs := NewSpecs()
	dir := &fakeDir{orgs: map[uuid.UUID]string{}, specs: specs}
	svc, err := New(Options{Store: st, Directory: dir, Finder: dir, Domains: st, Specs: specs, Resolver: okResolver,
		Issuer: "https://kc/realms/nava", Log: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatal(err)
	}
	return svc, st, dir
}

func TestOnboardStepFailuresStopTheRun(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	cases := []struct {
		name string
		set  func(*errStore, *fakeDir)
		want func(error) bool
	}{
		{"lock", func(s *errStore, _ *fakeDir) { s.lockErr = boom }, func(e error) bool { return errors.Is(e, boom) }},
		{"begin", func(s *errStore, _ *fakeDir) { s.beginErr = boom }, func(e error) bool { return errors.Is(e, boom) }},
		{"org exists elsewhere", func(_ *errStore, d *fakeDir) { d.createErr = identity.ErrAlreadyExists }, func(e error) bool { return errors.Is(e, ErrConflict) }},
		{"domain linked in keycloak", func(_ *errStore, d *fakeDir) { d.createErr = keycloak.ErrDomainLinked }, func(e error) bool { return errors.Is(e, ErrDomainClaimed) }},
		{"idp refused", func(_ *errStore, d *fakeDir) { d.createErr = identity.ErrInvalidInput }, func(e error) bool {
			var fe *FieldError
			return errors.As(e, &fe) && fe.Pointer == "/identity_provider"
		}},
		{"create fails", func(_ *errStore, d *fakeDir) { d.createErr = boom }, func(e error) bool { return errors.Is(e, boom) }},
		{"bind", func(s *errStore, _ *fakeDir) { s.failOnce = boom }, func(e error) bool { return errors.Is(e, boom) }},
		{"domain read", func(s *errStore, _ *fakeDir) { s.domainErr = boom }, func(e error) bool { return errors.Is(e, boom) }},
		{"domain not owned", func(s *errStore, _ *fakeDir) { s.notOwner = true }, func(e error) bool { return errors.Is(e, ErrDomainClaimed) }},
		{"seed", func(s *errStore, _ *fakeDir) { s.seedErr = boom }, func(e error) bool { return errors.Is(e, boom) }},
		{"admin group", func(_ *errStore, d *fakeDir) { d.groupErr = boom }, func(e error) bool { return errors.Is(e, boom) }},
		{"record admin group", func(s *errStore, _ *fakeDir) { s.adminErr = boom }, func(e error) bool { return errors.Is(e, boom) }},
		{"record admin group missing", func(s *errStore, _ *fakeDir) { s.adminErr = ErrAdminGroupMissing }, func(e error) bool { return errors.Is(e, ErrAdminGroupMissing) }},
		{"invite", func(_ *errStore, d *fakeDir) { d.inviteErr = boom }, func(e error) bool { return errors.Is(e, boom) }},
		{"activate", func(s *errStore, _ *fakeDir) { s.activateErr = boom }, func(e error) bool { return errors.Is(e, boom) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			svc, st, dir := svcWith(t)
			c.set(st, dir)
			res, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq())
			if err == nil || !c.want(err) {
				t.Fatalf("err = %v", err)
			}
			if res.TenantID != uuid.Nil || st.active[TenantID("op-1", "k")] {
				t.Fatalf("a failed run returned or activated a tenant: %+v", res)
			}
		})
	}
}

// A finder that fails is an error, never "no org yet": creating a second
// organization on top of an unknown one would orphan the first.
type failingFinder struct{}

func (failingFinder) FindTenantOrg(context.Context, uuid.UUID, string) (string, bool, error) {
	return "", false, errors.New("keycloak down")
}

func TestOnboardFinderFailureCreatesNothing(t *testing.T) {
	t.Parallel()
	st := newStore()
	specs := NewSpecs()
	dir := &fakeDir{orgs: map[uuid.UUID]string{}, specs: specs}
	svc, err := New(Options{Store: st, Directory: dir, Finder: failingFinder{}, Domains: st, Specs: specs, Resolver: okResolver,
		Issuer: "https://kc/realms/nava", Log: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq()); err == nil || !strings.Contains(err.Error(), "find org") {
		t.Fatalf("err = %v", err)
	}
	if dir.creates != 0 {
		t.Fatalf("created %d organizations after a finder failure", dir.creates)
	}
}

// Another tenant's domain is refused for this operator and nothing is created
// for the second tenant; the first tenant's rows are untouched.
func TestOnboardDomainOwnedByAnotherTenantIsRefused(t *testing.T) {
	t.Parallel()
	svc, st, dir := newSvc(t)
	first, err := svc.Onboard(t.Context(), op1, "k", []byte("h"), validReq())
	if err != nil {
		t.Fatal(err)
	}
	creates := dir.creates
	_, err = svc.Onboard(t.Context(), identity.Operator{PrincipalID: "op-2"}, "k", []byte("h"), validReq())
	if !errors.Is(err, ErrDomainClaimed) {
		t.Fatalf("err = %v", err)
	}
	if dir.creates != creates || st.domains["acme.test"] != first.TenantID {
		t.Fatalf("creates %d->%d, domain owner %v", creates, dir.creates, st.domains["acme.test"])
	}
}

func TestBuildSpecRefusals(t *testing.T) {
	t.Parallel()
	base := func() IdPRequest {
		return IdPRequest{ProviderID: "oidc", DisplayName: "Acme", Config: map[string]string{"clientId": "c"}}
	}
	many := map[string]string{}
	for i := range maxConfigKeys + 1 {
		many[strings.Repeat("k", i+1)] = "v"
	}
	for name, mut := range map[string]func(*IdPRequest){
		"provider":     func(r *IdPRequest) { r.ProviderID = "ldap" },
		"display name": func(r *IdPRequest) { r.DisplayName = strings.Repeat("x", maxDisplayName+1) },
		"too many":     func(r *IdPRequest) { r.Config = many },
		"unknown key":  func(r *IdPRequest) { r.Config = map[string]string{"alias": "evil"} },
		"policy key":   func(r *IdPRequest) { r.Config = map[string]string{"kc.org.domain": "x"} },
		"long value":   func(r *IdPRequest) { r.Config = map[string]string{"clientId": strings.Repeat("v", maxConfigValue+1)} },
		"http url":     func(r *IdPRequest) { r.Config = map[string]string{"tokenUrl": "http://idp.example.org/t"} },
		"private host": func(r *IdPRequest) { r.Config = map[string]string{"tokenUrl": "https://internal.example.org/t"} },
		"metadata ip":  func(r *IdPRequest) { r.Config = map[string]string{"tokenUrl": "https://169.254.169.254/t"} },
		"unresolved":   func(r *IdPRequest) { r.Config = map[string]string{"tokenUrl": "https://nowhere.example.org/t"} },
		"localhost":    func(r *IdPRequest) { r.Config = map[string]string{"tokenUrl": "https://localhost/t"} },
		"userinfo":     func(r *IdPRequest) { r.Config = map[string]string{"tokenUrl": "https://u:p@idp.example.org/t"} },
		"nat64 to private": func(r *IdPRequest) {
			r.Config = map[string]string{"tokenUrl": "https://[64:ff9b::a00:1]/t"}
		},
		"reserved range": func(r *IdPRequest) { r.Config = map[string]string{"tokenUrl": "https://198.51.100.7/t"} },
	} {
		r := base()
		mut(&r)
		if _, ptr := buildSpec(t.Context(), okResolver, r); ptr == "" {
			t.Errorf("%s: want a refusal", name)
		}
	}
	// Pinned values win over the request.
	r := base()
	r.Config["validateSignature"] = "false"
	r.Config["syncMode"] = "FORCE"
	spec, ptr := buildSpec(t.Context(), okResolver, r)
	if ptr != "" || spec.Config["validateSignature"] != "true" || spec.Config["syncMode"] != "IMPORT" {
		t.Fatalf("spec = %+v, %q", spec, ptr)
	}
}

func TestNetResolverResolvesLocalhost(t *testing.T) {
	t.Parallel()
	addrs, err := NewNetResolver().LookupNetIP(t.Context(), "ip", "localhost")
	if err != nil || len(addrs) == 0 {
		t.Fatalf("LookupNetIP = %v, %v", addrs, err)
	}
	for _, a := range addrs {
		if !a.IsLoopback() {
			t.Errorf("localhost resolved to %v", a)
		}
	}
}

// ---- PGStore and Registry on a database that is not there ----------------
//
// Every method must return its error, wrapped, and never a zero value that
// reads as "no row". The success paths run in the database tests.

func deadPool(t *testing.T, maxConns int32) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig("postgres://nava_onboard:x@127.0.0.1:1/nava?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = maxConns
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

type noBindings struct{}

func (noBindings) Binding(context.Context, uuid.UUID) (string, string, bool, error) {
	return "", "", false, errors.New("binding read failed")
}

func TestNewPGStoreValidation(t *testing.T) {
	t.Parallel()
	if _, err := NewPGStore(deadPool(t, 4), nil); err == nil {
		t.Error("NewPGStore without a binding reader must fail")
	}
	if _, err := NewPGStore(deadPool(t, 1), noBindings{}); err == nil {
		t.Error("NewPGStore on a one-connection pool must fail: the lock would starve the statements")
	}
	if _, err := NewPGStore(deadPool(t, 4), noBindings{}); err != nil {
		t.Errorf("NewPGStore: %v", err)
	}
}

func TestPGStoreLockHonoursContextAndFreesTheSlot(t *testing.T) {
	t.Parallel()
	s, err := NewPGStore(deadPool(t, 2), noBindings{})
	if err != nil {
		t.Fatal(err)
	}
	tenant := uuid.New()
	// The database is down: Lock fails and gives its slot back, so a second
	// call is not stuck behind the first.
	for range 3 {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		release, err := s.Lock(ctx, tenant)
		cancel()
		if err == nil || release != nil {
			t.Fatalf("Lock on a dead database = %v, %v", release != nil, err)
		}
	}
	// All slots taken and the caller gives up: it does not wait forever.
	s.sem <- struct{}{}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := s.Lock(ctx, tenant); !errors.Is(err, context.Canceled) {
		t.Fatalf("Lock with a full semaphore and a cancelled context = %v", err)
	}
}

func TestPGStoreDatabaseErrorsAreWrappedNotAbsence(t *testing.T) {
	t.Parallel()
	s, err := NewPGStore(deadPool(t, 4), noBindings{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, tenant := t.Context(), uuid.New()
	check := func(name string, err error) {
		t.Helper()
		if err == nil {
			t.Errorf("%s: want an error from a dead database", name)
		}
	}
	if st, err := s.Begin(ctx, tenant, []byte("h"), []string{"a.test"}); err == nil || st.Active ||
		errors.Is(err, ErrKeyReuse) || errors.Is(err, ErrDomainClaimed) {
		t.Errorf("Begin = %+v, %v", st, err)
	}
	err = s.Bind(ctx, tenant, "iss", "org", "idp")
	check("Bind", err)
	if errors.Is(err, ErrConflict) || errors.Is(err, ErrNotProvisioning) {
		t.Errorf("Bind on a dead database must not read as a refusal: %v", err)
	}
	err = s.RecordAdminGroup(ctx, tenant, "kc-g")
	check("RecordAdminGroup", err)
	if errors.Is(err, ErrAdminGroupMissing) {
		t.Errorf("RecordAdminGroup: %v", err)
	}
	if ins, err := s.SeedEntityType(ctx, tenant, entitytypes.SeedInput{}); err == nil || ins {
		t.Errorf("SeedEntityType = %v, %v", ins, err)
	}
	err = s.Activate(ctx, tenant)
	if err == nil || errors.Is(err, ErrNotProvisioning) || errors.Is(err, ErrNoTenantAdmin) {
		t.Errorf("Activate on a dead database = %v", err)
	}
	if n, err := s.ReleaseDomains(ctx, tenant); err == nil || n != 0 {
		t.Errorf("ReleaseDomains = %d, %v", n, err)
	}
	err = s.RecordInvite(ctx, tenant, "a@a.test", time.Now().Add(time.Hour))
	if err == nil || errors.Is(err, keycloak.ErrInviteConsumed) {
		t.Errorf("RecordInvite on a dead database = %v", err)
	}
}

func TestRegistryOnboardingHoldsNoUsersOrGroups(t *testing.T) {
	t.Parallel()
	r := NewRegistry(deadPool(t, 2))
	if _, err := r.UserKeycloakID(t.Context(), uuid.New(), uuid.New()); !errors.Is(err, keycloak.ErrRegistryNotFound) {
		t.Errorf("UserKeycloakID = %v", err)
	}
	if _, err := r.GroupKeycloakID(t.Context(), uuid.New(), uuid.New()); !errors.Is(err, keycloak.ErrRegistryNotFound) {
		t.Errorf("GroupKeycloakID = %v", err)
	}
}

func TestRegistryDatabaseErrorsAreNotAbsence(t *testing.T) {
	t.Parallel()
	r := NewRegistry(deadPool(t, 2))
	if iss, org, found, err := r.Binding(t.Context(), uuid.New()); err == nil || found || iss != "" || org != "" {
		t.Errorf("Binding = %q %q %v %v: a dead database must not read as 'not bound'", iss, org, found, err)
	}
	if got, err := r.BoundIdPAliases(t.Context(), uuid.New()); err == nil || len(got) != 0 {
		t.Errorf("BoundIdPAliases = %v, %v: a dead database must not read as 'no aliases'", got, err)
	}
}
