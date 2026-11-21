//go:build integration

// TICK-943 (S30 P11-11): adversarial isolation tests for composable access
// rules (ADR-010 I1, D4, D5b, D10). The attacker holds a valid admin token
// for one tenant and tries to make a format-2 document reach another tenant,
// or to keep access after a mistake voids a policy. Threat model and the
// finding table: docs/security/s30-composable-access-rules.md.

package server_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// s30Counts is every acl_versions row and acl_policy audit entry, both tenants.
func (e *etEnv) s30Counts(t *testing.T) (versions, entries int) {
	t.Helper()
	if err := e.super.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM acl_versions WHERE tenant_id IN ($1, $2)),
		(SELECT count(*) FROM audit_log WHERE tenant_id IN ($1, $2) AND subject_type = 'acl_policy')`, e.a.id, e.b.id).Scan(&versions, &entries); err != nil {
		t.Fatal(err)
	}
	return versions, entries
}

// s30Store publishes format-1 rules as tenant A's next version of job through
// the format-1 test seam (authz.PolicyStore.Publish): the only way a document
// the route refuses reaches storage is a row stored before TICK-939, and a
// stored format-1 row is exactly that.
func (e *etEnv) s30Store(t *testing.T, rules []authz.Rule) {
	t.Helper()
	if err := tenancy.InTenantTx(t.Context(), e.pool, e.a.id, func(tx pgx.Tx) error {
		_, err := acltest.PublishPolicy(t.Context(), tx, "job", rules, nil)
		return err
	}); err != nil {
		t.Fatalf("store format-1 rules: %v", err)
	}
}

// s30Broken is a format-1 rule the planner voids (child_count): 16 where
// conditions plus the subject make an all of 17 children. It names user-a, so
// if it ever granted, user-a would read every new record.
func (e *etEnv) s30Broken() authz.Rule {
	r := authz.Rule{ID: "broken", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Users: []uuid.UUID{e.a.user.PrincipalID}}}
	for range authz.MaxChildren {
		r.Where = append(r.Where, authz.Cond{Column: "state", Op: authz.CondIn, Values: []string{"new"}})
	}
	return r
}

// s30Key makes an Idempotency-Key of a test case name.
func s30Key(name string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return '-'
	}, name)
}

func s30Etag(v int64) string { return fmt.Sprintf(`W/"%d"`, v) }

// Attack 3 and the TICK-941 carry-over: a voided policy grants nothing on the
// list, the single read, PATCH (404, never 412 or 428) and the explainer, and
// the explainer reports it in invalid with its code and pointer. The decision
// comes from the valid policy alone.
//
// The route and the CHECK refuse every shape the planner voids, so the
// document cannot be published through PUT (asserted first); it is stored as
// the legacy format-1 row it can only be.
// covers: explainAccess 200
func TestS30VoidedPolicyGrantsNothingAnywhere(t *testing.T) {
	e := newPatchEnv(t)
	rec := e.mustCreateRec(t, "admin-a", "job", "s30-void", `{"short_description": "voided"}`)
	broken := e.s30Broken()
	valid := authz.Rule{ID: "valid", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Users: []uuid.UUID{e.a.admin.PrincipalID}}}

	// The route refuses the voidable rule: it never reaches storage this way.
	v0, n0 := e.s30Counts(t)
	where := strings.Repeat(`{"column": "state", "op": "in", "values": ["new"]}, `, authz.MaxChildren)
	put := e.access(t, http.MethodPut, accessPath, "admin-a", "s30-void-put", `W/"1"`,
		`{"rules": [{"id": "broken", "ops": ["read"], "who": {"users": [`+uuidList(e.a.user.PrincipalID)+`]}, "where": [`+strings.TrimSuffix(where, ", ")+`]}]}`)
	// Finding F3, fixed by TICK-996: Prepare applies the format-2 grammar to
	// the converted rule, so the child cap is a 422 at the rule, not a 500.
	if ps := pointerRules(t, put); put.Code != http.StatusUnprocessableEntity || !containsStr(ps, "/rules/0 child_count") {
		t.Fatalf("the route answered %d %s, want 422 child_count at /rules/0", put.Code, put.Body)
	}
	if v, n := e.s30Counts(t); v != v0 || n != n0 {
		t.Fatalf("a refused body wrote: versions %d -> %d, entries %d -> %d", v0, v, n0, n)
	}

	// One voided policy and one valid one.
	e.s30Store(t, []authz.Rule{broken, valid})
	missing := e.do(t, http.MethodGet, "/api/v1/records/job/"+uuid.NewString(), "user-a", "", "")
	wantProblem(t, missing, http.StatusNotFound, problem.NotFound)

	if p := e.mustList(t, "user-a", "job", ""); len(p.Items) != 0 || p.Total != 0 {
		t.Errorf("user-a lists %d records (total %d) under a voided policy", len(p.Items), p.Total)
	}
	if r := e.do(t, http.MethodGet, "/api/v1/records/job/"+rec.ID, "user-a", "", ""); r.Code != http.StatusNotFound || r.Body.String() != missing.Body.String() {
		t.Errorf("user-a reads the record = %d %s, want the unknown-record 404", r.Code, r.Body)
	}
	for name, ifMatch := range map[string]string{"current": s30Etag(rec.Version), "stale": s30Etag(rec.Version + 7), "none": ""} {
		r := e.patch(t, "user-a", "job", rec.ID, ifMatch, `{"note": "written through a voided policy"}`)
		if r.Code != http.StatusNotFound || r.Body.String() != missing.Body.String() {
			t.Errorf("PATCH with %s If-Match = %d %s, want the unknown-record 404 (never 412/428)", name, r.Code, r.Body)
		}
	}
	if r := e.do(t, http.MethodGet, "/api/v1/records/job/"+rec.ID, "admin-a", "", ""); r.Code != http.StatusOK || strings.Contains(r.Body.String(), "voided policy") {
		t.Errorf("admin-a (valid policy) reads = %d %s, want 200 and no write", r.Code, r.Body)
	}

	wantInvalid := func(t *testing.T, who string, x explanation) {
		t.Helper()
		if len(x.Invalid) != 1 || x.Invalid[0].Policy != "broken" || x.Invalid[0].Code != authz.RuleChildCount || x.Invalid[0].Pointer != "/policies/0" {
			t.Errorf("%s: invalid = %+v, want [broken child_count /policies/0]", who, x.Invalid)
		}
	}
	xu := decodeExplanation(t, e.explain(t, "admin-a", "job", rec.ID, e.a.user.PrincipalID.String()))
	if xu.Decision != "deny" || xu.Policy != nil || xu.RuleID != nil {
		t.Errorf("explain user-a = %+v, want deny: the only policy naming it is voided", xu)
	}
	wantInvalid(t, "user-a", xu)
	xa := decodeExplanation(t, e.explain(t, "admin-a", "job", rec.ID, e.a.admin.PrincipalID.String()))
	if xa.Decision != "allow" || xa.Policy == nil || *xa.Policy != "valid" {
		t.Errorf("explain admin-a = %+v, want allow by valid", xa)
	}
	wantInvalid(t, "admin-a", xa)

	// The only read policy is voided: deny for everyone, the admin included.
	e.s30Store(t, []authz.Rule{broken})
	for _, tok := range []string{"user-a", "admin-a"} {
		if r := e.do(t, http.MethodGet, "/api/v1/records/job/"+rec.ID, tok, "", ""); r.Code != http.StatusNotFound {
			t.Errorf("%s reads under a voided-only document = %d", tok, r.Code)
		}
		if p := e.mustList(t, tok, "job", ""); len(p.Items) != 0 {
			t.Errorf("%s lists %d records under a voided-only document", tok, len(p.Items))
		}
	}
	for _, target := range []uuid.UUID{e.a.user.PrincipalID, e.a.admin.PrincipalID} {
		x := decodeExplanation(t, e.explain(t, "admin-a", "job", rec.ID, target.String()))
		if x.Decision != "deny" || x.Policy != nil {
			t.Errorf("explain %s under a voided-only document = %+v, want deny", target, x)
		}
		wantInvalid(t, target.String(), x)
	}
}

// TICK-941 carry-over: a format-2 deny published through the route has a trace
// step with value false, and names no policy.
// covers: explainAccess 200
func TestS30Format2DenyTraceHasAFalseStep(t *testing.T) {
	e := newPatchEnv(t)
	rec := e.mustCreateRec(t, "user-a", "job", "s30-trace", `{"short_description": "new"}`)
	doc := `{"format": 2, "conditions": [{"name": "closed_only", "expr": {"cmp": {"column": "state", "op": "in", "values": ["closed"]}}}],
		"policies": [{"name": "closed_readers", "ops": ["read"], "when": {"all": [{"ref": "closed_only"}, {"member_of": {"users": [` + uuidList(e.a.user.PrincipalID) + `]}}]}}]}`
	if put := e.access(t, http.MethodPut, accessPath, "admin-a", "s30-trace-put", `W/"1"`, `{"rules": `+doc+`}`); put.Code != http.StatusOK {
		t.Fatalf("publish = %d: %s", put.Code, put.Body)
	}
	x := decodeExplanation(t, e.explain(t, "admin-a", "job", rec.ID, e.a.user.PrincipalID.String()))
	if x.Decision != "deny" || x.Policy != nil || len(x.Invalid) != 0 {
		t.Fatalf("explain = %+v, want a deny with no invalid policy", x)
	}
	var sawFalse bool
	for _, s := range x.Trace {
		if s.Condition == "closed_only" && s.Policy == "closed_readers" && !s.Value {
			sawFalse = true
		}
	}
	if !sawFalse {
		t.Errorf("trace = %+v, want closed_only false for closed_readers", x.Trace)
	}
	if r := e.do(t, http.MethodGet, "/api/v1/records/job/"+rec.ID, "user-a", "", ""); r.Code != http.StatusNotFound {
		t.Errorf("user-a reads = %d, the explainer said deny", r.Code)
	}
}

// Attack 2: a ref resolves only inside the document it is in. A name defined
// only in tenant B's document, or only in an older version of tenant A's,
// is unresolved_ref; the 422 is the same as for a name nobody defined, so it
// is no oracle for tenant B's condition names.
// covers: putEntityTypeAccess 200 422
func TestS30RefNeverResolvesAcrossTenantsOrVersions(t *testing.T) {
	e := newPatchEnv(t)
	bDoc := `{"format": 2, "conditions": [{"name": "b_only", "expr": {"member_of": {"groups": [` + uuidList(e.b.adminGroup) + `]}}}],
		"policies": [{"name": "b_admins", "ops": ["read"], "when": {"ref": "b_only"}}]}`
	if r := e.access(t, http.MethodPut, accessPath, "admin-b", "s30-ref-b", `W/"1"`, `{"rules": `+bDoc+`}`); r.Code != http.StatusOK {
		t.Fatalf("tenant B publish = %d: %s", r.Code, r.Body)
	}
	aDoc := `{"format": 2, "conditions": [{"name": "old", "expr": {"shared": true}}], "policies": [{"name": "p", "ops": ["read"], "when": {"ref": "old"}}]}`
	if r := e.access(t, http.MethodPut, accessPath, "admin-a", "s30-ref-a2", `W/"1"`, `{"rules": `+aDoc+`}`); r.Code != http.StatusOK {
		t.Fatalf("tenant A v2 = %d: %s", r.Code, r.Body)
	}
	refTo := func(name string) string {
		return `{"rules": {"format": 2, "conditions": [], "policies": [{"name": "p", "ops": ["read"], "when": {"ref": "` + name + `"}}]}}`
	}
	v0, n0 := e.s30Counts(t)
	none := e.access(t, http.MethodPut, accessPath, "admin-a", "s30-ref-none", `W/"2"`, refTo("never_defined"))
	wantProblem(t, none, http.StatusUnprocessableEntity, problem.InvalidAccessPolicy)
	for name, r := range map[string]string{"tenant B's condition": "b_only", "an older version's condition": "old"} {
		got := e.access(t, http.MethodPut, accessPath, "admin-a", "s30-ref-"+r, `W/"2"`, refTo(r))
		wantProblem(t, got, http.StatusUnprocessableEntity, problem.InvalidAccessPolicy)
		if ps := pointerRules(t, got); !containsStr(ps, "/rules/policies/0/when/ref unresolved_ref") {
			t.Errorf("%s: errors %v, want unresolved_ref", name, ps)
		}
		if sameProblem(t, got) != sameProblem(t, none) {
			t.Errorf("%s: 422 %s differs from a never-defined name %s", name, got.Body, none.Body)
		}
	}
	// Through a condition, too.
	viaCond := `{"rules": {"format": 2, "conditions": [{"name": "c", "expr": {"ref": "b_only"}}], "policies": [{"name": "p", "ops": ["read"], "when": {"ref": "c"}}]}}`
	got := e.access(t, http.MethodPut, accessPath, "admin-a", "s30-ref-cond", `W/"2"`, viaCond)
	if ps := pointerRules(t, got); got.Code != http.StatusUnprocessableEntity || !containsStr(ps, "/rules/conditions/0/expr/ref unresolved_ref") {
		t.Errorf("condition ref to tenant B's name = %d %v", got.Code, ps)
	}
	if v, n := e.s30Counts(t); v != v0 || n != n0 {
		t.Errorf("a refused body wrote: versions %d -> %d, entries %d -> %d", v0, v, n0, n)
	}
}

// Attack 4: with a format-2 policy in force in both tenants, tenant A's
// explainer cannot be pointed at tenant B's record, principal or group, and
// never shows tenant B's policy or condition names.
// covers: explainAccess 200 404
func TestS30ExplainerStaysInItsTenantUnderFormat2(t *testing.T) {
	e := newPatchEnv(t)
	recA := e.mustCreateRec(t, "user-a", "job", "s30-x-a", `{"short_description": "a"}`)
	recB := e.mustCreateRec(t, "user-b", "job", "s30-x-b", `{"short_description": "b"}`)
	bDoc := `{"format": 2, "conditions": [{"name": "b_secret_cond", "expr": {"member_of": {"users": [` + uuidList(e.b.user.PrincipalID) + `]}}}],
		"policies": [{"name": "b_secret_policy", "ops": ["read"], "when": {"ref": "b_secret_cond"}}]}`
	if r := e.access(t, http.MethodPut, accessPath, "admin-b", "s30-x-bput", `W/"1"`, `{"rules": `+bDoc+`}`); r.Code != http.StatusOK {
		t.Fatalf("tenant B publish = %d: %s", r.Code, r.Body)
	}
	if r := e.access(t, http.MethodPut, accessPath, "admin-a", "s30-x-aput", `W/"1"`, `{"rules": `+format2Doc(e.a.adminGroup)+`}`); r.Code != http.StatusOK {
		t.Fatalf("tenant A publish = %d: %s", r.Code, r.Body)
	}
	missing := e.explain(t, "admin-a", "job", uuid.NewString(), e.a.user.PrincipalID.String())
	wantProblem(t, missing, http.StatusNotFound, problem.NotFound)
	for name, r := range map[string]struct{ rec, user string }{
		"B's record, A's user":   {recB.ID, e.a.user.PrincipalID.String()},
		"B's record, B's user":   {recB.ID, e.b.user.PrincipalID.String()},
		"A's record, B's user":   {recA.ID, e.b.user.PrincipalID.String()},
		"A's record, B's admin":  {recA.ID, e.b.admin.PrincipalID.String()},
		"A's record, B's group":  {recA.ID, e.b.adminGroup.String()},
		"B's record id as user":  {recA.ID, recB.ID},
		"A's group id as record": {e.a.adminGroup.String(), e.a.user.PrincipalID.String()},
	} {
		got := e.explain(t, "admin-a", "job", r.rec, r.user)
		if got.Code != http.StatusNotFound || got.Body.String() != missing.Body.String() {
			t.Errorf("%s = %d %s, want the unknown-record 404", name, got.Code, got.Body)
		}
	}
	ok := e.explain(t, "admin-a", "job", recA.ID, e.a.admin.PrincipalID.String())
	if ok.Code != http.StatusOK {
		t.Fatalf("own explain = %d: %s", ok.Code, ok.Body)
	}
	for _, leak := range []string{"b_secret_cond", "b_secret_policy", e.b.user.PrincipalID.String(), recB.ID} {
		if strings.Contains(ok.Body.String(), leak) {
			t.Errorf("tenant A's explanation carries %q: %s", leak, ok.Body)
		}
	}
}

// Attack 5: GET and PUT of a key only tenant A has, as tenant B's admin, is
// the unknown-key 404, byte for byte, whatever If-Match says: never 428 (no
// If-Match) or 412 (a wrong one, whose ETag would reveal tenant A's
// version). Nothing is written in either tenant. Timing: both keys take the
// same path, one lookup of (tenant_id, key) that finds nothing; there is no
// second query to time.
// covers: getEntityTypeAccess 404
// covers: putEntityTypeAccess 404
func TestS30AnotherTenantsKeyIsTheUnknownKeyForGetAndPut(t *testing.T) {
	e := newPatchEnv(t)
	e.noTestACL = true
	if r := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "s30-k-only-a", etBody("widget", "WGT")); r.Code != http.StatusCreated {
		t.Fatalf("create widget = %d: %s", r.Code, r.Body)
	}
	if r := e.access(t, http.MethodPut, "/api/v1/entity-types/widget/access", "admin-a", "s30-k-a", `W/"0"`, `{"rules": `+format2Doc(e.a.adminGroup)+`}`); r.Code != http.StatusOK {
		t.Fatalf("tenant A publish = %d: %s", r.Code, r.Body)
	}
	const other, unknown = "/api/v1/entity-types/widget/access", "/api/v1/entity-types/nothing/access"
	v0, n0 := e.s30Counts(t)
	body := `{"rules": ` + format2Doc(e.b.adminGroup) + `}`
	for _, tc := range []struct {
		name, method, ifMatch string
	}{
		{"GET", http.MethodGet, ""},
		{"PUT no If-Match", http.MethodPut, ""},
		{"PUT tenant A's current version", http.MethodPut, `W/"1"`},
		{"PUT a stale version", http.MethodPut, `W/"0"`},
		{"PUT *", http.MethodPut, "*"},
	} {
		b := body
		if tc.method == http.MethodGet {
			b = ""
		}
		got := e.access(t, tc.method, other, "admin-b", s30Key("s30-k-o-"+tc.name), tc.ifMatch, b)
		want := e.access(t, tc.method, unknown, "admin-b", s30Key("s30-k-u-"+tc.name), tc.ifMatch, b)
		wantProblem(t, got, http.StatusNotFound, problem.NotFound)
		if got.Code != want.Code || sameProblem(t, got) != sameProblem(t, want) || got.Header().Get("ETag") != "" {
			t.Errorf("%s: another tenant's key = %d %s (ETag %q), unknown key = %d %s", tc.name, got.Code, got.Body, got.Header().Get("ETag"), want.Code, want.Body)
		}
	}
	if v, n := e.s30Counts(t); v != v0 || n != n0 {
		t.Errorf("tenant B's requests wrote: versions %d -> %d, entries %d -> %d", v0, v, n0, n)
	}
	// A non-admin of tenant B is 403 for both keys alike: the role check comes
	// before the key lookup.
	if a, b := e.access(t, http.MethodGet, other, "user-b", "", "", ""), e.access(t, http.MethodGet, unknown, "user-b", "", "", ""); a.Code != http.StatusForbidden || sameProblem(t, a) != sameProblem(t, b) {
		t.Errorf("non-admin: other %d %s, unknown %d %s", a.Code, a.Body, b.Code, b.Body)
	}
}

// Attack 6: masks are format-independent. The same masks under format-1 rules
// and under the equivalent format-2 document hide the same fields from the
// same users, on the record read and in the explainer; a format-2 rules value
// cannot carry masks of its own.
// covers: putEntityTypeAccess 200 422
func TestS30MasksBehaveTheSameUnderEitherFormat(t *testing.T) {
	e := newPatchEnv(t)
	rec := e.mustCreateRec(t, "admin-a", "job", "s30-mask", `{"short_description": "m", "note": "admins only"}`)
	masks := `[{"field": "note", "read": {"groups": [` + uuidList(e.a.adminGroup) + `]}, "write": {"groups": [` + uuidList(e.a.adminGroup) + `]}}]`
	users := uuidList(e.a.admin.PrincipalID, e.a.user.PrincipalID)
	f1 := `{"rules": [{"id": "desk", "ops": ["read", "write"], "who": {"users": [` + users + `]}}], "masks": ` + masks + `}`
	f2 := `{"rules": {"format": 2, "conditions": [], "policies": [{"name": "desk", "ops": ["read", "write"], "when": {"member_of": {"users": [` + users + `]}}}]}, "masks": ` + masks + `}`
	type view struct{ userBody, adminBody, userMasked string }
	look := func(t *testing.T) view {
		t.Helper()
		u := e.do(t, http.MethodGet, "/api/v1/records/job/"+rec.ID, "user-a", "", "")
		a := e.do(t, http.MethodGet, "/api/v1/records/job/"+rec.ID, "admin-a", "", "")
		if u.Code != http.StatusOK || a.Code != http.StatusOK {
			t.Fatalf("reads = %d, %d", u.Code, a.Code)
		}
		x := decodeExplanation(t, e.explain(t, "admin-a", "job", rec.ID, e.a.user.PrincipalID.String()))
		return view{u.Body.String(), a.Body.String(), strings.Join(x.MaskedFields, ",")}
	}
	if r := e.access(t, http.MethodPut, accessPath, "admin-a", "s30-mask-f1", `W/"1"`, f1); r.Code != http.StatusOK {
		t.Fatalf("format 1 = %d: %s", r.Code, r.Body)
	}
	v1 := look(t)
	if r := e.access(t, http.MethodPut, accessPath, "admin-a", "s30-mask-f2", `W/"2"`, f2); r.Code != http.StatusOK {
		t.Fatalf("format 2 = %d: %s", r.Code, r.Body)
	}
	v2 := look(t)
	if v1 != v2 {
		t.Errorf("masks differ by format:\nformat 1 %+v\nformat 2 %+v", v1, v2)
	}
	if strings.Contains(v2.userBody, "admins only") || !strings.Contains(v2.adminBody, "admins only") || v2.userMasked != "note" {
		t.Errorf("note is not masked from user-a only: %+v", v2)
	}
	// Masks inside the format-2 rules value are refused, not merged.
	v0, n0 := e.s30Counts(t)
	inside := `{"rules": {"format": 2, "conditions": [], "masks": ` + masks + `, "policies": []}}`
	got := e.access(t, http.MethodPut, accessPath, "admin-a", "s30-mask-inside", `W/"3"`, inside)
	if ps := pointerRules(t, got); got.Code != http.StatusUnprocessableEntity || !containsStr(ps, "/rules/masks unknown_key") {
		t.Errorf("masks inside rules = %d %v, want 422 unknown_key", got.Code, ps)
	}
	if v, n := e.s30Counts(t); v != v0 || n != n0 {
		t.Errorf("a refused body wrote")
	}
}

// s30InsertRaw inserts rules as tenant's next version of job as nava_app,
// past the route, with the audit entry the commit check needs: the one other
// writer nava_app has (a direct INSERT).
func (e *etEnv) s30InsertRaw(t *testing.T, tenant uuid.UUID, rules string) error {
	t.Helper()
	return tenancy.InTenantTx(t.Context(), e.pool, tenant, func(tx pgx.Tx) error {
		var def uuid.UUID
		var have int
		if err := tx.QueryRow(t.Context(), `SELECT d.id, coalesce(max(a.version), 0) FROM entity_definitions d
			LEFT JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
			WHERE d.tenant_id = $1 AND d.key = 'job' GROUP BY d.id`, tenant).Scan(&def, &have); err != nil {
			return err
		}
		if _, err := tx.Exec(t.Context(), `INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks)
			VALUES ($1, $2, $3, $4::text::jsonb, '[]')`, tenant, def, have+1, rules); err != nil {
			return err
		}
		sum := sha256.Sum256([]byte(rules))
		before, after := int64(have), int64(have+1)
		sys := "s30-adversarial"
		_, err := audit.Append(t.Context(), tx, audit.Entry{Domain: audit.DomainConfig, Action: audit.ActionPublish,
			SubjectType: audit.SubjectACLPolicy, SubjectID: def, VersionBefore: &before, VersionAfter: &after,
			Source: audit.SourceSystem, Actor: audit.Actor{System: &sys}, ContentSHA256: &sum})
		return err
	})
}

// S31 (TICK-950) makes `under` available. The route checks the org-unit ids
// of under {"of": [ids]} against the tenant's live units (NotInTenant, the
// TICK-939 check extended): tenant B's unit, a tombstoned unit of tenant A and
// an id no tenant has are each 422 foreign_id at the id's pointer, one body
// for all three, the id not echoed, and nothing is stored. Past the route
// (finding F1) the tenant-id trigger refuses tenant B's unit with 23514
// foreign_id (TICK-947, AR-S30-01 closed), so the API never falls through to
// that backstop. Tenant A's own live unit publishes, and grants exactly the
// records in its live subtree on every read path: list, read, PATCH and the
// explainer. A record in a tombstoned unit below it is outside.
// covers: putEntityTypeAccess 422
func TestS31UnderChecksUnitsAtTheRouteAndGrantsBySubtree(t *testing.T) {
	e := newPatchEnv(t)
	in := e.mustCreateRec(t, "user-a", "job", "s31-under-in", `{"short_description": "in"}`)
	out := e.mustCreateRec(t, "user-a", "job", "s31-under-out", `{"short_description": "out"}`)
	none := e.mustCreateRec(t, "user-a", "job", "s31-under-none", `{"short_description": "none"}`)
	ouB, ouA, ouA1, ouDead := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for _, u := range []struct {
		tenant, id uuid.UUID
		parent     any
		key        string
		dead       bool
	}{
		{e.b.id, ouB, nil, "s31-ou-b", false},
		{e.a.id, ouA, nil, "s31-ou-a", false},
		{e.a.id, ouA1, ouA, "s31-ou-a1", false},
		{e.a.id, ouDead, ouA, "s31-ou-dead", true},
	} {
		if _, err := e.super.Exec(t.Context(), `INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name, deleted_at)
			VALUES ($1, $2, $3, $4, $4, CASE WHEN $5::boolean THEN now() END)`, u.tenant, u.id, u.parent, u.key, u.dead); err != nil {
			t.Fatal(err)
		}
	}
	rebuildOrgUnitClosure(t, e.super)
	under := func(of string) string {
		return `{"format": 2, "conditions": [], "policies": [{"name": "unit", "ops": ["read", "write"], "when": {"under": {"of": ` + of + `}}}]}`
	}
	v0, n0 := e.s30Counts(t)
	var bodies []string
	for name, id := range map[string]uuid.UUID{"tenant B's unit": ouB, "a tombstoned unit of A": ouDead, "no tenant's unit": uuid.New()} {
		got := e.access(t, http.MethodPut, accessPath, "admin-a", s31Key("s31-under-"+name), `W/"1"`,
			`{"rules": `+under(`["`+ouA.String()+`", "`+id.String()+`"]`)+`}`)
		wantProblem(t, got, http.StatusUnprocessableEntity, problem.InvalidAccessPolicy)
		if ps := pointerRules(t, got); !slices.Equal(ps, []string{"/rules/policies/0/when/under/of/1 foreign_id"}) {
			t.Errorf("route, under of %s = %v, want only /rules/policies/0/when/under/of/1 foreign_id", name, ps)
		}
		if strings.Contains(got.Body.String(), id.String()) {
			t.Errorf("the 422 for %s repeats the org-unit id", name)
		}
		bodies = append(bodies, sameProblem(t, got))
	}
	if bodies[0] != bodies[1] || bodies[1] != bodies[2] {
		t.Errorf("the 422 bodies differ, an existence oracle: %q", bodies)
	}
	if v, n := e.s30Counts(t); v != v0 || n != n0 {
		t.Fatalf("a refused body wrote")
	}
	// Past the route: the trigger refuses tenant B's unit too.
	err := e.s30InsertRaw(t, e.a.id, under(`["`+ouB.String()+`"]`))
	if pe := (*pgconn.PgError)(nil); !errors.As(err, &pe) || pe.Code != "23514" || !strings.Contains(pe.Detail, `"foreign_id"`) {
		t.Fatalf("direct insert of under naming tenant B's unit = %v, want 23514 foreign_id", err)
	}

	// Tenant A's live unit publishes. The records: in is in A1 (below A),
	// out in the tombstoned unit below A, none has no unit. The write
	// pipeline sets org_unit_id from TICK-949; here it is set directly.
	if got := e.access(t, http.MethodPut, accessPath, "admin-a", s31Key("s31-under-own"), `W/"1"`, `{"rules": `+under(`["`+ouA.String()+`"]`)+`}`); got.Code != http.StatusOK {
		t.Fatalf("publish under of A's live unit = %d %s", got.Code, got.Body)
	}
	tx, err := e.super.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(t.Context()) }()
	if _, err := tx.Exec(t.Context(), "SET LOCAL session_replication_role = replica"); err != nil {
		t.Fatal(err)
	}
	for id, unit := range map[string]uuid.UUID{in.ID: ouA1, out.ID: ouDead} {
		if _, err := tx.Exec(t.Context(), `UPDATE records SET org_unit_id = $3 WHERE tenant_id = $1 AND id = $2`, e.a.id, id, unit); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	p := e.mustList(t, "user-a", "job", "")
	if len(p.Items) != 1 || p.Items[0].ID != in.ID {
		t.Errorf("user-a lists %d records, want only the one in A's live subtree", len(p.Items))
	}
	if r := e.do(t, http.MethodGet, "/api/v1/records/job/"+in.ID, "user-a", "", ""); r.Code != http.StatusOK {
		t.Errorf("user-a reads the record in A1 = %d, want 200", r.Code)
	}
	for name, rec := range map[string]recBody{"in a tombstoned unit": out, "with no unit": none} {
		if r := e.do(t, http.MethodGet, "/api/v1/records/job/"+rec.ID, "user-a", "", ""); r.Code != http.StatusNotFound {
			t.Errorf("user-a reads the record %s = %d, want 404", name, r.Code)
		}
		if r := e.patch(t, "user-a", "job", rec.ID, s30Etag(rec.Version), `{"note": "x"}`); r.Code != http.StatusNotFound {
			t.Errorf("user-a patches the record %s = %d, want 404", name, r.Code)
		}
	}
	x := decodeExplanation(t, e.explain(t, "admin-a", "job", in.ID, e.a.user.PrincipalID.String()))
	if x.Decision != "allow" || x.Policy == nil || *x.Policy != "unit" || len(x.Invalid) != 0 {
		t.Errorf("explain the record in A1 = %+v, want allow by policy unit", x)
	}
	if x := decodeExplanation(t, e.explain(t, "admin-a", "job", out.ID, e.a.user.PrincipalID.String())); x.Decision != "deny" || len(x.Invalid) != 0 {
		t.Errorf("explain the record in the tombstoned unit = %+v, want deny with no invalid policy", x)
	}
}

// SP-1d change 5 (TICK-1034): a policy naming 5 distinct under U is 422
// under_sets_exceeded at the policy's pointer, and nothing is stored; 4
// publishes. Repeated U (the same ids reordered) count once.
// covers: putEntityTypeAccess 422
func TestS31UnderAtMostFourDistinctSetsPerPolicy(t *testing.T) {
	e := newPatchEnv(t)
	units := make([]uuid.UUID, 5)
	for i := range units {
		units[i] = uuid.New()
		if _, err := e.super.Exec(t.Context(), `INSERT INTO org_units (tenant_id, id, keycloak_group_id, name) VALUES ($1, $2, $3, $3)`,
			e.a.id, units[i], fmt.Sprintf("s31-sets-%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	of := func(ids ...uuid.UUID) string {
		parts := make([]string, len(ids))
		for i, id := range ids {
			parts[i] = `"` + id.String() + `"`
		}
		return `{"under": {"of": [` + strings.Join(parts, ", ") + `]}}`
	}
	doc := func(leaves ...string) string {
		return `{"rules": {"format": 2, "conditions": [], "policies": [` +
			`{"name": "open", "ops": ["read"], "when": {"cmp": {"column": "state", "op": "eq", "values": ["new"]}}},` +
			`{"name": "regions", "ops": ["read"], "when": {"any": [` + strings.Join(leaves, ", ") + `]}}]}}`
	}
	v0, n0 := e.s30Counts(t)
	five := doc(of(units[0]), of(units[1]), of(units[2]), of(units[3]), `{"under": {"of": "caller"}}`)
	got := e.access(t, http.MethodPut, accessPath, "admin-a", s31Key("s31-sets-five"), `W/"1"`, five)
	wantProblem(t, got, http.StatusUnprocessableEntity, problem.InvalidAccessPolicy)
	if ps := pointerRules(t, got); !slices.Equal(ps, []string{"/rules/policies/1 under_sets_exceeded"}) {
		t.Errorf("five distinct U = %v, want only /rules/policies/1 under_sets_exceeded", ps)
	}
	if v, n := e.s30Counts(t); v != v0 || n != n0 {
		t.Fatalf("a refused body wrote")
	}
	four := doc(of(units[0], units[1]), of(units[1], units[0]), of(units[2]), of(units[3]), `{"under": {"of": "caller"}}`)
	if got := e.access(t, http.MethodPut, accessPath, "admin-a", s31Key("s31-sets-four"), `W/"1"`, four); got.Code != http.StatusOK {
		t.Fatalf("four distinct U (one repeated) = %d %s, want 200", got.Code, got.Body)
	}
}

// SP-1e (TICK-1034): an under leaf that lists a unit and one of its
// ancestors, in the tenant's closure, is 422 under_nested at the leaf's
// .../under/of pointer, the ids not echoed, and nothing is stored. Two
// sibling units publish.
// covers: putEntityTypeAccess 422
func TestS31UnderRefusesNestedListedUnits(t *testing.T) {
	e := newPatchEnv(t)
	a, a1, b := uuid.New(), uuid.New(), uuid.New()
	for _, u := range []struct {
		id     uuid.UUID
		parent any
		key    string
	}{{a, nil, "s31-nest-a"}, {a1, a, "s31-nest-a1"}, {b, nil, "s31-nest-b"}} {
		if _, err := e.super.Exec(t.Context(), `INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name) VALUES ($1, $2, $3, $4, $4)`,
			e.a.id, u.id, u.parent, u.key); err != nil {
			t.Fatal(err)
		}
	}
	rebuildOrgUnitClosure(t, e.super)
	doc := func(ids ...uuid.UUID) string {
		parts := make([]string, len(ids))
		for i, id := range ids {
			parts[i] = `"` + id.String() + `"`
		}
		return `{"rules": {"format": 2, "conditions": [], "policies": [` +
			`{"name": "open", "ops": ["read"], "when": {"cmp": {"column": "state", "op": "eq", "values": ["new"]}}},` +
			`{"name": "region", "ops": ["read"], "when": {"any": [{"shared": true}, {"under": {"of": [` + strings.Join(parts, ", ") + `]}}]}}]}}`
	}
	v0, n0 := e.s30Counts(t)
	for name, ids := range map[string][]uuid.UUID{"ancestor first": {a, a1}, "descendant first": {a1, a}, "beside a sibling": {b, a1, a}} {
		got := e.access(t, http.MethodPut, accessPath, "admin-a", s31Key("s31-nest-"+name), `W/"1"`, doc(ids...))
		wantProblem(t, got, http.StatusUnprocessableEntity, problem.InvalidAccessPolicy)
		if ps := pointerRules(t, got); !slices.Equal(ps, []string{"/rules/policies/1/when/any/1/under/of under_nested"}) {
			t.Errorf("%s: problems = %v, want only /rules/policies/1/when/any/1/under/of under_nested", name, ps)
		}
		for _, id := range ids {
			if strings.Contains(got.Body.String(), id.String()) {
				t.Errorf("%s: the 422 repeats an org-unit id", name)
			}
		}
	}
	if v, n := e.s30Counts(t); v != v0 || n != n0 {
		t.Fatalf("a refused body wrote")
	}
	if got := e.access(t, http.MethodPut, accessPath, "admin-a", s31Key("s31-nest-siblings"), `W/"1"`, doc(a, b)); got.Code != http.StatusOK {
		t.Fatalf("two sibling units = %d %s, want 200", got.Code, got.Body)
	}
}

// Attack (TICK-1034, the SP-1e delta): tenant A's admin lists two units of
// tenant B, one below the other in B's closure, in one under leaf. The
// nested check filters the closure by the transaction's tenant, so B's tree
// is never read for A: the answer is foreign_id at each id's pointer, never
// under_nested, and the body is the same as for two ids no tenant has. An
// under_nested here would be an oracle for tenant B's org structure.
// covers: putEntityTypeAccess 422
func TestS31NestedCheckNeverReadsAnotherTenantsTree(t *testing.T) {
	e := newPatchEnv(t)
	p, p1 := uuid.New(), uuid.New()
	for _, u := range []struct {
		id     uuid.UUID
		parent any
		key    string
	}{{p, nil, "s31-xnest-p"}, {p1, p, "s31-xnest-p1"}} {
		if _, err := e.super.Exec(t.Context(), `INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name) VALUES ($1, $2, $3, $4, $4)`,
			e.b.id, u.id, u.parent, u.key); err != nil {
			t.Fatal(err)
		}
	}
	rebuildOrgUnitClosure(t, e.super)
	doc := func(ids ...uuid.UUID) string {
		parts := make([]string, len(ids))
		for i, id := range ids {
			parts[i] = `"` + id.String() + `"`
		}
		return `{"rules": {"format": 2, "conditions": [], "policies": [` +
			`{"name": "region", "ops": ["read"], "when": {"under": {"of": [` + strings.Join(parts, ", ") + `]}}}]}}`
	}
	v0, n0 := e.s30Counts(t)
	want := []string{"/rules/policies/0/when/under/of/0 foreign_id", "/rules/policies/0/when/under/of/1 foreign_id"}
	var bodies []string
	for name, ids := range map[string][]uuid.UUID{
		"B's ancestor and descendant": {p, p1},
		"no tenant's ids":             {uuid.New(), uuid.New()},
	} {
		got := e.access(t, http.MethodPut, accessPath, "admin-a", s31Key("s31-xnest-"+name), `W/"1"`, doc(ids...))
		wantProblem(t, got, http.StatusUnprocessableEntity, problem.InvalidAccessPolicy)
		ps := pointerRules(t, got)
		slices.Sort(ps)
		if !slices.Equal(ps, want) {
			t.Errorf("%s: problems = %v, want %v (no under_nested)", name, ps, want)
		}
		for _, id := range ids {
			if strings.Contains(got.Body.String(), id.String()) {
				t.Errorf("%s: the 422 repeats an org-unit id", name)
			}
		}
		bodies = append(bodies, sameProblem(t, got))
	}
	if bodies[0] != bodies[1] {
		t.Errorf("tenant B's nested units and unknown ids give different bodies, an oracle: %q", bodies)
	}
	if v, n := e.s30Counts(t); v != v0 || n != n0 {
		t.Fatalf("a refused body wrote")
	}
}

// rebuildOrgUnitClosure fills org_unit_closure from org_units for every
// tenant, as the migration's backfill does (nava_migrator only). The tests
// seed org_units directly, so they seed the closure the same way; group sync
// keeps it current in production (TICK-1033).
func rebuildOrgUnitClosure(t *testing.T, super interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}) {
	t.Helper()
	ctx := t.Context()
	tx, err := super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE nava_migrator"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "SELECT org_unit_closure_rebuild_all()"); err != nil {
		t.Fatalf("rebuild the closure: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

// s31Key is an Idempotency-Key for an S31 step.
func s31Key(name string) string { return s30Key(name) }
