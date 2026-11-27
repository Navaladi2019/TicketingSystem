package metadata_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

// TICK-1121: PostgreSQL's jsonb refuses a \u0000 escape (22P05). The audit
// entry of a create or publish reads the schema back as jsonb, so a NUL that
// reached the insert gave a 500. CheckStorable refuses it with the pointer.
func TestCheckStorableRefusesNUL(t *testing.T) {
	const tail = `,"properties":{"state":{"type":"string","enum":["open","closed"]}}}`
	cases := []struct {
		name, raw string
		want      []string
	}{
		{"in a description", `{"type":"object","description":"x\u0000y"` + tail, []string{"/schema/description"}},
		{"in an enum value", `{"type":"object","properties":{"state":{"type":"string","enum":["open","cl\u0000sed"]}}}`, []string{"/schema/properties/state/enum/1"}},
		{"in a property name", `{"type":"object","properties":{"a\u0000b":{"type":"string"}}}`, []string{"/schema/properties/a\x00b"}},
		{"in a name with a slash", `{"type":"object","properties":{"a/\u0000":{"type":"string"}}}`, []string{"/schema/properties/a~1\x00"}},
		{"two of them", `{"type":"object","title":"\u0000","description":"\u0000"` + tail, []string{"/schema/description", "/schema/title"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := metadata.CheckStorable([]byte(c.raw))
			var ie *metadata.ErrInvalidDefinition
			if !errors.As(err, &ie) {
				t.Fatalf("CheckStorable = %v, want *ErrInvalidDefinition", err)
			}
			if !slices.Equal(ie.Paths, c.want) {
				t.Errorf("paths = %q, want %q", ie.Paths, c.want)
			}
		})
	}
}

func TestCheckStorableAccepts(t *testing.T) {
	for name, raw := range map[string]string{
		"an escaped backslash then u0000": `{"description":"x\\u0000y"}`,
		"a surrogate pair":                `{"description":"😀"}`,
		"a literal NUL-free schema":       `{"type":"object","properties":{}}`,
		// Not JSON, or a lone surrogate: Compile reports these, not us.
		"malformed":      `{"description":`,
		"lone surrogate": `{"description":"\ud800"}`,
	} {
		t.Run(name, func(t *testing.T) {
			if err := metadata.CheckStorable([]byte(raw)); err != nil {
				t.Errorf("CheckStorable = %v, want nil", err)
			}
		})
	}
}

// The stored-version path (Compile) must keep accepting a NUL: a version
// stored before 2026-11-03 may hold one and must still load.
func TestCompileStillAcceptsNUL(t *testing.T) {
	s := baseSchema()
	s["description"] = "x\x00y"
	mustCompile(t, baseDefinition(t, s))
}
