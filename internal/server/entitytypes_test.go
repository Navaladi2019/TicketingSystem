package server

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
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
