//go:build integration

// P3-34 (TICK-619): GET /api/v1/records/{entity} through the real handler on
// real PostgreSQL. The page and total are under the ACL's read predicate;
// filter, sort and the keyset cursor work inside it; a masked field named
// anywhere is 400 masked_field.

package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

type listPage struct {
	Items []recBody `json:"items"`
	Next  *string   `json:"next_cursor"`
	Total int       `json:"total"`
}

func (e *etEnv) list(t *testing.T, token, entity, query string) (*httptest.ResponseRecorder, listPage) {
	t.Helper()
	path := "/api/v1/records/" + entity
	if query != "" {
		path += "?" + query
	}
	rec := e.do(t, http.MethodGet, path, token, "", "")
	var p listPage
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
			t.Fatalf("list body: %v: %s", err, rec.Body)
		}
	}
	return rec, p
}

func (e *etEnv) mustList(t *testing.T, token, entity, query string) listPage {
	t.Helper()
	rec, p := e.list(t, token, entity, query)
	if rec.Code != http.StatusOK {
		t.Fatalf("list %s ?%s = %d: %s", token, query, rec.Code, rec.Body)
	}
	return p
}

func listCode(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	var p struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil || rec.Code != status || p.Code != code {
		t.Fatalf("response = %d %s, want %d %s", rec.Code, rec.Body, status, code)
	}
}

func ids(items []recBody) map[string]bool {
	out := map[string]bool{}
	for _, it := range items {
		out[it.ID] = true
	}
	return out
}

func wantIDs(t *testing.T, name string, got []recBody, want ...recBody) {
	t.Helper()
	g, w := ids(got), map[string]bool{}
	for _, r := range want {
		w[r.ID] = true
	}
	if len(g) != len(w) || len(got) != len(w) {
		t.Errorf("%s: %d items, want %d", name, len(got), len(w))
	}
	for id := range w {
		if !g[id] {
			t.Errorf("%s: record %s missing", name, id)
		}
	}
}

// serviceDeskACL: the desk group (lookalike-a, "service-desk") reads and writes
// the records of its own group, a requester reads its own, the admins group
// reads and writes everything (the default incident ACL, P3-35).
func (e *etEnv) serviceDeskACL(t *testing.T, entity string) {
	t.Helper()
	e.publishACL(t, e.a.id, entity,
		authz.Rule{ID: "desk", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Groups: []uuid.UUID{e.a.lookalikeGroup}},
			Where: []authz.Cond{{Column: "assignment_group_id", Op: authz.CondIn, Caller: authz.CallerMyGroups}}},
		authz.Rule{ID: "requester", Ops: []string{authz.OpRead}, Who: authz.Subject{Requester: true}},
		authz.Rule{ID: "admins", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Groups: []uuid.UUID{e.a.adminGroup}}},
	)
}

// covers: listRecords 200
func TestListRecordsUnderTheACL(t *testing.T) {
	e := newRecEnv(t)
	e.serviceDeskACL(t, "ticket")
	desk, other, user := e.a.lookalikeGroup, e.a.adminGroup, e.a.user.PrincipalID
	mk := func(key, extra string) recBody {
		return e.mustCreateRec(t, "admin-a", "ticket", key, `{"short_description": "s"`+extra+`}`)
	}
	d1 := mk("l1", fmt.Sprintf(`, "group": %q`, desk))
	d2 := mk("l2", fmt.Sprintf(`, "group": %q`, desk))
	d3 := mk("l3", fmt.Sprintf(`, "group": %q`, desk))
	if rec := e.patch(t, "admin-a", "ticket", d3.ID, `W/"1"`, `{"state": "closed"}`); rec.Code != http.StatusOK {
		t.Fatalf("close = %d: %s", rec.Code, rec.Body)
	}
	o1 := mk("l4", fmt.Sprintf(`, "group": %q`, other))
	r1 := mk("l5", fmt.Sprintf(`, "requester": %q`, user))
	r2 := mk("l6", fmt.Sprintf(`, "group": %q, "requester": %q`, other, user))
	n1 := mk("l7", "")

	// 1. The agent sees exactly its group's records; total is that count.
	p := e.mustList(t, "lookalike-a", "ticket", "")
	wantIDs(t, "agent", p.Items, d1, d2, d3)
	if p.Total != 3 || p.Next != nil {
		t.Errorf("agent total = %d, next = %v; want 3, nil", p.Total, p.Next)
	}
	// 2. The requester sees only their own.
	p = e.mustList(t, "user-a", "ticket", "")
	wantIDs(t, "requester", p.Items, r1, r2)
	if p.Total != 2 {
		t.Errorf("requester total = %d, want 2", p.Total)
	}
	// 3. The admin sees all.
	p = e.mustList(t, "admin-a", "ticket", "")
	wantIDs(t, "admin", p.Items, d1, d2, d3, o1, r1, r2, n1)
	if p.Total != 7 {
		t.Errorf("admin total = %d, want 7", p.Total)
	}
	// 4. Filter and sort, still ACL-restricted; newest first.
	p = e.mustList(t, "lookalike-a", "ticket", "filter[state]=new&sort=-created_at")
	wantIDs(t, "state=new", p.Items, d1, d2)
	if p.Total != 2 || p.Items[0].ID != d2.ID || p.Items[1].ID != d1.ID {
		t.Errorf("state=new: total %d, order %v; want 2, newest first", p.Total, []string{p.Items[0].ID, p.Items[1].ID})
	}
	p = e.mustList(t, "lookalike-a", "ticket", "sort=created_at&filter[state][in]=new,closed")
	if p.Total != 3 || p.Items[0].ID != d1.ID || p.Items[2].ID != d3.ID {
		t.Errorf("sort=created_at in: total %d, first %s, last %s", p.Total, p.Items[0].ID, p.Items[2].ID)
	}
	// A filter on another group's id finds nothing the caller may not read.
	p = e.mustList(t, "lookalike-a", "ticket", "filter[group]="+other.String())
	if len(p.Items) != 0 || p.Total != 0 {
		t.Errorf("filter on another group: %d items, total %d; want none", len(p.Items), p.Total)
	}
}

// Security finding 2: a record the caller cannot read is in neither the page
// nor total, whatever the paging and the filter.
func TestListRecordsHidesUnreadableFromPageAndTotal(t *testing.T) {
	e := newRecEnv(t)
	secret := e.mustCreateRec(t, "admin-a", "ticket", "u1", `{"short_description": "private"}`)
	mine := e.mustCreateRec(t, "admin-a", "ticket", "u2", fmt.Sprintf(`{"short_description": "mine", "requester": %q}`, e.a.user.PrincipalID))
	e.publishACL(t, e.a.id, "ticket",
		authz.Rule{ID: "req", Ops: []string{authz.OpRead}, Who: authz.Subject{Requester: true}},
		authz.Rule{ID: "admins", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Users: []uuid.UUID{e.a.admin.PrincipalID}}})
	for _, q := range []string{"", "limit=1", "filter[state]=new", "sort=updated_at", "filter[short_description]=private"} {
		p := e.mustList(t, "user-a", "ticket", q)
		if ids(p.Items)[secret.ID] {
			t.Errorf("?%s: an unreadable record is in the page", q)
		}
		if q == "filter[short_description]=private" && (p.Total != 0 || len(p.Items) != 0) {
			t.Errorf("?%s: total %d items %d; the unreadable match must not count", q, p.Total, len(p.Items))
		} else if q != "filter[short_description]=private" && (p.Total != 1 || !ids(p.Items)[mine.ID] && q != "limit=1") {
			t.Errorf("?%s: total %d; want 1 (only the readable record)", q, p.Total)
		}
	}
	// No ACL at all: nothing and total 0 (fail closed).
	e2 := newETEnv(t)
	e2.noTestACL = true
	if rec := e2.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "n1", recTypeBody("bare", "BAR")); rec.Code != http.StatusCreated {
		t.Fatal(rec.Body)
	}
	e2.seedSystemRecord(t, "bare")
	if p := e2.mustList(t, "admin-a", "bare", ""); len(p.Items) != 0 || p.Total != 0 {
		t.Errorf("no ACL: %d items, total %d; want none", len(p.Items), p.Total)
	}
}

// covers: listRecords 400
// covers: listRecords 401
// covers: listRecords 404
func TestListRecordsErrors(t *testing.T) {
	m := newMaskEnv(t)
	// user-a is outside the desk group, so cost, triage and assignee are masked.
	for name, q := range map[string]string{
		"filter":                    "filter[cost]=1",
		"filter in":                 "filter[triage][in]=a,b",
		"filter on a fixed column":  "filter[assignee]=" + uuid.NewString(),
		"sort on a fixed column":    "sort=assignee",
		"descending sort":           "sort=-assignee",
		"masked beside an ok field": "filter[short_description]=x&filter[cost]=1",
	} {
		rec, _ := m.list(t, "user-a", "job", q)
		listCode(t, rec, http.StatusBadRequest, "masked_field")
		if name == "filter" && bytes.Contains(rec.Body.Bytes(), []byte("1")) && bytes.Contains(rec.Body.Bytes(), []byte(`"value"`)) {
			t.Errorf("a masked_field body must not echo the value: %s", rec.Body)
		}
	}
	// The desk group may read them: the same queries succeed.
	for _, q := range []string{"filter[cost]=1", "sort=assignee", "filter[triage][in]=a,b"} {
		m.mustList(t, "lookalike-a", "job", q)
	}
	for _, q := range []string{"filter[nope]=1", "sort=nope", "sort=cost", "filter[cost][gt]=1", "filter[assignee]=x", "filter[token]=x"} {
		rec, _ := m.list(t, "lookalike-a", "job", q)
		listCode(t, rec, http.StatusBadRequest, "validation_failed")
	}
	for _, q := range []string{"limit=0", "limit=201"} {
		rec, _ := m.list(t, "lookalike-a", "job", q)
		listCode(t, rec, http.StatusBadRequest, "validation_failed")
	}
	rec, _ := m.list(t, "lookalike-a", "job", "cursor=garbage")
	listCode(t, rec, http.StatusBadRequest, "invalid_cursor")

	// 6. An unknown entity type is the 404 of GET /records/{entity}/{id}.
	sameResponse(t, "list of an unknown entity",
		m.do(t, http.MethodGet, "/api/v1/records/nope", "admin-a", "", ""),
		m.do(t, http.MethodGet, "/api/v1/records/nope/"+uuid.NewString(), "admin-a", "", ""))
	if rec := m.do(t, http.MethodGet, "/api/v1/records/job", "", "", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token = %d, want 401", rec.Code)
	}
}

// The sort key a cursor carries is checked too: a cursor whose key is a
// masked field answers 400 masked_field even when the request names no sort.
func TestListRecordsMaskedSortKeyInCursor(t *testing.T) {
	m := newMaskEnv(t)
	user := m.tokens["user-a"]
	ctx := identity.NewContext(context.Background(), user)
	kr, err := cursor.NewKeyring("k1", cursor.Key{ID: "k1", Secret: bytes.Repeat([]byte("s"), 32)})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := cursor.ClaimsFor(ctx, "listRecords", url.Values{"_entity": {"job"}})
	if err != nil {
		t.Fatal(err)
	}
	pos, _ := json.Marshal(map[string]any{"s": "assignee", "d": true, "v": nil, "i": uuid.NewString()})
	tok, err := kr.Encode(claims, pos)
	if err != nil {
		t.Fatal(err)
	}
	rec, _ := m.list(t, "user-a", "job", "cursor="+tok)
	listCode(t, rec, http.StatusBadRequest, "masked_field")
	// The same forged cursor is merely invalid for the sort it does not fit.
	pos, _ = json.Marshal(map[string]any{"s": "created_at", "d": true, "v": nil, "i": uuid.NewString()})
	tok, _ = kr.Encode(claims, pos)
	rec, _ = m.list(t, "user-a", "job", "cursor="+tok)
	listCode(t, rec, http.StatusBadRequest, "invalid_cursor")
}

// 7. Another tenant's records are in neither the page nor total.
func TestListRecordsCrossTenant(t *testing.T) {
	e := newRecEnv(t)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", "tb", recTypeBody("ticket", "TKT")); rec.Code != http.StatusCreated {
		t.Fatal(rec.Body)
	}
	a := e.mustCreateRec(t, "admin-a", "ticket", "x1", `{"short_description": "a"}`)
	b := e.mustCreateRec(t, "admin-b", "ticket", "x2", `{"short_description": "b"}`)
	b2 := e.mustCreateRec(t, "admin-b", "ticket", "x3", `{"short_description": "b2"}`)
	p := e.mustList(t, "admin-a", "ticket", "")
	if len(p.Items) != 1 || p.Items[0].ID != a.ID || p.Total != 1 {
		t.Errorf("tenant A list: %d items, total %d", len(p.Items), p.Total)
	}
	p = e.mustList(t, "admin-b", "ticket", "")
	wantIDs(t, "tenant B", p.Items, b, b2)
	if p.Total != 2 {
		t.Errorf("tenant B total = %d, want 2", p.Total)
	}
	// An entity type only tenant B has is a 404 for tenant A.
	sameResponse(t, "B's type seen by A",
		e.do(t, http.MethodGet, "/api/v1/records/incident", "admin-a", "", ""),
		e.do(t, http.MethodGet, "/api/v1/records/nope", "admin-a", "", ""))
	// A cursor made by tenant B does not work for tenant A.
	e.mustCreateRec(t, "admin-b", "ticket", "x4", `{"short_description": "b3"}`)
	pb := e.mustList(t, "admin-b", "ticket", "limit=1")
	if pb.Next == nil {
		t.Fatal("no cursor")
	}
	rec, _ := e.list(t, "admin-a", "ticket", "limit=1&cursor="+*pb.Next)
	listCode(t, rec, http.StatusBadRequest, "invalid_cursor")
}

// 8. A walk with inserts between pages: no duplicate and no gap among the
// records that existed when the walk started.
//
// 200 rows and pages of 25 put every keyset branch (ASC/DESC x tie/NULL/
// distinct) on a page boundary: about half the rows have an assignee drawn
// from three principals (ties; the rest are NULL, so a boundary crosses the
// non-NULL to NULL transition), and every row is in state "new" (one big tie
// group, broken by id). The predicate does not depend on N, so 10k rows would
// test the same branches.
func TestListRecordsKeysetWalkWithConcurrentInserts(t *testing.T) {
	e := newRecEnv(t)
	assignees := []uuid.UUID{e.a.admin.PrincipalID, e.a.user.PrincipalID, e.a.lookalike.PrincipalID}
	existing := map[string]bool{}
	for i := 0; i < 200; i++ {
		data := fmt.Sprintf(`{"short_description": "r%d"}`, i)
		if i%2 == 0 {
			data = fmt.Sprintf(`{"short_description": "r%d", "assignee": %q}`, i, assignees[i/2%len(assignees)])
		}
		existing[e.mustCreateRec(t, "admin-a", "ticket", fmt.Sprintf("w%d", i), data).ID] = true
	}
	sorts := []string{"", "sort=created_at", "sort=state", "sort=-state", "sort=-updated_at", "sort=assignee", "sort=-assignee"}
	for n, sort := range sorts {
		start := map[string]bool{}
		for id := range existing {
			start[id] = true
		}
		q := "limit=25"
		if sort != "" {
			q += "&" + sort
		}
		seen, pages, next := map[string]bool{}, 0, ""
		for {
			query := q
			if next != "" {
				query += "&cursor=" + next
			}
			p := e.mustList(t, "admin-a", "ticket", query)
			pages++
			if p.Total < len(start) {
				t.Errorf("%q page %d: total %d, want at least %d", sort, pages, p.Total, len(start))
			}
			for _, it := range p.Items {
				if seen[it.ID] {
					t.Fatalf("%q: duplicate %s on page %d", sort, it.ID, pages)
				}
				seen[it.ID] = true
			}
			// Inserts while the walk is in progress: one with an assignee, one
			// without. The newest rows sort behind the cursor of a descending
			// -created_at/-updated_at walk, and into the tie group of a state or
			// assignee walk; none may disturb the rows that existed at the start.
			if pages <= 3 {
				key := fmt.Sprintf("w-ins-%d-%d", n, pages)
				e.mustCreateRec(t, "admin-a", "ticket", key+"a", fmt.Sprintf(`{"short_description": "late", "assignee": %q}`, assignees[pages%len(assignees)]))
				e.mustCreateRec(t, "admin-a", "ticket", key+"n", `{"short_description": "late"}`)
			}
			if p.Next == nil {
				break
			}
			next = *p.Next
		}
		for id := range start {
			if !seen[id] {
				t.Errorf("%q: record %s existed at the start and was skipped", sort, id)
			}
		}
		// Later walks start from a larger set.
		for id := range seen {
			existing[id] = true
		}
	}
}

// Risk 1: a cursor minted for admin and sent by another caller of the same
// tenant is invalid, not a page.
func TestListRecordsCursorIsBoundToTheCaller(t *testing.T) {
	e := newRecEnv(t)
	for i := 0; i < 3; i++ {
		e.mustCreateRec(t, "admin-a", "ticket", fmt.Sprintf("cb%d", i), `{"short_description": "s"}`)
	}
	pa := e.mustList(t, "admin-a", "ticket", "limit=1")
	if pa.Next == nil {
		t.Fatal("no cursor")
	}
	if p := e.mustList(t, "admin-a", "ticket", "limit=1&cursor="+*pa.Next); len(p.Items) != 1 {
		t.Fatalf("control: the admin's own cursor gave %d items", len(p.Items))
	}
	rec, _ := e.list(t, "user-a", "ticket", "limit=1&cursor="+*pa.Next)
	listCode(t, rec, http.StatusBadRequest, "invalid_cursor")
}

// Risk 2: tenant A filtering on one of tenant B's group ids finds nothing.
func TestListRecordsFilterOnAnotherTenantsGroup(t *testing.T) {
	e := newRecEnv(t)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", "tg", recTypeBody("ticket", "TKT")); rec.Code != http.StatusCreated {
		t.Fatal(rec.Body)
	}
	e.mustCreateRec(t, "admin-b", "ticket", "g1", fmt.Sprintf(`{"short_description": "b", "group": %q}`, e.b.adminGroup))
	e.mustCreateRec(t, "admin-a", "ticket", "g2", `{"short_description": "a"}`)
	for _, q := range []string{"filter[group]=" + e.b.adminGroup.String(), "filter[group][in]=" + e.b.adminGroup.String() + "," + uuid.NewString()} {
		p := e.mustList(t, "admin-a", "ticket", q)
		if len(p.Items) != 0 || p.Total != 0 {
			t.Errorf("?%s: %d items, total %d; want none", q, len(p.Items), p.Total)
		}
	}
}

// Items are shaped by the caller's masks as GET is.
func TestListRecordsItemsAreShaped(t *testing.T) {
	m := newMaskEnv(t)
	r := m.mustCreateRec(t, "lookalike-a", "job", "sh1", `{"short_description": "s", "cost": 5}`)
	for token, wantCost := range map[string]bool{"lookalike-a": true, "user-a": false} {
		p := m.mustList(t, token, "job", "")
		if len(p.Items) != 1 || p.Items[0].ID != r.ID {
			t.Fatalf("%s: items %v", token, p.Items)
		}
		if _, has := p.Items[0].Data["cost"]; has != wantCost {
			t.Errorf("%s: cost in data = %v, want %v", token, has, wantCost)
		}
	}
}

// P3-34a (TICK-639): every page statement is answered by an index scan in the
// requested order. EXPLAIN (FORMAT JSON) of the statement the handler builds,
// for each sort a sort index serves (updated_at, created_at, state; the
// assignee, group and requester columns have access-path indexes, not sort
// indexes), both directions, with and without a cursor, must hold no Sort
// node. Dropping an index (P3-34b) or changing the ORDER BY turns it red.
// Sequential and bitmap scans are off so the planner can only use an index
// that matches; with 4000 rows a missing index otherwise hides behind a
// cheap sort.
func TestListRecordsPageStatementsNeedNoSort(t *testing.T) {
	e := newRecEnv(t)
	ctx := t.Context()
	var def uuid.UUID
	if err := e.super.QueryRow(ctx, "SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = 'ticket'", e.a.id).Scan(&def); err != nil {
		t.Fatal(err)
	}
	// Bulk rows with the triggers off (the records commit check wants an audit
	// entry per write; this test reads plans, not audit).
	tx, err := e.super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "SET LOCAL session_replication_role = replica"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO records (tenant_id, id, entity_definition_id, version, schema_version, number, state, created_at, updated_at, data)
		 SELECT $1, gen_random_uuid(), $2, 1, 1, 'PLAN-' || g, CASE WHEN g % 3 = 0 THEN 'closed' ELSE 'new' END,
		        now() - (g || ' seconds')::interval, now() - ((g * 7919 % 4000) || ' seconds')::interval, '{}'
		 FROM generate_series(1, 4000) g`, e.a.id, def); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := e.super.Exec(ctx, "ANALYZE records"); err != nil {
		t.Fatal(err)
	}
	for _, sort := range []string{"", "updated_at", "-updated_at", "created_at", "-created_at", "state", "-state"} {
		for _, cursor := range []bool{false, true} {
			name := fmt.Sprintf("sort=%q cursor=%v", sort, cursor)
			sql, args := server.ListPageStatement(t, sort, cursor, e.a.id, def)
			plan := explain(t, e, sql, args)
			if hasNode(plan, "Sort") || hasNode(plan, "Incremental Sort") {
				t.Errorf("%s: plan has a Sort node:\n%s", name, plan)
			}
			if !hasNode(plan, "Index Scan") && !hasNode(plan, "Index Only Scan") {
				t.Errorf("%s: plan is no index scan:\n%s", name, plan)
			}
		}
	}
}

func explain(t *testing.T, e *etEnv, sql string, args []any) string {
	t.Helper()
	ctx := t.Context()
	tx, err := e.super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, s := range []string{"SET LOCAL enable_seqscan = off", "SET LOCAL enable_bitmapscan = off", "SET LOCAL jit = off"} {
		if _, err := tx.Exec(ctx, s); err != nil {
			t.Fatal(err)
		}
	}
	var plan string
	if err := tx.QueryRow(ctx, "EXPLAIN (FORMAT JSON) "+sql, args...).Scan(&plan); err != nil {
		t.Fatalf("explain: %v\n%s", err, sql)
	}
	return plan
}

// hasNode reports whether the JSON plan holds a node of the given type.
func hasNode(plan, nodeType string) bool {
	var doc []struct{ Plan map[string]any }
	if err := json.Unmarshal([]byte(plan), &doc); err != nil || len(doc) == 0 {
		return false
	}
	var walk func(n map[string]any) bool
	walk = func(n map[string]any) bool {
		if n["Node Type"] == nodeType {
			return true
		}
		kids, _ := n["Plans"].([]any)
		for _, k := range kids {
			if m, ok := k.(map[string]any); ok && walk(m) {
				return true
			}
		}
		return false
	}
	return walk(doc[0].Plan)
}
