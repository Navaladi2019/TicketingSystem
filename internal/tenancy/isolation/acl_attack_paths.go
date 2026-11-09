package isolation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// aclAttackPaths are the S10 attack rows (P3-37, TICK-622): the record ACL,
// the field masks, the admin routes and the explainer, attacked through the
// real nava-core handler over the suite's database as nava_app (etIso).
//
// Every row builds the same fixture (s10Setup): tenant A's type iso_et_s10
// under a policy that also names tenant B's groups and users (published past
// the PUT route's validation, as a corrupted or hand-written row would be),
// tenant B's own iso_et_s10 (so tenant B's requests get past the type lookup
// and reach the record lookup), and three records. Every row has positive
// controls, so a dead route or a broken fixture fails instead of passing as a
// denial. A row deletes everything it writes (s10Cleanup, etIso.cleanup).
//
// Each row was shown red against a mutation of the defence it proves; the
// mutations are listed in the PR body (TICK-622).
func aclAttackPaths() []Path {
	db := func(name string, fn func(context.Context, *etIso, *s10Fixture) error) Path {
		return Path{Name: "S10 attack " + name, Case: "CI-8", Role: "nava_app", DB: func(ctx context.Context, cl Cluster) error {
			e, err := newETIso(ctx, cl)
			if err != nil {
				return err
			}
			defer e.stop()
			clean := func() error {
				if err := s10Cleanup(ctx, e); err != nil {
					return err
				}
				return e.cleanup()
			}
			if err := clean(); err != nil { // rows a killed earlier run left
				return err
			}
			fx, err := s10Setup(ctx, e)
			if err == nil {
				err = fn(ctx, e, fx)
			}
			if cerr := clean(); err == nil {
				err = cerr
			}
			return err
		}}
	}
	ci8d := db("", s10MaskedFilter)
	ci8d.Name = "records CI-8: (d) a filter on a masked field is 400 masked_field, and the same filter by a caller the mask admits is 200"
	return []Path{
		ci8d,
		db("1: a rule naming tenant B's groups and users, stored in tenant A's policy past the PUT validation, grants tenant B's members nothing on GET, PATCH, history or the list", s10ForeignGroupRule),
		db("2: a record the caller cannot read is a random UUID's 404, byte-identical, on GET, history and PATCH with no, a stale and the current If-Match; never 412 or 428", s10UnreadableIsMissing),
		db("3: a filter (eq, in), a sort (asc, desc) or a count (total) on a masked field is 400 masked_field, the same bytes whether a record matches or not, with no total", s10MaskOracle),
		db("4: history before, after and changed[] never hold a masked field; an entry that changed only masked fields is still listed", s10HistoryMasks),
		db("5: the explainer answers a cross-tenant record or user with the random id's 404, a non-admin with 403, and never names a field the admin cannot read", s10Explainer),
		db("6: an agent cannot PATCH or create a record into a group they are not in unless a rule allows the after-state", s10WriteScope),
		db("7: a record_shares row in tenant B, with tenant A's record, user and group ids, grants nothing in tenant A", s10ShareForgery),
		db("8: PUT access with tenant B's group or user in a mask's read, write or users is the unknown id's 422, and writes nothing", s10PutForeignMaskIDs),
	}
}

const (
	s10Key     = "iso_et_s10"
	s10Records = "/api/v1/records/" + s10Key
	// s10Random is an id no row has, the "missing" side of every comparison.
	s10Random = "0194f2c0-0000-7000-8000-0000000c1810"
	// s10UnknownID is a group or user id no tenant has.
	s10UnknownID = "00000000-0000-4000-8000-00000000dead"
	// The callers (newETIso): a1 is tenant A's admin (admins group a1), a2
	// is the desk agent (auditors group a2, the desk rule's group) or, with
	// no group, a plain user no rule names.
	s10Admin = etCallerA
	s10Desk  = auditCallerAAuditor
	s10Plain = auditCallerAAgent
	// s10Forged are the Keycloak ids of the share-forgery rows (row 7).
	s10Forged = "kc-iso-s10-forged"
)

// s10Schema: state, note, cost (masked to admins), triage (masked to a2
// only, so the admin cannot read it) and the three reference columns.
const s10Schema = `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,` +
	`"required":["state"],"properties":{"state":{"type":"string","enum":["new","closed"],"default":"new"},` +
	`"note":{"type":"string"},"cost":{"type":"number"},"triage":{"type":"string"},` +
	`"group":{"type":"string","format":"uuid"},"assignee":{"type":"string","format":"uuid"},"requester":{"type":"string","format":"uuid"}}}`

func s10TypeBody(prefix string) string {
	return fmt.Sprintf(`{"key": %q, "name": "Iso S10", "number_prefix": %q, "definition": {"schema": %s,
		"fixed_columns": {"state": "state", "group": "assignment_group_id", "assignee": "assignee_id", "requester": "requester_id"},
		"states": {"initial": "new", "states": ["new", "closed"], "transitions": [{"from": "new", "to": "closed"}]}}}`,
		s10Key, prefix, s10Schema)
}

func s10UUIDs(ids ...string) []uuid.UUID {
	out := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		out[i] = uuid.MustParse(id)
	}
	return out
}

// s10Rules is tenant A's policy for iso_et_s10. "foreign" names tenant B's
// groups and users: the PUT route refuses it (row 8), so it is published
// past it, as a hand-written or corrupted row would be (row 1).
func s10Rules() []authz.Rule {
	return []authz.Rule{
		{ID: "admins", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Groups: s10UUIDs(adminGroupA)}},
		{ID: "desk", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Groups: s10UUIDs(auditorGroupA)},
			Where: []authz.Cond{{Column: "assignment_group_id", Op: authz.CondIn, Caller: authz.CallerMyGroups}}},
		{ID: "requester", Ops: []string{authz.OpRead}, Who: authz.Subject{Requester: true}},
		{ID: "shared", Ops: []string{authz.OpRead}, Who: authz.Subject{Shared: true}},
		{ID: "foreign", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{
			Groups: s10UUIDs(GroupB1, adminGroupB, auditorGroupB), Users: s10UUIDs(PrincipalB1, principalB2)}},
	}
}

// s10Masks: cost and assignee (a fixed column, so it is sortable) are the
// admins'; triage is a2's only, so the admin cannot read it.
func s10Masks() []authz.Mask {
	admins := authz.Subject{Groups: s10UUIDs(adminGroupA)}
	a2 := authz.Subject{Users: s10UUIDs(principalA2)}
	return []authz.Mask{
		{Field: "cost", Read: admins, Write: admins},
		{Field: "assignee", Read: admins, Write: admins},
		{Field: "triage", Read: a2, Write: a2},
	}
}

// s10Fixture: desk is tenant A's record in the desk group, assigned to a1,
// at version 2 (the patch to version 2 changed cost only), so W/"1" is stale
// and W/"2" current; admin is tenant A's record in the admins group, which
// the desk cannot read; b is tenant B's own record; defB is tenant B's
// iso_et_s10.
type s10Fixture struct {
	desk, admin, b, defB string
}

func s10Setup(ctx context.Context, e *etIso) (*s10Fixture, error) {
	if code, body := e.do(ctx, http.MethodPost, "/api/v1/entity-types", s10Admin, s10TypeBody("ISX"), nil); code != http.StatusCreated {
		return nil, fmt.Errorf("control: tenant A creating %s = %d %s, want 201", s10Key, code, body)
	}
	if err := s10Publish(ctx, e, s10Rules()); err != nil {
		return nil, err
	}
	if code, body := e.do(ctx, http.MethodPost, "/api/v1/entity-types", etCallerBAdmin, s10TypeBody("ISY"), nil); code != http.StatusCreated {
		return nil, fmt.Errorf("control: tenant B creating its own %s = %d %s, want 201", s10Key, code, body)
	}
	fx := &s10Fixture{}
	var err error
	if fx.desk, err = s10Create(ctx, e, s10Admin, fmt.Sprintf(`{"data": {"note": "d1", "cost": 500, "group": %q, "assignee": %q}}`, auditorGroupA, principalA1)); err != nil {
		return nil, err
	}
	if r := e.ci8Do(ctx, e.h, http.MethodPatch, s10Records+"/"+fx.desk, s10Admin, `{"data": {"cost": 600}}`, ci8IfMatch(`W/"1"`)); r.code != http.StatusOK {
		return nil, fmt.Errorf("control: the admin patching cost to version 2 = %d %s, want 200", r.code, r.body)
	}
	if fx.admin, err = s10Create(ctx, e, s10Admin, fmt.Sprintf(`{"data": {"note": "s", "cost": 7, "group": %q}}`, adminGroupA)); err != nil {
		return nil, err
	}
	if fx.b, err = s10Create(ctx, e, auditCallerBAgent, `{"data": {"note": "b"}}`); err != nil {
		return nil, err
	}
	if err := e.super.QueryRow(ctx, `SELECT id::text FROM entity_definitions WHERE tenant_id = $1 AND key = $2`, TenantB, s10Key).Scan(&fx.defB); err != nil {
		return nil, err
	}
	// Positive controls of the policy: the desk reads the desk record, not the
	// admin record; the plain user reads neither.
	for _, c := range []struct {
		token, id string
		want      int
	}{{s10Desk, fx.desk, http.StatusOK}, {s10Desk, fx.admin, http.StatusNotFound}, {s10Plain, fx.desk, http.StatusNotFound}, {s10Admin, fx.admin, http.StatusOK}} {
		if r := e.ci8Do(ctx, e.h, http.MethodGet, s10Records+"/"+c.id, c.token, "", nil); r.code != c.want {
			return nil, fmt.Errorf("control: %s reading %s = %d %s, want %d", c.token, c.id, r.code, r.body, c.want)
		}
	}
	return fx, nil
}

// s10Publish publishes rules with s10Masks as tenant A's next ACL version of
// iso_et_s10, past the PUT route's validation.
func s10Publish(ctx context.Context, e *etIso, rules []authz.Rule) error {
	return tenancy.InTenantTx(ctx, e.pool, uuid.MustParse(TenantA), func(tx pgx.Tx) error {
		_, err := acltest.PublishPolicy(ctx, tx, s10Key, rules, s10Masks())
		return err
	})
}

func s10Create(ctx context.Context, e *etIso, token, body string) (string, error) {
	r := e.ci8Do(ctx, e.h, http.MethodPost, s10Records, token, body, nil)
	if r.code != http.StatusCreated {
		return "", fmt.Errorf("control: %s creating a record = %d %s, want 201", token, r.code, r.body)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(r.body), &out); err != nil || out.ID == "" {
		return "", fmt.Errorf("control: the created record has no id: %w: %s", err, r.body)
	}
	return out.ID, nil
}

// s10Cleanup deletes what the share forgery (row 7) writes past the API: every
// share of an iso_et_* record, and the forged tenant B principal and group.
// Triggers are off (session_replication_role), as they are when it writes.
func s10Cleanup(ctx context.Context, e *etIso) error {
	tx, err := e.super.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, q := range []string{
		`SET LOCAL session_replication_role = replica`,
		`DELETE FROM record_shares s USING records r, entity_definitions d
			WHERE r.tenant_id = s.tenant_id AND r.id = s.record_id AND d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id AND d.key LIKE 'iso\_et\_%'`,
		`DELETE FROM principals WHERE keycloak_user_id = '` + s10Forged + `'`,
		`DELETE FROM groups WHERE keycloak_group_id = '` + s10Forged + `'`,
	} {
		if _, err := tx.Exec(ctx, q); err != nil {
			return fmt.Errorf("s10 cleanup: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// s10Page is a list response.
type s10Page struct {
	Items []struct {
		ID string `json:"id"`
	} `json:"items"`
	Total *int `json:"total"`
}

func (p s10Page) ids() []string {
	out := make([]string, 0, len(p.Items))
	for _, it := range p.Items {
		out = append(out, it.ID)
	}
	return out
}

// s10List lists iso_et_s10 as token with query (no leading ?) and requires
// 200.
func s10List(ctx context.Context, e *etIso, token, query string) (s10Page, error) {
	path := s10Records
	if query != "" {
		path += "?" + query
	}
	r := e.ci8Do(ctx, e.h, http.MethodGet, path, token, "", nil)
	var p s10Page
	if r.code != http.StatusOK {
		return p, fmt.Errorf("control: %s listing %s = %d %s, want 200", token, path, r.code, r.body)
	}
	if err := json.Unmarshal([]byte(r.body), &p); err != nil || p.Total == nil {
		return p, fmt.Errorf("control: %s listing %s: not a page: %v: %s", token, path, err, r.body)
	}
	return p, nil
}

// s10ListHas: token's list holds exactly want, in any order, and total is
// len(want).
func s10ListHas(ctx context.Context, e *etIso, token, query string, want ...string) error {
	p, err := s10List(ctx, e, token, query)
	if err != nil {
		return err
	}
	got := p.ids()
	slices.Sort(got)
	want = slices.Clone(want)
	slices.Sort(want)
	if !slices.Equal(got, want) || *p.Total != len(want) {
		return fmt.Errorf("%s's list ?%s holds %v with total %d, want %v", token, query, got, *p.Total, want)
	}
	return nil
}

// s10Missing: token's request to id is the random UUID's, byte for byte
// (ci8Same), for every method and suffix of the record routes.
func s10Missing(ctx context.Context, e *etIso, token, id string) error {
	for _, c := range []struct {
		method, suffix, body string
		hdr                  http.Header
	}{
		{http.MethodGet, "", "", nil},
		{http.MethodGet, "/history", "", nil},
		{http.MethodPatch, "", `{"data": {"note": "forged"}}`, ci8IfMatch("")},
		{http.MethodPatch, "", `{"data": {"note": "forged"}}`, ci8IfMatch(`W/"1"`)},
		{http.MethodPatch, "", `{"data": {"note": "forged"}}`, ci8IfMatch(`W/"2"`)},
	} {
		what := fmt.Sprintf("%s %s%s If-Match %q as %s", c.method, id, c.suffix, c.hdr.Get("If-Match"), token)
		other := e.ci8Do(ctx, e.h, c.method, s10Records+"/"+id+c.suffix, token, c.body, c.hdr)
		missing := e.ci8Do(ctx, e.h, c.method, s10Records+"/"+s10Random+c.suffix, token, c.body, c.hdr)
		if other.code == http.StatusPreconditionFailed || other.code == http.StatusPreconditionRequired {
			return fmt.Errorf("%s = %d: an unreadable record answered a precondition: %s", what, other.code, other.body)
		}
		if err := ci8Same(what, other, missing); err != nil {
			return err
		}
	}
	return nil
}

// s10State is a record's version, group and note, read past RLS, with its
// audit row count.
func s10State(ctx context.Context, e *etIso, id string) (string, error) {
	var (
		tenant, group, note string
		v                   int64
		n                   int
	)
	if err := e.super.QueryRow(ctx, `SELECT tenant_id::text, version, coalesce(assignment_group_id::text, ''), coalesce(data->>'note', ''),
		(SELECT count(*) FROM audit_log WHERE subject_type = 'record' AND subject_id = $1)
		FROM records WHERE id = $1 AND tenant_id = $2`, id, TenantA).Scan(&tenant, &v, &group, &note, &n); err != nil {
		return "", err
	}
	return fmt.Sprintf("tenant %s version %d group %s note %q audits %d", tenant, v, group, note, n), nil
}

func s10Unchanged(ctx context.Context, e *etIso, id, before string) error {
	after, err := s10State(ctx, e, id)
	if err != nil {
		return err
	}
	if after != before {
		return fmt.Errorf("record %s changed: %s -> %s", id, before, after)
	}
	return nil
}

// Row 1. Tenant A's policy names tenant B's groups and users (rule
// "foreign"). Tenant B's members of those groups, holding valid tokens for
// tenant B, reach none of tenant A's records: the record lookup and the list
// run in tenant B's own transaction, and the rule is tenant A's.
func s10ForeignGroupRule(ctx context.Context, e *etIso, fx *s10Fixture) error {
	// Control: the stored policy does name tenant B's ids.
	var named bool
	if err := e.super.QueryRow(ctx, `SELECT rules::text LIKE '%' || $3 || '%' FROM acl_versions a JOIN entity_definitions d
		ON d.tenant_id = a.tenant_id AND d.id = a.entity_definition_id
		WHERE a.tenant_id = $1 AND d.key = $2 ORDER BY a.version DESC LIMIT 1`, TenantA, s10Key, GroupB1).Scan(&named); err != nil {
		return err
	}
	if !named {
		return errors.New("control: tenant A's newest iso_et_s10 policy does not name tenant B's group")
	}
	before, err := s10State(ctx, e, fx.desk)
	if err != nil {
		return err
	}
	// auditCallerBAgent is b1 in GroupB1; etCallerBAdmin is b2 in B's admins.
	for _, token := range []string{auditCallerBAgent, etCallerBAdmin} {
		// Control: the caller reads its own tenant's record.
		if r := e.ci8Do(ctx, e.h, http.MethodGet, s10Records+"/"+fx.b, token, "", nil); r.code != http.StatusOK {
			return fmt.Errorf("control: %s reading tenant B's own record = %d %s, want 200", token, r.code, r.body)
		}
		for _, id := range []string{fx.desk, fx.admin} {
			if err := s10Missing(ctx, e, token, id); err != nil {
				return err
			}
		}
		if err := s10ListHas(ctx, e, token, "", fx.b); err != nil {
			return err
		}
		// The foreign rule's own group, named as a filter, finds nothing either.
		if err := s10ListHas(ctx, e, token, "filter[group]="+GroupB1); err != nil {
			return err
		}
	}
	return s10Unchanged(ctx, e, fx.desk, before)
}

// Row 2. Inside tenant A: the plain user can read neither record, the desk
// cannot read the admin record. Each answer is the random UUID's.
func s10UnreadableIsMissing(ctx context.Context, e *etIso, fx *s10Fixture) error {
	// Controls: for a caller who can read it, the desk record does answer 412
	// to a stale ETag and 428 to none, so a 404 below is the ACL.
	if r := e.ci8Do(ctx, e.h, http.MethodPatch, s10Records+"/"+fx.desk, s10Desk, `{"data": {"note": "x"}}`, ci8IfMatch(`W/"1"`)); r.code != http.StatusPreconditionFailed {
		return fmt.Errorf("control: the desk's PATCH of the desk record with a stale ETag = %d %s, want 412", r.code, r.body)
	}
	if r := e.ci8Do(ctx, e.h, http.MethodPatch, s10Records+"/"+fx.desk, s10Desk, `{"data": {"note": "x"}}`, ci8IfMatch("")); r.code != http.StatusPreconditionRequired {
		return fmt.Errorf("control: the desk's PATCH of the desk record without If-Match = %d %s, want 428", r.code, r.body)
	}
	desk, err := s10State(ctx, e, fx.desk)
	if err != nil {
		return err
	}
	admin, err := s10State(ctx, e, fx.admin)
	if err != nil {
		return err
	}
	for _, c := range []struct{ token, id string }{{s10Plain, fx.desk}, {s10Plain, fx.admin}, {s10Desk, fx.admin}} {
		if err := s10Missing(ctx, e, c.token, c.id); err != nil {
			return err
		}
	}
	if err := s10Unchanged(ctx, e, fx.desk, desk); err != nil {
		return err
	}
	return s10Unchanged(ctx, e, fx.admin, admin)
}

// s10Problem is a problem body without its instance, and its code.
func s10Problem(r ci8Resp) (string, string, error) {
	body, err := withoutInstance(r.body)
	if err != nil {
		return "", "", err
	}
	var p struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal([]byte(r.body), &p); err != nil {
		return "", "", err
	}
	return body, p.Code, nil
}

// CI-8 (d) (D-3-19): the desk cannot read cost; filtering on it is 400
// masked_field. The admin, whom the mask admits, gets 200 for the same query.
func s10MaskedFilter(ctx context.Context, e *etIso, fx *s10Fixture) error {
	if err := s10ListHas(ctx, e, s10Admin, "filter[cost]=600", fx.desk); err != nil {
		return fmt.Errorf("control: %w", err)
	}
	r := e.ci8Do(ctx, e.h, http.MethodGet, s10Records+"?filter[cost]=600", s10Desk, "", nil)
	if _, code, err := s10Problem(r); err != nil || r.code != http.StatusBadRequest || code != "masked_field" {
		return fmt.Errorf("the desk's filter on cost = %d %s, want 400 masked_field", r.code, r.body)
	}
	return nil
}

// Row 3. The desk cannot read cost or assignee. Naming either
// in a filter or the sort is 400 masked_field before any SQL: the same bytes
// for a value that matches the desk record and one that matches nothing, so
// neither the status, the body nor a total tells the caller the value.
// total is the only count M1 has: there is no count or aggregate route.
func s10MaskOracle(ctx context.Context, e *etIso, fx *s10Fixture) error {
	// Controls: the admin, who may read both, gets 200 and the true totals;
	// the desk's filter on an open field works.
	for _, c := range []struct {
		query string
		want  []string
	}{
		{"filter[cost]=600", []string{fx.desk}}, {"filter[cost]=1", nil},
		{"filter[assignee]=" + principalA1, []string{fx.desk}}, {"sort=assignee", []string{fx.desk, fx.admin}},
	} {
		if err := s10ListHas(ctx, e, s10Admin, c.query, c.want...); err != nil {
			return fmt.Errorf("control: %w", err)
		}
	}
	if err := s10ListHas(ctx, e, s10Desk, "filter[note]=d1", fx.desk); err != nil {
		return fmt.Errorf("control: %w", err)
	}
	// Each pair: the first value matches the desk record, the second nothing.
	for _, pair := range [][2]string{
		{"filter[cost]=600", "filter[cost]=1"},
		{"filter[cost][in]=600,1", "filter[cost][in]=2,1"},
		{"filter[assignee]=" + principalA1, "filter[assignee]=" + s10Random},
		{"filter[assignee][in]=" + principalA1 + "," + s10Random, "filter[assignee][in]=" + s10UnknownID + "," + s10Random},
		{"filter[note]=d1&filter[cost]=600", "filter[note]=d1&filter[cost]=1"},
		{"sort=assignee", "sort=-assignee"},
		{"sort=assignee&filter[note]=d1", "sort=assignee&filter[note]=zz"},
	} {
		var bodies [2]string
		for i, q := range pair {
			r := e.ci8Do(ctx, e.h, http.MethodGet, s10Records+"?"+q, s10Desk, "", nil)
			body, code, err := s10Problem(r)
			switch {
			case err != nil:
				return fmt.Errorf("the desk's ?%s = %d: %w", q, r.code, err)
			case r.code != http.StatusBadRequest || code != "masked_field":
				return fmt.Errorf("the desk's ?%s = %d %s, want 400 masked_field", q, r.code, r.body)
			case strings.Contains(body, `"total"`) || strings.Contains(body, `"items"`):
				return fmt.Errorf("the desk's ?%s: the 400 carries a page or a total: %s", q, r.body)
			case strings.Contains(body, "600") || strings.Contains(body, principalA1):
				return fmt.Errorf("the desk's ?%s: the 400 repeats the value: %s", q, r.body)
			}
			bodies[i] = body
		}
		if bodies[0] != bodies[1] {
			return fmt.Errorf("the desk's 400 differs between ?%s (a match) and ?%s (none):\n%s\n%s", pair[0], pair[1], bodies[0], bodies[1])
		}
	}
	return nil
}

// s10HistoryEntries is a history page.
type s10HistoryEntries struct {
	Items []struct {
		Changed []string       `json:"changed"`
		Before  map[string]any `json:"before"`
		After   map[string]any `json:"after"`
	} `json:"items"`
}

func s10History(ctx context.Context, e *etIso, token, id string) (s10HistoryEntries, string, error) {
	r := e.ci8Do(ctx, e.h, http.MethodGet, s10Records+"/"+id+"/history", token, "", nil)
	var h s10HistoryEntries
	if r.code != http.StatusOK {
		return h, r.body, fmt.Errorf("control: %s reading %s's history = %d %s, want 200", token, id, r.code, r.body)
	}
	if err := json.Unmarshal([]byte(r.body), &h); err != nil {
		return h, r.body, fmt.Errorf("control: not a history page: %w: %s", err, r.body)
	}
	return h, r.body, nil
}

// Row 4. The desk record's history is its create (cost 500, assignee a1) and
// a patch that changed cost only. The desk reads both entries, with cost and
// assignee nowhere in before, after or changed, and the cost-only entry with
// an empty changed[].
func s10HistoryMasks(ctx context.Context, e *etIso, fx *s10Fixture) error {
	// Control: the admin's history shows cost and assignee.
	h, body, err := s10History(ctx, e, s10Admin, fx.desk)
	if err != nil {
		return err
	}
	if len(h.Items) != 2 || !strings.Contains(body, `"cost"`) || !strings.Contains(body, `"assignee"`) {
		return fmt.Errorf("control: the admin's history has %d entries, want 2 with cost and assignee: %s", len(h.Items), body)
	}
	h, body, err = s10History(ctx, e, s10Desk, fx.desk)
	if err != nil {
		return err
	}
	if len(h.Items) != 2 {
		return fmt.Errorf("the desk's history has %d entries, want 2 (the cost-only patch is still listed): %s", len(h.Items), body)
	}
	masked := []string{"cost", "assignee"}
	for i, it := range h.Items {
		for _, f := range masked {
			_, inBefore := it.Before[f]
			_, inAfter := it.After[f]
			if inBefore || inAfter || slices.Contains(it.Changed, f) {
				return fmt.Errorf("the desk's history entry %d holds the masked field %s: %s", i, f, body)
			}
		}
		if it.After == nil {
			return fmt.Errorf("control: the desk's history entry %d has no after: %s", i, body)
		}
	}
	for _, leak := range []string{`"cost"`, `"assignee"`} {
		if strings.Contains(body, leak) {
			return fmt.Errorf("the desk's history shows %s: %s", leak, body)
		}
	}
	// The patch changed cost only; its entry is the one without a before
	// missing, and lists no change the desk may see.
	for _, it := range h.Items {
		if it.Before != nil && len(it.Changed) != 0 {
			return fmt.Errorf("the desk's entry of the cost-only patch lists %v as changed: %s", it.Changed, body)
		}
	}
	return nil
}

type s10Explanation struct {
	Decision     string   `json:"decision"`
	RuleID       *string  `json:"rule_id"`
	MaskedFields []string `json:"masked_fields"`
}

func s10Explain(ctx context.Context, e *etIso, token, entity, record, user string) ci8Resp {
	q := url.Values{"entity": {entity}, "record": {record}, "user": {user}}
	return e.ci8Do(ctx, e.h, http.MethodGet, "/api/v1/access/explain?"+q.Encode(), token, "", nil)
}

// Row 5. The explainer (P3-35).
func s10Explainer(ctx context.Context, e *etIso, fx *s10Fixture) error {
	// Control: the admin explains the desk's read of the desk record: allowed
	// by "desk", with cost and assignee masked (the admin may read both).
	r := s10Explain(ctx, e, s10Admin, s10Key, fx.desk, principalA2)
	var x s10Explanation
	if r.code != http.StatusOK || json.Unmarshal([]byte(r.body), &x) != nil || x.Decision != "allow" || x.RuleID == nil || *x.RuleID != "desk" {
		return fmt.Errorf("control: the admin explaining a2 on the desk record = %d %s, want 200 allow by desk", r.code, r.body)
	}
	if slices.Sort(x.MaskedFields); !slices.Equal(x.MaskedFields, []string{"assignee", "cost"}) {
		return fmt.Errorf("control: masked_fields for a2 = %v, want [assignee cost]", x.MaskedFields)
	}
	// triage is hidden from a1, and a1 cannot read it either: explaining a1
	// must not name it.
	r = s10Explain(ctx, e, s10Admin, s10Key, fx.desk, principalA1)
	x = s10Explanation{}
	if r.code != http.StatusOK || json.Unmarshal([]byte(r.body), &x) != nil || x.Decision != "allow" {
		return fmt.Errorf("control: the admin explaining itself = %d %s, want 200 allow", r.code, r.body)
	}
	if len(x.MaskedFields) != 0 || strings.Contains(r.body, "triage") {
		return fmt.Errorf("the explainer names a field the admin cannot read: %s", r.body)
	}
	// Cross-tenant record or user: the random id's 404, byte for byte.
	for _, c := range []struct {
		what, token, record, user, randomRecord, randomUser string
	}{
		{"tenant B's record", s10Admin, fx.b, principalA2, s10Random, principalA2},
		{"tenant B's user", s10Admin, fx.desk, PrincipalB1, fx.desk, s10UnknownID},
		{"tenant B's admin on tenant A's record", etCallerBAdmin, fx.desk, principalB1, s10Random, principalB1},
		{"tenant B's admin on tenant A's user", etCallerBAdmin, fx.b, principalA2, fx.b, s10UnknownID},
	} {
		other := s10Explain(ctx, e, c.token, s10Key, c.record, c.user)
		missing := s10Explain(ctx, e, c.token, s10Key, c.randomRecord, c.randomUser)
		if err := ci8Same("explain "+c.what, other, missing); err != nil {
			return err
		}
	}
	// Control: tenant B's admin explains its own record with 200.
	if r := s10Explain(ctx, e, etCallerBAdmin, s10Key, fx.b, principalB1); r.code != http.StatusOK {
		return fmt.Errorf("control: tenant B's admin explaining its own record = %d %s, want 200", r.code, r.body)
	}
	// A non-admin: 403, the same for a real and a random record (the 403 is
	// decided before the record is looked up, so it is no oracle either).
	for _, token := range []string{s10Desk, s10Plain, auditCallerBAgent} {
		var bodies [2]string
		for i, rec := range []string{fx.desk, s10Random} {
			r := s10Explain(ctx, e, token, s10Key, rec, principalA2)
			body, code, err := s10Problem(r)
			if err != nil || r.code != http.StatusForbidden || code != "forbidden" {
				return fmt.Errorf("%s explaining %s = %d %s, want 403 forbidden", token, rec, r.code, r.body)
			}
			bodies[i] = body
		}
		if bodies[0] != bodies[1] {
			return fmt.Errorf("%s's 403 differs for a real and a random record:\n%s\n%s", token, bodies[0], bodies[1])
		}
	}
	return nil
}

// Row 6. The desk writes records in its own groups. Moving the desk record to
// the admins group, or creating one there, is 403 and writes nothing; once a
// rule allows the after-state, the same move succeeds.
func s10WriteScope(ctx context.Context, e *etIso, fx *s10Fixture) error {
	// Control: the desk can write the desk record.
	if r := e.ci8Do(ctx, e.h, http.MethodPatch, s10Records+"/"+fx.desk, s10Desk, `{"data": {"note": "d2"}}`, ci8IfMatch(`W/"2"`)); r.code != http.StatusOK {
		return fmt.Errorf("control: the desk patching the note = %d %s, want 200", r.code, r.body)
	}
	before, err := s10State(ctx, e, fx.desk)
	if err != nil {
		return err
	}
	move := fmt.Sprintf(`{"data": {"group": %q}}`, adminGroupA)
	if r := e.ci8Do(ctx, e.h, http.MethodPatch, s10Records+"/"+fx.desk, s10Desk, move, ci8IfMatch(`W/"3"`)); r.code != http.StatusForbidden {
		return fmt.Errorf("the desk moving its record to the admins group = %d %s, want 403", r.code, r.body)
	}
	if err := s10Unchanged(ctx, e, fx.desk, before); err != nil {
		return err
	}
	count := func() (int, error) {
		var n int
		err := e.super.QueryRow(ctx, `SELECT count(*) FROM records r JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
			WHERE r.tenant_id = $1 AND d.key = $2`, TenantA, s10Key).Scan(&n)
		return n, err
	}
	n, err := count()
	if err != nil {
		return err
	}
	if r := e.ci8Do(ctx, e.h, http.MethodPost, s10Records, s10Desk, fmt.Sprintf(`{"data": {"note": "x", "group": %q}}`, adminGroupA), nil); r.code != http.StatusForbidden {
		return fmt.Errorf("the desk creating a record in the admins group = %d %s, want 403", r.code, r.body)
	}
	if m, err := count(); err != nil || m != n {
		return fmt.Errorf("the refused create left %d records, want %d (%v)", m, n, err)
	}
	// Control: in its own group the desk does create.
	if _, err := s10Create(ctx, e, s10Desk, fmt.Sprintf(`{"data": {"note": "x", "group": %q}}`, auditorGroupA)); err != nil {
		return err
	}
	// A rule that allows the after-state lets the same move through.
	handoff := authz.Rule{ID: "handoff", Ops: []string{authz.OpWrite}, Who: authz.Subject{Groups: s10UUIDs(auditorGroupA)},
		Where: []authz.Cond{{Column: "assignment_group_id", Op: authz.CondIn, Values: []string{adminGroupA}}}}
	if err := s10Publish(ctx, e, append(s10Rules(), handoff)); err != nil {
		return err
	}
	if r := e.ci8Do(ctx, e.h, http.MethodPatch, s10Records+"/"+fx.desk, s10Desk, move, ci8IfMatch(`W/"3"`)); r.code != http.StatusOK {
		return fmt.Errorf("the desk's move under a rule that allows the after-state = %d %s, want 200", r.code, r.body)
	}
	return nil
}

// Row 7. The admin record is in the admins group, so neither a2 (plain) nor
// a2 (desk) can read it. A genuine tenant A share lets each read it (the
// control). Then tenant B is given a principal with a2's id, a group with the
// desk group's id and a record with the admin record's id, and shares that
// record with both, in tenant B: none of it lets a2 read tenant A's record.
func s10ShareForgery(ctx context.Context, e *etIso, fx *s10Fixture) error {
	exec := func(qs ...string) error {
		tx, err := e.super.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		for _, q := range qs {
			if _, err := tx.Exec(ctx, q); err != nil {
				return fmt.Errorf("%w: %s", err, q)
			}
		}
		return tx.Commit(ctx)
	}
	share := func(tenant, kind, col, id string) string {
		return fmt.Sprintf(`INSERT INTO record_shares (tenant_id, record_id, principal_kind, %s) VALUES ('%s', '%s', '%s', '%s')`, col, tenant, fx.admin, kind, id)
	}
	unshare := fmt.Sprintf(`DELETE FROM record_shares WHERE record_id = '%s'`, fx.admin)
	// Controls: a genuine share in tenant A does grant read, by user and by
	// group, on GET and in the list.
	for _, c := range []struct{ token, kind, col, id string }{
		{s10Plain, "user", "user_id", principalA2},
		{s10Desk, "group", "group_id", auditorGroupA},
	} {
		if err := exec(share(TenantA, c.kind, c.col, c.id)); err != nil {
			return err
		}
		if r := e.ci8Do(ctx, e.h, http.MethodGet, s10Records+"/"+fx.admin, c.token, "", nil); r.code != http.StatusOK {
			return fmt.Errorf("control: %s reading the admin record under a tenant A %s share = %d %s, want 200", c.token, c.kind, r.code, r.body)
		}
		want := []string{fx.desk} // the desk's own record, plus the shared one
		if c.token == s10Plain {
			want = nil
		}
		if err := s10ListHas(ctx, e, c.token, "", append(want, fx.admin)...); err != nil {
			return fmt.Errorf("control: %w", err)
		}
		if err := exec(unshare); err != nil {
			return err
		}
	}
	// The forgery, as a superuser past every trigger and policy.
	if err := exec(
		`SET LOCAL session_replication_role = replica`,
		fmt.Sprintf(`INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, federated_links, synced_at) VALUES ('%s', '%s', '%s', true, '[]', now())`,
			TenantB, principalA2, s10Forged),
		fmt.Sprintf(`INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ('%s', '%s', '%s', '/t/%s/iso-s10-forged')`,
			TenantB, auditorGroupA, s10Forged, TenantB),
		fmt.Sprintf(`INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, data) VALUES ('%s', '%s', 1, '%s', 1, 'ISY9999999', 'new', '{}')`,
			TenantB, fx.admin, fx.defB),
		share(TenantB, "user", "user_id", principalA2),
		share(TenantB, "group", "group_id", auditorGroupA),
	); err != nil {
		return err
	}
	var forged int
	if err := e.super.QueryRow(ctx, `SELECT count(*) FROM record_shares WHERE tenant_id = $1 AND record_id = $2`, TenantB, fx.admin).Scan(&forged); err != nil {
		return err
	}
	if forged != 2 {
		return fmt.Errorf("control: %d forged shares in tenant B, want 2", forged)
	}
	for _, token := range []string{s10Plain, s10Desk} {
		other := e.ci8Do(ctx, e.h, http.MethodGet, s10Records+"/"+fx.admin, token, "", nil)
		missing := e.ci8Do(ctx, e.h, http.MethodGet, s10Records+"/"+s10Random, token, "", nil)
		if err := ci8Same("GET of the admin record under tenant B's forged shares as "+token, other, missing); err != nil {
			return err
		}
	}
	if err := s10ListHas(ctx, e, s10Plain, ""); err != nil {
		return err
	}
	if err := s10ListHas(ctx, e, s10Desk, "", fx.desk); err != nil {
		return err
	}
	r := s10Explain(ctx, e, s10Admin, s10Key, fx.admin, principalA2)
	var x s10Explanation
	if r.code != http.StatusOK || json.Unmarshal([]byte(r.body), &x) != nil || x.Decision != "deny" {
		return fmt.Errorf("the explainer on a2 and the admin record under tenant B's forged shares = %d %s, want 200 deny", r.code, r.body)
	}
	return nil
}

// Row 8 (PR #531 review). Tenant A's admin publishes masks naming tenant B's
// group or user in read, write or users. Each is the 422 a random id gets,
// bytes equal but for instance, with the foreign id's pointer not_in_tenant,
// and no ACL version or audit entry is written.
func s10PutForeignMaskIDs(ctx context.Context, e *etIso, _ *s10Fixture) error {
	path := "/api/v1/entity-types/" + s10Key + "/access"
	get := e.ci8Do(ctx, e.h, http.MethodGet, path, s10Admin, "", nil)
	etag := get.hdr.Get("ETag")
	if get.code != http.StatusOK || etag == "" {
		return fmt.Errorf("control: the admin reading the ACL = %d %s (ETag %q), want 200 with an ETag", get.code, get.body, etag)
	}
	counts := func() (string, error) {
		var v, n int
		err := e.super.QueryRow(ctx, `SELECT (SELECT count(*) FROM acl_versions a JOIN entity_definitions d ON d.tenant_id = a.tenant_id AND d.id = a.entity_definition_id
			WHERE d.key = $1), (SELECT count(*) FROM audit_log WHERE subject_type = 'acl_policy')`, s10Key).Scan(&v, &n)
		return fmt.Sprintf("%d versions, %d acl_policy entries", v, n), err
	}
	put := func(n int, body string) ci8Resp {
		h := http.Header{"If-Match": {etag}, "Idempotency-Key": {fmt.Sprintf("iso-et-s10-put-%d", n)}}
		return e.ci8Do(ctx, e.h, http.MethodPut, path, s10Admin, body, h)
	}
	mask := func(read, write string) string {
		return `{"rules": [], "masks": [{"field": "note", "read": ` + read + `, "write": ` + write + `}]}`
	}
	q := func(id string) string { return fmt.Sprintf("%q", id) }
	before, err := counts()
	if err != nil {
		return err
	}
	for i, c := range []struct {
		name, pointer, rule string
		body                func(id string) string
		foreign             string
	}{
		{"a group in read and write", "/masks/0/write/groups/0", "not_in_tenant", func(id string) string {
			return mask(`{"groups": [`+q(id)+`]}`, `{"groups": [`+q(id)+`]}`)
		}, GroupB1},
		// Write must be within read, so a group in write alone is refused as
		// mask_write before its tenant is looked up: the same 422 either way.
		{"a group in write only", "/masks/0/write/groups/1", "mask_write", func(id string) string {
			return mask(`{"groups": [`+q(adminGroupA)+`]}`, `{"groups": [`+q(adminGroupA)+`, `+q(id)+`]}`)
		}, adminGroupB},
		{"a user in read", "/masks/0/read/users/0", "not_in_tenant", func(id string) string {
			return mask(`{"users": [`+q(id)+`]}`, `{}`)
		}, PrincipalB1},
		{"a user in read and write", "/masks/0/write/users/0", "not_in_tenant", func(id string) string {
			return mask(`{"users": [`+q(id)+`]}`, `{"users": [`+q(id)+`]}`)
		}, principalB2},
	} {
		var bodies [2]string
		for j, id := range []string{c.foreign, s10UnknownID} {
			r := put(2*i+j, c.body(id))
			body, code, err := s10Problem(r)
			switch {
			case err != nil || r.code != http.StatusUnprocessableEntity || code != "invalid_access_policy":
				return fmt.Errorf("%s = %s: %d %s, want 422 invalid_access_policy", c.name, id, r.code, r.body)
			case !strings.Contains(r.body, `{"pointer":"`+c.pointer+`","rule":"`+c.rule+`"}`):
				return fmt.Errorf("%s = %s: the 422 does not name %s %s: %s", c.name, id, c.pointer, c.rule, r.body)
			case strings.Contains(body, id):
				return fmt.Errorf("%s = %s: the 422 repeats the id: %s", c.name, id, r.body)
			}
			bodies[j] = body
		}
		if bodies[0] != bodies[1] {
			return fmt.Errorf("%s: tenant B's id and an unknown id answer differently:\n%s\n%s", c.name, bodies[0], bodies[1])
		}
		if after, err := counts(); err != nil || after != before {
			return fmt.Errorf("%s: wrote: %s -> %s (%v)", c.name, before, after, err)
		}
	}
	// Control: the same masks with tenant A's own ids publish.
	own := mask(`{"groups": [`+q(adminGroupA)+`], "users": [`+q(principalA2)+`]}`, `{"groups": [`+q(adminGroupA)+`], "users": [`+q(principalA2)+`]}`)
	if r := put(99, own); r.code != http.StatusOK {
		return fmt.Errorf("control: the admin publishing masks with tenant A's own ids = %d %s, want 200", r.code, r.body)
	}
	return nil
}
