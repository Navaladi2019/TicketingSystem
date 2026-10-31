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

	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
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
	// RecordAdminGroup writes, in one transaction, the tenant's admins group
	// (platform id AdminGroupID, Keycloak id kcGroupID, path /t/<tenant>/admins)
	// and the tenant_admin_group row naming it. Idempotent: a retry writes
	// neither a second group nor a second row.
	RecordAdminGroup(ctx context.Context, tenantID uuid.UUID, kcGroupID string) error
	// SeedEntityType inserts one entity type at version 1 with the ids in in,
	// idempotently (INSERT ... ON CONFLICT DO NOTHING), and reports whether it
	// inserted anything. Only a provisioning tenant can be seeded: the table
	// policies refuse an active one.
	SeedEntityType(ctx context.Context, tenantID uuid.UUID, in entitytypes.SeedInput) (inserted bool, err error)
	// Activate moves the tenant from provisioning to active. An already-active
	// tenant is ErrNotProvisioning; a tenant with no tenant_admin_group row is
	// ErrNoTenantAdmin and stays provisioning.
	Activate(ctx context.Context, tenantID uuid.UUID) error
}

// Directory is the part of identity.DirectoryAdmin onboarding uses.
type Directory interface {
	CreateTenantOrg(ctx context.Context, tenantID uuid.UUID, alias string, domains []string) (string, error)
	InviteUser(ctx context.Context, tenantID uuid.UUID, email string) error
	EnsureAdminGroup(ctx context.Context, tenantID uuid.UUID) (keycloakGroupID string, err error)
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

// AdminGroupID is the platform id of a tenant's admins group, derived from the
// tenant id so a retry writes the same row (D-P11). It is the id of the groups
// row onboarding inserts; if group sync created the group first, the recorded
// id is that group's real one (Store.RecordAdminGroup reads it back).
func AdminGroupID(tenantID uuid.UUID) uuid.UUID {
	return uuid.NewSHA1(Namespace, []byte("admin-group:"+tenantID.String()))
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
	if err := SeedEntityTypes(ctx, s.o.Store, tenantID); err != nil {
		return Result{}, err
	}
	// The admins group exists, and tenant_admin_group names it, before the
	// admin is invited into it. Each step is idempotent for a retry.
	kcGroupID, err := s.o.Directory.EnsureAdminGroup(ctx, tenantID)
	if err != nil {
		return Result{}, fmt.Errorf("onboard: admin group: %w", err)
	}
	if err := s.o.Store.RecordAdminGroup(ctx, tenantID, kcGroupID); err != nil {
		if errors.Is(err, ErrAdminGroupMissing) {
			// The operator runbook (docs/operator/onboarding.md, "Retry fails
			// with admin group missing") finds the stale groups row with these
			// two ids.
			s.o.Log.ErrorContext(ctx, "onboard: admins group has no groups row",
				slog.String("tenant_id", tenantID.String()),
				slog.String("keycloak_group_id", kcGroupID))
		}
		return Result{}, err
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

// EntityDefinitionID is the id of a tenant's entity type, UUIDv5 over
// (tenant_id, key), so a retried seed names the same row (D-3-2).
func EntityDefinitionID(tenantID uuid.UUID, key string) uuid.UUID {
	return uuid.NewSHA1(Namespace, []byte("entity-definition:"+tenantID.String()+"\x00"+key))
}

// EntityVersionID is the id of version n of a tenant's entity type, UUIDv5
// over (tenant_id, key, n).
func EntityVersionID(tenantID uuid.UUID, key string, n int) uuid.UUID {
	return uuid.NewSHA1(Namespace, []byte(fmt.Sprintf("entity-version:%s\x00%s\x00%d", tenantID, key, n)))
}

// entityTypeNames are the display names of the seed templates.
//
//nolint:gochecknoglobals // constant
var entityTypeNames = map[string]string{"incident": "Incident", "request": "Request", "change": "Change"}

// SeedEntityTypes seeds every template at version 1 through the entity type
// store. Each insert is idempotent on its deterministic ids, so a retry after
// a crash between templates finishes the missing ones. Onboarding calls it
// while the tenant is provisioning; it is exported so the database tests drive
// the same code against the real nava_onboard policies.
func SeedEntityTypes(ctx context.Context, store Store, tenantID uuid.UUID) error {
	for _, key := range metadata.TemplateKeys() {
		d, err := metadata.Template(key)
		if err != nil {
			return fmt.Errorf("onboard: template %s: %w", key, err)
		}
		if _, err := metadata.Compile(d); err != nil {
			return fmt.Errorf("onboard: template %s: %w", key, err)
		}
		name := entityTypeNames[key]
		if name == "" {
			name = key
		}
		if _, err := store.SeedEntityType(ctx, tenantID, entitytypes.SeedInput{
			DefinitionID: EntityDefinitionID(tenantID, key),
			VersionID:    EntityVersionID(tenantID, key, 1),
			Name:         name,
			Definition:   d,
		}); err != nil {
			return fmt.Errorf("onboard: seed %s: %w", key, err)
		}
	}
	return nil
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
