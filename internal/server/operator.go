package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/ratelimit"
)

// operatorPrefix is the part of the contract served to operators only. Its
// routes authenticate through the OperatorResolver, never the tenant resolver
// alone (D-P12).
const operatorPrefix = "/api/v1/operator/"

// maxOperatorBody bounds an operator request body; the IdP config is the
// largest field.
const maxOperatorBody = 256 << 10

// Onboarder is the onboarding service the operator route calls.
type Onboarder interface {
	Onboard(ctx context.Context, operator identity.Operator, key string, hash []byte, req onboard.Request) (onboard.Result, error)
}

// Option configures NewHandler.
type Option func(*options)

type options struct {
	operators identity.OperatorResolver
	onboarder Onboarder
	// operatorOnly makes every route but /api/v1/operator/* answer 404: the
	// nava-onboard process mounts the operator routes only.
	operatorOnly bool
	// rate and shared configure the per-tenant limiter (T24).
	rate   config.RateLimit
	shared ratelimit.Limiter
	// pool (nava_app) and cursors serve the tenant routes that read or write
	// the database.
	pool    *pgxpool.Pool
	cursors *cursor.Keyring
}

// WithPool gives the tenant routes their database (the nava_app pool). Tenant
// writes also need it for the Idempotency-Key store.
func WithPool(p *pgxpool.Pool) Option { return func(o *options) { o.pool = p } }

// WithCursors sets the keyring that signs list cursors. Without it the list
// endpoints answer 501 not_implemented.
func WithCursors(k *cursor.Keyring) Option { return func(o *options) { o.cursors = k } }

// OperatorOnly restricts the handler to the operator routes.
func OperatorOnly() Option { return func(o *options) { o.operatorOnly = true } }

// WithOperator turns the operator routes on: operators authenticates their
// tokens and onboarder serves POST /api/v1/operator/tenants. Without it those
// routes answer 501 not_implemented.
func WithOperator(operators identity.OperatorResolver, onboarder Onboarder) Option {
	return func(o *options) { o.operators, o.onboarder = operators, onboarder }
}

type operatorCtxKey struct{}

type operatorCall struct {
	operator identity.Operator
	hash     []byte
}

// routeAuth sends operator routes to the operator authentication and every
// other route to the tenant authentication. On the operator-only process a
// non-operator route is authenticated too, then answered 404.
func routeAuth(reg *problem.Registry, operatorOnly bool, tenant, operator func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		t, o := tenant(next), operator(next)
		if operatorOnly {
			t = tenant(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reg.Write(w, r, problem.NotFound) }))
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, operatorPrefix) {
				o.ServeHTTP(w, r)
				return
			}
			t.ServeHTTP(w, r)
		})
	}
}

// operatorAuth authenticates an operator token. A valid tenant token is 403, a
// missing or invalid token is 401, and an operator route with no operator
// resolver configured is 501. It also reads the body, fingerprints the request
// with idempotency.RequestHash (the path without its query, so a stray query
// parameter does not change the fingerprint), and puts the operator and hash in
// the context for the handler.
func operatorAuth(reg *problem.Registry, log *slog.Logger, ops identity.OperatorResolver, tenants identity.IdentityResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if ops == nil {
				reg.WriteError(w, r, log, problem.NotImplemented, errors.New("operator routes not configured"))
				return
			}
			raw, ok := bearerToken(r.Header)
			if !ok {
				w.Header().Set("WWW-Authenticate", "Bearer")
				reg.Write(w, r, problem.InvalidToken)
				return
			}
			op, err := ops.ResolveOperator(ctx, raw)
			switch {
			case errors.Is(err, identity.ErrInvalidToken), errors.Is(err, identity.ErrTenantUnresolved):
				if _, terr := tenants.Resolve(ctx, raw); terr == nil {
					log.InfoContext(ctx, "operator route refused a tenant token")
					reg.Write(w, r, problem.Forbidden)
					return
				}
				w.Header().Set("WWW-Authenticate", "Bearer")
				reg.Write(w, r, problem.InvalidToken)
				return
			case err != nil:
				reg.WriteError(w, r, log, problem.InternalError, err, slog.String("route", r.Pattern))
				return
			}
			body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxOperatorBody))
			if err != nil {
				reg.Write(w, r, problem.ValidationFailed, problem.FieldError{Pointer: "/body", Rule: "maxLength"})
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			call := operatorCall{operator: op, hash: idempotency.RequestHash(r.Method, r.URL.Path, body)}
			next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, operatorCtxKey{}, call)))
		})
	}
}

// bearerToken is the one Authorization header's Bearer token.
func bearerToken(h http.Header) (string, bool) {
	vals := h.Values("Authorization")
	if len(vals) != 1 {
		return "", false
	}
	scheme, tok, ok := strings.Cut(vals[0], " ")
	tok = strings.TrimSpace(tok)
	if !ok || !strings.EqualFold(scheme, "Bearer") || tok == "" || strings.ContainsAny(tok, " \t") {
		return "", false
	}
	return tok, true
}

// OnboardTenant serves POST /api/v1/operator/tenants. The tenant id comes from
// the service, derived from the operator and the Idempotency-Key; no field of
// the request, the query or the headers reaches it.
func (h *handlers) OnboardTenant(ctx context.Context, req api.OnboardTenantRequestObject) (api.OnboardTenantResponseObject, error) {
	call, ok := ctx.Value(operatorCtxKey{}).(operatorCall)
	if !ok || h.onboarder == nil {
		return nil, errNoPrincipal
	}
	if req.Body == nil {
		return nil, &problemError{problem.ValidationFailed, []problem.FieldError{{Pointer: "/body", Rule: "required"}}, errors.New("no body")}
	}
	in := onboard.Request{
		AdminEmail:   req.Body.AdminEmail,
		EmailDomains: req.Body.EmailDomains,
		IdP: onboard.IdPRequest{
			ProviderID: string(req.Body.IdentityProvider.ProviderId),
			Config:     req.Body.IdentityProvider.Config,
		},
	}
	if req.Body.IdentityProvider.DisplayName != nil {
		in.IdP.DisplayName = *req.Body.IdentityProvider.DisplayName
	}
	res, err := h.onboarder.Onboard(ctx, call.operator, req.Params.IdempotencyKey, call.hash, in)
	if err != nil {
		return nil, onboardError(err)
	}
	out := api.OnboardedTenant{TenantId: res.TenantID, State: api.Active}
	if res.Replayed {
		return api.OnboardTenant200JSONResponse(out), nil
	}
	return api.OnboardTenant201JSONResponse(out), nil
}

// problemError carries the registered code a handler error maps to.
type problemError struct {
	code   problem.Code
	fields []problem.FieldError
	err    error
}

func (e *problemError) Error() string { return e.err.Error() }
func (e *problemError) Unwrap() error { return e.err }

// onboardError maps a service error to a problem.
func onboardError(err error) error {
	var fe *onboard.FieldError
	switch {
	case errors.As(err, &fe) && strings.HasPrefix(fe.Pointer, "/identity_provider/"):
		return &problemError{problem.InvalidIdPConfig, []problem.FieldError{{Pointer: fe.Pointer, Rule: "invalid"}}, err}
	case errors.As(err, &fe):
		return &problemError{problem.ValidationFailed, []problem.FieldError{{Pointer: fe.Pointer, Rule: "invalid"}}, err}
	case errors.Is(err, onboard.ErrKeyReuse):
		return &problemError{problem.IdempotencyKeyReuse, nil, err}
	case errors.Is(err, onboard.ErrDomainClaimed):
		return &problemError{problem.DomainClaimed, nil, err}
	case errors.Is(err, onboard.ErrConflict):
		return &problemError{problem.OnboardingConflict, nil, err}
	}
	return err
}
