package isolation

import (
	"slices"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// TestRegistryIsWellFormed: every path has a unique name, a covered ADR test
// id and exactly one call; SQL paths run as a manifest role that may connect
// to the platform database; only a pending path may be HTTP until T17 adds
// the HTTP runner.
func TestRegistryIsWellFormed(t *testing.T) {
	m, err := grants.Parse(db.Grants())
	if err != nil {
		t.Fatal(err)
	}
	cov := Coverage()
	seen := map[string]bool{}
	for _, p := range Paths() {
		if p.Name == "" || seen[p.Name] {
			t.Errorf("path %q: empty or duplicate name", p.Name)
		}
		seen[p.Name] = true
		if _, ok := cov[p.Case]; !ok {
			t.Errorf("path %q: case %q is not in Coverage", p.Name, p.Case)
		}
		calls := 0
		for _, set := range []bool{p.SQL != nil, p.HTTP != nil, p.Check != nil, p.DB != nil} {
			if set {
				calls++
			}
		}
		if calls != 1 {
			t.Errorf("path %q: %d calls, want exactly one of SQL, HTTP, Check, DB", p.Name, calls)
		}
		if p.SQL != nil {
			r, ok := m.Roles[p.Role]
			if !ok || !slices.Contains(r.Database, "CONNECT") {
				t.Errorf("path %q: role %q is not a manifest role with CONNECT", p.Name, p.Role)
			}
			if p.SQL.Expect != OwnRowsOnly && p.SQL.Expect != Denied {
				t.Errorf("path %q: no expectation", p.Name)
			}
		}
		if p.HTTP != nil && p.Pending == "" {
			t.Errorf("path %q: HTTP paths have no runner yet; T17 adds it with the first runnable one", p.Name)
		}
	}
}

// TestExemptionsAreAllowListed: an exemption names an ADR-001 A.3 policy, on
// its table, for its command and role, whose USING clause is true. Anything
// else would let the suite pass a cross-tenant read that no ADR accepted.
func TestExemptionsAreAllowListed(t *testing.T) {
	allow := catalog.AllowList()
	for _, e := range Exemptions() {
		p, ok := allow[e.Table+"."+e.Policy]
		if !ok {
			t.Errorf("%+v: %s.%s is not on the allow-list", e, e.Table, e.Policy)
			continue
		}
		if p.Cmd != e.Command || !slices.Contains(strings.Split(p.Roles, ","), e.Role) {
			t.Errorf("%+v: the allow-list has it FOR %s TO %s", e, p.Cmd, p.Roles)
		}
		if e.Command == "SELECT" && p.Qual != "true" {
			t.Errorf("%+v: USING (%s) is not true; the generic checks should see no row", e, p.Qual)
		}
	}
}

// TestCursorPathsHold runs the CI-8 cursor checks without a database, so
// ci-unit catches a cursor regression before the isolation stage.
func TestCursorPathsHold(t *testing.T) {
	for _, p := range Paths() {
		if p.Check == nil {
			continue
		}
		if err := p.Check(t.Context()); err != nil {
			t.Errorf("%s: %v", p.Name, err)
		}
	}
}
