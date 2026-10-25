// Package auth is the core API's authentication middleware (ADR-001 A.1,
// PLAN-001 T17). Every /api/v1 operation runs behind it: it reads the bearer
// token, resolves (tenant, principal, groups, idp) through the identity port,
// and puts the principal in the request context, where handlers find it with
// identity.FromContext and pass its TenantID to tenancy.InTenantTx.
//
// The tenant comes only from the resolver. Nothing here, or downstream, reads
// a tenant from a header, the query, the body or baggage (I2): a request that
// names another tenant still runs as the token's tenant (CI-2 forged-tenant
// paths, internal/tenancy/isolation).
//
// The raw token and its claims are never logged; a refusal logs only the
// resolver's fixed reason.
package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// Results recorded on nava.auth.requests, beside the two 401 codes. Never a
// tenant or principal: those are unbounded and are log fields only.
const (
	resultOK    = "ok"
	resultError = "error"
)

// Authenticator builds the middleware.
type Authenticator struct {
	resolver identity.IdentityResolver
	reg      *problem.Registry
	log      *slog.Logger
	requests metric.Int64Counter
}

// New returns an Authenticator. mp must not be nil.
func New(resolver identity.IdentityResolver, reg *problem.Registry, log *slog.Logger, mp metric.MeterProvider) (*Authenticator, error) {
	if resolver == nil || reg == nil || log == nil || mp == nil {
		return nil, errors.New("auth: resolver, registry, logger and meter provider are required")
	}
	c, err := mp.Meter("github.com/Navaladi2019/TicketingSystem/internal/platform/auth").Int64Counter(
		"nava.auth.requests",
		metric.WithDescription("API requests by authentication result."),
		metric.WithUnit("{request}"))
	if err != nil {
		return nil, err
	}
	return &Authenticator{resolver: resolver, reg: reg, log: log, requests: c}, nil
}

// Middleware authenticates every request it wraps. A missing or malformed
// Authorization header and a token the resolver refuses get 401 problem+json
// (invalid_token or tenant_unresolved) with WWW-Authenticate: Bearer; a
// resolver that cannot decide gets 500.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		raw, ok := bearer(r.Header)
		if !ok {
			a.refuse(ctx, w, problem.InvalidToken, "no bearer token")
			a.reg.Write(w, r, problem.InvalidToken)
			return
		}
		p, err := a.resolver.Resolve(ctx, raw)
		switch {
		case errors.Is(err, identity.ErrInvalidToken):
			a.refuse(ctx, w, problem.InvalidToken, err.Error())
			a.reg.Write(w, r, problem.InvalidToken)
			return
		case errors.Is(err, identity.ErrTenantUnresolved):
			a.refuse(ctx, w, problem.TenantUnresolved, err.Error())
			a.reg.Write(w, r, problem.TenantUnresolved)
			return
		case err != nil:
			a.requests.Add(ctx, 1, metric.WithAttributes(attribute.String("result", resultError)))
			a.reg.WriteError(w, r, a.log, problem.InternalError, err, slog.String("route", r.Pattern))
			return
		}
		a.requests.Add(ctx, 1, metric.WithAttributes(attribute.String("result", resultOK)))
		next.ServeHTTP(w, r.WithContext(identity.NewContext(ctx, p)))
	})
}

// refuse counts and logs a 401 and sets its challenge; the caller writes the
// problem. reason is the resolver's fixed text (the sentinel and a bounded
// reason), never token or claim content.
func (a *Authenticator) refuse(ctx context.Context, w http.ResponseWriter, code problem.Code, reason string) {
	a.requests.Add(ctx, 1, metric.WithAttributes(attribute.String("result", string(code))))
	a.log.InfoContext(ctx, "authentication refused", slog.String("code", string(code)), slog.String("reason", reason))
	w.Header().Set("WWW-Authenticate", "Bearer")
}

// bearer returns the token of the one Authorization header, scheme Bearer
// (case-insensitive, RFC 6750). Two headers, another scheme or an empty token
// are refused.
func bearer(h http.Header) (string, bool) {
	vals := h.Values("Authorization")
	if len(vals) != 1 {
		return "", false
	}
	scheme, tok, ok := strings.Cut(vals[0], " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	tok = strings.TrimSpace(tok)
	if tok == "" || strings.ContainsAny(tok, " \t") {
		return "", false
	}
	return tok, true
}
