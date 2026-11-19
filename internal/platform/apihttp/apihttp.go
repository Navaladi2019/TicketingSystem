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
// Before the validator reads a body, StrictBody refuses one with a duplicate
// member name or invalid UTF-8 (TICK-964): the validator decodes with
// encoding/json v1, which keeps the last duplicate and replaces bad bytes, so
// it would judge a different document from the one a strict reader sees.
//
// The validator is a per-route middleware (StdHTTPServerOptions.Middlewares),
// so it runs after ServeMux set r.Pattern and a 400 keeps its http.route.
package apihttp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/strictjson"
)

// API holds what the error contract needs.
type API struct {
	spec    *openapi3.T
	reg     *problem.Registry
	log     *slog.Logger
	baseURL string
	// bodyCodes maps a route pattern to the 422 code its strict-JSON
	// refusal answers with (WithBodyCodes).
	bodyCodes map[string]problem.Code
}

// New returns an API for spec. baseURL must equal the BaseURL given to the
// generated StdHTTPServerOptions: it is the validator's Options.Prefix.
func New(spec *openapi3.T, reg *problem.Registry, log *slog.Logger, baseURL string) *API {
	return &API{spec: spec, reg: reg, log: log, baseURL: baseURL}
}

// WithBodyCodes sets, per route pattern ("POST /api/v1/records/{entity}"),
// the registered 422 code StrictBody answers with: each operation keeps the
// code it already uses for an unacceptable body. It returns a.
func (a *API) WithBodyCodes(codes map[string]problem.Code) *API {
	a.bodyCodes = codes
	return a
}

// Middleware returns the request validator, for StdHTTPServerOptions.Middlewares.
func (a *API) Middleware() func(http.Handler) http.Handler {
	// The server list is host validation we do not want, and spec is shared.
	spec := *a.spec
	spec.Servers = nil
	validate := nethttpmiddleware.OapiRequestValidatorWithOptions(&spec, &nethttpmiddleware.Options{
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
	return func(next http.Handler) http.Handler {
		validated := a.StrictBody(validate(next))
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodHead {
				validated.ServeHTTP(w, r)
				return
			}
			// ServeMux routes HEAD to a GET pattern, but the contract declares
			// no HEAD operation, so the validator would answer 405 (R3). Serve
			// HEAD as the GET it stands for, with no body.
			get := r.Clone(r.Context())
			get.Method = http.MethodGet
			validated.ServeHTTP(headWriter{w}, get)
		})
	}
}

// StrictBody refuses a request body with a duplicate object member name or
// invalid UTF-8, anywhere in the document, before next reads it. The answer
// is the route's code from WithBodyCodes (a 422) with one error: the JSON
// pointer under /body and the rule, duplicate or invalid_utf8. A route with
// no code configured answers validation_failed, so a missing entry still
// refuses the body. Any other body goes on to next byte-identical; a syntax
// error stays the validator's to answer. The body is already bounded where it
// is first read (the idempotency and operator middlewares).
func (a *API) StrictBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil || r.Body == http.NoBody {
			next.ServeHTTP(w, r)
			return
		}
		body, err := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if err != nil {
			a.internal(w, r, fmt.Errorf("read request body: %w", err))
			return
		}
		if v, bad := strictjson.Check(body); bad {
			code, ok := a.bodyCodes[r.Pattern]
			if !ok {
				code = problem.ValidationFailed
			}
			a.reg.Write(w, r, code, problem.FieldError{Pointer: "/body" + v.Pointer, Rule: v.Rule})
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		next.ServeHTTP(w, r)
	})
}

// headWriter drops the body of a HEAD response and keeps status and headers.
type headWriter struct{ http.ResponseWriter }

func (headWriter) Write(b []byte) (int, error) { return len(b), nil }

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
		a.internal(w, r, err)
	}
}

// internal reports the failure (span error, log with the error text) and
// answers internal_error.
func (a *API) internal(w http.ResponseWriter, r *http.Request, err error) {
	a.reg.WriteError(w, r, a.log, problem.InternalError, err, slog.String("route", r.Pattern))
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
	a.internal(w, r, err)
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
