package keycloak

// Directory is the Keycloak adapter of identity.DirectoryAdmin (T18, TICK-73).
//
// Directory membership is membership of a group under /t/<tenant_id>/ (ruling
// 2026-10-25, Option 2): the runtime account has users and groups roles only
// and cannot read Keycloak organization membership. The rules that keep that
// from becoming a cross-tenant door:
//   - the subtree /t/<tenant_id>/ is derived from the tenant argument, never
//     from input; a group or user outside it is identity.ErrNotFound;
//   - Keycloak user, group and IdP ids come only from the Registry, which the
//     caller backs with rows read under row-level security;
//   - a user federated to an IdP not bound to the tenant is never added to the
//     tenant's groups (identity.ErrIdPNotBound);
//   - CreateTenantOrg is the only method that uses the onboarding account. The
//     runtime path never holds its secret: leave Config.Onboarding nil.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

const (
	// FirstBrokerLoginFlow is the first-login flow of every tenant IdP (A.1):
	// create if unique, add to the organization, no link by email.
	FirstBrokerLoginFlow = "nava-first-broker-login"
	// IdPAliasPrefix prefixes the server-derived alias of a tenant's IdP.
	IdPAliasPrefix = "idp-"

	maxDirectoryItems = 20000
	maxEmailBytes     = 254
	rollbackTimeout   = 10 * time.Second
)

// ErrOnboardingDisabled is returned by CreateTenantOrg on a Directory built
// without onboarding credentials.
var ErrOnboardingDisabled = errors.New("keycloak: onboarding account not configured")

// ErrRegistryNotFound is what a Registry returns for an id it does not hold.
var ErrRegistryNotFound = errors.New("keycloak: registry: not found")

// Registry maps platform ids to Keycloak ids. The caller implements it by
// reading principals, groups and tenant_identity_binding inside
// tenancy.InTenantTx, so row-level security scopes every answer to the tenant.
// An id the tenant does not hold is ErrRegistryNotFound.
type Registry interface {
	UserKeycloakID(ctx context.Context, tenantID, principalID uuid.UUID) (string, error)
	GroupKeycloakID(ctx context.Context, tenantID, groupID uuid.UUID) (string, error)
	// BoundIdPAliases are the IdP aliases bound to the tenant.
	BoundIdPAliases(ctx context.Context, tenantID uuid.UUID) ([]string, error)
	// TenantOwnsEmailDomain reports whether the tenant owns the email domain
	// (lower case, no "@"). InviteUser creates a user only for an owned
	// domain, so a tenant cannot pre-create an address in another tenant's
	// domain and block that tenant's first brokered login (TICK-194).
	TenantOwnsEmailDomain(ctx context.Context, tenantID uuid.UUID, domain string) (bool, error)
}

// InviteWriter records the pending admin invite InviteUser makes for an email
// that has no Keycloak user yet (TICK-252). The caller backs it with a
// transaction that sets app.tenant_id, so row-level security scopes the write
// to the tenant. Re-recording a pending invite moves its expiry; a consumed
// invite stays consumed.
type InviteWriter interface {
	RecordInvite(ctx context.Context, tenantID uuid.UUID, email string, expiresAt time.Time) error
}

// DefaultInviteTTL is how long a pending invite stays valid when
// DirectoryConfig.InviteTTL is zero.
const DefaultInviteTTL = 7 * 24 * time.Hour

// IdPSpec is what a tenant's IdP needs beyond what the platform fixes. The
// alias, trustEmail, the first-login flow and mappers are never taken from it.
type IdPSpec struct {
	// ProviderID is "oidc" or "saml".
	ProviderID  string
	DisplayName string
	// Config is the provider config: endpoints, client id and secret.
	Config map[string]string
}

// IdPSource supplies the IdP spec of a tenant being onboarded. T21's operator
// handler provides it.
type IdPSource func(ctx context.Context, tenantID uuid.UUID) (IdPSpec, error)

// DirectoryConfig configures Directory.
type DirectoryConfig struct {
	// BaseURL is the Keycloak origin, such as https://id.example.com.
	BaseURL string
	// Realm defaults to "nava".
	Realm string
	// Runtime is the runtime service account (manage-users, query-users,
	// query-groups, view-users, and view-realm to read organization membership
	// in PlaceInvitee). Required.
	Runtime ClientCredentials
	// Onboarding is the onboarding service account (manage-realm,
	// manage-identity-providers). Nil on every runtime path.
	Onboarding *ClientCredentials
	// Registry is required.
	Registry Registry
	// IdPs is required for CreateTenantOrg.
	IdPs IdPSource
	// Invites is required to invite an email that has no user yet. Leave it
	// nil on a process that never invites: such an invite then fails closed.
	Invites InviteWriter
	// InviteTTL is how long a pending invite is valid. Zero is
	// DefaultInviteTTL.
	InviteTTL time.Duration
}

// Directory implements identity.DirectoryAdmin.
type Directory struct {
	api        *admin
	runtime    *account
	onboarding *account
	reg        Registry
	idps       IdPSource
	invites    InviteWriter
	inviteTTL  time.Duration
	now        func() time.Time
}

var _ identity.DirectoryAdmin = (*Directory)(nil)

// NewDirectory builds the adapter; every call goes through out.
func NewDirectory(cfg DirectoryConfig, out *telemetry.OutboundClient) (*Directory, error) {
	if out == nil {
		return nil, errors.New("keycloak: an outbound client is required")
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.RawQuery != "" || u.User != nil {
		return nil, errors.New("keycloak: invalid base URL")
	}
	if cfg.Runtime.ClientID == "" || cfg.Runtime.ClientSecret == "" {
		return nil, errors.New("keycloak: runtime credentials are required")
	}
	if cfg.Onboarding != nil {
		if cfg.Onboarding.ClientID == "" || cfg.Onboarding.ClientSecret == "" {
			return nil, errors.New("keycloak: onboarding credentials are incomplete")
		}
		if cfg.Onboarding.ClientID == cfg.Runtime.ClientID {
			return nil, errors.New("keycloak: onboarding and runtime must be different accounts")
		}
	}
	if cfg.Registry == nil {
		return nil, errors.New("keycloak: a registry is required")
	}
	if cfg.Realm == "" {
		cfg.Realm = "nava"
	}
	if cfg.InviteTTL < 0 {
		return nil, errors.New("keycloak: invite TTL must not be negative")
	}
	if cfg.InviteTTL == 0 {
		cfg.InviteTTL = DefaultInviteTTL
	}
	d := &Directory{
		api:     &admin{base: strings.TrimRight(cfg.BaseURL, "/"), realm: cfg.Realm, out: out, now: time.Now},
		runtime: &account{creds: cfg.Runtime},
		reg:     cfg.Registry,
		idps:    cfg.IdPs,
		invites: cfg.Invites, inviteTTL: cfg.InviteTTL, now: time.Now,
	}
	if cfg.Onboarding != nil {
		d.onboarding = &account{creds: *cfg.Onboarding}
	}
	return d, nil
}

// ---- Keycloak representations ----------------------------------------------

type kcUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Enabled   bool   `json:"enabled"`
	Username  string `json:"username"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
}

type kcGroup struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Path          string `json:"path"`
	SubGroupCount int    `json:"subGroupCount"`
}

// ---- subtree rules ---------------------------------------------------------

func rootPath(tenantID uuid.UUID) string { return "/t/" + tenantID.String() }

// inTenant reports whether path is the tenant's root group or below it.
func inTenant(tenantID uuid.UUID, path string) bool {
	root := rootPath(tenantID)
	return path == root || strings.HasPrefix(path, root+"/")
}

// inOtherTenant reports whether path is in some tenant's subtree other than
// tenantID's.
func inOtherTenant(tenantID uuid.UUID, path string) bool {
	return (path == "/t" || strings.HasPrefix(path, "/t/")) && !inTenant(tenantID, path)
}

// inTenantGroup reports whether the user is a member of a group in the
// tenant's subtree.
func (d *Directory) inTenantGroup(ctx context.Context, tenantID uuid.UUID, kcUserID string) (bool, error) {
	groups, err := pages[kcGroup](ctx, d.api, d.runtime, "/users/{id}/groups", "/users/"+url.PathEscape(kcUserID)+"/groups", nil)
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(groups, func(g kcGroup) bool { return inTenant(tenantID, g.Path) }), nil
}

func isOrgUnit(tenantID uuid.UUID, path string) bool {
	return strings.HasPrefix(path, rootPath(tenantID)+"/ou/")
}

func requireTenant(tenantID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("%w: no tenant", identity.ErrInvalidInput)
	}
	return nil
}

// tenantGroup reads a group by Keycloak id and requires it to be in the
// tenant's subtree, so a group id of another tenant is ErrNotFound.
func (d *Directory) tenantGroup(ctx context.Context, tenantID uuid.UUID, kcGroupID string) (kcGroup, error) {
	var g kcGroup
	err := d.api.getJSON(ctx, d.runtime, "/groups/{id}", "/groups/"+url.PathEscape(kcGroupID), nil, &g)
	if err != nil {
		return kcGroup{}, err
	}
	if !inTenant(tenantID, g.Path) {
		return kcGroup{}, identity.ErrNotFound
	}
	return g, nil
}

// rootGroup finds /t/<tenant_id>.
func (d *Directory) rootGroup(ctx context.Context, tenantID uuid.UUID) (kcGroup, error) {
	var g kcGroup
	err := d.api.getJSON(ctx, d.runtime, "/group-by-path/t/{tenant}", "/group-by-path/t/"+tenantID.String(), nil, &g)
	if err != nil {
		return kcGroup{}, err
	}
	if g.Path != rootPath(tenantID) {
		return kcGroup{}, identity.ErrNotFound
	}
	return g, nil
}

// ensureRootGroup finds /t/<tenant_id>, creating it under /t when missing.
func (d *Directory) ensureRootGroup(ctx context.Context, tenantID uuid.UUID) (kcGroup, error) {
	g, err := d.rootGroup(ctx, tenantID)
	if !errors.Is(err, identity.ErrNotFound) {
		return g, err
	}
	var top kcGroup
	if err := d.api.getJSON(ctx, d.runtime, "/group-by-path/t", "/group-by-path/t", nil, &top); err != nil {
		return kcGroup{}, err
	}
	_, err = d.api.do(ctx, d.runtime, http.MethodPost, "/groups/{id}/children", "/groups/"+url.PathEscape(top.ID)+"/children",
		nil, map[string]string{"name": tenantID.String()}, http.StatusCreated)
	if err != nil && !errors.Is(err, identity.ErrAlreadyExists) {
		return kcGroup{}, err
	}
	return d.rootGroup(ctx, tenantID)
}

// descendants lists every group below root, depth first, with each one's
// parent id. Any group whose path is outside the tenant subtree stops the walk.
func (d *Directory) descendants(ctx context.Context, tenantID uuid.UUID, root kcGroup) ([]kcGroup, map[string]string, error) {
	var out []kcGroup
	parents := map[string]string{}
	var walk func(g kcGroup) error
	walk = func(g kcGroup) error {
		if g.SubGroupCount == 0 && g.ID != root.ID {
			return nil
		}
		kids, err := pages[kcGroup](ctx, d.api, d.runtime, "/groups/{id}/children",
			"/groups/"+url.PathEscape(g.ID)+"/children", url.Values{"briefRepresentation": {"false"}})
		if err != nil {
			return err
		}
		for _, k := range kids {
			if !inTenant(tenantID, k.Path) {
				return errors.New("keycloak: group outside the tenant subtree")
			}
			if len(out) >= maxDirectoryItems {
				return errors.New("keycloak: directory too large")
			}
			out = append(out, k)
			parents[k.ID] = g.ID
			if err := walk(k); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(root); err != nil {
		return nil, nil, err
	}
	return out, parents, nil
}

func (d *Directory) groupMembers(ctx context.Context, kcGroupID string) ([]kcUser, error) {
	return pages[kcUser](ctx, d.api, d.runtime, "/groups/{id}/members",
		"/groups/"+url.PathEscape(kcGroupID)+"/members", url.Values{"briefRepresentation": {"false"}})
}

// ---- federated-link guard --------------------------------------------------

type kcFederated struct {
	IdentityProvider string `json:"identityProvider"`
	UserID           string `json:"userId"`
}

func (d *Directory) federatedLinks(ctx context.Context, kcUserID string) ([]kcFederated, error) {
	var links []kcFederated
	err := d.api.getJSON(ctx, d.runtime, "/users/{id}/federated-identity",
		"/users/"+url.PathEscape(kcUserID)+"/federated-identity", nil, &links)
	return links, err
}

// checkBound refuses a user federated to an IdP the tenant does not bind.
func (d *Directory) checkBound(ctx context.Context, tenantID uuid.UUID, kcUserID string) error {
	links, err := d.federatedLinks(ctx, kcUserID)
	if err != nil {
		return err
	}
	if len(links) == 0 {
		return nil
	}
	bound, err := d.reg.BoundIdPAliases(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("keycloak: bound idps: %w", registryErr(err))
	}
	for _, l := range links {
		if !slices.Contains(bound, l.IdentityProvider) {
			return identity.ErrIdPNotBound
		}
	}
	return nil
}

func registryErr(err error) error {
	if errors.Is(err, ErrRegistryNotFound) {
		return identity.ErrNotFound
	}
	return err
}

// ---- DirectoryAdmin --------------------------------------------------------

// InviteUser invites the email to the tenant. The email only ever travels in
// a request body or a query that the outbound client reduces to its route
// template.
//
// A new email gets no Keycloak user. When the tenant owns the email's domain
// (Registry.TenantOwnsEmailDomain), InviteUser records a pending invite
// (InviteWriter) that expires after the invite TTL. The invitee's first login
// through the tenant's IdP creates the user (nava-first-broker-login: create
// if unique, add to the organization; nothing is ever linked by email), and
// group sync then places that user in /t/<tenant_id> and consumes the invite
// (PlaceInvitee). Nothing here calls Keycloak's invitation API.
//
// An existing user is accepted only when (a) already in the tenant's subtree
// (idempotent), or (b) all of their federated links are bound to the tenant
// and they are in no /t/ group outside it. Anything else is ErrInviteRefused
// and changes nothing; a local user joining a second tenant needs a consent
// step that does not exist yet. The caller must answer a refusal like a
// success, so the call is no oracle for "this email exists elsewhere".
func (d *Directory) InviteUser(ctx context.Context, tenantID uuid.UUID, email string) error {
	if err := requireTenant(tenantID); err != nil {
		return err
	}
	email, err := normalizeEmail(email)
	if err != nil {
		return err
	}
	root, err := d.ensureRootGroup(ctx, tenantID)
	if err != nil {
		return err
	}
	userID, err := d.findByEmail(ctx, email)
	if err != nil {
		return err
	}
	if userID == "" {
		owns, err := d.reg.TenantOwnsEmailDomain(ctx, tenantID, email[strings.LastIndex(email, "@")+1:])
		if err != nil {
			return fmt.Errorf("keycloak: email domain: %w", registryErr(err))
		}
		if !owns {
			return identity.ErrInviteRefused
		}
		if d.invites == nil {
			return errors.New("keycloak: invites are not configured")
		}
		return d.invites.RecordInvite(ctx, tenantID, email, d.now().Add(d.inviteTTL))
	} else if err := d.checkInvitable(ctx, tenantID, userID); err != nil {
		return err
	}
	_, err = d.api.do(ctx, d.runtime, http.MethodPut, "/users/{id}/groups/{groupId}",
		"/users/"+url.PathEscape(userID)+"/groups/"+url.PathEscape(root.ID), nil, nil, http.StatusNoContent)
	return err
}

// checkInvitable applies the rules for inviting an existing user.
func (d *Directory) checkInvitable(ctx context.Context, tenantID uuid.UUID, kcUserID string) error {
	groups, err := pages[kcGroup](ctx, d.api, d.runtime, "/users/{id}/groups", "/users/"+url.PathEscape(kcUserID)+"/groups", nil)
	if err != nil {
		return err
	}
	if slices.ContainsFunc(groups, func(g kcGroup) bool { return inTenant(tenantID, g.Path) }) {
		return nil
	}
	if slices.ContainsFunc(groups, func(g kcGroup) bool { return inOtherTenant(tenantID, g.Path) }) {
		return identity.ErrInviteRefused
	}
	links, err := d.federatedLinks(ctx, kcUserID)
	if err != nil {
		return err
	}
	if len(links) == 0 {
		return identity.ErrInviteRefused // a local user, or the platform operator
	}
	bound, err := d.reg.BoundIdPAliases(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("keycloak: bound idps: %w", registryErr(err))
	}
	for _, l := range links {
		if !slices.Contains(bound, l.IdentityProvider) {
			return identity.ErrInviteRefused
		}
	}
	return nil
}

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	a, err := mail.ParseAddress(email)
	if err != nil || a.Address != email || a.Name != "" || len(email) > maxEmailBytes {
		return "", fmt.Errorf("%w: email", identity.ErrInvalidInput)
	}
	return email, nil
}

// findByEmail returns the id of the user with exactly this email, or "".
func (d *Directory) findByEmail(ctx context.Context, email string) (string, error) {
	var us []kcUser
	err := d.api.getJSON(ctx, d.runtime, "/users", "/users", url.Values{"email": {email}, "exact": {"true"}}, &us)
	if err != nil {
		return "", err
	}
	for _, u := range us {
		if strings.EqualFold(u.Email, email) {
			return u.ID, nil
		}
	}
	return "", nil
}

// DisableUser disables a user of the tenant and ends their sessions. A user
// who is in no group of the tenant's subtree is ErrNotFound.
func (d *Directory) DisableUser(ctx context.Context, tenantID, principalID uuid.UUID) error {
	if err := requireTenant(tenantID); err != nil {
		return err
	}
	kcID, err := d.reg.UserKeycloakID(ctx, tenantID, principalID)
	if err != nil {
		return registryErr(err)
	}
	groups, err := pages[kcGroup](ctx, d.api, d.runtime, "/users/{id}/groups", "/users/"+url.PathEscape(kcID)+"/groups", nil)
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(groups, func(g kcGroup) bool { return inTenant(tenantID, g.Path) }) {
		return identity.ErrNotFound
	}
	// Disabling is realm-wide: unless every /t/ group of the user is in this
	// tenant's subtree, it would also stop another tenant's tokens.
	if slices.ContainsFunc(groups, func(g kcGroup) bool { return inOtherTenant(tenantID, g.Path) }) {
		return identity.ErrInviteRefused
	}
	// Send the whole representation back: a partial PUT can blank profile
	// attributes.
	var raw map[string]any
	if err := d.api.getJSON(ctx, d.runtime, "/users/{id}", "/users/"+url.PathEscape(kcID), nil, &raw); err != nil {
		return err
	}
	raw["enabled"] = false
	if _, err := d.api.do(ctx, d.runtime, http.MethodPut, "/users/{id}", "/users/"+url.PathEscape(kcID), nil, raw, http.StatusNoContent); err != nil {
		return err
	}
	_, err = d.api.do(ctx, d.runtime, http.MethodPost, "/users/{id}/logout", "/users/"+url.PathEscape(kcID)+"/logout", nil, nil, http.StatusNoContent)
	return err
}

// SetGroupMembers makes members exactly the members of the group. The group
// must be in the tenant's subtree; every added user must pass the
// federated-link guard and already be in a group under /t/<tenant_id>/
// (ErrNotFound otherwise: a new user enters through InviteUser). Nothing
// changes if any check fails.
//
// It is not atomic: Keycloak has no transaction, so a write that fails
// halfway leaves the group partly changed, and two concurrent calls on one
// group can interleave. The caller (T25) takes a per-tenant lock and repeats
// the call; a repeat converges.
func (d *Directory) SetGroupMembers(ctx context.Context, tenantID, groupID uuid.UUID, members []uuid.UUID) error {
	if err := requireTenant(tenantID); err != nil {
		return err
	}
	kcGroupID, err := d.reg.GroupKeycloakID(ctx, tenantID, groupID)
	if err != nil {
		return registryErr(err)
	}
	g, err := d.tenantGroup(ctx, tenantID, kcGroupID)
	if err != nil {
		return err
	}
	want := map[string]bool{}
	for _, m := range members {
		kcID, err := d.reg.UserKeycloakID(ctx, tenantID, m)
		if err != nil {
			return registryErr(err)
		}
		want[kcID] = true
	}
	cur, err := d.groupMembers(ctx, g.ID)
	if err != nil {
		return err
	}
	have := map[string]bool{}
	for _, u := range cur {
		have[u.ID] = true
	}
	var add, remove []string
	for id := range want {
		if !have[id] {
			add = append(add, id)
		}
	}
	for id := range have {
		if !want[id] {
			remove = append(remove, id)
		}
	}
	sort.Strings(add)
	sort.Strings(remove)
	for _, id := range add {
		if err := d.checkBound(ctx, tenantID, id); err != nil {
			return err
		}
		member, err := d.inTenantGroup(ctx, tenantID, id)
		if err != nil {
			return err
		}
		if !member {
			return identity.ErrNotFound
		}
	}
	for _, id := range add {
		if _, err := d.api.do(ctx, d.runtime, http.MethodPut, "/users/{id}/groups/{groupId}",
			"/users/"+url.PathEscape(id)+"/groups/"+url.PathEscape(g.ID), nil, nil, http.StatusNoContent); err != nil {
			return err
		}
	}
	for _, id := range remove {
		if _, err := d.api.do(ctx, d.runtime, http.MethodDelete, "/users/{id}/groups/{groupId}",
			"/users/"+url.PathEscape(id)+"/groups/"+url.PathEscape(g.ID), nil, nil, http.StatusNoContent); err != nil {
			return err
		}
	}
	return nil
}

// ChangesSince returns the tenant's directory snapshot and its digest. A
// cursor equal to the digest returns Unchanged and nothing else. A tenant with
// no /t/<tenant_id> group yet has an empty directory.
func (d *Directory) ChangesSince(ctx context.Context, tenantID uuid.UUID, cursor string) (identity.DirectoryChanges, string, error) {
	if err := requireTenant(tenantID); err != nil {
		return identity.DirectoryChanges{}, "", err
	}
	snap, err := d.snapshot(ctx, tenantID)
	if err != nil {
		return identity.DirectoryChanges{}, "", err
	}
	digest, err := digestOf(snap)
	if err != nil {
		return identity.DirectoryChanges{}, "", err
	}
	if cursor == digest {
		return identity.DirectoryChanges{Unchanged: true}, digest, nil
	}
	return snap, digest, nil
}

func (d *Directory) snapshot(ctx context.Context, tenantID uuid.UUID) (identity.DirectoryChanges, error) {
	var snap identity.DirectoryChanges
	root, err := d.rootGroup(ctx, tenantID)
	if errors.Is(err, identity.ErrNotFound) {
		return snap, nil
	}
	if err != nil {
		return snap, err
	}
	desc, parents, err := d.descendants(ctx, tenantID, root)
	if err != nil {
		return snap, err
	}
	ouPath := rootPath(tenantID) + "/ou"
	users := map[string]kcUser{}
	collect := func(groupID string) ([]string, error) {
		ms, err := d.groupMembers(ctx, groupID)
		if err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(ms))
		for _, u := range ms {
			users[u.ID] = u
			ids = append(ids, u.ID)
		}
		sort.Strings(ids)
		return ids, nil
	}
	if _, err := collect(root.ID); err != nil {
		return snap, err
	}
	ouID := ""
	for _, g := range desc {
		if g.Path == ouPath {
			ouID = g.ID
		}
	}
	for _, g := range desc {
		if g.Path == ouPath {
			continue // the container of the org units is neither
		}
		ids, err := collect(g.ID)
		if err != nil {
			return snap, err
		}
		parent := parents[g.ID]
		if parent == root.ID || parent == ouID {
			parent = ""
		}
		dg := identity.DirectoryGroup{
			KeycloakGroupID: g.ID, ParentKeycloakGroupID: parent, Path: g.Path, Name: g.Name, MemberKeycloakUserIDs: ids,
		}
		if isOrgUnit(tenantID, g.Path) {
			snap.OrgUnits = append(snap.OrgUnits, dg)
		} else {
			snap.Groups = append(snap.Groups, dg)
		}
	}
	for id, u := range users {
		links, err := d.federatedLinks(ctx, id)
		if err != nil {
			return snap, err
		}
		du := identity.DirectoryUser{KeycloakUserID: id, Email: u.Email, Enabled: u.Enabled}
		for _, l := range links {
			du.FederatedLinks = append(du.FederatedLinks, identity.FederatedLink{IdPAlias: l.IdentityProvider, ExternalUserID: l.UserID})
		}
		sort.Slice(du.FederatedLinks, func(i, j int) bool { return du.FederatedLinks[i].IdPAlias < du.FederatedLinks[j].IdPAlias })
		snap.Users = append(snap.Users, du)
	}
	sort.Slice(snap.Users, func(i, j int) bool { return snap.Users[i].KeycloakUserID < snap.Users[j].KeycloakUserID })
	byPath := func(s []identity.DirectoryGroup) {
		sort.Slice(s, func(i, j int) bool { return s[i].Path < s[j].Path })
	}
	byPath(snap.Groups)
	byPath(snap.OrgUnits)
	return snap, nil
}

// digestOf is the opaque cursor: the SHA-256 of the snapshot's JSON.
func digestOf(c identity.DirectoryChanges) (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", errors.New("keycloak: digest")
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// ---- onboarding ------------------------------------------------------------

var (
	aliasRE  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)
	domainRE = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)
)

// TenantIdPAlias is the alias of the tenant's IdP, derived from the tenant id
// and nothing else.
func TenantIdPAlias(tenantID uuid.UUID) string { return IdPAliasPrefix + tenantID.String() }

// CreateTenantOrg creates the tenant's organization and its IdP, and links
// that IdP to that organization. It uses the onboarding account. It sets
// trustEmail=false and the first-login flow itself, creates no IdP mapper, and
// refuses an IdP that is already linked to an organization. On a failure after
// the first object exists it deletes what it created.
func (d *Directory) CreateTenantOrg(ctx context.Context, tenantID uuid.UUID, alias string, domains []string) (string, error) {
	if err := requireTenant(tenantID); err != nil {
		return "", err
	}
	if d.onboarding == nil || d.idps == nil {
		return "", ErrOnboardingDisabled
	}
	if !aliasRE.MatchString(alias) {
		return "", fmt.Errorf("%w: alias", identity.ErrInvalidInput)
	}
	for _, dom := range domains {
		if !domainRE.MatchString(dom) {
			return "", fmt.Errorf("%w: domain", identity.ErrInvalidInput)
		}
	}
	spec, err := d.idps(ctx, tenantID)
	if err != nil {
		return "", fmt.Errorf("keycloak: idp spec: %w", err)
	}
	if spec.ProviderID != "oidc" && spec.ProviderID != "saml" {
		return "", fmt.Errorf("%w: idp provider", identity.ErrInvalidInput)
	}
	if err := checkIdPConfig(spec); err != nil {
		return "", err
	}
	idpAlias := TenantIdPAlias(tenantID)

	// An IdP under the derived alias that already exists is not ours to
	// touch: refuse before writing anything. Keycloak validates the config
	// before it looks for a duplicate alias, so a bad config on an existing
	// alias is a 400, not a 409, and the rollback below would otherwise
	// delete a live tenant's IdP.
	switch _, err := d.readIdP(ctx, idpAlias); {
	case err == nil:
		return "", fmt.Errorf("%w: idp", identity.ErrAlreadyExists)
	case !errors.Is(err, identity.ErrNotFound):
		return "", err
	}

	doms := make([]map[string]any, 0, len(domains))
	for _, dom := range domains {
		doms = append(doms, map[string]any{"name": dom, "verified": false})
	}
	r, err := d.api.do(ctx, d.onboarding, http.MethodPost, "/organizations", "/organizations", nil,
		map[string]any{"name": alias, "alias": alias, "enabled": true, "domains": doms}, http.StatusCreated)
	if err != nil {
		return "", err
	}
	orgID, ok := idFromLocation(r.header)
	if !ok {
		return "", errors.New("keycloak: created organization has no location")
	}
	idpCreated := false
	fail := func(err error) (string, error) {
		rb, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackTimeout)
		defer cancel()
		// The alias was absent before we started, so an IdP under it is ours
		// if we may have created it. Delete it only if it is unlinked or
		// linked to the organization we just made.
		if idpCreated {
			if idp, rerr := d.readIdP(rb, idpAlias); rerr == nil && onlyLinkedTo(idp, orgID) {
				_, _ = d.api.do(rb, d.onboarding, http.MethodDelete, "/identity-provider/instances/{alias}",
					"/identity-provider/instances/"+url.PathEscape(idpAlias), nil, nil, http.StatusNoContent)
			}
		}
		_, _ = d.api.do(rb, d.onboarding, http.MethodDelete, "/organizations/{id}",
			"/organizations/"+url.PathEscape(orgID), nil, nil, http.StatusNoContent)
		return "", err
	}

	_, err = d.api.do(ctx, d.onboarding, http.MethodPost, "/identity-provider/instances", "/identity-provider/instances", nil,
		map[string]any{
			"alias":                     idpAlias,
			"displayName":               spec.DisplayName,
			"providerId":                spec.ProviderID,
			"enabled":                   true,
			"trustEmail":                false,
			"storeToken":                false,
			"linkOnly":                  false,
			"firstBrokerLoginFlowAlias": FirstBrokerLoginFlow,
			"config":                    spec.Config,
		}, http.StatusCreated)
	if err != nil {
		// The create may have reached Keycloak though the reply was lost:
		// clean up the alias (absent at the pre-check), except on a 409,
		// where someone else created it meanwhile and it is not ours.
		idpCreated = !errors.Is(err, identity.ErrAlreadyExists)
		return fail(err)
	}
	idpCreated = true

	idp, err := d.readIdP(ctx, idpAlias)
	if err != nil {
		return fail(err)
	}
	if len(idp.OrganizationLinks) != 0 {
		return fail(fmt.Errorf("%w: idp already linked to an organization", identity.ErrAlreadyExists))
	}
	if _, err := d.api.do(ctx, d.onboarding, http.MethodPost, "/organizations/{id}/identity-providers",
		"/organizations/"+url.PathEscape(orgID)+"/identity-providers", nil, idpAlias, http.StatusNoContent); err != nil {
		return fail(err)
	}

	// Verify what exists, not what was sent.
	idp, err = d.readIdP(ctx, idpAlias)
	if err != nil {
		return fail(err)
	}
	if len(idp.OrganizationLinks) != 1 || idp.OrganizationLinks[0].OrganizationID != orgID ||
		idp.TrustEmail || idp.FirstBrokerLoginFlowAlias != FirstBrokerLoginFlow || idp.PostBrokerLoginFlowAlias != "" {
		return fail(errors.New("keycloak: created idp does not match the onboarding policy"))
	}
	var mappers []json.RawMessage
	if err := d.api.getJSON(ctx, d.onboarding, "/identity-provider/instances/{alias}/mappers",
		"/identity-provider/instances/"+url.PathEscape(idpAlias)+"/mappers", nil, &mappers); err != nil {
		return fail(err)
	}
	if len(mappers) != 0 {
		return fail(errors.New("keycloak: new idp has a mapper"))
	}
	return orgID, nil
}

// onlyLinkedTo reports whether the IdP has no organization link, or only one
// to orgID.
func onlyLinkedTo(idp kcIdP, orgID string) bool {
	for _, l := range idp.OrganizationLinks {
		if l.OrganizationID != orgID {
			return false
		}
	}
	return true
}

type kcIdP struct {
	Alias                     string `json:"alias"`
	TrustEmail                bool   `json:"trustEmail"`
	FirstBrokerLoginFlowAlias string `json:"firstBrokerLoginFlowAlias"`
	PostBrokerLoginFlowAlias  string `json:"postBrokerLoginFlowAlias"`
	OrganizationLinks         []struct {
		OrganizationID string `json:"organizationId"`
	} `json:"organizationLinks"`
}

func (d *Directory) readIdP(ctx context.Context, alias string) (kcIdP, error) {
	var idp kcIdP
	err := d.api.getJSON(ctx, d.onboarding, "/identity-provider/instances/{alias}",
		"/identity-provider/instances/"+url.PathEscape(alias), nil, &idp)
	return idp, err
}

// Config keys an IdPSpec may carry. Anything else, such as kc.org.* or a
// policy key, is refused: the platform sets those itself.
//
//nolint:gochecknoglobals // read-only allow-lists
var (
	oidcConfigKeys = toSet("clientId", "clientSecret", "clientAuthMethod", "authorizationUrl", "tokenUrl", "logoutUrl",
		"userInfoUrl", "issuer", "jwksUrl", "useJwksUrl", "validateSignature", "defaultScope", "syncMode", "pkceEnabled",
		"pkceMethod", "uiLocales", "prompt", "loginHint", "disableUserInfo", "metadataDescriptorUrl", "allowedClockSkew")
	samlConfigKeys = toSet("entityId", "singleSignOnServiceUrl", "singleLogoutServiceUrl", "nameIDPolicyFormat", "principalType",
		"principalAttribute", "signingCertificate", "validateSignature", "wantAssertionsSigned", "wantAssertionsEncrypted",
		"wantAuthnRequestsSigned", "postBindingResponse", "postBindingAuthnRequest", "postBindingLogout", "forceAuthn",
		"signatureAlgorithm", "allowCreate", "syncMode", "metadataDescriptorUrl")
)

func toSet(keys ...string) map[string]bool {
	m := make(map[string]bool, len(keys))
	for _, k := range keys {
		m[k] = true
	}
	return m
}

func checkIdPConfig(spec IdPSpec) error {
	allowed := oidcConfigKeys
	if spec.ProviderID == "saml" {
		allowed = samlConfigKeys
	}
	for k := range spec.Config {
		if !allowed[k] {
			return fmt.Errorf("%w: idp config key %q", identity.ErrInvalidInput, k)
		}
	}
	return nil
}
