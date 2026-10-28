// Package onboard is operator tenant onboarding (ADR-001 A.3, PLAN-001 T21):
// it creates a tenant in provisioning, its Keycloak organization and IdP, the
// tenant_identity_binding and tenant_email_domain rows, invites the first
// admin, and marks the tenant active.
//
// The tenant id is derived on the server, UUIDv5(namespace, operator principal
// + Idempotency-Key), and is never read from the request: nava_onboard reads
// "its own row" (ADR-001-E3) only while the caller cannot choose
// app.tenant_id. The provisioning row, with its request hash, is the
// idempotency record. A failure part way leaves the tenant provisioning, and a
// retry with the same key and body resumes where it stopped.
package onboard

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"regexp"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
)

// Namespace is the UUIDv5 namespace of onboarding tenant ids (D-P11).
//
//nolint:gochecknoglobals // constant
var Namespace = uuid.MustParse("6f1b7c1e-3c52-5d0a-9a53-0e2b6d7a4f10")

// Errors the service returns. The HTTP layer maps each to a registered code.
var (
	// ErrKeyReuse: the Idempotency-Key was used with a different request.
	ErrKeyReuse = errors.New("onboard: idempotency key reuse")
	// ErrDomainClaimed: another tenant owns an email domain of the request.
	// Operator-only today; a self-service signup must answer generically.
	ErrDomainClaimed = errors.New("onboard: email domain already claimed")
	// ErrConflict: the organization, IdP or binding exists and is not this
	// tenant's.
	ErrConflict = errors.New("onboard: conflict")
	// ErrInvalid wraps a *FieldError.
	ErrInvalid = errors.New("onboard: invalid request")
)

// FieldError names the failing request field by JSON pointer. It never carries
// the value.
type FieldError struct{ Pointer string }

func (e *FieldError) Error() string { return "onboard: invalid field " + e.Pointer }
func (e *FieldError) Unwrap() error { return ErrInvalid }

// Request is a validated-on-entry onboarding request.
type Request struct {
	AdminEmail   string
	EmailDomains []string
	IdP          IdPRequest
}

// Result is what a successful call returns.
type Result struct {
	TenantID uuid.UUID
	// Replayed is true when the tenant was already active.
	Replayed bool
}

// State is what Begin found.
type State struct{ Active bool }

// Store is the tenant database as nava_onboard.
type Store interface {
	// Lock holds the per-tenant lock until the returned func is called.
	Lock(ctx context.Context, tenantID uuid.UUID) (release func(), err error)
	// Begin records the provisioning row with the request hash and claims the
	// email domains, in one transaction, or reports the existing row. A
	// different stored hash is ErrKeyReuse. A domain owned by another tenant is
	// ErrDomainClaimed and rolls the row back too, so nothing is left behind.
	// An existing row already owns its domains: nothing is inserted.
	Begin(ctx context.Context, tenantID uuid.UUID, hash []byte, domains []string) (State, error)
	// Bind writes the binding while the tenant is provisioning. It is
	// idempotent: if the tenant's binding row already exists (read as nava_app)
	// it inserts nothing. An organization bound to another tenant is
	// ErrConflict and rolls back. Onboard then reads domain ownership under the
	// tenant predicate as a second check.
	Bind(ctx context.Context, tenantID uuid.UUID, issuer, orgID, idpAlias string) error
	// Activate moves the tenant from provisioning to active. An already-active
	// tenant is an error.
	Activate(ctx context.Context, tenantID uuid.UUID) error
}

// Directory is the part of identity.DirectoryAdmin onboarding uses.
type Directory interface {
	CreateTenantOrg(ctx context.Context, tenantID uuid.UUID, alias string, domains []string) (string, error)
	InviteUser(ctx context.Context, tenantID uuid.UUID, email string) error
}

// OrgFinder finds the organization an interrupted onboarding already created
// for the tenant, so a retry reuses it instead of creating a second one. It
// returns found=false when there is none, and an error when what exists is
// not a complete, policy-conforming org and IdP of this tenant.
type OrgFinder interface {
	FindTenantOrg(ctx context.Context, tenantID uuid.UUID, alias string) (orgID string, found bool, err error)
}

// DomainOwnership reports whether tenantID owns an email domain, read under
// the tenant predicate (keycloak.Registry.TenantOwnsEmailDomain).
type DomainOwnership interface {
	TenantOwnsEmailDomain(ctx context.Context, tenantID uuid.UUID, domain string) (bool, error)
}

// Options configures Service.
type Options struct {
	Store     Store
	Directory Directory
	Finder    OrgFinder // optional
	// Domains reads, as nava_app under RLS, whether the tenant owns a domain.
	Domains  DomainOwnership
	Specs    *Specs
	Resolver Resolver
	// Issuer is the realm issuer written to tenant_identity_binding.
	Issuer string
	Log    *slog.Logger
}

// Service onboards tenants.
type Service struct {
	o Options
}

// New returns a Service.
func New(o Options) (*Service, error) {
	if o.Store == nil || o.Directory == nil || o.Specs == nil || o.Domains == nil || o.Resolver == nil || o.Log == nil || o.Issuer == "" {
		return nil, errors.New("onboard: store, directory, domains, specs, resolver, issuer and logger are required")
	}
	return &Service{o: o}, nil
}

// TenantID derives the tenant id of (operator, key).
func TenantID(operatorPrincipal, key string) uuid.UUID {
	return uuid.NewSHA1(Namespace, []byte(operatorPrincipal+"\x00"+key))
}

// Alias is the organization alias of a tenant, derived from the id alone.
func Alias(tenantID uuid.UUID) string { return "t-" + tenantID.String() }

// Onboard runs the onboarding of the operator's request. hash is
// idempotency.RequestHash of the HTTP request.
func (s *Service) Onboard(ctx context.Context, operator identity.Operator, key string, hash []byte, req Request) (Result, error) {
	tenantID := TenantID(operator.PrincipalID, key)
	domains, spec, err := s.validate(ctx, req)
	if err != nil {
		return Result{}, err
	}

	release, err := s.o.Store.Lock(ctx, tenantID)
	if err != nil {
		return Result{}, err
	}
	defer release()

	st, err := s.o.Store.Begin(ctx, tenantID, hash, domains)
	if err != nil {
		return Result{}, err
	}
	if st.Active {
		// Activation is the last step: an active tenant was bound and its
		// admin invited, so a replay changes nothing.
		return Result{TenantID: tenantID, Replayed: true}, nil
	}

	alias := Alias(tenantID)
	orgID, err := s.org(ctx, tenantID, alias, domains, spec)
	if err != nil {
		return Result{}, err
	}
	if err := s.o.Store.Bind(ctx, tenantID, s.o.Issuer, orgID, keycloak.TenantIdPAlias(tenantID)); err != nil {
		return Result{}, err
	}
	// Begin claimed the domains, and a domain owned by another tenant failed it
	// (ErrDomainClaimed). The tenant must still own every domain it asked for,
	// read with WHERE tenant_id as a second check.
	for _, d := range domains {
		owns, err := s.o.Domains.TenantOwnsEmailDomain(ctx, tenantID, d)
		if err != nil {
			return Result{}, fmt.Errorf("onboard: domain ownership: %w", err)
		}
		if !owns {
			return Result{}, ErrDomainClaimed
		}
	}
	if err := s.inviteAdmin(ctx, tenantID, req.AdminEmail); err != nil {
		return Result{}, err
	}
	if err := s.o.Store.Activate(ctx, tenantID); err != nil {
		return Result{}, err
	}
	s.o.Log.InfoContext(ctx, "tenant onboarded", slog.String("tenant_id", tenantID.String()))
	return Result{TenantID: tenantID}, nil
}

func (s *Service) org(ctx context.Context, tenantID uuid.UUID, alias string, domains []string, spec keycloak.IdPSpec) (string, error) {
	if s.o.Finder != nil {
		orgID, found, err := s.o.Finder.FindTenantOrg(ctx, tenantID, alias)
		if err != nil {
			return "", fmt.Errorf("onboard: find org: %w", err)
		}
		if found {
			return orgID, nil
		}
	}
	s.o.Specs.put(tenantID, spec)
	defer s.o.Specs.drop(tenantID)
	orgID, err := s.o.Directory.CreateTenantOrg(ctx, tenantID, alias, domains)
	switch {
	case errors.Is(err, identity.ErrAlreadyExists):
		return "", ErrConflict
	case errors.Is(err, keycloak.ErrDomainLinked):
		// Defence in depth: Begin already refused a domain another tenant
		// owns, so this is an organization created or left outside the platform
		// (e.g. a deleted tenant's).
		return "", ErrDomainClaimed
	case errors.Is(err, identity.ErrInvalidInput):
		return "", &FieldError{Pointer: "/identity_provider"}
	case err != nil:
		return "", fmt.Errorf("onboard: create org: %w", err)
	}
	return orgID, nil
}

// inviteAdmin invites the admin. ErrInviteRefused (an existing user, or an
// address outside the tenant's domains) is answered as a success: the caller
// must not be able to tell it from a new invite.
func (s *Service) inviteAdmin(ctx context.Context, tenantID uuid.UUID, email string) error {
	err := s.o.Directory.InviteUser(ctx, tenantID, email)
	switch {
	case err == nil, errors.Is(err, identity.ErrInviteRefused):
		return nil
	default:
		return fmt.Errorf("onboard: invite admin: %w", err)
	}
}

var domainRE = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)

const maxDomains = 20

func (s *Service) validate(ctx context.Context, req Request) ([]string, keycloak.IdPSpec, error) {
	fail := func(p string) ([]string, keycloak.IdPSpec, error) {
		return nil, keycloak.IdPSpec{}, &FieldError{Pointer: p}
	}
	if len(req.EmailDomains) == 0 || len(req.EmailDomains) > maxDomains {
		return fail("/email_domains")
	}
	seen := map[string]bool{}
	domains := make([]string, 0, len(req.EmailDomains))
	for _, d := range req.EmailDomains {
		d = strings.ToLower(strings.TrimSpace(d))
		if len(d) > 253 || !domainRE.MatchString(d) || publicEmailDomain(d) {
			return fail("/email_domains")
		}
		if !seen[d] {
			seen[d] = true
			domains = append(domains, d)
		}
	}
	addr, err := mail.ParseAddress(req.AdminEmail)
	if err != nil || addr.Address != req.AdminEmail || len(req.AdminEmail) > 254 {
		return fail("/admin_email")
	}
	at := strings.LastIndexByte(req.AdminEmail, '@')
	if !seen[strings.ToLower(req.AdminEmail[at+1:])] {
		return fail("/admin_email")
	}
	spec, ptr := buildSpec(ctx, s.o.Resolver, req.IdP)
	if ptr != "" {
		return fail(ptr)
	}
	return domains, spec, nil
}

// Specs carries the IdP spec of a tenant being onboarded from the service to
// the Keycloak adapter's keycloak.IdPSource. The service holds the per-tenant
// lock while an entry exists.
type Specs struct {
	mu sync.Mutex
	m  map[uuid.UUID]keycloak.IdPSpec
}

// NewSpecs returns an empty Specs.
func NewSpecs() *Specs { return &Specs{m: map[uuid.UUID]keycloak.IdPSpec{}} }

func (s *Specs) put(id uuid.UUID, spec keycloak.IdPSpec) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[id] = spec
}

func (s *Specs) drop(id uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, id)
}

// Source is the keycloak.IdPSource.
func (s *Specs) Source(_ context.Context, id uuid.UUID) (keycloak.IdPSpec, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	spec, ok := s.m[id]
	if !ok {
		return keycloak.IdPSpec{}, errors.New("onboard: no idp spec for tenant")
	}
	return spec, nil
}
