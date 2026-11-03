package entitytypes_test

// Adversarial tests for P5-14 (TICK-432, PR #413), Security & Multi-Tenancy
// review. Each case is a schema that metadata.Compile accepts, so a tenant
// admin can send it to POST /api/v1/entity-types or .../versions. It carries
// a plain-text value (no secret shape, so only the secret-path rule can catch
// it) at an instance location that the schema marks x-nava-secret or
// writeOnly. The audit path must refuse it with *secrets.ErrSecretInDiff
// (422 config_secret_in_content), and the error must never carry the value.
//
// The pipeline mirrors auditVersion: SecretPaths(after) + jsondiff.Diff from
// the stored versions + secrets.Check (as audit.Append calls it) in the
// config domain.

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/jsondiff"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/secrets"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

const advCanary = "hunter2-canary-value"

// advSchema wraps extra top-level properties (and optional $defs) in a schema
// that Compile accepts.
func advSchema(extraProps, defs string) string {
	s := `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,` +
		`"required":["title","state"],"properties":{` +
		`"title":{"type":"string","minLength":1,"maxLength":80},` +
		`"state":{"type":"string","enum":["open","closed"],"default":"open"}`
	if extraProps != "" {
		s += "," + extraProps
	}
	s += "}"
	if defs != "" {
		s += `,"$defs":{` + defs + "}"
	}
	return s + "}"
}

// advCompile compiles schema. With wantRefused it asserts Compile's refusal
// and reports true: the case never reaches the audit check. Otherwise it
// asserts Compile accepts the schema.
func advCompile(t *testing.T, schema string, wantRefused bool) bool {
	t.Helper()
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
	if wantRefused {
		var inv *metadata.ErrInvalidDefinition
		if !errors.As(err, &inv) {
			t.Fatalf("Compile accepted a schema that holds a secret at a marked location (err = %v)", err)
		}
		return true
	}
	if err != nil {
		t.Fatalf("precondition: Compile refuses the schema: %v", err)
	}
	return false
}

// advCheck runs the audit secret check on before -> after and wants a refusal
// that does not echo the canary.
func advCheck(t *testing.T, before, after string) {
	t.Helper()
	paths, err := entitytypes.SecretPaths(json.RawMessage(after))
	if err != nil {
		t.Fatal(err)
	}
	var b json.RawMessage
	if before != "" {
		b = json.RawMessage(before)
	}
	diff, err := jsondiff.Diff(b, json.RawMessage(after))
	if err != nil {
		t.Fatal(err)
	}
	err = secrets.Check(diff, paths, secrets.DomainConfig)
	var sec *secrets.ErrSecretInDiff
	if !errors.As(err, &sec) {
		t.Fatalf("secret value %q at a secret-marked location was accepted (err = %v); secret paths = %v",
			advCanary, err, paths)
	}
	if strings.Contains(err.Error(), advCanary) || strings.Contains(string(sec.Path), advCanary) {
		t.Errorf("the refusal echoes the secret value: %v", err)
	}
}

func TestAdversarialSecretDefaultIsRefused(t *testing.T) {
	q := fmt.Sprintf("%q", advCanary)
	cases := []struct {
		name, props, defs string
		auditAlso         bool // also run the audit check: the secret path covers this placement (defence in depth behind Compile)
	}{
		// Control: the case PR #413 covers. Must stay green.
		{"control: default on the secret property", `"pw":{"type":"string","x-nava-secret":true,"default":` + q + `}`, "", true},

		// F1: the secret lives under a parent object whose own default sets it.
		{"F1a parent object default sets the secret member",
			`"creds":{"type":"object","properties":{"pw":{"type":"string","x-nava-secret":true}},"default":{"pw":` + q + `}}`, "", false},
		{"F1b parent enum carries the secret member",
			`"creds":{"type":"object","properties":{"pw":{"type":"string","writeOnly":true}},"enum":[{"pw":` + q + `}]}`, "", false},
		{"F1c array default with secret items",
			`"pins":{"type":"array","items":{"type":"string","x-nava-secret":true},"default":[` + q + `]}`, "", false},
		{"F1d examples on the parent object",
			`"creds":{"type":"object","properties":{"pw":{"type":"string","x-nava-secret":true}},"examples":[{"pw":` + q + `}]}`, "", false},

		// F2: the marker applies to the same instance through an in-place applicator.
		{"F2a marker inside allOf on the same property",
			`"pw":{"type":"string","allOf":[{"x-nava-secret":true}],"default":` + q + `}`, "", false},
		{"F2b marker reached through $ref inside allOf",
			`"pw":{"allOf":[{"$ref":"#/$defs/secret"}],"default":` + q + `}`, `"secret":{"type":"string","x-nava-secret":true}`, false},

		// F3: references SecretPaths does not follow.
		{"F3a $ref by $anchor",
			`"pw":{"$ref":"#secret","default":` + q + `}`, `"secret":{"$anchor":"secret","type":"string","x-nava-secret":true}`, false},
		{"F3b $ref chain longer than maxRefHops",
			`"pw":{"$ref":"#/$defs/a0","default":` + q + `}`, advChain(9), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := advSchema(c.props, c.defs)
			// Compile refuses every one of these placements (ADR-004 R6).
			advCompile(t, s, true)
			if c.auditAlso {
				advCheck(t, "", s) // the whole-schema check refuses it too
			}
		})
	}
}

// advChain returns $defs a0 -> a1 -> ... -> a<n-1> -> secret.
func advChain(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		next := fmt.Sprintf("#/$defs/a%d", i+1)
		if i == n-1 {
			next = "#/$defs/secret"
		}
		fmt.Fprintf(&b, `"a%d":{"$ref":%q},`, i, next)
	}
	b.WriteString(`"secret":{"type":"string","x-nava-secret":true}`)
	return b.String()
}

// F4: the check runs on the diff, not on the new schema. A v2 that only adds
// the marker to a property whose default already holds the value is accepted,
// and the v2 version row stores a default at a secret path.
func TestAdversarialPublishMarkingAnExistingDefaultSecretIsRefused(t *testing.T) {
	q := fmt.Sprintf("%q", advCanary)
	v1 := advSchema(`"pw":{"type":"string","default":`+q+`}`, "")
	v2 := advSchema(`"pw":{"type":"string","x-nava-secret":true,"default":`+q+`}`, "")
	advCompile(t, v1, false)
	// Compile refuses a default on a marked property (ADR-004 R6); the full
	// schema check refuses it as well when Compile is bypassed.
	advCompile(t, v2, true)
	advCheck(t, "", v2) // CheckSchemaSecrets checks the whole schema, not the v1 -> v2 delta
}
