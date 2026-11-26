package entitytypes

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

// TICK-1121 option A: the previous schema of a publish audit entry is read
// with each NUL escaped, so jsonb accepts it, and the escape is reported.
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
			got, escaped, err := escapeNUL(json.RawMessage(c.in))
			if err != nil {
				t.Fatal(err)
			}
			if escaped != c.escaped {
				t.Errorf("escaped = %v, want %v", escaped, c.escaped)
			}
			if !escaped && string(got) != c.in {
				t.Errorf("unchanged input must keep its bytes: got %s, want %s", got, c.in)
			}
			for i := range got {
				if got[i] == 0 {
					t.Fatalf("output holds a NUL byte: %q", got)
				}
			}
			var v any
			dec := json.NewDecoder(bytes.NewReader(got))
			dec.UseNumber()
			if err := dec.Decode(&v); err != nil {
				t.Fatalf("output is not JSON: %v: %s", err, got)
			}
			if !reflect.DeepEqual(v, any(c.want)) {
				t.Errorf("got %s, want %v", got, c.want)
			}
		})
	}
	if _, _, err := escapeNUL(json.RawMessage(`{"a":`)); err == nil {
		t.Error("malformed input: want an error")
	}
	if _, _, err := escapeNUL(json.RawMessage(`{"p":{"a\u0000b":1,"a\\u0000b":2}}`)); !errors.Is(err, errNULKeyCollision) {
		t.Errorf("colliding keys: err = %v, want errNULKeyCollision", err)
	}
}
