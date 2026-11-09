//go:build integration

package server_test

import (
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
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
	mk := func(key, extra string) recBody {
		return e.mustCreateRec(t, "admin-a", "ticket", key, `{"short_description": "s"`+extra+`}`)
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

	var def uuid.UUID
	if err := e.super.QueryRow(ctx, "SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = 'ticket'", e.a.id).Scan(&def); err != nil {
		t.Fatal(err)
	}
	rows, err := e.super.Query(ctx, `SELECT r.id::text FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2
		AND (r.requester_id = $3 OR r.assignee_id = $3 OR EXISTS (SELECT 1 FROM record_shares s WHERE s.tenant_id = r.tenant_id AND s.record_id = r.id AND s.principal_kind = 'user' AND s.principal_id = $3))
		ORDER BY r.updated_at DESC, r.id DESC`, e.a.id, def, me)
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		order = append(order, id)
	}
	rows.Close()
	if len(order) != len(want) {
		t.Fatalf("fixture: single statement returns %d rows, built %d", len(order), len(want))
	}

	for _, limit := range []int{1, 7, 100} {
		var got []string
		next := ""
		for pages := 0; ; pages++ {
			q := fmt.Sprintf("limit=%d", limit)
			if next != "" {
				q += "&cursor=" + next
			}
			p := e.mustList(t, "user-a", "ticket", q)
			if p.Total != len(want) {
				t.Fatalf("limit %d: total %d, want %d", limit, p.Total, len(want))
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
			t.Errorf("limit %d: walk returned\n%v\nsingle statement\n%v", limit, got, order)
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
