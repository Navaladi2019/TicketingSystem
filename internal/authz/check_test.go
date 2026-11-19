package authz_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
)

func body(rules string) []byte { return []byte(`{"rules":` + rules + `}`) }

func pointersOf(ps []authz.Problem) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Pointer + " " + p.Rule
	}
	return out
}

// Every case of the shared fixture is refused by Prepare with exactly its
// problems, under /rules; the ones that need the tenant's ids are checked by
// the integration test of the route.
func TestPrepareRefusesTheFixture(t *testing.T) {
	t.Parallel()
	for _, f := range load[invalidFixture](t, "testdata/format2_invalid.json") {
		if f.NeedsTenant {
			continue
		}
		t.Run(f.Name, func(t *testing.T) {
			t.Parallel()
			limit := authz.DefaultMaxDocumentBytes
			if f.MaxBytes > 0 {
				limit = f.MaxBytes
			}
			p, ps := authz.Prepare(body(string(f.Doc)), limit)
			if p != nil {
				t.Fatalf("Prepare accepted %s", f.Doc)
			}
			var want []string
			for _, w := range f.Problems {
				want = append(want, "/rules"+w.Pointer+" "+w.Code)
			}
			got := pointersOf(ps)
			slices.Sort(want)
			slices.Sort(got)
			if !slices.Equal(got, want) {
				t.Fatalf("problems = %q, want %q", got, want)
			}
		})
	}
}

func TestPrepareAcceptsTheValidFixtures(t *testing.T) {
	t.Parallel()
	for _, f := range load[validFixture](t, "testdata/format2_valid.json") {
		t.Run(f.Name, func(t *testing.T) {
			t.Parallel()
			p, ps := authz.Prepare(body(string(f.Doc)), authz.DefaultMaxDocumentBytes)
			if strings.Contains(string(f.Doc), `"under"`) {
				// Decodes, but under is not available before S31.
				for _, pr := range ps {
					if pr.Rule != authz.RuleNotAvailable {
						t.Errorf("problem %s %s", pr.Pointer, pr.Rule)
					}
				}
				if len(ps) == 0 {
					t.Error("under was accepted")
				}
				return
			}
			if len(ps) > 0 {
				t.Fatalf("problems = %q", pointersOf(ps))
			}
			if p.Format != 2 {
				t.Errorf("Format = %d", p.Format)
			}
			if !sameJSON(t, p.Rules, f.Doc) {
				t.Errorf("stored rules %s differ from the submitted document %s", p.Rules, f.Doc)
			}
			if string(p.Masks) != "[]" {
				t.Errorf("masks = %s, want []", p.Masks)
			}
			if _, err := authz.Decode(p.Rules); err != nil {
				t.Errorf("the stored text does not decode: %v", err)
			}
		})
	}
}

// The stored text is the writer's canonical form, not the client's bytes.
func TestPrepareStoresTheCanonicalText(t *testing.T) {
	t.Parallel()
	p, ps := authz.Prepare([]byte(`{"masks":[],"rules":{ "policies":[{"when":{"shared":true},"ops":["read"],"name":"p"}],"format":2.0,"conditions":[]}}`), authz.DefaultMaxDocumentBytes)
	if len(ps) > 0 {
		t.Fatal(pointersOf(ps))
	}
	if got := string(p.Rules); !strings.HasPrefix(got, `{"format": 2.0, "policies": [{"ops": ["read"], "name": "p", "when": {"shared": true}}], "conditions": []}`) {
		t.Errorf("stored text = %s", got)
	}
}

func TestPrepareCapsAreInclusive(t *testing.T) {
	t.Parallel()
	shared := `{"shared":true}`
	anyOf := func(n int) string { return `{"any":[` + strings.TrimSuffix(strings.Repeat(shared+",", n), ",") + `]}` }
	allAnys := func(anys, per int, extra ...string) string {
		parts := make([]string, anys)
		for i := range parts {
			parts[i] = anyOf(per)
		}
		return `{"all":[` + strings.Join(append(parts, extra...), ",") + `]}`
	}
	nots := func(k int) string { return strings.Repeat(`{"not":`, k) + shared + strings.Repeat("}", k) }
	pol := func(name, when string) string { return fmt.Sprintf(`{"name":%q,"ops":["read"],"when":%s}`, name, when) }
	doc := func(pols ...string) string {
		return `{"format":2,"conditions":[],"policies":[` + strings.Join(pols, ",") + `]}`
	}
	manyPolicies := func(n int) string {
		ps := make([]string, n)
		for i := range ps {
			ps[i] = pol(fmt.Sprintf("p%d", i), shared)
		}
		return doc(ps...)
	}
	for _, tc := range []struct {
		name, rules, want string
	}{
		{"depth 8", doc(pol("p", nots(7))), ""},
		{"depth 9", doc(pol("p", nots(8))), "/rules/policies/0/when depth_exceeded"},
		{"16 children", doc(pol("p", anyOf(16))), ""},
		{"17 children", doc(pol("p", anyOf(17))), "/rules/policies/0/when/any child_count"},
		{"0 children", doc(pol("p", `{"all":[]}`)), "/rules/policies/0/when/all child_count"},
		{"128 leaves in a policy", doc(pol("p", allAnys(8, 16))), ""},
		{"129 leaves in a policy", doc(pol("p", allAnys(8, 16, shared))), "/rules/policies document_leaves_exceeded,/rules/policies/0/when policy_leaves_exceeded"},
		{"64 policies", manyPolicies(64), ""},
		{"65 policies", manyPolicies(65), "/rules/policies policies_exceeded"},
		{"128 leaves in a document", doc(pol("a", allAnys(4, 16)), pol("b", allAnys(4, 16))), ""},
		{"129 leaves in a document", doc(pol("a", allAnys(4, 16)), pol("b", allAnys(4, 16, shared))), "/rules/policies document_leaves_exceeded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, ps := authz.Prepare(body(tc.rules), authz.DefaultMaxDocumentBytes)
			got := strings.Join(pointersOf(ps), ",")
			if got != tc.want {
				t.Errorf("problems = %q, want %q", got, tc.want)
			}
		})
	}
}

// A doubling chain of conditions is counted, not expanded.
func TestPrepareCountsADoublingChainWithoutExpandingIt(t *testing.T) {
	t.Parallel()
	conds := []string{`{"name":"c0","expr":{"shared":true}}`}
	for i := 1; i < 60; i++ {
		conds = append(conds, fmt.Sprintf(`{"name":"c%d","expr":{"all":[{"ref":"c%d"},{"ref":"c%d"}]}}`, i, i-1, i-1))
	}
	rules := `{"format":2,"conditions":[` + strings.Join(conds, ",") + `],"policies":[{"name":"p","ops":["read"],"when":{"ref":"c59"}}]}`
	_, ps := authz.Prepare(body(rules), 1<<20)
	got := pointersOf(ps)
	if !slices.Contains(got, "/rules/policies/0/when depth_exceeded") {
		t.Errorf("problems = %q, want depth_exceeded", got)
	}
}

func TestPrepareByteLimitIsOnStoredTextAndOnMasks(t *testing.T) {
	t.Parallel()
	rules := `{"format":2,"conditions":[],"policies":[{"name":"p","ops":["read"],"when":{"shared":true}}]}`
	p, ps := authz.Prepare(body(rules), len(p0(t, rules)))
	if len(ps) > 0 || p == nil {
		t.Fatalf("a document exactly at the limit: %q", pointersOf(ps))
	}
	_, ps = authz.Prepare(body(rules), len(p0(t, rules))-1)
	if got := pointersOf(ps); !slices.Equal(got, []string{"/rules document_too_large"}) {
		t.Errorf("one byte over: %q", got)
	}
	masks := `[{"field":"f","read":{"shared":false},"write":{"shared":false}}]`
	_, ps = authz.Prepare([]byte(`{"rules":`+rules+`,"masks":`+masks+`}`), 20)
	if got := pointersOf(ps); !slices.Contains(got, "/masks document_too_large") {
		t.Errorf("masks over the limit: %q", got)
	}
	if got := authz.CheckSize([]byte(`{"rules":`+rules+`}`), 10); !slices.Equal(pointersOf(got), []string{"/rules document_too_large"}) {
		t.Errorf("CheckSize = %q", pointersOf(got))
	}
}

func p0(t *testing.T, rules string) []byte {
	t.Helper()
	p, ps := authz.Prepare(body(rules), 1<<20)
	if p == nil {
		t.Fatal(pointersOf(ps))
	}
	return p.Rules
}

func TestPrepareFormat1IsStoredAsFormat2(t *testing.T) {
	t.Parallel()
	p, ps := authz.Prepare([]byte(`{"rules":[{"id":"Read Own","ops":["read"],"who":{"requester":true}}],"masks":[]}`), authz.DefaultMaxDocumentBytes)
	if len(ps) > 0 {
		t.Fatal(pointersOf(ps))
	}
	if p.Format != 1 {
		t.Errorf("Format = %d, want 1", p.Format)
	}
	d, err := authz.Decode(p.Rules)
	if err != nil {
		t.Fatalf("stored text does not decode: %v", err)
	}
	if d.Format != authz.FormatV2 || len(d.Policies) != 1 || d.Policies[0].Name != "read_own" {
		t.Errorf("document = %+v", d)
	}
	// format-2 input refuses legacy_id, so it is not stored.
	if strings.Contains(string(p.Rules), "legacy_id") {
		t.Errorf("stored text carries legacy_id: %s", p.Rules)
	}
	if _, perr := authz.Prepare([]byte(`{"rules":`+string(p.Rules)+`}`), authz.DefaultMaxDocumentBytes); perr != nil {
		t.Errorf("the stored text is not a document the route accepts: %q", pointersOf(perr))
	}
}

func TestPrepareFormat1RuleWithNoSubjectIsChildCount(t *testing.T) {
	t.Parallel()
	_, ps := authz.Prepare([]byte(`{"rules":[{"id":"a","ops":["read"],"who":{"requester":true}},{"id":"b","ops":["read"],"who":{}}]}`), authz.DefaultMaxDocumentBytes)
	if got := pointersOf(ps); !slices.Equal(got, []string{"/rules/1/who child_count"}) {
		t.Errorf("problems = %q", got)
	}
}

// TICK-996 (F3): a format-1 rule whose converted tree breaks the child cap is
// refused by Prepare, so Decode in publishRaw never meets it.
func TestPrepareFormat1OverTheChildCapIsChildCount(t *testing.T) {
	t.Parallel()
	where := strings.TrimSuffix(strings.Repeat(`{"column":"state","op":"in","values":["new"]},`, authz.MaxChildren), ",")
	_, ps := authz.Prepare([]byte(`{"rules":[{"id":"a","ops":["read"],"who":{"requester":true}},{"id":"b","ops":["read"],"who":{"requester":true},"where":[`+where+`]}]}`), authz.DefaultMaxDocumentBytes)
	if got := pointersOf(ps); !slices.Equal(got, []string{"/rules/1 child_count"}) {
		t.Errorf("problems = %q", got)
	}
}

func TestPrepareFormat1KeepsItsOwnProblems(t *testing.T) {
	t.Parallel()
	_, ps := authz.Prepare([]byte(`{"rules":[{"id":"a","ops":["read"],"who":{"requester":true},"bad":1}]}`), authz.DefaultMaxDocumentBytes)
	if got := pointersOf(ps); !slices.Equal(got, []string{"/rules/0/bad unknown_key"}) {
		t.Errorf("problems = %q", got)
	}
}

func TestPrepareTopLevel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, body, want string }{
		{"not an object", `[]`, " type"},
		{"no rules", `{}`, "/rules required"},
		{"unknown key", `{"rules":[],"x":1}`, "/x unknown_key"},
		{"rules a string", `{"rules":"x"}`, "/rules type"},
		{"rules null", `{"rules":null}`, "/rules type"},
		{"masks not an array", `{"rules":[],"masks":{}}`, "/masks type"},
		{"duplicate key in format 2", `{"rules":{"format":2,"format":2,"conditions":[],"policies":[]}}`, "/rules/format unknown_key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, ps := authz.Prepare([]byte(tc.body), authz.DefaultMaxDocumentBytes)
			if got := strings.Join(pointersOf(ps), ","); got != tc.want {
				t.Errorf("problems = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPrepareRefsCarryBodyPointers(t *testing.T) {
	t.Parallel()
	g := "0b0b0b0b-0000-4000-8000-000000000001"
	p, ps := authz.Prepare([]byte(`{"rules":{"format":2,"conditions":[{"name":"c","expr":{"member_of":{"users":["`+g+`"]}}}],"policies":[{"name":"p","ops":["read"],"when":{"member_of":{"groups":["`+g+`"]}}}]},"masks":[{"field":"f","read":{"groups":["`+g+`"]},"write":{"groups":["`+g+`"]}}]}`), authz.DefaultMaxDocumentBytes)
	if len(ps) > 0 {
		t.Fatal(pointersOf(ps))
	}
	var got []string
	for _, r := range p.Refs {
		got = append(got, r.Pointer)
	}
	slices.Sort(got)
	want := []string{
		"/masks/0/read/groups/0", "/masks/0/write/groups/0",
		"/rules/conditions/0/expr/member_of/users/0", "/rules/policies/0/when/member_of/groups/0",
	}
	if !slices.Equal(got, want) {
		t.Errorf("refs = %q, want %q", got, want)
	}
}

func TestPrepareWarnsOnNotOverCallerLeaves(t *testing.T) {
	t.Parallel()
	pol := func(when string) string {
		return `{"format":2,"conditions":[{"name":"mine","expr":{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}},{"name":"plain","expr":{"cmp":{"column":"state","op":"eq","values":["new"]}}},{"name":"viamine","expr":{"ref":"mine"}},{"name":"team","expr":{"member_of":{"users":["0b0b0b0b-0000-4000-8000-000000000001"]}}}],"policies":[{"name":"p","ops":["read"],"when":` + when + `}]}`
	}
	for _, tc := range []struct{ name, when, want string }{
		{"not over self", `{"not":{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}}`, "/rules/policies/0/when/not"},
		{"not over shared", `{"all":[{"not":{"shared":true}},{"shared":true}]}`, "/rules/policies/0/when/all/0/not"},
		{"not over a ref to a caller condition", `{"not":{"ref":"mine"}}`, "/rules/policies/0/when/not"},
		{"not over a ref that reaches one", `{"not":{"ref":"viamine"}}`, "/rules/policies/0/when/not"},
		{"not over member_of", `{"not":{"member_of":{"groups":["0b0b0b0b-0000-4000-8000-000000000001"]}}}`, "/rules/policies/0/when/not"},
		{"not over a ref to a member_of condition", `{"not":{"ref":"team"}}`, "/rules/policies/0/when/not"},
		{"member_of outside a not", `{"member_of":{"users":["0b0b0b0b-0000-4000-8000-000000000001"]}}`, ""},
		{"not over plain data", `{"not":{"cmp":{"column":"state","op":"eq","values":["closed"]}}}`, ""},
		{"not over a plain ref", `{"not":{"ref":"plain"}}`, ""},
		{"caller leaf outside a not", `{"shared":true}`, ""},
		{"nested nots warn once", `{"not":{"not":{"shared":true}}}`, "/rules/policies/0/when/not"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, ps := authz.Prepare(body(pol(tc.when)), authz.DefaultMaxDocumentBytes)
			if len(ps) > 0 {
				t.Fatal(pointersOf(ps))
			}
			var got []string
			for _, w := range p.Warnings {
				if w.Code != authz.WarnNotOverCaller {
					t.Errorf("code = %q", w.Code)
				}
				got = append(got, w.Pointer)
			}
			var want []string
			if tc.want != "" {
				want = []string{tc.want}
			}
			if !slices.Equal(got, want) {
				t.Errorf("warnings = %q, want %q", got, want)
			}
		})
	}
}

// Whatever Prepare accepts, Decode reads back as the same document.
func TestPrepareOutputRoundTrips(t *testing.T) {
	t.Parallel()
	for _, f := range load[validFixture](t, "testdata/format2_valid.json") {
		p, ps := authz.Prepare(body(string(f.Doc)), authz.DefaultMaxDocumentBytes)
		if strings.Contains(string(f.Doc), `"under"`) {
			continue
		}
		if len(ps) > 0 {
			t.Fatalf("%s: %q", f.Name, pointersOf(ps))
		}
		d, err := authz.Decode(p.Rules)
		if err != nil {
			t.Fatalf("%s: %v", f.Name, err)
		}
		a, _ := json.Marshal(d)
		b, _ := json.Marshal(p.Document)
		if string(a) != string(b) {
			t.Errorf("%s: stored %s, document %s", f.Name, a, b)
		}
	}
}
