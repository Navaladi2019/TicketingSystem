package store_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
)

// ADR-010-A2 A2-5 (TICK-1036): a term marked Materialize is C3, its rows read
// into a MATERIALIZED CTE and then sorted; the count reads CountWhere or
// CountBranches when they are set, and Where or Branches otherwise.
func TestC3PageAndCountTerms(t *testing.T) {
	base := store.ListQuery{TenantID: uuid.New(), DefinitionID: uuid.New(), Args: []any{uuid.New()},
		SortColumn: "updated_at", Desc: true, Keyset: "(r.id) < ($4)", KeysetArgs: []any{uuid.New()}, Limit: 51}

	single := base
	single.Where, single.CountWhere, single.Materialize = "PAGE($3)", "COUNT($3)", []bool{true}
	sql, args, err := store.PageStatement(single)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"WITH acl_m AS MATERIALIZED (SELECT r.* FROM records r\nWHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND PAGE($3) AND (r.id) < ($4))",
		"FROM acl_m r\nJOIN entity_definitions d",
		"ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $5",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("C3 page lacks %q:\n%s", want, sql)
		}
	}
	if len(args) != 5 {
		t.Errorf("args = %v", args)
	}
	if c := store.CountStatement(single); !strings.Contains(c, "COUNT($3)") || strings.Contains(c, "PAGE(") {
		t.Errorf("count = %s, want CountWhere", c)
	}
	single.Materialize = []bool{false}
	if sql, _, _ := store.PageStatement(single); strings.Contains(sql, "MATERIALIZED") {
		t.Errorf("an unmarked term is materialized:\n%s", sql)
	}

	split := base
	split.Where = "true"
	split.Branches = []string{"(PAGE0($3))", "(r.requester_id = $3 AND NOT (PAGE0($3)))"}
	split.CountBranches = []string{"(COUNT0($3))", "(r.requester_id = $3 AND NOT (COUNT0($3)))"}
	split.Materialize = []bool{true, false}
	sql, _, err = store.PageStatement(split)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(sql, "MATERIALIZED"); n != 1 ||
		!strings.Contains(sql, "(WITH acl_m AS MATERIALIZED (SELECT ") ||
		!strings.Contains(sql, "AND (PAGE0($3)) AND (r.id) < ($4)) SELECT * FROM acl_m r ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $5)") {
		t.Errorf("branched C3 page:\n%s", sql)
	}
	if c := store.CountStatement(split); strings.Count(c, "COUNT0(") != 2 || strings.Contains(c, "PAGE0(") {
		t.Errorf("count = %s, want CountBranches", c)
	}
	split.CountBranches = nil
	if c := store.CountStatement(split); strings.Count(c, "PAGE0(") != 2 {
		t.Errorf("count without CountBranches = %s, want Branches", c)
	}
}
