package apihttp_test

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/apihttp"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// stringErr is an error whose kind is not a struct, so paramName must skip it.
type stringErr string

func (e stringErr) Error() string { return string(e) }

func newAPI(t *testing.T) (*apihttp.API, *env) {
	t.Helper()
	e := newEnv(t)
	return apihttp.New(nil, e.reg, slog.New(slog.DiscardHandler), ""), e
}

func problemBody(t *testing.T, rec *httptest.ResponseRecorder, status int, code problem.Code) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d: %s", rec.Code, status, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != problem.ContentType {
		t.Fatalf("content type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), `"code":"`+string(code)+`"`) {
		t.Fatalf("body does not carry code %s: %s", code, rec.Body)
	}
}

func TestValidationErrorMapsEveryOutcome(t *testing.T) {
	t.Parallel()
	a, _ := newAPI(t)
	matched := &nethttpmiddleware.ErrorHandlerOptsMatchedRoute{}
	tests := []struct {
		name   string
		err    error
		opts   nethttpmiddleware.ErrorHandlerOpts
		status int
		code   problem.Code
	}{
		{"unknown method", routersMethodNotAllowed(), nethttpmiddleware.ErrorHandlerOpts{StatusCode: 405}, 405, problem.MethodNotAllowed},
		{"unknown route", errors.New("no route"), nethttpmiddleware.ErrorHandlerOpts{StatusCode: 404}, 404, problem.NotFound},
		{"bad request", &openapi3filter.RequestError{}, nethttpmiddleware.ErrorHandlerOpts{StatusCode: 400, MatchedRoute: matched}, 400, problem.ValidationFailed},
		{"unauthorized", errors.New("auth"), nethttpmiddleware.ErrorHandlerOpts{StatusCode: 401, MatchedRoute: matched}, 401, problem.InvalidToken},
		{"anything else is internal", errors.New("boom SECRET"), nethttpmiddleware.ErrorHandlerOpts{StatusCode: 500, MatchedRoute: matched}, 500, problem.InternalError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/x", nil)
			a.ValidationError(tt.err, rec, r, tt.opts)
			problemBody(t, rec, tt.status, tt.code)
			if strings.Contains(rec.Body.String(), "SECRET") {
				t.Fatalf("body echoes the error text: %s", rec.Body)
			}
		})
	}
}

func TestStrictHandlersAnswerProblems(t *testing.T) {
	t.Parallel()
	a, _ := newAPI(t)
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/x", nil)

	rec := httptest.NewRecorder()
	a.StrictRequestError(rec, r, errors.New("decode SECRET"))
	problemBody(t, rec, 400, problem.ValidationFailed)

	rec = httptest.NewRecorder()
	a.StrictResponseError(rec, r, errors.New("write SECRET"))
	problemBody(t, rec, 500, problem.InternalError)
	if strings.Contains(rec.Body.String(), "SECRET") {
		t.Fatalf("body echoes the error text: %s", rec.Body)
	}
}

func TestWrapLeavesRedirectsToTheMux(t *testing.T) {
	t.Parallel()
	a, _ := newAPI(t)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /a/b", func(http.ResponseWriter, *http.Request) {})
	rec := httptest.NewRecorder()
	// A path with a dot segment is redirected to the cleaned path by ServeMux.
	a.Wrap(mux).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/a/./b", nil))
	if rec.Code != http.StatusTemporaryRedirect || rec.Header().Get("Location") != "/a/b" {
		t.Fatalf("got %d Location=%q, want the mux's own redirect to /a/b", rec.Code, rec.Header().Get("Location"))
	}
}

func TestLoadSpecRejectsBadDocuments(t *testing.T) {
	t.Parallel()
	if _, err := apihttp.LoadSpec([]byte("{not yaml or json: [")); err == nil || !strings.Contains(err.Error(), "load openapi spec") {
		t.Fatalf("unparseable document: err = %v", err)
	}
	// Parses, but is not a valid OpenAPI document (no info, no paths).
	if _, err := apihttp.LoadSpec([]byte(`{"openapi":"3.0.3"}`)); err == nil || !strings.Contains(err.Error(), "validate openapi spec") {
		t.Fatalf("invalid document: err = %v", err)
	}
}

func TestFieldErrorsReadOnlyTheContract(t *testing.T) {
	t.Parallel()
	// An error chain with a non-struct link and no ParamName yields nothing.
	if got := apihttp.FieldErrors(stringErr("SECRET")); len(got) != 0 {
		t.Fatalf("got %v, want none", got)
	}
	// A RequestError with neither a schema nor a parse cause names only the base.
	got := apihttp.FieldErrors(&openapi3filter.RequestError{})
	if len(got) != 1 || got[0].Pointer != "/body" || got[0].Rule != "" {
		t.Fatalf("got %+v, want one /body error without a rule", got)
	}
	// A parameter on the request error names the parameter.
	got = apihttp.FieldErrors(&openapi3filter.RequestError{Parameter: &openapi3.Parameter{Name: "limit"}})
	if len(got) != 1 || got[0].Pointer != "/parameters/limit" {
		t.Fatalf("got %+v", got)
	}
}

func routersMethodNotAllowed() error { return routers.ErrMethodNotAllowed }
