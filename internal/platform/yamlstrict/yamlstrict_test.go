package yamlstrict

import (
	"strings"
	"testing"
)

type doc struct {
	Name  string            `yaml:"name"`
	Items map[string]string `yaml:"items"`
}

func TestDecode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      string
		wantErr string
	}{
		{"valid", "name: a\nitems: {x: y}\n", ""},
		{"valid with explicit document start", "---\nname: a\n", ""},
		{"typo'd key", "name: a\nitmes: {x: y}\n", "field itmes not found"},
		{"duplicate key", "name: a\nname: b\n", `"name" already defined`},
		{"duplicate nested key", "items:\n  x: y\n  x: z\n", `"x" already defined`},
		{"second document", "name: a\n---\nname: b\n", "more than one document"},
		{"empty second document", "name: a\n---\n", "more than one document"},
		{"broken second document", "name: a\n---\n: : :\n", "after the first document"},
		{"empty", "", "empty document"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var d doc
			err := DecodeBytes([]byte(tt.in), &d)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if d.Name != "a" {
					t.Fatalf("name = %q, want a", d.Name)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want one containing %q", err, tt.wantErr)
			}
		})
	}
}
