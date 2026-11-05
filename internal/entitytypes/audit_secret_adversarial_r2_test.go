package entitytypes_test

// Review round 2 of PR #413 (TICK-432), Security & Multi-Tenancy Reviewer.
//
// The edge the engineer flagged: with a top-level property "pw" marked
// x-nava-secret, a subschema that applies to the same instance location
// (/pw) without being /properties/pw itself can still carry the value. Each
// case below is a schema a tenant admin can send to POST /api/v1/entity-types
// or .../versions. The value is plain text, so only the secret-path rule could
// catch it. The schema must be refused by metadata.Compile or by
// entitytypes.CheckSchemaSecrets (422 config_secret_in_content). A schema both
// accept stores the value in entity_definition_versions and in the append-only
// audit_log diff.

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

// r2Schema is a schema Compile accepts, with "pw" marked secret, plus extra
// root-level keywords (a JSON object body without braces, or "").
func r2Schema(rootExtra string) string {
	s := `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,` +
		`"required":["title","state"],"properties":{` +
		`"title":{"type":"string","minLength":1,"maxLength":80},` +
		`"state":{"type":"string","enum":["open","closed"],"default":"open"},` +
		`"pw":{"type":"string","x-nava-secret":true}}`
	if rootExtra != "" {
		s += "," + rootExtra
	}
	return s + "}"
}

func r2Compile(schema string) error {
	_, err := metadata.Compile(metadata.Definition{
		Key:          "ticket",
		Schema:       json.RawMessage(schema),
		NumberPrefix: "TKT",
		FixedColumns: map[string]string{"state": metadata.ColumnState},
		States: metadata.StateModel{
			Initial:     "open",
			States:      []string{"open", "closed"},
			Transitions: []metadata.Transition{{From: "open", To: "closed"}},
		},
	})
	return err
}

func TestAdversarialR2SecretViaRootApplicatorIsRefused(t *testing.T) {
	q := fmt.Sprintf("%q", advCanary)
	pwConst := `{"properties":{"pw":{"const":` + q + `}}}`
	cases := []struct{ name, root string }{
		{"E1 root allOf sets pw const", `"allOf":[` + pwConst + `]`},
		{"E2 root anyOf sets pw enum", `"anyOf":[{"properties":{"pw":{"enum":[` + q + `]}}}]`},
		{"E3 root oneOf sets pw examples", `"oneOf":[{"properties":{"pw":{"examples":[` + q + `]}}}]`},
		{"E4 root if/then sets pw const", `"if":{"required":["title"]},"then":` + pwConst},
		{"E5 root else sets pw const", `"if":{"required":["title"]},"else":` + pwConst},
		{"E6 root not wraps it", `"not":{"not":` + pwConst + `}`},
		{"E7 const on the root instance through allOf", `"allOf":[{"const":{"pw":` + q + `}}]`},
		{"E8 enum on the root instance through anyOf", `"anyOf":[{"enum":[{"pw":` + q + `}]}]`},
		{"E9 examples on the root instance through allOf", `"allOf":[{"examples":[{"pw":` + q + `}]}]`},
		{"E10 patternProperties matching pw", `"patternProperties":{"^pw$":{"const":` + q + `}}`},
		{"E11 dependentSchemas sets pw const", `"dependentSchemas":{"title":` + pwConst + `}`},
		{"E12 root allOf $ref into $defs", `"allOf":[{"$ref":"#/$defs/x"}],"$defs":{"x":` + pwConst + `}`},
		{"E13 root allOf $ref by anchor", `"allOf":[{"$ref":"#x"}],"$defs":{"x":{"$anchor":"x","properties":{"pw":{"const":` + q + `}}}}`},
	}
	t.Run("control", func(t *testing.T) {
		s := r2Schema("")
		if err := r2Compile(s); err != nil {
			t.Fatalf("Compile refuses the base schema: %v", err)
		}
		if err := entitytypes.CheckSchemaSecrets(json.RawMessage(s)); err != nil {
			t.Fatalf("CheckSchemaSecrets refuses the base schema: %v", err)
		}
	})
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := r2Schema(c.root)
			if !json.Valid([]byte(s)) {
				t.Fatalf("test bug: invalid JSON %s", s)
			}
			cerr := r2Compile(s)
			var inv *metadata.ErrInvalidDefinition
			if cerr != nil && !errors.As(cerr, &inv) {
				t.Fatalf("Compile: unexpected error %v", cerr)
			}
			serr := entitytypes.CheckSchemaSecrets(json.RawMessage(s))
			var sec *entitytypes.SecretInContentError
			if serr != nil && !errors.As(serr, &sec) {
				t.Fatalf("CheckSchemaSecrets: unexpected error %v", serr)
			}
			if cerr == nil && serr == nil {
				t.Fatalf("secret value %q reaches /pw (marked x-nava-secret) and both Compile and CheckSchemaSecrets accept it", advCanary)
			}
			for _, err := range []error{cerr, serr} {
				if err != nil && strings.Contains(err.Error(), advCanary) {
					t.Errorf("the refusal echoes the secret value: %v", err)
				}
			}
		})
	}
}
