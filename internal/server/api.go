package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/apihttp"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/auth"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// baseURL is oapi-codegen's BaseURL and the validator's Options.Prefix. The
// contract's paths already start with /api/v1, so both are empty.
const baseURL = ""

// newAPI registers the /api/v1 routes of the embedded contract on mux and
// returns the handler to serve. Authentication (routeAuth) wraps the whole
// mux, outside the generated router: the generated wrapper binds path, query
// and header parameters before it runs a MiddlewareFunc, so an auth
// middleware inside it would answer 400 to an unauthenticated request with a
// bad or missing parameter. Behind authentication each route runs the
// validator. The spec and docs routes are public, and a request no route
// matches gets the 404/405 problem without authentication.
func newAPI(log *slog.Logger, tel *telemetry.Telemetry, resolver identity.IdentityResolver, paths identity.GroupPathReader, mux *http.ServeMux, opts options) (http.Handler, error) {
	specData, err := api.Files.ReadFile("openapi.yaml")
	if err != nil {
		return nil, fmt.Errorf("read openapi.yaml: %w", err)
	}
	spec, err := apihttp.LoadSpec(specData)
	if err != nil {
		return nil, err
	}
	errData, err := api.Files.ReadFile("errors.yaml")
	if err != nil {
		return nil, fmt.Errorf("read errors.yaml: %w", err)
	}
	reg, err := problem.LoadRegistry(errData)
	if err != nil {
		return nil, err
	}
	authn, err := auth.New(resolver, reg, log, tel.MeterProvider)
	if err != nil {
		return nil, err
	}

	a := apihttp.New(spec, reg, log, baseURL)
	h := &handlers{reg: reg, log: log, onboarder: opts.onboarder, paths: paths}
	api.HandlerWithOptions(
		api.NewStrictHandlerWithOptions(h, nil, api.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  a.StrictRequestError,
			ResponseErrorHandlerFunc: h.responseError(a),
		}),
		api.StdHTTPServerOptions{
			BaseURL:          baseURL,
			BaseRouter:       mux,
			Middlewares:      []api.MiddlewareFunc{a.Middleware()},
			ErrorHandlerFunc: a.ErrorHandler,
		},
	)
	mux.Handle(specPattern, specHandler(specData))
	mux.Handle(docsPattern, docsHandler(spec))

	served := a.Wrap(mux)
	authed := routeAuth(reg, opts.operatorOnly, authn.Middleware, operatorAuth(reg, log, opts.operators, resolver))(served)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pattern := mux.Handler(r)
		if pattern == "" || pattern == specPattern || pattern == docsPattern {
			served.ServeHTTP(w, r)
			return
		}
		r.Pattern = pattern // the authentication logs name the route
		authed.ServeHTTP(w, r)
	}), nil
}

const (
	specPattern = "GET /api/v1/openapi.yaml"
	docsPattern = "GET /api/v1/docs"
)

// handlers implements the generated strict server. Operations the contract
// declares but a later task implements answer 501 not_implemented.
type handlers struct {
	reg       *problem.Registry
	log       *slog.Logger
	onboarder Onboarder
	// paths is read by GetMe only (ADR-002).
	paths identity.GroupPathReader
}

// errNoPrincipal means a handler ran outside the authentication middleware:
// a wiring bug, answered 500.
var errNoPrincipal = errors.New("server: no resolved principal in the request context")

// GetMe returns the caller as the middleware resolved it (T17).
func (h *handlers) GetMe(ctx context.Context, _ api.GetMeRequestObject) (api.GetMeResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok {
		return nil, errNoPrincipal
	}
	groups := p.Groups
	if groups == nil {
		groups = []uuid.UUID{}
	}
	raw, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID)
	if err != nil {
		return nil, err
	}
	// Strip the tenant prefix; an entry without it is dropped and logged
	// without its path, so one bad row cannot take /me down (ADR-002 section 3).
	prefix := identity.TenantGroupPrefix(p.TenantID)
	paths := make([]api.GroupPath, 0, len(raw))
	for _, gp := range raw {
		rest, ok := strings.CutPrefix(gp.Path, prefix)
		if !ok {
			h.log.ErrorContext(ctx, "group path outside tenant prefix",
				slog.String("tenant_id", p.TenantID.String()), slog.String("group_id", gp.ID.String()))
			continue
		}
		paths = append(paths, api.GroupPath{Id: gp.ID, Path: "/" + rest})
	}
	return api.GetMe200JSONResponse{UserId: p.PrincipalID, TenantId: p.TenantID, Groups: groups, IdpAlias: p.IdPAlias, GroupPaths: paths}, nil
}

// responseError answers a handler's problemError with its registered code and
// hands every other error to the shared handler.
func (h *handlers) responseError(a *apihttp.API) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		var pe *problemError
		if errors.As(err, &pe) {
			if pe.code == problem.ValidationFailed {
				h.reg.Write(w, r, problem.ValidationFailed, pe.fields...)
				return
			}
			h.reg.WriteError(w, r, h.log, pe.code, pe.err)
			return
		}
		a.StrictResponseError(w, r, err)
	}
}
