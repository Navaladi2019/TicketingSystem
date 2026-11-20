package config_test

import (
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

func TestLoadACLMaxDocumentBytes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		env  map[string]string
		want int
		ok   bool
	}{
		{"unset", nil, 32768, true},
		{"empty", map[string]string{"NAVA_ACL_MAX_DOCUMENT_BYTES": ""}, 32768, true},
		{"lowest", map[string]string{"NAVA_ACL_MAX_DOCUMENT_BYTES": "16384"}, 16384, true},
		{"highest", map[string]string{"NAVA_ACL_MAX_DOCUMENT_BYTES": "131072"}, 131072, true},
		{"between", map[string]string{"NAVA_ACL_MAX_DOCUMENT_BYTES": "65536"}, 65536, true},
		{"one below the range", map[string]string{"NAVA_ACL_MAX_DOCUMENT_BYTES": "16383"}, 0, false},
		{"one above the range", map[string]string{"NAVA_ACL_MAX_DOCUMENT_BYTES": "131073"}, 0, false},
		{"not an integer", map[string]string{"NAVA_ACL_MAX_DOCUMENT_BYTES": "abc"}, 0, false},
		{"a decimal", map[string]string{"NAVA_ACL_MAX_DOCUMENT_BYTES": "32768.0"}, 0, false},
		{"a plus sign", map[string]string{"NAVA_ACL_MAX_DOCUMENT_BYTES": "+32768"}, 0, false},
		{"negative", map[string]string{"NAVA_ACL_MAX_DOCUMENT_BYTES": "-1"}, 0, false},
		{"zero", map[string]string{"NAVA_ACL_MAX_DOCUMENT_BYTES": "0"}, 0, false},
		{"padded", map[string]string{"NAVA_ACL_MAX_DOCUMENT_BYTES": " 32768"}, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := config.Load(env(tc.env))
			if !tc.ok {
				if err == nil {
					t.Fatalf("Load = %+v, want an error", cfg.ACL)
				}
				if !strings.Contains(err.Error(), "NAVA_ACL_MAX_DOCUMENT_BYTES") {
					t.Errorf("error %q does not name the variable", err)
				}
				if v := tc.env["NAVA_ACL_MAX_DOCUMENT_BYTES"]; len(strings.TrimSpace(v)) > 3 && strings.Contains(err.Error(), v) {
					t.Errorf("error %q echoes the value", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if cfg.ACL.MaxDocumentBytes != tc.want {
				t.Errorf("MaxDocumentBytes = %d, want %d", cfg.ACL.MaxDocumentBytes, tc.want)
			}
		})
	}
}
