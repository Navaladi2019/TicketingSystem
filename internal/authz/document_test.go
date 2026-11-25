package authz_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
)

// updateGolden rewrites the format-1 golden files instead of comparing them:
// NAVA_UPDATE_GOLDEN=1 go test ./internal/authz -run TestFormat1ConvertsToGoldenTree
func updateGolden() bool { return os.Getenv("NAVA_UPDATE_GOLDEN") == "1" }

type validFixture struct {
	Name string          `json:"name"`
	Doc  json.RawMessage `json:"doc"`
}

// invalidFixture is one case of testdata/format2_invalid.json, shared with the
// publish route (TICK-939 P11-07) and the database CHECK (TICK-940 P11-08).
// Doc is kept raw so a duplicate key survives loading. GoOnly marks a case the
// CHECK cannot see, because jsonb has already dropped the duplicate.
type invalidFixture struct {
	Name   string `json:"name"`
	GoOnly bool   `json:"go_only"`
	// Check marks a case Decode accepts and the save-time checks (Prepare, D4)
	// refuse; the database CHECK refuses it too unless RouteOnly. NeedsTenant
	// marks a case that needs the tenant's ids; MaxBytes is the byte limit the
	// case is checked against, 0 for the default.
	Check       bool            `json:"check"`
	RouteOnly   bool            `json:"route_only"`
	NeedsTenant bool            `json:"needs_tenant"`
	MaxBytes    int             `json:"max_bytes"`
	Doc         json.RawMessage `json:"doc"`
	// Body, when set, is the whole publish body Prepare is given instead of
	// {"rules": Doc}, for a problem outside the rules value (a repeated
	// top-level name). Its problems are pointers into the body, not into Doc.
	Body     json.RawMessage `json:"body"`
	Problems []struct {
		Pointer string `json:"pointer"`
		Code    string `json:"code"`
	} `json:"problems"`
}

func load[T any](t *testing.T, path string) []T {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []T
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	if len(out) == 0 {
		t.Fatalf("%s has no cases", path)
	}
	return out
}

func sameJSON(t *testing.T, a, b []byte) bool {
	t.Helper()
	var x, y any
	if err := json.Unmarshal(a, &x); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &y); err != nil {
		t.Fatal(err)
	}
	return reflect.DeepEqual(x, y)
}

func TestDecodeFormat2RoundTrips(t *testing.T) {
	t.Parallel()
	for _, f := range load[validFixture](t, "testdata/format2_valid.json") {
		t.Run(f.Name, func(t *testing.T) {
			t.Parallel()
			d, err := authz.Decode(f.Doc)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if d.Format != authz.FormatV2 {
				t.Fatalf("format = %d, want 2", d.Format)
			}
			out, err := authz.Encode(d)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			if !sameJSON(t, f.Doc, out) {
				t.Fatalf("round trip changed the document:\n in: %s\nout: %s", f.Doc, out)
			}
			// The stored form decodes to the same document.
			again, err := authz.Decode(out)
			if err != nil || !reflect.DeepEqual(again, d) {
				t.Fatalf("re-decode = %+v, %v; want %+v", again, err, d)
			}
		})
	}
}

func TestDecodeInvalidFixturesYieldExactlyTheirProblems(t *testing.T) {
	t.Parallel()
	for _, f := range load[invalidFixture](t, "testdata/format2_invalid.json") {
		if f.Check {
			continue // Decode accepts it; TestPrepareRefusesTheFixture covers it
		}
		t.Run(f.Name, func(t *testing.T) {
			t.Parallel()
			d, err := authz.Decode(f.Doc)
			var de *authz.DecodeError
			if !errors.As(err, &de) {
				t.Fatalf("Decode = %+v, %v; want a *DecodeError", d, err)
			}
			if !reflect.DeepEqual(d, authz.Document{}) {
				t.Errorf("Decode returned a document with its error: %+v", d)
			}
			var got, want []string
			for _, p := range de.Problems {
				got = append(got, p.Pointer+" "+p.Rule)
			}
			for _, p := range f.Problems {
				want = append(want, p.Pointer+" "+p.Code)
			}
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Fatalf("problems = %q, want %q", got, want)
			}
			if !strings.HasPrefix(err.Error(), "authz: invalid access document: ") {
				t.Errorf("error text = %q", err.Error())
			}
		})
	}
}

// Every C1 code a fixture names is a declared code, so a typo in the shared
// fixture cannot pass by matching nothing.
func TestInvalidFixtureCodesAreDeclared(t *testing.T) {
	t.Parallel()
	declared := []string{
		authz.RuleRequired, authz.RuleType, authz.RuleUnknownKey, authz.RuleValue, authz.RuleEmpty, authz.RuleCondition,
		authz.RuleInvalidName, authz.RuleDuplicateName, authz.RuleUnresolvedRef, authz.RuleRefCycle, authz.RuleChildCount,
		authz.RuleDepthExceeded, authz.RulePolicyLeavesExceeded, authz.RulePoliciesExceeded, authz.RuleDocLeavesExceeded,
		authz.RuleForeignID, authz.RuleNotAvailable, authz.RuleDocumentTooLarge, authz.RuleDuplicate,
		authz.RuleUnderSetsExceeded, authz.RuleUnderDuplicateID,
	}
	for _, f := range load[invalidFixture](t, "testdata/format2_invalid.json") {
		for _, p := range f.Problems {
			if !slices.Contains(declared, p.Code) {
				t.Errorf("%s: code %q is not declared", f.Name, p.Code)
			}
		}
	}
}

func TestDecodeRefusesWhatIsNotJSON(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{"", "   ", "null", "2", `{"format": 2`, `{"format":2,"conditions":[],"policies":[]} {}`, `[{"id":`, `[1] [2]`} {
		_, err := authz.Decode([]byte(raw))
		var de *authz.DecodeError
		if !errors.As(err, &de) || len(de.Problems) == 0 {
			t.Errorf("Decode(%q) = %v, want a *DecodeError", raw, err)
		}
	}
}

func TestDecodeFormat1IsCheckedAsToday(t *testing.T) {
	t.Parallel()
	for raw, want := range map[string]string{
		`[{"id":"r","ops":["read"],"who":{"requester":true},"bad":1}]`:                            "/0/bad unknown_key",
		`[{"id":"r","ops":["read"],"who":{"requester":true}},{"id":"r","ops":["read"],"who":{}}]`: "/1/id duplicate",
		`[{"id":"r","ops":["read"],"who":{"requester":"yes"}}]`:                                   "/0/who/requester type",
		`[{"id":"r","id":"s","ops":["read"],"who":{}}]`:                                           "/0/id unknown_key",
	} {
		_, err := authz.Decode([]byte(raw))
		var de *authz.DecodeError
		if !errors.As(err, &de) {
			t.Errorf("Decode(%s) = %v, want a *DecodeError", raw, err)
			continue
		}
		var got []string
		for _, p := range de.Problems {
			got = append(got, p.Pointer+" "+p.Rule)
		}
		if !slices.Equal(got, []string{want}) {
			t.Errorf("Decode(%s) problems = %q, want [%q]", raw, got, want)
		}
	}
}

// Every stored-shape format-1 document converts to its documented tree. The
// inputs are the shapes internal/authz and its callers store today: the
// onboarding default, the handbook example, every subject at once, the
// pipeline's shapes (a rule with no subject among them) and rule ids that are
// not valid names. Set NAVA_UPDATE_GOLDEN=1 to rewrite the golden files,
// then review the diff.
func TestFormat1ConvertsToGoldenTree(t *testing.T) {
	t.Parallel()
	inputs, err := filepath.Glob("testdata/format1/*.json")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, in := range inputs {
		if strings.HasSuffix(in, ".golden.json") {
			continue
		}
		n++
		t.Run(filepath.Base(in), func(t *testing.T) {
			raw, err := os.ReadFile(in)
			if err != nil {
				t.Fatal(err)
			}
			d, err := authz.Decode(raw)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			out, err := json.MarshalIndent(d, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, '\n')
			golden := strings.TrimSuffix(in, ".json") + ".golden.json"
			if updateGolden() {
				if err := os.WriteFile(golden, out, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("%v (run NAVA_UPDATE_GOLDEN=1 go test ./internal/authz -run TestFormat1ConvertsToGoldenTree)", err)
			}
			if !bytes.Equal(bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n")), out) {
				t.Fatalf("%s: converted tree differs from the golden file:\n%s", in, out)
			}
			// Conversion is deterministic: the same row gives the same names.
			d2, _ := authz.Decode(raw)
			if !reflect.DeepEqual(d, d2) {
				t.Fatal("two conversions of the same row differ")
			}
		})
	}
	if n < 5 {
		t.Fatalf("found %d format-1 fixtures, want at least 5", n)
	}
}

func TestConvertedPolicyNamesAreValidAndUnique(t *testing.T) {
	t.Parallel()
	rules := []authz.Rule{
		{ID: "a-b"}, {ID: "a_b"}, {ID: "a_b"}, {ID: ""}, {ID: "_x"}, {ID: "é"},
		{ID: strings.Repeat("z", 70)}, {ID: strings.Repeat("z", 63)}, {ID: strings.Repeat("Z", 70)},
	}
	d := authz.ConvertFormat1(rules)
	seen := map[string]bool{}
	for i, p := range d.Policies {
		if !authz.ValidName(p.Name) {
			t.Errorf("policy %d name %q is not valid", i, p.Name)
		}
		if seen[p.Name] {
			t.Errorf("policy %d name %q repeats", i, p.Name)
		}
		seen[p.Name] = true
		// A renamed policy keeps its stored id; a kept name has no legacy id.
		if renamed := p.Name != rules[i].ID; (renamed && p.LegacyID != rules[i].ID) || (!renamed && p.LegacyID != "") {
			t.Errorf("policy %d (%q): legacy_id %q", i, p.Name, p.LegacyID)
		}
	}
	if d.Policies[1].Name != "a_b" || d.Policies[0].Name == "a_b" {
		t.Errorf("a valid id keeps its name over a derived one: %q, %q", d.Policies[0].Name, d.Policies[1].Name)
	}
}

func TestConvertedDocumentEncodesAndDecodes(t *testing.T) {
	t.Parallel()
	g, u := uuid.New(), uuid.New()
	d := authz.ConvertFormat1([]authz.Rule{{
		ID: "desk", Ops: []string{authz.OpRead},
		Who:   authz.Subject{Groups: []uuid.UUID{g}, Users: []uuid.UUID{u}, Requester: true},
		Where: []authz.Cond{{Column: "state", Op: authz.CondIn, Values: []string{"open"}}},
	}})
	raw, err := authz.Encode(d)
	if err != nil {
		t.Fatal(err)
	}
	back, err := authz.Decode(raw)
	if err != nil {
		t.Fatalf("a converted document does not decode as format 2: %v\n%s", err, raw)
	}
	if !reflect.DeepEqual(back, d) {
		t.Fatalf("decode(encode(d)) = %+v, want %+v", back, d)
	}
}

func TestEncodeEmptyDocumentHasArrays(t *testing.T) {
	t.Parallel()
	raw, err := authz.Encode(authz.Document{Format: authz.FormatV2})
	if err != nil || string(raw) != `{"format":2,"conditions":[],"policies":[]}` {
		t.Fatalf("Encode = %s, %v", raw, err)
	}
}

func TestEncodeRefusesANodeWithNoKind(t *testing.T) {
	t.Parallel()
	_, err := authz.Encode(authz.Document{Format: authz.FormatV2, Policies: []authz.NamedPolicy{{Name: "p", Ops: []string{"read"}}}})
	if err == nil {
		t.Fatal("Encode accepted a policy whose when has no kind")
	}
}

func TestNodeKind(t *testing.T) {
	t.Parallel()
	for want, n := range map[string]authz.Node{
		"":                 {},
		authz.NodeAll:      {All: []authz.Node{}},
		authz.NodeAny:      {Any: []authz.Node{}},
		authz.NodeNot:      {Not: &authz.Node{}},
		authz.NodeRef:      {Ref: "x"},
		authz.NodeMemberOf: {MemberOf: &authz.MemberOf{}},
		authz.NodeCmp:      {Cmp: &authz.Cond{}},
		authz.NodeShared:   {Shared: true},
		authz.NodeUnder:    {Under: &authz.Under{}},
	} {
		if got := n.Kind(); got != want {
			t.Errorf("Kind() = %q, want %q", got, want)
		}
	}
}
