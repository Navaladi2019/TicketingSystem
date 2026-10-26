// Package keycloak is the Keycloak adapter of the identity port (ADR-001 A.1,
// A.4). It is the only package that imports a JWT/JWKS library or a Keycloak
// client, calls the Keycloak admin API, or reads Keycloak-specific claims
// (organization, idp, realm_access, resource_access); depguard and navalint
// enforce that (CI-11).
//
// Access-token validation (A.1 step 1, TICK-94 item 11):
//   - iss must be on the configured allow-list, checked before any HTTP
//     request, so a token cannot make the service fetch from a host it names;
//   - the signing key comes only from the JWKS that iss's own discovery
//     document names, so a key of realm X never verifies a token with iss=Y;
//   - alg must be RS256 and kid must be set;
//   - exp is required; exp and nbf allow 30 s of skew;
//   - typ must be Bearer and azp set, so an ID token is rejected;
//   - one audience per resolver: nava-api for tenants, nava-operator-api for
//     operators;
//   - an operator must also be on Config.Operators, so a role mapping alone
//     never makes anyone an operator (AR-030).
//
// An unknown kid forces at most one JWKS fetch per issuer per 30 s, shared by
// concurrent callers, then looks once more; so a rotated key is accepted
// without a restart, and unauthenticated callers cannot make the service
// fetch keys on every request.
package keycloak

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// Defaults and limits.
const (
	// DefaultAudience is the audience of tenant access tokens.
	DefaultAudience = "nava-api"
	// DefaultOperatorAudience is the audience of operator access tokens.
	DefaultOperatorAudience = "nava-operator-api"
	// OperatorRole is the realm role an operator token must carry (D-P12).
	OperatorRole = "nava-operator"

	skew          = 30 * time.Second
	maxTokenBytes = 16 << 10
)

// Config configures the adapter.
type Config struct {
	// Issuers is the exact iss allow-list, such as
	// https://id.example.com/realms/nava. Required.
	Issuers []string
	// Audience is the tenant token audience. Default DefaultAudience.
	Audience string
	// OperatorAudience is the operator token audience. Default
	// DefaultOperatorAudience.
	OperatorAudience string
	// Logger defaults to slog.Default().
	Logger *slog.Logger
	// MeterProvider records nava.identity.jwks.fetch. Nil records nothing.
	MeterProvider metric.MeterProvider
	// Store holds the tenant bindings and principals Resolve reads (A.1
	// steps 3-4). Without one, Resolve refuses every token: an operator-only
	// adapter needs none.
	Store identity.Store
	// Operators is the operator allow-list, from platform config and never
	// from a request (AR-030). ResolveOperator accepts a token only if its
	// (iss, sub) is listed, as well as holding the nava-operator role. Empty
	// refuses every operator token.
	Operators []OperatorSubject
}

// OperatorSubject is one operator allow-list entry: the IdP user id (sub) in
// one trusted issuer. The issuer is part of the key, so a user of another
// trusted realm with the same id is not an operator.
type OperatorSubject struct {
	Issuer  string
	Subject string
}

// Adapter validates access tokens. It is an identity.IdentityResolver and an
// identity.OperatorResolver. Close it to stop the key refresh.
type Adapter struct {
	keys             *keySource
	store            identity.Store
	issuers          []string
	audience         string
	operatorAudience string
	operators        map[OperatorSubject]bool
	log              *slog.Logger
	now              func() time.Time
}

var (
	_ identity.IdentityResolver = (*Adapter)(nil)
	_ identity.OperatorResolver = (*Adapter)(nil)
)

// New builds the adapter. Every discovery and JWKS fetch goes through out,
// whose timeouts bound the background refresh too (TICK-94 item 2). Keys are
// fetched on first use of each issuer. ctx bounds the refresh workers' life.
func New(ctx context.Context, cfg Config, out *telemetry.OutboundClient) (*Adapter, error) {
	return newAdapter(ctx, cfg, out, time.Now)
}

func newAdapter(ctx context.Context, cfg Config, out *telemetry.OutboundClient, now func() time.Time) (*Adapter, error) {
	if out == nil {
		return nil, errors.New("keycloak: an outbound client is required")
	}
	if len(cfg.Issuers) == 0 {
		return nil, errors.New("keycloak: at least one issuer is required")
	}
	for _, iss := range cfg.Issuers {
		if iss == "" {
			return nil, errors.New("keycloak: empty issuer")
		}
	}
	if cfg.Audience == "" {
		cfg.Audience = DefaultAudience
	}
	if cfg.OperatorAudience == "" {
		cfg.OperatorAudience = DefaultOperatorAudience
	}
	if cfg.Audience == cfg.OperatorAudience {
		return nil, errors.New("keycloak: tenant and operator audiences must differ")
	}
	operators := make(map[OperatorSubject]bool, len(cfg.Operators))
	for _, o := range cfg.Operators {
		if o.Subject == "" {
			return nil, errors.New("keycloak: empty operator subject")
		}
		if !slices.Contains(cfg.Issuers, o.Issuer) {
			return nil, errors.New("keycloak: operator issuer is not a trusted issuer")
		}
		operators[o] = true
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.MeterProvider == nil {
		cfg.MeterProvider = noop.NewMeterProvider()
	}
	keys, err := newKeySource(ctx, out, cfg.Issuers, now, cfg.Logger, cfg.MeterProvider)
	if err != nil {
		return nil, err
	}
	return &Adapter{
		keys:             keys,
		store:            cfg.Store,
		issuers:          slices.Clone(cfg.Issuers),
		audience:         cfg.Audience,
		operatorAudience: cfg.OperatorAudience,
		operators:        operators,
		log:              cfg.Logger,
		now:              now,
	}, nil
}

// Close stops the key cache.
func (a *Adapter) Close(ctx context.Context) error { return a.keys.close(ctx) }

// Resolve validates a tenant access token and resolves its tenant (A.1
// steps 1-4). Every check fails closed:
//   - exactly one organization (step 2);
//   - the binding of (iss, organization id) gives the tenant (step 4), never
//     a claim value (I2);
//   - an idp claim, if present, must be one of the tenant's bound aliases
//     (step 3);
//   - independently of the claim, the principal synced for sub must exist,
//     its federated links must be known (not NULL) and each one a bound
//     alias, and it must be enabled (step 3, TICK-69 review).
//
// A refusal wraps ErrTenantUnresolved or ErrInvalidToken with a fixed reason
// that never quotes a claim. A store failure wraps neither, for a 500.
func (a *Adapter) Resolve(ctx context.Context, raw string) (identity.Principal, error) {
	tok, err := a.verify(ctx, raw, a.audience)
	if err != nil {
		return identity.Principal{}, err
	}
	orgID, err := organizationID(tok)
	if err != nil {
		return identity.Principal{}, err
	}
	idp, err := idpAlias(tok)
	if err != nil {
		return identity.Principal{}, err
	}
	if a.store == nil {
		return identity.Principal{}, unresolved("no store")
	}
	iss, _ := tok.Issuer()
	sub, _ := tok.Subject()

	b, err := a.store.Binding(ctx, iss, orgID)
	switch {
	case errors.Is(err, identity.ErrNoBinding):
		return identity.Principal{}, unresolved("no tenant binding")
	case err != nil:
		return identity.Principal{}, fmt.Errorf("keycloak: resolve: %w", err)
	}
	if idp != "" && !slices.Contains(b.IdPAliases, idp) {
		return identity.Principal{}, unresolved("idp not bound")
	}

	p, err := a.store.Principal(ctx, b.TenantID, sub)
	switch {
	case errors.Is(err, identity.ErrNoPrincipal):
		return identity.Principal{}, unresolved("principal not synced")
	case err != nil:
		return identity.Principal{}, fmt.Errorf("keycloak: resolve: %w", err)
	}
	if !p.LinksKnown {
		return identity.Principal{}, unresolved("federated links unknown")
	}
	for _, l := range p.FederatedIdPs {
		if !slices.Contains(b.IdPAliases, l) {
			return identity.Principal{}, unresolved("federated link not bound")
		}
	}
	if !p.Enabled {
		return identity.Principal{}, unresolved("principal disabled")
	}
	groups := p.Groups
	if groups == nil {
		groups = []uuid.UUID{}
	}
	return identity.Principal{TenantID: b.TenantID, PrincipalID: p.ID, Groups: groups, IdPAlias: idp}, nil
}

// ResolveOperator validates an operator access token: audience
// nava-operator-api, realm role nava-operator, and (iss, sub) on the
// operator allow-list, all required.
func (a *Adapter) ResolveOperator(ctx context.Context, raw string) (identity.Operator, error) {
	tok, err := a.verify(ctx, raw, a.operatorAudience)
	if err != nil {
		return identity.Operator{}, err
	}
	if !hasRealmRole(tok, OperatorRole) {
		return identity.Operator{}, invalid("not an operator")
	}
	iss, _ := tok.Issuer()
	sub, _ := tok.Subject()
	if !a.operators[OperatorSubject{Issuer: iss, Subject: sub}] {
		// A valid token holding the role for an unlisted user means the role
		// was mapped outside the platform's control: worth an alert. The
		// subject is not logged; the IdP's admin events name the user.
		a.log.WarnContext(ctx, "operator token refused: subject not on the operator allow-list")
		return identity.Operator{}, invalid("not an allow-listed operator")
	}
	return identity.Operator{PrincipalID: sub}, nil
}

func invalid(reason string) error {
	return fmt.Errorf("%w: %s", identity.ErrInvalidToken, reason)
}

func unresolved(reason string) error {
	return fmt.Errorf("%w: %s", identity.ErrTenantUnresolved, reason)
}

// verify checks raw for audience aud and returns its claims. Every error
// wraps identity.ErrInvalidToken with a fixed reason; library error text,
// which can quote token values, never leaves.
func (a *Adapter) verify(ctx context.Context, raw, aud string) (jwt.Token, error) {
	if raw == "" || len(raw) > maxTokenBytes {
		return nil, invalid("malformed")
	}
	// Compact serialisation only: one protected header, one signature.
	if strings.Count(raw, ".") != 2 || strings.HasPrefix(raw, "{") {
		return nil, invalid("malformed")
	}
	msg, err := jws.ParseString(raw)
	if err != nil || len(msg.Signatures()) != 1 {
		return nil, invalid("malformed")
	}
	hdr := msg.Signatures()[0].ProtectedHeaders()
	if alg, ok := hdr.Algorithm(); !ok || alg != jwa.RS256() {
		return nil, invalid("alg")
	}
	kid, ok := hdr.KeyID()
	if !ok || kid == "" {
		return nil, invalid("kid")
	}

	// The issuer picks the key set, so read it before verifying, and check
	// it against the allow-list before any HTTP request.
	unverified, err := jwt.ParseInsecure([]byte(raw))
	if err != nil {
		return nil, invalid("malformed")
	}
	iss, _ := unverified.Issuer()
	if !slices.Contains(a.issuers, iss) {
		return nil, invalid("issuer")
	}

	key, err := a.keys.key(ctx, iss, kid)
	switch {
	case errors.Is(err, errUnknownKID):
		return nil, invalid("kid")
	case err != nil:
		return nil, invalid("keys unavailable")
	}
	if !keyFitsRS256(key) {
		return nil, invalid("kid")
	}

	tok, err := jwt.Parse([]byte(raw),
		jwt.WithKey(jwa.RS256(), key),
		jwt.WithValidate(true),
		jwt.WithIssuer(iss),
		jwt.WithAudience(aud),
		jwt.WithRequiredClaim("exp"),
		jwt.WithRequiredClaim("sub"),
		jwt.WithAcceptableSkew(skew),
		jwt.WithClock(jwt.ClockFunc(a.now)),
	)
	if err != nil {
		return nil, invalid(parseReason(err))
	}
	if typ, _ := stringClaim(tok, "typ"); typ != "Bearer" {
		return nil, invalid("typ")
	}
	if azp, _ := stringClaim(tok, "azp"); azp == "" {
		return nil, invalid("azp")
	}
	return tok, nil
}

// keyFitsRS256 rejects a key that is not an RSA signing key for RS256, so an
// HS256 header can never be checked against an RSA key's bytes.
func keyFitsRS256(k jwk.Key) bool {
	if k.KeyType() != jwa.RSA() {
		return false
	}
	if alg, ok := k.Algorithm(); ok && alg.String() != jwa.RS256().String() {
		return false
	}
	if use, ok := k.KeyUsage(); ok && use != "sig" {
		return false
	}
	return true
}

func parseReason(err error) string {
	switch {
	case errors.Is(err, jwt.TokenExpiredError()):
		return "expired"
	case errors.Is(err, jwt.TokenNotYetValidError()):
		return "not yet valid"
	case errors.Is(err, jwt.InvalidIssuedAtError()):
		return "iat"
	case errors.Is(err, jwt.InvalidAudienceError()):
		return "audience"
	case errors.Is(err, jwt.InvalidIssuerError()):
		return "issuer"
	case errors.Is(err, jwt.ValidateError()):
		return "claims"
	}
	return "signature"
}

func stringClaim(tok jwt.Token, name string) (string, bool) {
	var v any
	if err := tok.Get(name, &v); err != nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// organizationID reads Keycloak's organization claim: an object keyed by
// alias, of which there must be exactly one, holding the organization id
// (A.1 step 2). The id, not the alias, is the binding key.
func organizationID(tok jwt.Token) (string, error) {
	var v any
	if err := tok.Get("organization", &v); err != nil {
		return "", fmt.Errorf("%w: no organization", identity.ErrTenantUnresolved)
	}
	orgs, ok := v.(map[string]any)
	if !ok || len(orgs) != 1 {
		return "", fmt.Errorf("%w: organization count", identity.ErrTenantUnresolved)
	}
	for _, o := range orgs {
		org, ok := o.(map[string]any)
		if !ok {
			break
		}
		if id, ok := org["id"].(string); ok && id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("%w: organization id", identity.ErrTenantUnresolved)
}

// idpAlias reads the idp claim: the brokering IdP's alias, written by the
// realm's session note mapper (A.1 realm export). Absent means a local login
// and returns "". A claim that is present but not a non-empty string is
// refused, never read as absent.
func idpAlias(tok jwt.Token) (string, error) {
	if !tok.Has("idp") {
		return "", nil
	}
	if s, ok := stringClaim(tok, "idp"); ok && s != "" {
		return s, nil
	}
	return "", unresolved("idp claim")
}

// hasRealmRole reports whether realm_access.roles holds role.
func hasRealmRole(tok jwt.Token, role string) bool {
	var v any
	if err := tok.Get("realm_access", &v); err != nil {
		return false
	}
	ra, ok := v.(map[string]any)
	if !ok {
		return false
	}
	roles, ok := ra["roles"].([]any)
	if !ok {
		return false
	}
	for _, r := range roles {
		if s, ok := r.(string); ok && s == role {
			return true
		}
	}
	return false
}
