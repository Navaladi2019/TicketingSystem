package entitytypes_test

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
)

func TestSecretPaths(t *testing.T) {
	cases := []struct {
		name   string
		schema string
		want   []string // the property whose data keywords are secret, as a pointer
	}{
		{"none", `{"type":"object","properties":{"a":{"type":"string"}}}`, nil},
		{"x-nava-secret", `{"properties":{"tok":{"x-nava-secret":true}}}`, []string{"/properties/tok"}},
		{"writeOnly", `{"properties":{"tok":{"writeOnly":true}}}`, []string{"/properties/tok"}},
		{"false marker", `{"properties":{"tok":{"x-nava-secret":false}}}`, nil},
		{"escaped name", `{"properties":{"a/b":{"x-nava-secret":true}}}`, []string{"/properties/a~1b"}},
		{"non-object property is skipped", `{"properties":{"x":true,"tok":{"x-nava-secret":true}}}`, []string{"/properties/tok"}},
		{"properties of another shape", `{"properties":"no"}`, nil},
		{"nested marker is not derived (Compile refuses it)", `{"properties":{"l":{"items":{"x-nava-secret":true}}}}`, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := entitytypes.SecretPaths(json.RawMessage(c.schema))
			if err != nil {
				t.Fatal(err)
			}
			var want []patchwalk.Pointer
			for _, p := range c.want {
				for _, kw := range []string{"const", "default", "enum", "examples"} {
					want = append(want, patchwalk.Pointer(p+"/"+kw))
				}
			}
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Errorf("paths = %v, want %v", got, want)
			}
		})
	}
	if _, err := entitytypes.SecretPaths(json.RawMessage(`{`)); err == nil {
		t.Error("malformed schema: want an error")
	}
}

func TestCheckSchemaSecretsLeavesBadDocumentsToCompile(t *testing.T) {
	for name, schema := range map[string]string{
		"not JSON":      `{`,
		"not an object": `[1]`,
		"invalid UTF-8": "{\"properties\":{\"tok\":{\"x-nava-secret\":true,\"title\":\"ÿ\"}}}",
		"bool property": `{"properties":{"x":true}}`,
	} {
		if err := entitytypes.CheckSchemaSecrets(json.RawMessage(schema)); err != nil {
			t.Errorf("%s: err = %v, want nil (Compile reports it)", name, err)
		}
	}
	var sec *entitytypes.SecretInContentError
	err := entitytypes.CheckSchemaSecrets(json.RawMessage(`{"properties":{"x":true,"tok":{"x-nava-secret":true,"default":"v"}}}`))
	if !errors.As(err, &sec) || sec.Path != "/properties/tok/default" {
		t.Errorf("err = %v, want a secret at /properties/tok/default", err)
	}
}
