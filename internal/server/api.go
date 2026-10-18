package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/apihttp"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// baseURL is oapi-codegen's BaseURL and the validator's Options.Prefix. The
// contract's paths already start with /api/v1, so both are empty.
const baseURL = ""

// newAPI registers the /api/v1 routes of the embedded contract on mux, with
// the validator as a per-route middleware, and returns the error-contract
// helper whose Wrap goes around mux.
func newAPI(log *slog.Logger, mux *http.ServeMux) (*apihttp.API, error) {
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

	a := apihttp.New(spec, reg, log, baseURL)
	api.HandlerWithOptions(
		api.NewStrictHandlerWithOptions(&handlers{reg: reg}, nil, api.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  a.StrictRequestError,
			ResponseErrorHandlerFunc: a.StrictResponseError,
		}),
		api.StdHTTPServerOptions{
			BaseURL:          baseURL,
			BaseRouter:       mux,
			Middlewares:      []api.MiddlewareFunc{a.Middleware()},
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
}

// GetMe is declared by T20 and implemented by T17.
func (h *handlers) GetMe(ctx context.Context, _ api.GetMeRequestObject) (api.GetMeResponseObject, error) {
	b := h.reg.Body(ctx, problem.NotImplemented)
	return api.GetMe501ApplicationProblemPlusJSONResponse{
		Type: b.Type, Title: b.Title, Status: b.Status,
		Code: string(b.Code), Detail: b.Detail, Instance: b.Instance,
	}, nil
}
