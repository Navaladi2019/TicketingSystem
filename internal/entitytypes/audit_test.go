package entitytypes_test

import (
	"encoding/json"
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
