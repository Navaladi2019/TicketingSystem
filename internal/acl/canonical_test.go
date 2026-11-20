package acl_test

import (
	"errors"
	"math/rand"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/acl"
)

// The expected texts are what PostgreSQL prints for '<input>'::jsonb::text;
// the integration parity test (internal/db/dbtest) asserts it against a real
// server.
func TestCanonicalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, in, want string }{
		{"empty containers", `{"a":[],"b":{}}`, `{"a": [], "b": {}}`},
		{"spacing", "{ \"a\" :\n[ 1 , 2 ] }", `{"a": [1, 2]}`},
		{"keys by length then bytes", `{"bb":1,"a":2,"ab":3,"b":4}`, `{"a": 2, "b": 4, "ab": 3, "bb": 1}`},
		{"repeated key keeps the last", `{"a":1,"a":2}`, `{"a": 2}`},
		{"literals", `[true,false,null]`, `[true, false, null]`},
		{"slash and angle brackets", `"a/b<c>&d"`, `"a/b<c>&d"`},
		{"escaped slash", `"a\/b"`, `"a/b"`},
		{"quote and backslash", `"\"\\"`, `"\"\\"`},
		{"b f n r t", `"\b\f\n\r\t"`, `"\b\f\n\r\t"`},
		{"other C0 controls", `"\u0001\u001f"`, `"\u0001\u001f"`},
		{"DEL is raw", "\"\u007f\"", "\"\u007f\""},
		{"U+2028 and U+2029 are raw", `"  "`, "\"  \""},
		{"escaped ascii is raw", `"Aé"`, `"Aé"`},
		{"non-ASCII", `"日本語😀"`, `"日本語😀"`},
		{"surrogate pair", `"😀"`, `"😀"`},
		{"integer", `2`, `2`},
		{"scale kept", `2.0`, `2.0`},
		{"exponent gone", `1e2`, `100`},
		{"upper exponent and plus", `1E+2`, `100`},
		{"negative exponent", `25e-1`, `2.5`},
		{"small", `1e-3`, `0.001`},
		{"leading zero of fraction", `0.50`, `0.50`},
		{"negative zero", `-0`, `0`},
		{"negative zero scale", `-0.0`, `0.0`},
		{"negative", `-12.50`, `-12.50`},
		{"exponent with fraction", `1.5e3`, `1500`},
		{"exponent into fraction", `12.345e-1`, `1.2345`},
		{"key escapes", `{"a\nb":1}`, `{"a\nb": 1}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := acl.CanonicalJSON([]byte(tc.in))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("CanonicalJSON(%s) = %s, want %s", tc.in, got, tc.want)
			}
			n, err := acl.CanonicalSize([]byte(tc.in))
			if err != nil || n != len(tc.want) {
				t.Errorf("CanonicalSize = %d, %v, want %d", n, err, len(tc.want))
			}
		})
	}
}

func TestCanonicalJSONRefuses(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, in string
		want     error
	}{
		{"invalid UTF-8", "\"\xff\"", acl.ErrInvalidText},
		{"truncated UTF-8", "\"\xe2\x82\"", acl.ErrInvalidText},
		{"NUL escape", `"a\u0000b"`, acl.ErrInvalidText},
		{"lone high surrogate", `"\ud800"`, acl.ErrInvalidText},
		{"lone low surrogate", `"\udc00"`, acl.ErrInvalidText},
		{"high surrogate then plain", `"\ud800a"`, acl.ErrInvalidText},
		{"high surrogate then high", `"\ud800\ud800"`, acl.ErrInvalidText},
		{"NUL in a key", `{"\u0000":1}`, acl.ErrInvalidText},
		{"huge exponent", `1e100000`, acl.ErrNumber},
		{"huge negative exponent", `1e-100000`, acl.ErrNumber},
		{"not JSON", `{`, acl.ErrNotJSON},
		{"empty", ``, acl.ErrNotJSON},
		{"two values", `1 2`, acl.ErrNotJSON},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := acl.CanonicalJSON([]byte(tc.in)); !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

// Writing the canonical text again changes nothing, whatever the input.
func TestCanonicalJSONIsIdempotent(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(1))
	for range 500 {
		in := randomJSON(r, 4)
		a, err := acl.CanonicalJSON([]byte(in))
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		b, err := acl.CanonicalJSON(a)
		if err != nil || string(a) != string(b) {
			t.Fatalf("%s: %s then %s, %v", in, a, b, err)
		}
	}
}

// RandomJSON is shared with the parity test's corpus by its seed.
func randomJSON(r *rand.Rand, depth int) string {
	atoms := []string{`null`, `true`, `false`, `0`, `-1`, `1.50`, `1e2`, `"x"`, `"a/b<>&"`, `" "`, `"é\n"`, `"\u0001"`, `""`}
	if depth == 0 || r.Intn(3) == 0 {
		return atoms[r.Intn(len(atoms))]
	}
	n := r.Intn(4)
	var parts []string
	if r.Intn(2) == 0 {
		for range n {
			parts = append(parts, randomJSON(r, depth-1))
		}
		return "[" + strings.Join(parts, ",") + "]"
	}
	keys := []string{"a", "b", "ab", "name", "é", "a/b", "when", ""}
	for range n {
		k := keys[r.Intn(len(keys))]
		parts = append(parts, `"`+k+`":`+randomJSON(r, depth-1))
	}
	return "{" + strings.Join(parts, ",") + "}"
}
