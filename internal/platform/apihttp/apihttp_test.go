package apihttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/apitest"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/apihttp"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/httpobs"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// secret is sent in every bad request below; no response body may contain it.
const secret = "SECRET-PII-VALUE-xyz"

const goodID = "0190f0f0-0000-7000-8000-000000000000"

type fixture struct{}

func (fixture) GetItem(_ context.Context, req apitest.GetItemRequestObject) (apitest.GetItemResponseObject, error) {
	return apitest.GetItem200JSONResponse{Id: req.Id, Name: "x"}, nil
}

func (fixture) CreateItem(context.Context, apitest.CreateItemRequestObject) (apitest.CreateItemResponseObject, error) {
	return apitest.CreateItem201JSONResponse{Name: "x"}, nil
}

type env struct {
	handler http.Handler
	spans   *tracetest.InMemoryExporter
	reg     *problem.Registry
}

func newEnv(t *testing.T) *env {
	t.Helper()
	specData, err := os.ReadFile("../../apitest/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	spec, err := apihttp.LoadSpec(specData)
	if err != nil {
		t.Fatal(err)
	}
	errData, err := api.Files.ReadFile("errors.yaml")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := problem.LoadRegistry(errData)
	if err != nil {
		t.Fatal(err)
	}
	e := &env{spans: tracetest.NewInMemoryExporter(), reg: reg}
	tel, err := telemetry.New(t.Context(), config.Telemetry{}, telemetry.WithSpanExporter(e.spans))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })

	log := slog.New(slog.DiscardHandler)
	a := apihttp.New(spec, reg, log, "")
	mux := http.NewServeMux()
	apitest.HandlerWithOptions(
		apitest.NewStrictHandlerWithOptions(fixture{}, nil, apitest.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  a.StrictRequestError,
			ResponseErrorHandlerFunc: a.StrictResponseError,
		}),
		apitest.StdHTTPServerOptions{
			BaseRouter:       mux,
			Middlewares:      []apitest.MiddlewareFunc{a.Middleware()},
			ErrorHandlerFunc: a.ErrorHandler,
		})
	e.handler = httpobs.Middleware(log, tel, a.Wrap(mux))
	return e
}

func (e *env) do(t *testing.T, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, target, bytes.NewReader([]byte(body)))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

// TestErrorsAreRegisteredProblemsWithoutInput is the contract test: each bad
// request answers problem+json with a registered code and echoes no input.
func TestErrorsAreRegisteredProblemsWithoutInput(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	tests := []struct {
		name, method, target, body string
		status                     int
		code                       problem.Code
		pointer, rule              string
	}{
		{"bad uuid", "GET", "/api/v1/items/" + secret, "", 400, problem.ValidationFailed, "/parameters/id", "format"},
		{"bad uuid, near miss", "GET", "/api/v1/items/0190f0f0-0000-7000-8000-00000000000g", "", 400, problem.ValidationFailed, "/parameters/id", "format"},
		{"query out of range", "GET", "/api/v1/items/" + goodID + "?limit=999", "", 400, problem.ValidationFailed, "/parameters/limit", "maximum"},
		{"query not a number", "GET", "/api/v1/items/" + goodID + "?limit=" + secret, "", 400, problem.ValidationFailed, "/parameters/limit", "format"},
		{"pattern violation", "POST", "/api/v1/items", `{"name":"` + secret + `"}`, 400, problem.ValidationFailed, "/body/name", "pattern"},
		{"missing field", "POST", "/api/v1/items", `{}`, 400, problem.ValidationFailed, "", ""},
		{"unknown property", "POST", "/api/v1/items", `{"name":"ok","` + secret + `":1}`, 400, problem.ValidationFailed, "", ""},
		{"malformed json", "POST", "/api/v1/items", `{"name":"` + secret, 400, problem.ValidationFailed, "", ""},
		{"405", "DELETE", "/api/v1/items/" + goodID, "", 405, problem.MethodNotAllowed, "", ""},
		{"404", "GET", "/api/v1/" + secret, "", 404, problem.NotFound, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := e.do(t, tt.method, tt.target, tt.body)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tt.status, rec.Body)
			}
			if ct := rec.Header().Get("Content-Type"); ct != problem.ContentType {
				t.Fatalf("Content-Type = %q, want %q", ct, problem.ContentType)
			}
			if strings.Contains(rec.Body.String(), secret) {
				t.Fatalf("body echoes the input: %s", rec.Body)
			}
			var p problem.Body
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatalf("body is not a problem: %v", err)
			}
			entry, ok := e.reg.Lookup(p.Code)
			if !ok || p.Code != tt.code {
				t.Fatalf("code = %q, want registered %q", p.Code, tt.code)
			}
			if p.Status != tt.status || entry.Status != tt.status || p.Title != entry.Title || p.Detail != entry.Detail {
				t.Errorf("body %+v does not match registry entry %+v", p, entry)
			}
			if !strings.HasPrefix(p.Instance, "urn:nava:request:") || p.Instance == "urn:nava:request:" {
				t.Errorf("instance = %q, want the request id", p.Instance)
			}
			if tt.pointer != "" && (len(p.Errors) != 1 || p.Errors[0].Pointer != tt.pointer || p.Errors[0].Rule != tt.rule) {
				t.Errorf("errors = %+v, want %s (%s)", p.Errors, tt.pointer, tt.rule)
			}
			if tt.status == http.StatusMethodNotAllowed && rec.Header().Get("Allow") == "" {
				t.Error("405 has no Allow header")
			}
		})
	}
}

func TestValidRequestsPass(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	if rec := e.do(t, "GET", "/api/v1/items/"+goodID, ""); rec.Code != http.StatusOK {
		t.Errorf("GET item = %d: %s", rec.Code, rec.Body)
	}
	if rec := e.do(t, "POST", "/api/v1/items", `{"name":"abc"}`); rec.Code != http.StatusCreated {
		t.Errorf("POST item = %d: %s", rec.Code, rec.Body)
	}
}

// TestValidatorErrorKeepsRoute: the validator runs after ServeMux set
// r.Pattern, so a 400 still records http.route; an unknown path records none.
func TestValidatorErrorKeepsRoute(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.do(t, "GET", "/api/v1/items/not-a-uuid", "")
	e.do(t, "GET", "/api/v1/x", "")

	spans := e.spans.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("got %d spans, want 2", len(spans))
	}
	route := func(i int) (string, bool) {
		for _, kv := range spans[i].Attributes {
			if kv.Key == attribute.Key("http.route") {
				return kv.Value.AsString(), true
			}
		}
		return "", false
	}
	if got, ok := route(0); !ok || got != "/api/v1/items/{id}" {
		t.Errorf("400 http.route = %q (set %v), want /api/v1/items/{id}", got, ok)
	}
	if got, ok := route(1); ok {
		t.Errorf("unknown path http.route = %q, want none", got)
	}
}

// TestUUIDFormatIsNotGlobal: the format is registered per validation, never
// through the package-global DefineStringFormatValidator (gochecknoglobals).
func TestUUIDFormatIsNotGlobal(t *testing.T) {
	t.Parallel()
	newEnv(t)
	if _, ok := openapi3.SchemaStringFormats["uuid"]; ok {
		t.Error("uuid format registered globally; use openapi3.WithStringFormatValidators")
	}
}
