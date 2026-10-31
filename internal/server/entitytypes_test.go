package server

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/apihttp"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

const utSchema = `{ "type" : "object", "additionalProperties": false, "properties": {"state": {"type":"string","enum":["new","closed"],"default":"new"}} }`

func utBody(mutate func(m map[string]any)) json.RawMessage {
	m := map[string]any{
		"key": "incident", "name": "Incident", "number_prefix": "INC",
		"definition": map[string]any{
			"schema":        json.RawMessage(utSchema),
			"fixed_columns": map[string]string{"state": "state"},
			"states": map[string]any{
				"initial": "new", "states": []string{"new", "closed"},
				"transitions": []map[string]string{{"from": "new", "to": "closed"}},
			},
		},
	}
	if mutate != nil {
		mutate(m)
	}
	b, _ := json.Marshal(m)
	return b
}

func TestParseCreateAcceptsAValidBodyAndKeepsTheSchemaBytes(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"key":"incident","name":"Incident","number_prefix":"INC","definition":{"schema": ` + utSchema + `,
		"fixed_columns":{"state":"state"},"states":{"initial":"new","states":["new","closed"],"transitions":[{"from":"new","to":"closed"}]}}}`)
	cr, perr := parseCreate(raw)
	if perr != nil {
		t.Fatalf("parseCreate: %v %v", perr, perr.fields)
	}
	if string(cr.def.Schema) != utSchema {
		t.Errorf("schema = %q, want the request bytes %q", cr.def.Schema, utSchema)
	}
	if cr.key != "incident" || cr.name != "Incident" || cr.prefix != "INC" {
		t.Errorf("fields = %+v", cr)
	}
}

func TestParseCreateRejectsWithPointers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  json.RawMessage
		want []string
	}{
		{"not an object", json.RawMessage(`[]`), []string{""}},
		{"null", json.RawMessage(`null`), []string{""}},
		{"empty object", json.RawMessage(`{}`), []string{"/key", "/name", "/number_prefix", "/definition"}},
		{"bad key", utBody(func(m map[string]any) { m["key"] = "Bad-Key" }), []string{"/key"}},
		{"key too short", utBody(func(m map[string]any) { m["key"] = "a" }), []string{"/key"}},
		{"key too long", utBody(func(m map[string]any) { m["key"] = "a" + strings.Repeat("b", 41) }), []string{"/key"}},
		{"key wrong type", utBody(func(m map[string]any) { m["key"] = 5 }), []string{"/key"}},
		{"empty name", utBody(func(m map[string]any) { m["name"] = "" }), []string{"/name"}},
		{"name over 120", utBody(func(m map[string]any) { m["name"] = strings.Repeat("n", 121) }), []string{"/name"}},
		{"lower-case prefix", utBody(func(m map[string]any) { m["number_prefix"] = "inc" }), []string{"/number_prefix"}},
		{"prefix of one letter", utBody(func(m map[string]any) { m["number_prefix"] = "I" }), []string{"/number_prefix"}},
		{"prefix of seven", utBody(func(m map[string]any) { m["number_prefix"] = "ABCDEFG" }), []string{"/number_prefix"}},
		{"key and prefix", utBody(func(m map[string]any) { m["key"], m["number_prefix"] = "X", "x" }), []string{"/key", "/number_prefix"}},
		{"definition wrong type", utBody(func(m map[string]any) { m["definition"] = "x" }), []string{"/definition"}},
		{"schema not an object schema", utBody(func(m map[string]any) {
			m["definition"] = map[string]any{"schema": json.RawMessage(`{"type":"array"}`), "fixed_columns": map[string]string{}, "states": map[string]any{}}
		}), nil}, // pointers come from Compile; checked below
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, perr := parseCreate(tc.raw)
			if perr == nil || perr.code != problem.InvalidDefinition {
				t.Fatalf("parseCreate = %v, want invalid_definition", perr)
			}
			var got []string
			for _, f := range perr.fields {
				got = append(got, f.Pointer)
			}
			if tc.want == nil {
				if len(got) == 0 {
					t.Fatal("no pointers")
				}
				for _, g := range got {
					if !strings.HasPrefix(g, "/definition/") {
						t.Errorf("pointer %q is not under /definition/", g)
					}
				}
				return
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("pointers = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseCreateErrorsNeverEchoInput(t *testing.T) {
	t.Parallel()
	_, perr := parseCreate(utBody(func(m map[string]any) { m["key"] = "SECRET-VALUE" }))
	if perr == nil {
		t.Fatal("accepted")
	}
	for _, f := range perr.fields {
		if strings.Contains(f.Pointer+f.Rule, "SECRET") {
			t.Errorf("field error %+v echoes the input", f)
		}
	}
}

func TestEntityTypeResponseWritesTheStoredSchemaBytes(t *testing.T) {
	t.Parallel()
	const schema = "{\n  \"type\": \"object\",\n  \"x\": \"<&>\"\n}"
	sum := sha256.Sum256([]byte(schema))
	v := &entitytypes.Version{
		Key: "incident", Name: "Incident", NumberPrefix: "INC", Version: 1, Current: true,
		Schema: json.RawMessage(schema), SchemaSHA256: sum, Compat: "initial",
		States:      metadata.StateModel{Initial: "new", States: []string{"new"}},
		PublishedAt: time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC),
	}
	rec := httptest.NewRecorder()
	if err := (&entityTypeResponse{status: http.StatusCreated, v: v}).VisitCreateEntityTypeResponse(rec); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusCreated || rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("status %d, Content-Type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), `"schema":`+schema+`,"schema_sha256":"`) {
		t.Errorf("schema bytes not verbatim: %s", rec.Body)
	}
	var out api.EntityTypeVersion
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("not valid JSON: %v: %s", err, rec.Body)
	}
	if out.Key != "incident" || out.Version != 1 || !out.Current || out.Compat != "initial" || out.FixedColumns == nil {
		t.Errorf("decoded = %+v", out)
	}
}

func TestNotFoundBodyIsIdenticalAndHasNoInstance(t *testing.T) {
	t.Parallel()
	reg := loadTestRegistry(t)
	h := &handlers{reg: reg}
	a := h.notFound(context.Background())
	b := h.notFound(context.Background())
	if string(a.body) != string(b.body) {
		t.Errorf("bodies differ: %s / %s", a.body, b.body)
	}
	if strings.Contains(string(a.body), "instance") || !strings.Contains(string(a.body), `"code":"not_found"`) {
		t.Errorf("body = %s", a.body)
	}
	rec := httptest.NewRecorder()
	if err := a.VisitGetEntityTypeResponse(rec); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusNotFound || rec.Header().Get("Content-Type") != problem.ContentType {
		t.Errorf("status %d, Content-Type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
}

func loadTestRegistry(t *testing.T) *problem.Registry {
	t.Helper()
	data, err := api.Files.ReadFile("errors.yaml")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := problem.LoadRegistry(data)
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

// The field errors of a 422 reach the client: responseError writes them for
// any code, not only validation_failed.
func TestResponseErrorWritesFieldsForEveryCode(t *testing.T) {
	t.Parallel()
	reg := loadTestRegistry(t)
	h := &handlers{reg: reg, log: slog.New(slog.DiscardHandler)}
	respond := h.responseError(apihttp.New(nil, reg, h.log, ""))

	for _, code := range []problem.Code{problem.InvalidDefinition, problem.InvalidIdPConfig, problem.ValidationFailed} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/entity-types", nil)
		respond(rec, req, &problemError{code, []problem.FieldError{{Pointer: "/key", Rule: "pattern"}}, errors.New("x")})
		var got struct {
			Code   problem.Code         `json:"code"`
			Errors []problem.FieldError `json:"errors"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Code != code || len(got.Errors) != 1 || got.Errors[0].Pointer != "/key" {
			t.Errorf("%s: body = %s, want the code and errors[0].pointer /key", code, rec.Body)
		}
	}

	// A problem with no fields still goes through WriteError (no errors[]).
	rec := httptest.NewRecorder()
	respond(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil), &problemError{problem.Forbidden, nil, errors.New("x")})
	if rec.Code != http.StatusForbidden || strings.Contains(rec.Body.String(), `"errors"`) {
		t.Errorf("forbidden = %d %s", rec.Code, rec.Body)
	}
}

// Every registered code that carries field errors must be written by
// responseError with its errors[]. A code is either in the switch or listed
// here as one that never carries fields; a new code that is neither fails, so
// its fields cannot be dropped silently.
func TestResponseErrorKnowsEveryCode(t *testing.T) {
	t.Parallel()
	noFields := map[problem.Code]bool{
		problem.TenantUnresolved: true, problem.InvalidToken: true, problem.IdempotencyKeyReuse: true,
		problem.IdempotencyInFlight: true, problem.Forbidden: true, problem.DomainClaimed: true,
		problem.OnboardingConflict: true, problem.InvalidCursor: true, problem.RateLimited: true,
		problem.NotFound: true, problem.MethodNotAllowed: true, problem.NotImplemented: true,
		problem.InternalError: true, problem.EntityTypeExists: true, problem.NumberPrefixTaken: true,
	}
	reg := loadTestRegistry(t)
	h := &handlers{reg: reg, log: slog.New(slog.DiscardHandler)}
	respond := h.responseError(apihttp.New(nil, reg, h.log, ""))
	for _, code := range problem.Codes() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)
		respond(rec, req, &problemError{code, []problem.FieldError{{Pointer: "/x", Rule: "r"}}, errors.New("x")})
		carries := strings.Contains(rec.Body.String(), `"errors"`)
		switch {
		case noFields[code] && carries:
			t.Errorf("%s is listed as carrying no fields but responseError writes them", code)
		case !noFields[code] && !carries:
			t.Errorf("%s is missing from the responseError switch (or from this test's noFields list): %s", code, rec.Body)
		}
	}
}

// A request with a nil principal id is refused before any row is touched: 500
// errNoPrincipal (a wiring bug), never a write attributed to the nil uuid.
func TestPublishWithANilPrincipalIDIsErrNoPrincipal(t *testing.T) {
	t.Parallel()
	h := &handlers{reg: loadTestRegistry(t), log: slog.New(slog.DiscardHandler)} // no pool: nothing can be written
	ctx := identity.NewContext(t.Context(), identity.Principal{TenantID: uuid.New()})
	body := json.RawMessage(`{}`)
	resp, err := h.PublishEntityTypeVersion(ctx, api.PublishEntityTypeVersionRequestObject{Key: "incident", Body: &body})
	if !errors.Is(err, errNoPrincipal) || resp != nil {
		t.Fatalf("got (%v, %v), want (nil, errNoPrincipal)", resp, err)
	}
}

// An invalid name and an invalid schema are reported in one response.
func TestParseCreateReportsFieldAndDefinitionErrorsTogether(t *testing.T) {
	t.Parallel()
	raw := `{"key": "x1", "name": "", "number_prefix": "AB", "definition": {"schema": {"type": "array"}, "fixed_columns": {}, "states": {}}}`
	_, perr := parseCreate(json.RawMessage(raw))
	if perr == nil {
		t.Fatal("accepted")
	}
	var got []string
	for _, f := range perr.fields {
		got = append(got, f.Pointer)
	}
	if len(got) < 2 || got[0] != "/name" {
		t.Fatalf("pointers = %v, want /name first, then /definition/...", got)
	}
	for _, g := range got[1:] {
		if !strings.HasPrefix(g, "/definition/") {
			t.Errorf("pointer %q is not under /definition/", g)
		}
	}
	// A bad key is listed once even though Compile reports it too.
	_, perr = parseCreate(utBody(func(m map[string]any) { m["key"] = "Bad" }))
	n := 0
	for _, f := range perr.fields {
		if f.Pointer == "/key" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("/key listed %d times, want 1: %v", n, perr.fields)
	}
}

func TestParsePublish(t *testing.T) {
	t.Parallel()
	good := []byte(`{"definition":{"schema": ` + utSchema + `,"fixed_columns":{"state":"state"},"states":{"initial":"new"}}}`)
	def, perr := parsePublish(good)
	if perr != nil || string(def.Schema) != utSchema || def.FixedColumns["state"] != "state" {
		t.Fatalf("parsePublish = %+v, %v", def, perr)
	}
	for name, body := range map[string]string{
		"number_prefix": `{"number_prefix":"INC","definition":{}}`,
		"missing":       `{}`,
		"not object":    `[]`,
		"bad type":      `{"definition":5}`,
		"null":          `null`,
	} {
		if _, perr := parsePublish([]byte(body)); perr == nil || perr.code != problem.InvalidDefinition || len(perr.fields) == 0 {
			t.Errorf("%s: err = %v, want invalid_definition with fields", name, perr)
		}
	}
}
