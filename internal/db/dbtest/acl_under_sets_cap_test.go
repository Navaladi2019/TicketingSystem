//go:build integration

package dbtest

import (
	"context"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// TICK-1059 (TODO-S31-01; SP-1d change 5 on TICK-946): acl_versions_rules_check
// refuses a format-2 policy that names more than authz.MaxUnderSets distinct
// under start sets U after its refs expand, as Prepare does since TICK-1034.

const underSetsCapMigration = "20261121161036_acl2_under_sets_cap.sql"

// underSetsCapFunctions are the functions underSetsCapMigration creates.
var underSetsCapFunctions = []string{
	"public.acl2_max_under_sets()", "public.acl2_under_key(jsonb)", "public.acl2_under_union(text,text)",
}

// beforeUnderSetsCap returns m without the grants on underSetsCapFunctions
// when fsys lacks underSetsCapMigration, so a run that stops at an earlier
// version grants only functions it has created.
func beforeUnderSetsCap(fsys fs.FS, m *grants.Manifest) *grants.Manifest {
	if _, err := fs.Stat(fsys, underSetsCapMigration); err == nil {
		return m
	}
	gm := *m
	gm.Roles = make(map[string]grants.Role, len(m.Roles))
	for name, r := range m.Roles {
		functions := make(map[string][]string, len(r.Functions))
		for fn, privs := range r.Functions {
			if !strings.HasPrefix(fn, "public.acl2_max_under_sets(") && !strings.HasPrefix(fn, "public.acl2_under_key(") &&
				!strings.HasPrefix(fn, "public.acl2_under_union(") {
				functions[fn] = privs
			}
		}
		r.Functions = functions
		gm.Roles[name] = r
	}
	return &gm
}

// unit is the n-th org-unit id of these tests.
func unit(n int) string { return fmt.Sprintf("0b0b0b0b-0000-4000-8000-%012d", 1059000+n) }

// underDoc is a format-2 document with one policy p whose when is any over
// whens, and one condition c with expr cond, which p refs when ref is set.
func underDoc(cond string, ref bool, whens ...string) string {
	if ref {
		whens = append(whens, `{"ref": "c"}`)
	}
	return `{"format": 2, "conditions": [{"name": "c", "expr": ` + cond + `}], "policies": [{"name": "p", "ops": ["read"], "when": {"any": [` +
		strings.Join(whens, ", ") + `]}}]}`
}

func under(ids ...string) string {
	if len(ids) == 0 {
		return `{"under": {"of": "caller"}}`
	}
	return `{"under": {"of": ["` + strings.Join(ids, `", "`) + `"]}}`
}

func TestTICK1059UnderSetsCap(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedACL(t, super, tenantA)
	app := asRole(t, "nava_app")

	// The cap is written once in the database and equals D4's.
	var dbCap int
	if err := super.QueryRow(context.Background(), `SELECT acl2_max_under_sets()`).Scan(&dbCap); err != nil {
		t.Fatal(err)
	}
	if dbCap != authz.MaxUnderSets {
		t.Fatalf("acl2_max_under_sets() = %d, authz.MaxUnderSets = %d; they must match", dbCap, authz.MaxUnderSets)
	}

	version := 1
	insert := func(conn *pgx.Conn, tenant, rules string) error {
		version++
		_, err := execIn(conn, tenant, insertACLShape, tenantA, version, rules, `[]`)
		return err
	}
	u1, u2, u3, u4 := unit(1), unit(2), unit(3), unit(4)
	cond := `{"all": [` + under(u1) + `, ` + under(u1, u2) + `]}`

	accepted := []struct{ name, doc string }{
		// caller, {u1}, {u1,u2} through the ref, {u3}: four. The same set
		// again in another order or in upper case is not a new U. (An id
		// repeated inside one leaf is refused by the route since TICK-1036,
		// under_duplicate_id; see the duplicate case below.)
		{"four distinct U through a ref, repeats folded", underDoc(cond, true,
			under(), under(u2, u1), under(strings.ToUpper(u1), u2), under(u3))},
		{"four distinct U written out", underDoc(under(u1), false, under(), under(u1), under(u2), under(u3, u4))},
		// Only policies are capped; an unused condition is not (D4).
		{"five U in a condition no policy refs", underDoc(`{"any": [`+under()+`, `+under(u1)+`, `+under(u2)+`, `+under(u3)+`, `+under(u4)+`]}`,
			false, under())},
	}
	for _, c := range accepted {
		seedACLIDs(t, super, tenantA, c.doc)
		if err := insert(app, tenantA, c.doc); err != nil {
			t.Errorf("%s: nava_app insert = %v, want accepted", c.name, err)
		}
	}

	refused := []struct{ name, doc string }{
		{"five distinct U through a ref", underDoc(cond, true, under(), under(u3), under(u4))},
		{"five distinct U written out", underDoc(under(u1), false, under(), under(u1), under(u2), under(u3), under(u4))},
	}
	for _, c := range refused {
		seedACLIDs(t, super, tenantA, c.doc)
		for _, as := range []struct {
			name   string
			conn   *pgx.Conn
			tenant string
		}{{"nava_app", app, tenantA}, {"superuser", super, ""}} {
			what := as.name + " inserts " + c.name
			pe := refusal(t, what, insert(as.conn, as.tenant, c.doc))
			if pe == nil {
				continue
			}
			if pe.ConstraintName != "acl_versions_rules_check" {
				t.Errorf("%s: constraint %q, want acl_versions_rules_check", what, pe.ConstraintName)
			}
			if d := detailOf(t, what, pe); d != (aclDetail{Column: "rules", Code: authz.RuleUnderSetsExceeded, Pointer: "/policies/0"}) {
				t.Errorf("%s: DETAIL %+v, want rules under_sets_exceeded at /policies/0", what, d)
			}
		}
	}

	// A2-8 (TICK-1036): an id listed twice in one leaf is refused by the
	// route only. The database CHECK still accepts it, and counts it as one
	// U; Prepare refuses it with under_duplicate_id at the repeat. The
	// agreement loop below therefore does not hold this document.
	dup := underDoc(under(), false, under(u1, u2, u1))
	seedACLIDs(t, super, tenantA, dup)
	if err := insert(app, tenantA, dup); err != nil {
		t.Errorf("a leaf listing u1 twice: nava_app insert = %v, want accepted (route-only rule)", err)
	}
	if _, ps := authz.Prepare([]byte(`{"rules": `+dup+`}`), authz.DefaultMaxDocumentBytes); len(ps) != 1 ||
		ps[0].Rule != authz.RuleUnderDuplicateID || ps[0].Pointer != "/rules/policies/0/when/any/0/under/of/2" {
		t.Errorf("a leaf listing u1 twice: Prepare problems %v, want only under_duplicate_id at /rules/policies/0/when/any/0/under/of/2", ps)
	}

	// Go and the database agree on the same documents.
	for _, c := range accepted {
		if _, ps := authz.Prepare([]byte(`{"rules": `+c.doc+`}`), authz.DefaultMaxDocumentBytes); len(ps) != 0 {
			t.Errorf("%s: Prepare problems %v, the database accepts it", c.name, ps)
		}
	}
	for _, c := range refused {
		_, ps := authz.Prepare([]byte(`{"rules": `+c.doc+`}`), authz.DefaultMaxDocumentBytes)
		if len(ps) != 1 || ps[0].Pointer != "/rules/policies/0" || ps[0].Rule != authz.RuleUnderSetsExceeded {
			t.Errorf("%s: Prepare problems %v, want only /rules/policies/0 under_sets_exceeded", c.name, ps)
		}
	}
}
