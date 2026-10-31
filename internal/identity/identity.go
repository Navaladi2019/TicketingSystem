// Package identity is the platform's port to the identity provider (ADR-001
// A.4, PLAN-001 T15). The core reaches the IdP only through the interfaces
// here; internal/identity/keycloak is the one adapter in M1, and the only code
// that imports a Keycloak client, calls the Keycloak admin API or reads
// Keycloak-specific claims (D-P6, CI-11).
//
// Every adapter must pass the suite in internal/identity/contracttest.
package identity

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// The two errors a resolver returns for a request it refuses. An adapter wraps
// them with a bounded reason; callers match with errors.Is, never on the text
// (T20 maps them to 401 problem responses).
var (
	// ErrTenantUnresolved: the token is valid but resolves to no tenant, or to
	// more than one, or the brokering IdP is in doubt (401 tenant_unresolved).
	ErrTenantUnresolved = errors.New("tenant_unresolved")
	// ErrInvalidToken: the token failed validation (401 invalid_token).
	ErrInvalidToken = errors.New("invalid_token")
)

// Principal is who a tenant request runs as. TenantID comes only from
// tenant_identity_binding, never from a claim value (I2).
type Principal struct {
	TenantID    uuid.UUID
	PrincipalID uuid.UUID
	// Groups are platform group ids of this tenant, never IdP group names.
	Groups []uuid.UUID
	// IdPAlias is the brokering IdP bound to the tenant, or empty for a
	// local login.
	IdPAlias string
}

// IdentityResolver turns a raw access token into a tenant principal. Only the
// core API's authentication middleware calls it.
//
// It returns an error wrapping ErrInvalidToken or ErrTenantUnresolved for a
// token it refuses, and a zero Principal with any error. An error wrapping
// neither means it could not decide (its Store failed); the middleware answers
// that with 500, never with a 401 that would make a client drop a good token.
type IdentityResolver interface { //nolint:revive // name fixed by PLAN-001's interface contract
	Resolve(ctx context.Context, rawAccessToken string) (Principal, error)
}

// Operator is a platform operator (D-P12). PrincipalID is the IdP subject.
type Operator struct {
	PrincipalID string
}

// OperatorResolver turns a raw access token into a platform operator: realm
// role nava-operator + aud nava-operator-api, both required (D-P12, TICK-57).
// Audience alone is never enough. A tenant token returns ErrInvalidToken.
type OperatorResolver interface {
	ResolveOperator(ctx context.Context, rawAccessToken string) (Operator, error)
}

// The errors a DirectoryAdmin returns for a request it refuses. Match with
// errors.Is; the text never carries an email or an id.
var (
	// ErrNotFound: the user or group is unknown, or is not in the tenant's
	// subtree. The two cases are not told apart.
	ErrNotFound = errors.New("directory: not found")
	// ErrIdPNotBound: the user is federated to an IdP that is not bound to the
	// target tenant (A.1 platform rules).
	ErrIdPNotBound = errors.New("directory: federated to an IdP not bound to the tenant")
	// ErrAlreadyExists: the organization or IdP already exists, or an IdP is
	// already linked to an organization.
	ErrAlreadyExists = errors.New("directory: already exists")
	// ErrInviteRefused: InviteUser refused an existing user, or an address in
	// a domain the tenant does not own; DisableUser refused a user who is in
	// another tenant's subtree (disabling is realm-wide). Nothing was
	// changed. Callers must not tell the requester which case it was.
	ErrInviteRefused = errors.New("directory: refused")
	// ErrInvalidInput: an argument is malformed.
	ErrInvalidInput = errors.New("directory: invalid input")
)

// DirectoryAdmin manages tenant organizations, users and groups in the IdP.
// Only tenant onboarding (nava_onboard) and group sync (nava_groupsync) call
// it. T18 (TICK-73) implements it for Keycloak.
type DirectoryAdmin interface {
	CreateTenantOrg(ctx context.Context, tenantID uuid.UUID, alias string, domains []string) (orgID string, err error)
	// InviteUser puts the invitee in the tenant's group (creating the user if
	// the email is new). It rejects a user federated to an IdP not bound to the
	// tenant with ErrIdPNotBound.
	InviteUser(ctx context.Context, tenantID uuid.UUID, email string) error
	DisableUser(ctx context.Context, tenantID, principalID uuid.UUID) error
	SetGroupMembers(ctx context.Context, tenantID, groupID uuid.UUID, members []uuid.UUID) error
	// EnsureAdminGroup returns the IdP id of the tenant's AdminGroupName group
	// (/t/<tenant_id>/admins), creating it when missing. It is idempotent.
	// Onboarding records the platform id of that group in tenant_admin_group;
	// InviteUser and invitee placement put the invited admin in it.
	EnsureAdminGroup(ctx context.Context, tenantID uuid.UUID) (keycloakGroupID string, err error)
	// ChangesSince is poll and diff (D-P13): a full snapshot plus a digest
	// cursor; Unchanged when the digest matches cursor.
	ChangesSince(ctx context.Context, tenantID uuid.UUID, cursor string) (DirectoryChanges, string, error)
}

// AdminGroupName is the name of every tenant's admin group, a child of the
// tenant's root group. Only the directory adapter and onboarding use the name;
// authorization uses the platform id in tenant_admin_group, never the path.
const AdminGroupName = "admins"

// DirectoryChanges is one ChangesSince snapshot. Directory membership is
// membership of a group under /t/<tenant_id>/ (TICK-73 ruling 2026-10-25), so
// Users are the members of those groups, not the members of the Keycloak
// organization.
type DirectoryChanges struct {
	Unchanged bool
	Users     []DirectoryUser  // members of any group under /t/<tenant_id>/
	Groups    []DirectoryGroup // under /t/<tenant_id>/, excluding /ou/
	OrgUnits  []DirectoryGroup // under /t/<tenant_id>/ou/
}

// DirectoryUser is a user in a tenant's directory snapshot.
type DirectoryUser struct {
	KeycloakUserID string
	Email          string
	Enabled        bool
	FederatedLinks []FederatedLink
}

// FederatedLink is one federated identity of a user.
type FederatedLink struct {
	IdPAlias       string
	ExternalUserID string
}

// DirectoryGroup is a group or org unit in a tenant's directory snapshot.
// ParentKeycloakGroupID is empty for a top-level group or unit.
type DirectoryGroup struct {
	KeycloakGroupID       string
	ParentKeycloakGroupID string
	Path                  string
	Name                  string
	MemberKeycloakUserIDs []string
}

// Binding is a tenant_identity_binding row: the tenant an (issuer,
// organization id) pair is bound to, and the IdP aliases bound to it.
type Binding struct {
	TenantID   uuid.UUID
	IdPAliases []string
	// TenantState is the bound tenant's tenants.state. A resolver refuses
	// every state but TenantStateActive, the zero value included (T17a).
	TenantState string
}

// TenantStateActive is the only tenants.state a resolver accepts. Onboarding
// moves a tenant from provisioning to active as its last step (T21).
const TenantStateActive = "active"

// PrincipalRecord is a principals row as group sync recorded it, with the
// principal's group ids.
type PrincipalRecord struct {
	ID      uuid.UUID
	Enabled bool
	// LinksKnown is false when federated_links is NULL: sync has not recorded
	// the user's links, which the resolver treats as unknown and refuses
	// (A.1 step 3, docs/data/identity.md).
	LinksKnown bool
	// FederatedIdPs are the identityProvider aliases of the user's federated
	// links. Empty with LinksKnown means synced, no federated identity.
	FederatedIdPs []string
	Groups        []uuid.UUID
}

// Errors a Store returns when a row is missing. A resolver refuses the token
// with ErrTenantUnresolved for either.
var (
	ErrNoBinding   = errors.New("identity: no tenant binding")
	ErrNoPrincipal = errors.New("identity: no principal")
)

// Store reads the platform's own identity rows (A.1 steps 3-4). An adapter's
// resolver calls it after validating the token; internal/identity/pgstore is
// the PostgreSQL implementation.
type Store interface {
	// Binding looks (issuer, orgID) up in tenant_identity_binding, with the
	// bound tenant's state. It returns ErrNoBinding for no row, and an error
	// for more than one.
	Binding(ctx context.Context, issuer, orgID string) (Binding, error)
	// Principal reads the principal whose IdP user id is subject, in tenantID
	// only. It returns ErrNoPrincipal for no row.
	Principal(ctx context.Context, tenantID uuid.UUID, subject string) (PrincipalRecord, error)
}

// TenantGroupPrefix is the path prefix of every group of a tenant in the
// directory: groups sit at /t/<tenant_id>/... (D26). Group sync and /me both
// use it, so the layout is written down once.
func TenantGroupPrefix(tenantID uuid.UUID) string { return "/t/" + tenantID.String() + "/" }

// GroupPath is one group a principal belongs to, with the group's raw path as
// stored (it still carries TenantGroupPrefix).
type GroupPath struct {
	ID   uuid.UUID
	Path string
}

// GroupPathReader reads the paths of the groups a principal belongs to, in
// tenantID only, sorted by group id. It reads by principal, never by group id,
// so a caller cannot list groups it is not in. Paths are for display only
// (ADR-002 section 1.4): no authorization decision, ACL, cache key or metric
// may key on one. Only GetMe calls it; the forbidigo rules in .golangci.yml
// hold that. It is separate from Store so the resolver never sees paths.
//
// TODO(TODO-017): the I5 ACL compiler takes group uuids only; a test asserts
// it never reads groups.path or identity.GroupPath.
type GroupPathReader interface {
	GroupPaths(ctx context.Context, tenantID, principalID uuid.UUID) ([]GroupPath, error)
}

type ctxKey struct{}

// NewContext returns ctx carrying p. Only the authentication middleware calls
// it, with the principal its resolver returned.
func NewContext(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

// FromContext returns the request's resolved principal, and false outside an
// authenticated request.
func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(Principal)
	return p, ok
}
