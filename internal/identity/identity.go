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
// It returns an error wrapping ErrInvalidToken or ErrTenantUnresolved, and a
// zero Principal with any error.
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
	// ChangesSince is poll and diff (D-P13): a full snapshot plus a digest
	// cursor; Unchanged when the digest matches cursor.
	ChangesSince(ctx context.Context, tenantID uuid.UUID, cursor string) (DirectoryChanges, string, error)
}

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
