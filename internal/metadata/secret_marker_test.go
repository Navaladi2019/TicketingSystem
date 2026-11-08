package metadata_test

import (
	"encoding/json"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

// ADR-004 R6 (TICK-432): x-nava-secret and writeOnly are valid only directly
// on a top-level property, and a marked property holds no value.
func TestCompileSecretMarkerPlacement(t *testing.T) {
	str := func() map[string]any { return map[string]any{"type": "string"} }
	marked := func(kw string) map[string]any { m := str(); m[kw] = true; return m }
	tests := []struct {
		name   string
		mutate func(s map[string]any)
		want   string // "" = accepted
	}{
		{"marker on a top-level property", func(s map[string]any) { props(s)["pw"] = marked("x-nava-secret") }, ""},
		{"writeOnly on a top-level property", func(s map[string]any) { props(s)["pw"] = marked("writeOnly") }, ""},
		{"marker false is allowed on a top-level property", func(s map[string]any) {
			m := str()
			m["x-nava-secret"] = false
			props(s)["pw"] = m
		}, ""},
		{"non-boolean marker", func(s map[string]any) { m := str(); m["x-nava-secret"] = "yes"; props(s)["pw"] = m }, "/schema/properties/pw/x-nava-secret"},
		{"marker in allOf", func(s map[string]any) {
			props(s)["pw"] = map[string]any{"allOf": []any{marked("x-nava-secret")}}
		}, "/schema/properties/pw/allOf/0/x-nava-secret"},
		{"marker nested in an object", func(s map[string]any) {
			props(s)["c"] = map[string]any{"type": "object", "properties": map[string]any{"pw": marked("writeOnly")}}
		}, "/schema/properties/c/properties/pw/writeOnly"},
		{"marker under items", func(s map[string]any) {
			props(s)["l"] = map[string]any{"type": "array", "items": marked("x-nava-secret")}
		}, "/schema/properties/l/items/x-nava-secret"},
		{"marker in $defs", func(s map[string]any) {
			s["$defs"] = map[string]any{"secret": marked("x-nava-secret")}
		}, "/schema/$defs/secret/x-nava-secret"},
		{"marker on the root", func(s map[string]any) { s["writeOnly"] = true }, "/schema/writeOnly"},
		{"default on a marked property", func(s map[string]any) {
			m := marked("x-nava-secret")
			m["default"] = "v"
			props(s)["pw"] = m
		}, "/schema/properties/pw/default"},
		{"enum on a marked property", func(s map[string]any) {
			m := marked("x-nava-secret")
			m["enum"] = []any{"v"}
			props(s)["pw"] = m
		}, "/schema/properties/pw/enum"},
		{"const on the root with a marked member", func(s map[string]any) {
			props(s)["pw"] = marked("x-nava-secret")
			s["const"] = map[string]any{"pw": "v"}
		}, "/schema/const"},
		{"const in allOf of a marked property", func(s map[string]any) {
			m := marked("x-nava-secret")
			m["allOf"] = []any{map[string]any{"const": "hunter2"}}
			props(s)["pw"] = m
		}, "/schema/properties/pw/allOf/0/const"},
		{"applicator on a marked property", func(s map[string]any) {
			m := marked("x-nava-secret")
			m["not"] = map[string]any{"const": "hunter2"}
			props(s)["pw"] = m
		}, "/schema/properties/pw/not"},
		{"enum in anyOf of a marked property", func(s map[string]any) {
			m := marked("x-nava-secret")
			m["anyOf"] = []any{map[string]any{"enum": []any{"hunter2"}}}
			props(s)["pw"] = m
		}, "/schema/properties/pw/anyOf/0/enum"},
		{"root allOf sets the marked member", func(s map[string]any) {
			props(s)["pw"] = marked("x-nava-secret")
			s["allOf"] = []any{map[string]any{"properties": map[string]any{"pw": map[string]any{"const": "hunter2"}}}}
		}, "/schema/allOf/0/properties/pw/const"},
		{"root if matches the marked member", func(s map[string]any) {
			props(s)["pw"] = marked("x-nava-secret")
			s["if"] = map[string]any{"properties": map[string]any{"pw": map[string]any{"const": "hunter2"}}}
			s["then"] = map[string]any{}
		}, "/schema/if/properties/pw/const"},
		{"root allOf const sets the member", func(s map[string]any) {
			props(s)["pw"] = marked("x-nava-secret")
			s["allOf"] = []any{map[string]any{"const": map[string]any{"pw": "hunter2"}}}
		}, "/schema/allOf/0/const"},
		{"enum inside a non-marked property is fine", func(s map[string]any) {
			props(s)["pw"] = marked("x-nava-secret")
			props(s)["kind"] = map[string]any{"type": "string", "enum": []any{"a", "b"}}
		}, ""},
		{"pattern on a marked property", func(s map[string]any) {
			m := marked("x-nava-secret")
			m["pattern"] = "^hunter2$"
			props(s)["pw"] = m
		}, "/schema/properties/pw/pattern"},
		{"pattern on the marked member under root allOf", func(s map[string]any) {
			props(s)["pw"] = marked("x-nava-secret")
			s["allOf"] = []any{map[string]any{"properties": map[string]any{"pw": map[string]any{"pattern": "^hunter2$"}}}}
		}, "/schema/allOf/0/properties/pw/pattern"},
		{"pattern inside an unmarked property is fine", func(s map[string]any) {
			props(s)["pw"] = marked("x-nava-secret")
			props(s)["code"] = map[string]any{"type": "string", "pattern": "^[A-Z]+$"}
		}, ""},
		{"root allOf $ref outside an unmarked property", func(s map[string]any) {
			props(s)["pw"] = marked("x-nava-secret")
			s["$defs"] = map[string]any{"x": map[string]any{"type": "object"}}
			s["allOf"] = []any{map[string]any{"$ref": "#/$defs/x"}}
		}, "/schema/allOf/0/$ref"},
		{"$ref inside an unmarked property is fine", func(s map[string]any) {
			props(s)["pw"] = marked("x-nava-secret")
			s["$defs"] = map[string]any{"x": map[string]any{"type": "string"}}
			props(s)["kind"] = map[string]any{"$ref": "#/$defs/x"}
		}, ""},
		{"$ref to a marked property", func(s map[string]any) {
			props(s)["pw"] = marked("x-nava-secret")
			props(s)["copy"] = map[string]any{"$ref": "#/properties/pw"}
		}, "/schema/properties/copy/$ref"},
		{"$ref by anchor to a marked property", func(s map[string]any) {
			m := marked("x-nava-secret")
			m["$anchor"] = "secret"
			props(s)["pw"] = m
			props(s)["copy"] = map[string]any{"$ref": "#secret"}
		}, "/schema/properties/copy/$ref"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := baseSchema()
			tt.mutate(s)
			raw, err := json.Marshal(s)
			if err != nil {
				t.Fatal(err)
			}
			d := baseDefinition(t, s)
			d.Schema = raw
			_, err = metadata.Compile(d)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("Compile: %v", err)
				}
				return
			}
			wantPath(t, err, tt.want)
		})
	}
}

// SecretProperties lists the top-level properties marked x-nava-secret or
// writeOnly, sorted, and nothing else: the record pipeline turns each into
// the secret pointer "/<name>" of a record's audit diff (TICK-452).
func TestSecretProperties(t *testing.T) {
	s := baseSchema()
	props(s)["pw"] = map[string]any{"type": "string", "x-nava-secret": true}
	props(s)["api_key"] = map[string]any{"type": "string", "writeOnly": true}
	props(s)["plain"] = map[string]any{"type": "string", "writeOnly": false}
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	d := baseDefinition(t, s)
	d.Schema = raw
	c, err := metadata.Compile(d)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if got := c.SecretProperties(); len(got) != 2 || got[0] != "api_key" || got[1] != "pw" {
		t.Errorf("SecretProperties = %q, want [api_key pw]", got)
	}

	d = baseDefinition(t, baseSchema())
	if c, err = metadata.Compile(d); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if got := c.SecretProperties(); len(got) != 0 {
		t.Errorf("SecretProperties of an unmarked schema = %q, want none", got)
	}
}
