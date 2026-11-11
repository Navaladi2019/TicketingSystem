//go:build integration

package server_test

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// TICK-644 (P3-34c): the requester list is answered as a UNION ALL of the
// requester, assignee and share branches. It must return exactly the rows,
// order and total of the single OR statement, never a row the OR hides.
func TestListRecordsMineBranchesMatchTheSingleStatement(t *testing.T) {
	e := newRecEnv(t)
	e.publishACL(t, e.a.id, "ticket",
		authz.Rule{ID: "mine", Ops: []string{authz.OpRead}, Who: authz.Subject{Requester: true, Assignee: true, Shared: true}},
		authz.Rule{ID: "admins", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Users: []uuid.UUID{e.a.admin.PrincipalID}}})
	ctx := t.Context()
	me, mate := e.a.user.PrincipalID, e.a.lookalike.PrincipalID
	var made []string
	mk := func(key, extra string) recBody {
		r := e.mustCreateRec(t, "admin-a", "ticket", key, `{"short_description": "s"`+extra+`}`)
		made = append(made, r.ID)
		return r
	}
	share := func(tenant, rec string, user uuid.UUID) {
		t.Helper()
		if _, err := e.super.Exec(ctx, `INSERT INTO record_shares (tenant_id, record_id, user_id, principal_kind) VALUES ($1, $2, $3, 'user')`, tenant, rec, user); err != nil {
			t.Fatal(err)
		}
	}
	want := map[string]bool{}
	for i := 0; i < 40; i++ {
		switch i % 8 {
		case 0: // requester only
			want[mk(fmt.Sprintf("m%d", i), fmt.Sprintf(`, "requester": %q`, me)).ID] = true
		case 1: // assignee only
			want[mk(fmt.Sprintf("m%d", i), fmt.Sprintf(`, "assignee": %q`, me)).ID] = true
		case 2: // shared only
			r := mk(fmt.Sprintf("m%d", i), "")
			share(e.a.id.String(), r.ID, me)
			want[r.ID] = true
		case 3: // requester, assignee and shared: appears once
			r := mk(fmt.Sprintf("m%d", i), fmt.Sprintf(`, "requester": %q, "assignee": %q`, me, me))
			share(e.a.id.String(), r.ID, me)
			want[r.ID] = true
		case 4: // somebody else's
			mk(fmt.Sprintf("m%d", i), fmt.Sprintf(`, "requester": %q, "assignee": %q`, mate, mate))
		case 5: // shared with somebody else only
			r := mk(fmt.Sprintf("m%d", i), "")
			share(e.a.id.String(), r.ID, mate)
		case 6: // a share that was revoked
			r := mk(fmt.Sprintf("m%d", i), "")
			share(e.a.id.String(), r.ID, me)
			if _, err := e.super.Exec(ctx, `DELETE FROM record_shares WHERE tenant_id = $1 AND record_id = $2`, e.a.id, r.ID); err != nil {
				t.Fatal(err)
			}
		default: // requester and assignee, two branches
			want[mk(fmt.Sprintf("m%d", i), fmt.Sprintf(`, "requester": %q, "assignee": %q`, me, me)).ID] = true
		}
	}
	// Tenant B: same principal id as requester and as a share, in another tenant.
	b := e.mustCreateRec(t, "admin-b", "incident", "xb", `{"short_description": "b"}`)
	// The foreign keys refuse a requester from another tenant, so the rows are
	// forged with triggers and constraints off (superuser, this connection only).
	if _, err := e.super.Exec(ctx, `SET session_replication_role = replica`); err != nil {
		t.Fatal(err)
	}
	if _, err := e.super.Exec(ctx, `UPDATE records SET requester_id = $3, assignee_id = $3 WHERE tenant_id = $1 AND id = $2`, e.b.id, b.ID, me); err != nil {
		t.Fatal(err)
	}
	bs := e.mustCreateRec(t, "admin-b", "incident", "xb2", `{"short_description": "b"}`)
	share(e.b.id.String(), bs.ID, me)
	if _, err := e.super.Exec(ctx, `RESET session_replication_role`); err != nil {
		t.Fatal(err)
	}

	// Mixed states: every third record of the lot is closed, so the state sorts
	// have ties and the tie-breakers (updated_at, id) decide the order.
	for i, id := range made {
		if i%3 == 0 {
			if rec := e.patch(t, "admin-a", "ticket", id, `W/"1"`, `{"state": "closed"}`); rec.Code != http.StatusOK {
				t.Fatalf("close = %d: %s", rec.Code, rec.Body)
			}
		}
	}

	var def uuid.UUID
	if err := e.super.QueryRow(ctx, "SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = 'ticket'", e.a.id).Scan(&def); err != nil {
		t.Fatal(err)
	}
	// single is the order of the single OR statement for a sort, as the store
	// documents it: NULLS FIRST descending, NULLS LAST ascending, state and the
	// foreign keys tie-break on updated_at, then id.
	single := func(col, dir, filter string) []string {
		t.Helper()
		nulls, tie := "LAST", ""
		if dir == "DESC" {
			nulls = "FIRST"
		}
		if col != "created_at" && col != "updated_at" {
			tie = "r.updated_at " + dir + ", "
		}
		rows, err := e.super.Query(ctx, `SELECT r.id::text FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2`+filter+`
		AND (r.requester_id = $3 OR r.assignee_id = $3 OR EXISTS (SELECT 1 FROM record_shares s WHERE s.tenant_id = r.tenant_id AND s.record_id = r.id AND s.principal_kind = 'user' AND s.principal_id = $3))
		ORDER BY r.`+col+` `+dir+` NULLS `+nulls+`, `+tie+`r.id `+dir, e.a.id, def, me)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var order []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			order = append(order, id)
		}
		return order
	}
	order := single("updated_at", "DESC", "")
	if len(order) != len(want) {
		t.Fatalf("fixture: single statement returns %d rows, built %d", len(order), len(want))
	}

	// Every sort the list takes, at limits 1, 4 and 100, against the single statement.
	for _, s := range []struct{ query, col, dir, filter string }{
		{"", "updated_at", "DESC", ""},
		{"sort=updated_at", "updated_at", "ASC", ""},
		{"sort=created_at", "created_at", "ASC", ""},
		{"sort=-created_at", "created_at", "DESC", ""},
		{"sort=state", "state", "ASC", ""},
		{"sort=-state", "state", "DESC", ""},
		{"sort=assignee", "assignee_id", "ASC", ""},
		{"sort=-assignee", "assignee_id", "DESC", ""},
		{"filter[state]=closed", "updated_at", "DESC", " AND r.state = 'closed'"},
	} {
		order := single(s.col, s.dir, s.filter)
		if len(order) == 0 || (s.filter == "" && len(order) != len(want)) {
			t.Fatalf("%q: fixture gives %d rows", s.query, len(order))
		}
		for _, limit := range []int{1, 4, 100} {
			var got []string
			next := ""
			for pages := 0; ; pages++ {
				q := fmt.Sprintf("limit=%d", limit)
				if s.query != "" {
					q += "&" + s.query
				}
				if next != "" {
					q += "&cursor=" + next
				}
				p := e.mustList(t, "user-a", "ticket", q)
				if p.Total != len(order) {
					t.Fatalf("%q limit %d: total %d, want %d", s.query, limit, p.Total, len(order))
				}
				for _, it := range p.Items {
					got = append(got, it.ID)
				}
				if p.Next == nil {
					break
				}
				next = *p.Next
				if pages > 100 {
					t.Fatal("walk does not end")
				}
			}
			if !slices.Equal(got, order) {
				t.Errorf("%q limit %d: walk returned\n%v\nsingle statement\n%v", s.query, limit, got, order)
			}
		}
	}
	// The tenant predicate on its own: a tenant B row that carries tenant A's
	// entity_definition_id and the caller as requester is hidden by the arms'
	// r.tenant_id = $1 alone. It is read as the superuser, which bypasses RLS,
	// so only the statement's own predicate stands between it and the page.
	forged := uuid.New()
	if _, err := e.super.Exec(ctx, `SET session_replication_role = replica`); err != nil {
		t.Fatal(err)
	}
	if _, err := e.super.Exec(ctx, `INSERT INTO records (tenant_id, id, entity_definition_id, version, schema_version, number, state, requester_id, created_at, updated_at, data)
		VALUES ($1, $2, $3, 1, 1, 'FORGED-1', 'new', $4, now() + interval '1 hour', now() + interval '1 hour', '{}')`, e.b.id, forged, def, me); err != nil {
		t.Fatal(err)
	}
	if _, err := e.super.Exec(ctx, `RESET session_replication_role`); err != nil {
		t.Fatal(err)
	}
	var visibleToAnyTenant int
	if err := e.super.QueryRow(ctx, `SELECT count(*) FROM records WHERE entity_definition_id = $1 AND requester_id = $2 AND id = $3`, def, me, forged).Scan(&visibleToAnyTenant); err != nil || visibleToAnyTenant != 1 {
		t.Fatalf("control: the forged row is not in the table: %d, %v", visibleToAnyTenant, err)
	}
	pol := &authz.Policy{Version: 1, Rules: []authz.Rule{{ID: "mine", Ops: []string{authz.OpRead}, Who: authz.Subject{Requester: true, Assignee: true, Shared: true}}}}
	lq := pipeline.ListQuery{TenantID: e.a.id, DefinitionID: def, Where: "true", SortColumn: "updated_at", Desc: true, Limit: 1000}
	lq.Branches, lq.Args = authz.ReadBranches(pol, identity.Principal{TenantID: e.a.id, PrincipalID: me}, "r", 2)
	if len(lq.Branches) != 3 {
		t.Fatalf("branches = %d, want 3", len(lq.Branches))
	}
	sql, args, err := pipeline.PageStatement(lq)
	if err != nil {
		t.Fatal(err)
	}
	prows, err := e.super.Query(ctx, sql, args...)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for prows.Next() {
		vals, err := prows.Values()
		if err != nil {
			t.Fatal(err)
		}
		if got, tn := uuid.UUID(vals[1].([16]byte)), uuid.UUID(vals[0].([16]byte)); got == forged || tn != e.a.id {
			t.Errorf("the page statement without RLS returned %v (tenant %v)", got, vals[0])
		}
		n++
	}
	prows.Close()
	// The page joins entity_definitions on the row's tenant, which would also
	// drop the forged row (tenant B has no such definition), so on its own that
	// does not prove the arms' predicate. A page of one does: the forged row is
	// the newest, and an arm without the tenant predicate would hand it to the
	// merge's LIMIT 1, the join would drop it, and the page would be empty.
	lq1 := lq
	lq1.Limit = 1
	sql1, args1, err := pipeline.PageStatement(lq1)
	if err != nil {
		t.Fatal(err)
	}
	one, err := e.super.Query(ctx, sql1, args1...)
	if err != nil {
		t.Fatal(err)
	}
	got1 := 0
	for one.Next() {
		got1++
	}
	one.Close()
	if got1 != 1 {
		t.Errorf("page of one without RLS returned %d rows, want 1: an arm let the other tenant's row take the slot", got1)
	}
	if n != len(want) {
		t.Errorf("page statement without RLS returned %d rows, want %d", n, len(want))
	}
	{
		csql := pipeline.CountStatement(lq)
		var total int
		if err := e.super.QueryRow(ctx, csql, append([]any{e.a.id, def}, lq.Args...)...).Scan(&total); err != nil || total != len(want) {
			t.Errorf("count statement without RLS = %d, %v; want %d", total, err, len(want))
		}
	}

	for _, id := range []string{b.ID, bs.ID} {
		if slices.Contains(order, id) || ids(e.mustList(t, "user-a", "ticket", "limit=100").Items)[id] {
			t.Errorf("tenant B record %s is in tenant A's list", id)
		}
	}
	rec, _ := e.list(t, "user-a", "incident", "")
	listCode(t, rec, http.StatusNotFound, "not_found")
}

// P3-34 scenario 8 for a requester principal: a walk of the branched list with
// inserts between pages (new requester, assignee and shared rows sort ahead of
// the cursor) has no duplicate and no gap among the rows that existed at the start.
func TestListRecordsMineKeysetWalkWithConcurrentInserts(t *testing.T) {
	e := newRecEnv(t)
	e.publishACL(t, e.a.id, "ticket",
		authz.Rule{ID: "mine", Ops: []string{authz.OpRead}, Who: authz.Subject{Requester: true, Assignee: true, Shared: true}},
		authz.Rule{ID: "admins", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Users: []uuid.UUID{e.a.admin.PrincipalID}}})
	me := e.a.user.PrincipalID
	start := map[string]bool{}
	for i := 0; i < 90; i++ {
		data := fmt.Sprintf(`{"short_description": "r%d", "requester": %q}`, i, me)
		switch i % 3 {
		case 1:
			data = fmt.Sprintf(`{"short_description": "r%d", "assignee": %q}`, i, me)
		case 2:
			data = fmt.Sprintf(`{"short_description": "r%d", "requester": %q, "assignee": %q}`, i, me, me)
		}
		start[e.mustCreateRec(t, "admin-a", "ticket", fmt.Sprintf("k%d", i), data).ID] = true
	}
	seen, next := map[string]bool{}, ""
	for pages := 1; ; pages++ {
		q := "limit=7"
		if next != "" {
			q += "&cursor=" + next
		}
		p := e.mustList(t, "user-a", "ticket", q)
		if p.Total < len(start) {
			t.Errorf("page %d: total %d, want at least %d", pages, p.Total, len(start))
		}
		for _, it := range p.Items {
			if seen[it.ID] {
				t.Fatalf("duplicate %s on page %d", it.ID, pages)
			}
			seen[it.ID] = true
		}
		if pages <= 4 {
			e.mustCreateRec(t, "admin-a", "ticket", fmt.Sprintf("kn%d", pages), fmt.Sprintf(`{"short_description": "late", "requester": %q}`, me))
			e.mustCreateRec(t, "admin-a", "ticket", fmt.Sprintf("ka%d", pages), fmt.Sprintf(`{"short_description": "late", "assignee": %q}`, me))
		}
		if p.Next == nil {
			break
		}
		next = *p.Next
	}
	for id := range start {
		if !seen[id] {
			t.Errorf("record %s existed at the start and was skipped", id)
		}
	}
}

// mineACL publishes the requester / assignee / share rule that the list reads
// as three branches, plus an admin rule.
func (e *etEnv) mineACL(t *testing.T) {
	t.Helper()
	e.publishACL(t, e.a.id, "ticket",
		authz.Rule{ID: "mine", Ops: []string{authz.OpRead}, Who: authz.Subject{Requester: true, Assignee: true, Shared: true}},
		authz.Rule{ID: "admins", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Users: []uuid.UUID{e.a.admin.PrincipalID}}})
}

// TICK-647 with TICK-644: the branched count caps at totalCap per arm and in
// the outer limit. The requester arm alone holds more than the cap, and the
// assignee arm adds a few: the sum is still capped, not 1,000 + 3.
func TestListRecordsMineTotalIsCappedAtTheCap(t *testing.T) {
	e := newRecEnv(t)
	e.mineACL(t)
	user := e.a.user.PrincipalID
	p := pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, pipeline.NoopSink{}, pipeline.PoolRunner{Pool: e.pool}, pipeline.SystemClock{})
	const rows = totalCap + 5
	if err := tenancy.InTenantTx(t.Context(), e.pool, e.a.id, func(tx pgx.Tx) error {
		for i := range rows + 3 {
			data := map[string]any{"short_description": "bulk"}
			if i < rows {
				data["requester"] = user.String()
			} else {
				data["assignee"] = user.String()
			}
			if _, err := p.Write(t.Context(), tx, pipeline.Command{Op: pipeline.OpCreate, EntityKey: "ticket", Data: data,
				Actor: identity.Principal{TenantID: e.a.id}, Source: pipeline.SourceSystem}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	got := e.mustList(t, "user-a", "ticket", "limit=5")
	if got.Total != totalCap || got.Capped == nil || !*got.Capped || len(got.Items) != 5 {
		t.Errorf("requester: total %d capped %v items %d; want %d, true, 5", got.Total, got.Capped, len(got.Items), totalCap)
	}
	// Under the cap the total is exact: the assignee arm alone, filtered.
	got = e.mustList(t, "user-a", "ticket", "filter[assignee]="+user.String())
	if got.Total != 3 || got.Capped != nil {
		t.Errorf("assignee filter: total %d capped %v; want exact 3, absent", got.Total, got.Capped)
	}
}

// TICK-647 with TICK-644: the data-filter statement timeout also bounds the
// branched page and count. A superuser holds ACCESS EXCLUSIVE on records, so
// the branched list blocks until the timeout and answers 422
// filter_too_expensive; once released the same request is served.
func TestListMineDataFilterTimeoutAnswers422(t *testing.T) {
	e := newETEnvWith(t, server.WithDataFilterTimeout(300*time.Millisecond))
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "t-a", recTypeBody("ticket", "TKT")); rec.Code != http.StatusCreated {
		t.Fatalf("seed type = %d: %s", rec.Code, rec.Body)
	}
	e.mineACL(t)
	e.mustCreateRec(t, "admin-a", "ticket", "k1", fmt.Sprintf(`{"short_description": "x", "requester": %q}`, e.a.user.PrincipalID))

	lock, err := e.super.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	released := false
	defer func() {
		if !released {
			_ = lock.Rollback(context.Background())
		}
	}()
	if _, err := lock.Exec(t.Context(), "LOCK TABLE records IN ACCESS EXCLUSIVE MODE"); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	rec, _ := e.list(t, "user-a", "ticket", "filter[short_description]=x")
	listCode(t, rec, http.StatusUnprocessableEntity, "filter_too_expensive")
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("timed-out branched list took %s, want about the 300 ms timeout", d)
	}
	if err := lock.Rollback(t.Context()); err != nil {
		t.Fatal(err)
	}
	released = true
	if p := e.mustList(t, "user-a", "ticket", "filter[short_description]=x"); p.Total != 1 {
		t.Errorf("branched list after the timeout: total %d, want 1", p.Total)
	}
}
