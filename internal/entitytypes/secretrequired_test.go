package entitytypes

import (
	"errors"
	"slices"
	"testing"
)

func TestCheckSchemaSecretRequired(t *testing.T) {
	const props = `"properties": {"pw": {"type": "string", "x-nava-secret": true}, "tok": {"type": "string", "writeOnly": true}, "name": {"type": "string"}}`
	tests := []struct {
		name   string
		schema string
		want   []string // nil: accepted
	}{
		{"top-level required", `{` + props + `, "required": ["name", "pw"]}`, []string{"/required/1"}},
		{"writeOnly required", `{` + props + `, "required": ["tok"]}`, []string{"/required/0"}},
		{"allOf", `{` + props + `, "allOf": [{"required": ["pw"]}]}`, []string{"/allOf/0/required/0"}},
		{"anyOf", `{` + props + `, "anyOf": [{"required": ["name"]}, {"required": ["pw"]}]}`, []string{"/anyOf/1/required/0"}},
		{"oneOf", `{` + props + `, "oneOf": [{"required": ["pw"]}]}`, []string{"/oneOf/0/required/0"}},
		{"if", `{` + props + `, "if": {"required": ["pw"]}}`, []string{"/if/required/0"}},
		{"then", `{` + props + `, "if": {}, "then": {"required": ["pw"]}}`, []string{"/then/required/0"}},
		{"else", `{` + props + `, "if": {}, "else": {"required": ["pw"]}}`, []string{"/else/required/0"}},
		{"not", `{` + props + `, "not": {"required": ["pw"]}}`, []string{"/not/required/0"}},
		{"dependentRequired", `{` + props + `, "dependentRequired": {"name": ["pw"]}}`, []string{"/dependentRequired/name/0"}},
		{"dependentSchemas", `{` + props + `, "dependentSchemas": {"name": {"required": ["pw"]}}}`, []string{"/dependentSchemas/name/required/0"}},
		{"nested", `{` + props + `, "allOf": [{"anyOf": [{"then": {"required": ["pw"]}}]}]}`, []string{"/allOf/0/anyOf/0/then/required/0"}},
		{"escaped key", `{` + props + `, "dependentRequired": {"a/b": ["pw"]}}`, []string{"/dependentRequired/a~1b/0"}},
		{"several, sorted", `{` + props + `, "required": ["tok", "pw"]}`, []string{"/required/0", "/required/1"}},
		{"marker without required", `{` + props + `, "required": ["name"]}`, nil},
		{"no marker", `{"properties": {"name": {"type": "string"}}, "required": ["name"]}`, nil},
		{"required names a trigger only", `{` + props + `, "dependentRequired": {"pw": ["name"]}}`, nil},
		{"child location is not walked", `{` + props + `, "properties": {"o": {"required": ["pw"]}}}`, nil},
		{"not an object", `[1]`, nil},
		{"not JSON", `{`, nil},
		{"required of another shape", `{` + props + `, "required": "pw"}`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckSchemaSecretRequired([]byte(tt.schema))
			if tt.want == nil {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				return
			}
			var sr *SecretRequiredError
			if !errors.As(err, &sr) {
				t.Fatalf("err = %v, want *SecretRequiredError", err)
			}
			if !slices.Equal(sr.Paths, tt.want) {
				t.Errorf("paths = %v, want %v", sr.Paths, tt.want)
			}
		})
	}
}
