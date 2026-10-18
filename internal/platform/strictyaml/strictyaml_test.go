package strictyaml_test

import (
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/strictyaml"
)

type doc struct {
	Name string `yaml:"name"`
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr string
	}{
		{"ok", "name: a\n", ""},
		{"unknown field", "name: a\nnam: b\n", "decode yaml"},
		{"second document", "name: a\n---\nname: b\n", "more than one document"},
		{"empty", "", "decode yaml"},
		{"malformed", "name: [", "decode yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d doc
			err := strictyaml.Decode([]byte(tt.in), &d)
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}
