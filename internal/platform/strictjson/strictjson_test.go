package strictjson_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/strictjson"
)

// esc writes each %u in s as a JSON backslash-u escape, so the test bodies
// say exactly which bytes go on the wire.
func esc(s string) string { return strings.ReplaceAll(s, "%u", `\u`) }

func TestCheck(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		body    string
		want    strictjson.Violation
		wantBad bool
	}{
		{name: "clean object", body: `{"a":{"b":[1,{"c":1}]},"d":"é"}`},
		{name: "empty", body: ``},
		{name: "whitespace", body: " \n"},
		{name: "same name in sibling objects", body: `[{"a":1},{"a":2}]`},
		{name: "names differ only by case", body: `{"a":1,"A":2}`},
		{name: "valid surrogate pair", body: esc(`{"a":"%ud83d%ude00"}`)},
		{name: "U+2028 and U+2029", body: esc(`{"a":"%u2028%u2029"}`)},
		// Syntax errors are not this check's business: the request validator
		// answers them 400, as before.
		{name: "truncated", body: `{"a":1`},
		{name: "not JSON", body: `hello`},
		{name: "trailing value", body: `{"a":1} {"a":1,"a":2}`},

		{name: "duplicate at top", body: `{"a":1,"a":2}`, wantBad: true,
			want: strictjson.Violation{Pointer: "/a", Rule: strictjson.RuleDuplicate}},
		{name: "duplicate at nested depth", body: `{"a":{"b":[1,{"c":1,"c":2}]}}`, wantBad: true,
			want: strictjson.Violation{Pointer: "/a/b/1/c", Rule: strictjson.RuleDuplicate}},
		{name: "duplicate spelled with an escape", body: esc(`{"%u0061":1,"a":2}`), wantBad: true,
			want: strictjson.Violation{Pointer: "/a", Rule: strictjson.RuleDuplicate}},
		{name: "duplicate name needing pointer escape", body: `{"x":{"a/b~":1,"a/b~":2}}`, wantBad: true,
			want: strictjson.Violation{Pointer: "/x/a~1b~0", Rule: strictjson.RuleDuplicate}},
		{name: "lone high surrogate", body: `{"a":["\ud800"]}`, wantBad: true,
			want: strictjson.Violation{Pointer: "/a/0", Rule: strictjson.RuleInvalidUTF8}},
		{name: "lone low surrogate", body: `{"a":"\udc00x"}`, wantBad: true,
			want: strictjson.Violation{Pointer: "/a", Rule: strictjson.RuleInvalidUTF8}},
		{name: "overlong UTF-8", body: "{\"a\":{\"b\":\"\xc0\xaf\"}}", wantBad: true,
			want: strictjson.Violation{Pointer: "/a/b", Rule: strictjson.RuleInvalidUTF8}},
		{name: "UTF-8 encoded surrogate", body: "{\"a\":\"\xed\xa0\x80\"}", wantBad: true,
			want: strictjson.Violation{Pointer: "/a", Rule: strictjson.RuleInvalidUTF8}},
		// The bad bytes are in a name, so the pointer stops at its object and
		// never carries them.
		{name: "invalid UTF-8 in a name", body: "{\"o\":{\"k\xff\":1}}", wantBad: true,
			want: strictjson.Violation{Pointer: "/o", Rule: strictjson.RuleInvalidUTF8}},
		{name: "invalid UTF-8 before a duplicate", body: "{\"a\":\"\xff\",\"a\":1}", wantBad: true,
			want: strictjson.Violation{Pointer: "/a", Rule: strictjson.RuleInvalidUTF8}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, bad := strictjson.Check([]byte(c.body))
			if bad != c.wantBad || got != c.want {
				t.Fatalf("Check(%q) = %+v, %v; want %+v, %v", c.body, got, bad, c.want, c.wantBad)
			}
		})
	}
}

func TestUnmarshalRefuses(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		`{"a":1,"a":2}`,
		`{"a":{"b":1,"b":2}}`,
		`{"a":"\ud800"}`,
		"{\"a\":\"\xc0\xaf\"}",
	} {
		var v map[string]any
		if err := strictjson.Unmarshal([]byte(body), &v); err == nil {
			t.Errorf("Unmarshal(%q) = nil error, want a refusal", body)
		}
	}
}

// TestUnmarshalRefusesInsideRawMembers: the request parsers decode the body
// into map[string]json.RawMessage and decode each member later, so the
// refusal must cover the nested bytes of a raw member too.
func TestUnmarshalRefusesInsideRawMembers(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		`{"data":{"a":{"b":1,"b":2}}}`,
		`{"data":[{"x":"\udfff"}]}`,
		"{\"data\":{\"a\":\"\xf0\x80\x80\xaf\"}}",
	} {
		var v map[string]json.RawMessage
		if err := strictjson.Unmarshal([]byte(body), &v); err == nil {
			t.Errorf("Unmarshal(%q) into raw members = nil error, want a refusal", body)
		}
	}
}

func TestUnmarshalAccepts(t *testing.T) {
	t.Parallel()
	var v struct {
		Name string `json:"name"`
	}
	if err := strictjson.Unmarshal([]byte(`{"name":"x","other":1}`), &v); err != nil {
		t.Fatal(err)
	}
	if v.Name != "x" {
		t.Fatalf("Name = %q, want x", v.Name)
	}
}
