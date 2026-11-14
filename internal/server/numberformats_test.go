package server

import (
	"encoding/json"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

func TestParseNumberFormat(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, body string
		create     bool
		code       problem.Code // "" = accepted
		pointer    string
		rule       string
	}{
		{"create ok", `{"key":"monthly","document":{"template":"{prefix}-{yyyy}{MM}-{seq:4}","overflow":"grow"},"tenant_id":"forged"}`, true, "", "", ""},
		{"publish ok", `{"document":{"template":"{prefix}{seq:7}","overflow":"grow"}}`, false, "", "", ""},
		{"not an object", `[]`, true, problem.NumberFormatInvalid, "", "type"},
		{"null", `null`, false, problem.NumberFormatInvalid, "", "type"},
		{"no key", `{"document":{}}`, true, problem.NumberFormatInvalid, "/key", "required"},
		{"key type", `{"key":1,"document":{}}`, true, problem.NumberFormatInvalid, "/key", "type"},
		{"key pattern", `{"key":"Bad Key","document":{}}`, true, problem.NumberFormatInvalid, "/key", "pattern"},
		{"no document", `{"key":"monthly"}`, true, problem.NumberFormatInvalid, "/document", "required"},
		{"schema refusal", `{"document":{"template":"{prefix}{seq:4}"}}`, false, problem.NumberFormatInvalid, "/document/overflow", "invalid"},
		{"prefix required", `{"document":{"template":"X-{prefix}{seq:4}","overflow":"grow"}}`, false, problem.NumberFormatPrefixRequired, "/document/template", "position:0"},
		{"no seq", `{"document":{"template":"{prefix}-{yyyy}","overflow":"grow"}}`, false, problem.NumberFormatInvalid, "/document/template", "position:15"},
		{"bad zone", `{"document":{"template":"{prefix}{seq:4}","overflow":"grow","time_zone":"Mars/Base"}}`, false, problem.NumberFormatInvalid, "/document/time_zone", "invalid"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in, perr := parseNumberFormat(json.RawMessage(c.body), c.create)
			if c.code == "" {
				if perr != nil {
					t.Fatalf("refused: %v", perr)
				}
				if in.compiled.CompiledSHA256 == "" || len(in.document) == 0 {
					t.Fatalf("accepted without a compiled form: %+v", in)
				}
				return
			}
			if perr == nil {
				t.Fatalf("accepted, want %s", c.code)
			}
			if perr.code != c.code || len(perr.fields) != 1 || perr.fields[0].Pointer != c.pointer || perr.fields[0].Rule != c.rule {
				t.Errorf("got %s %+v, want %s %s %s", perr.code, perr.fields, c.code, c.pointer, c.rule)
			}
		})
	}
}
