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
	normalized := map[string]bool{}
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
			n, changed := e1Normalize(t, w, rules[file], goldenPrincipals()[caller].PrincipalID)
			if changed {
				normalized[file+" "+caller] = true
			}
			if !slices.Equal(g.Branches, n.Branches) || !slices.Equal(g.Args, n.Args) {
				t.Errorf("%s %s: SQL text or arguments differ from main after the ADR-010-E1 rewrites\n got %q %q\nwant %q %q", file, caller, g.Branches, g.Args, n.Branches, n.Args)
			}
		}
	}
	// The rewrites touch exactly the 8 cases ADR-010-E1 lists, and no other:
	// a rewrite that starts matching elsewhere would hide a regression there.
	wantNormalized := []string{"onboard_default_incident.json service_desk_group"}
	for _, c := range []string{"admin_group", "auditor_group", "id_no_groups", "listed_user", "listed_user_group", "other_group", "service_desk_group"} {
		wantNormalized = append(wantNormalized, "every_subject.json "+c)
	}
	for _, k := range wantNormalized {
		if !normalized[k] {
			t.Errorf("%s: ADR-010-E1 lists it, but no rewrite applied", k)
		}
		delete(normalized, k)
	}
	for k := range normalized {
		t.Errorf("%s: an ADR-010-E1 rewrite applied to a case E1 does not list", k)
	}
}

// e1SelfAny is main's rendering of a uuid cmp against a one-element set.
var e1SelfAny = regexp.MustCompile(`\(r\.(\w+) IS NOT NULL AND r\.(\w+) = ANY\(\$(\d+)::uuid\[\]\)\)`)

// e1Normalize applies to main's golden w exactly the two rewrites ADR-010-E1
// (E1-3) allows, for the fixture's rules and the caller's id, and reports
// whether either applied:
//
//	(a) the constant true conjunct is removed: "(true AND " becomes "(";
//	(b) a where cmp with caller: self, which main rendered as
//	    "= ANY($n::uuid[])" bound to []string{caller}, becomes "= $n::uuid"
//	    bound to the caller's uuid.UUID.
//
// (b) applies only on a column the fixture compares with caller: self in a
// where, and only to the argument equal to the caller's id. A column that
// also has a literal where condition holding the caller's id is ambiguous and
// fails the test: E1-3 forbids a blind rewrite of every one-element ANY, which
// would hide an "in" list regressing to a scalar.
func e1Normalize(t *testing.T, w goldenBranches, rules []authz.Rule, caller uuid.UUID) (goldenBranches, bool) {
	t.Helper()
	selfCols, literal := map[string]bool{}, map[string]bool{}
	for _, r := range rules {
		for _, c := range r.Where {
			switch {
			case c.Caller == authz.CallerSelf:
				selfCols[c.Column] = true
			case caller != uuid.Nil && slices.Contains(c.Values, caller.String()):
				literal[c.Column] = true
			}
		}
	}
	for col := range selfCols {
		if literal[col] {
			t.Fatalf("ADR-010-E1 (b) is ambiguous: %s has both caller: self and a literal %s", col, caller)
		}
	}
	out := goldenBranches{Branches: make([]string, len(w.Branches)), Args: slices.Clone(w.Args)}
	changed := false
	for i, b := range w.Branches {
		if strings.Contains(b, "(true AND ") {
			b = strings.ReplaceAll(b, "(true AND ", "(")
			changed = true
		}
		b = e1SelfAny.ReplaceAllStringFunc(b, func(m string) string {
			sub := e1SelfAny.FindStringSubmatch(m)
			col, n := sub[1], sub[3]
			at, err := strconv.Atoi(n)
			if err != nil || col != sub[2] || !selfCols[col] || caller == uuid.Nil {
				return m
			}
			at -= 3 // ReadBranches(..., argBase 2): $3 is Args[0]
			if at < 0 || at >= len(w.Args) || w.Args[at] != "[]string ["+caller.String()+"]" {
				return m
			}
			out.Args[at] = "uuid.UUID " + caller.String()
			changed = true
			return "(r." + col + " IS NOT NULL AND r." + col + " = $" + n + "::uuid)"
		})
		out.Branches[i] = b
	}
	return out, changed
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
