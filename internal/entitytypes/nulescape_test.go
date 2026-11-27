package entitytypes

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

// TICK-1121 option A, ADR-004-E10 CI-A19 (e): the previous schema of a
// publish audit entry is read with each NUL escaped, so jsonb accepts it,
// and the escape is reported.
func TestEscapeNUL(t *testing.T) {
	cases := []struct {
		name, in string
		want     map[string]any
		escaped  bool
	}{
		{"none", `{"a":"b"}`, map[string]any{"a": "b"}, false},
		{"literal backslash-u0000 is left alone", `{"d":"x\\u0000y"}`, map[string]any{"d": `x\u0000y`}, false},
		{"NUL in a value", `{"d":"x\u0000y"}`, map[string]any{"d": `x\u0000y`}, true},
		{"NUL in a key", `{"p":{"a\u0000b":{}}}`, map[string]any{"p": map[string]any{`a\u0000b`: map[string]any{}}}, true},
		{"NUL in an array", `{"e":["\u0000",1.50]}`, map[string]any{"e": []any{`\u0000`, json.Number("1.50")}}, true},
		// The trap: a real NUL beside a literal one. Only the real one changes.
		{"both", `{"r":"\u0000","l":"\\u0000"}`, map[string]any{"r": `\u0000`, "l": `\u0000`}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := escapeNUL(json.RawMessage(c.in))
			if err != nil {
				t.Fatal(err)
			}
			if got.Escaped != c.escaped || got.Collided {
				t.Errorf("escaped, collided = %v, %v, want %v, false", got.Escaped, got.Collided, c.escaped)
			}
			if !got.Escaped && string(got.Doc) != c.in {
				t.Errorf("unchanged input must keep its bytes: got %s, want %s", got.Doc, c.in)
			}
			var v any
			dec := json.NewDecoder(bytes.NewReader(got.Doc))
			dec.UseNumber()
			if err := dec.Decode(&v); err != nil {
				t.Fatalf("output is not JSON: %v: %s", err, got.Doc)
			}
			if !reflect.DeepEqual(v, any(c.want)) {
				t.Errorf("got %s, want %v", got.Doc, c.want)
			}
		})
	}
	if _, err := escapeNUL(json.RawMessage(`{"a":`)); err == nil {
		t.Error("malformed input: want an error")
	}
}

// ADR-004-E10 E10-5, CI-A19 (f): two keys of one object whose escaped forms
// are equal are a collision, reported and not refused, in both shapes.
func TestEscapeNULCollision(t *testing.T) {
	for name, in := range map[string]string{
		"a NUL key next to a literal key":      `{"p":{"a\u0000b":1,"a\\u0000b":2}}`,
		"two NUL keys with one escaped form":   `{"p":{"a\u0000\\u0000":1,"a\\u0000\u0000":2}}`,
		"literal key first, then the NUL key":  `{"p":{"a\\u0000b":2,"a\u0000b":1}}`,
		"collision nested in an array element": `{"e":[{"\u0000":1,"\\u0000":2}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			got, err := escapeNUL(json.RawMessage(in))
			if err != nil {
				t.Fatal(err)
			}
			if !got.Collided || !got.Escaped || got.Doc != nil {
				t.Errorf("got %+v, want collided and escaped with no document", got)
			}
		})
	}
}
