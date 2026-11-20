//go:build integration

// TICK-937 P11-05 (ADR-010 D5, D6) against a real PostgreSQL as nava_app,
// under RLS: for format-2 documents over every node kind, NOT over NULL
// columns, refs, folded caller-only leaves and voided policies,
// SELECT ... WHERE <authz.CompileRead> returns exactly the rows authz.Allows
// admits, for callers with and without an id and groups. ReadBranches' branches
// partition the same rows. The random agreement suite over format-2 trees is
// TICK-942 P11-10's.
package pipeline_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

func TestFormat2TreesAgreeInSQLAndGo(t *testing.T) {
	e := migrated(t)
	w := seedACLWorld(t, e)
	requester := `{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}`
	assignee := `{"cmp":{"column":"assignee_id","op":"eq","caller":"self"}}`
	myGroup := `{"cmp":{"column":"assignment_group_id","op":"in","caller":"my_groups"}}`
	open := `{"cmp":{"column":"state","op":"in","values":["open"]}}`
	inG2 := fmt.Sprintf(`{"cmp":{"column":"assignment_group_id","op":"eq","values":[%q]}}`, w.g2)
	byU2 := fmt.Sprintf(`{"cmp":{"column":"requester_id","op":"in","values":[%q]}}`, w.u2)
	memberG1 := fmt.Sprintf(`{"member_of":{"groups":[%q]}}`, w.g1)
	memberU2 := fmt.Sprintf(`{"member_of":{"users":[%q]}}`, w.u2)
	not := func(n string) string { return `{"not":` + n + `}` }
	all := func(ns ...string) string { return `{"all":[` + strings.Join(ns, ",") + `]}` }
	anyOf := func(ns ...string) string { return `{"any":[` + strings.Join(ns, ",") + `]}` }
	conds := `[{"name":"mine","expr":` + anyOf(requester, assignee) + `},
		{"name":"desk","expr":` + all(myGroup, not(open)) + `},
		{"name":"visible","expr":` + anyOf(`{"ref":"mine"}`, `{"ref":"desk"}`, `{"shared":true}`) + `}]`
	doc := func(whens ...string) *authz.Policy {
		t.Helper()
		ps := make([]string, len(whens))
		for i, wh := range whens {
			ps[i] = fmt.Sprintf(`{"name":"p%d","ops":["read"],"when":%s}`, i, wh)
		}
		d, err := authz.Decode([]byte(`{"format":2,"conditions":` + conds + `,"policies":[` + strings.Join(ps, ",") + `]}`))
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		return &authz.Policy{Version: 1, EntityType: aclKey, Document: d}
	}
	policies := map[string]*authz.Policy{
		"not requester (NULL is true)":     doc(not(requester)),
		"not assignee and not my group":    doc(all(not(assignee), not(myGroup))),
		"not shared":                       doc(not(`{"shared":true}`)),
		"not literal group":                doc(not(inG2)),
		"not literal requester, open":      doc(all(not(byU2), open)),
		"refs":                             doc(`{"ref":"visible"}`),
		"not a ref":                        doc(not(`{"ref":"visible"}`)),
		"desk or mine, two policies":       doc(`{"ref":"desk"}`, all(`{"ref":"mine"}`, open)),
		"member_of folds":                  doc(anyOf(memberG1, all(memberU2, not(open)))),
		"not member_of":                    doc(all(not(memberG1), `{"ref":"mine"}`)),
		"distributed all":                  doc(all(anyOf(requester, assignee, `{"shared":true}`, myGroup), not(open))),
		"voided under not, beside a grant": doc(not(`{"ref":"missing"}`), `{"ref":"mine"}`),
		"empty any from format 1": {Document: authz.ConvertFormat1([]authz.Rule{
			{ID: "nobody", Ops: []string{authz.OpRead}},
			{ID: "req", Ops: []string{authz.OpRead}, Who: authz.Subject{Requester: true}},
		})},
	}
	principals := map[string]identity.Principal{
		"u1 in g1":       {TenantID: w.f.id, PrincipalID: w.u1, Groups: []uuid.UUID{w.g1}},
		"u1 in no group": {TenantID: w.f.id, PrincipalID: w.u1},
		"u2 in g2, g3":   {TenantID: w.f.id, PrincipalID: w.u2, Groups: []uuid.UUID{w.g2, w.g3}},
		"no principal":   {TenantID: w.f.id, Groups: []uuid.UUID{w.g1}},
	}
	proper := 0
	for pn, pol := range policies {
		for qn, p := range principals {
			bySQL, byGo := w.visible(t, e, pol, p)
			if !slices.Equal(bySQL, byGo) {
				t.Errorf("policy %q, principal %q: SQL admits %d rows %v, Allows %d rows %v", pn, qn, len(bySQL), bySQL, len(byGo), byGo)
			}
			if len(byGo) > 0 && len(byGo) < len(w.records) {
				proper++
			}
			if byBranches := w.branchRows(t, e, pol, p); byBranches != nil && !slices.Equal(byBranches, bySQL) {
				t.Errorf("policy %q, principal %q: ReadBranches admit %v, CompileRead %v", pn, qn, byBranches, bySQL)
			}
		}
	}
	// The table must discriminate, or agreement proves nothing.
	if proper < 20 {
		t.Fatalf("only %d policy/principal pairs admit a proper subset of the rows", proper)
	}
}

// branchRows runs the ReadBranches branches as one UNION ALL and returns the rows of
// all of them, sorted; a row admitted by two branches fails the test. It
// returns nil when there are fewer than two branches (the list falls back to
// CompileRead).
func (w aclWorld) branchRows(t *testing.T, e env, pol *authz.Policy, p identity.Principal) []uuid.UUID {
	t.Helper()
	ctx := context.Background()
	branches, args := authz.ReadBranches(pol, p, "r", 2)
	if len(branches) < 2 {
		return nil
	}
	// One UNION ALL, as the list runs them: every branch shares the
	// arguments, and an earlier branch does not reference all of them.
	selects := make([]string, len(branches))
	for i, b := range branches {
		selects[i] = `SELECT r.id FROM records r JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
			WHERE r.tenant_id = $1 AND d.key = $2 AND ` + b
	}
	var out []uuid.UUID
	inTx(t, e, w.f.id, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, strings.Join(selects, " UNION ALL "), append([]any{w.f.id, aclKey}, args...)...)
		if err != nil {
			return fmt.Errorf("branches %v: %w", branches, err)
		}
		out, err = pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		return err
	})
	for i := 1; i < len(out); i++ {
		if slices.Contains(out[:i], out[i]) {
			t.Errorf("row %s is admitted by two branches", out[i])
		}
	}
	slices.SortFunc(out, compareUUID)
	if out == nil {
		out = []uuid.UUID{}
	}
	return out
}
