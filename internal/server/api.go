package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

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

// newAPI registers the /api/v1 routes of the embedded contract on mux, each
// behind the authentication middleware and then the validator, and returns
// the error-contract helper whose Wrap goes around mux. The spec and docs
// routes are public.
func newAPI(log *slog.Logger, tel *telemetry.Telemetry, resolver identity.IdentityResolver, mux *http.ServeMux) (*apihttp.API, error) {
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
	api.HandlerWithOptions(
		api.NewStrictHandlerWithOptions(&handlers{reg: reg, log: log}, nil, api.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  a.StrictRequestError,
			ResponseErrorHandlerFunc: a.StrictResponseError,
		}),
		api.StdHTTPServerOptions{
			BaseURL:    baseURL,
			BaseRouter: mux,
			// The last one is outermost: an unauthenticated request gets 401
			// before the validator looks at its input.
			Middlewares:      []api.MiddlewareFunc{a.Middleware(), authn.Middleware},
			ErrorHandlerFunc: a.ErrorHandler,
		},
	)
	mux.Handle("GET /api/v1/openapi.yaml", specHandler(specData))
	mux.Handle("GET /api/v1/docs", docsHandler(spec))
	return a, nil
}

// handlers implements the generated strict server. Operations the contract
// declares but a later task implements answer 501 not_implemented.
type handlers struct {
	reg *problem.Registry
	log *slog.Logger
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
	return api.GetMe200JSONResponse{UserId: p.PrincipalID, TenantId: p.TenantID, Groups: groups, IdpAlias: p.IdPAlias}, nil
}
