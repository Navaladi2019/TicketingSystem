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
