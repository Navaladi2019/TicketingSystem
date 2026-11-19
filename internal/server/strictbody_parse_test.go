package server

import (
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// TestParsersRefuseNonStrictJSON: each request-body parser decodes with
// encoding/json/v2, so a duplicate member name or invalid UTF-8 anywhere in
// the body is refused with its pointer and rule even when the parser is
// reached without the boundary middleware (TICK-964).
func TestParsersRefuseNonStrictJSON(t *testing.T) {
	t.Parallel()
	parsers := map[string]struct {
		code  problem.Code
		parse func([]byte) *problemError
	}{
		"create entity type": {problem.InvalidDefinition, func(b []byte) *problemError { _, pe := parseCreate(b); return pe }},
		"publish version":    {problem.InvalidDefinition, func(b []byte) *problemError { _, pe := parsePublish(b); return pe }},
		"create record":      {problem.RecordInvalid, func(b []byte) *problemError { _, pe := parseRecordBody(b); return pe }},
		"patch record":       {problem.RecordInvalid, func(b []byte) *problemError { _, pe := parsePatchBody(b, "state"); return pe }},
	}
	bodies := []struct {
		name, body, pointer, rule string
	}{
		{"top-level duplicate", `{"data":{},"data":{}}`, "/data", "duplicate"},
		{"duplicate at nested depth", `{"data":{"a":[{"b":1,"b":2}]}}`, "/data/a/0/b", "duplicate"},
		{"lone surrogate", `{"data":{"a":"\ud800"}}`, "/data/a", "invalid_utf8"},
		{"overlong UTF-8", "{\"data\":{\"a\":\"\xe0\x80\xaf\"}}", "/data/a", "invalid_utf8"},
	}
	for name, p := range parsers {
		for _, b := range bodies {
			t.Run(name+"/"+b.name, func(t *testing.T) {
				t.Parallel()
				pe := p.parse([]byte(b.body))
				if pe == nil || pe.code != p.code {
					t.Fatalf("problem = %+v, want %s", pe, p.code)
				}
				if len(pe.fields) != 1 || pe.fields[0].Pointer != b.pointer || pe.fields[0].Rule != b.rule {
					t.Errorf("fields = %+v, want %s (%s)", pe.fields, b.pointer, b.rule)
				}
			})
		}
	}
}
