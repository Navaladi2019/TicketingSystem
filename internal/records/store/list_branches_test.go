package store_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
)

// A query with two or more Branches is one arm per branch, each with its own
// tenant, definition, filter, keyset, ORDER BY and LIMIT (TICK-644); one branch
// or none is the single statement.
func TestBranchedStatements(t *testing.T) {
	q := store.ListQuery{TenantID: uuid.New(), DefinitionID: uuid.New(), Where: "r.state = $3", Args: []any{"new", uuid.New()},
		Branches:   []string{"(r.requester_id = $4)", "(r.assignee_id = $4 AND NOT (r.requester_id = $4))"},
		SortColumn: "state", Desc: true, Keyset: "(r.id) < ($5)", KeysetArgs: []any{uuid.New()}, Limit: 51}
	sql, args, err := store.PageStatement(q)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(sql, "UNION ALL"); n != 1 {
		t.Errorf("UNION ALL count = %d, want 1:\n%s", n, sql)
	}
	for _, want := range []string{"r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.state = $3 AND (r.requester_id = $4) AND (r.id) < ($5)",
		"ORDER BY r.state DESC NULLS FIRST, r.updated_at DESC, r.id DESC LIMIT $6", "ORDER BY sort_key DESC NULLS FIRST, updated_at DESC, id DESC LIMIT $6", "x.sort_key::text"} {
		if !strings.Contains(sql, want) {
			t.Errorf("statement lacks %q:\n%s", want, sql)
		}
	}
	if len(args) != 6 || args[5] != 51 {
		t.Errorf("args = %v, want tenant, definition, 2 args, keyset, limit", args)
	}
	count := store.CountStatement(q)
	if strings.Count(count, fmt.Sprintf("LIMIT %d", store.TotalCap+1)) != 3 || !strings.Contains(count, "UNION ALL") {
		t.Errorf("count = %s", count)
	}

	q.Branches = q.Branches[:1]
	if sql, _, _ := store.PageStatement(q); strings.Contains(sql, "UNION ALL") {
		t.Errorf("one branch is not split:\n%s", sql)
	}
}
