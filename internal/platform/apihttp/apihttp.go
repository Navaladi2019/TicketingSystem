// Package apihttp wires an oapi-codegen std-http-server router to the error
// contract: every error response is problem+json with a registered code, and
// none echoes request input.
//
// Three defaults of the generated code and the request validator would break
// that contract, and this package replaces each of them:
//
//  1. format: uuid is not checked by default. The formats are registered per
//     validation (openapi3.WithStringFormatValidators), never through the
//     package-global DefineStringFormatValidator.
//  2. Validation errors echo input (Value, and Reason for pattern). The body
//     is built only from SchemaField and JSONPointer, never Reason or Error().
//  3. Several error paths write text/plain. ErrorHandlerFunc, the strict
//     request and response handlers, the validator's ErrorHandlerWithOpts and
//     a 404/405 wrapper around the mux all write problem+json instead.
//
// The validator is a per-route middleware (StdHTTPServerOptions.Middlewares),
// so it runs after ServeMux set r.Pattern and a 400 keeps its http.route.
package apihttp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// API holds what the error contract needs.
type API struct {
	spec    *openapi3.T
	reg     *problem.Registry
	log     *slog.Logger
	baseURL string
}

// New returns an API for spec. baseURL must equal the BaseURL given to the
// generated StdHTTPServerOptions: it is the validator's Options.Prefix.
func New(spec *openapi3.T, reg *problem.Registry, log *slog.Logger, baseURL string) *API {
	return &API{spec: spec, reg: reg, log: log, baseURL: baseURL}
}

// Middleware returns the request validator, for StdHTTPServerOptions.Middlewares.
func (a *API) Middleware() func(http.Handler) http.Handler {
	// The server list is host validation we do not want, and spec is shared.
	spec := *a.spec
	spec.Servers = nil
	return nethttpmiddleware.OapiRequestValidatorWithOptions(&spec, &nethttpmiddleware.Options{
		Options: openapi3filter.Options{
			SchemaValidationOptions: []openapi3.SchemaValidationOption{
				openapi3.WithStringFormatValidators(stringFormats()),
			},
			MultiError: true,
		},
		Prefix:               a.baseURL,
		DoNotValidateServers: true,
		ErrorHandlerWithOpts: a.validationError,
	})
}

func (a *API) validationError(ctx context.Context, err error, w http.ResponseWriter, r *http.Request, opts nethttpmiddleware.ErrorHandlerOpts) {
	switch {
	case opts.MatchedRoute == nil && errors.Is(err, routers.ErrMethodNotAllowed):
		a.reg.Write(w, r, problem.MethodNotAllowed)
	case opts.MatchedRoute == nil:
		a.reg.Write(w, r, problem.NotFound)
	case opts.StatusCode == http.StatusBadRequest:
		a.reg.Write(w, r, problem.ValidationFailed, fieldErrors(err)...)
	case opts.StatusCode == http.StatusUnauthorized:
		a.reg.Write(w, r, problem.InvalidToken)
	default:
		a.internal(ctx, w, r, err)
	}
}

// internal logs the failure by type only (error text may carry input) and
// answers internal_error.
func (a *API) internal(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	a.log.LogAttrs(ctx, slog.LevelError, "api error",
		slog.String("error_type", errorType(err)), slog.String("route", r.Pattern))
	a.reg.Write(w, r, problem.InternalError)
}

// ErrorHandler is for StdHTTPServerOptions.ErrorHandlerFunc: the generated
// wrapper calls it when binding a path, query or header parameter fails.
func (a *API) ErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	a.reg.Write(w, r, problem.ValidationFailed, fieldErrors(err)...)
}

// StrictRequestError is for StrictHTTPServerOptions.RequestErrorHandlerFunc:
// the request body could not be decoded.
func (a *API) StrictRequestError(w http.ResponseWriter, r *http.Request, err error) {
	a.reg.Write(w, r, problem.ValidationFailed, fieldErrors(err)...)
}

// StrictResponseError is for StrictHTTPServerOptions.ResponseErrorHandlerFunc:
// a handler returned an error, or its response could not be written.
func (a *API) StrictResponseError(w http.ResponseWriter, r *http.Request, err error) {
	a.internal(r.Context(), w, r, err)
}

// Wrap puts the 404/405 problem+json wrapper around mux. ServeMux answers an
// unknown path or method with text/plain; this answers with the registered
// problem and the Allow header. It serves matched requests through mux itself,
// so ServeMux sets r.Pattern on the request the caller handed in.
func (a *API) Wrap(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); pattern != "" {
			mux.ServeHTTP(w, r)
			return
		}
		probe := &statusProbe{header: http.Header{}}
		mux.ServeHTTP(probe, r.Clone(r.Context()))
		switch probe.code {
		case http.StatusNotFound:
			a.reg.Write(w, r, problem.NotFound)
		case http.StatusMethodNotAllowed:
			w.Header().Set("Allow", probe.header.Get("Allow"))
			a.reg.Write(w, r, problem.MethodNotAllowed)
		default: // a redirect to the cleaned path: ServeMux's own answer is right
			mux.ServeHTTP(w, r)
		}
	})
}

// statusProbe records what ServeMux would answer, without sending it.
type statusProbe struct {
	header http.Header
	code   int
}

func (p *statusProbe) Header() http.Header         { return p.header }
func (p *statusProbe) WriteHeader(code int)        { p.code = code }
func (p *statusProbe) Write(b []byte) (int, error) { return len(b), nil }

// stringFormats are the formats the contract uses, registered per validation.
func stringFormats() map[string]openapi3.StringFormatValidator {
	return map[string]openapi3.StringFormatValidator{
		"uuid": openapi3.NewRegexpFormatValidator(openapi3.FormatOfStringForUUIDOfRFC9562),
	}
}

func errorType(err error) string {
	var b strings.Builder
	for ; err != nil; err = errors.Unwrap(err) {
		if b.Len() > 0 {
			b.WriteString(">")
		}
		fmt.Fprintf(&b, "%T", err)
	}
	return b.String()
}

// LoadSpec parses and validates an OpenAPI 3.x document.
func LoadSpec(data []byte) (*openapi3.T, error) {
	spec, err := openapi3.NewLoader().LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("load openapi spec: %w", err)
	}
	if err := spec.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("validate openapi spec: %w", err)
	}
	return spec, nil
}
