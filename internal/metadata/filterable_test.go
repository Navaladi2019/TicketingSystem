package metadata_test

import "testing"

// HasFilterable (P7-02): true only when a top-level property carries
// "x-nava-filterable": true. A false or non-boolean value, or a flag nested
// below a top-level property, does not count.
func TestHasFilterable(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(s map[string]any)
		want   bool
	}{
		{"no flag", func(map[string]any) {}, false},
		{"top-level flag", func(s map[string]any) { props(s)["site"] = map[string]any{"type": "string", "x-nava-filterable": true} }, true},
		{"flag false", func(s map[string]any) {
			props(s)["site"] = map[string]any{"type": "string", "x-nava-filterable": false}
		}, false},
		{"flag not a boolean", func(s map[string]any) {
			props(s)["site"] = map[string]any{"type": "string", "x-nava-filterable": "yes"}
		}, false},
		{"flag nested", func(s map[string]any) {
			props(s)["c"] = map[string]any{"type": "object", "properties": map[string]any{
				"site": map[string]any{"type": "string", "x-nava-filterable": true}}}
		}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := baseSchema()
			tc.mutate(s)
			if got := mustCompile(t, baseDefinition(t, s)).HasFilterable(); got != tc.want {
				t.Errorf("HasFilterable = %v, want %v", got, tc.want)
			}
		})
	}
}
