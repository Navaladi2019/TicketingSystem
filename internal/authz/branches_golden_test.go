package authz_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// goldenBranches is one ReadBranches call: its branches and its arguments,
// each argument as its Go type and value so a change of encoding shows.
type goldenBranches struct {
	Branches []string `json:"branches"`
	Args     []string `json:"args"`
}

// goldenPrincipals are callers chosen to fold every format-1 fixture each way:
// no id, an id in no listed group, each listed group, and the listed user.
func goldenPrincipals() map[string]identity.Principal {
	id := func(s string) uuid.UUID { return uuid.MustParse(s) }
	t := id("0a000000-0000-4000-8000-000000000001")
	c := id("0a000000-0000-4000-8000-0000000000c1")
	listed := id("44444444-4444-4444-8444-444444444444")
	return map[string]identity.Principal{
		"no_id_no_groups":    {TenantID: t},
		"id_no_groups":       {TenantID: t, PrincipalID: c},
		"admin_group":        {TenantID: t, PrincipalID: c, Groups: []uuid.UUID{id("11111111-1111-4111-8111-111111111111")}},
		"service_desk_group": {TenantID: t, PrincipalID: c, Groups: []uuid.UUID{id("22222222-2222-4222-8222-222222222222")}},
		"auditor_group":      {TenantID: t, PrincipalID: c, Groups: []uuid.UUID{id("33333333-3333-4333-8333-333333333333")}},
		"other_group":        {TenantID: t, PrincipalID: c, Groups: []uuid.UUID{id("66666666-6666-4666-8666-666666666666")}},
		"listed_user":        {TenantID: t, PrincipalID: listed},
		"listed_user_group":  {TenantID: t, PrincipalID: listed, Groups: []uuid.UUID{id("55555555-5555-4555-8555-555555555555")}},
	}
}

// Every stored format-1 document, the D-3-27 seed allow-all ACL among them,
// gives the same ReadBranches branches and bind arguments as main before S30
// (TICK-938 P11-06, ADR-010 D6, CI-6 as amended by ADR-010-E1), in every case,
// after main's golden is normalized by exactly the two rewrites E1 allows
// (e1Normalize). The golden file was written by this test run on main
// (2277d6b0) and must not be regenerated on the story branch to make a diff
// pass: a diff here is a format-1 regression.
func TestReadBranchesFormat1Golden(t *testing.T) {
	t.Parallel()
	inputs, err := filepath.Glob("testdata/format1/*.json")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]map[string]goldenBranches{}
	rules := map[string][]authz.Rule{}
	for _, in := range inputs {
		if strings.HasSuffix(in, ".golden.json") {
			continue
		}
		raw, err := os.ReadFile(in)
		if err != nil {
			t.Fatal(err)
		}
		pol := format1Policy(t, raw)
		var rs []authz.Rule
		if err := json.Unmarshal(raw, &rs); err != nil {
			t.Fatal(err)
		}
		rules[filepath.Base(in)] = rs
		byCaller := map[string]goldenBranches{}
		for name, p := range goldenPrincipals() {
			br, args := authz.ReadBranches(pol, p, "r", 2)
			g := goldenBranches{Branches: br, Args: make([]string, len(args))}
			for i, a := range args {
				g.Args[i] = fmt.Sprintf("%T %v", a, a)
			}
			byCaller[name] = g
		}
		got[filepath.Base(in)] = byCaller
	}
	if len(got) < 6 {
		t.Fatalf("found %d format-1 fixtures, want at least 6", len(got))
	}
	out, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	out = append(out, '\n')
	const golden = "testdata/format1/read_branches.golden.json"
	if os.Getenv("NAVA_UPDATE_BRANCH_GOLDEN") == "1" {
		// Only on main (see above): the goldens are main's output.
		if err := os.WriteFile(golden, out, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]map[string]goldenBranches
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if len(want) != len(got) {
		t.Fatalf("golden has %d fixtures, the test %d", len(want), len(got))
	}
	normalized := map[string]string{}
	for file, byCaller := range want {
		for caller, w := range byCaller {
			g, ok := got[file][caller]
			if !ok {
				t.Errorf("%s %s: in the golden, not computed", file, caller)
				continue
			}
			if len(g.Branches) != len(w.Branches) {
				t.Errorf("%s %s: %d branches, main had %d", file, caller, len(g.Branches), len(w.Branches))
				continue
			}
			n, applied := e1Normalize(t, w, rules[file], goldenPrincipals()[caller])
			if len(applied) > 0 {
				normalized[file+" "+caller] = applied
			}
			if !slices.Equal(g.Branches, n.Branches) || !slices.Equal(g.Args, n.Args) {
				t.Errorf("%s %s: SQL text or arguments differ from main after the ADR-010-E1/E3 rewrites\n got %q %q\nwant %q %q", file, caller, g.Branches, g.Args, n.Branches, n.Args)
			}
		}
	}
	// The rewrites touch exactly the cases ADR-010-E1 and E3 (E3-3) list, each
	// with exactly the rewrites listed for it: a rewrite that starts matching
	// elsewhere would hide a regression there.
	wantNormalized := map[string]string{"onboard_default_incident.json service_desk_group": "ac"}
	for _, c := range []string{"admin_group", "auditor_group", "id_no_groups", "other_group", "service_desk_group"} {
		wantNormalized["every_subject.json "+c] = "b"
	}
	for _, c := range []string{"listed_user", "listed_user_group"} {
		wantNormalized["every_subject.json "+c] = "ab"
	}
	for k, want := range wantNormalized {
		if got := normalized[k]; got != want {
			t.Errorf("%s: rewrites %q applied, ADR-010-E1/E3 list %q", k, got, want)
		}
		delete(normalized, k)
	}
	for k, got := range normalized {
		t.Errorf("%s: rewrites %q applied to a case ADR-010-E1/E3 do not list", k, got)
	}
}

// e1SelfAny is main's rendering of a uuid cmp against a one-element set.
var e1SelfAny = regexp.MustCompile(`\(r\.(\w+) IS NOT NULL AND r\.(\w+) = ANY\(\$(\d+)::uuid\[\]\)\)`)

// e1Normalize applies to main's golden w exactly the rewrites ADR-010-E1
// (E1-3) and ADR-010-E3 (E3-3) allow, for the fixture's rules and the caller
// p, and returns which applied, as a sorted subset of "abc":
//
//	(a) the constant true conjunct is removed: "(true AND " becomes "(";
//	(b) a where cmp with caller: self, which main rendered as
//	    "= ANY($n::uuid[])" bound to []string{caller}, becomes "= $n::uuid"
//	    bound to the caller's uuid.UUID;
//	(c) a where cmp with caller: my_groups for a caller with exactly one
//	    group g, which main rendered as "= ANY($n::uuid[])" bound to
//	    []string{g}, becomes "= $n::uuid" bound to g as a uuid.UUID.
//
// (b) applies only on a column the fixture compares with caller: self in a
// where, and only to the argument equal to the caller's id; (c) only on a
// column compared with caller: my_groups, only for a one-group caller, and
// only to the argument equal to that group. A column that also has a literal
// where condition holding that id is ambiguous and fails the test: E1-3
// forbids a blind rewrite of every one-element ANY, which would hide an "in"
// list regressing to a scalar.
func e1Normalize(t *testing.T, w goldenBranches, rules []authz.Rule, p identity.Principal) (goldenBranches, string) {
	t.Helper()
	caller := p.PrincipalID
	var group uuid.UUID
	if len(p.Groups) == 1 {
		group = p.Groups[0]
	}
	selfCols, groupCols, literal := map[string]bool{}, map[string]bool{}, map[string]uuid.UUID{}
	for _, r := range rules {
		for _, c := range r.Where {
			switch {
			case c.Caller == authz.CallerSelf:
				selfCols[c.Column] = true
			case c.Caller == authz.CallerMyGroups:
				groupCols[c.Column] = true
			case caller != uuid.Nil && slices.Contains(c.Values, caller.String()):
				literal[c.Column] = caller
			case group != uuid.Nil && slices.Contains(c.Values, group.String()):
				literal[c.Column] = group
			}
		}
	}
	for col := range selfCols {
		if literal[col] == caller && caller != uuid.Nil {
			t.Fatalf("ADR-010-E1 (b) is ambiguous: %s has both caller: self and a literal %s", col, caller)
		}
	}
	for col := range groupCols {
		if literal[col] == group && group != uuid.Nil {
			t.Fatalf("ADR-010-E3 (c) is ambiguous: %s has both caller: my_groups and a literal %s", col, group)
		}
	}
	out := goldenBranches{Branches: make([]string, len(w.Branches)), Args: slices.Clone(w.Args)}
	applied := map[byte]bool{}
	for i, b := range w.Branches {
		if strings.Contains(b, "(true AND ") {
			b = strings.ReplaceAll(b, "(true AND ", "(")
			applied['a'] = true
		}
		b = e1SelfAny.ReplaceAllStringFunc(b, func(m string) string {
			sub := e1SelfAny.FindStringSubmatch(m)
			col, n := sub[1], sub[3]
			at, err := strconv.Atoi(n)
			if err != nil || col != sub[2] {
				return m
			}
			at -= 3 // ReadBranches(..., argBase 2): $3 is Args[0]
			if at < 0 || at >= len(w.Args) {
				return m
			}
			var id uuid.UUID
			var rw byte
			switch {
			case selfCols[col] && caller != uuid.Nil:
				id, rw = caller, 'b'
			case groupCols[col] && group != uuid.Nil:
				id, rw = group, 'c'
			default:
				return m
			}
			if w.Args[at] != "[]string ["+id.String()+"]" {
				return m
			}
			out.Args[at] = "uuid.UUID " + id.String()
			applied[rw] = true
			return "(r." + col + " IS NOT NULL AND r." + col + " = $" + n + "::uuid)"
		})
		out.Branches[i] = b
	}
	var keys string
	for _, k := range "abc" {
		if applied[byte(k)] {
			keys += string(k)
		}
	}
	return out, keys
}

// format1Policy is a stored format-1 rules column as the read path loads it:
// decoded and converted to a format-2 document (ADR-010 D11). On main, where
// the goldens were written, it built Policy{Rules: rules}.
func format1Policy(t *testing.T, raw []byte) *authz.Policy {
	t.Helper()
	d, err := authz.Decode(raw)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	return &authz.Policy{Version: 1, EntityType: "golden", Document: d}
}
