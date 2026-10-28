package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
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
}

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

// publicRoutes is the allow-list of /api/v1 routes served without a token. The
// match is exact on the decoded path (no prefix, case folding or trailing
// slash), and on GET or HEAD only.
//
//nolint:gochecknoglobals // fixed allow-list, never written
var publicRoutes = map[string]bool{
	"/api/v1/openapi.yaml": true,
	"/api/v1/docs":         true,
}

// authGate authenticates a request before anything reads its input: it wraps
// the whole router, outside the generated parameter binding. Public routes
// pass; operator routes go to the operator authentication; every other path
// under /api/v1 fails closed through the tenant authentication, so an unknown path with no
// token is 401, never 404 or 400. On the operator-only process a
// non-operator path is authenticated and then answered 404.
func authGate(reg *problem.Registry, operatorOnly bool, tenant, operator func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		tenantNext := next
		if operatorOnly {
			tenantNext = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reg.Write(w, r, problem.NotFound) })
		}
		// The middleware hands next a copy of the request carrying the
		// principal, and ServeMux sets r.Pattern on that copy. The access log
		// and span read it from the original, so copy it back.
		through := func(auth func(http.Handler) http.Handler, next http.Handler, w http.ResponseWriter, r *http.Request) {
			var inner *http.Request
			auth(http.HandlerFunc(func(w http.ResponseWriter, r2 *http.Request) {
				inner = r2
				next.ServeHTTP(w, r2)
			})).ServeHTTP(w, r)
			if inner != nil {
				r.Pattern = inner.Pattern
			}
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case (r.Method == http.MethodGet || r.Method == http.MethodHead) && publicRoutes[r.URL.Path]:
				next.ServeHTTP(w, r)
			case !strings.HasPrefix(r.URL.Path, "/api/v1"):
				next.ServeHTTP(w, r) // not contract: the router answers 404
			case strings.HasPrefix(r.URL.Path, operatorPrefix):
				through(operator, next, w, r)
			default:
				through(tenant, tenantNext, w, r)
			}
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
